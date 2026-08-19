package library

import (
	"context"
	"hkorpo/book/internal/book"
)

type LibraryAPI interface {
	SearchBookByString(q string, page int) (books []*book.Book, err error)
	GetBookByKey(key string) (books *book.Book, err error)
}

// Service looks up books on the external OpenLibrary catalog.
type Service struct {
	bookAPI LibraryAPI
}

func NewService(bookAPI LibraryAPI) *Service {
	return &Service{bookAPI: bookAPI}
}

func (s *Service) Search(ctx context.Context, query string) ([]*book.Book, error) {
	return s.bookAPI.SearchBookByString(query, 1)
}

func (s *Service) GetBookByKey(key string) (*book.Book, error) {
	return s.bookAPI.GetBookByKey(key)
}
