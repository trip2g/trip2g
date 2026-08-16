// @ts-check
/**
 * E2E tests for admin-minted sign-in links (`createHatLink` → `/_system/hat`).
 *
 * The link is a way in for someone who already exists. It must not create the
 * account it names, and it must not grant a role — provisioning belongs to the
 * callers that hold the JWT secret (the login-link CLI and the fleet), not to
 * the admin API.
 */
import { test, expect } from '@playwright/test';
import { graphqlSignIn, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:8081';

const CREATE_HAT_LINK = `
  mutation CreateHatLink($input: CreateHatLinkInput!) {
    admin {
      data: createHatLink(input: $input) {
        __typename
        ... on CreateHatLinkPayload { url }
        ... on ErrorPayload { message byFields { name value } }
      }
    }
  }
`;

async function createHatLink(request, cookie, input) {
  const res = await request.post(`${APP_URL}/_system/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query: CREATE_HAT_LINK, variables: { input } },
  });
  expect(res.ok()).toBeTruthy();

  const body = await res.json();
  expect(body.errors).toBeUndefined();

  return body.data.admin.data;
}

test.describe('admin-minted sign-in link', () => {
  let cookie;

  test.beforeAll(async ({ request }) => {
    const jwt = await graphqlSignIn(request);
    cookie = `${USER_TOKEN_COOKIE_NAME}=${jwt}`;
  });

  test('a link for an unknown address is refused and creates nobody', async ({ request }) => {
    const email = `nobody-${Date.now()}@example.com`;
    const link = await createHatLink(request, cookie, { email });
    expect(link.__typename).toBe('CreateHatLinkPayload');

    const res = await request.get(link.url, { maxRedirects: 0 });
    expect(res.status()).toBe(401);
    expect(await res.text()).toContain('no user with email');

    // Redeeming it twice must stay refused: a first exchange that quietly
    // created the account would make the second one succeed.
    const again = await request.get(link.url, { maxRedirects: 0 });
    expect(again.status()).toBe(401);
  });

  test('a link for an existing user signs in and lands on its redirect', async ({ request }) => {
    const link = await createHatLink(request, cookie, {
      email: 'hello@example.com',
      redirectUrl: '/admin/users',
    });
    expect(link.__typename).toBe('CreateHatLinkPayload');

    const res = await request.get(link.url, { maxRedirects: 0 });
    expect(res.status()).toBe(302);
    expect(res.headers()['location']).toBe('/admin/users');
  });

  test('a link with no redirect lands on the site root', async ({ request }) => {
    const link = await createHatLink(request, cookie, { email: 'hello@example.com' });

    const res = await request.get(link.url, { maxRedirects: 0 });
    expect(res.status()).toBe(302);
    expect(res.headers()['location']).toBe('/');
  });

  test('an off-site redirect is refused at mint time', async ({ request }) => {
    for (const redirectUrl of ['//evil.example.com/', 'https://evil.example.com/']) {
      const result = await createHatLink(request, cookie, { email: 'hello@example.com', redirectUrl });

      expect(result.__typename).toBe('ErrorPayload');
      expect(result.byFields.map((f) => f.name)).toContain('redirectUrl');
    }
  });
});
