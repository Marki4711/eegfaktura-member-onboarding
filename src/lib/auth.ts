import type { NextAuthOptions } from "next-auth";
import KeycloakProvider from "next-auth/providers/keycloak";

export interface KeycloakToken {
  roles: string[];
  tenant: string[];
  sub: string;
  name?: string;
  email?: string;
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
          const ts = new Date().toISOString();
          console.log(`[auth] ${ts} JWT payload keys:`, Object.keys(payload));
          console.log(`[auth] ${ts} realm_access:`, JSON.stringify(realmAccess));
          console.log(`[auth] ${ts} tenant raw:`, JSON.stringify(payload["tenant"]));
          console.log(`[auth] ${ts} tenant parsed:`, JSON.stringify(token.tenant));
          console.log(`[auth] ${ts} roles resolved:`, JSON.stringify(token.roles));
        } catch (e) {
          console.error("[auth] Failed to decode access token:", e);
          token.roles = [];
          token.tenant = [];
        }
      }
      return token;
    },
    async session({ session, token }) {
      session.accessToken = token.accessToken as string;
      session.idToken = token.idToken as string;
      session.roles = (token.roles as string[]) ?? [];
      session.tenant = (token.tenant as string[]) ?? [];
      session.userId = token.sub ?? "";
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
