package book

import (
	"context"
	"fmt"
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

func (s *Service) UploadNewBook(newBook string) error {
	fmt.Println(s)
	fmt.Println(s.bucketRepo)
	fmt.Println(s.queueRepo)
	if err := s.bucketRepo.UploadStringAsTextFile(context.Background(), primitive.BooksBucket, "uploads/test.txt", newBook); err != nil {
		return err
	}
	if err := s.queueRepo.PostMessage("newbookXXX"); err != nil {

	}
	return nil
}
