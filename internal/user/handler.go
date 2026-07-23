package user

import (
	"hkorpo/book/pkg/errorwrapper"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
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

	h.router.Get("/:id",
		MiddlewareUserExists(userService),
		h.Get,
	)

	h.router.Post("/",
		h.Register,
	)

}

func (h *Handler) Get(c fiber.Ctx) error {
	return c.JSON(c.Locals("user").(*User))
}

func (h *Handler) Register(c fiber.Ctx) error {
	var body RegisterRequestDTO

	if err := c.Bind().Body(&body); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := validator.New().Struct(&body); err != nil {
		return errorwrapper.Wrap(err)
	}

	createdUser, err := h.userService.Create(c.RequestCtx(), &User{
		Firstname: body.Firstname,
		Lastname:  body.Lastname,
		Email:     body.Email,
	})
	if err != nil {
		return err
	}

	token, err := h.userService.GenerateToken(c.RequestCtx(), createdUser, h.userService.configJWT.PrivateKey, h.userService.configJWT.JWT_TOKEN_EXP)
	if err != nil {
		return err
	}

	refreshToken, err := h.userService.GenerateToken(c.RequestCtx(), createdUser, h.userService.configJWT.RefreshPrivateKey, h.userService.configJWT.JWT_REFRESH_TOKEN_EXP)
	if err != nil {
		return err
	}

	return c.JSON(map[string]any{
		"token":         token,
		"refresh_token": refreshToken,
	})
}
