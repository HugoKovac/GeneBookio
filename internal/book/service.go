package book

import (
	"context"
	"hkorpo/book/internal/primitive"
)

type LibraryAPI interface {
	SearchBookByString(q string, page int) (books []*Book, err error)
	GetBookByKey(key string) (books *Book, err error)
}

type QueueRepo interface {
	PostMessage(msg string) error
}

type BucketRepo interface {
	UploadStringAsTextFile(ctx context.Context, bucketName primitive.Bucket, path, content string) error
	GetBucketFileAsString(ctx context.Context, bucket primitive.Bucket, path string) (string, error)
}

type Service struct {
	bookAPI    LibraryAPI
	queueRepo  QueueRepo
	bucketRepo BucketRepo
}

func WithLibraryAPI(lAPI LibraryAPI) func(*Service) {
	return func(s *Service) {
		s.bookAPI = lAPI
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

func (s *Service) GetUploadBook(name string) (string, error) {
	return s.bucketRepo.GetBucketFileAsString(context.Background(), primitive.BooksBucket, "uploads/"+name)
}

func (s *Service) UploadNewBook(name, data string) error {
	if err := s.bucketRepo.UploadStringAsTextFile(context.Background(), primitive.BooksBucket, "uploads/"+name, data); err != nil {
		return err
	}
	if err := s.queueRepo.PostMessage("uploads/" + name); err != nil {
		return err
	}
	return nil
}
