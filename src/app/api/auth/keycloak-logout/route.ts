import { NextResponse } from "next/server";
import { getServerSession } from "next-auth";
import { authOptions } from "@/lib/auth";

export async function GET() {
  const ts = new Date().toISOString();
  const session = await getServerSession(authOptions);
  const idToken = session?.idToken;

  const issuer = process.env.KEYCLOAK_ISSUER;
  const nextauthUrl = process.env.NEXTAUTH_URL ?? "";

  console.log(`[keycloak-logout] ${ts} idToken present: ${!!idToken}, issuer: ${issuer}, nextauthUrl: ${nextauthUrl}`);

  const endSessionUrl = new URL(`${issuer}/protocol/openid-connect/logout`);
  if (idToken) {
    endSessionUrl.searchParams.set("id_token_hint", idToken);
  }
  endSessionUrl.searchParams.set(
    "post_logout_redirect_uri",
    `${nextauthUrl}/admin/applications`
  );

  console.log(`[keycloak-logout] ${ts} redirecting to: ${endSessionUrl.toString().slice(0, 120)}`);

  // Delete the NextAuth session cookie directly on the redirect response.
  // Using response.cookies is more reliable than cookies() from next/headers
  // when combined with a redirect.
  const response = NextResponse.redirect(endSessionUrl);
  response.cookies.delete("__Secure-next-auth.session-token");
  response.cookies.delete("next-auth.session-token");
  return response;
}
