import { NextRequest, NextResponse } from "next/server";
import { getServerSession } from "next-auth";
import { authOptions } from "@/lib/auth";

// POST /api/auth/core-token
//
// Server-side exchange of a Keycloak authorisation code (obtained by the
// silent-SSO bootstrap against the Faktura-frontend Keycloak client) for an
// access token. The exchanged token has azp = at.ourproject.vfeeg.app, which
// the Faktura backend accepts whereas it silently rejects tokens with
// azp = eegfaktura-member-onboarding (the stealth-filter bug, see
// docs/architecture.md "Core Auth Mode").
//
// Auth: caller must be logged in to the Onboarding (regular session). The
// route is not callable anonymously — without an Onboarding session there is
// no place to store the exchanged token anyway.
//
// Body: { code: string, redirectUri: string }
// Response: { access_token, refresh_token?, expires_at, error? }
//
// expires_at is wall-clock seconds (not relative seconds), so the frontend
// can compare against Date.now()/1000 without re-arithmetic.
export async function POST(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }

  let body: { code?: unknown; redirectUri?: unknown };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "invalid_request_body" }, { status: 400 });
  }

  const code = typeof body.code === "string" ? body.code : "";
  const redirectUri = typeof body.redirectUri === "string" ? body.redirectUri : "";
  if (!code || !redirectUri) {
    return NextResponse.json({ error: "missing_code_or_redirect_uri" }, { status: 400 });
  }

  const issuer = process.env.KEYCLOAK_ISSUER;
  const coreClientId = process.env.KEYCLOAK_CORE_CLIENT_ID;
  if (!issuer || !coreClientId) {
    return NextResponse.json(
      { error: "core_client_not_configured" },
      { status: 503 },
    );
  }

  // Public client (no client_secret). The redirect_uri MUST match exactly the
  // value used at the Authorize step — Keycloak rejects with 400 otherwise.
  const params = new URLSearchParams({
    grant_type: "authorization_code",
    client_id: coreClientId,
    code,
    redirect_uri: redirectUri,
  });

  const tokenRes = await fetch(`${issuer}/protocol/openid-connect/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: params,
    cache: "no-store",
  });

  if (!tokenRes.ok) {
    let detail = "exchange_failed";
    try {
      const errBody = (await tokenRes.json()) as { error_description?: string };
      if (errBody.error_description) detail = errBody.error_description;
    } catch {
      /* ignore — keep generic message */
    }
    return NextResponse.json(
      { error: detail },
      { status: tokenRes.status === 400 || tokenRes.status === 401 ? tokenRes.status : 502 },
    );
  }

  const tokenData = (await tokenRes.json()) as {
    access_token: string;
    refresh_token?: string;
    expires_in: number;
  };

  return NextResponse.json({
    access_token: tokenData.access_token,
    refresh_token: tokenData.refresh_token,
    expires_at: Math.floor(Date.now() / 1000) + tokenData.expires_in,
  });
}
