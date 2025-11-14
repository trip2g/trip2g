// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Test Vault', () => {
  test('home page renders and shows all sections', async ({ page }) => {
    await page.goto('/');

    // Check main heading
    await expect(page.locator('h1')).toContainText('Test Vault');

    // Check main sections exist
    await expect(page.getByText('Link Resolution Tests')).toBeVisible();
    await expect(page.getByText('Publishing Features Tests')).toBeVisible();
    await expect(page.getByText('Subgraph (Premium Course) Tests')).toBeVisible();
  });

  test('public page is accessible without auth', async ({ page }) => {
    await page.goto('/public');

    await expect(page.locator('h1')).toContainText('Public Content');
    await expect(page.getByText('publicly accessible')).toBeVisible();
  });

  test('custom layout page uses correct template', async ({ page }) => {
    await page.goto('/with_layout');

    // Check page title
    await expect(page.locator('h1')).toContainText('Custom Layout Test');

    // Check custom layout elements
    await expect(page.locator('header nav')).toBeVisible();
    await expect(page.locator('footer')).toContainText('2025 Test Vault');

    // Check navigation links in custom layout
    await expect(page.locator('a[href="/"]')).toContainText('Home');
  });

  test('premium content preview works', async ({ page }) => {
    await page.goto('/paid_with_preview');

    await expect(page.locator('h1')).toContainText('Premium Content');

    // Should show first paragraphs (free preview)
    await expect(page.getByText('first paragraph that everyone can read')).toBeVisible();
    await expect(page.getByText('second free paragraph')).toBeVisible();
  });

  test('table of contents page renders', async ({ page }) => {
    await page.goto('/toc_test');

    await expect(page.locator('h1')).toContainText('TOC Test Page');

    // Check sections exist
    await expect(page.getByRole('heading', { name: 'Section 1' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Section 2' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Section 3' })).toBeVisible();
  });

  test('cyrillic URLs work correctly', async ({ page }) => {
    await page.goto('/cyrillic_названия');

    await expect(page.locator('h1')).toContainText('Кириллица');
    await expect(page.getByText('кириллическими ссылками')).toBeVisible();
  });

  test('files with spaces in names work', async ({ page }) => {
    await page.goto('/file_with_spaces');

    await expect(page.locator('h1')).toContainText('Testing Spaces');
    await expect(page.getByText('URL normalization')).toBeVisible();
  });

  test('code and media page renders', async ({ page }) => {
    await page.goto('/code_and_media');

    await expect(page.locator('h1')).toContainText('Code and Media');

    // Check code block exists
    await expect(page.locator('pre code')).toBeVisible();
    await expect(page.locator('code')).toContainText('def hello_world');
  });

  test('complex content with markdown features', async ({ page }) => {
    await page.goto('/complex_content');

    await expect(page.locator('h1')).toContainText('Complex Content');

    // Check various markdown elements
    await expect(page.locator('ul')).toBeVisible(); // Lists
    await expect(page.locator('table')).toBeVisible(); // Tables
    await expect(page.locator('blockquote')).toBeVisible(); // Blockquotes

    // Check task list
    await expect(page.locator('input[type="checkbox"]')).toBeVisible();
  });

  test('navigation between pages works', async ({ page }) => {
    await page.goto('/');

    // Click on a link
    await page.click('text=public');

    // Verify navigation happened
    await expect(page).toHaveURL(/\/public/);
    await expect(page.locator('h1')).toContainText('Public Content');
  });

  test('premium course home page', async ({ page }) => {
    await page.goto('/premium');

    await expect(page.locator('h1')).toContainText('Welcome to Premium Course');
    await expect(page.getByText('premium subgraph')).toBeVisible();
  });
});

test.describe('Link Resolution', () => {
  test('unique filename resolution', async ({ page }) => {
    await page.goto('/unique');

    await expect(page.locator('h1')).toContainText('Unique File');
    await expect(page.getByText('should find /folder/deep.md')).toBeVisible();
  });

  test('duplicate filename priority (root wins)', async ({ page }) => {
    await page.goto('/folder/source');

    await expect(page.locator('h1')).toContainText('Source File');
    // Проверяем что есть упоминание о дубликатах
    await expect(page.getByText(/dup/)).toBeVisible();
  });

  test('headers and block references', async ({ page }) => {
    await page.goto('/headers');

    await expect(page.locator('h1')).toContainText('Headers Test');
    await expect(page.getByRole('heading', { name: 'Section One' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Section Two' })).toBeVisible();
  });
});

test.describe('Special Features', () => {
  test('redirect works', async ({ page }) => {
    await page.goto('/redirect_test');

    // Should redirect to /public
    await page.waitForURL(/\/public/);
    await expect(page.locator('h1')).toContainText('Public Content');
  });

  test('embedding markdown works', async ({ page }) => {
    await page.goto('/embedding');

    await expect(page.locator('h1')).toContainText('Embedding Test');
    await expect(page.getByText('Global embed')).toBeVisible();
  });
});
