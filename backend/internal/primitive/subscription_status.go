package primitive

// SubscriptionStatus mirrors Stripe's own Subscription.Status values
// directly (stripe.SubscriptionStatus) rather than an independent
// vocabulary, since we store exactly what Stripe reports and never derive
// or translate it ourselves — see internal/subscription/service.go.
type SubscriptionStatus string

const (
	SubscriptionIncomplete        SubscriptionStatus = "incomplete"
	SubscriptionIncompleteExpired SubscriptionStatus = "incomplete_expired"
	SubscriptionTrialing          SubscriptionStatus = "trialing"
	SubscriptionActive            SubscriptionStatus = "active"
	SubscriptionPastDue           SubscriptionStatus = "past_due"
	SubscriptionCanceled          SubscriptionStatus = "canceled"
	SubscriptionUnpaid            SubscriptionStatus = "unpaid"
	SubscriptionPaused            SubscriptionStatus = "paused"
)

func (s SubscriptionStatus) String() string {
	return string(s)
}

func (s SubscriptionStatus) Values() (statuses []string) {
	for _, s := range []SubscriptionStatus{
		SubscriptionIncomplete,
		SubscriptionIncompleteExpired,
		SubscriptionTrialing,
		SubscriptionActive,
		SubscriptionPastDue,
		SubscriptionCanceled,
		SubscriptionUnpaid,
		SubscriptionPaused,
	} {
		statuses = append(statuses, string(s))
	}

	return
}
