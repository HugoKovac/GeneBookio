import { useEffect, useMemo, useState, type ReactNode } from 'react';
import i18n from '../../i18n';
import { configureRevenueCat } from '../subscription/nativePurchases';
import { getUser } from '../user/api';
import { login as loginRequest, register as registerRequest } from './api';
import { AuthContext, type AuthContextValue } from './context';
import { decodeExpiry, decodeUserID } from './jwt';
import { readTokens, refreshTokens, subscribeTokens, writeTokens, type AuthTokens } from './tokenStore';

// Proactively refresh the access token once it's within this margin of
// expiring, instead of always waiting for a request to hit a 401 first.
// This is what makes reopening the app after a long idle period silently
// pick up a fresh token instead of the first screen surfacing a 401 on
// e.g. /subscriptions/me.
const refreshMarginMs = 5 * 60 * 1000;

export function AuthProvider({ children }: { children: ReactNode }) {
  const [tokens, setTokens] = useState<AuthTokens | null>(readTokens);
  const userID = tokens?.token ? decodeUserID(tokens.token) : null;

  // Keep local state in sync with tokenStore even when it changes outside
  // of this component's own calls — e.g. apiClient.authFetch silently
  // refreshing after a 401, or another refresh triggered below.
  useEffect(() => subscribeTokens(setTokens), []);

  // Check the stored token's expiry on load and whenever the app/tab
  // regains focus, refreshing ahead of time if it's stale or about to
  // expire. A failed refresh is left to whatever request eventually 401s.
  useEffect(() => {
    function maybeRefresh() {
      const current = readTokens();
      if (!current) return;
      const exp = decodeExpiry(current.token);
      if (exp !== null && exp - Date.now() > refreshMarginMs) return;
      refreshTokens().catch(() => {});
    }

    maybeRefresh();

    function onVisible() {
      if (document.visibilityState === 'visible') maybeRefresh();
    }
    document.addEventListener('visibilitychange', onVisible);
    window.addEventListener('focus', maybeRefresh);
    return () => {
      document.removeEventListener('visibilitychange', onVisible);
      window.removeEventListener('focus', maybeRefresh);
    };
  }, []);

  // On native (iOS/Android), RevenueCat must be configured with our own
  // user id — App Store/Play Store require in-app purchases for digital
  // content bought inside a native app, unlike the web app which keeps
  // using Stripe. No-op on web (configureRevenueCat checks
  // Capacitor.isNativePlatform() itself).
  useEffect(() => {
    if (userID) configureRevenueCat(userID);
  }, [userID]);

  // The account's language is the source of truth for the UI once signed
  // in — sync it on login/register and whenever a stored session is
  // restored on app load.
  useEffect(() => {
    if (!userID || !tokens?.token) return;
    getUser(userID, tokens.token)
      .then((user) => i18n.changeLanguage(user.Language))
      .catch(() => {});
  }, [userID, tokens?.token]);

  const value = useMemo<AuthContextValue>(() => ({
    isAuthenticated: Boolean(tokens?.token),
    userID,
    token: tokens?.token ?? null,
    login: async (input) => writeTokens(await loginRequest(input)),
    register: async (input) => writeTokens(await registerRequest(input)),
    logout: () => writeTokens(null),
  }), [tokens, userID]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
