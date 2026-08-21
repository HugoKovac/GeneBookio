package subscription

import (
	"errors"
	"hkorpo/book/internal/user"
	"hkorpo/book/pkg/ent"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type Handler struct {
	router      fiber.Router
	service     *Service
	userService *user.Service
}

// NewHandler mounts subscription's routes on router. requireAuth gates
// every route except the webhook, which Stripe authenticates itself via the
// Stripe-Signature header instead of a bearer token.
func NewHandler(router fiber.Router, service *Service, userService *user.Service, requireAuth fiber.Handler) {
	h := &Handler{
		router:      router,
		service:     service,
		userService: userService,
	}

	h.router.Post("/checkout", requireAuth, h.CreateCheckout)
	h.router.Get("/me", requireAuth, h.GetMe)
	h.router.Post("/portal", requireAuth, h.CreatePortal)
	h.router.Post("/webhook", h.HandleWebhook)
}

// respondError maps a service error to an explicit HTTP response. The
// shared access-log error-mapping middleware (pkg/fiber/logger) only
// recognizes validation/ent-constraint/ent-not-found/minio/fiber-bind
// errors — anything else it logs correctly but never actually writes to the
// response, which falls through to a misleading 200 with an empty body.
// Stripe API failures (the dominant failure mode here) aren't in that list,
// so these handlers set their own status rather than `return err`.
func respondError(c fiber.Ctx, err error) error {
	if ent.IsNotFound(err) {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": http.StatusText(http.StatusNotFound)})
	}
	return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": "unable to complete this request, please try again"})
}

type checkoutResponseDTO struct {
	CheckoutURL string `json:"checkoutUrl"`
}

// CreateCheckout starts a subscription checkout and returns the
// Stripe-hosted page to redirect the user to.
//
// @Summary      Start a subscription checkout
// @Tags         subscriptions
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  checkoutResponseDTO
// @Router       /subscriptions/checkout [post]
func (h *Handler) CreateCheckout(c fiber.Ctx) error {
	authUserID, ok := c.Locals("authUserID").(uuid.UUID)
	if !ok {
		return c.SendStatus(http.StatusUnauthorized)
	}

	u, err := h.userService.GetByID(c.RequestCtx(), authUserID)
	if err != nil {
		return respondError(c, err)
	}

	checkoutURL, err := h.service.CreateCheckoutSession(c.RequestCtx(), authUserID, u.Email)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(checkoutResponseDTO{CheckoutURL: checkoutURL})
}

type subscriptionStatusDTO struct {
	Status            string  `json:"status"`
	CurrentPeriodEnd  *string `json:"currentPeriodEnd,omitempty"`
	CancelAtPeriodEnd bool    `json:"cancelAtPeriodEnd"`
}

// GetMe returns the authenticated user's subscription status. If a
// sessionID query param is present (the frontend's return page passes back
// Stripe's {CHECKOUT_SESSION_ID}), it reconciles from that Checkout Session
// first — see Service.ReconcileFromCheckoutSession.
//
// @Summary      Get the authenticated user's subscription status
// @Tags         subscriptions
// @Security     BearerAuth
// @Produce      json
// @Param        sessionID  query  string  false  "Stripe Checkout Session ID to reconcile from"
// @Success      200  {object}  subscriptionStatusDTO
// @Router       /subscriptions/me [get]
func (h *Handler) GetMe(c fiber.Ctx) error {
	authUserID, ok := c.Locals("authUserID").(uuid.UUID)
	if !ok {
		return c.SendStatus(http.StatusUnauthorized)
	}

	if sessionID := c.Query("sessionID"); sessionID != "" {
		if err := h.service.ReconcileFromCheckoutSession(c.RequestCtx(), authUserID, sessionID); err != nil {
			return respondError(c, err)
		}
	}

	sub, err := h.service.GetStatus(c.RequestCtx(), authUserID)
	if err != nil {
		return respondError(c, err)
	}

	if sub == nil {
		return c.JSON(subscriptionStatusDTO{Status: "none"})
	}

	dto := subscriptionStatusDTO{
		Status:            sub.Status.String(),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}
	if sub.CurrentPeriodEnd != nil {
		formatted := sub.CurrentPeriodEnd.Format(http.TimeFormat)
		dto.CurrentPeriodEnd = &formatted
	}

	return c.JSON(dto)
}

type portalResponseDTO struct {
	PortalURL string `json:"portalUrl"`
}

// CreatePortal returns a Stripe-hosted Billing Portal URL where the user
// can update payment details or cancel their subscription.
//
// @Summary      Start a billing portal session
// @Tags         subscriptions
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  portalResponseDTO
// @Router       /subscriptions/portal [post]
func (h *Handler) CreatePortal(c fiber.Ctx) error {
	authUserID, ok := c.Locals("authUserID").(uuid.UUID)
	if !ok {
		return c.SendStatus(http.StatusUnauthorized)
	}

	portalURL, err := h.service.CreatePortalSession(c.RequestCtx(), authUserID)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(portalResponseDTO{PortalURL: portalURL})
}

// HandleWebhook processes a Stripe webhook delivery. It responds 200 on
// success, 401 on a bad signature, 500 otherwise — a non-2xx is what makes
// Stripe retry the delivery instead of considering it handled.
//
// @Summary      Stripe webhook
// @Tags         subscriptions
// @Success      200
// @Failure      401
// @Router       /subscriptions/webhook [post]
func (h *Handler) HandleWebhook(c fiber.Ctx) error {
	rawBody := c.Body()
	signature := c.Get("Stripe-Signature")

	if err := h.service.HandleWebhookEvent(c.RequestCtx(), rawBody, signature); err != nil {
		if errors.Is(err, ErrInvalidWebhookSignature) {
			return c.SendStatus(http.StatusUnauthorized)
		}
		return c.SendStatus(http.StatusInternalServerError)
	}

	return c.SendStatus(http.StatusOK)
}
