import { test, expect } from "@playwright/test";

// PROJ-5: Keycloak-secured Admin Area
//
// NOTE: Full authentication flow tests require a live Keycloak instance.
// These tests cover what can be verified without Keycloak:
//   - The unauthorized page renders correctly
//   - Unauthenticated requests to /admin are redirected (to /api/auth/signin)
//
// The Keycloak authentication flow (login, token exchange, tenant-admin sync,
// role-based scoping) must be verified manually against a real Keycloak server.

test("AC-unauthorized: /unauthorized page renders error message and action buttons", async ({ page }) => {
  await page.goto("/unauthorized");
  await expect(page.getByRole("heading", { name: /Kein Zugriff/i })).toBeVisible();
  await expect(page.getByText(/keine Berechtigung/i)).toBeVisible();
  await expect(page.getByRole("link", { name: /Anderes Konto/i })).toBeVisible();
  await expect(page.getByRole("link", { name: /Zurück zur Startseite/i })).toBeVisible();
});

test("AC-auth-redirect: unauthenticated GET /admin redirects away from admin area", async ({ page }) => {
  const response = await page.goto("/admin");
  // Must NOT land on the admin area without a session — should redirect to signin
  await expect(page).not.toHaveURL(/^http:\/\/localhost:\d+\/admin(\?|$)/);
});

test("AC-auth-redirect: unauthenticated GET /admin/applications redirects away from admin area", async ({ page }) => {
  await page.goto("/admin/applications");
  await expect(page).not.toHaveURL(/^http:\/\/localhost:\d+\/admin\/applications(\?|$)/);
});

test("AC-unauthorized-responsive: unauthorized page is readable on mobile (375px)", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto("/unauthorized");
  await expect(page.getByRole("heading", { name: /Kein Zugriff/i })).toBeVisible();
  await expect(page.getByText(/keine Berechtigung/i)).toBeVisible();
});
