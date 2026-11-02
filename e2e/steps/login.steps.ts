import { createBdd } from 'playwright-bdd';
import { expect } from '@playwright/test';

const { Given, When, Then } = createBdd();

const AUTH_EMULATOR = process.env.FIREBASE_AUTH_EMULATOR_HOST || '127.0.0.1:9099';

Given('a test user exists with email {string} and password {string}', async ({ request }, email: string, password: string) => {
  const resp = await request.post(
    `http://${AUTH_EMULATOR}/identitytoolkit.googleapis.com/v1/accounts:signUp?key=fake-key`,
    {
      data: { email, password, returnSecureToken: true },
      headers: { 'Content-Type': 'application/json' },
    }
  );
  if (!resp.ok()) {
    throw new Error(`Failed to create test user: HTTP ${resp.status()} — ${await resp.text()}`);
  }
});

When('I visit the login page', async ({ page }) => {
  await page.goto('/login');
  await page.waitForSelector('form[data-testid="login-form"]', { timeout: 5000 });
});

When('I sign in with email {string} and password {string}', async ({ page }, email: string, password: string) => {
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');
});

Then('I should be on the matches page', async ({ page }) => {
  await expect(page).toHaveURL('/matches', { timeout: 10000 });
});

When('I sign out', async ({ page }) => {
  await page.goto('/auth/logout');
  await page.waitForURL('/matches', { timeout: 5000 });
});
