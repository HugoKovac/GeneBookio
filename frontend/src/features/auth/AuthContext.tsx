import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { configureRevenueCat } from '../subscription/nativePurchases';
import { login as loginRequest, register as registerRequest, type AuthTokens } from './api';
import { AuthContext, type AuthContextValue } from './context';
import { decodeUserID } from './jwt';

const storageKey = 'genebookio.auth';

function readTokens(): AuthTokens | null {
  try {
    const value = localStorage.getItem(storageKey);
    return value ? (JSON.parse(value) as AuthTokens) : null;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [tokens, setTokens] = useState<AuthTokens | null>(readTokens);
  const userID = tokens?.token ? decodeUserID(tokens.token) : null;

  // On native (iOS/Android), RevenueCat must be configured with our own
  // user id — App Store/Play Store require in-app purchases for digital
  // content bought inside a native app, unlike the web app which keeps
  // using Stripe. No-op on web (configureRevenueCat checks
  // Capacitor.isNativePlatform() itself).
  useEffect(() => {
    if (userID) configureRevenueCat(userID);
  }, [userID]);

  const saveTokens = (nextTokens: AuthTokens) => {
    localStorage.setItem(storageKey, JSON.stringify(nextTokens));
    setTokens(nextTokens);
  };

  const value = useMemo<AuthContextValue>(() => ({
    isAuthenticated: Boolean(tokens?.token),
    userID,
    token: tokens?.token ?? null,
    login: async (input) => saveTokens(await loginRequest(input)),
    register: async (input) => saveTokens(await registerRequest(input)),
    logout: () => {
      localStorage.removeItem(storageKey);
      setTokens(null);
    },
  }), [tokens, userID]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
