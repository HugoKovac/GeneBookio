package preparation

import (
	"context"
	"fmt"
	"hkorpo/book/internal/book"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/pricing"
	"hkorpo/book/internal/primitive"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type AiAPI interface {
	// maxOutputTokens, when positive, caps the response length — see
	// BudgetAPI.CapOutputTokens.
	Request(ctx context.Context, request string, maxOutputTokens int64) (string, primitive.ModelUsage, error)
	ModelName() string
}

// BudgetAPI checks a stage's AI usage cost against a EUR budget.
type BudgetAPI interface {
	// CheckBudget is the post-call check: did this call's real (reported)
	// usage exceed the budget?
	CheckBudget(ctx context.Context, stage, model string, usage primitive.ModelUsage, limitEUR float64) error
	// CapOutputTokens is the pre-call check: given an estimated input size,
	// how many output tokens can the remaining budget still afford? It
	// returns an error — before any request is made — if the (estimated)
	// input alone already exceeds the budget.
	CapOutputTokens(ctx context.Context, stage, model string, inputTokens int64, limitEUR float64) (int64, error)
}

// budgetEUR is the maximum a single book may spend on preparation before
// the book fails permanently (see MapOnChunks).
const budgetEUR = 1.0

// Service AI-preprocesses each chapter chunk of a book in parallel.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	queueRepo  book.QueueRepo
	aiAPI      AiAPI
	budgetAPI  BudgetAPI
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, queueRepo book.QueueRepo, aiAPI AiAPI, budgetAPI BudgetAPI) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, queueRepo: queueRepo, aiAPI: aiAPI, budgetAPI: budgetAPI}
}

func (s *Service) GenerateChapterPreparation(ctx context.Context, count int, bookID, chunkName, prompt string) (primitive.ModelUsage, error) {
	content, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.BooksBucket, chunkName)
	if err != nil {
		return primitive.ModelUsage{}, err
	}

	request := prompt + content

	// Cap this request's output at what the (whole) stage budget still
	// affords, given its estimated input size. This bounds a single chunk's
	// worst case before spending anything on it — it doesn't by itself
	// guarantee the *sum* across every chunk of this book stays under
	// budget (chunks run concurrently, see MapOnChunks), which is why
	// MapOnChunks still runs CheckBudget on the aggregate afterwards.
	maxOutputTokens, err := s.budgetAPI.CapOutputTokens(ctx, primitive.Prepare, s.aiAPI.ModelName(), pricing.EstimateTokens(request), budgetEUR)
	if err != nil {
		return primitive.ModelUsage{}, err
	}

	output, usage, err := s.aiAPI.Request(ctx, request, maxOutputTokens)
	if err != nil {
		return primitive.ModelUsage{}, err
	}

	if err := s.bucketRepo.UploadString(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/preparation/%03d.txt", bookID, count), output, pbucket.TEXT); err != nil {
		return primitive.ModelUsage{}, err
	}
	return usage, nil
}

func (s *Service) MapOnChunks(ctx context.Context, bookID string, fn func(ctx context.Context, count int, bookID, chunkName, prompt string) (primitive.ModelUsage, error)) error {
	b, err := s.repo.GetBookByID(ctx, uuid.MustParse(bookID))
	if err != nil {
		return err
	}

	promptPrepareChapter, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.PromptFile(primitive.NoneFictionPrepareChapter, b.Language))
	if err != nil {
		return err
	}

	iter := s.bucketRepo.GetFilesIteratorOfDir(ctx, primitive.BooksBucket, "chunks/"+bookID+"/")

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	var (
		usageMu    sync.Mutex
		totalUsage primitive.ModelUsage
	)

	count := 0
	for i := range iter {
		c := count
		chunkKey := i.Key

		g.Go(func() error {
			usage, err := fn(gctx, c, bookID, chunkKey, promptPrepareChapter)
			if err != nil {
				return err
			}
			usageMu.Lock()
			totalUsage = totalUsage.Add(usage)
			usageMu.Unlock()
			return nil
		})

		count++
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("map on chunks failed: %w", err)
	}

	if err := s.repo.AddTokenUsage(ctx, uuid.MustParse(bookID), s.aiAPI.ModelName(), totalUsage); err != nil {
		return err
	}

	if err := s.budgetAPI.CheckBudget(ctx, primitive.Prepare, s.aiAPI.ModelName(), totalUsage, budgetEUR); err != nil {
		return err
	}

	if err := s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), book.Prepared); err != nil {
		return err
	}

	if err := s.queueRepo.PostMessage(bookID); err != nil {
		return err
	}

	return nil
}
