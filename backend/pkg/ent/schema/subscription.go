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
// row per user, kept in sync from Stripe's customer.subscription.* webhooks
// — see internal/subscription/service.go. Stripe is the source of truth for
// billing state; this row is a read cache for the audio-access gate.
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
