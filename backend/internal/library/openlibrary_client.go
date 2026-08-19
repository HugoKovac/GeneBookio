package library

import (
	"encoding/json"
	"fmt"
	"hkorpo/book/internal/book"
	"hkorpo/book/pkg/errorwrapper"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
)

// https://openlibrary.org/dev/docs/api/search
// https://openlibrary.org/dev/docs/api/covers
// https://libgen.li/json.php
// https://libgen.li/index.php

type bookAuthor struct {
	Author struct {
		Key string `json:"key"`
	} `json:"author"`
}

type docsSearchAPIResponse struct {
	AuthorNames []string `json:"author_name"`
	Languages   []string `json:"language"`
	Title       string   `json:"title"`
	CoverID     int      `json:"cover_i"`
	Key         string   `json:"key"`
}

type searchAPIResponse struct {
	Docs []docsSearchAPIResponse `json:"docs"`
}

type descriptionWorksApiResponse struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type worksApiResponse struct {
	Description      descriptionWorksApiResponse `json:"description"`
	Title            string                      `json:"title"`
	Key              string                      `json:"key"`
	FirstPublishDate string                      `json:"first_publish_date"`
	Covers           []int                       `json:"covers"`
	Authors          []bookAuthor
}

type LibraryClient struct {
	client *http.Client
}

func NewOpenLibraryClient() *LibraryClient {
	return &LibraryClient{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *LibraryClient) SearchBookByString(q string, page int) (books []*book.Book, err error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://openlibrary.org/search.json?q=%s&limit=5&page=%d", strings.ReplaceAll(html.EscapeString(q), " ", "%20"), page), nil)
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
		return nil, errorwrapper.Wrap(fmt.Errorf("Error reading response body: %w", err))
	}

	var parsedBody searchAPIResponse

	if err := json.Unmarshal(body, &parsedBody); err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	for _, b := range parsedBody.Docs {
		split := strings.Split(b.Key, "/")

		if len(split) != 3 {
			return nil, errorwrapper.Wrap(fmt.Errorf("can't parse following book key: %s", b.Key))
		}

		books = append(books, &book.Book{
			Key:         split[2],
			Title:       b.Title,
			AuthorNames: b.AuthorNames,
			CoverURL:    fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg\n", b.CoverID),
		})

	}
	return books, nil
}

func authorsToStringSlice(ba []bookAuthor) (as []string) {
	for _, a := range ba {
		as = append(as, a.Author.Key)
	}
	return
}

func (c *LibraryClient) GetBookByKey(key string) (books *book.Book, err error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://openlibrary.org/works/%s.json", key), nil)
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

	var parsedBody worksApiResponse
	if err := json.Unmarshal(body, &parsedBody); err != nil {
		return nil, err
	}

	keySplit := strings.Split(parsedBody.Key, "/")
	if len(keySplit) != 3 {
		return nil, fmt.Errorf("can't parse following book key: %s", parsedBody.Key)
	}

	var coverID int = -1
	if len(parsedBody.Covers) > 0 {
		coverID = parsedBody.Covers[0]
	}

	for i := range parsedBody.Authors {
		authorSplit := strings.Split(parsedBody.Authors[i].Author.Key, "/")
		if len(authorSplit) != 3 {
			return nil, fmt.Errorf("can't parse following author key: %s", parsedBody.Authors[i].Author.Key)
		}
		parsedBody.Authors[i].Author.Key = authorSplit[2]
	}

	return &book.Book{
		Key:         keySplit[2],
		Title:       parsedBody.Title,
		AuthorKeys:  authorsToStringSlice(parsedBody.Authors),
		CoverURL:    fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg\n", coverID),
		Description: parsedBody.Description.Value,
	}, nil
}
