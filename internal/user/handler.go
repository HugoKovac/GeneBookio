package user

import (
	"context"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type Handler struct {
	validate *validator.Validate
	router   fiber.Router

	userService *Service
}

func NewHandler(router fiber.Router, userService *Service) {
	h := &Handler{
		validate: validator.New(validator.WithRequiredStructEnabled()),
		router:   router,

		userService: userService,
	}

	h.router.Get("/:id", h.Get)
}

func (h *Handler) Get(c fiber.Ctx) error {
	var (
		ctx context.Context = context.Background()
		uri UserIDURI
	)
	if err := c.Bind().URI(&uri); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := h.validate.Struct(&uri); err != nil {
		return errorwrapper.Wrap(err)
	}

	user, err := h.userService.GetByID(ctx, uuid.MustParse(uri.ID))
	if err != nil {
		return err
	}

	return c.JSON(user)
}
