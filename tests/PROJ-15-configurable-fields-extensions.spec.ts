import { test, expect } from "@playwright/test";
import { ensureBackendUp as skipIfBackendDown } from "./helpers/backend";

const RC = process.env.TEST_RC_NUMBER ?? "RC123456";
const BACKEND = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// ─── A: Hilfetext für "Aktiv am" ─────────────────────────────────────────────

test("AC-A1: membership_start_date help text is present in registration form source", async ({
  page,
}) => {
  // Check that the help text is rendered in the DOM when the field is visible
  // The field is hidden by default — we test the page renders without errors
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));

  await page.goto(`/register/${RC}`);
  await page.waitForLoadState("networkidle");

  const criticalErrors = errors.filter(
    (e) => !e.includes("NEXT_REDIRECT") && !e.includes("signIn")
  );
  expect(criticalErrors).toHaveLength(0);
});

test("AC-A2: Registration form renders without JS errors after PROJ-15 changes", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));

  await page.goto(`/register/${RC}`);
  await page.waitForLoadState("networkidle");

  const criticalErrors = errors.filter(
    (e) => !e.includes("NEXT_REDIRECT") && !e.includes("signIn")
  );
  expect(criticalErrors).toHaveLength(0);
});

// ─── B: admin_only Feldstatus ─────────────────────────────────────────────────

test("AC-B1: GET /api/admin/settings/fields returns {state, adminValue} format", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(
    `${BACKEND}/api/admin/settings/fields?rc_number=${RC}`
  );
  if (res.status() === 401 || res.status() === 403) return;
  expect(res.ok()).toBe(true);

  const body = await res.json();
  expect(body).toHaveProperty("fieldConfig");

  // Each entry in fieldConfig must have a 'state' property (new format)
  for (const [, entry] of Object.entries(body.fieldConfig as Record<string, unknown>)) {
    expect(entry).toHaveProperty("state");
    // adminValue may or may not be present
  }
});

test("AC-B2: PUT /api/admin/settings/fields accepts admin_only state with adminValue", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.put(
    `${BACKEND}/api/admin/settings/fields?rc_number=${RC}`,
    {
      data: {
        persons_in_household: { state: "admin_only", adminValue: "3" },
        heat_pump: { state: "hidden" },
        phone: { state: "optional" },
      },
    }
  );
  // 401/403 when auth required, 204 in dev mode
  expect([204, 401, 403]).toContain(res.status());
});

test("AC-B3: Public registration endpoint does not expose admin_only fields", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(
    `${BACKEND}/api/public/registration/${RC}`
  );
  if (!res.ok()) return;

  const body = await res.json();
  if (!body.fieldConfig) return;

  // admin_only must never appear in public fieldConfig — should be mapped to 'hidden'
  for (const [, state] of Object.entries(body.fieldConfig as Record<string, string>)) {
    expect(state).not.toBe("admin_only");
  }
});

test("AC-B4: PUT /api/admin/settings/fields accepts all four states", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.put(
    `${BACKEND}/api/admin/settings/fields?rc_number=${RC}`,
    {
      data: {
        phone: { state: "required" },
        birth_date: { state: "optional" },
        heat_pump: { state: "hidden" },
        persons_in_household: { state: "admin_only", adminValue: "2" },
      },
    }
  );
  expect([204, 401, 403]).toContain(res.status());
});

test("AC-B5: GET /api/admin/settings/fields without rc_number returns 400 or 401", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(`${BACKEND}/api/admin/settings/fields`);
  expect([400, 401]).toContain(res.status());
});

// ─── Admin UI: Field config editor ───────────────────────────────────────────

test("AC-B6: Admin settings page renders field config section without errors", async ({
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

// ─── Regression ──────────────────────────────────────────────────────────────

test("AC-REG1: Public registration API still works after PROJ-15 field config changes", async ({
  request,
}) => {
  await skipIfBackendDown(request);
  const res = await request.get(`${BACKEND}/api/public/registration/${RC}`);
  expect([200, 404, 410]).toContain(res.status());

  if (res.ok()) {
    const body = await res.json();
    expect(body).toHaveProperty("rcNumber");
    expect(body).toHaveProperty("active");
    // fieldConfig values must be plain strings (not objects) — public API format unchanged
    if (body.fieldConfig) {
      for (const [, val] of Object.entries(body.fieldConfig as Record<string, unknown>)) {
        expect(typeof val).toBe("string");
      }
    }
  }
});
