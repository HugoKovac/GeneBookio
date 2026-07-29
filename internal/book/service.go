package book

import (
	"context"
)

type LibraryAPI interface {
	SearchBookByString(q string, page int) (books []*Book, err error)
	GetBookByKey(key string) (books *Book, err error)
}

type Service struct {
	bookAPI LibraryAPI
}

func NewService(bookAPI LibraryAPI) *Service {
	return &Service{
		bookAPI: bookAPI,
	}
}

func (s *Service) Search(ctx context.Context, query string) ([]*Book, error) {
	return s.bookAPI.SearchBookByString(query, 1)
}

func (s *Service) GetBookByKey(key string) (books *Book, err error) {
	return s.bookAPI.GetBookByKey(key)
}
