package preparation

import (
	"context"
	"fmt"
	"hkorpo/book/internal/book"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type AiAPI interface {
	Request(ctx context.Context, request string) (string, error)
}

// Service AI-preprocesses each chapter chunk of a book in parallel.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	queueRepo  book.QueueRepo
	aiAPI      AiAPI
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, queueRepo book.QueueRepo, aiAPI AiAPI) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, queueRepo: queueRepo, aiAPI: aiAPI}
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

	if err := s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), book.Prepared); err != nil {
		return err
	}

	if err := s.queueRepo.PostMessage(bookID); err != nil {
		return err
	}

	return nil
}
