package book

import (
	"context"
	"hkorpo/book/internal/primitive"
	"log"
	"strings"
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
	GetBucketFileAsBytes(ctx context.Context, bucket primitive.Bucket, path string) ([]byte, error)
}

type EpubParser interface {
	ExtractEPUB(epubContent []byte) (map[string]string, error)
}

type Service struct {
	bookAPI    LibraryAPI
	queueRepo  QueueRepo
	bucketRepo BucketRepo
	epubParser EpubParser
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

func (s *Service) UploadNewBook(ctx context.Context, name, data string) error {
	if err := s.bucketRepo.UploadStringAsTextFile(ctx, primitive.BooksBucket, "uploads/"+name, data); err != nil {
		return err
	}
	if err := s.queueRepo.PostMessage(name); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetBookAsChunks(ctx context.Context, fileName string) (map[string]string, error) {
	bookContent, err := s.bucketRepo.GetBucketFileAsBytes(ctx, primitive.BooksBucket, "uploads/"+fileName)
	if err != nil {
		return nil, err
	}
	return s.epubParser.ExtractEPUB(bookContent)
}

func (s *Service) UploadBookChunks(ctx context.Context, fileName string, chunks map[string]string) error {
	bookName := strings.TrimSuffix(fileName, ".epub")
	for name, content := range chunks {
		if err := s.bucketRepo.UploadStringAsTextFile(ctx, primitive.BooksBucket, "chunks/"+bookName+"/"+name, content); err != nil {
			log.Fatal(err)
		}
	}
	if err := s.queueRepo.PostMessage(bookName); err != nil {
		return err
	}
	return nil
}
