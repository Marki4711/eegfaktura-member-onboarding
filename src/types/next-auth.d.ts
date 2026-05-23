import "next-auth";
import "next-auth/jwt";

declare module "next-auth" {
  interface Session {
    accessToken: string;
    idToken: string;
    roles: string[];
    tenant: string[];
    userId: string;
    error?: string;
    // CORE_AUTH_MODE=exchange: separate token for outgoing Faktura-Core
    // REST-Calls, obtained via silent SSO against the Faktura-frontend
    // Keycloak client. Undefined when the mode is "direct" or the SSO
    // bootstrap has not yet completed.
    coreAccessToken?: string;
    coreExpiresAt?: number;
    coreError?: string;
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    accessToken?: string;
    idToken?: string;
    refreshToken?: string;
    expiresAt?: number;
    roles?: string[];
    tenant?: string[];
    error?: string;
    coreAccessToken?: string;
    coreRefreshToken?: string;
    coreExpiresAt?: number;
    coreError?: string;
  }
}
