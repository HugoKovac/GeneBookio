package user

import (
	"hkorpo/book/pkg/errorwrapper"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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

// MiddlewareAuth verifies the request's bearer JWT and stores the
// authenticated user's ID and role in locals ("authUserID", "authUserRole").
// The token is normally read from the Authorization header; if that's
// absent, it falls back to a "token" query parameter, since an <audio>/<video>
// element's src can't carry custom headers but does need to hit an
// authenticated, range-request-capable endpoint directly.
func MiddlewareAuth(userService *Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenString, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
		if !ok || tokenString == "" {
			tokenString = c.Query("token")
		}
		if tokenString == "" {
			return c.SendStatus(http.StatusUnauthorized)
		}

		claims, err := userService.ParseToken(tokenString, userService.configJWT.PublicKey)
		if err != nil {
			return c.SendStatus(http.StatusUnauthorized)
		}

		c.Locals("authUserID", claims.UserID)
		c.Locals("authUserRole", claims.Role)

		return c.Next()
	}
}

// MiddlewareRequireSelf ensures the authenticated user (set by MiddlewareAuth)
// matches the target user (set by MiddlewareUserExists).
func MiddlewareRequireSelf(c fiber.Ctx) error {
	authUserID, ok := c.Locals("authUserID").(uuid.UUID)
	if !ok {
		return c.SendStatus(http.StatusUnauthorized)
	}

	targetUser, ok := c.Locals("user").(*User)
	if !ok {
		return c.SendStatus(http.StatusUnauthorized)
	}

	if authUserID != targetUser.ID {
		return c.SendStatus(http.StatusForbidden)
	}

	return c.Next()
}
