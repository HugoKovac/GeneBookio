//go:build integration

// Package integration_test drives the full book pipeline — upload, EPUB
// split, AI preparation, AI script generation, TTS synthesis — through real
// MySQL, MinIO and RabbitMQ, wired exactly like each cmd/*/main.go does.
// All three AI-driven stages use their package's substitution client (the
// same stand-in AI_TEST_MODE wires in production) so the test never spends
// real money on AI tokens or audio synthesis.
//
// Requires the docker-hybrid infra running (`make docker-hybrid` from
// backend/) and a test/.env file (copy test/.env.example) pointing at it.
//
// Run with:
//
//	go test -tags integration ./test/... -v -timeout 10m
package integration_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"errors"
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/catalog"
	"hkorpo/book/internal/parsing"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/database"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/preparation"
	"hkorpo/book/internal/pricing"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/internal/script"
	"hkorpo/book/internal/tts"
	"hkorpo/book/internal/upload"
	"hkorpo/book/pkg/env"
	"hkorpo/book/pkg/errorpkg"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/minio/minio-go/v7"
	amqp091 "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	queue.ConfigQueue
	bucket.ConfigBucket
	database.ConfigDB
}

// stageResult carries a pipeline stage's outcome across the consumer
// goroutine boundary so a failure surfaces immediately instead of as a
// generic wait timeout.
type stageResult struct {
	bookID string
	err    error
}

func TestFullPipeline(t *testing.T) {
	env.LoadEnv()

	var cfg testConfig
	require.NoError(t, envconfig.Process("", &cfg))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	minioClient, err := bucket.Init(&cfg.ConfigBucket)
	require.NoError(t, err)
	ensureBuckets(ctx, t, minioClient)
	seedPrompts(ctx, t, minioClient)

	dbClient, err := database.Init(&cfg.ConfigDB)
	require.NoError(t, err)
	t.Cleanup(func() { dbClient.Close() })

	repo := book.NewRepositoryImpl(dbClient)
	bucketRepo := book.NewBucketRepoImpl(minioClient)

	// Use dedicated, uniquely-named queues instead of the shared
	// split/prepare/generate_script/generate_tts channels: this test's
	// consumers would otherwise compete with any real epub_parser /
	// prepare_chapters / generate_script / generate_tts worker already
	// running (e.g. via `make docker`), and RabbitMQ round-robins a
	// message to whichever competing consumer it picks — silently
	// starving this test's own handler.
	runID := uuid.NewString()[:8]
	splitChannel := primitive.QueueChannel("test-" + string(primitive.Split) + "-" + runID)
	prepareChannel := primitive.QueueChannel("test-" + string(primitive.Prepare) + "-" + runID)
	scriptChannel := primitive.QueueChannel("test-" + string(primitive.GenerateScript) + "-" + runID)
	ttsChannel := primitive.QueueChannel("test-" + string(primitive.GenerateTTS) + "-" + runID)

	splitQ, splitCh, err := queue.InitProducer(&cfg.ConfigQueue, splitChannel)
	require.NoError(t, err)
	prepareQ, prepareCh, err := queue.InitProducer(&cfg.ConfigQueue, prepareChannel)
	require.NoError(t, err)
	scriptQ, scriptCh, err := queue.InitProducer(&cfg.ConfigQueue, scriptChannel)
	require.NoError(t, err)
	ttsQ, ttsAMQPCh, err := queue.InitProducer(&cfg.ConfigQueue, ttsChannel)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = splitCh.QueueDelete(string(splitQ.Name), false, false, false)
		_, _ = prepareCh.QueueDelete(string(prepareQ.Name), false, false, false)
		_, _ = scriptCh.QueueDelete(string(scriptQ.Name), false, false, false)
		_, _ = ttsAMQPCh.QueueDelete(string(ttsQ.Name), false, false, false)
	})

	// Substitution clients always cost $0 (their "test-mode"/"test-mode-tts"
	// model isn't in the pricing table), so budget checks short-circuit
	// without ever calling the exchange-rate API — safe to use the real
	// Calculator here.
	pricingCalculator := pricing.NewCalculator(pricing.NewExchangeRateClient())

	uploadService := upload.NewService(repo, bucketRepo, book.NewQueueRepoImpl(splitQ, splitCh))
	parsingService := parsing.NewService(repo, bucketRepo, book.NewQueueRepoImpl(prepareQ, prepareCh), parsing.NewEpubParserImpl())
	preparationService := preparation.NewService(repo, bucketRepo, book.NewQueueRepoImpl(scriptQ, scriptCh), book.NewSubstitutionAiClient(), pricingCalculator)
	scriptService := script.NewService(repo, bucketRepo, book.NewQueueRepoImpl(ttsQ, ttsAMQPCh), book.NewSubstitutionAiClient(), pricingCalculator)
	ttsService := tts.NewService(repo, bucketRepo, tts.NewSubstitutionTTSClient(), pricingCalculator)

	// Wire one consumer per stage, exactly like each cmd/*/main.go does,
	// but report success/failure on a channel instead of just logging it.
	splitDone := make(chan stageResult, 1)
	require.NoError(t, queue.InitConsumer(&cfg.ConfigQueue, splitChannel, func(d amqp091.Delivery) error {
		id := string(d.Body)
		chunks, err := parsingService.GetBookAsChunks(ctx, id)
		if err == nil {
			err = parsingService.UploadBookChunks(ctx, id, chunks)
		}
		splitDone <- stageResult{bookID: id, err: err}
		return err
	}))

	preparedDone := make(chan stageResult, 1)
	require.NoError(t, queue.InitConsumer(&cfg.ConfigQueue, prepareChannel, func(d amqp091.Delivery) error {
		id := string(d.Body)
		err := preparationService.MapOnChunks(ctx, id, preparationService.GenerateChapterPreparation)
		preparedDone <- stageResult{bookID: id, err: err}
		return err
	}))

	scriptDone := make(chan stageResult, 1)
	require.NoError(t, queue.InitConsumer(&cfg.ConfigQueue, scriptChannel, func(d amqp091.Delivery) error {
		id := string(d.Body)
		err := scriptService.GenerateScript(ctx, id)
		scriptDone <- stageResult{bookID: id, err: err}
		return err
	}))

	ttsDone := make(chan stageResult, 1)
	require.NoError(t, queue.InitConsumer(&cfg.ConfigQueue, ttsChannel, func(d amqp091.Delivery) error {
		id := string(d.Body)
		err := ttsService.CreateAudioFromScript(ctx, id)
		ttsDone <- stageResult{bookID: id, err: err}
		return err
	}))

	// --- seed a book row + a tiny synthetic EPUB, then kick off the pipeline ---

	key := "it-" + uuid.NewString()[:8]
	created, err := repo.CreateBook(ctx, &book.Book{
		Title:       "Pipeline Integration Test",
		Key:         key,
		AuthorNames: []string{"Integration Test"},
		Description: "Book created by TestFullPipeline.",
		Language:    primitive.English,
	})
	require.NoError(t, err)
	bookID := created.ID.String()

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = dbClient.Book.DeleteOneID(created.ID).Exec(cleanupCtx)
		removePrefix(cleanupCtx, minioClient, primitive.BooksBucket, "uploads/"+bookID)
		removePrefix(cleanupCtx, minioClient, primitive.BooksBucket, "chunks/"+bookID+"/")
		removePrefix(cleanupCtx, minioClient, primitive.ScriptsBucket, bookID+"/")
		removePrefix(cleanupCtx, minioClient, primitive.AudioBucket, bookID)
	})

	epubBytes := buildTestEPUB(t)
	require.NoError(t, uploadService.UploadNewBook(ctx, bookID, string(epubBytes)))
	t.Logf("uploaded book %s, waiting on pipeline...", bookID)

	waitStage(t, "split", bookID, splitDone, 30*time.Second)
	waitStage(t, "prepare", bookID, preparedDone, 30*time.Second)
	waitStage(t, "generate_script", bookID, scriptDone, 30*time.Second)
	waitStage(t, "generate_tts", bookID, ttsDone, 30*time.Second)

	// --- assert the pipeline actually did the work at every stage ---

	final, err := repo.GetBookByID(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, final.Uploaded, "book should be marked uploaded")
	require.True(t, final.Parsed, "book should be marked parsed")
	require.True(t, final.Prepared, "book should be marked prepared")
	require.True(t, final.ScriptGenerated, "book should be marked script_generated")
	require.Contains(t, final.TokenUsage, "test-mode", "usage from the substitution AI/TTS clients should have been recorded")

	// The domain model doesn't surface tts_generated yet, so check ent directly.
	entBook, err := dbClient.Book.Get(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, entBook.TtsGenerated, "book should be marked tts_generated")

	chunkCount := 0
	for range bucketRepo.GetFilesIteratorOfDir(ctx, primitive.BooksBucket, "chunks/"+bookID+"/") {
		chunkCount++
	}
	require.Equal(t, 1, chunkCount, "expected exactly one chapter chunk to have been split out")

	preparedContent, err := bucketRepo.GetBucketFileAsString(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/preparation/000.txt", bookID))
	require.NoError(t, err)
	require.Contains(t, preparedContent, "==START OF REQUEST==", "preparation output should come from the AI substitution client, not a real AI call")

	scriptContent, err := bucketRepo.GetBucketFileAsString(ctx, primitive.ScriptsBucket, bookID+"/script.txt")
	require.NoError(t, err)
	require.Contains(t, scriptContent, "==START OF REQUEST==", "script output should come from the AI substitution client, not a real AI call")

	audioReader, audioLen, _, err := bucketRepo.GetBucketObjectAsReader(ctx, primitive.AudioBucket, bookID)
	require.NoError(t, err)
	if closer, ok := audioReader.(io.Closer); ok {
		defer closer.Close()
	}
	require.Greater(t, audioLen, int64(0), "synthesized audio should be non-empty")

	t.Logf("pipeline completed: book %s produced %d bytes of audio", bookID, audioLen)
}

// fakeExpensiveAiAPI reports usage that's over budget under any of gpt-5.2's
// published rates, without ever calling a real AI model.
type fakeExpensiveAiAPI struct{}

func (fakeExpensiveAiAPI) ModelName() string { return "gpt-5.2" }
func (fakeExpensiveAiAPI) Request(_ context.Context, _ string, _ int64) (string, primitive.ModelUsage, error) {
	return "fake output", primitive.ModelUsage{OutputTokens: 1_000_000}, nil
}

// TestPreparationOverBudgetFailsPermanently checks the budget guardrail's
// reactive path: fakeExpensiveAiAPI's declared usage is what a real call
// would report, so pricing.Calculator.CapOutputTokens's pre-call check (the
// input here is tiny, so it lets the call through) doesn't intercept it —
// the aggregate CheckBudget after the call does. Either way the effect is
// the same: the book fails permanently (RetryDisabled), and
// catalog.Service.RetryFailedStage refuses to retry it — retrying would
// just spend more money repeating already-over-budget work. See
// internal/pricing's own tests for the pre-call CapOutputTokens path.
func TestPreparationOverBudgetFailsPermanently(t *testing.T) {
	env.LoadEnv()

	var cfg testConfig
	require.NoError(t, envconfig.Process("", &cfg))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	minioClient, err := bucket.Init(&cfg.ConfigBucket)
	require.NoError(t, err)
	ensureBuckets(ctx, t, minioClient)
	seedPrompts(ctx, t, minioClient)

	dbClient, err := database.Init(&cfg.ConfigDB)
	require.NoError(t, err)
	t.Cleanup(func() { dbClient.Close() })

	repo := book.NewRepositoryImpl(dbClient)
	bucketRepo := book.NewBucketRepoImpl(minioClient)
	pricingCalculator := pricing.NewCalculator(pricing.NewExchangeRateClient())

	key := "it-budget-" + uuid.NewString()[:8]
	created, err := repo.CreateBook(ctx, &book.Book{
		Title:       "Over Budget Test",
		Key:         key,
		AuthorNames: []string{"Integration Test"},
		Description: "Book created by TestPreparationOverBudgetFailsPermanently.",
		Language:    primitive.English,
	})
	require.NoError(t, err)
	bookID := created.ID.String()

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = dbClient.Book.DeleteOneID(created.ID).Exec(cleanupCtx)
		removePrefix(cleanupCtx, minioClient, primitive.BooksBucket, "chunks/"+bookID+"/")
		removePrefix(cleanupCtx, minioClient, primitive.ScriptsBucket, bookID+"/")
	})

	require.NoError(t, bucketRepo.UploadString(ctx, primitive.BooksBucket, "chunks/"+bookID+"/000.txt", "Chapter content.", bucket.TEXT))

	preparationService := preparation.NewService(repo, bucketRepo, book.NewQueueRepoImpl(nil, nil), fakeExpensiveAiAPI{}, pricingCalculator)

	err = preparationService.MapOnChunks(ctx, bookID, preparationService.GenerateChapterPreparation)
	require.Error(t, err, "preparation should fail once its cost exceeds the €1 budget")
	require.True(t, pricing.IsBudgetExceeded(err), "the error should be a *pricing.BudgetExceededError, got %T: %v", err, err)

	// RecordPermanentFailure deliberately returns cause (wrapped), not nil,
	// so the caller's existing error-logging path still fires — mirroring
	// book.RecordFailure. What matters here is the DB write it made.
	require.Error(t, book.RecordPermanentFailure(ctx, repo, bookID, primitive.Prepare, err))

	final, err := repo.GetBookByID(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, final.Failed, "book should be marked failed")
	require.True(t, final.RetryDisabled, "an over-budget failure must not be retryable")
	require.Equal(t, string(primitive.Prepare), final.FailedStage)
	require.False(t, final.Prepared, "book must not advance to the next stage once over budget")

	catalogService := catalog.NewService(repo, bucketRepo, nil, pricingCalculator)
	retryErr := catalogService.RetryFailedStage(ctx, bookID)
	require.Error(t, retryErr, "retrying a permanently-failed book must be refused")

	var budgetErr *pricing.BudgetExceededError
	require.False(t, errors.As(retryErr, &budgetErr), "the retry refusal itself isn't a budget error")
}

func waitStage(t *testing.T, name, bookID string, ch <-chan stageResult, timeout time.Duration) {
	t.Helper()
	select {
	case res := <-ch:
		if res.err != nil {
			if st := errorpkg.DeepestErrorWrapper(res.err); st != nil {
				t.Logf("%s stage trace: %+v", name, st.FormatTrace())
			}
		}
		require.NoError(t, res.err, "%s stage failed", name)
		require.Equal(t, bookID, res.bookID, "%s stage processed the wrong book", name)
		t.Logf("%s stage completed", name)
	case <-time.After(timeout):
		t.Fatalf("timed out after %s waiting for %s stage", timeout, name)
	}
}

func ensureBuckets(ctx context.Context, t *testing.T, client *minio.Client) {
	t.Helper()
	for _, b := range []primitive.Bucket{primitive.PromptsBucket, primitive.BooksBucket, primitive.ScriptsBucket, primitive.AudioBucket} {
		exists, err := client.BucketExists(ctx, string(b))
		require.NoError(t, err)
		if !exists {
			require.NoError(t, client.MakeBucket(ctx, string(b), minio.MakeBucketOptions{}))
		}
	}
}

// seedPrompts uploads backend/backup-prompts/*.md into the prompts bucket
// under their own filenames, matching primitive.PromptFile's naming
// convention, so preparation/script.Service can load them.
func seedPrompts(ctx context.Context, t *testing.T, client *minio.Client) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	promptsDir := filepath.Join(filepath.Dir(thisFile), "..", "backup-prompts")

	files, err := filepath.Glob(filepath.Join(promptsDir, "*.md"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no prompt files found in %s", promptsDir)

	for _, f := range files {
		content, err := os.ReadFile(f)
		require.NoError(t, err)
		_, err = client.PutObject(ctx, string(primitive.PromptsBucket), filepath.Base(f),
			bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{ContentType: "text/plain; charset=utf-8"})
		require.NoError(t, err)
	}
}

func removePrefix(ctx context.Context, client *minio.Client, b primitive.Bucket, prefix string) {
	if !isDirPrefix(prefix) {
		_ = client.RemoveObject(ctx, string(b), prefix, minio.RemoveObjectOptions{})
		return
	}
	for obj := range client.ListObjects(ctx, string(b), minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			continue
		}
		_ = client.RemoveObject(ctx, string(b), obj.Key, minio.RemoveObjectOptions{})
	}
}

func isDirPrefix(prefix string) bool {
	return len(prefix) > 0 && prefix[len(prefix)-1] == '/'
}

// buildTestEPUB builds a minimal, valid, single-chapter EPUB in memory so
// the test never depends on a checked-in (gitignored) book fixture, and so
// real TTS synthesis of its narration script stays fast.
func buildTestEPUB(t *testing.T) []byte {
	t.Helper()

	const containerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	const contentOPF = `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata></metadata>
  <manifest>
    <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chap1"/>
  </spine>
</package>`

	const chapter1 = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter One</title></head>
<body><h1>Chapter One</h1><p>Hello world. This is a short integration test chapter used to exercise the narration pipeline end to end.</p></body>
</html>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": containerXML,
		"OEBPS/content.opf":      contentOPF,
		"OEBPS/chapter1.xhtml":   chapter1,
	}
	for name, content := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	return buf.Bytes()
}
