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
          token.tenant = (payload["tenant"] as string[]) ?? [];
          const ts = new Date().toISOString();
          console.log(`[auth] ${ts} JWT payload keys:`, Object.keys(payload));
          console.log(`[auth] ${ts} realm_access:`, JSON.stringify(realmAccess));
          console.log(`[auth] ${ts} tenant:`, JSON.stringify(payload["tenant"]));
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

export function isSuperuser(roles: string[]): boolean {
  return roles.includes("superuser");
}

export function isTenantAdmin(tenant: string[]): boolean {
  return tenant.length > 0;
}

export function hasAdminAccess(roles: string[], tenant: string[]): boolean {
  return isSuperuser(roles) || isTenantAdmin(tenant);
}
