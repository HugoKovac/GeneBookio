package catalog

import (
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"
	"strconv"
	"strings"

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
// retrying a pipeline stage isn't a regular-user action. requireActiveSubscription,
// when non-nil, gates audio streaming behind a paid subscription — cmd/admin
// passes nil since it has no auth/subscription concept at all.
func NewHandler(router fiber.Router, service *Service, enableRetry bool, requireActiveSubscription fiber.Handler) {
	h := &Handler{
		validate: validator.New(validator.WithRequiredStructEnabled()),
		router:   router,

		service: service,
	}

	h.router.Get("/",
		h.GetBooks,
	)

	if requireActiveSubscription != nil {
		h.router.Get("/audio/:query",
			requireActiveSubscription,
			h.GetAudioBook,
		)
	} else {
		h.router.Get("/audio/:query",
			h.GetAudioBook,
		)
	}

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

// GetAudioBook streams the TTS-synthesized audio for a book. It honors the
// Range header (single "bytes=start-end" range only) so a native <audio>
// element can play it back and seek without downloading the whole file
// first.
//
// @Summary      Stream audio book
// @Description  Streams the synthesized audio file for the given book ID from MinIO
// @Tags         books
// @Security     BearerAuth
// @Produce      octet-stream
// @Param        query  path  string  true  "Book ID (UUID)"
// @Success      200  {file}    binary
// @Success      206  {file}    binary
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      416  {object}  map[string]string
// @Router       /books/audio/{query} [get]
func (h *Handler) GetAudioBook(c fiber.Ctx) error {
	var queryURI book.QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := h.validate.Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	rangeStart, rangeEnd := int64(-1), int64(-1)
	if rangeHeader := c.Get(fiber.HeaderRange); rangeHeader != "" {
		start, end, ok := parseRangeHeader(rangeHeader)
		if !ok {
			return c.SendStatus(fiber.StatusRequestedRangeNotSatisfiable)
		}
		rangeStart, rangeEnd = start, end
	}

	audioReader, size, totalSize, ctype, err := h.service.GetBucketObjectAsReader(c.RequestCtx(), primitive.AudioBucket, queryURI.Query, rangeStart, rangeEnd)
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, ctype)
	c.Set(fiber.HeaderAcceptRanges, "bytes")

	if rangeStart >= 0 {
		end := rangeStart + size - 1
		c.Set(fiber.HeaderContentRange, "bytes "+strconv.FormatInt(rangeStart, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(totalSize, 10))
		c.Status(fiber.StatusPartialContent)
	}

	return c.SendStream(audioReader, int(size))
}

// parseRangeHeader parses a single "bytes=start-end" or "bytes=start-" Range
// header. Multi-range ("bytes=0-10,20-30") and suffix-length ("bytes=-500")
// requests aren't supported and report ok=false — audio elements don't send
// either form in practice, they only ever request from a start offset.
func parseRangeHeader(header string) (start, end int64, ok bool) {
	spec, found := strings.CutPrefix(header, "bytes=")
	if !found {
		return 0, 0, false
	}

	if strings.Contains(spec, ",") {
		return 0, 0, false
	}

	before, after, found := strings.Cut(spec, "-")
	if !found || before == "" {
		return 0, 0, false
	}

	start, err := strconv.ParseInt(before, 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false
	}

	if after == "" {
		return start, 0, true
	}

	end, err = strconv.ParseInt(after, 10, 64)
	if err != nil || end < start {
		return 0, 0, false
	}

	return start, end, true
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
