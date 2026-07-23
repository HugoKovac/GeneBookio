package book

import (
	"encoding/json"
	"fmt"
	"hkorpo/book/pkg/errorwrapper"
	"io"
	"net/http"
	"time"
)

type OpenLibraryClient struct {
	client *http.Client
}

func NewOpenLibraryClient() *OpenLibraryClient {
	return &OpenLibraryClient{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *OpenLibraryClient) GetBookByID(q string, page int) (books []*Book, err error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://openlibrary.org/search.json?q=%s&limit=5&page=%d", q, page), nil)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	defer resp.Body.Close()

	// Read and print response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %w", err)
	}

	var parsedBody SearchAPIResponse

	if err := json.Unmarshal(body, &parsedBody); err != nil {
		return nil, err
	}

	// fmt.Println(string(body))

	for _, b := range parsedBody.Docs {
		books = append(books, &Book{
			Title:       b.Title,
			AuthorNames: b.AuthorNames,
			CoverURL:    fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg\n", b.CoverID),
		})

	}
	return books, nil
}
