// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Layout CSS Hot-Reload', () => {
  test('custom layout applies updated CSS styles', async ({ page }) => {
    await page.goto('/with_layout');

    // Verify the page loaded correctly
    await expect(page.locator('h1').first()).toContainText('Custom Layout');

    // Check that body has red text color (from updated styles.css)
    await expect(page.locator('body')).toHaveCSS('color', 'rgb(255, 0, 0)');
  });
});
