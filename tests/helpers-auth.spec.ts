// Smoke-Test für die Auth-Fixture (PROJ-AUDIT §5h).
// Bestätigt, dass der TestHeaderAuthMiddleware-Pfad in CI funktioniert.
// Lokal ohne TEST_AUTH_MODE=headers wird der Test geskippt (Backend
// erwartet dann echten Keycloak-JWT).

import { test, expect } from "@playwright/test";
import { ensureBackendUp } from "./helpers/backend";
import { superuserHeaders, tenantAdminHeaders } from "./helpers/auth";

const BACKEND = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

test("Auth-Fixture: ohne Header → 401 auf Admin-Endpoint", async ({ request }) => {
  await ensureBackendUp(request);
  const res = await request.get(`${BACKEND}/api/admin/applications`);
  expect(res.status()).toBe(401);
});

test("Auth-Fixture: Tenant-Admin-Header → 200 auf Admin-Listing", async ({ request }) => {
  await ensureBackendUp(request);
  test.skip(process.env.TEST_AUTH_MODE !== "headers",
    "Backend nicht im TEST_AUTH_MODE=headers (lokal ohne CI-Flag)");
  const res = await request.get(`${BACKEND}/api/admin/applications`, {
    headers: tenantAdminHeaders(),
  });
  // Bei aktiver Test-Auth muss der Request durchgehen — auch wenn die
  // Liste leer ist, ist das Ergebnis 200 + JSON-Antwort.
  expect(res.status()).toBe(200);
});

test("Auth-Fixture: Superuser-Header → 200 auf Admin-Listing ohne Tenant", async ({ request }) => {
  await ensureBackendUp(request);
  test.skip(process.env.TEST_AUTH_MODE !== "headers",
    "Backend nicht im TEST_AUTH_MODE=headers (lokal ohne CI-Flag)");
  const res = await request.get(`${BACKEND}/api/admin/applications`, {
    headers: superuserHeaders(),
  });
  expect(res.status()).toBe(200);
});
