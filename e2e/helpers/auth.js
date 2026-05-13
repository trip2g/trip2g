// @ts-check
import fs from 'fs';
import path from 'path';

export const USER_TOKEN_COOKIE_NAME = process.env.USER_TOKEN_COOKIE_NAME || 'trip2g_token';

export const ADMIN_JWT_CACHE_PATH = path.join(process.cwd(), 'tmp', '.test-admin-jwt');

/**
 * Create a personal token via GraphQL mutation.
 * @param {import('@playwright/test').APIRequestContext} request
 * @param {string} baseURL
 * @param {string} cookie - full cookie header value, e.g. "trip2g_e2e=<token>"
 * @param {{ name: string, expiresInDays?: number | null }} opts
 * @returns {Promise<{ plaintextToken: string, id: string }>}
 */
export async function createPersonalToken(request, baseURL, cookie, { name, expiresInDays = 90 }) {
  const res = await request.post(`${baseURL}/_system/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: {
      query: `
        mutation CreateUserToken($input: CreateUserTokenInput!) {
          createUserToken(input: $input) {
            ... on CreateUserTokenPayload {
              plaintextToken
              token { id }
            }
            ... on ErrorPayload { message }
          }
        }
      `,
      variables: { input: { name, expiresInDays: expiresInDays ?? null } },
    },
  });
  if (!res.ok()) throw new Error(`createPersonalToken HTTP ${res.status()}`);
  const body = await res.json();
  const payload = body.data?.createUserToken;
  if (!payload) throw new Error(`createPersonalToken: no data: ${JSON.stringify(body)}`);
  if (payload.message) throw new Error(`createPersonalToken error: ${payload.message}`);
  return { plaintextToken: payload.plaintextToken, id: payload.token.id };
}

/**
 * Revoke a personal token via GraphQL mutation.
 * @param {import('@playwright/test').APIRequestContext} request
 * @param {string} baseURL
 * @param {string} cookie
 * @param {string} id - token ID (UUID)
 * @returns {Promise<void>}
 */
export async function revokePersonalToken(request, baseURL, cookie, id) {
  const res = await request.post(`${baseURL}/_system/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: {
      query: `
        mutation RevokeUserToken($input: RevokeUserTokenInput!) {
          revokeUserToken(input: $input) {
            ... on RevokeUserTokenPayload { token { id } }
            ... on ErrorPayload { message }
          }
        }
      `,
      variables: { input: { id } },
    },
  });
  if (!res.ok()) throw new Error(`revokePersonalToken HTTP ${res.status()}`);
  const body = await res.json();
  const payload = body.data?.revokeUserToken;
  if (payload?.message) throw new Error(`revokePersonalToken error: ${payload.message}`);
}

/**
 * Sign in as admin user (hello@example.com) using dev code 111111
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<string>} Session cookie value in format "trip2g_e2e=<value>"
 */
export async function signInAsAdmin(page) {
  await page.goto(process.env.START_PATH || '/');

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
  const sessionCookie = cookies.find(c => c.name === USER_TOKEN_COOKIE_NAME);
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
 * @param {{ useCache?: boolean }} options
 * @returns {Promise<string>} Bearer token for Authorization header
 */
export async function graphqlSignIn(request, email = 'hello@example.com', code = '111111', { useCache = true } = {}) {
  // Return cached admin JWT to avoid parallel sign-in code contention.
  if (useCache && email === 'hello@example.com') {
    try {
      const cached = fs.readFileSync(ADMIN_JWT_CACHE_PATH, 'utf8').trim();
      if (cached) return cached;
    } catch {
      // cache miss — fall through to full sign-in
    }
  }

  // 1. Request email code
  const requestCodeResponse = await request.post('/_system/graphql', {
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
  const requestCodeResult = requestCodeData.data.requestEmailSignInCode;
  if (requestCodeResult.message) {
    throw new Error(`Request code failed: ${requestCodeResult.message}`);
  }

  // 2. Sign in with code
  const signInResponse = await request.post('/_system/graphql', {
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

  // Check for error payload (message field is only set on ErrorPayload)
  if (signInData.data.signInByEmail.message) {
    throw new Error(`Sign in failed: ${signInData.data.signInByEmail.message}`);
  }

  const token = signInData.data.signInByEmail.token;
  if (!token) {
    throw new Error('Sign in succeeded but no token returned');
  }

  return token;
}
