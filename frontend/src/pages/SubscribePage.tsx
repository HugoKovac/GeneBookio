import { Alert, Button, Card, Stack, Text, Title } from '@mantine/core';
import { Capacitor } from '@capacitor/core';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../features/auth/useAuth';
import { createCheckoutSession, reconcileRevenueCat } from '../features/subscription/api';
import { getCurrentOffering, purchaseCurrentPackage } from '../features/subscription/nativePurchases';
import { useSubscription } from '../features/subscription/useSubscription';

const priceDisplay = import.meta.env.VITE_SUBSCRIPTION_PRICE_DISPLAY ?? '€5.99/month';

function isUserCancelled(cause: unknown): boolean {
  return typeof cause === 'object' && cause !== null && 'userCancelled' in cause && (cause as { userCancelled?: boolean }).userCancelled === true;
}

export default function SubscribePage() {
  const { token } = useAuth();
  const { refetch } = useSubscription();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubscribeWeb = async () => {
    if (!token) return;
    const { checkoutUrl } = await createCheckoutSession(token);
    window.location.href = checkoutUrl;
  };

  const handleSubscribeNative = async () => {
    if (!token) return;
    const offering = await getCurrentOffering();
    const aPackage = offering?.availablePackages[0];
    if (!aPackage) {
      throw new Error(t('subscribe.error'));
    }
    await purchaseCurrentPackage(aPackage);
    // The purchase succeeded on-device, but our backend's row isn't
    // populated yet — reconcile synchronously rather than waiting on the
    // async RevenueCat webhook, mirroring the web flow's Checkout Session
    // reconcile (see SubscribeReturnPage.tsx).
    await reconcileRevenueCat(token);
    refetch();
    navigate('/dashboard');
  };

  const handleSubscribe = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      if (Capacitor.isNativePlatform()) {
        await handleSubscribeNative();
      } else {
        await handleSubscribeWeb();
      }
    } catch (cause) {
      // A user backing out of the native purchase sheet isn't an error worth surfacing.
      if (isUserCancelled(cause)) return;
      setError(cause instanceof Error ? cause.message : t('subscribe.error'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Stack align="center" justify="center" mih="70vh" px="lg">
      <Card radius="lg" padding="xl" shadow="sm" w={{ base: '100%', sm: 420 }}>
        <Stack gap="md" align="center">
          <Title order={2} ta="center">{t('subscribe.title')}</Title>
          <Text c="dimmed" ta="center">{t('subscribe.subtitle')}</Text>
          <Text fw={700} size="xl">{priceDisplay}</Text>

          {error && <Alert color="red" radius="lg" w="100%">{error}</Alert>}

          <Button size="md" radius="xl" fullWidth loading={loading} onClick={handleSubscribe}>
            {t('subscribe.cta')}
          </Button>
        </Stack>
      </Card>
    </Stack>
  );
}
