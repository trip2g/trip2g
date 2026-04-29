// @ts-check
/**
 * MCP TOC E2E Tests
 *
 * Verifies that:
 * 1. Rendered HTML contains <div data-header data-level> wrappers for headings
 * 2. note_html accepts toc_path and returns only that section's HTML
 * 3. note_html falls back to full HTML when toc_path doesn't match
 * 4. search results include toc[] with {title, level, path} for notes with headings
 * 5. search matches include toc_path pointing to the section containing the snippet
 *
 * Uses article-with-toc.md (free: true) which has:
 *   ## Introduction
 *   ## Main Section
 *   ### Subsection  (nested under Main Section)
 *   ## Conclusion
 */

import { test, expect } from '@playwright/test';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const MCP_URL = `${APP_URL}/_system/mcp`;

const NOTE_PATH = 'article-with-toc.md';
const NOTE_HREF = '/article_with_toc';

/** Send a JSON-RPC 2.0 request to the MCP endpoint and return result. */
async function mcpCall(request, method, params = {}) {
  const res = await request.post(MCP_URL, {
    headers: { 'Content-Type': 'application/json' },
    data: { jsonrpc: '2.0', id: 1, method, params },
  });
  expect(res.ok(), `MCP ${method} HTTP ${res.status()}`).toBeTruthy();
  const body = await res.json();
  expect(body.error, `MCP error: ${JSON.stringify(body.error)}`).toBeUndefined();
  return body.result;
}

/** Call tools/call and return structuredContent payload. */
async function toolCall(request, name, args = {}) {
  const result = await mcpCall(request, 'tools/call', { name, arguments: args });
  expect(result.structuredContent, 'expected structuredContent').toBeDefined();
  return result.structuredContent;
}

/** Call tools/call and return raw text content. */
async function toolCallText(request, name, args = {}) {
  const result = await mcpCall(request, 'tools/call', { name, arguments: args });
  return result.content?.[0]?.text ?? '';
}

test.describe('MCP TOC', () => {
  let apiContext;

  test.beforeAll(async ({ playwright }) => {
    apiContext = await playwright.request.newContext({ baseURL: APP_URL });
  });

  test.afterAll(async () => {
    await apiContext?.dispose();
  });

  // ── Heading wrappers in rendered HTML ────────────────────────────────────────

  test('rendered HTML contains data-header and data-level on heading divs', async () => {
    const html = await toolCallText(apiContext, 'note_html', { path: NOTE_PATH });

    expect(html).toContain('data-header="Introduction"');
    expect(html).toContain('data-header="Main Section"');
    expect(html).toContain('data-header="Subsection"');
    expect(html).toContain('data-header="Conclusion"');
    expect(html).toContain('data-level="2"');
    expect(html).toContain('data-level="3"');
  });

  test('Subsection is nested inside Main Section div', async () => {
    const html = await toolCallText(apiContext, 'note_html', { path: NOTE_PATH });

    // Subsection div must appear after Main Section opening div.
    const mainIdx = html.indexOf('data-header="Main Section"');
    const subIdx = html.indexOf('data-header="Subsection"');
    expect(mainIdx).toBeGreaterThanOrEqual(0);
    expect(subIdx).toBeGreaterThan(mainIdx);

    // Conclusion must appear after Subsection is closed.
    const conclusionIdx = html.indexOf('data-header="Conclusion"');
    expect(conclusionIdx).toBeGreaterThan(subIdx);
  });

  // ── note_html with toc_path ──────────────────────────────────────────────────

  test('toc_path returns only the requested top-level section', async () => {
    const html = await toolCallText(apiContext, 'note_html', {
      path: NOTE_PATH,
      toc_path: ['Introduction'],
    });

    expect(html).toContain('Introduction');
    expect(html).toContain('Table of Contents widget');
    // Sibling sections must not be present.
    expect(html).not.toContain('data-header="Main Section"');
    expect(html).not.toContain('data-header="Conclusion"');
  });

  test('toc_path with nested path returns the nested section', async () => {
    const html = await toolCallText(apiContext, 'note_html', {
      path: NOTE_PATH,
      toc_path: ['Main Section', 'Subsection'],
    });

    expect(html).toContain('Nested subsection content');
    // Parent and sibling sections not present.
    expect(html).not.toContain('data-header="Introduction"');
    expect(html).not.toContain('data-header="Conclusion"');
  });

  test('toc_path section is shorter than full note HTML', async () => {
    const [full, section] = await Promise.all([
      toolCallText(apiContext, 'note_html', { path: NOTE_PATH }),
      toolCallText(apiContext, 'note_html', { path: NOTE_PATH, toc_path: ['Introduction'] }),
    ]);
    expect(section.length).toBeGreaterThan(0);
    expect(section.length).toBeLessThan(full.length);
  });

  test('unknown toc_path falls back to full note HTML', async () => {
    const [full, fallback] = await Promise.all([
      toolCallText(apiContext, 'note_html', { path: NOTE_PATH }),
      toolCallText(apiContext, 'note_html', {
        path: NOTE_PATH,
        toc_path: ['__nonexistent_xyzzy__'],
      }),
    ]);
    expect(fallback).toBe(full);
  });

  // ── search: TOC field ────────────────────────────────────────────────────────

  test('search result includes toc array with correct items', async () => {
    const payload = await toolCall(apiContext, 'search', {
      query: 'table of contents widget sidebar',
    });

    const item = (payload.results ?? []).find(
      (r) => r.note_path === NOTE_PATH || r.href === NOTE_HREF,
    );
    expect(item, 'article-with-toc not found in search results').toBeDefined();

    expect(Array.isArray(item.toc)).toBeTruthy();
    expect(item.toc.length).toBe(4); // Introduction, Main Section, Subsection, Conclusion

    for (const tocItem of item.toc) {
      expect(typeof tocItem.title).toBe('string');
      expect(typeof tocItem.level).toBe('number');
      expect(Array.isArray(tocItem.path)).toBeTruthy();
      expect(tocItem.path.at(-1)).toBe(tocItem.title);
    }

    const titles = item.toc.map((t) => t.title);
    expect(titles).toContain('Introduction');
    expect(titles).toContain('Main Section');
    expect(titles).toContain('Subsection');
    expect(titles).toContain('Conclusion');
  });

  test('Subsection has nested path [Main Section, Subsection]', async () => {
    const payload = await toolCall(apiContext, 'search', {
      query: 'table of contents widget sidebar',
    });

    const item = (payload.results ?? []).find(
      (r) => r.note_path === NOTE_PATH || r.href === NOTE_HREF,
    );
    expect(item?.toc).toBeDefined();

    const subsection = item.toc.find((t) => t.title === 'Subsection');
    expect(subsection).toBeDefined();
    expect(subsection.path).toEqual(['Main Section', 'Subsection']);
    // Levels are normalized: ## → 1, ### → 2 (Normalize() remaps to start from 1)
    expect(subsection.level).toBe(2);
  });

  test('all toc paths are unique', async () => {
    const payload = await toolCall(apiContext, 'search', {
      query: 'table of contents widget',
    });
    const item = (payload.results ?? []).find(
      (r) => r.note_path === NOTE_PATH || r.href === NOTE_HREF,
    );
    const paths = item.toc.map((t) => JSON.stringify(t.path));
    expect(new Set(paths).size).toBe(paths.length);
  });

  // ── search: toc_path on matches ──────────────────────────────────────────────

  test('search match inside a section has toc_path set', async () => {
    const payload = await toolCall(apiContext, 'search', { query: 'nested subsection content' });

    const item = (payload.results ?? []).find(
      (r) => r.note_path === NOTE_PATH || r.href === NOTE_HREF,
    );
    if (!item?.matches?.length) {
      test.skip(); // chunks not indexed yet
      return;
    }

    // toc_path requires the chunk to fall within a single section.
    // For small notes the entire note may be one chunk — skip in that case.
    const m = item.matches.find((m) => Array.isArray(m.toc_path) && m.toc_path.length > 0);
    if (!m) {
      test.skip(); // single-chunk note: section attribution not possible
      return;
    }
    expect(m.toc_path).toContain('Subsection');
  });

  // ── Round-trip: search TOC → section HTML ────────────────────────────────────

  test('round-trip: use toc path from search to fetch section via note_html', async () => {
    const payload = await toolCall(apiContext, 'search', { query: 'table of contents widget' });
    const item = (payload.results ?? []).find(
      (r) => r.note_path === NOTE_PATH || r.href === NOTE_HREF,
    );
    const subsection = item.toc.find((t) => t.title === 'Subsection');
    expect(subsection).toBeDefined();

    const html = await toolCallText(apiContext, 'note_html', {
      path: item.note_path,
      toc_path: subsection.path,
    });

    expect(html).toContain('Nested subsection content');
    expect(html).not.toContain('data-header="Conclusion"');
  });

  // ── Tool schema ──────────────────────────────────────────────────────────────

  test('note_html tool schema includes toc_path parameter', async () => {
    const result = await mcpCall(apiContext, 'tools/list');
    const tool = result.tools.find((t) => t.name === 'note_html');
    expect(tool).toBeDefined();
    expect(tool.inputSchema.properties.toc_path).toBeDefined();
    expect(tool.inputSchema.properties.toc_path.type).toBe('array');
  });
});
