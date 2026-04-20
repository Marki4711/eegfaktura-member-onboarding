import { NextResponse } from "next/server";
import { getServerSession } from "next-auth";
import { authOptions } from "@/lib/auth";
import { cookies } from "next/headers";

export async function GET() {
  const session = await getServerSession(authOptions);
  const idToken = session?.idToken;

  const issuer = process.env.KEYCLOAK_ISSUER;
  const nextauthUrl = process.env.NEXTAUTH_URL ?? "";

  // Delete the NextAuth session cookie so the user is logged out locally.
  // NextAuth uses __Secure- prefix on HTTPS, plain prefix otherwise.
  const cookieStore = await cookies();
  cookieStore.delete("__Secure-next-auth.session-token");
  cookieStore.delete("next-auth.session-token");

  // Redirect to Keycloak's end_session endpoint to terminate the SSO session.
  const endSessionUrl = new URL(`${issuer}/protocol/openid-connect/logout`);
  if (idToken) {
    endSessionUrl.searchParams.set("id_token_hint", idToken);
  }
  endSessionUrl.searchParams.set(
    "post_logout_redirect_uri",
    `${nextauthUrl}/admin/applications`
  );

  return NextResponse.redirect(endSessionUrl);
}
