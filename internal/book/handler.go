package book

import (
	"hkorpo/book/pkg/errorwrapper"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	validate *validator.Validate
	router   fiber.Router

	bookService *Service
}

func NewHandler(router fiber.Router, bookService *Service) {
	h := &Handler{
		validate: validator.New(validator.WithRequiredStructEnabled()),
		router:   router,

		bookService: bookService,
	}

	h.router.Get("/search/:query",
		h.Search,
	)

	h.router.Get("/:query",
		h.GetBookByKey,
	)
}

func (h *Handler) Search(c fiber.Ctx) error {
	var queryURI QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := validator.New().Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	books, err := h.bookService.Search(c.RequestCtx(), queryURI.Query)
	if err != nil {
		return err
	}

	var booksDTO []*BookDTO
	for _, b := range books {

		booksDTO = append(booksDTO, &BookDTO{
			Title:       b.Title,
			AuthorNames: b.AuthorNames,
			CoverURL:    b.CoverURL,
			Key:         b.Key,
		})
	}

	return c.JSON(books)
}

func (h *Handler) GetBookByKey(c fiber.Ctx) error {
	var queryURI QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := validator.New().Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	b, err := h.bookService.GetBookByKey(queryURI.Query)
	if err != nil {
		return err
	}

	return c.JSON(BookDTO{
		Title:       b.Title,
		Authors:     b.AuthorIDs,
		CoverURL:    b.CoverURL,
		Key:         b.Key,
		Descriptiom: b.Description,
	})
}
