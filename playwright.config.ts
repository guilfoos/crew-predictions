import { defineConfig } from '@playwright/test';
import { defineBddConfig } from 'playwright-bdd';

const testDir = defineBddConfig({
  features: 'e2e/features/**/*.feature',
  steps: 'e2e/steps/**/*.steps.ts',
});

export default defineConfig({
  testDir,
  workers: 2,
  globalSetup: './e2e/global-setup.ts',
  reporter: process.env.CI
    ? [['list'], ['github'], ['html', { open: 'never' }], ['junit', { outputFile: 'playwright-results.xml' }]]
    : [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: 'http://localhost:8083',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  projects: [
    // Auth scenarios: no @reset, no shared mutable state. Run in parallel.
    {
      name: 'auth',
      grepInvert: /@reset/,
    },
    // App scenarios: tagged @reset because each one calls DELETE /admin/reset
    // against the shared Go server's in-memory store. Forced to workers: 1
    // because multiple workers would race on global state — Worker A seeds
    // m-A, Worker B resets and wipes m-A, Worker A asserts on m-A → fails.
    // To run @reset in parallel, give each worker its own Go server + Vite
    // preview on port-shifted bases. See BACKLOG: "Per-worker server
    // isolation" (trigger when this project's runtime exceeds 90s).
    {
      name: 'app',
      grep: /@reset/,
      workers: 1,
    },
  ],
  webServer: [
    {
      command: process.env.CI ? './server' : 'go run ./cmd/server',
      port: 8092,
      reuseExistingServer: !process.env.CI,
      env: {
        PORT: '8092',
        FIREBASE_AUTH_EMULATOR_HOST: '127.0.0.1:9099',
        FIREBASE_PROJECT_ID: 'crew-predictions-510902',
        FIREBASE_API_KEY: 'fake-api-key',
        FIREBASE_AUTH_DOMAIN: 'localhost',
        TEST_MODE: '1',
      },
    },
    {
      command: process.env.CI ? 'vite preview --port 8083' : 'vite build --logLevel silent && vite preview --port 8083',
      port: 8083,
      reuseExistingServer: !process.env.CI,
      timeout: 120000,
    },
  ],
});
