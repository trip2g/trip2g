// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

test.describe('Sign-in Wall', () => {
  test.describe.configure({ mode: 'serial' });

  /** @type {string} */
  let adminToken;
  /** @type {number|null} */
  let subgraphId = null;
  /** @type {string} */
  let subgraphColor = '';
  /** @type {boolean} */
  let subgraphHidden = false;

  test.beforeAll(async ({ request }) => {
    adminToken = await graphqlSignIn(request);
    const authHeaders = { 'Cookie': `trip2g_e2e=${adminToken}` };

    // Find the "signin_required" subgraph and save its current settings
    const listResponse = await request.post('/graphql', {
      headers: authHeaders,
      data: {
        query: `{
          admin {
            allSubgraphs {
              nodes {
                id
                name
                color
                hidden
                requireSignin
              }
            }
          }
        }`
      }
    });

    expect(listResponse.ok()).toBeTruthy();
    const listData = await listResponse.json();
    const subgraphs = listData.data?.admin?.allSubgraphs?.nodes ?? [];
    const sg = subgraphs.find(s => s.name === 'signin_required');

    if (sg) {
      subgraphId = sg.id;
      subgraphColor = sg.color ?? '';
      subgraphHidden = sg.hidden ?? false;

      // Set requireSignin=true on the signin_required subgraph
      const updateResponse = await request.post('/graphql', {
        headers: authHeaders,
        data: {
          query: `
            mutation UpdateSubgraph($input: UpdateSubgraphInput!) {
              admin {
                data: updateSubgraph(input: $input) {
                  __typename
                  ... on UpdateSubgraphPayload { subgraph { id requireSignin } }
                  ... on ErrorPayload { message }
                }
              }
            }
          `,
          variables: {
            input: {
              id: subgraphId,
              color: subgraphColor,
              hidden: subgraphHidden,
              requireSignin: true,
            }
          }
        }
      });

      expect(updateResponse.ok()).toBeTruthy();
      const updateData = await updateResponse.json();
      if (updateData.data?.admin?.data?.__typename === 'ErrorPayload') {
        throw new Error(`updateSubgraph failed: ${updateData.data.admin.data.message}`);
      }

      // Wait for noteloader to pick up the new require_signin flag
      await new Promise(r => setTimeout(r, 5000));
    }
  });

  test('guest sees sign-in wall on require_signin page', async ({ browser }) => {
    if (subgraphId === null) {
      test.skip(true, 'signin_required subgraph not found in test DB');
      return;
    }

    const context = await browser.newContext({
      baseURL: process.env.APP_URL || 'http://localhost:20080',
    });
    const page = await context.newPage();

    try {
      await page.goto('/signin_wall');

      // Sign-in wall widget should be visible
      const signinWall = page.locator('mol_view[trip2g_user_signinwall_container]');
      await expect(signinWall).toBeVisible();

      // Note title should be rendered above the wall
      await expect(page.locator('h1.content__title').first()).toBeVisible();

      // Email input should be visible inside the sign-in wall (use signinwall-specific selector)
      await expect(page.locator('input[trip2g_user_signinwall_auth_emailform_email_control]')).toBeVisible();
    } finally {
      await context.close();
    }
  });

  test('guest signs in from sign-in wall and sees content', async ({ browser }) => {
    if (subgraphId === null) {
      test.skip(true, 'signin_required subgraph not found in test DB');
      return;
    }

    const context = await browser.newContext({
      baseURL: process.env.APP_URL || 'http://localhost:20080',
    });
    const page = await context.newPage();

    try {
      await page.goto('/signin_wall');

      // Confirm sign-in wall is shown
      await expect(page.locator('mol_view[trip2g_user_signinwall_container]')).toBeVisible();

      // Use signinwall-specific selectors to avoid conflict with space widget
      const emailInput = page.locator('input[trip2g_user_signinwall_auth_emailform_email_control]');
      await emailInput.clear();
      await emailInput.fill('hello@example.com');
      await page.keyboard.press('Enter');

      // Wait for code input
      await page.locator('input[trip2g_user_signinwall_auth_codeform_code_control]').waitFor({ state: 'visible' });

      // Enter dev code and submit with Enter key
      await page.locator('input[trip2g_user_signinwall_auth_codeform_code_control]').clear();
      await page.locator('input[trip2g_user_signinwall_auth_codeform_code_control]').fill('111111');
      await page.keyboard.press('Enter');

      // After sign-in the page reloads; wait for sign-in wall to disappear
      await page.waitForTimeout(1500);

      // Sign-in wall should no longer be shown
      await expect(page.locator('div#signinwall[mol_view_root="$trip2g_user_signinwall"]')).not.toBeVisible();
    } finally {
      await context.close();
    }
  });

  test('guest sees normal content on non-require_signin page', async ({ browser }) => {
    const context = await browser.newContext({
      baseURL: process.env.APP_URL || 'http://localhost:20080',
    });
    const page = await context.newPage();

    try {
      await page.goto('/public');

      // No sign-in wall should be shown
      await expect(page.locator('div#signinwall[mol_view_root="$trip2g_user_signinwall"]')).not.toBeVisible();

      // Content should be visible
      await expect(page.locator('h1').first()).toContainText('Public Content');
    } finally {
      await context.close();
    }
  });
});
