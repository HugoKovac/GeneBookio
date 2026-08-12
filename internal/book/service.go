package book

import (
	"context"
	"fmt"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"
	"io"
	"iter"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"golang.org/x/sync/errgroup"
)

type LibraryAPI interface {
	SearchBookByString(q string, page int) (books []*Book, err error)
	GetBookByKey(key string) (books *Book, err error)
}

type QueueRepo interface {
	PostMessage(msg string) error
}

type Repository interface {
	CreateBook(ctx context.Context, book *Book) (*Book, error)
	UpdateBookStage(ctx context.Context, bookID uuid.UUID, s Stage) error
	GetSavedBookByKey(ctx context.Context, bookKey string) (*Book, error)
}

type BucketRepo interface {
	UploadString(ctx context.Context, bucketName primitive.Bucket, path, content string, ctype pbucket.ContentType) error
	GetBucketFileAsString(ctx context.Context, bucket primitive.Bucket, path string) (string, error)
	GetBucketFileAsBytes(ctx context.Context, bucket primitive.Bucket, path string) ([]byte, error)
	GetFilesIteratorOfDir(ctx context.Context, bucket primitive.Bucket, path string) iter.Seq[minio.ObjectInfo]
	UploadReader(ctx context.Context, bucketName primitive.Bucket, path string, content io.ReadCloser, len int64, ctype pbucket.ContentType) error
}

type TTSAPI interface {
	CreateAudioFromString(ctx context.Context, content string) (io.ReadCloser, int64, error)
}

type EpubParser interface {
	ExtractEPUB(epubContent []byte) (map[string]string, error)
}

type AiAPI interface {
	Request(ctx context.Context, request string) (string, error)
}

type Service struct {
	bookAPI    LibraryAPI
	queueRepo  QueueRepo
	bucketRepo BucketRepo
	epubParser EpubParser
	aiAPI      AiAPI
	repo       Repository
	TTSAPI     TTSAPI
}

func WithRepository(repo Repository) func(*Service) {
	return func(s *Service) {
		s.repo = repo
	}
}

func WithTTSAPI(client TTSAPI) func(*Service) {
	return func(s *Service) {
		s.TTSAPI = client
	}
}

func WithLibraryAPI(lAPI LibraryAPI) func(*Service) {
	return func(s *Service) {
		s.bookAPI = lAPI
	}
}

func WithAiAPI(aiAPI AiAPI) func(*Service) {
	return func(s *Service) {
		s.aiAPI = aiAPI
	}
}

func WithQueueRepo(qRepo QueueRepo) func(*Service) {
	return func(s *Service) {
		s.queueRepo = qRepo
	}
}

func WithBucketRepo(bRepo BucketRepo) func(*Service) {
	return func(s *Service) {
		s.bucketRepo = bRepo
	}
}

func WithEpubParser(eParser EpubParser) func(*Service) {
	return func(s *Service) {
		s.epubParser = eParser
	}
}

func NewService(options ...func(*Service)) *Service {
	srv := &Service{}
	for _, o := range options {
		o(srv)
	}
	return srv
}

func (s *Service) Search(ctx context.Context, query string) ([]*Book, error) {
	return s.bookAPI.SearchBookByString(query, 1)
}

func (s *Service) GetBookByKey(key string) (books *Book, err error) {
	return s.bookAPI.GetBookByKey(key)
}

func (s *Service) GetUploadBook(ctx context.Context, name string) (string, error) {
	return s.bucketRepo.GetBucketFileAsString(ctx, primitive.BooksBucket, "uploads/"+name)
}

func (s *Service) UploadNewBook(ctx context.Context, bookID, data string) error {
	if err := s.bucketRepo.UploadString(ctx, primitive.BooksBucket, "uploads/"+bookID, data, pbucket.EPUB); err != nil {
		return err
	}
	if err := s.queueRepo.PostMessage(bookID); err != nil {
		return err
	}
	if err := s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), Uploaded); err != nil {
		return err
	}
	return nil
}

func (s *Service) SaveBook(ctx context.Context, book *Book) (*Book, error) {
	return s.repo.CreateBook(ctx, book)
}

func (s *Service) GetSavedBookByKey(ctx context.Context, bookKey string) (*Book, error) {
	return s.repo.GetSavedBookByKey(ctx, bookKey)
}

func (s *Service) GetBookAsChunks(ctx context.Context, bookID string) (map[string]string, error) {
	bookContent, err := s.bucketRepo.GetBucketFileAsBytes(ctx, primitive.BooksBucket, "uploads/"+bookID)
	if err != nil {
		return nil, err
	}
	return s.epubParser.ExtractEPUB(bookContent)
}

func (s *Service) CreateAudioFromScript(ctx context.Context, bookID string) error {
	bookContent, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/script.txt", bookID))
	if err != nil {
		return err
	}
	audio, len, err := s.TTSAPI.CreateAudioFromString(ctx, bookContent)
	err = s.bucketRepo.UploadReader(ctx, primitive.AudioBucket, bookID, audio, len, pbucket.WAV)
	if err != nil {
		return err
	}
	return s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), ScriptGenerated)
}

func (s *Service) UploadBookChunks(ctx context.Context, bookID string, chunks map[string]string) error {
	for name, content := range chunks {
		if err := s.bucketRepo.UploadString(ctx, primitive.BooksBucket, "chunks/"+bookID+"/"+name, content, pbucket.TEXT); err != nil {
			log.Fatal(err)
		}
	}
	if err := s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), Parsed); err != nil {
		return err
	}
	if err := s.queueRepo.PostMessage(bookID); err != nil {
		return err
	}
	return nil
}

func (s *Service) GenerateChapterPreparation(ctx context.Context, count int, bookID, chunkName, prompt string) error {
	content, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.BooksBucket, chunkName)
	if err != nil {
		return err
	}

	output, err := s.aiAPI.Request(ctx, prompt+content)
	if err != nil {
		return err
	}

	if err := s.bucketRepo.UploadString(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/preparation/%d.txt", bookID, count), output, pbucket.TEXT); err != nil {
		return err
	}
	return nil
}

func (s *Service) MapOnChunks(ctx context.Context, bookID string, fn func(ctx context.Context, count int, bookID, chunkName, prompt string) error) error {
	promptPrepareChapter, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.NoneFictionPrepareChapter)
	if err != nil {
		return err
	}

	iter := s.bucketRepo.GetFilesIteratorOfDir(ctx, primitive.BooksBucket, "chunks/"+bookID+"/")

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	count := 0
	for i := range iter {
		c := count
		chunkKey := i.Key

		g.Go(func() error {
			return fn(ctx, c, bookID, chunkKey, promptPrepareChapter)
		})

		count++
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("map on chunks failed: %w", err)
	}

	if err := s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), Prepared); err != nil {
		return err
	}

	if err := s.queueRepo.PostMessage(bookID); err != nil {
		return err
	}

	return nil
}

func (s *Service) GenerateScript(ctx context.Context, bookID string) error {
	var builder strings.Builder
	iter := s.bucketRepo.GetFilesIteratorOfDir(ctx, primitive.ScriptsBucket, bookID+"/preparation/")
	for i := range iter {
		content, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.ScriptsBucket, i.Key)
		if err != nil {
			return err
		}
		if _, err := builder.WriteString(content); err != nil {
			return err
		}
	}
	result := builder.String()

	promptGenerateScript, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.NoneFictionGenerateScript)
	if err != nil {
		return err
	}

	output, err := s.aiAPI.Request(ctx, promptGenerateScript+result)
	if err != nil {
		return err
	}

	if err := s.bucketRepo.UploadString(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/script.txt", bookID), output, pbucket.TEXT); err != nil {
		return err
	}

	err = s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), ScriptGenerated)
	if err != nil {
		return err
	}

	return s.queueRepo.PostMessage(bookID)
}
