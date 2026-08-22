function decodeClaims(token: string): Record<string, unknown> | null {
  try {
    const payload = token.split('.')[1];
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(json) as Record<string, unknown>;
  } catch {
    return null;
  }
}

export function decodeUserID(token: string): string | null {
  const claims = decodeClaims(token);
  return typeof claims?.userID === 'string' ? claims.userID : null;
}

// decodeExpiry returns the token's "exp" claim in epoch milliseconds, or
// null if it's missing/unparsable.
export function decodeExpiry(token: string): number | null {
  const claims = decodeClaims(token);
  return typeof claims?.exp === 'number' ? claims.exp * 1000 : null;
}
