// @ts-check
import { test, expect } from '@playwright/test';

/**
 * RSS & Sitemap tests — проверяют генерацию RSS-фида (шаблонного, /feed.xml)
 * и sitemap.xml
 */

test.describe('RSS Feed', () => {
  test('returns valid RSS XML with the expected 2.0 shape', async ({ request }) => {
    const response = await request.get('/feed.xml');
    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('application/rss+xml');

    const body = await response.text();
    expect(body).toContain('<rss version="2.0"');
    expect(body).toContain('<channel>');
    expect(body).toContain('<item>');
    expect(body).toContain('<title>');
    expect(body).toContain('<link>');
    expect(body).toContain('<guid>');
  });

  test('includes a free note and excludes a paid note', async ({ request }) => {
    const response = await request.get('/feed.xml');
    const body = await response.text();

    // public.md is free: true.
    expect(body).toContain('/public');
    expect(body).toContain('Public Content Page');

    // paid_with_cut.md has no `free` key (defaults to paid).
    expect(body).not.toContain('/paid_with_cut');
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
