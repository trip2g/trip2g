// @ts-check

/**
 * Sign in as admin user (hello@example.com) using dev code 111111
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<string>} Session cookie value in format "trip2g_e2e=<value>"
 */
export async function signInAsAdmin(page) {
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
  await page.locator('input[trip2g_auth_code_form_code_control]').clear();
  await page.locator('input[trip2g_auth_code_form_code_control]').fill('111111');

  // 5. Click sign up
  await page.locator('mol_button_major[trip2g_auth_code_form_signup]').click();

  // Wait for sign in to complete
  await page.waitForTimeout(500);

  // Extract session cookie (trip2g_e2e in E2E tests via USER_TOKEN_COOKIE_NAME)
  const cookies = await page.context().cookies();
  const sessionCookie = cookies.find(c => c.name === 'trip2g_e2e');
  if (!sessionCookie) {
    throw new Error('Session cookie not found after sign in');
  }

  return `${sessionCookie.name}=${sessionCookie.value}`;
}
