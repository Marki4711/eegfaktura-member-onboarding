/**
 * Screenshot generator for user documentation.
 *
 * Usage:
 *   npx tsx scripts/generate-screenshots.ts
 *
 * Requires:
 *   - App running at http://localhost:3000
 *   - Playwright browsers installed (npx playwright install chromium)
 *   - A valid RC number set via RC_NUMBER env var (default: RC123456)
 *   - Admin session available via ADMIN_COOKIE env var (optional, skips admin
 *     shots if missing). Format: "<cookie-name>=<cookie-value>". The relevant
 *     NextAuth session cookie is usually "__Secure-next-auth.session-token"
 *     (HTTPS) or "next-auth.session-token" (local dev over HTTP).
 *   - APPROVED_APP_ID env var (optional): UUID of an application in the
 *     'approved' status. If set, opens that application directly so the
 *     import dialog (tariff + member-number) screenshot can be captured.
 *   - IMPORTED_APP_ID env var (optional): UUID of an application in the
 *     'imported' status, used for the Reset-Import action screenshot.
 *
 * Outputs screenshots to: docs/user-guide/images/
 */

import { chromium } from '@playwright/test'
import path from 'path'

const BASE_URL = process.env.BASE_URL ?? 'http://localhost:3000'
const RC_NUMBER = process.env.RC_NUMBER ?? 'RC123456'
const APPROVED_APP_ID = process.env.APPROVED_APP_ID
const IMPORTED_APP_ID = process.env.IMPORTED_APP_ID
const OUT_DIR = path.resolve('docs/user-guide/images')

const VIEWPORT = { width: 1280, height: 800 }

async function main() {
  const browser = await chromium.launch({ headless: true })
  const ctx = await browser.newContext({ viewport: VIEWPORT })
  const page = await ctx.newPage()

  // ── Registration form ──────────────────────────────────────────────────────

  await page.goto(`${BASE_URL}/register/${RC_NUMBER}`)
  await page.waitForLoadState('networkidle')
  await page.screenshot({ path: `${OUT_DIR}/register-form-start.png`, fullPage: false })
  console.log('✓ register-form-start.png')

  // Scroll to metering point section
  const meteringHeading = page.locator('text=Zählpunkt').first()
  if (await meteringHeading.count() > 0) {
    await meteringHeading.scrollIntoViewIfNeeded()
    await page.screenshot({ path: `${OUT_DIR}/register-form-metering-points.png`, fullPage: false })
    console.log('✓ register-form-metering-points.png')
  }

  // ── Admin section ──────────────────────────────────────────────────────────
  // Capture the Keycloak login page first (unauthenticated)
  await page.goto(`${BASE_URL}/admin`)
  await page.waitForLoadState('networkidle')
  await page.screenshot({ path: `${OUT_DIR}/admin-login-keycloak.png`, fullPage: false })
  console.log('✓ admin-login-keycloak.png')

  const adminCookie = process.env.ADMIN_COOKIE
  if (!adminCookie) {
    console.warn('⚠ ADMIN_COOKIE not set — skipping authenticated admin screenshots')
    console.warn('  Set ADMIN_COOKIE="<name>=<value>" and re-run to generate admin screenshots')
    await browser.close()
    return
  }

  const [cookieName, ...cookieRest] = adminCookie.split('=')
  await ctx.addCookies([
    {
      name: cookieName.trim(),
      value: cookieRest.join('=').trim(),
      domain: new URL(BASE_URL).hostname,
      path: '/',
    },
  ])

  // Applications list — shows the new Mitgliedsnummer column + sort affordance
  await page.goto(`${BASE_URL}/admin/applications`)
  await page.waitForLoadState('networkidle')
  await page.screenshot({ path: `${OUT_DIR}/admin-applications-list.png`, fullPage: false })
  console.log('✓ admin-applications-list.png')

  // Filter panel
  const filterToggle = page.locator('[data-testid="filter-toggle"], button:has-text("Filter")')
  if (await filterToggle.count() > 0) {
    await filterToggle.first().click()
    await page.waitForTimeout(400)
  }
  await page.screenshot({ path: `${OUT_DIR}/admin-filter-panel.png`, fullPage: false })
  console.log('✓ admin-filter-panel.png')

  // Open first application row, capture detail (split into top + bottom halves
  // to match the existing -1 / -2 layout in the user guide)
  const firstRow = page.locator('table tbody tr').first()
  if (await firstRow.count() > 0) {
    await firstRow.click()
    await page.waitForLoadState('networkidle')

    // Detail top half — Stammdaten + Statusaktionen
    await page.evaluate(() => window.scrollTo(0, 0))
    await page.waitForTimeout(200)
    await page.screenshot({ path: `${OUT_DIR}/admin-application-detail-1.png`, fullPage: false })
    console.log('✓ admin-application-detail-1.png')

    // Detail bottom half — Zählpunkte + Admin-Notiz + Statusverlauf
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await page.waitForTimeout(200)
    await page.screenshot({ path: `${OUT_DIR}/admin-application-detail-2.png`, fullPage: false })
    console.log('✓ admin-application-detail-2.png')

    // Status actions area
    const statusArea = page.locator('[data-testid="status-actions"], :text("Status-Aktionen")')
    if (await statusArea.count() > 0) {
      await statusArea.first().scrollIntoViewIfNeeded()
      await page.screenshot({ path: `${OUT_DIR}/admin-status-actions.png`, fullPage: false })
      console.log('✓ admin-status-actions.png')
    }

    // Status log area
    const logArea = page.locator('[data-testid="status-log"], :text("Statusverlauf")')
    if (await logArea.count() > 0) {
      await logArea.first().scrollIntoViewIfNeeded()
      await page.screenshot({ path: `${OUT_DIR}/admin-status-log.png`, fullPage: false })
      console.log('✓ admin-status-log.png')
    }

    // Logout button
    const logoutBtn = page.locator('button:has-text("Abmelden"), a:has-text("Abmelden")')
    if (await logoutBtn.count() > 0) {
      await logoutBtn.first().scrollIntoViewIfNeeded()
      await page.screenshot({ path: `${OUT_DIR}/admin-logout.png`, fullPage: false })
      console.log('✓ admin-logout.png')
    }
  } else {
    console.warn('⚠ No application rows found — detail screenshots skipped')
  }

  // Import dialog — must be captured on an APPROVED application
  if (APPROVED_APP_ID) {
    await page.goto(`${BASE_URL}/admin/applications/${APPROVED_APP_ID}`)
    await page.waitForLoadState('networkidle')

    const importBtn = page.locator('button:has-text("eegFaktura importieren"), button:has-text("In eegFaktura importieren")')
    if (await importBtn.count() > 0) {
      await importBtn.first().click()
      // Dialog opens with tariff fetch + next-member-number prefill — wait for both
      await page.waitForSelector('text=Mitgliedsnummer', { timeout: 10_000 })
      await page.waitForTimeout(600)
      await page.screenshot({ path: `${OUT_DIR}/admin-import-action.png`, fullPage: false })
      console.log('✓ admin-import-action.png (tariff + member-number dialog)')
    } else {
      console.warn('⚠ APPROVED_APP_ID provided but no import button found — is the application still in approved status?')
    }
  } else {
    console.warn('⚠ APPROVED_APP_ID not set — skipping admin-import-action.png')
    console.warn('  Set APPROVED_APP_ID=<uuid> to capture the import dialog')
  }

  // Reset-Import dialog — must be captured on an IMPORTED application
  if (IMPORTED_APP_ID) {
    await page.goto(`${BASE_URL}/admin/applications/${IMPORTED_APP_ID}`)
    await page.waitForLoadState('networkidle')

    const resetBtn = page.locator('button:has-text("Import zurücksetzen")')
    if (await resetBtn.count() > 0) {
      await resetBtn.scrollIntoViewIfNeeded()
      await page.screenshot({ path: `${OUT_DIR}/admin-reset-import.png`, fullPage: false })
      console.log('✓ admin-reset-import.png')
    } else {
      console.warn('⚠ IMPORTED_APP_ID provided but no reset button found — is the application still in imported status?')
    }
  }

  await browser.close()
  console.log(`\nScreenshots written to ${OUT_DIR}`)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
