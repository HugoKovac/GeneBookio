import { Alert, Button, Loader, Stack, Text, Title } from '@mantine/core';
import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useAuth } from '../features/auth/useAuth';
import { getSubscriptionStatus } from '../features/subscription/api';

// SUBSCRIPTION_SUCCESS_URL appends ?session_id={CHECKOUT_SESSION_ID} — the
// backend's GET /subscriptions/me reconciles directly from that Checkout
// Session before returning status, so a single call here is enough (no
// webhook wait, no polling needed).
export default function SubscribeReturnPage() {
  const { token } = useAuth();
  const [searchParams] = useSearchParams();
  const [status, setStatus] = useState<'checking' | 'active' | 'pending'>('checking');
  const [error, setError] = useState('');

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    const sessionID = searchParams.get('session_id') ?? undefined;

    getSubscriptionStatus(token, sessionID)
      .then((info) => {
        if (cancelled) return;
        setStatus(info.status === 'active' || info.status === 'trialing' || info.status === 'past_due' ? 'active' : 'pending');
      })
      .catch((cause) => {
        if (!cancelled) {
          setError(cause instanceof Error ? cause.message : 'Unable to confirm your subscription.');
          setStatus('pending');
        }
      });

    return () => {
      cancelled = true;
    };
  }, [token, searchParams]);

  return (
    <Stack align="center" justify="center" mih="70vh" px="lg" gap="md">
      {status === 'checking' && (
        <>
          <Loader />
          <Text c="dimmed">Confirming your payment…</Text>
        </>
      )}

      {status === 'active' && (
        <>
          <Title order={2}>You&apos;re subscribed!</Title>
          <Text c="dimmed" ta="center">Your subscription is active — enjoy the library.</Text>
          <Button component={Link} to="/dashboard" radius="xl">Go to your library</Button>
        </>
      )}

      {status === 'pending' && (
        <>
          <Title order={2}>Almost there</Title>
          <Text c="dimmed" ta="center">
            We haven&apos;t confirmed your payment yet — this can take a moment. Refresh this page shortly, or check your library.
          </Text>
          <Button component={Link} to="/dashboard" radius="xl" variant="light">Go to your library</Button>
        </>
      )}

      {error && <Alert color="yellow" radius="lg">{error}</Alert>}
    </Stack>
  );
}
