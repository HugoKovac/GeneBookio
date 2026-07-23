package user

import (
	"hkorpo/book/pkg/errorwrapper"
	"log"

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

	log.Println(body)

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

	return c.JSON(createdUser)
}
