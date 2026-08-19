package catalog

import (
	"context"
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/primitive"
	"io"
)

// Service manages saved book records and their generated audio.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo}
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
