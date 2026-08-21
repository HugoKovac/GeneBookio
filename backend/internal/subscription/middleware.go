package subscription

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// MiddlewareRequireActiveSubscription gates a route behind an active (or
// trialing/past-due grace-window) subscription. It performs a DB lookup on
// every request rather than trusting a claim embedded in the JWT (unlike
// primitive.UserRole in user.MiddlewareAuth) — a canceled or lapsed
// subscription must lose access immediately, not only after the access
// token's JWT_TOKEN_EXP (default 60m) naturally expires, which is too long
// a window for a paid gate.
//
// Must run after user.MiddlewareAuth, which populates "authUserID".
func MiddlewareRequireActiveSubscription(service *Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUserID, ok := c.Locals("authUserID").(uuid.UUID)
		if !ok {
			return c.SendStatus(http.StatusUnauthorized)
		}

		sub, err := service.GetStatus(c.RequestCtx(), authUserID)
		if err != nil {
			return err
		}

		if !sub.IsActive() {
			return c.SendStatus(http.StatusPaymentRequired)
		}

		return c.Next()
	}
}
