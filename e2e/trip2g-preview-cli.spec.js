// @ts-check
/**
 * trip2g-preview CLI ↔ /_system/renderlayout wiring.
 *
 * The tool's pure logic (arg parse, payload build, target resolution) is unit-
 * tested in scripts/trip2g-preview.test.mjs. This spec covers the end-to-end
 * wiring against a live server: the spawned process actually POSTs a valid
 * payload, prints a working preview URL, honours --fetch, and uses the documented
 * exit codes.
 */
import { test, expect } from '@playwright/test';
import { execFileSync } from 'child_process';
import fs from 'fs';
import path from 'path';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const TOOL = path.join(process.cwd(), 'scripts', 'trip2g-preview.mjs');

/** Run the tool; capture {code, stdout, stderr} (execFileSync throws on non-zero exit). */
function runTool(args, env = {}) {
  try {
    const stdout = execFileSync('node', [TOOL, ...args], {
      encoding: 'utf8',
      env: { ...process.env, ...env },
    });
    return { code: 0, stdout, stderr: '' };
  } catch (e) {
    return {
      code: typeof e.status === 'number' ? e.status : 1,
      stdout: e.stdout ? e.stdout.toString() : '',
      stderr: e.stderr ? e.stderr.toString() : '',
    };
  }
}

test.describe('trip2g-preview CLI', () => {
  let apiKey = '';
  test.beforeAll(() => {
    const p = path.join(process.cwd(), '.test-api-key');
    if (!fs.existsSync(p)) throw new Error('.test-api-key not found — run setup.spec.js first');
    apiKey = fs.readFileSync(p, 'utf8').trim();
  });

  const auth = () => ['--api-url', APP_URL, '--api-key', apiKey];

  test('one-shot render → prints a working preview URL, exit 0', async ({ request }) => {
    const r = runTool([
      ...auth(),
      '--layout-path', '/_cli.html',
      '--layout-src', '<h1 id="t">{{ note.Title() }}</h1>',
      '--note-src', '# CLI Wire',
    ]);
    expect(r.code).toBe(0);
    const url = r.stdout.trim();
    expect(url).toMatch(/\/_system\/renderlayout\?preview_id=/);

    const res = await request.get(url);
    expect(res.status()).toBe(200);
    expect(await res.text()).toContain('CLI Wire');
  });

  test('--fetch prints rendered HTML directly', () => {
    const r = runTool([
      ...auth(),
      '--layout-path', '/_cli.html',
      '--layout-src', '<h1 id="t">{{ note.Title() }}</h1>',
      '--note-src', '# Fetched CLI',
      '--fetch',
    ]);
    expect(r.code).toBe(0);
    expect(r.stdout).toContain('Fetched CLI');
  });

  test('Jet error → warnings on stderr, exit 1', () => {
    const r = runTool([
      ...auth(),
      '--layout-path', '/_bad.html',
      '--layout-src', '<h1>{{ note.NoSuchMethod() }}</h1>',
      '--note-src', '# x',
    ]);
    expect(r.code).toBe(1);
    expect(r.stderr).toContain('WARNINGS');
  });

  test('no API key → hint on stderr, exit 1', () => {
    const r = runTool(
      [
        '--api-url', APP_URL,
        '--folder', '/nonexistent-vault-xyz',
        '--layout-path', '/_x.html',
        '--layout-src', 'x',
      ],
      { TRIP2G_API_KEY: '', TRIP2G_API_URL: '' },
    );
    expect(r.code).toBe(1);
    expect(r.stderr).toMatch(/no API key/i);
  });
});
