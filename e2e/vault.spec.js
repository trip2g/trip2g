// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Test Vault', () => {
  test('home page renders and shows all sections', async ({ page }) => {
    await page.goto('/');

    // Check main heading - use first() to avoid strict mode violation
    await expect(page.locator('h1').first()).toContainText('Test Vault');

    // Check main sections exist - use getByRole to avoid TOC link duplicates
    await expect(page.getByRole('heading', { name: 'Link Resolution Tests' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Publishing Features Tests' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Subgraph (Premium Course) Tests' })).toBeVisible();
  });

  test('public page is accessible without auth', async ({ page }) => {
    await page.goto('/public');

    await expect(page.locator('h1').first()).toContainText('Public Content');
    await expect(page.getByText('publicly accessible')).toBeVisible();
  });

  test('custom layout page uses correct template', async ({ page }) => {
    await page.goto('/with_layout');

    // Check page title
    await expect(page.locator('h1').first()).toContainText('Custom Layout');

    // Check custom layout elements
    await expect(page.locator('header nav')).toBeVisible();
    await expect(page.locator('footer')).toContainText('2025 Test Vault');

    // Check navigation links in custom layout
    await expect(page.locator('a[href="/"]')).toContainText('Home');
  });

  test('premium content preview works', async ({ page }) => {
    await page.goto('/paid_with_preview');

    // Check title is visible
    await expect(page.locator('#noteview-content h1').first()).toContainText('Premium Content with Preview');

    // Page without free flag should show subscription message
    await expect(page.getByText('Эта страница доступна только для подписчиков')).toBeVisible();
  });

  test('table of contents page renders', async ({ page }) => {
    await page.goto('/toc_test');

    // No title set, so it uses filename in lowercase
    await expect(page.locator('h1').first()).toContainText('toc_test');

    // Check sections exist
    // TODO: check better
    // await expect(page.getByRole('heading', { name: 'Section 1' })).toBeVisible();
    // await expect(page.getByRole('heading', { name: 'Section 2' })).toBeVisible();
    // await expect(page.getByRole('heading', { name: 'Section 3' })).toBeVisible();
  });

  test('cyrillic URLs work correctly', async ({ page }) => {
    await page.goto('/cyrillic_названия');

    // Title is set in frontmatter
    await expect(page.locator('h1').first()).toContainText('Проверка кириллицы');
    await expect(page.getByText('кириллическими ссылками')).toBeVisible();
  });

  test('files with spaces in names work', async ({ page }) => {
    await page.goto('/file_with_spaces');

    await expect(page.locator('h1').first()).toContainText('File Name With Spaces');
    await expect(page.getByText('URL normalization')).toBeVisible();
  });

  test('code and media page renders', async ({ page }) => {
    await page.goto('/code_and_media');

    await expect(page.locator('h1').first()).toContainText('Code and Media');

    // Check content exists
    await expect(page.getByText('code blocks and media embeds')).toBeVisible();
  });

  test('complex content with markdown features', async ({ page }) => {
    await page.goto('/complex_content');

    // Check title is visible
    await expect(page.locator('#noteview-content h1').first()).toContainText('Complex Content Example');

    // Page without free flag should show subscription message
    await expect(page.getByText('Эта страница доступна только для подписчиков')).toBeVisible();
  });

  test('navigation between pages works', async ({ page }) => {
    await page.goto('/');

    // Click on a link
    await page.click('text=public');

    // Verify navigation happened
    await expect(page).toHaveURL(/\/public/);
    await expect(page.locator('h1').first()).toContainText('Public Content');
  });

  test('premium course home page', async ({ page }) => {
    await page.goto('/premium');

    // Check title is visible
    await expect(page.locator('.sidebar__homepage a').first()).toContainText('Premium Course Home');

    // Page without free flag should show subscription message
    await expect(page.getByText('Эта страница доступна только для подписчиков')).toBeVisible();
  });
});

test.describe('Link Resolution', () => {
  test('unique filename resolution', async ({ page }) => {
    await page.goto('/unique');

    // No title set, uses filename
    await expect(page.locator('h1').first()).toContainText('unique');
    await expect(page.getByText('should find /folder/deep.md')).toBeVisible();
  });

  test('duplicate filename priority (root wins)', async ({ page }) => {
    await page.goto('/folder/source');

    // No title set, uses filename
    await expect(page.locator('h1').first()).toContainText('source');

    // Check that link to 'dup' resolves to root /dup, not local /folder/dup
    // This verifies Obsidian's global resolution with root priority
    await expect(page.locator('a[href="/dup?version=latest"]')).toBeVisible();
  });

  test('headers and block references', async ({ page }) => {
    await page.goto('/headers');

    // No title set, uses filename
    await expect(page.locator('h1').first()).toContainText('headers');
    await expect(page.getByRole('heading', { name: 'Section One' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Section Two' })).toBeVisible();
  });
});

test.describe('Special Features', () => {
  test('redirect works', async ({ page }) => {
    await page.goto('/redirect_test');

    // Should redirect to /public
    await page.waitForURL(/\/public/);
    await expect(page.locator('h1').first()).toContainText('Public Content');
  });

  test('embedding markdown works', async ({ page }) => {
    await page.goto('/embedding');

    // No title set, uses filename
    await expect(page.locator('h1').first()).toContainText('embedding');

    // Check that embedded banners appear exactly once
    await expect(page.getByText("I'm the ROOT banner at /_banner.md")).toHaveCount(1);
    await expect(page.getByText("I'm the banner at /projectA/_banner.md")).toHaveCount(1);
    await expect(page.getByText("I'm the banner at /projectB/_banner.md")).toHaveCount(1);
  });
});
