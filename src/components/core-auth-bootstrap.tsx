"use client";

import { useEffect, useRef } from "react";
import { useSession } from "next-auth/react";

// CoreAuthBootstrap installs and keeps fresh the Faktura-side core-access-
// token in the NextAuth session when CORE_AUTH_MODE=exchange. Runs only on
// the admin pages (mounted from admin/layout.tsx).
//
// Config is passed as props from the parent server-component because
// Next.js inlines `process.env.NEXT_PUBLIC_*` at build-time into client
// bundles — runtime helm env changes would otherwise have no effect.
//
// State machine:
//   - No coreAccessToken at all → trigger full Authorize-Flow (top-level
//     redirect to Keycloak with prompt=none).
//   - coreAccessToken present, > refreshLeadSeconds remaining → idle.
//   - coreAccessToken present, < refreshLeadSeconds remaining → silent
//     server-side refresh via /api/auth/core-refresh. On success: install
//     the new token via session.update(). On 404 (no refresh-token) or any
//     non-200: fall back to the full Authorize-Flow.
//
// Polling cadence: refreshCheckIntervalMs (every 30 s). Cheap — just a few
// timestamp comparisons, no network call unless the token actually needs
// refreshing.
//
// Outside of exchange-mode this component is a no-op.

// How early (in seconds) before token expiry we refresh. Picked larger than
// the SetInterval cadence so we always have a refresh opportunity before
// the token actually dies.
const refreshLeadSeconds = 60;

// How often we check whether the token needs refreshing. Independent of
// expiry — we just sample wall-clock and compare against the stored
// expiresAt timestamp.
const refreshCheckIntervalMs = 30_000;

interface Props {
  // "direct" | "exchange" — passed from the server layout so we don't
  // depend on build-time inlining of NEXT_PUBLIC_* in this client bundle.
  authMode: string;
  // Keycloak issuer URL (e.g. https://login.eegfaktura.at/realms/EEGFaktura).
  issuer: string;
  // Public client-id of the Faktura-frontend Keycloak client.
  coreClientId: string;
}

export function CoreAuthBootstrap({ authMode, issuer, coreClientId }: Props) {
  const { data: session, status, update } = useSession();

  // Guards against concurrent refresh attempts (e.g. if two interval ticks
  // happen to overlap with a slow Keycloak response, or React re-renders
  // during the await).
  const refreshInFlight = useRef(false);

  useEffect(() => {
    if (authMode !== "exchange") return;
    if (status !== "authenticated" || !session) return;

    let cancelled = false;

    const handle = setInterval(() => {
      if (cancelled) return;
      void evaluate();
    }, refreshCheckIntervalMs);

    // Also evaluate immediately on mount — don't wait for the first tick.
    void evaluate();

    return () => {
      cancelled = true;
      clearInterval(handle);
    };

    async function evaluate() {
      if (refreshInFlight.current) return;
      const expiresAt = session?.coreExpiresAt ?? 0;
      const nowSec = Date.now() / 1000;
      const hasToken = !!session?.coreAccessToken;
      const expiringSoon = expiresAt - nowSec < refreshLeadSeconds;

      // Case 1: token is still healthy — nothing to do.
      if (hasToken && !expiringSoon) return;

      // Case 2: token is present but about to expire — try the cheap path
      // (server-side refresh-token flow) first.
      if (hasToken && expiringSoon) {
        refreshInFlight.current = true;
        try {
          const res = await fetch("/api/auth/core-refresh", { method: "POST" });
          if (res.ok) {
            const data = (await res.json()) as {
              access_token: string;
              refresh_token?: string;
              expires_at: number;
            };
            await update({
              type: "core-token",
              accessToken: data.access_token,
              refreshToken: data.refresh_token,
              expiresAt: data.expires_at,
            });
            return;
          }
          // Refresh failed (expired refresh-token, etc.) — fall through to
          // the full Authorize-Flow below. Don't log loudly: this is an
          // expected branch on Keycloak session timeout.
          console.warn("[core-auth] refresh-token flow failed, falling back to Authorize");
        } catch (e) {
          console.warn("[core-auth] refresh-token network error, falling back to Authorize", e);
        } finally {
          refreshInFlight.current = false;
        }
      }

      // Case 3: no token at all OR refresh failed — start a fresh
      // Authorize-Flow. We're about to leave the page via top-level redirect,
      // so stash the current URL so the callback can return the user here.
      if (cancelled) return;
      const returnTo = window.location.pathname + window.location.search;
      sessionStorage.setItem("core-auth:return-to", returnTo);

      if (!issuer || !coreClientId) {
        console.error(
          "[core-auth] issuer or coreClientId not set on CoreAuthBootstrap props — cannot bootstrap core auth",
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
      // prompt=none returns a code immediately if Keycloak has a valid SSO
      // session, or an error (login_required) otherwise — the callback
      // page handles both branches.
      authorizeUrl.searchParams.set("prompt", "none");

      window.location.href = authorizeUrl.toString();
    }
  }, [session, status, update, authMode, issuer, coreClientId]);

  return null;
}
