// @ts-check
import { test, expect } from '@playwright/test';

test.describe('JSON Layout', () => {
  test('renders page with JSON layout and sidebar', async ({ page }) => {
    await page.goto('/json_layout_test');

    // Check page title in header
    await expect(page.locator('#json-layout-header h1')).toContainText('JSON Layout Test Page');

    // Check main content
    await expect(page.locator('#json-layout-main')).toContainText('JSON layout file');

    // Check footer
    await expect(page.locator('#json-layout-footer')).toContainText('JSON Layout Footer');

    // Check sidebar is visible (show_sidebar: true)
    await expect(page.locator('#json-layout-sidebar')).toBeVisible();
    await expect(page.locator('#json-layout-sidebar')).toContainText('Sidebar loaded via note_content');
  });

  test('conditional rendering hides sidebar when show_sidebar is false', async ({ page }) => {
    await page.goto('/json_layout_no_sidebar');

    // Check page title in header
    await expect(page.locator('#json-layout-header h1')).toContainText('JSON Layout No Sidebar');

    // Check main content
    await expect(page.locator('#json-layout-main')).toContainText('sidebar should NOT be visible');

    // Check footer is still visible
    await expect(page.locator('#json-layout-footer')).toContainText('JSON Layout Footer');

    // Sidebar should NOT be visible (show_sidebar: false)
    await expect(page.locator('#json-layout-sidebar')).toHaveCount(0);
  });

  test('include_note shows fallback message for missing files', async ({ page }) => {
    await page.goto('/json_layout_missing_include');

    // Check page renders
    await expect(page.locator('h1').first()).toContainText('JSON Layout Missing Include');

    // Check include_note shows fallback message
    await expect(page.locator('#include-missing-test')).toContainText('Create file: /_nonexistent_file.md');

    // Check main content still renders
    await expect(page.locator('main')).toContainText('tests include_note with a missing file');
  });
});
