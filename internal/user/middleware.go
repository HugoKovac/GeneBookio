package user

import (
	"hkorpo/book/pkg/errorwrapper"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

func MiddlewareUserExists(userService *Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		var uri UserIDURI

		if err := c.Bind().URI(&uri); err != nil {
			return errorwrapper.Wrap(err)
		}

		if err := validator.New(validator.WithRequiredStructEnabled()).Struct(&uri); err != nil {
			return errorwrapper.Wrap(err)
		}

		user, err := userService.GetByID(c.RequestCtx(), uri.ID)
		if err != nil {
			return err
		}

		c.Locals("user", user)

		return c.Next()
	}
}
