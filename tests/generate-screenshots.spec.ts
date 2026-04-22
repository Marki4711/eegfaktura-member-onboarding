import { test, expect } from '@playwright/test'
import path from 'path'

const RC_NUMBER = process.env.RC_NUMBER ?? 'RC123456'
const OUT = path.resolve('docs/user-guide/images')

test.use({ viewport: { width: 1280, height: 800 } })

test('register-form-start', async ({ page }) => {
  await page.goto(`/register/${RC_NUMBER}`)
  await page.waitForLoadState('networkidle')
  await page.screenshot({ path: `${OUT}/register-form-start.png` })
})

test('register-form-metering-points', async ({ page }) => {
  await page.goto(`/register/${RC_NUMBER}`)
  await page.waitForLoadState('networkidle')
  const meteringSection = page.locator('text=Zählpunkt').first()
  await meteringSection.scrollIntoViewIfNeeded()
  await page.screenshot({ path: `${OUT}/register-form-metering-points.png` })
})

test('admin-login-keycloak', async ({ page }) => {
  await page.goto('/admin')
  await page.waitForLoadState('networkidle')
  await page.screenshot({ path: `${OUT}/admin-login-keycloak.png` })
})
