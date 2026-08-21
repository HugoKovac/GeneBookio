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

// NewHandler mounts catalog's routes on router. enableRetry gates the
// admin-only POST /:query/retry route — it's only wired up by cmd/admin
// (see its Service's queueRepos), not the user-facing cmd/api, since
// retrying a pipeline stage isn't a regular-user action.
func NewHandler(router fiber.Router, service *Service, enableRetry bool) {
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

	if enableRetry {
		h.router.Post("/:query/retry",
			h.RetryFailedStage,
		)
	}
}

// GetBooks returns the first page of saved books from the local database,
// each annotated with its AI usage cost (see catalog.BookWithCost).
//
// @Summary      List saved books
// @Description  Returns up to 100 books saved in the local database
// @Tags         books
// @Security     BearerAuth
// @Produce      json
// @Success      200  {array}   catalog.BookWithCost
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

// RetryFailedStage re-queues a book from the pipeline stage it last failed
// at.
//
// @Summary      Retry a failed book
// @Description  Re-publishes the book onto the queue for the stage it last failed at, and clears the failure
// @Tags         books
// @Security     BearerAuth
// @Param        query  path  string  true  "Book ID (UUID)"
// @Success      204
// @Failure      400  {object}  map[string]string
// @Router       /books/{query}/retry [post]
func (h *Handler) RetryFailedStage(c fiber.Ctx) error {
	var queryURI book.QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := h.validate.Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := h.service.RetryFailedStage(c.RequestCtx(), queryURI.Query); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
