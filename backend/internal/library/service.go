package library

import (
	"context"
	"hkorpo/book/internal/book"
)

// BooksAPI is the port to an external book catalog — GoogleBooksClient is
// its only implementation.
type BooksAPI interface {
	SearchBookByString(q string, page int) (books []*book.Book, err error)
	GetBookByKey(key string) (books *book.Book, err error)
}

// Service looks up books on an external catalog (see BooksAPI).
type Service struct {
	booksAPI BooksAPI
}

func NewService(booksAPI BooksAPI) *Service {
	return &Service{booksAPI: booksAPI}
}

func (s *Service) Search(ctx context.Context, query string) ([]*book.Book, error) {
	return s.booksAPI.SearchBookByString(query, 1)
}

func (s *Service) GetBookByKey(key string) (*book.Book, error) {
	return s.booksAPI.GetBookByKey(key)
}
