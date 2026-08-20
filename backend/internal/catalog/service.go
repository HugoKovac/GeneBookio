package catalog

import (
	"context"
	"fmt"
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"
	"io"

	"github.com/google/uuid"
)

// Service manages saved book records and their generated audio.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	// queueRepos publishes a retry message onto the queue for a given failed
	// stage (keyed by primitive.QueueChannel, e.g. "prepare"), so a failed
	// book can resume from wherever it stopped instead of from the start.
	queueRepos map[string]book.QueueRepo
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, queueRepos map[string]book.QueueRepo) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, queueRepos: queueRepos}
}

func (s *Service) SaveBook(ctx context.Context, b *book.Book) (*book.Book, error) {
	return s.repo.CreateBook(ctx, b)
}

func (s *Service) GetSavedBookByKey(ctx context.Context, bookKey string) (*book.Book, error) {
	return s.repo.GetSavedBookByKey(ctx, bookKey)
}

func (s *Service) GetBooks(ctx context.Context, page, limit int) ([]*book.Book, error) {
	return s.repo.GetBooks(ctx, page, limit)
}

func (s *Service) GetBucketObjectAsReader(ctx context.Context, bucket primitive.Bucket, path string) (io.Reader, int64, string, error) {
	return s.bucketRepo.GetBucketObjectAsReader(ctx, bucket, path)
}

// RetryFailedStage re-publishes bookID onto the queue for the stage it last
// failed at, so the pipeline resumes from there instead of from the start.
func (s *Service) RetryFailedStage(ctx context.Context, bookID string) error {
	id, err := uuid.Parse(bookID)
	if err != nil {
		return errorwrapper.Wrap(err)
	}

	b, err := s.repo.GetBookByID(ctx, id)
	if err != nil {
		return err
	}

	if !b.Failed {
		return errorwrapper.Wrap(fmt.Errorf("book %s has no failed stage to retry", bookID))
	}

	queueRepo, ok := s.queueRepos[b.FailedStage]
	if !ok {
		return errorwrapper.Wrap(fmt.Errorf("unknown failed stage %q", b.FailedStage))
	}

	if err := queueRepo.PostMessage(bookID); err != nil {
		return err
	}

	return s.repo.ClearBookFailure(ctx, id)
}
