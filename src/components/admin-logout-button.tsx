"use client";

import { signOut, useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";

interface Props {
  username: string;
  keycloakIssuer: string;
}

export function AdminLogoutButton({ username, keycloakIssuer }: Props) {
  const { data: session } = useSession();

  const handleLogout = async () => {
    const idToken = session?.idToken;

    // Let NextAuth clear all its cookies (session, CSRF, state, PKCE).
    await signOut({ redirect: false });

    // Then terminate the Keycloak SSO session so the user must re-authenticate.
    const endSessionUrl = new URL(`${keycloakIssuer}/protocol/openid-connect/logout`);
    if (idToken) {
      endSessionUrl.searchParams.set("id_token_hint", idToken);
    }
    endSessionUrl.searchParams.set(
      "post_logout_redirect_uri",
      `${window.location.origin}/admin/applications`
    );
    window.location.href = endSessionUrl.toString();
  };

  return (
    <div className="flex items-center gap-3 ml-auto">
      <span className="text-sm text-muted-foreground hidden sm:block">{username}</span>
      <Button variant="ghost" size="sm" onClick={handleLogout}>
        Abmelden
      </Button>
    </div>
  );
}
