"use client";

import { useEffect } from "react";
import { useSession } from "next-auth/react";

// CoreAuthBootstrap installs the Faktura-side core-access-token in the
// NextAuth session when CORE_AUTH_MODE=exchange. It runs once per session.
//
// Flow (only when NEXT_PUBLIC_CORE_AUTH_MODE === "exchange"):
//   1. Session is logged in but session.coreAccessToken is missing/expired
//   2. Component triggers a top-level redirect to the Keycloak authorize
//      endpoint of the Faktura-frontend client (at.ourproject.vfeeg.app)
//      with prompt=none. The user is typically already logged in to
//      eegfaktura.at, so SSO returns immediately without a login dialog.
//   3. Keycloak redirects back to /auth/core-callback with ?code=...
//   4. That callback page exchanges the code via /api/auth/core-token and
//      installs the result in the session via useSession().update(...)
//
// Outside of exchange-mode this component is a no-op.
//
// We mount this once at the admin-layout top-level so every admin route
// has the core-token available without extra wiring per page.
export function CoreAuthBootstrap() {
  const { data: session, status } = useSession();

  useEffect(() => {
    if (process.env.NEXT_PUBLIC_CORE_AUTH_MODE !== "exchange") return;
    if (status !== "authenticated" || !session) return;

    // Token still valid (60s buffer to avoid races) — nothing to do.
    const expiresAt = session.coreExpiresAt ?? 0;
    if (session.coreAccessToken && Date.now() / 1000 < expiresAt - 60) return;

    // We're about to leave the page via top-level redirect. Stash the
    // current URL so the callback can route the user back to where they
    // came from.
    const returnTo = window.location.pathname + window.location.search;
    sessionStorage.setItem("core-auth:return-to", returnTo);

    // Build the authorize URL.
    const issuer = process.env.NEXT_PUBLIC_KEYCLOAK_ISSUER;
    const coreClientId = process.env.NEXT_PUBLIC_KEYCLOAK_CORE_CLIENT_ID;
    if (!issuer || !coreClientId) {
      console.error(
        "[core-auth] NEXT_PUBLIC_KEYCLOAK_ISSUER or NEXT_PUBLIC_KEYCLOAK_CORE_CLIENT_ID not set — cannot bootstrap core auth",
      );
      return;
    }

    const state = crypto.randomUUID();
    sessionStorage.setItem("core-auth:state", state);

    const redirectUri = `${window.location.origin}/auth/core-callback`;
    const authorizeUrl = new URL(`${issuer}/protocol/openid-connect/auth`);
    authorizeUrl.searchParams.set("client_id", coreClientId);
    authorizeUrl.searchParams.set("response_type", "code");
    authorizeUrl.searchParams.set("scope", "openid profile email");
    authorizeUrl.searchParams.set("redirect_uri", redirectUri);
    authorizeUrl.searchParams.set("state", state);
    // prompt=none: if Keycloak has a valid SSO session, it returns a code
    // immediately. If not, it returns an error (login_required) — the
    // callback page handles that gracefully (banner: "bitte erst im
    // Faktura-Frontend einloggen").
    authorizeUrl.searchParams.set("prompt", "none");

    window.location.href = authorizeUrl.toString();
  }, [session, status]);

  return null;
}
