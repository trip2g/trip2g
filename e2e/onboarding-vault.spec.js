// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:8081';
const ENDPOINT = `${APP_URL}/_system/onboarding-vault`;

test.describe('Onboarding vault download', () => {
  test('returns 401 for anonymous request', async ({ request }) => {
    const res = await request.get(ENDPOINT);
    expect(res.status()).toBe(401);
  });

  test('returns zip archive for admin', async ({ request }) => {
    const token = await graphqlSignIn(request);
    const res = await request.get(ENDPOINT, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status()).toBe(200);
    expect(res.headers()['content-type']).toBe('application/zip');
    const body = await res.body();
    expect(body.length).toBeGreaterThan(0);
  });
});
