// @ts-check
import { test, expect } from '@playwright/test';

/**
 * Smoke tests - быстрые базовые проверки что сайт работает
 * Запускаются первыми, если упадут - остальные тесты не имеют смысла
 */

test.describe('Smoke Tests', () => {
  test('home page loads', async ({ page }) => {
    await page.goto('/');

    // Check page loaded - use first() to avoid strict mode violation
    await expect(page).toHaveTitle(/Test Vault/);
    await expect(page.locator('h1').first()).toContainText('Test Vault');
  });

  test('basic pages return 200', async ({ page }) => {
    const pages = [
      '/',
      '/public',
      '/with_layout',
      '/toc_test',
    ];

    for (const path of pages) {
      const response = await page.goto(path);
      expect(response.status()).toBe(200);
    }
  });

  test('app health endpoint works', async ({ request }) => {
    const response = await request.get('http://localhost:20082/healthz');
    expect(response.ok()).toBeTruthy();
  });
});
