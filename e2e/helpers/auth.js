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

/**
 * Sign in via GraphQL API (for request fixture in beforeAll hooks)
 * @param {import('@playwright/test').APIRequestContext} request
 * @param {string} email - Email address (default: hello@example.com)
 * @param {string} code - Auth code (default: 111111 dev code)
 * @returns {Promise<string>} Bearer token for Authorization header
 */
export async function graphqlSignIn(request, email = 'hello@example.com', code = '111111') {
  // 1. Request email code
  const requestCodeResponse = await request.post('/graphql', {
    data: {
      query: `
        mutation RequestCode($input: RequestEmailSignInCodeInput!) {
          requestEmailSignInCode(input: $input) {
            ... on RequestEmailSignInCodePayload {
              success
            }
            ... on ErrorPayload {
              message
            }
          }
        }
      `,
      variables: {
        input: { email }
      }
    }
  });

  if (!requestCodeResponse.ok()) {
    throw new Error(`Request code failed: ${requestCodeResponse.status()}`);
  }

  const requestCodeData = await requestCodeResponse.json();
  if (requestCodeData.data.requestEmailSignInCode.__typename === 'ErrorPayload') {
    throw new Error(`Request code failed: ${requestCodeData.data.requestEmailSignInCode.message}`);
  }

  // 2. Sign in with code
  const signInResponse = await request.post('/graphql', {
    data: {
      query: `
        mutation SignIn($input: SignInByEmailInput!) {
          signInByEmail(input: $input) {
            ... on SignInPayload {
              token
            }
            ... on ErrorPayload {
              message
            }
          }
        }
      `,
      variables: {
        input: { email, code }
      }
    }
  });

  if (!signInResponse.ok()) {
    throw new Error(`Sign in request failed: ${signInResponse.status()}`);
  }

  const signInData = await signInResponse.json();

  // Check for error payload
  if (signInData.data.signInByEmail.__typename === 'ErrorPayload') {
    throw new Error(`Sign in failed: ${signInData.data.signInByEmail.message}`);
  }

  const token = signInData.data.signInByEmail.token;
  if (!token) {
    throw new Error('Sign in succeeded but no token returned');
  }

  return token;
}
