import { NextRequest, NextResponse } from "next/server";
import { getServerSession } from "next-auth";
import { authOptions } from "@/lib/auth";

// POST /api/auth/core-refresh
//
// Refreshes the Faktura-side core access-token without going through a fresh
// silent-SSO round-trip. The refresh-token is sent in the body by the
// caller (CoreAuthBootstrap) — we cannot pull it from the NextAuth JWT-Cookie
// because session.update() does not work reliably in NextAuth v4.24, so the
// core-token lives in localStorage instead of the encrypted session cookie.
//
// Auth: the caller must still hold a valid Onboarding session — otherwise
// this is a token-exchange oracle for anyone with a stolen refresh-token.
//
// Body: { refreshToken: string }
// Response: { access_token, refresh_token?, expires_at }
//
// Errors:
//   - 401: no logged-in session at all
//   - 400: malformed body / refresh-token rejected by Keycloak
//   - 503: KEYCLOAK_ISSUER or KEYCLOAK_CORE_CLIENT_ID missing
//   - 502: Keycloak unreachable
export async function POST(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }

  let body: { refreshToken?: unknown };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "invalid_request_body" }, { status: 400 });
  }
  const refreshToken = typeof body.refreshToken === "string" ? body.refreshToken : "";
  if (!refreshToken) {
    return NextResponse.json({ error: "missing_refresh_token" }, { status: 400 });
  }

  const issuer = process.env.KEYCLOAK_ISSUER;
  const coreClientId = process.env.KEYCLOAK_CORE_CLIENT_ID;
  if (!issuer || !coreClientId) {
    return NextResponse.json(
      { error: "core_client_not_configured" },
      { status: 503 },
    );
  }

  const params = new URLSearchParams({
    grant_type: "refresh_token",
    client_id: coreClientId,
    refresh_token: refreshToken,
  });

  const refreshRes = await fetch(`${issuer}/protocol/openid-connect/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: params,
    cache: "no-store",
  });

  if (!refreshRes.ok) {
    let detail = "refresh_failed";
    try {
      const errBody = (await refreshRes.json()) as { error_description?: string };
      if (errBody.error_description) detail = errBody.error_description;
    } catch {
      /* keep generic */
    }
    return NextResponse.json(
      { error: detail },
      { status: refreshRes.status === 400 ? 400 : 502 },
    );
  }

  const tokenData = (await refreshRes.json()) as {
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
