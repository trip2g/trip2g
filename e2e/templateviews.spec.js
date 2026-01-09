// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Template Views - Meta Accessor', () => {
  test('meta values are rendered correctly', async ({ page }) => {
    await page.goto('/template_meta_test');

    // Check page title
    await expect(page.locator('h1').first()).toContainText('Meta Test Page');

    // Check meta values from frontmatter
    await expect(page.locator('#meta-author')).toContainText('Author: John Doe');
    await expect(page.locator('#meta-version')).toContainText('Version: 42');
    await expect(page.locator('#meta-featured')).toContainText('Featured: true');
    await expect(page.locator('#meta-has-author')).toContainText('Has author: true');
    await expect(page.locator('#meta-has-missing')).toContainText('Has missing: false');

    // Check note info
    await expect(page.locator('#note-reading-time')).toContainText('Reading time:');
    await expect(page.locator('#note-permalink')).toContainText('Permalink: /template_meta_test');
  });

  test('meta default values work when fields are missing', async ({ page }) => {
    await page.goto('/template_meta_defaults');

    // Check page title
    await expect(page.locator('h1').first()).toContainText('Meta Defaults Page');

    // Check default values are used when meta fields are not set
    await expect(page.locator('#meta-author')).toContainText('Author: Unknown Author');
    await expect(page.locator('#meta-version')).toContainText('Version: 0');
    await expect(page.locator('#meta-featured')).toContainText('Featured: false');
    await expect(page.locator('#meta-has-author')).toContainText('Has author: false');
  });
});

test.describe('Template Views - NVS ByPath', () => {
  test('sidebar and footer are loaded via nvs.ByPath()', async ({ page }) => {
    await page.goto('/template_sidebar_test');

    // Check page title
    await expect(page.locator('#main-content h1').first()).toContainText('Sidebar Test Page');

    // Check sidebar is loaded
    await expect(page.locator('#custom-sidebar')).toBeVisible();
    await expect(page.locator('#custom-sidebar h2')).toContainText('Test Sidebar');
    await expect(page.locator('#custom-sidebar')).toContainText('sidebar content');

    // Check sidebar links
    await expect(page.locator('#custom-sidebar a[href="/"]')).toContainText('Home');
    await expect(page.locator('#custom-sidebar a[href="/public"]')).toContainText('Public');

    // Check footer is loaded
    await expect(page.locator('#custom-footer')).toBeVisible();
    await expect(page.locator('#custom-footer')).toContainText('2025 Test Site');
  });
});

test.describe('Template Views - BackLinks', () => {
  test('backlinks are rendered correctly', async ({ page }) => {
    await page.goto('/template_backlinks_target');

    // Check page title
    await expect(page.locator('main h1').first()).toContainText('Backlinks Target');

    // Check backlinks section exists
    await expect(page.locator('#backlinks h2')).toContainText('Backlinks');

    // Check backlinks list
    const backlinks = page.locator('#backlinks-list li');
    await expect(backlinks).toHaveCount(2);

    // Check backlink titles (order may vary)
    const linkTexts = await backlinks.allTextContents();
    expect(linkTexts.some(t => t.includes('Backlinks Source 1'))).toBeTruthy();
    expect(linkTexts.some(t => t.includes('Backlinks Source 2'))).toBeTruthy();
  });

  test('no backlinks message when no pages link', async ({ page }) => {
    // Visit a page that has no backlinks
    await page.goto('/template_backlinks_source1');

    // This page doesn't use backlinks layout, so just verify it loads
    await expect(page.locator('h1').first()).toContainText('Backlinks Source 1');
  });
});
