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
  currentPeriodEnd?: string;
  cancelAtPeriodEnd: boolean;
};
