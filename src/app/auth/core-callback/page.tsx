"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useSession } from "next-auth/react";

// /auth/core-callback — landing page after the silent-SSO authorize call
// in CoreAuthBootstrap. Behaviour:
//   - With a `code` param: exchange it via /api/auth/core-token, install the
//     resulting access-token in the NextAuth session via update(), then
//     redirect back to the URL that initiated the flow.
//   - With an `error` param (typically `login_required` because Keycloak's
//     SSO session was missing): show a banner that asks the user to log in
//     to the Faktura frontend first. No redirect — the user can navigate
//     away manually.
//
// State is verified against what CoreAuthBootstrap stashed in sessionStorage
// to defend against tampered redirects.
//
// Suspense wrapper: Next.js requires useSearchParams() to be inside a
// <Suspense> boundary so the static prerenderer can bail out for the params
// branch without failing the whole build.
export default function CoreCallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="flex h-screen items-center justify-center text-sm text-muted-foreground">
          Faktura-Zugang wird hergestellt …
        </div>
      }
    >
      <CoreCallbackInner />
    </Suspense>
  );
}

function CoreCallbackInner() {
  const params = useSearchParams();
  const router = useRouter();
  const { update } = useSession();
  const [status, setStatus] = useState<"working" | "error" | "ssorequired">("working");
  const [errorMessage, setErrorMessage] = useState<string>("");

  useEffect(() => {
    const code = params.get("code");
    const state = params.get("state");
    const error = params.get("error");

    const expectedState = sessionStorage.getItem("core-auth:state");
    const returnTo = sessionStorage.getItem("core-auth:return-to") ?? "/admin/applications";
    sessionStorage.removeItem("core-auth:state");
    sessionStorage.removeItem("core-auth:return-to");

    if (error) {
      // The common case is `login_required`: Keycloak rejected prompt=none
      // because the user has no SSO session for the Faktura client. Treat as
      // a non-fatal "please log in over there first" state.
      if (error === "login_required" || error === "interaction_required") {
        setStatus("ssorequired");
      } else {
        setStatus("error");
        setErrorMessage(error);
      }
      return;
    }

    if (!code || !state) {
      setStatus("error");
      setErrorMessage("Authorization-Code oder State fehlt in der Callback-URL.");
      return;
    }
    if (state !== expectedState) {
      setStatus("error");
      setErrorMessage("State-Parameter stimmt nicht überein — möglicherweise manipulierter Redirect.");
      return;
    }

    const redirectUri = `${window.location.origin}/auth/core-callback`;
    fetch("/api/auth/core-token", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ code, redirectUri }),
    })
      .then(async (res) => {
        if (!res.ok) {
          const body = (await res.json().catch(() => ({}))) as { error?: string };
          throw new Error(body.error ?? `HTTP ${res.status}`);
        }
        return res.json();
      })
      .then(async (data: { access_token: string; refresh_token?: string; expires_at: number }) => {
        await update({
          type: "core-token",
          accessToken: data.access_token,
          refreshToken: data.refresh_token,
          expiresAt: data.expires_at,
        });
        router.replace(returnTo);
      })
      .catch((err: Error) => {
        setStatus("error");
        setErrorMessage(err.message);
      });
  }, [params, router, update]);

  if (status === "working") {
    return (
      <div className="flex h-screen items-center justify-center text-sm text-muted-foreground">
        Faktura-Zugang wird hergestellt …
      </div>
    );
  }

  if (status === "ssorequired") {
    return (
      <div className="flex h-screen items-center justify-center p-8">
        <div className="max-w-md rounded-md border border-amber-500/40 bg-amber-50 p-6 text-sm text-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
          <div className="font-medium">Faktura-Zugang erforderlich</div>
          <p className="mt-2">
            Bitte logge dich zuerst in eegFaktura ein (anderer Tab oder neues
            Fenster auf <code>https://eegfaktura.at</code>). Danach diese Seite
            neu laden — die Verknüpfung wird dann automatisch hergestellt.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen items-center justify-center p-8">
      <div className="max-w-md rounded-md border border-destructive/40 bg-destructive/10 p-6 text-sm">
        <div className="font-medium">Fehler beim Core-Token-Tausch</div>
        <p className="mt-2 break-words">{errorMessage}</p>
      </div>
    </div>
  );
}
