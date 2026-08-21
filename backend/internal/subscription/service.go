package subscription

import (
	"context"
	"errors"
	"hkorpo/book/pkg/ent"
	"hkorpo/book/pkg/errorwrapper"
	"net/http"

	"github.com/google/uuid"
)

// ErrInvalidWebhookSignature is returned by HandleWebhookEvent when the
// Stripe-Signature header doesn't match the raw body — callers (the HTTP
// handler) use errors.Is to map it to a 401 instead of a 500.
var ErrInvalidWebhookSignature = errors.New("invalid stripe webhook signature")

// ErrInvalidRevenueCatWebhookAuth is returned by HandleRevenueCatWebhookEvent
// when the Authorization header doesn't match the configured secret.
var ErrInvalidRevenueCatWebhookAuth = errors.New("invalid revenuecat webhook authorization")

// StripeAPI is the port to Stripe — StripeClient is its only implementation.
// Declared here per repo convention (cf. library.BooksAPI).
type StripeAPI interface {
	// CreateCheckoutSession returns the URL of a Stripe-hosted Checkout page
	// for a new subscription.
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutSessionRequest) (checkoutURL string, err error)
	// CreateBillingPortalSession returns the URL of a Stripe-hosted portal
	// where the user can update payment details or cancel.
	CreateBillingPortalSession(ctx context.Context, stripeCustomerID, returnURL string) (portalURL string, err error)
	// GetCheckoutSessionUserID reads back a completed Checkout Session by
	// ID, for the no-webhook-required reconciliation path (see
	// ReconcileFromCheckoutSession).
	GetCheckoutSessionUserID(ctx context.Context, sessionID string) (userID uuid.UUID, stripeSubscriptionID string, complete bool, err error)
	GetSubscriptionSnapshot(ctx context.Context, stripeSubscriptionID string) (*SubscriptionSnapshot, error)
	// ConstructWebhookEvent verifies the signature and, for a
	// customer.subscription.* event, returns the resulting snapshot;
	// snapshot is nil for event types this service doesn't act on.
	ConstructWebhookEvent(rawBody []byte, signatureHeader string) (eventType string, snapshot *SubscriptionSnapshot, err error)
}

// RevenueCatAPI is the port to RevenueCat — RevenueCatClient is its only
// implementation. Declared here per repo convention (cf. StripeAPI above).
type RevenueCatAPI interface {
	// ConstructWebhookEvent verifies the Authorization header and, for an
	// entitlement-relevant event, returns the resulting snapshot; snapshot
	// is nil for event types this service doesn't act on.
	ConstructWebhookEvent(rawBody []byte, authorizationHeader string) (eventType string, snapshot *RevenueCatSnapshot, err error)
	// GetEntitlementSnapshot is the reconcile-without-waiting-for-webhook
	// fast path (see ReconcileFromRevenueCat) — used right after a native
	// purchase completes client-side.
	GetEntitlementSnapshot(ctx context.Context, appUserID string) (*RevenueCatSnapshot, error)
}

type CreateCheckoutSessionRequest struct {
	UserID        uuid.UUID
	CustomerEmail string
	SuccessURL    string
	CancelURL     string
}

// PlanConfig is the flat single-plan pricing configuration. The actual
// price/product is created once in the Stripe Dashboard; we only ever
// reference it by ID.
type PlanConfig struct {
	PriceID         string `envconfig:"STRIPE_PRICE_ID"`
	SuccessURL      string `envconfig:"SUBSCRIPTION_SUCCESS_URL"`
	CancelURL       string `envconfig:"SUBSCRIPTION_CANCEL_URL"`
	PortalReturnURL string `envconfig:"SUBSCRIPTION_PORTAL_RETURN_URL"`
}

type Service struct {
	repo       Repository
	stripe     StripeAPI
	revenueCat RevenueCatAPI
	cfg        PlanConfig
}

func NewService(repo Repository, stripeAPI StripeAPI, revenueCatAPI RevenueCatAPI, cfg PlanConfig) *Service {
	return &Service{repo: repo, stripe: stripeAPI, revenueCat: revenueCatAPI, cfg: cfg}
}

// CreateCheckoutSession returns the Stripe-hosted Checkout URL to redirect
// the user to. It doesn't touch our own DB at all — the local Subscription
// row is only ever written from Stripe's own state (webhook or reconcile).
func (s *Service) CreateCheckoutSession(ctx context.Context, userID uuid.UUID, email string) (string, error) {
	return s.stripe.CreateCheckoutSession(ctx, CreateCheckoutSessionRequest{
		UserID:        userID,
		CustomerEmail: email,
		SuccessURL:    s.cfg.SuccessURL,
		CancelURL:     s.cfg.CancelURL,
	})
}

// GetStatus returns the user's subscription, or (nil, nil) if they have
// never subscribed — callers must treat that as "inactive", not an error.
func (s *Service) GetStatus(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return sub, nil
}

// ReconcileFromCheckoutSession syncs the local row directly from a
// completed Checkout Session. The frontend's return page calls this (via
// the session_id Stripe appends to the success URL) so the UI reflects the
// new subscription immediately after redirect, without waiting on webhook
// delivery — which also means local dev works without a public webhook URL.
func (s *Service) ReconcileFromCheckoutSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	gotUserID, stripeSubscriptionID, complete, err := s.stripe.GetCheckoutSessionUserID(ctx, sessionID)
	if err != nil {
		return err
	}
	if gotUserID != userID {
		return errorwrapper.Wrap(errors.New("checkout session does not belong to this user"))
	}
	if !complete || stripeSubscriptionID == "" {
		return nil
	}

	snapshot, err := s.stripe.GetSubscriptionSnapshot(ctx, stripeSubscriptionID)
	if err != nil {
		return err
	}
	_, err = s.repo.Upsert(ctx, snapshot)
	return err
}

// HandleWebhookEvent verifies and processes a Stripe webhook delivery.
// Delivery is at-least-once, but Upsert always just overwrites with the
// event's own snapshot, so re-processing the same or an out-of-order event
// is harmless — no separate idempotency tracking is needed.
func (s *Service) HandleWebhookEvent(ctx context.Context, rawBody []byte, signatureHeader string) error {
	_, snapshot, err := s.stripe.ConstructWebhookEvent(rawBody, signatureHeader)
	if err != nil {
		return errorwrapper.Wrap(errorwrapper.WithStatus(http.StatusUnauthorized, ErrInvalidWebhookSignature))
	}
	if snapshot == nil {
		return nil // an event type this service doesn't act on
	}
	_, err = s.repo.Upsert(ctx, snapshot)
	return err
}

// HandleRevenueCatWebhookEvent verifies and processes a RevenueCat webhook
// delivery. Like Stripe's, this is idempotent-by-construction: it always
// overwrites the revenuecat_* columns with the event's own computed
// snapshot, so re-processing (or out-of-order delivery) is harmless.
func (s *Service) HandleRevenueCatWebhookEvent(ctx context.Context, rawBody []byte, authorizationHeader string) error {
	_, snapshot, err := s.revenueCat.ConstructWebhookEvent(rawBody, authorizationHeader)
	if err != nil {
		return errorwrapper.Wrap(errorwrapper.WithStatus(http.StatusUnauthorized, ErrInvalidRevenueCatWebhookAuth))
	}
	if snapshot == nil {
		return nil // an event type this service doesn't act on
	}
	_, err = s.repo.UpsertRevenueCat(ctx, snapshot)
	return err
}

// ReconcileFromRevenueCat is the native-purchase equivalent of
// ReconcileFromCheckoutSession — called right after Purchases.purchasePackage()
// resolves client-side, so the UI reflects the new entitlement immediately
// instead of waiting on async webhook delivery.
func (s *Service) ReconcileFromRevenueCat(ctx context.Context, userID uuid.UUID) error {
	snapshot, err := s.revenueCat.GetEntitlementSnapshot(ctx, userID.String())
	if err != nil {
		return err
	}
	_, err = s.repo.UpsertRevenueCat(ctx, snapshot)
	return err
}

// CreatePortalSession returns a Stripe-hosted Billing Portal URL where the
// user can update payment details or cancel — Stripe's native replacement
// for a custom cancel endpoint.
func (s *Service) CreatePortalSession(ctx context.Context, userID uuid.UUID) (string, error) {
	sub, err := s.GetStatus(ctx, userID)
	if err != nil {
		return "", err
	}
	if sub == nil || sub.StripeCustomerID == "" {
		return "", errorwrapper.Wrap(errors.New("no billing account for this user yet"))
	}
	return s.stripe.CreateBillingPortalSession(ctx, sub.StripeCustomerID, s.cfg.PortalReturnURL)
}
