// @ts-check
/**
 * /_system/renderlayout E2E Tests
 *
 * Response shape (success):
 *   { previewID, previewURL, warnings: { layout: [], note: [], files: [] } }
 *
 * Response shape (error):
 *   { error: "..." }
 *
 * Body combinations:
 * 1. layout.path + layout.src + note.src — self-contained inline render
 * 2. layout.path + layout.src + overrideFiles — component override
 * 3. Jet compile error → warnings.layout non-empty, previewID still returned
 * 4. Jet runtime error → warnings.layout non-empty
 * 5. Missing layout.path → 400 error
 *
 * Auth:
 * A. No API key  → 401
 * B. Wrong key   → 401
 *
 * GET:
 * D. GET ?preview_id → 200 HTML, no auth
 * E. GET ?preview_id=missing → 404
 * F. GET ?live → HTML with long-poll script
 * G. GET ?longpolling&since=0 → {action, version}
 * H. GET no params, no auth → 401
 *
 * Live reload integration:
 * I. POST → version increments → longpolling resolves with reload
 */

import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const ENDPOINT = `${APP_URL}/_system/renderlayout`;

const SIMPLE_LAYOUT = `<!DOCTYPE html>
<html><head><title>{{ note.Title() }}</title></head>
<body><h1 id="title">{{ note.Title() }}</h1><div id="content">{{ note.HTMLString() }}</div></body>
</html>`;

const LAYOUT_WITH_INCLUDE = `<!DOCTYPE html>
<html><body>
{{ include "/preview_header.html" }}
<main>{{ note.Title() }}</main>
</body></html>`;

const HEADER_COMPONENT = `<header id="header">Preview Header</header>`;

const MARKDOWN_NOTE = `---
title: Test Preview Note
---

Hello from the **preview** endpoint.`;

/** Posts to the renderlayout endpoint and returns { status, body }. */
async function postPreview(request, apiKey, body) {
  const res = await request.post(ENDPOINT, {
    headers: {
      'Content-Type': 'application/json',
      ...(apiKey ? { 'X-API-Key': apiKey } : {}),
    },
    data: body,
  });
  return { status: res.status(), body: await res.json().catch(() => null) };
}

/** Asserts warnings object has the expected shape with empty arrays. */
function expectCleanWarnings(warnings) {
  expect(warnings).toMatchObject({ layout: [], note: [], files: [] });
}

test.describe('/_system/renderlayout', () => {
  let apiKey = '';

  test.beforeAll(() => {
    const apiKeyPath = path.join(process.cwd(), '.test-api-key');
    if (!fs.existsSync(apiKeyPath)) {
      throw new Error('.test-api-key not found — run setup.spec.js first');
    }
    apiKey = fs.readFileSync(apiKeyPath, 'utf8').trim();
  });

  // ── Auth ──────────────────────────────────────────────────────────────────

  test('A: no API key → 401', async ({ request }) => {
    const { status } = await postPreview(request, '', {
      layout: { path: '_layouts/test.html', src: SIMPLE_LAYOUT },
    });
    expect(status).toBe(401);
  });

  test('B: wrong API key → 401', async ({ request }) => {
    const { status } = await postPreview(request, 'wrong-key-xyz', {
      layout: { path: '_layouts/test.html', src: SIMPLE_LAYOUT },
    });
    expect(status).toBe(401);
  });

  // ── Body combinations ─────────────────────────────────────────────────────

  test('1: inline layout + inline note (self-contained)', async ({ request }) => {
    const { status, body } = await postPreview(request, apiKey, {
      layout: { path: '_layouts/test.html', src: SIMPLE_LAYOUT },
      note: { src: MARKDOWN_NOTE },
    });

    expect(status).toBe(200);
    expect(body.previewID).toBeTruthy();
    expect(body.previewURL).toMatch(/preview_id=/);
    expectCleanWarnings(body.warnings);

    const htmlRes = await request.get(`${ENDPOINT}?preview_id=${body.previewID}`);
    expect(htmlRes.status()).toBe(200);
    const html = await htmlRes.text();
    expect(html).toContain('Test Preview Note');
    expect(html).toContain('preview');
  });

  test('2: layout + overrideFiles (component override)', async ({ request }) => {
    const { status, body } = await postPreview(request, apiKey, {
      layout: { path: '_layouts/with_header.html', src: LAYOUT_WITH_INCLUDE },
      note: { src: '# Component Test' },
      overrideFiles: [{ path: 'preview_header.html', src: HEADER_COMPONENT }],
    });

    expect(status).toBe(200);
    expectCleanWarnings(body.warnings);

    const htmlRes = await request.get(`${ENDPOINT}?preview_id=${body.previewID}`);
    const html = await htmlRes.text();
    expect(html).toContain('Preview Header');
    expect(html).toContain('Component Test');
  });

  test('3: Jet compile error → warnings.layout non-empty, previewID returned', async ({ request }) => {
    const { status, body } = await postPreview(request, apiKey, {
      layout: { path: '_layouts/test.html', src: `<html>{{ undefined_func( }}</html>` },
      note: { src: '# Error Test' },
    });

    expect(status).toBe(200);
    expect(body.previewID).toBeTruthy();
    expect(body.warnings.layout.length).toBeGreaterThan(0);
    expect(body.warnings.layout[0]).toMatch(/compile/);
  });

  test('4: Jet runtime error → warnings.layout non-empty', async ({ request }) => {
    const { status, body } = await postPreview(request, apiKey, {
      layout: { path: '_layouts/test.html', src: `<html>{{ nonexistent.SomeMethod() }}</html>` },
      note: { src: '# Runtime Test' },
    });

    expect(status).toBe(200);
    expect(body.previewID).toBeTruthy();
    expect(body.warnings.layout.length).toBeGreaterThan(0);
  });

  test('5: missing layout.path → 400', async ({ request }) => {
    const { status, body } = await postPreview(request, apiKey, {
      note: { src: '# No layout' },
    });
    expect(status).toBe(400);
    expect(body.error).toMatch(/layout.path/);
  });

  // ── GET scenarios ─────────────────────────────────────────────────────────

  test('D: GET ?preview_id returns HTML without auth', async ({ request }) => {
    const { body } = await postPreview(request, apiKey, {
      layout: { path: '_layouts/test.html', src: '<html><body id="ok">ok</body></html>' },
      note: { src: '# ok' },
    });

    const res = await request.get(`${ENDPOINT}?preview_id=${body.previewID}`);
    expect(res.status()).toBe(200);
    expect(res.headers()['content-type']).toContain('text/html');
    expect(await res.text()).toContain('id="ok"');
  });

  test('E: GET ?preview_id=missing → 404', async ({ request }) => {
    const res = await request.get(`${ENDPOINT}?preview_id=00000000`);
    expect(res.status()).toBe(404);
  });

  test('F: GET ?live injects long-poll script', async ({ request }) => {
    await postPreview(request, apiKey, {
      layout: { path: '_layouts/test.html', src: '<html><body>live</body></html>' },
      note: { src: '# live' },
    });

    const res = await request.get(`${ENDPOINT}?live`, {
      headers: { 'X-API-Key': apiKey },
    });
    expect(res.status()).toBe(200);
    const html = await res.text();
    expect(html).toContain('longpolling');
    expect(html).toContain('location.reload');
  });

  test('G: GET ?longpolling returns {action, version}', async ({ request }) => {
    await postPreview(request, apiKey, {
      layout: { path: '_layouts/test.html', src: '<html>poll</html>' },
      note: { src: '# poll' },
    });

    const res = await request.get(`${ENDPOINT}?longpolling&since=0`);
    expect(res.status()).toBe(200);
    const json = await res.json();
    expect(json.action).toBe('reload');
    expect(typeof json.version).toBe('number');
    expect(json.version).toBeGreaterThan(0);
  });

  test('H: GET no params, no auth → 401', async ({ request }) => {
    const res = await request.get(ENDPOINT);
    expect(res.status()).toBe(401);
  });

  // ── Live reload integration ───────────────────────────────────────────────

  test('I: POST increments version → longpolling resolves with reload', async ({ request }) => {
    // Get current version.
    const vRes = await request.get(`${ENDPOINT}?longpolling&since=0`);
    const { version: v } = await vRes.json();

    // Post new render.
    const { status } = await postPreview(request, apiKey, {
      layout: { path: '_layouts/test.html', src: '<html>updated</html>' },
      note: { src: '# updated' },
    });
    expect(status).toBe(200);

    // Long-poll with old version → should immediately return reload.
    const pollRes = await request.get(`${ENDPOINT}?longpolling&since=${v}`);
    const poll = await pollRes.json();
    expect(poll.action).toBe('reload');
    expect(poll.version).toBeGreaterThan(v);
  });
});
