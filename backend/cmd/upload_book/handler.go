package main

import (
	"hkorpo/book/internal/catalog"
	"hkorpo/book/internal/library"
	"hkorpo/book/internal/upload"
	"hkorpo/book/pkg/ent"
	"io"
	"net/http"

	_ "embed"

	"github.com/gofiber/fiber/v3"
)

type uploadHandlers struct {
	router fiber.Router

	libraryService *library.Service
	catalogService *catalog.Service
	uploadService  *upload.Service
}

func newUploadHandlers(router fiber.Router, libraryService *library.Service, catalogService *catalog.Service, uploadService *upload.Service) {
	h := &uploadHandlers{
		router: router,

		libraryService: libraryService,
		catalogService: catalogService,
		uploadService:  uploadService,
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

func (uploadHandlers) UploadPage(c fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(uploadHTML)
}

func (h *uploadHandlers) Search(c fiber.Ctx) error {
	q := c.FormValue("q")

	if q == "" {
		return c.SendStatus(http.StatusUnprocessableEntity)
	}

	books, err := h.libraryService.Search(c.RequestCtx(), q)
	if err != nil {
		return err
	}

	return c.JSON(books)
}

func (h *uploadHandlers) Upload(c fiber.Ctx) error {
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

	bookData, err := h.catalogService.GetSavedBookByKey(c.RequestCtx(), c.FormValue("book_key"))
	if err != nil {
		if !ent.IsNotFound(err) {
			return c.Status(http.StatusConflict).SendString("already uploaded")
		} else if bookData != nil && bookData.Uploaded == true {
			return c.Status(http.StatusConflict).SendString("already uploaded")
		}
	}

	// todo: handle book exists but not uploaded

	bookData, err = h.libraryService.GetBookByKey(c.FormValue("book_key"))
	if err != nil {
		return err
	}

	bookData, err = h.catalogService.SaveBook(c.RequestCtx(), bookData)
	if err != nil {
		return err
	}

	if err := h.uploadService.UploadNewBook(c.RequestCtx(), bookData.ID.String(), string(data)); err != nil {
		return err
	}

	return c.SendStatus(http.StatusCreated)
}
