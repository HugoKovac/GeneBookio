package user

import (
	"hkorpo/book/pkg/errorwrapper"
	"net/http"

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

	h.router.Post("/register",
		h.Register,
	)

	h.router.Post("/login",
		h.Login,
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

	hash, err := h.userService.HashPassword(body.Password)
	if err != nil {
		return err
	}

	createdUser, err := h.userService.Create(c.RequestCtx(), &User{
		Firstname:    body.Firstname,
		Lastname:     body.Lastname,
		Email:        body.Email,
		PasswordHash: hash,
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

	return c.Status(http.StatusCreated).JSON(map[string]any{
		"token":         token,
		"refresh_token": refreshToken,
	})
}

func (h *Handler) Login(c fiber.Ctx) error {
	var body LoginRequestDTO

	if err := c.Bind().Body(&body); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := validator.New().Struct(&body); err != nil {
		return errorwrapper.Wrap(err)
	}

	user, err := h.userService.GetByEmail(c.RequestCtx(), body.Email)
	if err != nil {
		return err
	}

	ok, err := h.userService.ValidatePasswordHash(user.PasswordHash, []byte(body.Password))
	if err != nil {
		return err
	}

	if !ok {
		return c.SendStatus(http.StatusForbidden)
	}

	token, err := h.userService.GenerateToken(c.RequestCtx(), user, h.userService.configJWT.PrivateKey, h.userService.configJWT.JWT_TOKEN_EXP)
	if err != nil {
		return err
	}

	refreshToken, err := h.userService.GenerateToken(c.RequestCtx(), user, h.userService.configJWT.RefreshPrivateKey, h.userService.configJWT.JWT_REFRESH_TOKEN_EXP)
	if err != nil {
		return err
	}

	return c.JSON(map[string]any{
		"token":         token,
		"refresh_token": refreshToken,
	})
}
