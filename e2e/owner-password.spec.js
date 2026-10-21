// @ts-check
/**
 * E2E tests for owner password sign-in (OWNER_PASSWORD_HASH).
 *
 * The test stack sets the hash of OWNER_PASSWORD below for hello@example.com,
 * so the sign-in screen offers a password next to the email code.
 */
import { test, expect } from '@playwright/test';

const OWNER_EMAIL = 'hello@example.com';
const OWNER_PASSWORD = 'e2e owner password';

async function openPasswordForm(page) {
  await page.goto('/');
  await page.locator('mol_button_minor[trip2g_user_space_signinbutton]').click();
  await page.locator('[trip2g_auth_email_form_password_mode]').click();
  await page.locator('input[trip2g_auth_password_form_email_control]').fill(OWNER_EMAIL);
}

test.describe('owner password sign-in', () => {
  test('a wrong password is refused on the form', async ({ page }) => {
    await openPasswordForm(page);
    await page.locator('input[trip2g_auth_password_form_password_control_pass]').fill('not the password');
    await page.locator('[trip2g_auth_password_form_submit]').click();

    await expect(page.getByText('Email or password is invalid')).toBeVisible();
  });

  test('the owner signs in with the password', async ({ page }) => {
    await openPasswordForm(page);
    await page.locator('input[trip2g_auth_password_form_password_control_pass]').fill(OWNER_PASSWORD);
    await page.locator('[trip2g_auth_password_form_submit]').click();

    await expect(page.locator('mol_view[trip2g_user_space_home_email_labeler_content]')).toContainText(OWNER_EMAIL);
  });
});
