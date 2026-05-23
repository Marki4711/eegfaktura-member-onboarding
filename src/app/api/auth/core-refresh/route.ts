import { NextRequest, NextResponse } from "next/server";
import { getToken } from "next-auth/jwt";

// POST /api/auth/core-refresh
//
// Refreshes the Faktura-side core access token without going through a fresh
// silent-SSO round-trip. Reads the refresh-token from the NextAuth JWT-Cookie
// (server-side only — the refresh-token is never exposed to the browser),
// hits Keycloak's token endpoint with grant_type=refresh_token, and returns
// the new access-token + expires_at to the caller. The frontend then calls
// session.update() to install the new token in the session.
//
// Errors:
//   - 401: no logged-in session at all → caller should signIn() again
//   - 404: session has no coreRefreshToken yet → caller should trigger the
//          full Authorize-Flow (CoreAuthBootstrap does this)
//   - 502/400: refresh attempt rejected (expired refresh-token, invalid
//          client, etc.) → caller should fall back to full Authorize-Flow
export async function POST(req: NextRequest) {
  const token = await getToken({
    req,
    secret: process.env.NEXTAUTH_SECRET,
  });
  if (!token) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }
  if (!token.coreRefreshToken) {
    return NextResponse.json({ error: "no_refresh_token" }, { status: 404 });
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
    refresh_token: token.coreRefreshToken,
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
    // 400 from Keycloak typically means the refresh-token itself is no longer
    // valid (expired, revoked, used after rotation). Pass that through so the
    // frontend can decide to fall back to a fresh Authorize-Flow.
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
