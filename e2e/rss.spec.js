// @ts-check
import { test, expect } from '@playwright/test';

/**
 * RSS & Sitemap tests — проверяют генерацию RSS-фидов и sitemap.xml
 */

test.describe('RSS Feed', () => {
  test('returns valid RSS XML for a note', async ({ request }) => {
    const response = await request.get('/.rss.xml');
    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('application/rss+xml');

    const body = await response.text();
    expect(body).toContain('<?xml');
    expect(body).toContain('<rss version="2.0"');
    expect(body).toContain('<channel>');
    expect(body).toContain('<item>');
  });

  test('RSS feed contains links from the note', async ({ request }) => {
    const response = await request.get('/.rss.xml');
    const body = await response.text();

    // _index.md contains [[public]] wikilink which resolves to /public
    expect(body).toContain('/public');
  });

  test('RSS items have metadata from target notes', async ({ request }) => {
    const response = await request.get('/.rss.xml');
    const body = await response.text();

    // public.md has description "This is a public page available to everyone"
    expect(body).toContain('This is a public page available to everyone');
  });

  test('returns 404 for non-existent note RSS', async ({ request }) => {
    const response = await request.get('/nonexistent-page-xyz.rss.xml');
    // Should fall through to 404 handler since note doesn't exist
    expect(response.status()).not.toBe(200);
  });
});

test.describe('Sitemap', () => {
  test('returns valid sitemap XML', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('application/xml');

    const body = await response.text();
    expect(body).toContain('<?xml');
    expect(body).toContain('<urlset');
    expect(body).toContain('sitemaps.org');
  });

  test('sitemap contains free pages', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    const body = await response.text();

    // public.md is free: true
    expect(body).toContain('/public');
  });

  test('sitemap excludes system pages', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    const body = await response.text();

    // System pages like /_banner should not appear
    expect(body).not.toContain('/_banner');
    expect(body).not.toContain('/_sidebar');
  });
});
