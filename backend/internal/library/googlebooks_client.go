package library

import (
	"encoding/json"
	"fmt"
	"hkorpo/book/internal/book"
	"hkorpo/book/pkg/errorwrapper"
	"io"
	"net/http"
	"net/url"
	"time"
)

// https://developers.google.com/books/docs/v1/using

type ConfigGoogleBooks struct {
	APIKey string `envconfig:"GOOGLE_API_BOOKS"`
}

type volumeInfo struct {
	Title       string   `json:"title"`
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	ImageLinks  struct {
		Thumbnail string `json:"thumbnail"`
	} `json:"imageLinks"`
}

type volumeItem struct {
	ID         string     `json:"id"`
	VolumeInfo volumeInfo `json:"volumeInfo"`
}

type volumesSearchResponse struct {
	Items []volumeItem `json:"items"`
}

type GoogleBooksClient struct {
	client *http.Client
	apiKey string
}

func NewGoogleBooksClient(apiKey string) *GoogleBooksClient {
	return &GoogleBooksClient{
		client: &http.Client{Timeout: time.Second * 10},
		apiKey: apiKey,
	}
}

func (c *GoogleBooksClient) SearchBookByString(q string, page int) (books []*book.Book, err error) {
	const limit = 5
	startIndex := max((page-1)*limit, 0)

	reqURL := fmt.Sprintf(
		"https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=%d&startIndex=%d&key=%s",
		url.QueryEscape(q), limit, startIndex, url.QueryEscape(c.apiKey),
	)

	body, err := c.get(reqURL)
	if err != nil {
		return nil, err
	}

	var parsed volumesSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	for _, item := range parsed.Items {
		books = append(books, volumeToBook(item))
	}
	return books, nil
}

func (c *GoogleBooksClient) GetBookByKey(key string) (*book.Book, error) {
	reqURL := fmt.Sprintf(
		"https://www.googleapis.com/books/v1/volumes/%s?key=%s",
		url.PathEscape(key), url.QueryEscape(c.apiKey),
	)

	body, err := c.get(reqURL)
	if err != nil {
		return nil, err
	}

	var parsed volumeItem
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	return volumeToBook(parsed), nil
}

// The public Google Books API intermittently returns 5xx errors under normal
// load, so a request is retried a couple of times before giving up.
const googleBooksMaxAttempts = 3

func (c *GoogleBooksClient) get(reqURL string) ([]byte, error) {
	var lastErr error

	for attempt := 1; attempt <= googleBooksMaxAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(300 * time.Millisecond)
		}

		body, status, err := c.doGet(reqURL)
		if err == nil && status == http.StatusOK {
			return body, nil
		}

		if err != nil {
			lastErr = errorwrapper.Wrap(err)
			continue
		}
		lastErr = errorwrapper.Wrap(fmt.Errorf("google books api error: %d: %s", status, string(body)))
	}

	return nil, lastErr
}

func (c *GoogleBooksClient) doGet(reqURL string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return body, resp.StatusCode, nil
}

func volumeToBook(item volumeItem) *book.Book {
	return &book.Book{
		Key:         item.ID,
		Title:       item.VolumeInfo.Title,
		AuthorNames: item.VolumeInfo.Authors,
		CoverURL:    item.VolumeInfo.ImageLinks.Thumbnail,
		Description: item.VolumeInfo.Description,
	}
}
