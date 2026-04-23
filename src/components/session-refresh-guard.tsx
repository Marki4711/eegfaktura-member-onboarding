"use client";

import { useSession, signIn } from "next-auth/react";
import { useEffect } from "react";

export function SessionRefreshGuard() {
  const { data: session } = useSession();

  useEffect(() => {
    if (session?.error === "RefreshAccessTokenError") {
      signIn("keycloak");
    }
  }, [session?.error]);

  return null;
}
