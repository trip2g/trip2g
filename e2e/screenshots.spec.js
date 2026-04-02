// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Screenshot Tests', () => {
  test('home page light theme', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveScreenshot('home-light.png', {
      fullPage: true,
    });
  });

  test('home page dark theme', async ({ page }) => {
    await page.goto('/');
    // Toggle to dark theme
    await page.locator('mol_lights_toggle').click();
    await page.waitForTimeout(300);
    await expect(page).toHaveScreenshot('home-dark.png', {
      fullPage: true,
    });
  });
});
