package subscription

import (
	"hkorpo/book/internal/primitive"
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	StripeCustomerID     string
	StripeSubscriptionID string
	Status               primitive.SubscriptionStatus
	CurrentPeriodEnd     *time.Time
	CancelAtPeriodEnd    bool

	// RevenueCat fields — namespaced and independent of the Stripe fields
	// above (see Repository.Upsert/UpsertRevenueCat). RevenueCatActive is a
	// computed-at-write-time snapshot of "now < expiry", not a raw event
	// flag — RevenueCat's own CANCELLATION event, for instance, still leaves
	// the user active until RevenueCatExpiresAt.
	RevenueCatActive                bool
	RevenueCatExpiresAt             *time.Time
	RevenueCatStore                 string
	RevenueCatEntitlementID         string
	RevenueCatOriginalTransactionID string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Subscription) stripeIsActive() bool {
	return s.Status == primitive.SubscriptionActive ||
		s.Status == primitive.SubscriptionTrialing ||
		s.Status == primitive.SubscriptionPastDue
}

func (s *Subscription) revenueCatIsActive() bool {
	return s.RevenueCatActive && s.RevenueCatExpiresAt != nil && s.RevenueCatExpiresAt.After(time.Now())
}

// IsActive reports whether s currently grants access to gated content —
// true if either Stripe (web) or RevenueCat (native iOS/Android) says the
// user is subscribed, regardless of which platform they paid on. Nil-safe
// so callers with no Subscription row at all (a brand new user) can call it
// directly.
func (s *Subscription) IsActive() bool {
	return s != nil && (s.stripeIsActive() || s.revenueCatIsActive())
}

// SubscriptionSnapshot is Stripe's view of a subscription's state, as
// extracted from either a customer.subscription.* webhook or a direct
// Retrieve call — see StripeAPI. It's what Repository.Upsert persists.
type SubscriptionSnapshot struct {
	UserID               uuid.UUID
	StripeCustomerID     string
	StripeSubscriptionID string
	Status               primitive.SubscriptionStatus
	CurrentPeriodEnd     *time.Time
	CancelAtPeriodEnd    bool
}

// RevenueCatSnapshot is RevenueCat's view of a user's entitlement state, as
// extracted from either a webhook or a direct REST fetch — see
// RevenueCatAPI. It's what Repository.UpsertRevenueCat persists. Active is
// already computed (now < expiry) by the caller building this snapshot, not
// derived from the raw webhook event type.
type RevenueCatSnapshot struct {
	UserID                uuid.UUID
	Active                bool
	ExpiresAt             *time.Time
	Store                 string
	EntitlementID         string
	OriginalTransactionID string
}
