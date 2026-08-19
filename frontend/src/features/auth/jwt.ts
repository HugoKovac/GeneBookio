export function decodeUserID(token: string): string | null {
  try {
    const payload = token.split('.')[1];
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    const claims = JSON.parse(json) as { userID?: string };
    return claims.userID ?? null;
  } catch {
    return null;
  }
}
