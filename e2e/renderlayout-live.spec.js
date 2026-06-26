// @ts-check
/**
 * /_system/renderlayout ?live — browser-level auto-reload.
 *
 * renderlayout.spec.js already covers the HTTP contract (POST, ?preview_id,
 * ?live script injection, ?longpolling version bumps). This spec adds the one
 * thing request-level tests cannot prove: an OPEN ?live page in a real browser
 * actually reloads itself when a new render is POSTed — i.e. the injected
 * long-poll script fires location.reload().
 *
 * The preview buffer is global and specs run in parallel, so we do NOT assert
 * exact rendered content (a concurrent POST could be "latest"). Instead we plant
 * a sentinel on the live page and assert it is wiped by a reload after a new
 * render lands.
 */
import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const ENDPOINT = `${APP_URL}/_system/renderlayout`;

const LAYOUT = (m) =>
  `<!DOCTYPE html><html><head><title>live</title></head><body><h1 id="m">${m}</h1></body></html>`;

test.describe('/_system/renderlayout ?live auto-reload (browser)', () => {
  let apiKey = '';
  test.beforeAll(() => {
    const p = path.join(process.cwd(), '.test-api-key');
    if (!fs.existsSync(p)) throw new Error('.test-api-key not found — run setup.spec.js first');
    apiKey = fs.readFileSync(p, 'utf8').trim();
  });

  test('open ?live, POST a new render → the page reloads itself', async ({ browser }) => {
    // API key on the context so both the ?live GET and the post-reload GET are authed.
    const context = await browser.newContext({ extraHTTPHeaders: { 'X-API-Key': apiKey } });
    const page = await context.newPage();

    // Ensure a render exists, then open the live page.
    await context.request.post(ENDPOINT, {
      data: { layout: { path: '_layouts/live.html', src: LAYOUT(`v1_${Date.now()}`) }, note: { src: '# v1' } },
    });
    await page.goto(`${ENDPOINT}?live`);
    await expect(page.locator('body')).toBeVisible();

    // Plant a sentinel that only survives until the page reloads.
    await page.evaluate(() => { window.__livePreviewSentinel = 'present'; });
    expect(await page.evaluate(() => window.__livePreviewSentinel)).toBe('present');

    // POST a new render → version increments → the long-poll script reloads the page.
    await context.request.post(ENDPOINT, {
      data: { layout: { path: '_layouts/live.html', src: LAYOUT(`v2_${Date.now()}`) }, note: { src: '# v2' } },
    });

    // The reload wipes the sentinel — without any manual navigation.
    await expect
      .poll(
        async () => {
          try {
            return await page.evaluate(() => window.__livePreviewSentinel);
          } catch {
            return 'pending'; // execution context torn down mid-reload — keep polling
          }
        },
        { timeout: 15000 },
      )
      .toBeUndefined();

    // Still a rendered page after the reload.
    await expect(page.locator('body')).toBeVisible();

    await context.close();
  });
});
