import { useCallback, useEffect, useState } from 'react';
import { useAuth } from '../auth/useAuth';
import { getSubscriptionStatus } from './api';
import type { SubscriptionInfo } from './types';

const activeStatuses = new Set(['active', 'trialing', 'past_due']);

export function useSubscription() {
  const { token } = useAuth();
  const [info, setInfo] = useState<SubscriptionInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const refetch = useCallback(() => {
    if (!token) {
      setInfo(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    getSubscriptionStatus(token)
      .then(setInfo)
      .catch((cause) => setError(cause instanceof Error ? cause.message : 'Unable to load your subscription.'))
      .finally(() => setLoading(false));
  }, [token]);

  useEffect(() => {
    refetch();
  }, [refetch]);

  return {
    status: info?.status ?? 'none',
    isActive: info ? activeStatuses.has(info.status) : false,
    info,
    loading,
    error,
    refetch,
  };
}
