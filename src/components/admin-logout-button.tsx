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

    // Clear the NextAuth session without triggering a redirect
    await signOut({ redirect: false });

    // Redirect to Keycloak's end_session endpoint to terminate the SSO session.
    // Keycloak then redirects back to /admin/applications, which re-triggers
    // the admin layout's signin redirect with the correct callbackUrl.
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
