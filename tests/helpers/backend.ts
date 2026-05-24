// Backend-Reachability-Helper für Playwright-Specs.
//
// Konsolidiert acht zuvor pro-Datei duplizierte `skipIfBackendDown`-Varianten.
// In CI (`process.env.CI === 'true'`) ist ein nicht erreichbares Backend ein
// hard fail — sonst wären grüne CI-Runs bei totem Backend möglich (Audit
// 2026-05-23, AUDIT-TODO §5i). Lokal bleibt das Verhalten skip+Hinweis,
// damit Frontend-Devs ohne Go-Backend trotzdem Specs ausführen können.
//
// Akzeptiert sowohl `Page` als auch `APIRequestContext`, weil manche Specs
// nur Request-Fixtures haben (kein Browser-Page-Context).

import { test, type APIRequestContext, type Page } from "@playwright/test";

const BACKEND = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function ensureBackendUp(target: Page | APIRequestContext): Promise<void> {
  const request = "request" in target ? target.request : target;
  let reachable = false;
  try {
    const res = await request.get(`${BACKEND}/health`);
    reachable = res.ok();
  } catch {
    reachable = false;
  }
  if (reachable) return;

  const msg = `Backend not reachable at ${BACKEND}/health`;
  if (process.env.CI === "true") {
    throw new Error(`${msg} — refusing to skip in CI (set CI=false to skip locally)`);
  }
  test.skip(true, `${msg} — skipping test`);
}
