import { defineConfig, devices } from '@playwright/test';

// E2E contra el stack real (plan 13 fase 10): compose con --profile frontend
// sirve el SPA buildeado en :5173, con /api en el mismo origen.
// Los helpers de setup acceden al gateway TLS local con ignoreHTTPSErrors.
// Los recorridos Anonymous y Customer corren en desktop + mobile; Admin solo
// desktop (la tabla admin es una superficie desktop-first).
export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.js',
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:5173',
    ignoreHTTPSErrors: true,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium-desktop',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } },
    },
    {
      name: 'chromium-mobile',
      use: { ...devices['Pixel 7'] },
      grepInvert: /@desktop-only/,
    },
  ],
});
