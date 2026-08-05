package book

import (
	"hkorpo/book/pkg/errorwrapper"
	"io"
	"net/http"
	"path/filepath"
	"strings"

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
		Authors:     b.AuthorIDs,
		CoverURL:    b.CoverURL,
		Key:         b.Key,
		Descriptiom: b.Description,
	})
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
}

func (UploadHandlers) UploadPage(c fiber.Ctx) error {
	html := `
		<!DOCTYPE html>
		<html">
		<head>
			<meta charset="UTF-8">
			<title>Upload EPUB</title>
		</head>
		<body>
			<h1>Upload .epub files</h1>
			<form action="/upload" method="POST" enctype="multipart/form-data">
				<input type="file" name="epub" accept=".epub" required>
				<button type="submit">Upload</button>
			</form>
		</body>
		</html>
	`
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html)
}

func (uh *UploadHandlers) Upload(c fiber.Ctx) error {
	file, err := c.FormFile("epub")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("no file received (" + err.Error() + ")")
	}

	if strings.ToLower(filepath.Ext(file.Filename)) != ".epub" {
		return c.Status(fiber.StatusBadRequest).SendString("file extension should be .epub")
	}

	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	b, err := uh.bookService.GetUploadBook(file.Filename)
	if err != nil {
		if err.Error() != "The specified key does not exist." {
			return err
		}
	}

	if b != "" {
		return errorwrapper.Wrap("already exists")
	}

	if err := uh.bookService.UploadNewBook(file.Filename, string(data)); err != nil {
		return err
	}

	return c.SendStatus(http.StatusCreated)
}
