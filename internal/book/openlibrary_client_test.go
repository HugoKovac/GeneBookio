package book_test

import (
	"fmt"
	"hkorpo/book/internal/book"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSearchBookByString(t *testing.T) {
	client := book.NewOpenLibraryClient()
	n := time.Now()
	books, err := client.SearchBookByString("prince", 1)
	log.Println("time: ", time.Since(n))
	assert.Nil(t, err)
	assert.Equal(t, 5, len(books))
}

func TestGetBookByKey(t *testing.T) {
	client := book.NewOpenLibraryClient()
	n := time.Now()
	book, err := client.GetBookByKey("OL10263W")
	log.Println("time: ", time.Since(n))
	assert.Nil(t, err)
	fmt.Println("> ", book)
}
