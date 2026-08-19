package library_test

import (
	"fmt"
	"hkorpo/book/internal/library"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSearchBookByString(t *testing.T) {
	client := library.NewOpenLibraryClient()
	n := time.Now()
	books, err := client.SearchBookByString("prince", 1)
	log.Println("time: ", time.Since(n))
	assert.Nil(t, err)
	assert.Equal(t, 5, len(books))
}

func TestGetBookByKey(t *testing.T) {
	client := library.NewOpenLibraryClient()
	n := time.Now()
	b, err := client.GetBookByKey("OL10263W")
	log.Println("time: ", time.Since(n))
	assert.Nil(t, err)
	fmt.Println("> ", b)
}
