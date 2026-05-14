/**
 * Screenshot generator for user documentation.
 *
 * Usage:
 *   npm run screenshots                # interactive on first run, headless on re-runs
 *   npm run screenshots -- --re-login  # force a fresh Keycloak login
 *
 * What it does:
 *   1. If a valid cookie cache exists, reuses it. Otherwise opens a visible
 *      Chromium pointed at /admin so you can log in via Keycloak. Cookies are
 *      captured and cached at .cache/screenshots-cookies.json (gitignored).
 *   2. Auto-discovers an `approved` and an `imported` application via the
 *      admin list — no manual UUIDs needed.
 *   3. Runs all screenshots in headless mode and writes them to
 *      docs/user-guide/images/.
 *
 * Env overrides (all optional):
 *   BASE_URL      Default http://localhost:3000
 *   RC_NUMBER     Default RC123456 (must exist in registration_entrypoint)
 *
 * Prerequisites:
 *   - Backend + Frontend running locally
 *   - `npx playwright install chromium` once
 */

import { chromium, type BrowserContext, type Browser, type Cookie, type Page } from "@playwright/test"
import fs from "fs"
import path from "path"

const BASE_URL = process.env.BASE_URL ?? "http://localhost:3000"
const RC_NUMBER = process.env.RC_NUMBER ?? "RC123456"
const OUT_DIR = path.resolve("docs/user-guide/images")
const COOKIE_CACHE = path.resolve(".cache/screenshots-cookies.json")
const FORCE_RELOGIN = process.argv.includes("--re-login")

const VIEWPORT = { width: 1280, height: 800 }
const LOGIN_TIMEOUT_MS = 5 * 60 * 1000 // 5 minutes for the user to log in

function log(msg: string) {
  console.log(msg)
}

async function ensureLoggedInContext(browser: Browser): Promise<BrowserContext> {
  // Try cached cookies first
  if (!FORCE_RELOGIN && fs.existsSync(COOKIE_CACHE)) {
    try {
      const cached = JSON.parse(fs.readFileSync(COOKIE_CACHE, "utf8")) as Cookie[]
      const ctx = await browser.newContext({ viewport: VIEWPORT })
      await ctx.addCookies(cached)
      const probe = await ctx.newPage()
      await probe.goto(`${BASE_URL}/admin/applications`, { waitUntil: "domcontentloaded" })
      // If we landed on a Keycloak page, the cached session is stale.
      const onAdmin = probe.url().startsWith(`${BASE_URL}/admin`)
      await probe.close()
      if (onAdmin) {
        log("✓ Re-using cached admin session")
        return ctx
      }
      log("⚠ Cached cookies are stale — re-running login")
      await ctx.close()
    } catch (err) {
      log(`⚠ Could not load cookie cache (${(err as Error).message}); re-running login`)
    }
  }

  // Interactive login
  log("🔐 Opening browser for Keycloak login — please sign in.")
  const loginBrowser = await chromium.launch({ headless: false })
  const loginCtx = await loginBrowser.newContext({ viewport: VIEWPORT })
  const page = await loginCtx.newPage()
  await page.goto(`${BASE_URL}/admin`)

  // Wait until we're back on our own /admin/* domain (and not on a NextAuth signin page)
  const ownOrigin = new URL(BASE_URL).origin
  try {
    await page.waitForURL(
      (url) => url.origin === ownOrigin && url.pathname.startsWith("/admin") && !url.pathname.includes("signin"),
      { timeout: LOGIN_TIMEOUT_MS }
    )
  } catch {
    await loginBrowser.close()
    throw new Error(`Login did not complete within ${LOGIN_TIMEOUT_MS / 1000}s`)
  }
  await page.waitForLoadState("networkidle")

  const cookies = await loginCtx.cookies()
  fs.mkdirSync(path.dirname(COOKIE_CACHE), { recursive: true })
  fs.writeFileSync(COOKIE_CACHE, JSON.stringify(cookies, null, 2))
  log(`✓ Cookies cached to ${path.relative(process.cwd(), COOKIE_CACHE)}`)
  await loginBrowser.close()

  // Return a fresh headless context seeded with the captured cookies
  const headlessCtx = await browser.newContext({ viewport: VIEWPORT })
  await headlessCtx.addCookies(cookies)
  return headlessCtx
}

async function findFirstAppIdByStatus(page: Page, status: string): Promise<string | null> {
  await page.goto(`${BASE_URL}/admin/applications?status=${encodeURIComponent(status)}&page_size=1`)
  await page.waitForLoadState("networkidle")

  // First strategy: scrape the first detail link in the rendered table
  const allLinks = await page.locator('a[href*="/admin/applications/"]').all()
  for (const link of allLinks) {
    const href = await link.getAttribute("href")
    if (!href) continue
    const m = href.match(/\/admin\/applications\/([0-9a-f-]{36})/i)
    if (m) return m[1]
  }
  return null
}

async function shoot(page: Page, name: string, opts: { fullPage?: boolean } = {}) {
  await page.screenshot({ path: `${OUT_DIR}/${name}`, fullPage: opts.fullPage ?? false })
  log(`  ✓ ${name}`)
}

async function main() {
  const browser = await chromium.launch({ headless: true })

  // ── Public screens (no auth needed) ──────────────────────────────────────
  log("\nPublic screens")
  {
    const ctx = await browser.newContext({ viewport: VIEWPORT })
    const page = await ctx.newPage()

    await page.goto(`${BASE_URL}/register/${RC_NUMBER}`)
    await page.waitForLoadState("networkidle")
    await shoot(page, "register-form-start.png")

    const meteringHeading = page.locator("text=Zählpunkt").first()
    if ((await meteringHeading.count()) > 0) {
      await meteringHeading.scrollIntoViewIfNeeded()
      await shoot(page, "register-form-metering-points.png")
    }

    // Keycloak login page — visit /admin in a fresh, unauthenticated context
    await page.goto(`${BASE_URL}/admin`)
    await page.waitForLoadState("networkidle")
    await shoot(page, "admin-login-keycloak.png")

    await ctx.close()
  }

  // ── Authenticated admin screens ──────────────────────────────────────────
  log("\nAdmin screens (requires Keycloak login)")
  const adminCtx = await ensureLoggedInContext(browser)
  const page = await adminCtx.newPage()

  // Applications list — shows Mitgliedsnummer column + sort affordance
  await page.goto(`${BASE_URL}/admin/applications`)
  await page.waitForLoadState("networkidle")
  await shoot(page, "admin-applications-list.png")

  // Filter panel
  const filterToggle = page.locator('[data-testid="filter-toggle"], button:has-text("Filter")')
  if ((await filterToggle.count()) > 0) {
    await filterToggle.first().click()
    await page.waitForTimeout(400)
    await shoot(page, "admin-filter-panel.png")
  }

  // Detail view — top + bottom halves
  const firstRow = page.locator("table tbody tr").first()
  if ((await firstRow.count()) > 0) {
    await firstRow.click()
    await page.waitForLoadState("networkidle")

    await page.evaluate(() => window.scrollTo(0, 0))
    await page.waitForTimeout(200)
    await shoot(page, "admin-application-detail-1.png")

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await page.waitForTimeout(200)
    await shoot(page, "admin-application-detail-2.png")

    const statusArea = page.locator('[data-testid="status-actions"], :text("Status-Aktionen")').first()
    if ((await statusArea.count()) > 0) {
      await statusArea.scrollIntoViewIfNeeded()
      await shoot(page, "admin-status-actions.png")
    }

    const logArea = page.locator('[data-testid="status-log"], :text("Statusverlauf")').first()
    if ((await logArea.count()) > 0) {
      await logArea.scrollIntoViewIfNeeded()
      await shoot(page, "admin-status-log.png")
    }

    const logoutBtn = page.locator('button:has-text("Abmelden"), a:has-text("Abmelden")').first()
    if ((await logoutBtn.count()) > 0) {
      await logoutBtn.scrollIntoViewIfNeeded()
      await shoot(page, "admin-logout.png")
    }
  } else {
    log("  ⚠ No application rows found — detail screenshots skipped (seed missing?)")
  }

  // Import dialog — auto-discover an approved application
  const approvedId = await findFirstAppIdByStatus(page, "approved")
  if (approvedId) {
    await page.goto(`${BASE_URL}/admin/applications/${approvedId}`)
    await page.waitForLoadState("networkidle")
    const importBtn = page
      .locator('button:has-text("eegFaktura importieren"), button:has-text("In eegFaktura importieren")')
      .first()
    if ((await importBtn.count()) > 0) {
      await importBtn.click()
      // Wait for tariff fetch + next-member-number prefill
      await page.waitForSelector("text=Mitgliedsnummer", { timeout: 10_000 })
      await page.waitForTimeout(600)
      await shoot(page, "admin-import-action.png")
    } else {
      log("  ⚠ Found approved app but no import button — UI changed?")
    }
  } else {
    log("  ⚠ No application in status `approved` — admin-import-action.png skipped")
  }

  // Reset-Import — auto-discover an imported application
  const importedId = await findFirstAppIdByStatus(page, "imported")
  if (importedId) {
    await page.goto(`${BASE_URL}/admin/applications/${importedId}`)
    await page.waitForLoadState("networkidle")
    const resetBtn = page.locator('button:has-text("Import zurücksetzen")').first()
    if ((await resetBtn.count()) > 0) {
      await resetBtn.scrollIntoViewIfNeeded()
      await shoot(page, "admin-reset-import.png")
    } else {
      log("  ⚠ Found imported app but no reset button — UI changed?")
    }
  } else {
    log("  ⚠ No application in status `imported` — admin-reset-import.png skipped")
  }

  await browser.close()
  log(`\nScreenshots written to ${path.relative(process.cwd(), OUT_DIR)}`)
}

main().catch((err) => {
  console.error("\n❌ Screenshot run failed:", err)
  process.exit(1)
})
