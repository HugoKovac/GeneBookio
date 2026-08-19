package main

import (
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/catalog"
	"hkorpo/book/internal/library"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/internal/user"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type bookHandlers struct {
	validate *validator.Validate
	router   fiber.Router

	libraryService *library.Service
	catalogService *catalog.Service
}

func newBookHandlers(router fiber.Router, libraryService *library.Service, catalogService *catalog.Service, userService *user.Service) {
	h := &bookHandlers{
		validate: validator.New(validator.WithRequiredStructEnabled()),
		router:   router,

		libraryService: libraryService,
		catalogService: catalogService,
	}

	h.router.Use(user.MiddlewareAuth(userService))

	h.router.Get("/search/:query",
		h.Search,
	)

	h.router.Get("/:query",
		h.GetBookByKey,
	)

	h.router.Get("/",
		h.GetBooks,
	)

	h.router.Get("/audio/:query",
		h.GetAudioBook,
	)

}

// Search searches books on OpenLibrary by keyword or title.
//
// @Summary      Search books
// @Description  Search for books on OpenLibrary by title or keyword
// @Tags         books
// @Security     BearerAuth
// @Produce      json
// @Param        query  path  string  true  "Search query"
// @Success      200  {array}   book.BookDTO
// @Failure      400  {object}  map[string]string
// @Router       /books/search/{query} [get]
func (h *bookHandlers) Search(c fiber.Ctx) error {
	var queryURI book.QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := validator.New().Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	books, err := h.libraryService.Search(c.RequestCtx(), queryURI.Query)
	if err != nil {
		return err
	}

	var booksDTO []*book.BookDTO
	for _, b := range books {

		booksDTO = append(booksDTO, &book.BookDTO{
			Title:       b.Title,
			AuthorNames: b.AuthorNames,
			CoverURL:    b.CoverURL,
			Key:         b.Key,
		})
	}

	return c.JSON(books)
}

// GetBookByKey fetches a single book from OpenLibrary by its key.
//
// @Summary      Get book by key
// @Description  Fetch a book from OpenLibrary by its key (e.g. /works/OL12345W)
// @Tags         books
// @Security     BearerAuth
// @Produce      json
// @Param        query  path  string  true  "OpenLibrary book key"
// @Success      200  {object}  book.BookDTO
// @Failure      400  {object}  map[string]string
// @Router       /books/{query} [get]
func (h *bookHandlers) GetBookByKey(c fiber.Ctx) error {
	var queryURI book.QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := validator.New().Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	b, err := h.libraryService.GetBookByKey(queryURI.Query)
	if err != nil {
		return err
	}

	return c.JSON(book.BookDTO{
		Title:       b.Title,
		Authors:     b.AuthorKeys,
		CoverURL:    b.CoverURL,
		Key:         b.Key,
		Descriptiom: b.Description,
	})
}

// GetBooks returns the first page of saved books from the local database.
//
// @Summary      List saved books
// @Description  Returns up to 5 books saved in the local database
// @Tags         books
// @Security     BearerAuth
// @Produce      json
// @Success      200  {array}   book.BookDTO
// @Failure      500  {object}  map[string]string
// @Router       /books/ [get]
func (h *bookHandlers) GetBooks(c fiber.Ctx) error {
	books, err := h.catalogService.GetBooks(c.RequestCtx(), 0, 5)
	if err != nil {
		return err
	}
	return c.JSON(books)
}

// GetAudioBook streams the TTS-synthesized audio for a book.
//
// @Summary      Stream audio book
// @Description  Streams the synthesized audio file for the given book ID from MinIO
// @Tags         books
// @Security     BearerAuth
// @Produce      octet-stream
// @Param        query  path  string  true  "Book ID (UUID)"
// @Success      200  {file}    binary
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /books/audio/{query} [get]
func (h *bookHandlers) GetAudioBook(c fiber.Ctx) error {
	var queryURI book.QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := validator.New().Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	audioReader, len, ctype, err := h.catalogService.GetBucketObjectAsReader(c.RequestCtx(), primitive.AudioBucket, queryURI.Query)
	if err != nil {
		return err
	}

	c.Set("Content-Type", ctype)

	return c.SendStream(audioReader, int(len))
}
