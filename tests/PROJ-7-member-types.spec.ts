import { test, expect } from "@playwright/test";

// Uses a real RC number seeded in dev DB. Override with env var if needed.
const RC = process.env.TEST_RC_NUMBER ?? "RC123456";
const FORM_URL = `/register/${RC}`;

// Helper: open the member type Select dropdown and choose an option by label.
async function selectMemberType(page: import("@playwright/test").Page, label: string | RegExp) {
  await page.getByRole("combobox", { name: /Mitgliedstyp/i }).click();
  await page.getByRole("option", { name: label }).click();
}

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
  // Metering point (masked input, name-based selector)
  const mpInput = page.locator('[name="meteringPoints.0.meteringPoint"]');
  await mpInput.click();
  await mpInput.fill("AT 003100 00000 000000000000 00000001");
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

test("AC-1: member type selector shows five options", async ({ page }) => {
  await page.goto(FORM_URL);
  await page.getByRole("combobox", { name: /Mitgliedstyp/i }).click();
  const options = page.getByRole("option");
  await expect(options).toHaveCount(5);
});

test("AC-2: default selection is 'Privatperson / Kleinunternehmer'", async ({ page }) => {
  await page.goto(FORM_URL);
  await expect(page.getByRole("combobox", { name: /Mitgliedstyp/i })).toContainText("Privatperson / Kleinunternehmer");
});

test("AC-3: member type options show VAT hint", async ({ page }) => {
  await page.goto(FORM_URL);
  await page.getByRole("combobox", { name: /Mitgliedstyp/i }).click();
  // Options are shown in the open dropdown — check for partial text in the listbox.
  // Privatperson hat keinen USt-Hint mehr (für Endkunden missverständlich),
  // Kleinunternehmer behält „(0 % USt.)" als rechtlich relevante Information.
  const listbox = page.getByRole("listbox");
  await expect(listbox.getByText("0 % USt.", { exact: false }).first()).toBeVisible();
  await expect(listbox.getByText("13 % USt.", { exact: false }).first()).toBeVisible();
  await expect(listbox.getByText("20 % USt.", { exact: false }).first()).toBeVisible();
});

test("AC-4: selecting a type updates the form fields immediately", async ({ page }) => {
  await page.goto(FORM_URL);
  // Initially person fields visible
  await expect(page.locator('[name="firstname"]')).toBeVisible();
  // Switch to company
  await selectMemberType(page, /Unternehmen/i);
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
  await selectMemberType(page, /Pauschalierter Landwirt/i);
  await expect(page.locator('[name="firstname"]')).toBeVisible();
  await expect(page.locator('[name="lastname"]')).toBeVisible();
  await expect(page.locator('[name="birthDate"]')).toBeVisible();
  await expect(page.locator('[name="companyName"]')).not.toBeVisible();
});

test("AC-7: municipality type shows companyName (required) and uidNumber (optional), no person fields", async ({ page }) => {
  await page.goto(FORM_URL);
  await selectMemberType(page, /Gemeinde/i);
  await expect(page.locator('[name="companyName"]')).toBeVisible();
  await expect(page.locator('[name="uidNumber"]')).toBeVisible();
  await expect(page.locator('[name="registerNumber"]')).not.toBeVisible();
  await expect(page.locator('[name="firstname"]')).not.toBeVisible();
  await expect(page.locator('[name="lastname"]')).not.toBeVisible();
  await expect(page.locator('[name="birthDate"]')).not.toBeVisible();
});

test("AC-8: company type shows companyName, uidNumber, registerNumber as required", async ({ page }) => {
  await page.goto(FORM_URL);
  await selectMemberType(page, /Unternehmen/i);
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
  await selectMemberType(page, /Unternehmen/i);
  // Switch back to private
  await selectMemberType(page, /Privatperson/i);
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
  await selectMemberType(page, /Unternehmen/i);
  await fillCommonFields(page);
  await page.fill('[name="companyName"]', "Muster GmbH");
  // uid and registerNumber intentionally left empty
  await acceptConsents(page);
  await page.getByRole("button", { name: /einreichen/i }).click();
  await expect(page.getByText(/UID-Nummer ist erforderlich/i)).toBeVisible();
  await expect(page.getByText(/Firmenbuchnummer ist erforderlich/i)).toBeVisible();
});

test("AC-12: municipality type does not require uidNumber", async ({ page }) => {
  await page.goto(FORM_URL);
  await selectMemberType(page, /Gemeinde/i);
  await fillCommonFields(page);
  await page.fill('[name="companyName"]', "Gemeinde Musterort");
  // uid intentionally left empty — must NOT show error
  await acceptConsents(page);
  await page.getByRole("button", { name: /einreichen/i }).click();
  await expect(page.getByText(/UID-Nummer ist erforderlich/i)).not.toBeVisible();
});
