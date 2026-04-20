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
    async jwt({ token, account, profile }) {
      // Persist the Keycloak access_token and custom claims on first sign-in
      if (account) {
        token.accessToken = account.access_token;
        token.refreshToken = account.refresh_token;
        token.expiresAt = account.expires_at;
      }
      if (profile) {
        const p = profile as Record<string, unknown>;
        const realmAccess = p["realm_access"] as { roles?: string[] } | undefined;
        token.roles = realmAccess?.roles ?? [];
        token.tenant = (p["tenant"] as string[]) ?? [];
      }
      return token;
    },
    async session({ session, token }) {
      session.accessToken = token.accessToken as string;
      session.roles = (token.roles as string[]) ?? [];
      session.tenant = (token.tenant as string[]) ?? [];
      session.userId = token.sub ?? "";
      return session;
    },
  },
  pages: {
    signIn: "/api/auth/signin",
    error: "/admin/unauthorized",
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
