package schema

import (
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/ent/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Subscription holds the schema definition for the Subscription entity. One
// row per user, kept in sync from two independent origins: Stripe's
// customer.subscription.* webhooks (web) and RevenueCat's webhooks/REST API
// (native iOS/Android in-app purchases, required by App Store/Play Store
// policy for digital content bought inside a native app). The two origins'
// fields are deliberately namespaced and never written by the same code
// path, so a webhook from one origin can never clobber the other's active
// state — see internal/subscription/repository.go's Upsert/UpsertRevenueCat.
// Subscription.IsActive() grants access if either origin is active.
type Subscription struct {
	ent.Schema
}

func (Subscription) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Mixin{},
	}
}

func (Subscription) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.UUID("user_id", uuid.UUID{}).
			Unique(),
		field.String("stripe_customer_id").MaxLen(50).Optional(),
		field.String("stripe_subscription_id").MaxLen(50).Optional().Unique(),
		field.Enum("status").GoType(primitive.SubscriptionStatus("")).
			Default(primitive.SubscriptionIncomplete.String()),
		field.Time("current_period_end").Optional().Nillable(),
		field.Bool("cancel_at_period_end").Default(false),
		field.Bool("revenuecat_active").Default(false),
		field.Time("revenuecat_expires_at").Optional().Nillable(),
		field.String("revenuecat_store").MaxLen(20).Optional(),
		field.String("revenuecat_entitlement_id").MaxLen(100).Optional(),
		field.String("revenuecat_original_transaction_id").MaxLen(100).Optional(),
	}
}

func (Subscription) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
	}
}

func (Subscription) Edges() []ent.Edge {
	return nil
}
