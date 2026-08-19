import { useMemo, useState, type ReactNode } from 'react';
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

  const saveTokens = (nextTokens: AuthTokens) => {
    localStorage.setItem(storageKey, JSON.stringify(nextTokens));
    setTokens(nextTokens);
  };

  const value = useMemo<AuthContextValue>(() => ({
    isAuthenticated: Boolean(tokens?.token),
    userID: tokens?.token ? decodeUserID(tokens.token) : null,
    token: tokens?.token ?? null,
    login: async (input) => saveTokens(await loginRequest(input)),
    register: async (input) => saveTokens(await registerRequest(input)),
    logout: () => {
      localStorage.removeItem(storageKey);
      setTokens(null);
    },
  }), [tokens]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
