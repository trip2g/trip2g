// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

const CUSTOM_HOST = 'customdomain.test';
const APP_URL = process.env.APP_URL || 'http://localhost:20081';

/**
 * Multi-domain routing tests.
 * Uses extraHTTPHeaders / per-request Host header to simulate custom domain requests
 * without requiring real DNS — the server reads Host from the HTTP header.
 */

/** GET request to APP_URL with a custom Host header. */
function domainGet(request, path, host = CUSTOM_HOST) {
  return request.get(`${APP_URL}${path}`, { headers: { Host: host } });
}

test.describe('Multi-domain routing', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeAll(async ({ request }) => {
    // Create a frontmatter patch that adds a route to multidomain/no_route.md
    const token = await graphqlSignIn(request);
    const authHeaders = { Cookie: `trip2g_e2e=${token}` };

    const authPost = (query, variables) =>
      request.post('/graphql', {
        headers: authHeaders,
        data: variables ? { query, variables } : { query },
      });

    // Only create the patch if it doesn't exist yet (idempotent)
    const listResp = await authPost(
      `{ admin { allFrontmatterPatches { nodes { id description } } } }`
    );
    const listData = await listResp.json();
    const alreadyExists = listData.data?.admin?.allFrontmatterPatches?.nodes?.some(
      (n) => n.description === 'Test: add route via patch'
    );

    if (!alreadyExists) {
      await authPost(
        `mutation CreatePatch($input: CreateFrontmatterPatchInput!) {
          admin { createFrontmatterPatch(input: $input) { __typename } }
        }`,
        {
          input: {
            description: 'Test: add route via patch',
            includePatterns: ['multidomain/no_route.md'],
            excludePatterns: [],
            jsonnet: '{ "route": "customdomain.test/patch-target" }',
            priority: 0,
            enabled: true,
          },
        }
      );
    }

    // Notes reload automatically after patch creation — wait a moment
    await new Promise((r) => setTimeout(r, 1000));
  });

  test('custom domain root serves correct note', async ({ request }) => {
    const response = await domainGet(request, '/');
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('Custom Domain Root');
  });

  test('custom domain subpath serves correct note', async ({ request }) => {
    const response = await domainGet(request, '/about');
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('Custom Domain About');
  });

  test('multi-route note accessible on custom domain', async ({ request }) => {
    const response = await domainGet(request, '/multi');
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('Multi-Route Note');
  });

  test('multi-route note accessible via main domain alias', async ({ request }) => {
    const response = await request.get(`${APP_URL}/multi-alias`);
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('Multi-Route Note');
  });

  test('main domain alias not served on custom domain', async ({ request }) => {
    // /multi-alias is a main-domain-only route, should 404 on custom domain
    const response = await domainGet(request, '/multi-alias');
    expect(response.status()).toBe(404);
  });

  test('custom domain does not serve notes without explicit routes', async ({ request }) => {
    // Notes without explicit custom-domain routes return 404 on known custom domains.
    // Domain routing is strict: only notes with route: customdomain.test/... are served.
    const response = await domainGet(request, '/public');
    expect(response.status()).toBe(404);
  });

  test('route added via frontmatter patch is accessible', async ({ request }) => {
    // multidomain/no_route.md has no route in frontmatter;
    // the patch in beforeAll adds: route: customdomain.test/patch-target
    const response = await domainGet(request, '/patch-target');
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('No Route');
  });

  test('custom domain sitemap contains domain-specific routes', async ({ request }) => {
    const response = await domainGet(request, '/sitemap.xml');
    expect(response.status()).toBe(200);
    const body = await response.text();

    // Sitemap should include the custom domain routes
    expect(body).toContain('customdomain.test/');
    expect(body).toContain('customdomain.test/about');

    // Main-domain-only routes should not appear
    expect(body).not.toContain('/multi-alias');
  });

  test('main domain: link to custom-domain-only note uses full URL', async ({ request }) => {
    // domain-link-a.md (in multidomain/ subfolder) has route: customdomain.test/domain-link-a
    // and links to domain-link-b.md which has route: customdomain.test/b-on-domain (only).
    // When accessed on the MAIN domain via canonical permalink, [[domain-link-b]]
    // should use https://customdomain.test/b-on-domain, not the canonical permalink.
    //
    // Canonical permalink: /multidomain/domain_link_a (subfolder + hyphens → underscores).
    const response = await request.get(`${APP_URL}/multidomain/domain_link_a`);
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('href="https://customdomain.test/b-on-domain"');
  });

  test('domain-aware links: wikilink uses domain path, not permalink', async ({ request }) => {
    // domain-link-a.md has route: customdomain.test/domain-link-a
    // and links to domain-link-b.md which has route: customdomain.test/b-on-domain
    // When served on custom domain, the link should use /b-on-domain (domain path),
    // NOT /domain-link-b (permalink derived from filename).
    const response = await domainGet(request, '/domain-link-a');
    expect(response.status()).toBe(200);
    const body = await response.text();

    // The link to domain-link-b should use the domain path
    expect(body).toContain('href="/b-on-domain"');

    // The permalink-based path should NOT appear in the rendered link.
    // domain-link-b.md is in multidomain/ subfolder, so canonical permalink is
    // /multidomain/domain_link_b (hyphens normalized to underscores).
    expect(body).not.toContain('href="/multidomain/domain_link_b"');
  });
});
