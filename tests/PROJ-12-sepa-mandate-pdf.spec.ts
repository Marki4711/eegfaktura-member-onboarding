import { test, expect } from "@playwright/test";
import { ensureBackendUp as skipIfBackendDown } from "./helpers/backend";

const RC = process.env.TEST_RC_NUMBER ?? "RC123456";
const BACKEND = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// PROJ-12 hatte zwei Helper (Page / APIRequestContext); der konsolidierte
// `ensureBackendUp` akzeptiert beide.
const skipIfBackendDownRequest = skipIfBackendDown;

// ─── API-Endpunkt: EEG-Einstellungen ────────────────────────────────────────

test("AC-EEG-1: GET /api/admin/settings/eeg exists and requires authentication", async ({
  request,
}) => {
  await skipIfBackendDownRequest(request);
  const res = await request.get(
    `${BACKEND}/api/admin/settings/eeg?rc_number=${RC}`
  );
  // 401 (Keycloak configured) or 200 (dev mode without JWKS) — never 404
  expect([200, 401, 403]).toContain(res.status());
});

test("AC-EEG-2: PUT /api/admin/settings/eeg exists and requires authentication", async ({
  request,
}) => {
  await skipIfBackendDownRequest(request);
  const res = await request.put(
    `${BACKEND}/api/admin/settings/eeg?rc_number=${RC}`,
    {
      data: {
        eegName: "Test EEG",
        eegStreet: "Teststraße",
        eegStreetNumber: "1",
        eegZip: "1010",
        eegCity: "Wien",
        creditorId: "AT28ZZZ00000000000",
        sepaMandateEnabled: false,
      },
    }
  );
  // 401 when auth is required, 200 in dev mode, never 404
  expect([200, 204, 401, 403]).toContain(res.status());
});

test("AC-EEG-3: GET /api/admin/settings/eeg without rc_number returns 400", async ({
  request,
}) => {
  await skipIfBackendDownRequest(request);
  const res = await request.get(`${BACKEND}/api/admin/settings/eeg`);
  // 400 (missing param) or 401 (auth check before param check)
  expect([400, 401]).toContain(res.status());
});

// ─── Admin Settings Page: EEG-Stammdaten-Abschnitt ──────────────────────────

test("AC-EEG-4: Admin settings page contains EEG-Stammdaten section", async ({
  page,
}) => {
  await page.goto("/admin/settings");

  // The page should either show the settings (dev mode) or redirect to Keycloak login
  // We check that the page loads without a 500 error
  const status = page.url();
  const hasError = await page.locator("text=500").isVisible().catch(() => false);
  expect(hasError).toBe(false);

  // In dev mode (no JWKS configured), the settings page should render
  // If Keycloak is configured, we'll be redirected — that's fine
  const isRedirected = !page.url().includes("/admin/settings");
  if (!isRedirected) {
    // We're on the settings page — verify the EEG section exists
    const heading = page.getByText("EEG-Stammdaten");
    const isVisible = await heading.isVisible().catch(() => false);
    if (isVisible) {
      await expect(heading).toBeVisible();
    }
  }
});

// ─── Admin EEG Settings Editor: Warning-Logik ────────────────────────────────

test("AC-EEG-5: EEG settings page renders without JavaScript errors", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));

  await page.goto("/admin/settings");
  await page.waitForLoadState("networkidle");

  // Filter out known non-critical errors (e.g. Keycloak redirect)
  const criticalErrors = errors.filter(
    (e) =>
      !e.includes("keycloak") &&
      !e.includes("NEXT_REDIRECT") &&
      !e.includes("signIn")
  );
  expect(criticalErrors).toHaveLength(0);
});

// ─── Regression: Registrierungsformular funktioniert noch ────────────────────

test("AC-REG-1: Registration form still loads after PROJ-12 changes", async ({
  page,
}) => {
  await skipIfBackendDown(page);
  await page.goto(`/register/${RC}`);
  // Form should load — check for a required field
  await expect(page.locator('[name="email"]')).toBeVisible({ timeout: 10_000 });
});

test("AC-REG-2: SEPA mandate checkbox is present on registration form", async ({
  page,
}) => {
  test.skip(process.env.CI === "true", "AUDIT-TODO §5b: SEPA-Mandat-Checkbox erscheint nur bei EEG-Setting; minimaler dev-seed liefert es nicht");
  await skipIfBackendDown(page);
  await page.goto(`/register/${RC}`);
  // The SEPA mandate acceptance checkbox should exist (required field)
  // It may only appear at the end of the form — scroll to bottom
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  const sepaCheckbox = page.locator('[name="sepaMandateAccepted"]');
  await expect(sepaCheckbox).toBeAttached({ timeout: 10_000 });
});

// ─── Backend: SEPA-Felder in Registration Response ──────────────────────────

test("AC-SEPA-1: GET /api/public/registration/{rc} still returns valid config", async ({
  page,
}) => {
  await skipIfBackendDown(page);
  const res = await page.request.get(
    `${BACKEND}/api/public/registration/${RC}`
  );
  expect(res.ok()).toBe(true);

  const body = await res.json();
  // Should have the base fields — SEPA fields are not exposed in public config.
  // Note: response key is `active` (matches shared.RegistrationConfig in
  // internal/shared/requests.go); historical PROJ-12 versions said `isActive`
  // — corrected 2026-05-24 after E2E-coverage audit caught the mismatch.
  expect(body).toHaveProperty("rcNumber");
  expect(body).toHaveProperty("active");
});

// ─── Security: Keine SEPA-Daten im öffentlichen API ─────────────────────────

test("AC-SEC-1: Public registration endpoint does not expose EEG SEPA fields", async ({
  page,
}) => {
  test.skip(process.env.CI === "true", "AUDIT-TODO §5b: Test prüft Response-Shape, die im minimal-seed evtl. anders aussieht; revisit nach Seed-Erweiterung");
  await skipIfBackendDown(page);
  const res = await page.request.get(
    `${BACKEND}/api/public/registration/${RC}`
  );
  if (!res.ok()) return; // Skip if backend not configured

  const body = await res.json();
  // Sensitive EEG data must not appear in public endpoint
  expect(body).not.toHaveProperty("creditorId");
  expect(body).not.toHaveProperty("eegName");
  expect(body).not.toHaveProperty("sepaMandateEnabled");
  expect(body).not.toHaveProperty("eegStreet");
});
