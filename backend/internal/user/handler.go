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

	h.router.Get("/",
		h.List,
	)

	h.router.Get("/:id",
		MiddlewareUserExists(userService),
		h.Get,
	)

	h.router.Patch("/:id",
		MiddlewareUserExists(userService),
		MiddlewareAuth(userService),
		MiddlewareRequireSelf,
		h.Update,
	)

	h.router.Delete("/:id",
		MiddlewareUserExists(userService),
		MiddlewareAuth(userService),
		MiddlewareRequireSelf,
		h.Delete,
	)

	h.router.Post("/register",
		h.Register,
	)

	h.router.Post("/login",
		h.Login,
	)

}

// Get returns a user by UUID.
//
// @Summary      Get user
// @Description  Fetch a user record by their UUID
// @Tags         users
// @Produce      json
// @Param        id  path  string  true  "User UUID"
// @Success      200  {object}  User
// @Failure      404  {object}  map[string]string
// @Router       /users/{id} [get]
func (h *Handler) Get(c fiber.Ctx) error {
	return c.JSON(c.Locals("user").(*User))
}

// List returns all users.
//
// @Summary      List users
// @Description  Fetch all non-deleted user records
// @Tags         users
// @Produce      json
// @Success      200  {array}   User
// @Router       /users [get]
func (h *Handler) List(c fiber.Ctx) error {
	users, err := h.userService.List(c.RequestCtx())
	if err != nil {
		return err
	}

	return c.JSON(users)
}

// Update updates a user's firstname and lastname.
//
// @Summary      Update user
// @Description  Update a user's firstname and lastname; requires a bearer token for the same user
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string               true  "User UUID"
// @Param        body  body      UpdateUserRequestDTO  true  "Update payload"
// @Success      200   {object}  User
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      403   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /users/{id} [patch]
func (h *Handler) Update(c fiber.Ctx) error {
	user := c.Locals("user").(*User)

	var body UpdateUserRequestDTO

	if err := c.Bind().Body(&body); err != nil {
		return errorwrapper.Wrap(err)
	}

	if err := h.validate.Struct(&body); err != nil {
		return errorwrapper.Wrap(err)
	}

	updatedUser, err := h.userService.Update(c.RequestCtx(), user.ID, body.Firstname, body.Lastname, body.Language)
	if err != nil {
		return err
	}

	return c.JSON(updatedUser)
}

// Delete soft-deletes a user account, preventing future logins.
//
// @Summary      Delete user
// @Description  Soft-delete a user account; requires a bearer token for the same user; the account can no longer log in
// @Tags         users
// @Security     BearerAuth
// @Param        id  path  string  true  "User UUID"
// @Success      204
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /users/{id} [delete]
func (h *Handler) Delete(c fiber.Ctx) error {
	user := c.Locals("user").(*User)

	if err := h.userService.SoftDelete(c.RequestCtx(), user.ID); err != nil {
		return err
	}

	return c.SendStatus(http.StatusNoContent)
}

// Register creates a new user account and returns JWT tokens.
//
// @Summary      Register user
// @Description  Create a new user account; returns JWT access and refresh tokens
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequestDTO  true  "Registration payload"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Router       /users/register [post]
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

// Login authenticates a user and returns JWT tokens.
//
// @Summary      Login user
// @Description  Authenticate with email and password; returns JWT access and refresh tokens
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      LoginRequestDTO  true  "Login credentials"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      403   {string}  string
// @Router       /users/login [post]
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
