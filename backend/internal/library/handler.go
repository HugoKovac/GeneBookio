package library

import (
	"hkorpo/book/internal/book"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	validate *validator.Validate
	router   fiber.Router

	service *Service
}

func NewHandler(router fiber.Router, service *Service) {
	h := &Handler{
		validate: validator.New(validator.WithRequiredStructEnabled()),
		router:   router,

		service: service,
	}

	h.router.Get("/search/:query",
		h.Search,
	)

	h.router.Get("/:query",
		h.GetBookByKey,
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
func (h *Handler) Search(c fiber.Ctx) error {
	var queryURI book.QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := h.validate.Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	books, err := h.service.Search(c.RequestCtx(), queryURI.Query)
	if err != nil {
		return err
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
func (h *Handler) GetBookByKey(c fiber.Ctx) error {
	var queryURI book.QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := h.validate.Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	b, err := h.service.GetBookByKey(queryURI.Query)
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
