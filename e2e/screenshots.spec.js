// @ts-check
// To update snapshots: ./scripts/update_screenshots.sh
import { test, expect } from '@playwright/test';

test.describe('Screenshot Tests', () => {
  // The baselines belong to the seeded e2e vault. A run against the dev instance
  // (default baseURL :8081 serves the landing page) would otherwise rewrite them
  // with a different site, and the landing has no $mol header to open.
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toHaveText('Test Vault Home');
  });

  test('home page light theme', async ({ page }) => {
    await expect(page).toHaveScreenshot('home-light.png', {
      fullPage: true,
    });
  });

  test('home page dark theme', async ({ page }) => {
    await page.locator('[trip2g_user_space_options_pop]').click();
    await page.locator('mol_lights_toggle').click();
    await page.waitForTimeout(300);
    await expect(page).toHaveScreenshot('home-dark.png', {
      fullPage: true,
    });
  });
});
