package book

import (
	"context"
)

type LibraryAPI interface {
	SearchBookByString(q string, page int) (books []*Book, err error)
	GetBookByKey(key string) (books *Book, err error)
}

type QueueRepo interface {
	PostMessage(msg string) error
}

type Service struct {
	bookAPI   LibraryAPI
	queueRepo QueueRepo
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
