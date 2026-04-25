import type { NextAuthOptions } from "next-auth";
import type { JWT } from "next-auth/jwt";
import KeycloakProvider from "next-auth/providers/keycloak";

export interface KeycloakToken {
  roles: string[];
  tenant: string[];
  sub: string;
  name?: string;
  email?: string;
}

async function refreshAccessToken(token: JWT): Promise<JWT> {
  const tokenUrl = `${process.env.KEYCLOAK_ISSUER}/protocol/openid-connect/token`;
  const response = await fetch(tokenUrl, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "refresh_token",
      client_id: process.env.KEYCLOAK_CLIENT_ID!,
      client_secret: process.env.KEYCLOAK_CLIENT_SECRET!,
      refresh_token: token.refreshToken ?? "",
    }),
  });

  const refreshed = await response.json() as Record<string, unknown>;
  if (!response.ok) {
    throw new Error((refreshed.error as string) ?? "refresh_failed");
  }

  let roles: string[] = token.roles ?? [];
  let tenant: string[] = token.tenant ?? [];
  try {
    const payload = JSON.parse(
      Buffer.from((refreshed.access_token as string).split(".")[1], "base64url").toString()
    ) as Record<string, unknown>;
    const realmAccess = payload["realm_access"] as { roles?: string[] } | undefined;
    roles = realmAccess?.roles ?? [];
    tenant = parseTenant(payload["tenant"]);
  } catch { /* keep existing values */ }

  return {
    ...token,
    accessToken: refreshed.access_token as string,
    idToken: (refreshed.id_token as string | undefined) ?? token.idToken,
    refreshToken: (refreshed.refresh_token as string | undefined) ?? token.refreshToken,
    expiresAt: Math.floor(Date.now() / 1000) + (refreshed.expires_in as number),
    roles,
    tenant,
    error: undefined,
  };
}

export const authOptions: NextAuthOptions = {
  providers: [
    KeycloakProvider({
      clientId: process.env.KEYCLOAK_CLIENT_ID!,
      clientSecret: process.env.KEYCLOAK_CLIENT_SECRET!,
      issuer: process.env.KEYCLOAK_ISSUER!,
    }),
  ],
  callbacks: {
    async jwt({ token, account }) {
      if (account?.access_token) {
        token.accessToken = account.access_token;
        token.idToken = account.id_token;
        token.refreshToken = account.refresh_token;
        token.expiresAt = account.expires_at;
        // Decode access token — realm_access and tenant are in the access token,
        // not the ID token, so we read them here instead of from profile.
        try {
          const payload = JSON.parse(
            Buffer.from(account.access_token.split(".")[1], "base64url").toString()
          ) as Record<string, unknown>;
          const realmAccess = payload["realm_access"] as { roles?: string[] } | undefined;
          token.roles = realmAccess?.roles ?? [];
          token.tenant = parseTenant(payload["tenant"]);
        } catch (e) {
          console.error("[auth] Failed to decode access token:", e);
          token.roles = [];
          token.tenant = [];
        }
        return token;
      }

      // Token still valid (30s buffer to avoid edge-case expiry mid-request)
      if (Date.now() < (token.expiresAt ?? 0) * 1000 - 30_000) {
        return token;
      }

      // Access token expired — try silent refresh via refresh token
      try {
        return await refreshAccessToken(token);
      } catch (e) {
        console.error("[auth] Token refresh failed:", e);
        return { ...token, error: "RefreshAccessTokenError" };
      }
    },
    async session({ session, token }) {
      session.accessToken = token.accessToken as string;
      session.idToken = token.idToken as string;
      session.roles = (token.roles as string[]) ?? [];
      session.tenant = (token.tenant as string[]) ?? [];
      session.userId = token.sub ?? "";
      if (token.error) session.error = token.error;
      return session;
    },
  },
  pages: {
    error: "/unauthorized",
  },
};

// Keycloak stores the tenant attribute as a JSON array string e.g. '["RC101665","RC101294"]'.
// The mapper emits it as a plain string claim — parse it here into a proper string array.
function parseTenant(raw: unknown): string[] {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw.map(String);
  if (typeof raw === "string") {
    if (raw.startsWith("[")) {
      try {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) return parsed.map(String);
      } catch { /* fall through */ }
    }
    return [raw];
  }
  return [];
}

export function isSuperuser(roles: string[]): boolean {
  return roles.includes("superuser");
}

export function isTenantAdmin(tenant: string[]): boolean {
  return tenant.length > 0;
}

export function hasAdminAccess(roles: string[], tenant: string[]): boolean {
  return isSuperuser(roles) || isTenantAdmin(tenant);
}
