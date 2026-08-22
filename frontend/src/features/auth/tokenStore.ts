export type AuthTokens = {
  token: string;
  refresh_token: string;
};

const storageKey = 'genebookio.auth';
const apiBaseUrl = import.meta.env.VITE_API_URL ?? '/api';

type Listener = (tokens: AuthTokens | null) => void;
const listeners = new Set<Listener>();

export function readTokens(): AuthTokens | null {
  try {
    const value = localStorage.getItem(storageKey);
    return value ? (JSON.parse(value) as AuthTokens) : null;
  } catch {
    return null;
  }
}

// writeTokens is the single place session storage is mutated — every
// caller (login/register, a silent refresh, logout) goes through here so
// subscribeTokens listeners (AuthContext's React state) stay in sync even
// when the update happens outside of a React event, e.g. from apiClient's
// background refresh-and-retry.
export function writeTokens(tokens: AuthTokens | null): void {
  if (tokens) {
    localStorage.setItem(storageKey, JSON.stringify(tokens));
  } else {
    localStorage.removeItem(storageKey);
  }
  listeners.forEach((listener) => listener(tokens));
}

export function subscribeTokens(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

let refreshPromise: Promise<AuthTokens> | null = null;

// refreshTokens exchanges the stored refresh token for a new access token
// (POST /users/refresh — backend/internal/user/handler.go). Concurrent
// callers (e.g. several API calls hitting a 401 at once after the app was
// idle) share the same in-flight request instead of each redeeming the
// refresh token separately. A failed refresh clears the session so the app
// falls back to its logged-out state.
export function refreshTokens(): Promise<AuthTokens> {
  if (refreshPromise) return refreshPromise;

  const current = readTokens();
  if (!current) {
    return Promise.reject(new Error('Not authenticated.'));
  }

  refreshPromise = fetch(`${apiBaseUrl}/users/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: current.refresh_token }),
  })
    .then((response) => {
      if (!response.ok) throw new Error('Session expired.');
      return response.json() as Promise<AuthTokens>;
    })
    .then((nextTokens) => {
      writeTokens(nextTokens);
      return nextTokens;
    })
    .catch((cause) => {
      writeTokens(null);
      throw cause;
    })
    .finally(() => {
      refreshPromise = null;
    });

  return refreshPromise;
}
