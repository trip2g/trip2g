// @ts-check
import { test, expect } from '@playwright/test';

/**
 * Render-all smoke test.
 *
 * The admin render-all page derives every list-level admin route from the nav
 * catalog and renders each in a same-origin iframe. A component that fails to
 * render exposes a [mol_view_error] attribute in its iframe document.
 *
 * This spec exercises the built-in fault emulation:
 *  1. clean load  -> zero render errors across all admin routes
 *  2. emulate on  -> exactly two routes fail (cronjob catalog + healthchecks)
 *  3. emulate off -> back to zero errors
 *
 * Ground truth is the [mol_view_error] attribute (value != 'Promise', which
 * only marks a still-loading view), not the render-all cell status text.
 */

const RENDERALL_URI = '/admin#!nav=system/system_nav=renderall';

/**
 * Sign in as admin (hello@example.com, dev code 111111) via GraphQL. The
 * signInByEmail mutation sets the session cookie on the page's context, which
 * authorizes subsequent /admin navigation. Cookie based so it does not depend
 * on the homepage rendering a sign-in widget (the marketing landing does not).
 */
async function signInAdmin(page) {
  const requestCode = await page.request.post('/_system/graphql', {
    data: {
      query: `mutation ($input: RequestEmailSignInCodeInput!) {
        requestEmailSignInCode(input: $input) {
          ... on RequestEmailSignInCodePayload { success }
          ... on ErrorPayload { message }
        }
      }`,
      variables: { input: { email: 'hello@example.com' } },
    },
  });
  expect(requestCode.ok(), 'requestEmailSignInCode HTTP ok').toBeTruthy();

  const signIn = await page.request.post('/_system/graphql', {
    data: {
      query: `mutation ($input: SignInByEmailInput!) {
        signInByEmail(input: $input) {
          ... on SignInPayload { token }
          ... on ErrorPayload { message }
        }
      }`,
      variables: { input: { email: 'hello@example.com', code: '111111' } },
    },
  });
  expect(signIn.ok(), 'signInByEmail HTTP ok').toBeTruthy();
  const body = await signIn.json();
  expect(body.data?.signInByEmail?.token, `sign in failed: ${JSON.stringify(body)}`).toBeTruthy();
}

/** Collect iframes whose document carries a real (non-loading) render error. */
async function errorFrames(page) {
  return await page.evaluate(() => {
    const out = [];
    for (const f of Array.from(document.querySelectorAll('iframe'))) {
      let doc;
      try { doc = f.contentDocument; } catch { continue; }
      if (!doc || !doc.body) continue;
      const errors = Array.from(doc.querySelectorAll('[mol_view_error]'))
        .map(el => el.getAttribute('mol_view_error'))
        .filter(v => v !== 'Promise');
      if (errors.length) {
        out.push({
          src: f.getAttribute('src') || '',
          errors: [...new Set(errors)],
          text: (doc.body.innerText || '').replace(/\s+/g, ' ').trim(),
        });
      }
    }
    return out;
  });
}

/** Wait until every iframe src matches the expected test_fail marker state. */
async function waitFramesMarked(page, shouldContain) {
  await expect.poll(async () => {
    return await page.evaluate((want) => {
      const iframes = Array.from(document.querySelectorAll('iframe'));
      if (!iframes.length) return false;
      return iframes.every(f => ((f.getAttribute('src') || '').includes('test_fail=1')) === want);
    }, shouldContain);
  }, { timeout: 30_000, intervals: [500, 1000, 2000] }).toBe(true);
}

/** Wait until no iframe is still loading a view (no mol_view_error="Promise"). */
async function waitFramesSettled(page) {
  await expect.poll(async () => {
    return await page.evaluate(() => {
      const iframes = Array.from(document.querySelectorAll('iframe'));
      if (!iframes.length) return -1;
      let pending = 0;
      for (const f of iframes) {
        let doc;
        try { doc = f.contentDocument; } catch { continue; }
        if (!doc || !doc.body) { pending++; continue; }
        pending += doc.querySelectorAll('[mol_view_error="Promise"]').length;
      }
      return pending;
    });
  }, { timeout: 60_000, intervals: [1000, 2000, 3000] }).toBe(0);
}

test.describe.serial('Render All smoke test', () => {
  test('clean load then fault emulation across admin routes', async ({ page }) => {
    test.setTimeout(240_000);

    await signInAdmin(page);

    // 1. Clean load: every admin route renders without error.
    await page.goto(RENDERALL_URI);
    await page.waitForSelector('iframe', { timeout: 30_000 });
    await waitFramesMarked(page, false);
    await waitFramesSettled(page);

    const clean = await errorFrames(page);
    expect(clean, `expected a clean render, got: ${JSON.stringify(clean)}`).toHaveLength(0);

    // 2. Emulate failures: toggle the checkbox on.
    await page.getByText('Emulate failures').click();
    await waitFramesMarked(page, true);
    await waitFramesSettled(page);

    await expect.poll(async () => (await errorFrames(page)).length, { timeout: 60_000 }).toBe(2);

    const faulted = await errorFrames(page);
    expect(faulted).toHaveLength(2);

    const cron = faulted.find(f => f.src.includes('system_nav=cronjobs'));
    const health = faulted.find(f => f.src.includes('system_nav=healthchecks'));
    expect(cron, `cronjob catalog route not flagged: ${JSON.stringify(faulted)}`).toBeTruthy();
    expect(health, `healthchecks route not flagged: ${JSON.stringify(faulted)}`).toBeTruthy();
    expect(cron.src).toContain('test_fail=1');
    expect(health.src).toContain('test_fail=1');
    expect(cron.text).toContain('test_fail: emulated cronjob catalog failure');
    expect(health.text).toContain('test_fail: emulated healthchecks failure');

    // 3. Clean again: toggle the checkbox off.
    await page.getByText('Emulate failures').click();
    await waitFramesMarked(page, false);
    await waitFramesSettled(page);

    await expect.poll(async () => (await errorFrames(page)).length, { timeout: 60_000 }).toBe(0);
  });
});
