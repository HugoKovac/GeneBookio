import { refreshTokens } from '../features/auth/tokenStore';

export const apiBaseUrl = import.meta.env.VITE_API_URL ?? '/api';

// authFetch performs an authenticated request against cmd/api. The access
// token is a short-lived JWT (JWT_TOKEN_EXP, 60 minutes — see
// backend/cmd/api/.env), so it routinely expires on a client that's been
// idle or backgrounded ("reconnecting after a while" gets a 401 on
// whatever endpoint happens to load first). Rather than surfacing that 401
// straight to the caller, this redeems the stored refresh token once via
// tokenStore.refreshTokens() and retries the request with the new access
// token. If the refresh itself fails (refresh token also expired/invalid),
// the session is cleared and the original 401 propagates.
export async function authFetch(path: string, token: string, init?: RequestInit): Promise<Response> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: { Authorization: `Bearer ${token}`, ...init?.headers },
  });

  if (response.status !== 401) return response;

  const refreshed = await refreshTokens();

  return fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: { Authorization: `Bearer ${refreshed.token}`, ...init?.headers },
  });
}
