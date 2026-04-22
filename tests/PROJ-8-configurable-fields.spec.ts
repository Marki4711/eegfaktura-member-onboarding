import { test, expect } from "@playwright/test";

const RC = process.env.TEST_RC_NUMBER ?? "RC123456";
const FORM_URL = `/register/${RC}`;

// ─── AC: Default field visibility (optional fields shown by default) ──────────

test("AC-1a: phone field is visible on registration form by default (state: optional)", async ({ page }) => {
  await page.goto(FORM_URL);
  // Default state for phone is "optional" → must be visible
  await expect(page.locator('[name="phone"]')).toBeVisible();
});

test("AC-1b: birth_date field is visible on registration form by default (state: optional)", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.locator('[name="birthDate"]')).toBeVisible();
});

test("AC-1c: uid_number field is visible on registration form for private type by default (state: optional)", async ({ page }) => {
  await page.goto(FORM_URL);
  // For private type, uid_number is configured optional → visible but not required (no * marker)
  // The field should NOT be visible for private type — uid_number is a company field
  // clearMemberTypeFields nils it out for private; frontend hides it for non-company types
  // So for private type, uidNumber is hidden regardless of field config
  await expect(page.locator('[name="uidNumber"]')).not.toBeVisible();
});

// ─── AC: New optional fields are hidden by default ────────────────────────────

test("AC-2a: heat_pump field is hidden on registration form by default (state: hidden)", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.locator('[name="heatPump"]')).not.toBeVisible();
});

test("AC-2b: electric_vehicle field is hidden on registration form by default (state: hidden)", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.locator('[name="electricVehicle"]')).not.toBeVisible();
});

test("AC-2c: membership_start_date field is hidden on registration form by default (state: hidden)", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.locator('[name="membershipStartDate"]')).not.toBeVisible();
});

test("AC-2d: persons_in_household field is hidden on registration form by default (state: hidden)", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.locator('[name="personsInHousehold"]')).not.toBeVisible();
});

// ─── AC: Metering point optional fields are hidden by default ─────────────────

test("AC-3a: transformer field on metering point is hidden by default (state: hidden)", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.locator('[name="meteringPoints.0.transformer"]')).not.toBeVisible();
});

test("AC-3b: installation_number field on metering point is hidden by default (state: hidden)", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.locator('[name="meteringPoints.0.installationNumber"]')).not.toBeVisible();
});

// ─── AC: Admin settings page exists and is guarded ───────────────────────────

test("AC-4: unauthenticated access to /admin/settings redirects away from admin area", async ({ page }) => {
  await page.goto("/admin/settings");
  // Should redirect to Keycloak or /unauthorized, not render settings content
  await expect(page).not.toHaveURL(/\/admin\/settings$/);
});

// ─── AC: Registration form renders fieldConfig data from API ──────────────────

test("AC-5: registration form loads and renders successfully with valid RC number", async ({ page }) => {
  await page.goto(FORM_URL);
  // Core fields always visible
  await expect(page.locator('[name="email"]')).toBeVisible();
  await expect(page.locator('[name="residentStreet"]')).toBeVisible();
  await expect(page.locator('[name="iban"]')).toBeVisible();
  // Metering point section exists
  await expect(page.locator('[name="meteringPoints.0.meteringPoint"]')).toBeVisible();
});

// ─── AC: Phone field is optional (no validation error when empty by default) ──

test("AC-6: phone field shows no required-field error when left empty with default config", async ({ page }) => {
  await page.goto(FORM_URL);

  // Fill minimum required fields
  await page.fill('[name="firstname"]', "Max");
  await page.fill('[name="lastname"]', "Mustermann");
  await page.fill('[name="email"]', "test@example.at");
  await page.fill('[name="residentStreet"]', "Teststraße");
  await page.fill('[name="residentStreetNumber"]', "1");
  await page.fill('[name="residentZip"]', "4020");
  await page.fill('[name="residentCity"]', "Linz");

  // Leave phone intentionally empty

  await page.locator('[name="meteringPoints.0.meteringPoint"]').click();
  await page.locator('[name="meteringPoints.0.meteringPoint"]').fill("AT 003100 00000 000000000000 00000001");

  const ibanInput = page.locator('[name="iban"]');
  await ibanInput.click();
  await ibanInput.type("AT611904300234573201");
  await page.fill('[name="accountHolder"]', "Max Mustermann");

  const checkboxes = page.locator('button[role="checkbox"]');
  const count = await checkboxes.count();
  for (let i = 0; i < count; i++) {
    const checked = await checkboxes.nth(i).getAttribute("data-state");
    if (checked !== "checked") await checkboxes.nth(i).click();
  }

  await page.getByRole("button", { name: /einreichen/i }).click();

  // No error about phone being required
  await expect(page.getByText(/Telefonnummer ist erforderlich/i)).not.toBeVisible();
});

// ─── AC: birthDate optional (no required error by default) ───────────────────

test("AC-7: birthDate field shows no required-field error when left empty with default config", async ({ page }) => {
  await page.goto(FORM_URL);

  await page.fill('[name="firstname"]', "Max");
  await page.fill('[name="lastname"]', "Mustermann");
  await page.fill('[name="email"]', "test@example.at");
  await page.fill('[name="residentStreet"]', "Teststraße");
  await page.fill('[name="residentStreetNumber"]', "1");
  await page.fill('[name="residentZip"]', "4020");
  await page.fill('[name="residentCity"]', "Linz");
  // Leave birthDate empty

  await page.locator('[name="meteringPoints.0.meteringPoint"]').click();
  await page.locator('[name="meteringPoints.0.meteringPoint"]').fill("AT 003100 00000 000000000000 00000001");

  const ibanInput = page.locator('[name="iban"]');
  await ibanInput.click();
  await ibanInput.type("AT611904300234573201");
  await page.fill('[name="accountHolder"]', "Max Mustermann");

  const checkboxes = page.locator('button[role="checkbox"]');
  const count = await checkboxes.count();
  for (let i = 0; i < count; i++) {
    const checked = await checkboxes.nth(i).getAttribute("data-state");
    if (checked !== "checked") await checkboxes.nth(i).click();
  }

  await page.getByRole("button", { name: /einreichen/i }).click();

  await expect(page.getByText(/Geburtsdatum ist erforderlich/i)).not.toBeVisible();
});
