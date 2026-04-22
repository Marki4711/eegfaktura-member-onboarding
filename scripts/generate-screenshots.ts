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
 *   - Admin session available via ADMIN_COOKIE env var (optional, skips admin shots if missing)
 *
 * Outputs screenshots to: docs/user-guide/images/
 */

import { chromium } from '@playwright/test'
import path from 'path'

const BASE_URL = process.env.BASE_URL ?? 'http://localhost:3000'
const RC_NUMBER = process.env.RC_NUMBER ?? 'RC123456'
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
  await page.locator('text=Zählpunkt').first().scrollIntoViewIfNeeded()
  await page.screenshot({ path: `${OUT_DIR}/register-form-metering-points.png`, fullPage: false })
  console.log('✓ register-form-metering-points.png')

  // ── Admin section ──────────────────────────────────────────────────────────
  // Navigate to /admin and capture the Keycloak redirect (login page)
  await page.goto(`${BASE_URL}/admin`)
  await page.waitForLoadState('networkidle')
  await page.screenshot({ path: `${OUT_DIR}/admin-login-keycloak.png`, fullPage: false })
  console.log('✓ admin-login-keycloak.png')

  // If a session cookie is provided, capture admin views
  const adminCookie = process.env.ADMIN_COOKIE
  if (adminCookie) {
    // Inject session cookie so we bypass Keycloak
    const [name, ...rest] = adminCookie.split('=')
    await ctx.addCookies([
      {
        name: name.trim(),
        value: rest.join('=').trim(),
        domain: new URL(BASE_URL).hostname,
        path: '/',
      },
    ])

    await page.goto(`${BASE_URL}/admin/applications`)
    await page.waitForLoadState('networkidle')

    // Full applications list
    await page.screenshot({ path: `${OUT_DIR}/admin-applications-list.png`, fullPage: false })
    console.log('✓ admin-applications-list.png')

    // Filter panel — open it if it has a toggle button
    const filterToggle = page.locator('[data-testid="filter-toggle"], button:has-text("Filter")')
    if (await filterToggle.count() > 0) {
      await filterToggle.first().click()
      await page.waitForTimeout(400)
    }
    await page.screenshot({ path: `${OUT_DIR}/admin-filter-panel.png`, fullPage: false })
    console.log('✓ admin-filter-panel.png')

    // Open first application row
    const firstRow = page.locator('table tbody tr').first()
    if (await firstRow.count() > 0) {
      await firstRow.click()
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: `${OUT_DIR}/admin-application-detail.png`, fullPage: false })
      console.log('✓ admin-application-detail.png')

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

      // Import action (visible only on approved applications)
      const importBtn = page.locator('button:has-text("eegFaktura importieren")')
      if (await importBtn.count() > 0) {
        await importBtn.scrollIntoViewIfNeeded()
        await page.screenshot({ path: `${OUT_DIR}/admin-import-action.png`, fullPage: false })
        console.log('✓ admin-import-action.png')
      }

      // Logout button
      const logoutBtn = page.locator('button:has-text("Abmelden"), a:has-text("Abmelden")')
      if (await logoutBtn.count() > 0) {
        await logoutBtn.first().scrollIntoViewIfNeeded()
        await page.screenshot({ path: `${OUT_DIR}/admin-logout.png`, fullPage: false })
        console.log('✓ admin-logout.png')
      }
    } else {
      console.warn('⚠ No application rows found — admin screenshots incomplete')
    }
  } else {
    console.warn('⚠ ADMIN_COOKIE not set — skipping authenticated admin screenshots')
    console.warn('  Set ADMIN_COOKIE="<name>=<value>" and re-run to generate admin screenshots')
  }

  await browser.close()
  console.log(`\nScreenshots written to ${OUT_DIR}`)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
