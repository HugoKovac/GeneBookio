package upload

import (
	"hkorpo/book/internal/catalog"
	"hkorpo/book/internal/library"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/internal/user"
	"hkorpo/book/pkg/ent"
	"io"
	"net/http"
	"slices"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	router fiber.Router

	libraryService *library.Service
	catalogService *catalog.Service
	userService    *user.Service
	service        *Service
}

func NewHandler(router fiber.Router, libraryService *library.Service, catalogService *catalog.Service, service *Service, userService *user.Service) {
	h := &Handler{
		router: router,

		libraryService: libraryService,
		catalogService: catalogService,
		userService:    userService,
		service:        service,
	}

	h.router.Post("/upload",
		h.Upload,
	)

	h.router.Get("/search",
		h.Search,
	)
}

func (h *Handler) Search(c fiber.Ctx) error {
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

func (h *Handler) Upload(c fiber.Ctx) error {
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

	language := primitive.Language(c.FormValue("language", string(primitive.French)))
	if !slices.Contains(language.Values(), string(language)) {
		return c.Status(http.StatusUnprocessableEntity).SendString("invalid language")
	}
	bookData.Language = language

	genre := primitive.Genre(c.FormValue("genre", string(primitive.NoneFiction)))
	if !slices.Contains(genre.Values(), string(genre)) {
		return c.Status(http.StatusUnprocessableEntity).SendString("invalid genre")
	}
	bookData.Genre = genre

	bookData, err = h.catalogService.SaveBook(c.RequestCtx(), bookData)
	if err != nil {
		return err
	}

	if err := h.service.UploadNewBook(c.RequestCtx(), bookData.ID.String(), string(data)); err != nil {
		return err
	}

	return c.SendStatus(http.StatusCreated)
}
