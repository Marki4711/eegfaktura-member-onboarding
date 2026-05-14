/**
 * Screenshot generator for user documentation.
 *
 * Usage:
 *   npm run screenshots                   # interactive on first run, headless on re-runs
 *   npm run screenshots -- --re-login     # force a fresh Keycloak login
 *   npm run screenshots -- --public-only  # only the screens that need no admin login
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

// Pull screenshot-stack overrides from .env.screenshots.local (if present)
// without dragging in a real dotenv dependency. Setup-script writes that file.
const SCREENSHOT_ENV = path.resolve(".env.screenshots.local")
if (fs.existsSync(SCREENSHOT_ENV)) {
  for (const line of fs.readFileSync(SCREENSHOT_ENV, "utf8").split(/\r?\n/)) {
    const m = line.match(/^\s*([A-Z0-9_]+)\s*=\s*(.*)\s*$/)
    if (m && !line.startsWith("#")) {
      process.env[m[1]] ??= m[2]
    }
  }
}

const BASE_URL = process.env.BASE_URL ?? "http://localhost:3000"
const RC_NUMBER = process.env.RC_NUMBER ?? "RC123456"
const OUT_DIR = path.resolve("docs/user-guide/images")
const COOKIE_CACHE = path.resolve(".cache/screenshots-cookies.json")
const FORCE_RELOGIN = process.argv.includes("--re-login")
const PUBLIC_ONLY = process.argv.includes("--public-only")

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

  // Auto-login path: SCREENSHOT_BOT_USERNAME + _PASSWORD set (typically by
  // .env.screenshots.local) → run headless, fill the Keycloak form ourselves.
  const botUser = process.env.SCREENSHOT_BOT_USERNAME
  const botPass = process.env.SCREENSHOT_BOT_PASSWORD
  const headless = Boolean(botUser && botPass)

  if (headless) {
    log("🔐 Auto-login as bot user (headless)")
  } else {
    log("🔐 Opening browser for Keycloak login — please sign in.")
  }
  const loginBrowser = await chromium.launch({ headless })
  const loginCtx = await loginBrowser.newContext({ viewport: VIEWPORT })
  const page = await loginCtx.newPage()
  await page.goto(`${BASE_URL}/admin`)

  if (headless && botUser && botPass) {
    // Wait for Keycloak's login form (NextAuth's intermediate /signin page
    // immediately submits to Keycloak's authorize endpoint).
    try {
      await page.waitForSelector('input[name="username"], input#username', { timeout: 30_000 })
    } catch {
      // Some NextAuth setups render a provider-picker first — click "Sign in with Keycloak"
      const provider = page.locator('button:has-text("Keycloak"), a:has-text("Keycloak")').first()
      if ((await provider.count()) > 0) {
        await provider.click()
        await page.waitForSelector('input[name="username"], input#username', { timeout: 30_000 })
      } else {
        throw new Error("Could not locate the Keycloak username field")
      }
    }
    await page.fill('input[name="username"], input#username', botUser)
    await page.fill('input[name="password"], input#password', botPass)
    await Promise.all([
      page.waitForLoadState("networkidle"),
      page.click('input[type="submit"], button[type="submit"], #kc-login'),
    ])
  }

  // Wait until we're back on our own /admin/* domain (and not on a NextAuth signin page)
  const ownOrigin = new URL(BASE_URL).origin
  try {
    await page.waitForURL(
      (url) => url.origin === ownOrigin && url.pathname.startsWith("/admin") && !url.pathname.includes("signin"),
      { timeout: headless ? 30_000 : LOGIN_TIMEOUT_MS }
    )
  } catch {
    await loginBrowser.close()
    throw new Error(`Login did not complete within the timeout`)
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

  // Rows use a router.push click handler — no href to scrape. Click the first
  // row and read the UUID out of the resulting URL, then navigate back so the
  // caller can proceed without losing the listing context.
  const firstRow = page.locator("table tbody tr").first()
  if ((await firstRow.count()) === 0) return null
  await firstRow.click()
  try {
    await page.waitForURL(/\/admin\/applications\/[0-9a-f-]{36}/i, { timeout: 10_000 })
  } catch {
    return null
  }
  const m = page.url().match(/\/admin\/applications\/([0-9a-f-]{36})/i)
  return m ? m[1] : null
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

    // Keycloak login page — visit /admin in a fresh, unauthenticated context.
    // NextAuth lands on the provider picker first; click "Sign in with Keycloak"
    // so the screenshot shows the actual Keycloak username/password form, not
    // the NextAuth intermediate page.
    await page.goto(`${BASE_URL}/admin`)
    await page.waitForLoadState("networkidle")
    const kcProvider = page
      .locator('button:has-text("Keycloak"), a:has-text("Keycloak")')
      .first()
    if ((await kcProvider.count()) > 0) {
      await kcProvider.click()
      try {
        await page.waitForSelector('input[name="username"], input#username', { timeout: 15_000 })
        await page.waitForTimeout(300)
      } catch {
        // fall through and capture whatever ended up on screen
      }
    }
    await shoot(page, "admin-login-keycloak.png")

    await ctx.close()
  }

  if (PUBLIC_ONLY) {
    log("\n--public-only set — skipping admin screens")
    await browser.close()
    log(`\nScreenshots written to ${path.relative(process.cwd(), OUT_DIR)}`)
    return
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

    // Section-specific shots: capture a clipped region from the heading
    // downward instead of trying to scroll the section to the top of the
    // viewport — that doesn't work for sections that already sit near the
    // bottom of the page (scrollIntoView clamps and the viewport stays put).
    const shootSection = async (text: string, name: string, height = 320) => {
      const elem = page.getByText(text, { exact: false }).first()
      if ((await elem.count()) === 0) {
        log(`  ⚠ ${name} — could not locate "${text}" heading`)
        return
      }
      await elem.evaluate((el) => el.scrollIntoView({ block: "center", behavior: "instant" }))
      await page.waitForTimeout(150)
      const box = await elem.boundingBox()
      if (!box) {
        log(`  ⚠ ${name} — no bounding box for "${text}"`)
        return
      }
      const pageHeight = await page.evaluate(() => document.documentElement.clientHeight)
      const clipY = Math.max(0, box.y - 16)
      const clipHeight = Math.min(height, pageHeight - clipY)
      await page.screenshot({
        path: `${OUT_DIR}/${name}`,
        clip: { x: 0, y: clipY, width: VIEWPORT.width, height: clipHeight },
      })
      log(`  ✓ ${name}`)
    }
    await shootSection("Statusaktionen", "admin-status-actions.png", 180)
    await shootSection("Statusverlauf", "admin-status-log.png", 220)

    // Logout button lives in the header; scroll to top, then clip to the
    // header strip so we see "Abmelden" without the rest of the detail page.
    await page.evaluate(() => window.scrollTo(0, 0))
    await page.waitForTimeout(150)
    await page.screenshot({
      path: `${OUT_DIR}/admin-logout.png`,
      clip: { x: 0, y: 0, width: VIEWPORT.width, height: 80 },
    })
    log("  ✓ admin-logout.png")
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
