import { Alert, Button, Loader, Stack, Text, Title } from '@mantine/core';
import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../features/auth/useAuth';
import { getSubscriptionStatus } from '../features/subscription/api';

// SUBSCRIPTION_SUCCESS_URL appends ?session_id={CHECKOUT_SESSION_ID} — the
// backend's GET /subscriptions/me reconciles directly from that Checkout
// Session before returning status, so a single call here is enough (no
// webhook wait, no polling needed).
export default function SubscribeReturnPage() {
  const { token } = useAuth();
  const [searchParams] = useSearchParams();
  const { t } = useTranslation();
  const [status, setStatus] = useState<'checking' | 'active' | 'pending'>('checking');
  const [error, setError] = useState('');

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    const sessionID = searchParams.get('session_id') ?? undefined;

    getSubscriptionStatus(token, sessionID)
      .then((info) => {
        if (cancelled) return;
        setStatus(info.isActive ? 'active' : 'pending');
      })
      .catch((cause) => {
        if (!cancelled) {
          setError(cause instanceof Error ? cause.message : t('subscribeReturn.error'));
          setStatus('pending');
        }
      });

    return () => {
      cancelled = true;
    };
  }, [token, searchParams, t]);

  return (
    <Stack align="center" justify="center" mih="70vh" px="lg" gap="md">
      {status === 'checking' && (
        <>
          <Loader />
          <Text c="dimmed">{t('subscribeReturn.confirming')}</Text>
        </>
      )}

      {status === 'active' && (
        <>
          <Title order={2}>{t('subscribeReturn.activeTitle')}</Title>
          <Text c="dimmed" ta="center">{t('subscribeReturn.activeSubtitle')}</Text>
          <Button component={Link} to="/dashboard" radius="xl">{t('subscribeReturn.goToLibrary')}</Button>
        </>
      )}

      {status === 'pending' && (
        <>
          <Title order={2}>{t('subscribeReturn.pendingTitle')}</Title>
          <Text c="dimmed" ta="center">
            {t('subscribeReturn.pendingSubtitle')}
          </Text>
          <Button component={Link} to="/dashboard" radius="xl" variant="light">{t('subscribeReturn.goToLibrary')}</Button>
        </>
      )}

      {error && <Alert color="yellow" radius="lg">{error}</Alert>}
    </Stack>
  );
}
