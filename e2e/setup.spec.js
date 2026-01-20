// @ts-check
import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';

/**
 * Setup test - runs first
 * Tests onboarding page, then creates API key via UI for loading data via push_notes.py
 */

test.describe.serial('Setup', () => {
  // Onboarding tests - must run BEFORE any data is loaded
  test('shows onboarding page for guest when no notes', async ({ page }) => {
    await page.goto('/');

    // Should show onboarding, not 404
    await expect(page.locator('[data-onboarding]')).toBeVisible();
    await expect(page.getByText('Сайт в процессе настройки')).toBeVisible();
    await expect(page.getByText('Site is being set up')).toBeVisible();
  });

  test('onboarding-vault returns 401 for guest', async ({ page }) => {
    const response = await page.request.get('/_system/onboarding-vault');
    expect(response.status()).toBe(401);
  });

  test('sign in and create API key via UI', async ({ page }) => {
    await page.goto('/');

    // 1. Click sign in button
    await page.locator('mol_button_minor[trip2g_user_space_signinbutton]').click();

    // 2. Enter email
    await page.locator('input[trip2g_auth_email_form_email_control]').fill('hello@example.com');

    // 3. Click request code
    await page.locator('button[trip2g_auth_email_form_requestcode]').click();

    // Wait for code input to appear
    await page.locator('input[trip2g_auth_code_form_code_control]').waitFor({ state: 'visible' });

    // 4. Enter code (111111 is dev code)
    await page.locator('input[trip2g_auth_code_form_code_control]').clear()
    await page.locator('input[trip2g_auth_code_form_code_control]').fill('111111');

    // 5. Click sign up
    await page.locator('mol_button_major[trip2g_auth_code_form_signup]').click();

    // Wait for sign in to complete
    await page.waitForTimeout(500);

    // Verify onboarding page shows download link for admin
    await page.goto('/');
    await expect(page.locator('[data-onboarding]')).toBeVisible();
    await expect(page.getByText('Добро пожаловать!')).toBeVisible();
    await expect(page.getByText('Welcome!')).toBeVisible();
    await expect(page.locator('a[href="/_system/onboarding-vault"]').first()).toBeVisible();

    // Verify admin can download onboarding vault
    const vaultResponse = await page.request.get('/_system/onboarding-vault');
    expect(vaultResponse.status()).toBe(200);
    expect(vaultResponse.headers()['content-type']).toBe('application/zip');

    await page.goto('/admin');

    // 8. Click API Keys
    await page.getByText('API Keys').click();

    // 9. Click '+ Add'
    await page.getByText('+ Add').click();

    // Wait for create form
    await page.waitForTimeout(500);

    // 10. Click Submit
    await page.locator('mol_button_major[trip2g_admin_apikey_create_submit]').click();

    // Wait for API key to be created
    await page.waitForTimeout(1000);

    // 11. Get the token value
    const tokenElement = await page.locator('trip2g_admin_cell[trip2g_admin_apikey_create_apikeyvalue]');
    await tokenElement.waitFor({ state: 'visible' });

    const apiKey = await tokenElement.textContent();
    expect(apiKey).toBeTruthy();
    expect(apiKey.length).toBeGreaterThan(10);

    console.log(`✓ API Key created via UI: ${apiKey.substring(0, 20)}...`);

    // Save API key to file for use in push_notes.py
    const apiKeyPath = path.join(process.cwd(), '.test-api-key');
    fs.writeFileSync(apiKeyPath, apiKey, 'utf8');
    console.log(`✓ API Key saved to ${apiKeyPath}`);
  });
});
