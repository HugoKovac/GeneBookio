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

func (h *Handlers) GetBooks(c fiber.Ctx) error {
	books, err := h.bookService.GetBooks(c.RequestCtx(), 0, 5)
	if err != nil {
		return err
	}
	return c.JSON(books)
}

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
