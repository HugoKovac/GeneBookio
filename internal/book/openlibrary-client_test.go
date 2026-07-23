package book_test

import (
	"hkorpo/book/internal/book"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBookByID(t *testing.T) {
	client := book.NewOpenLibraryClient()
	books, err := client.GetBookByID("prince", 1)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(books))
}
