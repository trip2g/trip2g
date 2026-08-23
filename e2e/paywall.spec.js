// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

// The wall's copy is $mol and localized, so pin the browser language: these tests
// assert on the English text a visitor actually reads.
test.use({ locale: 'en-US' });

test.describe('Paywall', () => {
  // Both tests flip the global show_waitlists config, so they must not overlap.
  test.describe.configure({ mode: 'serial' });

  /** @type {string} */
  let adminCookie;

  /**
   * @param {import('@playwright/test').APIRequestContext} request
   * @param {boolean} value
   */
  async function setShowWaitlists(request, value) {
    const res = await request.post('/_system/graphql', {
      headers: { Cookie: adminCookie },
      data: {
        query: `
          mutation SetShowWaitlists($input: SetConfigBoolValueInput!) {
            admin {
              data: setConfigBoolValue(input: $input) {
                __typename
                ... on SetConfigBoolValueSuccess { configValue { id } }
                ... on ErrorPayload { message }
              }
            }
          }
        `,
        variables: { input: { id: 'show_waitlists', value } },
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const payload = body.data?.admin?.data;
    if (!payload || payload.message) {
      throw new Error(`setConfigBoolValue failed: ${JSON.stringify(body)}`);
    }
  }

  test.beforeAll(async ({ request }) => {
    const token = await graphqlSignIn(request);
    adminCookie = `${USER_TOKEN_COOKIE_NAME}=${token}`;
    await setShowWaitlists(request, false);
  });

  test.afterAll(async ({ request }) => {
    await setShowWaitlists(request, false);
  });

  /**
   * The wall renders the offers block only after `viewer.offers` answers, so the
   * "no wait list" assertions below would pass on a page that simply had not
   * loaded yet. Wait for that response before checking what is absent.
   * @param {import('@playwright/test').Page} page
   */
  function offersAnswered(page) {
    return page.waitForResponse(res =>
      res.url().includes('/graphql') && (res.request().postData() || '').includes('offers'));
  }

  test('guest on a closed page is told it is closed and can sign in', async ({ page }) => {
    const offers = offersAnswered(page);
    await page.goto('/premium');
    await offers;

    await expect(page.locator('h1.content__title').first()).toContainText('Premium Course Home');
    await expect(page.locator('div.paywall-page')).toBeVisible();

    await expect(page.getByText('This page is closed')).toBeVisible();
    await expect(page.getByText('Ask the owner of this knowledge base for access')).toBeVisible();

    // The sign-in form sits on the page itself, not only behind the header widget.
    await expect(page.locator('input[trip2g_user_paywall_auth_emailform_email_control]')).toBeVisible();

    // show_waitlists is off by default, so there is no wait-list e-mail field to
    // mistake for the sign-in field — the confusion this page used to cause.
    await expect(page.getByText('There are no offers right now')).not.toBeVisible();
    await expect(page.locator('input[trip2g_user_paywall_conversationprompt_email_email_control]')).not.toBeVisible();
  });

  test('show_waitlists brings the wait list back', async ({ page, request }) => {
    await setShowWaitlists(request, true);

    await page.goto('/premium');

    await expect(page.getByText('There are no offers right now')).toBeVisible();
    await expect(page.locator('input[trip2g_user_paywall_conversationprompt_email_email_control]')).toBeVisible();
  });
});
