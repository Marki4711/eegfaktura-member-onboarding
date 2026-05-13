"use client";

import { useSession, signIn } from "next-auth/react";
import { useEffect } from "react";

export function SessionRefreshGuard() {
  const { data: session } = useSession();

  // Client-side refresh-token failure (next-auth's signal).
  useEffect(() => {
    if (session?.error === "RefreshAccessTokenError") {
      signIn("keycloak");
    }
  }, [session?.error]);

  // Server-side rejection (backend returned 401). adminRequest emits this
  // event once per session burst; redirect to Keycloak so the user sees a
  // login screen instead of stale error banners from individual API calls.
  useEffect(() => {
    function onAuthExpired() {
      signIn("keycloak");
    }
    window.addEventListener("auth:expired", onAuthExpired);
    return () => window.removeEventListener("auth:expired", onAuthExpired);
  }, []);

  return null;
}
