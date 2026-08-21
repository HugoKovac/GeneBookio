import type { SubscriptionInfo } from './types';

const apiBaseUrl = import.meta.env.VITE_API_URL ?? '/api';

async function request(path: string, token: string, method: 'GET' | 'POST' = 'GET'): Promise<Response> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    method,
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error('We could not complete your request. Please try again.');
  }

  return response;
}

// getSubscriptionStatus optionally reconciles from a completed Stripe
// Checkout Session first (see internal/subscription's GetMe handler) — pass
// the session_id Stripe appends to the success URL right after checkout.
export async function getSubscriptionStatus(token: string, sessionID?: string): Promise<SubscriptionInfo> {
  const query = sessionID ? `?sessionID=${encodeURIComponent(sessionID)}` : '';
  const response = await request(`/subscriptions/me${query}`, token);
  return response.json() as Promise<SubscriptionInfo>;
}

export async function createCheckoutSession(token: string): Promise<{ checkoutUrl: string }> {
  const response = await request('/subscriptions/checkout', token, 'POST');
  return response.json() as Promise<{ checkoutUrl: string }>;
}

// createPortalSession returns a Stripe-hosted Billing Portal URL where the
// user can update payment details or cancel their subscription.
export async function createPortalSession(token: string): Promise<{ portalUrl: string }> {
  const response = await request('/subscriptions/portal', token, 'POST');
  return response.json() as Promise<{ portalUrl: string }>;
}
