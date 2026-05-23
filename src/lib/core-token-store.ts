// LocalStorage-based store for the Faktura-side core access-token
// (CORE_AUTH_MODE=exchange path). We started by routing the token through
// the NextAuth session via useSession().update(), but in NextAuth v4.24 the
// update() resolves to undefined and never POSTs to /api/auth/session — so
// the jwt-callback never runs with trigger="update" and the token never
// reaches the session cookie. Rather than fight the NextAuth internals we
// keep the core-token in localStorage and read it directly where needed.
//
// Security note: localStorage is JS-readable, so an XSS payload could
// exfiltrate the token. That is the same risk profile as the existing
// NextAuth accessToken (which we already pass to fetch() from JS, see
// adminRequest in api.ts). Mitigations remain CSP + sanitiser layers, not
// storage choice.

const ACCESS_TOKEN_KEY = "core-auth:access-token";
const REFRESH_TOKEN_KEY = "core-auth:refresh-token";
const EXPIRES_AT_KEY = "core-auth:expires-at";

export interface CoreTokenSnapshot {
  accessToken: string;
  refreshToken?: string;
  expiresAt: number; // wall-clock seconds since epoch
}

export function loadCoreToken(): CoreTokenSnapshot | null {
  if (typeof window === "undefined") return null;
  const accessToken = localStorage.getItem(ACCESS_TOKEN_KEY);
  const expiresAtRaw = localStorage.getItem(EXPIRES_AT_KEY);
  if (!accessToken || !expiresAtRaw) return null;
  const expiresAt = parseInt(expiresAtRaw, 10);
  if (Number.isNaN(expiresAt)) return null;
  return {
    accessToken,
    refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY) ?? undefined,
    expiresAt,
  };
}

export function saveCoreToken(snapshot: CoreTokenSnapshot): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(ACCESS_TOKEN_KEY, snapshot.accessToken);
  localStorage.setItem(EXPIRES_AT_KEY, snapshot.expiresAt.toString());
  if (snapshot.refreshToken) {
    localStorage.setItem(REFRESH_TOKEN_KEY, snapshot.refreshToken);
  }
}

export function clearCoreToken(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(EXPIRES_AT_KEY);
}
