// E2E for the trip2g OIDC / SSO login flow (docs/dev/oidc.md).
//
// Drives the real backend flow against a host-run mock OIDC issuer
// (e2e/helpers/mock-idp.js): /_system/auth/oidc -> mock /authorize ->
// /_system/auth/oidc/callback -> session cookie. Covers the account policy
// (auto_provision off/on + domain/group gating) and the error redirects.
import { test, expect } from '@playwright/test';
import { startMockIdp } from './helpers/mock-idp.js';
import { graphqlSignIn } from './helpers/auth.js';

const COOKIE = process.env.USER_TOKEN_COOKIE_NAME || 'trip2g_e2e';

async function adminGql(request, token, query, variables) {
  const res = await request.post('/_system/graphql', {
    headers: { Cookie: `${COOKIE}=${token}` },
    data: { query, variables },
  });
  expect(res.ok(), `graphql HTTP ${res.status()}`).toBeTruthy();
  return res.json();
}

async function createUser(request, token, email) {
  const data = await adminGql(request, token, `
    mutation($input: CreateUserInput!) {
      admin { createUser(input: $input) {
        ... on CreateUserPayload { user { id } }
        ... on ErrorPayload { message }
      } }
    }
  `, { input: { email } });
  return data.data.admin.createUser;
}

// Creates an active OIDC credential pointing at the mock issuer. createOIDCCredentials
// deactivates any prior credential and validates via a discovery probe to the issuer,
// so the mock must already be running (and reachable from the app container).
async function createOidcCred(request, token, issuer, opts = {}) {
  const data = await adminGql(request, token, `
    mutation($input: CreateOIDCCredentialsInput!) {
      admin { createOIDCCredentials(input: $input) {
        ... on CreateOIDCCredentialsPayload { credentials { id active issuer autoProvision } }
        ... on ErrorPayload { message }
      } }
    }
  `, {
    input: {
      name: 'Mock IdP',
      issuer,
      clientId: 'trip2g-test',
      clientSecret: 'test-secret',
      scopes: 'openid email profile groups',
      autoProvision: opts.autoProvision ?? false,
      allowedEmailDomain: opts.allowedEmailDomain ?? '',
      requiredGroup: opts.requiredGroup ?? '',
    },
  });
  const payload = data.data.admin.createOIDCCredentials;
  expect(payload.message, `createOIDCCredentials error: ${payload.message}`).toBeUndefined();
  return payload.credentials;
}

async function sessionCookie(page) {
  const cookies = await page.context().cookies();
  return cookies.find((c) => c.name === COOKIE && c.value);
}

// Runs the full browser login flow and returns the final URL.
async function login(page, redirect = '/') {
  await page.goto(`/_system/auth/oidc?redirect=${encodeURIComponent(redirect)}`, {
    waitUntil: 'load',
  });
  return page.url();
}

test.describe.serial('OIDC SSO login', () => {
  let idp;
  let adminToken;

  test.beforeAll(async ({ request }) => {
    idp = await startMockIdp();
    adminToken = await graphqlSignIn(request);
    await createUser(request, adminToken, 'existing-oidc@example.com');
  });

  test.afterAll(async () => {
    await idp?.close();
  });

  test('no active provider -> oauth_not_configured', async ({ page }) => {
    const url = await login(page);
    expect(url).toContain('berror=oauth_not_configured');
    expect(await sessionCookie(page)).toBeFalsy();
  });

  test('existing user, auto_provision off -> logged in', async ({ page, request }) => {
    await createOidcCred(request, adminToken, idp.issuer, { autoProvision: false });
    idp.setUser({ email: 'existing-oidc@example.com', email_verified: true });

    const url = await login(page);
    expect(url).not.toContain('berror=');
    expect(await sessionCookie(page)).toBeTruthy();
  });

  test('missing UserInfo verification falls back to matching signed ID token', async ({ page, request }) => {
    await createOidcCred(request, adminToken, idp.issuer, { autoProvision: false });
    idp.setUser({ email: 'existing-oidc@example.com', email_verified: undefined });
    idp.setIdTokenClaims({ email: 'existing-oidc@example.com', email_verified: true });

    const url = await login(page);
    expect(url).not.toContain('berror=');
    expect(await sessionCookie(page)).toBeTruthy();
  });

  test('ID-token fallback rejects a different email', async ({ page, request }) => {
    await createOidcCred(request, adminToken, idp.issuer, { autoProvision: false });
    idp.setUser({ email: 'existing-oidc@example.com', email_verified: undefined });
    idp.setIdTokenClaims({ email: 'other@example.com', email_verified: true });

    const url = await login(page);
    expect(url).toContain('berror=email_not_verified');
    expect(await sessionCookie(page)).toBeFalsy();
  });

  test('unknown email, auto_provision off -> user_not_found', async ({ page, request }) => {
    await createOidcCred(request, adminToken, idp.issuer, { autoProvision: false });
    idp.setUser({ email: 'ghost@example.com', email_verified: true });

    const url = await login(page);
    expect(url).toContain('berror=user_not_found');
    expect(await sessionCookie(page)).toBeFalsy();
  });

  test('unknown email, auto_provision on -> provisioned and logged in', async ({ page, request }) => {
    await createOidcCred(request, adminToken, idp.issuer, { autoProvision: true });
    idp.setUser({ email: 'fresh-user@example.com', email_verified: true });

    const url = await login(page);
    expect(url).not.toContain('berror=');
    expect(await sessionCookie(page)).toBeTruthy();
  });

  test('explicit UserInfo false overrides a verified ID token', async ({ page, request }) => {
    await createOidcCred(request, adminToken, idp.issuer, { autoProvision: true });
    idp.setUser({ email: 'unverified@example.com', email_verified: false });
    idp.setIdTokenClaims({ email: 'unverified@example.com', email_verified: true });

    const url = await login(page);
    expect(url).toContain('berror=email_not_verified');
    expect(await sessionCookie(page)).toBeFalsy();
  });

  test('auto_provision on + required group: rejected when missing, allowed when present', async ({ page, request }) => {
    await createOidcCred(request, adminToken, idp.issuer, {
      autoProvision: true,
      requiredGroup: 'trip2g-admins',
    });

    idp.setUser({ email: 'nogroup@example.com', email_verified: true, groups: ['other'] });
    expect(await login(page)).toContain('berror=email_not_allowed');
    expect(await sessionCookie(page)).toBeFalsy();

    idp.setUser({ email: 'ingroup@example.com', email_verified: true, groups: ['trip2g-admins'] });
    const url = await login(page);
    expect(url).not.toContain('berror=');
    expect(await sessionCookie(page)).toBeTruthy();
  });

  test('callback with bad state -> invalid_state', async ({ page }) => {
    // A credential is active from the previous test; hitting the callback directly
    // (no oauth_state cookie set by the start handler) must fail CSRF validation.
    await page.goto('/_system/auth/oidc/callback?code=x&state=bogus', { waitUntil: 'load' });
    expect(page.url()).toContain('berror=invalid_state');
    expect(await sessionCookie(page)).toBeFalsy();
  });
});
