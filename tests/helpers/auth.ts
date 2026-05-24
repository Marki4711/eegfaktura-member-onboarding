// E2E-Auth-Helper für Admin-API-Pfade.
//
// Backend in CI läuft mit TEST_AUTH_MODE=headers (siehe
// .github/workflows/ci.yml + internal/http/auth_middleware.go).
// Damit dürfen Tests synthetische Claims via Header injizieren statt
// einen echten Keycloak-Token zu beschaffen.
//
// Tenant-Admin (Zugriff auf bestimmte RC-Numbers):
//   const headers = adminAuthHeaders({ tenant: [TEST_RC_NUMBER] });
//
// Superuser (alle RC-Numbers, kein Tenant-Check):
//   const headers = adminAuthHeaders({ superuser: true });
//
// Ohne Header → 401, mit Header → Request läuft. So sind Auth-Required-Tests
// und authenticated-Path-Tests aus derselben Spec heraus möglich.
//
// Sicherheit: die Header werden vom Backend nur akzeptiert, wenn
// TEST_AUTH_MODE=headers gesetzt ist. Production verweigert den Start mit
// diesem Flag (siehe cmd/server/main.go). Lokal ohne das Flag liefert das
// Backend regulär 401, die Specs sind ENV-agnostisch.

import { TEST_RC_NUMBER } from "./test-data";

export type AdminAuthOptions = {
  tenant?: string[];
  superuser?: boolean;
  subject?: string;
};

export function adminAuthHeaders(opts: AdminAuthOptions = {}): Record<string, string> {
  const headers: Record<string, string> = {};
  if (opts.tenant && opts.tenant.length > 0) {
    headers["X-Test-Tenant"] = opts.tenant.join(",");
  }
  if (opts.superuser) {
    headers["X-Test-Superuser"] = "true";
  }
  if (opts.subject) {
    headers["X-Test-Subject"] = opts.subject;
  }
  return headers;
}

// Convenience: häufigste Variante — Tenant-Admin für die test-RC-Number.
export function tenantAdminHeaders(): Record<string, string> {
  return adminAuthHeaders({ tenant: [TEST_RC_NUMBER] });
}

// Convenience: Superuser.
export function superuserHeaders(): Record<string, string> {
  return adminAuthHeaders({ superuser: true });
}
