import { createBdd } from 'playwright-bdd'
import { expect } from '@playwright/test'

const { Given, When, Then } = createBdd()

const SMOKE_PASSWORD = process.env.SMOKE_TEST_PASSWORD!

Given('I am on an iPhone 15 viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
})

Given('I am on a Galaxy S24 viewport', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 780 })
})

When('I visit the staging login page', async ({ page }) => {
  await page.goto('/login')
})

When('I visit the staging sign-up page', async ({ page }) => {
  await page.goto('/signup')
})

When('I sign in with email {string}', async ({ page }, email: string) => {
  await page.fill('input[type="email"]', email)
  await page.fill('input[type="password"]', SMOKE_PASSWORD)
  await page.click('button[type="submit"]')
})

When('I sign up with email {string}', async ({ page }, email: string) => {
  await page.fill('input[type="email"]', email)
  await page.fill('input[type="password"]', SMOKE_PASSWORD)
  await page.click('button[type="submit"]')
})

When('I click the Google sign-in button', async ({ page }) => {
  await page.click('button[data-testid="google-signin"]')
})

Then('I should be on the staging matches page', async ({ page }) => {
  await page.waitForURL('**/matches', { timeout: 15000 })
})

Then('I should see {string} in the staging header', async ({ page }, text: string) => {
  await expect(page.locator('.site-header')).toContainText(text, { timeout: 10000 })
})

Then('the page should navigate toward Google for authentication', async ({ page }) => {
  await page.waitForURL(/accounts\.google\.com|subtle-canto-513501-i3\.firebaseapp\.com\/__\/auth\/handler/, { timeout: 10000 })
})
