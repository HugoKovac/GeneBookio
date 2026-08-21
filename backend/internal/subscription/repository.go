package subscription

import (
	"context"
	"hkorpo/book/pkg/ent"
	entsubscription "hkorpo/book/pkg/ent/subscription"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/google/uuid"
)

type Repository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	// Upsert writes the latest known Stripe state for a user's subscription,
	// creating the row on its first sync. Safe to call repeatedly with the
	// same snapshot — Stripe webhook delivery is at-least-once, and this is
	// naturally idempotent since it always just overwrites with the latest
	// state rather than accumulating anything. Touches only the stripe_*
	// columns — never the revenuecat_* ones, see UpsertRevenueCat.
	Upsert(ctx context.Context, snapshot *SubscriptionSnapshot) (*Subscription, error)
	// UpsertRevenueCat is Upsert's RevenueCat counterpart: same
	// update-then-create-on-miss shape, but touches only the revenuecat_*
	// columns. Kept as a separate narrow method (rather than folding into
	// one generic upsert) specifically so a Stripe webhook can never
	// overwrite a still-active RevenueCat entitlement, and vice versa — each
	// origin's sync path is structurally incapable of touching the other's
	// columns.
	UpsertRevenueCat(ctx context.Context, snapshot *RevenueCatSnapshot) (*Subscription, error)
}

type RepositoryImpl struct {
	dbClient *ent.Client
}

func NewRepositoryImpl(dbClient *ent.Client) *RepositoryImpl {
	return &RepositoryImpl{
		dbClient: dbClient,
	}
}

func (r *RepositoryImpl) GetByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	e, err := r.dbClient.Subscription.Query().Where(
		entsubscription.UserID(userID),
	).Only(ctx)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	return fromEntSubscription(e), nil
}

func (r *RepositoryImpl) Upsert(ctx context.Context, snapshot *SubscriptionSnapshot) (*Subscription, error) {
	update := r.dbClient.Subscription.Update().
		Where(entsubscription.UserID(snapshot.UserID)).
		SetStatus(snapshot.Status).
		SetCancelAtPeriodEnd(snapshot.CancelAtPeriodEnd)
	if snapshot.StripeCustomerID != "" {
		update.SetStripeCustomerID(snapshot.StripeCustomerID)
	}
	if snapshot.StripeSubscriptionID != "" {
		update.SetStripeSubscriptionID(snapshot.StripeSubscriptionID)
	}
	if snapshot.CurrentPeriodEnd != nil {
		update.SetCurrentPeriodEnd(*snapshot.CurrentPeriodEnd)
	} else {
		update.ClearCurrentPeriodEnd()
	}

	affected, err := update.Save(ctx)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	if affected == 0 {
		// No existing row for this user (their first-ever webhook sync) —
		// a concurrent duplicate Create here is theoretically possible but
		// not realistic given Stripe's per-subscription webhook delivery is
		// effectively serial for one user.
		create := r.dbClient.Subscription.Create().
			SetUserID(snapshot.UserID).
			SetStatus(snapshot.Status).
			SetCancelAtPeriodEnd(snapshot.CancelAtPeriodEnd)
		if snapshot.StripeCustomerID != "" {
			create.SetStripeCustomerID(snapshot.StripeCustomerID)
		}
		if snapshot.StripeSubscriptionID != "" {
			create.SetStripeSubscriptionID(snapshot.StripeSubscriptionID)
		}
		if snapshot.CurrentPeriodEnd != nil {
			create.SetCurrentPeriodEnd(*snapshot.CurrentPeriodEnd)
		}

		e, err := create.Save(ctx)
		if err != nil {
			return nil, errorwrapper.Wrap(err)
		}
		return fromEntSubscription(e), nil
	}

	return r.GetByUserID(ctx, snapshot.UserID)
}

func (r *RepositoryImpl) UpsertRevenueCat(ctx context.Context, snapshot *RevenueCatSnapshot) (*Subscription, error) {
	update := r.dbClient.Subscription.Update().
		Where(entsubscription.UserID(snapshot.UserID)).
		SetRevenuecatActive(snapshot.Active)
	if snapshot.ExpiresAt != nil {
		update.SetRevenuecatExpiresAt(*snapshot.ExpiresAt)
	} else {
		update.ClearRevenuecatExpiresAt()
	}
	if snapshot.Store != "" {
		update.SetRevenuecatStore(snapshot.Store)
	}
	if snapshot.EntitlementID != "" {
		update.SetRevenuecatEntitlementID(snapshot.EntitlementID)
	}
	if snapshot.OriginalTransactionID != "" {
		update.SetRevenuecatOriginalTransactionID(snapshot.OriginalTransactionID)
	}

	affected, err := update.Save(ctx)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	if affected == 0 {
		// No existing row for this user (their first-ever RevenueCat sync,
		// and they've never subscribed via Stripe either) — same
		// theoretical-only concurrent-Create caveat as Upsert.
		create := r.dbClient.Subscription.Create().
			SetUserID(snapshot.UserID).
			SetRevenuecatActive(snapshot.Active)
		if snapshot.ExpiresAt != nil {
			create.SetRevenuecatExpiresAt(*snapshot.ExpiresAt)
		}
		if snapshot.Store != "" {
			create.SetRevenuecatStore(snapshot.Store)
		}
		if snapshot.EntitlementID != "" {
			create.SetRevenuecatEntitlementID(snapshot.EntitlementID)
		}
		if snapshot.OriginalTransactionID != "" {
			create.SetRevenuecatOriginalTransactionID(snapshot.OriginalTransactionID)
		}

		e, err := create.Save(ctx)
		if err != nil {
			return nil, errorwrapper.Wrap(err)
		}
		return fromEntSubscription(e), nil
	}

	return r.GetByUserID(ctx, snapshot.UserID)
}

func fromEntSubscription(e *ent.Subscription) *Subscription {
	return &Subscription{
		ID:                              e.ID,
		UserID:                          e.UserID,
		StripeCustomerID:                e.StripeCustomerID,
		StripeSubscriptionID:            e.StripeSubscriptionID,
		Status:                          e.Status,
		CurrentPeriodEnd:                e.CurrentPeriodEnd,
		CancelAtPeriodEnd:               e.CancelAtPeriodEnd,
		RevenueCatActive:                e.RevenuecatActive,
		RevenueCatExpiresAt:             e.RevenuecatExpiresAt,
		RevenueCatStore:                 e.RevenuecatStore,
		RevenueCatEntitlementID:         e.RevenuecatEntitlementID,
		RevenueCatOriginalTransactionID: e.RevenuecatOriginalTransactionID,
		CreatedAt:                       e.CreatedAt,
		UpdatedAt:                       e.UpdatedAt,
	}
}
