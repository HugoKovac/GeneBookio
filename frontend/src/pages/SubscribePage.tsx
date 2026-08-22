import { ActionIcon, Alert, Box, Button, Group, Stack, Text, ThemeIcon, Title, UnstyledButton } from '@mantine/core';
import { IconCheck, IconHeadphones, IconX } from '@tabler/icons-react';
import { Capacitor } from '@capacitor/core';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../features/auth/useAuth';
import { createCheckoutSession, reconcileRevenueCat } from '../features/subscription/api';
import { getCurrentOffering, purchaseCurrentPackage, restorePurchases } from '../features/subscription/nativePurchases';
import { useSubscription } from '../features/subscription/useSubscription';

const isNative = Capacitor.isNativePlatform();
const defaultPriceDisplay = import.meta.env.VITE_SUBSCRIPTION_PRICE_DISPLAY ?? '€5.99';

function isUserCancelled(cause: unknown): boolean {
  return typeof cause === 'object' && cause !== null && 'userCancelled' in cause && (cause as { userCancelled?: boolean }).userCancelled === true;
}

export default function SubscribePage() {
  const { token } = useAuth();
  const { refetch } = useSubscription();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [restoring, setRestoring] = useState(false);
  const [error, setError] = useState('');
  const [price, setPrice] = useState(defaultPriceDisplay);

  // Native storefronts show a localized, tax-inclusive price that can differ
  // from the static env display value — swap it in once the offering loads,
  // falling back to the env value if that fails (no blocked CTA either way).
  useEffect(() => {
    if (!isNative) return;
    getCurrentOffering()
      .then((offering) => {
        const priceString = offering?.availablePackages[0]?.product.priceString;
        if (priceString) setPrice(priceString);
      })
      .catch(() => {});
  }, []);

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
      if (isNative) {
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

  const handleRestore = async () => {
    if (!token) return;
    setRestoring(true);
    setError('');
    try {
      await restorePurchases();
      const info = await reconcileRevenueCat(token);
      refetch();
      if (info.isActive) {
        navigate('/dashboard');
      } else {
        setError(t('subscribe.restoreNone'));
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t('subscribe.error'));
    } finally {
      setRestoring(false);
    }
  };

  return (
    <Box style={{ minHeight: 'calc(100dvh - 64px)', display: 'flex', flexDirection: 'column' }}>
      <Group justify="flex-end" p="md">
        <ActionIcon variant="subtle" color="gray" radius="xl" size="lg" onClick={() => navigate(-1)} aria-label={t('subscribe.close')}>
          <IconX size={22} />
        </ActionIcon>
      </Group>

      <Stack style={{ flex: 1 }} justify="center" px="lg" gap="xl" pb="xl">
        <Stack align="center" gap={6}>
          <ThemeIcon size={72} radius="xl" variant="light" color="violet">
            <IconHeadphones size={36} stroke={1.5} />
          </ThemeIcon>
          <Title order={1} ta="center" style={{ fontSize: 30 }}>{t('subscribe.title')}</Title>
          <Text c="dimmed" ta="center" maw={320}>{t('subscribe.subtitle')}</Text>
        </Stack>

        <Stack gap="sm" w="100%" maw={360} mx="auto">
          <FeatureRow text={t('subscribe.feature1')} />
          <FeatureRow text={t('subscribe.feature2')} />
          <FeatureRow text={t('subscribe.feature3')} />
        </Stack>
      </Stack>

      <Stack px="lg" pb="xl" gap="sm" w="100%" maw={420} mx="auto">
        {error && <Alert color="red" radius="lg">{error}</Alert>}

        <Text ta="center">
          <Text span fw={700} size="xl">{price}</Text>
          <Text span c="dimmed" size="sm"> {t('subscribe.perMonth')}</Text>
        </Text>

        <Button size="lg" radius="xl" fullWidth loading={loading} onClick={handleSubscribe}>
          {t('subscribe.cta')}
        </Button>

        {isNative && (
          <UnstyledButton onClick={handleRestore} disabled={restoring} style={{ alignSelf: 'center' }}>
            <Text size="sm" c="dimmed" td="underline">{restoring ? t('subscribe.restoring') : t('subscribe.restore')}</Text>
          </UnstyledButton>
        )}

        <Text size="xs" c="dimmed" ta="center">{t('subscribe.finePrint')}</Text>
      </Stack>
    </Box>
  );
}

function FeatureRow({ text }: { text: string }) {
  return (
    <Group gap="sm" wrap="nowrap">
      <ThemeIcon size={24} radius="xl" color="violet" variant="light">
        <IconCheck size={14} stroke={2.5} />
      </ThemeIcon>
      <Text>{text}</Text>
    </Group>
  );
}
