package catalog

import (
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/primitive"
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

	h.router.Get("/",
		h.GetBooks,
	)

	h.router.Get("/audio/:query",
		h.GetAudioBook,
	)
}

// GetBooks returns the first page of saved books from the local database.
//
// @Summary      List saved books
// @Description  Returns up to 100 books saved in the local database
// @Tags         books
// @Security     BearerAuth
// @Produce      json
// @Success      200  {array}   book.BookDTO
// @Failure      500  {object}  map[string]string
// @Router       /books/ [get]
func (h *Handler) GetBooks(c fiber.Ctx) error {
	books, err := h.service.GetBooks(c.RequestCtx(), 0, 100)
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
func (h *Handler) GetAudioBook(c fiber.Ctx) error {
	var queryURI book.QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := h.validate.Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	audioReader, len, ctype, err := h.service.GetBucketObjectAsReader(c.RequestCtx(), primitive.AudioBucket, queryURI.Query)
	if err != nil {
		return err
	}

	c.Set("Content-Type", ctype)

	return c.SendStream(audioReader, int(len))
}
