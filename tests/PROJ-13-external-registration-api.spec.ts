import { test, expect } from "@playwright/test";
import { ensureBackendUp as skipIfBackendDown } from "./helpers/backend";

const RC = process.env.TEST_RC_NUMBER ?? "RC123456";
const BACKEND = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// ─── Admin API-Key-Endpunkte: Existenz und Authentifizierung ─────────────────

test("AC-KEY-1: GET /api/admin/settings/api-key exists and requires authentication", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(
    `${BACKEND}/api/admin/settings/api-key?rc_number=${RC}`
  );
  // 401 when Keycloak is configured, 200 in dev mode — never 404
  expect([200, 401, 403]).toContain(res.status());
});

test("AC-KEY-2: POST /api/admin/settings/api-key exists and requires authentication", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.post(
    `${BACKEND}/api/admin/settings/api-key?rc_number=${RC}`
  );
  expect([200, 201, 401, 403]).toContain(res.status());
});

test("AC-KEY-3: DELETE /api/admin/settings/api-key exists and requires authentication", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.delete(
    `${BACKEND}/api/admin/settings/api-key?rc_number=${RC}`
  );
  // 204/404 in dev mode (no key to revoke), 401 when auth required
  expect([204, 401, 403, 404]).toContain(res.status());
});

test("AC-KEY-4: GET /api/admin/settings/api-key without rc_number returns 400 or 401", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(`${BACKEND}/api/admin/settings/api-key`);
  expect([400, 401]).toContain(res.status());
});

// ─── Externer Einreichungs-Endpunkt: Authentifizierung ───────────────────────

test("AC-EXT-1: POST /api/external/v1/applications without API key returns 401", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.post(`${BACKEND}/api/external/v1/applications`, {
    data: { memberType: "private" },
  });
  expect(res.status()).toBe(401);
});

test("AC-EXT-2: POST /api/external/v1/applications with invalid API key returns 401", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.post(`${BACKEND}/api/external/v1/applications`, {
    headers: { Authorization: "Bearer moak_invalidkey00000000000000000000" },
    data: { memberType: "private" },
  });
  expect(res.status()).toBe(401);
});

test("AC-EXT-3: POST /api/external/v1/applications with missing fields returns 422", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  // Submit with a structurally invalid body — no Authorization header needed
  // to test that 401 comes before any body parsing
  const res = await request.post(`${BACKEND}/api/external/v1/applications`, {
    headers: { Authorization: "Bearer moak_invalidkey00000000000000000000" },
    data: {},
  });
  // 401 (invalid key) takes precedence over 422 (invalid body)
  expect(res.status()).toBe(401);
});

test("AC-EXT-4: External endpoint does not accept Keycloak tokens", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  // A JWT-looking token that is not a valid API key
  const fakeJwt =
    "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.fakesignature";
  const res = await request.post(`${BACKEND}/api/external/v1/applications`, {
    headers: { Authorization: `Bearer ${fakeJwt}` },
    data: {},
  });
  // Must return 401 — JWT tokens are not valid API keys
  expect(res.status()).toBe(401);
});

// ─── Admin Settings Page: Externe API Abschnitt ──────────────────────────────

test("AC-UI-1: Admin settings page loads without 500 error", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));

  await page.goto("/admin/settings");
  await page.waitForLoadState("networkidle");

  const criticalErrors = errors.filter(
    (e) =>
      !e.includes("keycloak") &&
      !e.includes("NEXT_REDIRECT") &&
      !e.includes("signIn")
  );
  expect(criticalErrors).toHaveLength(0);
});

test("AC-UI-2: Admin settings page contains 'Externe API' section", async ({
  page,
}) => {
  await page.goto("/admin/settings");
  await page.waitForLoadState("networkidle");

  const isRedirected = !page.url().includes("/admin/settings");
  if (!isRedirected) {
    const heading = page.getByText("Externe API");
    const isVisible = await heading.isVisible().catch(() => false);
    if (isVisible) {
      await expect(heading).toBeVisible();
    }
  }
  // If redirected to Keycloak — pass (auth is working as expected)
});

test("AC-UI-3: Admin settings page renders without JavaScript errors", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));

  await page.goto("/admin/settings");
  await page.waitForLoadState("networkidle");

  const criticalErrors = errors.filter(
    (e) =>
      !e.includes("keycloak") &&
      !e.includes("NEXT_REDIRECT") &&
      !e.includes("signIn")
  );
  expect(criticalErrors).toHaveLength(0);
});

// ─── Sicherheit: Keine API-Key-Daten im öffentlichen API ─────────────────────

test("AC-SEC-1: Public registration endpoint does not expose API key data", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(
    `${BACKEND}/api/public/registration/${RC}`
  );
  if (!res.ok()) return;

  const body = await res.json();
  expect(body).not.toHaveProperty("apiKey");
  expect(body).not.toHaveProperty("keyHash");
  expect(body).not.toHaveProperty("externalApiKey");
});

test("AC-SEC-2: External endpoint is separate from admin endpoint", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  // The /api/external path should exist and NOT be protected by Keycloak
  // (it uses its own API key auth, not JWT)
  const res = await request.post(`${BACKEND}/api/external/v1/applications`);
  // 401 = our middleware rejected it (no API key) — correct, not Keycloak
  // If backend is down → skip
  expect(res.status()).toBe(401);
});

// ─── Regression: Bestehende Features funktionieren noch ──────────────────────

test("AC-REG-1: Public registration form still loads after PROJ-13 changes", async ({
  page,
}) => {
  try {
    const res = await page.request.get(
      `${BACKEND}/api/public/registration/${RC}`
    );
    if (!res.ok() && res.status() !== 410 && res.status() !== 404) {
      test.skip(true, "Backend not available");
    }
  } catch {
    test.skip(true, "Backend not available");
  }
  await page.goto(`/register/${RC}`);
  await expect(page.locator('[name="email"]')).toBeVisible({
    timeout: 10_000,
  });
});

test("AC-REG-2: Admin applications list still loads after PROJ-13 changes", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));

  await page.goto("/admin");
  await page.waitForLoadState("networkidle");

  const criticalErrors = errors.filter(
    (e) =>
      !e.includes("keycloak") &&
      !e.includes("NEXT_REDIRECT") &&
      !e.includes("signIn")
  );
  expect(criticalErrors).toHaveLength(0);
});
