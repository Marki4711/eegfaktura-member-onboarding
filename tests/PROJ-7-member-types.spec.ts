import { test, expect } from "@playwright/test";

// Uses a real RC number seeded in dev DB. Override with env var if needed.
const RC = process.env.TEST_RC_NUMBER ?? "TEST-RC-001";
const FORM_URL = `/register/${RC}`;

// Shared valid address/bank data to fill out non-type-specific sections.
async function fillCommonFields(page: import("@playwright/test").Page) {
  await page.fill('[name="email"]', "test@example.at");
  await page.fill('[name="residentStreet"]', "Teststraße");
  await page.fill('[name="residentStreetNumber"]', "1");
  await page.fill('[name="residentZip"]', "4020");
  await page.fill('[name="residentCity"]', "Linz");
  // IBAN masked input — type into the underlying input
  const ibanInput = page.locator('[name="iban"]');
  await ibanInput.click();
  await ibanInput.type("AT611904300234573201");
  await page.fill('[name="accountHolder"]', "Test Inhaber");
  // Metering point
  await page.locator('input[placeholder*="AT003"]').first().fill("AT003100000000000000000000000001");
}

async function acceptConsents(page: import("@playwright/test").Page) {
  const checkboxes = page.locator('button[role="checkbox"]');
  const count = await checkboxes.count();
  for (let i = 0; i < count; i++) {
    const checked = await checkboxes.nth(i).getAttribute("data-state");
    if (checked !== "checked") {
      await checkboxes.nth(i).click();
    }
  }
}

// ─── AC: Typ-Auswahl ──────────────────────────────────────────────────────────

test("AC-1: member type selector shows four options", async ({ page }) => {
  await page.goto(FORM_URL);
  const options = page.locator('[id^="mt-"]');
  await expect(options).toHaveCount(4);
});

test("AC-2: default selection is 'Privat / Kleinunternehmer'", async ({ page }) => {
  await page.goto(FORM_URL);
  const privateRadio = page.locator("#mt-private");
  await expect(privateRadio).toBeChecked();
});

test("AC-3: member type options show VAT hint", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.getByText("0 % USt.")).toBeVisible();
  await expect(page.getByText("13 % USt.")).toBeVisible();
  await expect(page.getByText("20 % USt.")).toBeVisible();
});

test("AC-4: selecting a type updates the form fields immediately", async ({ page }) => {
  await page.goto(FORM_URL);
  // Initially person fields visible
  await expect(page.locator('[name="firstname"]')).toBeVisible();
  // Switch to company
  await page.locator("#mt-company").click();
  // Person fields gone, org fields appear
  await expect(page.locator('[name="firstname"]')).not.toBeVisible();
  await expect(page.locator('[name="companyName"]')).toBeVisible();
});

// ─── AC: Felder je Typ ───────────────────────────────────────────────────────

test("AC-5: private type shows firstname, lastname, birthDate as required", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.locator('[name="firstname"]')).toBeVisible();
  await expect(page.locator('[name="lastname"]')).toBeVisible();
  await expect(page.locator('[name="birthDate"]')).toBeVisible();
  // Org fields must not be visible
  await expect(page.locator('[name="companyName"]')).not.toBeVisible();
  await expect(page.locator('[name="uidNumber"]')).not.toBeVisible();
});

test("AC-6: farmer type shows same person fields as private", async ({ page }) => {
  await page.goto(FORM_URL);
  await page.locator("#mt-farmer").click();
  await expect(page.locator('[name="firstname"]')).toBeVisible();
  await expect(page.locator('[name="lastname"]')).toBeVisible();
  await expect(page.locator('[name="birthDate"]')).toBeVisible();
  await expect(page.locator('[name="companyName"]')).not.toBeVisible();
});

test("AC-7: municipality type shows companyName (required) and uidNumber (optional), no person fields", async ({ page }) => {
  await page.goto(FORM_URL);
  await page.locator("#mt-municipality").click();
  await expect(page.locator('[name="companyName"]')).toBeVisible();
  await expect(page.locator('[name="uidNumber"]')).toBeVisible();
  await expect(page.locator('[name="registerNumber"]')).not.toBeVisible();
  await expect(page.locator('[name="firstname"]')).not.toBeVisible();
  await expect(page.locator('[name="lastname"]')).not.toBeVisible();
  await expect(page.locator('[name="birthDate"]')).not.toBeVisible();
});

test("AC-8: company type shows companyName, uidNumber, registerNumber as required", async ({ page }) => {
  await page.goto(FORM_URL);
  await page.locator("#mt-company").click();
  await expect(page.locator('[name="companyName"]')).toBeVisible();
  await expect(page.locator('[name="uidNumber"]')).toBeVisible();
  await expect(page.locator('[name="registerNumber"]')).toBeVisible();
  await expect(page.locator('[name="firstname"]')).not.toBeVisible();
});

test("AC-9: switching type clears type-specific fields", async ({ page }) => {
  await page.goto(FORM_URL);
  // Fill person fields
  await page.fill('[name="firstname"]', "Max");
  await page.fill('[name="lastname"]', "Muster");
  // Switch to company
  await page.locator("#mt-company").click();
  // Switch back to private
  await page.locator("#mt-private").click();
  await expect(page.locator('[name="firstname"]')).toHaveValue("");
  await expect(page.locator('[name="lastname"]')).toHaveValue("");
});

// ─── AC: Client-side validation errors ───────────────────────────────────────

test("AC-10: private type shows error when firstname is empty on submit", async ({ page }) => {
  await page.goto(FORM_URL);
  await fillCommonFields(page);
  await page.fill('[name="lastname"]', "Muster");
  await page.fill('[name="birthDate"]', "1990-01-01");
  await acceptConsents(page);
  await page.getByRole("button", { name: /einreichen/i }).click();
  await expect(page.getByText(/Vorname ist erforderlich/i)).toBeVisible();
});

test("AC-11: company type shows errors when uid and registerNumber missing", async ({ page }) => {
  await page.goto(FORM_URL);
  await page.locator("#mt-company").click();
  await fillCommonFields(page);
  await page.fill('[name="companyName"]', "Muster GmbH");
  // uid and registerNumber intentionally left empty
  await acceptConsents(page);
  await page.getByRole("button", { name: /einreichen/i }).click();
  await expect(page.getByText(/UID-Nummer ist erforderlich/i)).toBeVisible();
  await expect(page.getByText(/Firmenbuch/i)).toBeVisible();
});

test("AC-12: municipality type does not require uidNumber", async ({ page }) => {
  await page.goto(FORM_URL);
  await page.locator("#mt-municipality").click();
  await fillCommonFields(page);
  await page.fill('[name="companyName"]', "Gemeinde Musterort");
  // uid intentionally left empty — must NOT show error
  await page.fill('[name="birthDate"]', ""); // not shown, ignore
  await acceptConsents(page);
  await page.getByRole("button", { name: /einreichen/i }).click();
  await expect(page.getByText(/UID-Nummer ist erforderlich/i)).not.toBeVisible();
});
