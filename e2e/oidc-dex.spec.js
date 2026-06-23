// E2E for the OIDC login flow against a REAL Dex IdP (docker-compose `dex` service),
// complementing the fast node-mock spec (oidc.spec.js).
//
// Networking: Dex's issuer is http://host.docker.internal:5556. The app container
// reaches it via the host gateway (extra_hosts in docker-compose.test.yml); the
// browser reaches it via the host-resolver rule below (Dex is published on :5556).
import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

const COOKIE = process.env.USER_TOKEN_COOKIE_NAME || 'trip2g_e2e';
const DEX_ISSUER = 'http://host.docker.internal:5556';

// Make the browser resolve the issuer host to the published Dex port.
test.use({
  launchOptions: { args: ['--host-resolver-rules=MAP host.docker.internal 127.0.0.1'] },
});

async function adminGql(request, token, query, variables) {
  const res = await request.post('/_system/graphql', {
    headers: { Cookie: `${COOKIE}=${token}` },
    data: { query, variables },
  });
  expect(res.ok(), `graphql HTTP ${res.status()}`).toBeTruthy();
  return res.json();
}

test.describe.serial('OIDC login via real Dex IdP', () => {
  let adminToken;

  test.beforeAll(async ({ request }) => {
    adminToken = await graphqlSignIn(request);
    // Register Dex as the active OIDC provider (validation probes Dex discovery),
    // auto-provision on so the Dex user logs in without being pre-created.
    const data = await adminGql(request, adminToken, `
      mutation($input: CreateOIDCCredentialsInput!) {
        admin { createOIDCCredentials(input: $input) {
          ... on CreateOIDCCredentialsPayload { credentials { id active } }
          ... on ErrorPayload { message }
        } }
      }
    `, {
      input: {
        name: 'Dex',
        issuer: DEX_ISSUER,
        clientId: 'trip2g-sso-client',
        clientSecret: 'trip2g-secret',
        scopes: 'openid email profile',
        autoProvision: true,
      },
    });
    const payload = data.data.admin.createOIDCCredentials;
    expect(payload.message, `createOIDCCredentials error: ${payload.message}`).toBeUndefined();
  });

  test('user signs in through Dex and gets a trip2g session', async ({ page }) => {
    await page.goto('/_system/auth/oidc?redirect=/', { waitUntil: 'load' });

    // Dex local-password login form.
    await page.fill('input[name="login"]', 'alice@example.com');
    await page.fill('input[name="password"]', 'password123');
    await Promise.all([
      page.waitForLoadState('load'),
      page.click('#submit-login'),
    ]);
    await page.waitForLoadState('networkidle').catch(() => {});

    expect(page.url()).not.toContain('berror=');
    const cookies = await page.context().cookies();
    expect(cookies.find((c) => c.name === COOKIE && c.value)).toBeTruthy();
  });
});
