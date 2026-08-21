package preparation

import (
	"context"
	"fmt"
	"hkorpo/book/internal/book"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type AiAPI interface {
	Request(ctx context.Context, request string) (string, primitive.ModelUsage, error)
	ModelName() string
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

func (s *Service) GenerateChapterPreparation(ctx context.Context, count int, bookID, chunkName, prompt string) (primitive.ModelUsage, error) {
	content, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.BooksBucket, chunkName)
	if err != nil {
		return primitive.ModelUsage{}, err
	}

	output, usage, err := s.aiAPI.Request(ctx, prompt+content)
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

	if err := s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), book.Prepared); err != nil {
		return err
	}

	if err := s.queueRepo.PostMessage(bookID); err != nil {
		return err
	}

	return nil
}
