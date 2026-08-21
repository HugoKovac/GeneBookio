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
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// IsActive reports whether s currently grants access to gated content.
// PastDue keeps access during Stripe's payment-retry grace window; Trialing
// counts too since it's still a functioning subscription. Nil-safe so
// callers with no Subscription row at all (a brand new user) can call it
// directly.
func (s *Subscription) IsActive() bool {
	return s != nil && (s.Status == primitive.SubscriptionActive ||
		s.Status == primitive.SubscriptionTrialing ||
		s.Status == primitive.SubscriptionPastDue)
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
