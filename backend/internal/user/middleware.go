package user

import (
	"errors"
	"hkorpo/book/pkg/errorwrapper"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
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
func MiddlewareAuth(userService *Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get(fiber.HeaderAuthorization)

		tokenString, ok := strings.CutPrefix(authHeader, "Bearer ")
		if !ok || tokenString == "" {
			return c.SendStatus(http.StatusUnauthorized)
		}

		var claims UserTokenClaims

		token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, errors.New("unexpected signing method")
			}

			return userService.configJWT.PublicKey, nil
		})
		if err != nil || !token.Valid {
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
