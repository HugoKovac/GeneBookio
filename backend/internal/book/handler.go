package book

import (
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/ent"
	"hkorpo/book/pkg/errorwrapper"
	"io"
	"net/http"

	_ "embed"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type Handlers struct {
	validate *validator.Validate
	router   fiber.Router

	bookService *Service
}

func NewHandlers(router fiber.Router, bookService *Service) {
	h := &Handlers{
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
// @Produce      json
// @Param        query  path  string  true  "Search query"
// @Success      200  {array}   BookDTO
// @Failure      400  {object}  map[string]string
// @Router       /books/search/{query} [get]
func (h *Handlers) Search(c fiber.Ctx) error {
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

// GetBookByKey fetches a single book from OpenLibrary by its key.
//
// @Summary      Get book by key
// @Description  Fetch a book from OpenLibrary by its key (e.g. /works/OL12345W)
// @Tags         books
// @Produce      json
// @Param        query  path  string  true  "OpenLibrary book key"
// @Success      200  {object}  BookDTO
// @Failure      400  {object}  map[string]string
// @Router       /books/{query} [get]
func (h *Handlers) GetBookByKey(c fiber.Ctx) error {
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
// @Produce      json
// @Success      200  {array}   BookDTO
// @Failure      500  {object}  map[string]string
// @Router       /books/ [get]
func (h *Handlers) GetBooks(c fiber.Ctx) error {
	books, err := h.bookService.GetBooks(c.RequestCtx(), 0, 5)
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
// @Produce      octet-stream
// @Param        query  path  string  true  "Book ID (UUID)"
// @Success      200  {file}    binary
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /books/audio/{query} [get]
func (h *Handlers) GetAudioBook(c fiber.Ctx) error {
	var queryURI QueryURI

	if err := c.Bind().URI(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := validator.New().Struct(&queryURI); err != nil {
		return errorwrapper.Wrap(err)
	}

	audioReader, len, ctype, err := h.bookService.GetBucketObjectAsReader(c.RequestCtx(), primitive.AudioBucket, queryURI.Query)
	if err != nil {
		return err
	}

	c.Set("Content-Type", ctype)

	return c.SendStream(audioReader, int(len))
}

type UploadHandlers struct {
	router fiber.Router

	bookService *Service
}

func NewUploadHandlers(router fiber.Router, bookService *Service) {
	h := &UploadHandlers{
		router: router,

		bookService: bookService,
	}

	h.router.Get("/",
		h.UploadPage,
	)

	h.router.Post("/upload",
		h.Upload,
	)

	h.router.Get("/search",
		h.Search,
	)
}

//go:embed  html/upload.html
var uploadHTML string

func (UploadHandlers) UploadPage(c fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(uploadHTML)
}

func (uh *UploadHandlers) Search(c fiber.Ctx) error {
	q := c.FormValue("q")

	if q == "" {
		return c.SendStatus(http.StatusUnprocessableEntity)
	}

	books, err := uh.bookService.Search(c.RequestCtx(), q)
	if err != nil {
		return err
	}

	return c.JSON(books)
}

func (uh *UploadHandlers) Upload(c fiber.Ctx) error {
	file, err := c.FormFile("epub")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("no file received (" + err.Error() + ")")
	}

	// todo: verify epub magic number

	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	bookData, err := uh.bookService.GetSavedBookByKey(c.RequestCtx(), c.FormValue("book_key"))
	if err != nil {
		if !ent.IsNotFound(err) {
			return c.Status(http.StatusConflict).SendString("already uploaded")
		} else if bookData != nil && bookData.Uploaded == true {
			return c.Status(http.StatusConflict).SendString("already uploaded")
		}
	}

	// todo: handle book exists but not uploaded

	bookData, err = uh.bookService.GetBookByKey(c.FormValue("book_key"))
	if err != nil {
		return err
	}

	bookData, err = uh.bookService.SaveBook(c.RequestCtx(), bookData)
	if err != nil {
		return err
	}

	if err := uh.bookService.UploadNewBook(c.RequestCtx(), bookData.ID.String(), string(data)); err != nil {
		return err
	}

	return c.SendStatus(http.StatusCreated)
}
