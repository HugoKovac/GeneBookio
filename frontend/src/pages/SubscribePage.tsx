import { Alert, Button, Card, Stack, Text, Title } from '@mantine/core';
import { useState } from 'react';
import { useAuth } from '../features/auth/useAuth';
import { createCheckoutSession } from '../features/subscription/api';

const priceDisplay = import.meta.env.VITE_SUBSCRIPTION_PRICE_DISPLAY ?? '€5.99/month';

export default function SubscribePage() {
  const { token } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubscribe = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const { checkoutUrl } = await createCheckoutSession(token);
      window.location.href = checkoutUrl;
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to start checkout. Please try again.');
      setLoading(false);
    }
  };

  return (
    <Stack align="center" justify="center" mih="70vh" px="lg">
      <Card radius="lg" padding="xl" shadow="sm" w={{ base: '100%', sm: 420 }}>
        <Stack gap="md" align="center">
          <Title order={2} ta="center">Subscribe to listen</Title>
          <Text c="dimmed" ta="center">Unlock narrated audio for every book in your library.</Text>
          <Text fw={700} size="xl">{priceDisplay}</Text>

          {error && <Alert color="red" radius="lg" w="100%">{error}</Alert>}

          <Button size="md" radius="xl" fullWidth loading={loading} onClick={handleSubscribe}>
            Subscribe
          </Button>
        </Stack>
      </Card>
    </Stack>
  );
}
