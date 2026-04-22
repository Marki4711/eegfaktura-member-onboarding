import { test, BrowserContext } from '@playwright/test'
import path from 'path'

// Override BASE_URL to target a remote environment, e.g.:
//   BASE_URL=https://member-onboarding-test.eegfaktura.at RC_NUMBER=RC123456 \
//   ADMIN_COOKIE_0=... ADMIN_COOKIE_1=... npm run test:e2e -- tests/generate-screenshots.spec.ts
const BASE_URL = process.env.BASE_URL ?? 'http://localhost:3000'
const RC_NUMBER = process.env.RC_NUMBER ?? 'RC123456'
const OUT = path.resolve('docs/user-guide/images')

const domain = new URL(BASE_URL).hostname

// Support split cookies (__Secure-next-auth.session-token.0 / .1)
async function injectAdminSession(ctx: BrowserContext) {
  const base = { domain, path: '/', httpOnly: true, secure: BASE_URL.startsWith('https'), sameSite: 'Lax' as const }
  if (process.env.ADMIN_COOKIE_0) {
    await ctx.addCookies([{ ...base, name: '__Secure-next-auth.session-token.0', value: process.env.ADMIN_COOKIE_0 }])
  }
  if (process.env.ADMIN_COOKIE_1) {
    await ctx.addCookies([{ ...base, name: '__Secure-next-auth.session-token.1', value: process.env.ADMIN_COOKIE_1 }])
  }
}

function url(path: string) {
  return `${BASE_URL}${path}`
}

test.use({ viewport: { width: 1280, height: 800 } })

test('register-form-start', async ({ page }) => {
  await page.goto(url(`/register/${RC_NUMBER}`))
  await page.waitForLoadState('networkidle')
  await page.screenshot({ path: `${OUT}/register-form-start.png` })
})

test('register-form-metering-points', async ({ page }) => {
  await page.goto(url(`/register/${RC_NUMBER}`))
  await page.waitForLoadState('networkidle')
  await page.locator('text=Zählpunkt').first().scrollIntoViewIfNeeded()
  await page.screenshot({ path: `${OUT}/register-form-metering-points.png` })
})

test('admin-login-keycloak', async ({ page }) => {
  await page.goto(url('/admin'))
  await page.waitForLoadState('networkidle')
  await page.screenshot({ path: `${OUT}/admin-login-keycloak.png` })
})

test('admin-applications-list', async ({ page, context }) => {
  await injectAdminSession(context)
  await page.goto(url('/admin/applications'))
  await page.waitForLoadState('networkidle')
  await page.screenshot({ path: `${OUT}/admin-applications-list.png` })
})

test('admin-filter-panel', async ({ page, context }) => {
  await injectAdminSession(context)
  await page.goto(url('/admin/applications'))
  await page.waitForLoadState('networkidle')
  const toggle = page.locator('button:has-text("Filter"), [data-testid="filter-toggle"]').first()
  if (await toggle.isVisible()) await toggle.click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: `${OUT}/admin-filter-panel.png` })
})

test('admin-application-detail', async ({ page, context }) => {
  await injectAdminSession(context)
  await page.goto(url('/admin/applications'))
  await page.waitForLoadState('networkidle')
  const firstRow = page.locator('table tbody tr').first()
  if (await firstRow.isVisible()) {
    await firstRow.click()
    await page.waitForLoadState('networkidle')
    await page.screenshot({ path: `${OUT}/admin-application-detail.png` })
  }
})

test('admin-status-actions', async ({ page, context }) => {
  await injectAdminSession(context)
  await page.goto(url('/admin/applications'))
  await page.waitForLoadState('networkidle')
  const firstRow = page.locator('table tbody tr').first()
  if (await firstRow.isVisible()) {
    await firstRow.click()
    await page.waitForLoadState('networkidle')
    const statusArea = page.locator(':text("Status")').first()
    await statusArea.scrollIntoViewIfNeeded()
    await page.screenshot({ path: `${OUT}/admin-status-actions.png` })
  }
})

test('admin-status-log', async ({ page, context }) => {
  await injectAdminSession(context)
  await page.goto(url('/admin/applications'))
  await page.waitForLoadState('networkidle')
  const firstRow = page.locator('table tbody tr').first()
  if (await firstRow.isVisible()) {
    await firstRow.click()
    await page.waitForLoadState('networkidle')
    const logArea = page.locator(':text("Statusverlauf")').first()
    await logArea.scrollIntoViewIfNeeded()
    await page.screenshot({ path: `${OUT}/admin-status-log.png` })
  }
})

test('admin-logout', async ({ page, context }) => {
  await injectAdminSession(context)
  await page.goto(url('/admin/applications'))
  await page.waitForLoadState('networkidle')
  const logoutBtn = page.locator('button:has-text("Abmelden"), a:has-text("Abmelden")').first()
  if (await logoutBtn.isVisible()) {
    await logoutBtn.scrollIntoViewIfNeeded()
    await page.screenshot({ path: `${OUT}/admin-logout.png` })
  }
})
