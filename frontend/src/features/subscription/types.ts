// Mirrors Stripe's own Subscription.status values directly, plus 'none' for
// "never subscribed" (no local row at all).
export type SubscriptionStatusValue =
  | 'none'
  | 'incomplete'
  | 'incomplete_expired'
  | 'trialing'
  | 'active'
  | 'past_due'
  | 'canceled'
  | 'unpaid'
  | 'paused';

export type SubscriptionInfo = {
  status: SubscriptionStatusValue;
  // Authoritative regardless of origin — true if either Stripe (web) or
  // RevenueCat (native iOS/Android) says the user is subscribed. Computed
  // server-side (Subscription.IsActive()); don't re-derive this from
  // `status` alone, which only reflects the Stripe side.
  isActive: boolean;
  currentPeriodEnd?: string;
  cancelAtPeriodEnd: boolean;
};
