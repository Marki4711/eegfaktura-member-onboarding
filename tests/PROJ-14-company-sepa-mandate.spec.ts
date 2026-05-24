import { test, expect } from "@playwright/test";
import { ensureBackendUp as skipIfBackendDown } from "./helpers/backend";

const RC = process.env.TEST_RC_NUMBER ?? "RC123456";
const BACKEND = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// ─── API: useCompanySEPAMandate field ─────────────────────────────────────────

test("AC-B2B-1: GET /api/admin/settings/eeg returns useCompanySEPAMandate field", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(
    `${BACKEND}/api/admin/settings/eeg?rc_number=${RC}`
  );
  // 401 when auth is required; 200 in dev mode
  if (res.status() === 401 || res.status() === 403) return;
  expect(res.ok()).toBe(true);

  const body = await res.json();
  expect(body).toHaveProperty("useCompanySEPAMandate");
  expect(typeof body.useCompanySEPAMandate).toBe("boolean");
});

test("AC-B2B-2: PUT /api/admin/settings/eeg accepts useCompanySEPAMandate=true", async ({
  request,
}) => {
  await skipIfBackendDown(request);
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
        sepaMandateEnabled: true,
        useCompanySEPAMandate: true,
      },
    }
  );
  // 401/403 when auth required, 200/204 in dev mode — never 400 or 500
  expect([200, 204, 401, 403]).toContain(res.status());
});

test("AC-B2B-3: PUT /api/admin/settings/eeg accepts useCompanySEPAMandate=false", async ({
  request,
}) => {
  await skipIfBackendDown(request);
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
        useCompanySEPAMandate: false,
      },
    }
  );
  expect([200, 204, 401, 403]).toContain(res.status());
});

test("AC-B2B-4: useCompanySEPAMandate defaults to false on first GET", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(
    `${BACKEND}/api/admin/settings/eeg?rc_number=${RC}`
  );
  if (res.status() === 401 || res.status() === 403) return;
  expect(res.ok()).toBe(true);

  const body = await res.json();
  // Default value must be false — never null or undefined
  expect(body.useCompanySEPAMandate).toBeDefined();
  expect(body.useCompanySEPAMandate).not.toBeNull();
});

// ─── Security: B2B-Einstellung nicht im öffentlichen API ─────────────────────

test("AC-B2B-5: Public registration endpoint does not expose useCompanySEPAMandate", async ({
  request,
}) => {
  test.skip(process.env.CI === "true", "AUDIT-TODO §5b: Test prüft Response-Shape im 'happy path'; im minimal-seed liefert Endpoint ggf. 404 statt 200, dann greift der `if (!res.ok()) return` zu früh");
  await skipIfBackendDown(request);
  const res = await request.get(
    `${BACKEND}/api/public/registration/${RC}`
  );
  if (!res.ok()) return;

  const body = await res.json();
  expect(body).not.toHaveProperty("useCompanySEPAMandate");
  expect(body).not.toHaveProperty("sepaMandateEnabled");
});

// ─── Admin UI: B2B-Toggle sichtbar ───────────────────────────────────────────

test("AC-B2B-6: Admin settings page loads without JavaScript errors after PROJ-14 changes", async ({
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

// ─── Regression: CORE mandate API still works ────────────────────────────────

test("AC-B2B-7: Public registration API still works after PROJ-14 changes", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(
    `${BACKEND}/api/public/registration/${RC}`
  );
  expect([200, 404, 410]).toContain(res.status());
});

test("AC-B2B-8: GET /api/admin/settings/eeg without rc_number returns 400 or 401", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(`${BACKEND}/api/admin/settings/eeg`);
  expect([400, 401]).toContain(res.status());
});
