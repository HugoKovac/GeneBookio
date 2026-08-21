package subscription

import (
	"hkorpo/book/internal/user"
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
	h.router.Post("/webhook/revenuecat", h.HandleRevenueCatWebhook)
	h.router.Post("/reconcile/revenuecat", requireAuth, h.ReconcileRevenueCat)
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
		return err
	}

	checkoutURL, err := h.service.CreateCheckoutSession(c.RequestCtx(), authUserID, u.Email)
	if err != nil {
		return err
	}

	return c.JSON(checkoutResponseDTO{CheckoutURL: checkoutURL})
}

type subscriptionStatusDTO struct {
	Status            string  `json:"status"`
	IsActive          bool    `json:"isActive"` // sub.IsActive() — authoritative regardless of origin (Stripe or RevenueCat)
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
			return err
		}
	}

	sub, err := h.service.GetStatus(c.RequestCtx(), authUserID)
	if err != nil {
		return err
	}

	if sub == nil {
		return c.JSON(subscriptionStatusDTO{Status: "none", IsActive: false})
	}

	dto := subscriptionStatusDTO{
		Status:            sub.Status.String(),
		IsActive:          sub.IsActive(),
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
		return err
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
		return err
	}

	return c.SendStatus(http.StatusOK)
}

// HandleRevenueCatWebhook processes a RevenueCat webhook delivery (a native
// iOS/Android in-app purchase event) — the same 200/401/500 contract as
// HandleWebhook above, for the same reasons.
//
// @Summary      RevenueCat webhook
// @Tags         subscriptions
// @Success      200
// @Failure      401
// @Router       /subscriptions/webhook/revenuecat [post]
func (h *Handler) HandleRevenueCatWebhook(c fiber.Ctx) error {
	rawBody := c.Body()
	authHeader := c.Get(fiber.HeaderAuthorization)

	if err := h.service.HandleRevenueCatWebhookEvent(c.RequestCtx(), rawBody, authHeader); err != nil {
		return err
	}

	return c.SendStatus(http.StatusOK)
}

// ReconcileRevenueCat syncs the authenticated user's subscription row
// directly from RevenueCat's current entitlement state — called by the
// native app right after a purchase completes, so the UI doesn't have to
// wait on async webhook delivery. Mirrors GetMe's ?sessionID= Stripe
// reconcile path.
//
// @Summary      Reconcile subscription status from RevenueCat
// @Tags         subscriptions
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  subscriptionStatusDTO
// @Router       /subscriptions/reconcile/revenuecat [post]
func (h *Handler) ReconcileRevenueCat(c fiber.Ctx) error {
	authUserID, ok := c.Locals("authUserID").(uuid.UUID)
	if !ok {
		return c.SendStatus(http.StatusUnauthorized)
	}

	if err := h.service.ReconcileFromRevenueCat(c.RequestCtx(), authUserID); err != nil {
		return err
	}

	sub, err := h.service.GetStatus(c.RequestCtx(), authUserID)
	if err != nil {
		return err
	}
	if sub == nil {
		return c.JSON(subscriptionStatusDTO{Status: "none", IsActive: false})
	}

	dto := subscriptionStatusDTO{
		Status:            sub.Status.String(),
		IsActive:          sub.IsActive(),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}
	if sub.CurrentPeriodEnd != nil {
		formatted := sub.CurrentPeriodEnd.Format(http.TimeFormat)
		dto.CurrentPeriodEnd = &formatted
	}

	return c.JSON(dto)
}
