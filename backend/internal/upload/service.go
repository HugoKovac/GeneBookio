package upload

import (
	"context"
	"hkorpo/book/internal/book"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"

	"github.com/google/uuid"
)

// Service accepts a raw EPUB, stores it, and kicks off the pipeline.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	queueRepo  book.QueueRepo
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, queueRepo book.QueueRepo) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, queueRepo: queueRepo}
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
	if err := s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), book.Uploaded); err != nil {
		return err
	}
	return nil
}
