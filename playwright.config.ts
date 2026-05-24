import { defineConfig, devices } from '@playwright/test'

// Browser-Matrix:
// - Lokal (default): Chromium + Firefox + WebKit + Mobile-Safari — deckt die
//   in AT/DE relevanten Engines ab; Mobile-Chromium ist weggelassen, weil es
//   dieselbe Engine wie Desktop-Chrome verwendet.
// - PR-CI (PLAYWRIGHT_BROWSERS=chromium): nur Chromium, um die PR-CI-Laufzeit
//   unter ~10 Minuten zu halten. Multi-Browser-Regressionen werden in einem
//   separaten nightly/wöchentlichen Workflow gefangen (eigenes Sub-Ticket).
const browsersEnv = process.env.PLAYWRIGHT_BROWSERS
const allProjects = [
  { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
  { name: 'webkit', use: { ...devices['Desktop Safari'] } },
  { name: 'Mobile Safari', use: { ...devices['iPhone 13'] } },
]
const projects = browsersEnv
  ? allProjects.filter((p) =>
      browsersEnv.split(',').map((b) => b.trim().toLowerCase()).includes(p.name.toLowerCase()),
    )
  : allProjects

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'html',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  projects,
  // CI startet Backend + Frontend separat via Workflow-Steps; lokal startet
  // Playwright den Frontend-Dev-Server selbst (Backend wird vom Developer
  // beigesteuert, sonst greift der ensureBackendUp-Skip).
  webServer: process.env.CI
    ? undefined
    : {
        command: 'npm run dev',
        url: 'http://localhost:3000',
        reuseExistingServer: true,
      },
})
