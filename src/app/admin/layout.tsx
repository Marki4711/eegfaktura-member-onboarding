import Link from "next/link";
import { redirect } from "next/navigation";
import { getServerSession } from "next-auth";
import { Toaster } from "@/components/ui/sonner";
import { AdminLogoutButton } from "@/components/admin-logout-button";
import { authOptions, hasAdminAccess, isSuperuser } from "@/lib/auth";

function getBaseUrl(): string {
  return process.env.BACKEND_URL ?? "http://localhost:8080";
}

export default async function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await getServerSession(authOptions);

  if (!session) {
    redirect(`/api/auth/signin?callbackUrl=${encodeURIComponent("/admin/applications")}`);
  }

  if (!hasAdminAccess(session.roles ?? [], session.tenant ?? [])) {
    redirect("/unauthorized");
  }

  // Trigger sync for tenant-admins once per server render (idempotent via ON CONFLICT DO NOTHING)
  if (
    session.accessToken &&
    !isSuperuser(session.roles ?? []) &&
    (session.tenant ?? []).length > 0
  ) {
    fetch(`${getBaseUrl()}/api/admin/sync`, {
      method: "POST",
      headers: { Authorization: `Bearer ${session.accessToken}` },
    }).catch(() => {/* ignore sync errors */});
  }

  const username =
    (session as unknown as { user?: { name?: string; email?: string } })?.user?.name ??
    (session as unknown as { user?: { name?: string; email?: string } })?.user?.email ??
    "Admin";

  return (
    <div className="min-h-screen bg-background">
      <header className="bg-card border-b border-border">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center gap-6">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 bg-primary flex items-center justify-center shrink-0">
              <svg viewBox="0 0 24 24" fill="none" className="w-4 h-4" aria-hidden="true">
                <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" fill="currentColor" className="text-primary-foreground" />
              </svg>
            </div>
            <span className="font-bold text-foreground text-sm tracking-wide">
              eegFaktura
            </span>
            <span className="text-primary text-xs font-normal">Admin</span>
          </div>
          <nav className="flex gap-4">
            <Link
              href="/admin/applications"
              className="text-sm text-muted-foreground hover:text-primary transition-colors"
            >
              Anträge
            </Link>
          </nav>
          <AdminLogoutButton username={username} keycloakIssuer={process.env.KEYCLOAK_ISSUER!} />
        </div>
      </header>
      <main className="max-w-7xl mx-auto px-6 py-8">{children}</main>
      <Toaster />
    </div>
  );
}
