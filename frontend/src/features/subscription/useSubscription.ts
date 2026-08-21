import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../auth/useAuth';
import { getSubscriptionStatus } from './api';
import type { SubscriptionInfo } from './types';

export function useSubscription() {
  const { t } = useTranslation();
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
      .catch((cause) => setError(cause instanceof Error ? cause.message : t('subscription.loadError')))
      .finally(() => setLoading(false));
  }, [token, t]);

  useEffect(() => {
    refetch();
  }, [refetch]);

  return {
    status: info?.status ?? 'none',
    isActive: info?.isActive ?? false,
    info,
    loading,
    error,
    refetch,
  };
}
