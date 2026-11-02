import { request } from '@playwright/test';

const AUTH_EMULATOR = process.env.FIREBASE_AUTH_EMULATOR_HOST || '127.0.0.1:9099';
const PROJECT_ID = process.env.FIREBASE_PROJECT_ID || 'crew-predictions';
const APP_URL = 'http://127.0.0.1:8080';

export default async function globalSetup() {
  const ctx = await request.newContext();
  try {
    // Clear all Firebase Auth emulator accounts for a clean slate
    await ctx.delete(
      `http://${AUTH_EMULATOR}/emulator/v1/projects/${PROJECT_ID}/accounts`,
      { headers: { Authorization: 'Bearer owner' } }
    );
  } catch {
    // Emulator not running — auth tests will fail on their own
  }

  try {
    // Reset in-memory prediction/result stores (test mode endpoint)
    await ctx.delete(`${APP_URL}/admin/reset`);
  } catch {
    // Server not up yet or not in test mode — safe to ignore
  } finally {
    await ctx.dispose();
  }
}
