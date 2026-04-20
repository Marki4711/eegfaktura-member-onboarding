"use client";

import { Button } from "@/components/ui/button";

interface Props {
  username: string;
}

export function AdminLogoutButton({ username }: Props) {
  return (
    <div className="flex items-center gap-3 ml-auto">
      <span className="text-sm text-muted-foreground hidden sm:block">{username}</span>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => { window.location.href = "/api/auth/keycloak-logout"; }}
      >
        Abmelden
      </Button>
    </div>
  );
}
