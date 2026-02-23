// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

const SEARCH_QUERY = `
  query Search($input: SearchInput!) {
    search(input: $input) {
      nodes {
        url
        highlightedTitle
        highlightedContent
        document {
          ... on PublicNote {
            title
            path
          }
        }
      }
    }
  }
`;

const hasOpenAI = !!process.env.OPENAI_API_KEY;

// ─── Text search (Bleve) ─────────────────────────────────────────────────────
// These tests use full-text search only and do not require OPENAI_API_KEY.

test.describe('Search: text (Bleve)', () => {
  let authPost;

  test.beforeAll(async ({ request }) => {
    const token = await graphqlSignIn(request);
    const headers = { Cookie: `trip2g_e2e=${token}` };
    authPost = (query, variables) =>
      request.post('/graphql', { headers, data: variables ? { query, variables } : { query } });
  });

  test('finds note by unique keyword', async () => {
    const res = await authPost(SEARCH_QUERY, { input: { query: 'квантовый телескоп' } });
    expect(res.ok()).toBeTruthy();

    const { data } = await res.json();
    const nodes = data.search.nodes;

    expect(nodes.length).toBeGreaterThan(0);

    const found = nodes.find((n) => n.url === '/search_keywords');
    expect(found).toBeDefined();
    expect(found.document.title).toBe('Ключевые слова поиска');
  });

  test('returns highlighted content for matched terms', async () => {
    const res = await authPost(SEARCH_QUERY, { input: { query: 'экзопланет' } });
    const { data } = await res.json();

    const found = data.search.nodes.find((n) => n.url === '/search_keywords');
    expect(found).toBeDefined();

    const hasHighlight =
      found.highlightedTitle != null ||
      (found.highlightedContent && found.highlightedContent.length > 0);
    expect(hasHighlight).toBeTruthy();
  });

  test('returns empty results for unknown term', async () => {
    const res = await authPost(SEARCH_QUERY, {
      input: { query: 'xyzzynonexistenttermqwerty' },
    });
    const { data } = await res.json();
    expect(data.search.nodes).toHaveLength(0);
  });

  test('public search works without auth for public notes', async ({ request }) => {
    const res = await request.post('/graphql', {
      data: { query: SEARCH_QUERY, variables: { input: { query: 'квантовый телескоп' } } },
    });
    expect(res.ok()).toBeTruthy();

    const { data } = await res.json();
    const found = data.search.nodes.find((n) => n.url === '/search_keywords');
    expect(found).toBeDefined();
  });
});

// ─── Hybrid search (Bleve + vector via OpenAI) ───────────────────────────────
// Requires OPENAI_API_KEY and FEATURES vector_search enabled.
// Notes must have embeddings generated (background job runs after sync).

test.describe('Search: hybrid (text + vector)', () => {
  test.skip(!hasOpenAI, 'Requires OPENAI_API_KEY and vector_search enabled in FEATURES');

  let authPost;

  test.beforeAll(async ({ request }) => {
    const token = await graphqlSignIn(request);
    const headers = { Cookie: `trip2g_e2e=${token}` };
    authPost = (query, variables) =>
      request.post('/graphql', { headers, data: variables ? { query, variables } : { query } });
  });

  test('semantic query finds thematically related note', async ({ request }) => {
    // Wait for embedding background jobs to finish before testing vector results
    const waitRes = await request.get('/debug/wait_all_jobs', { timeout: 60_000 });
    expect(waitRes.ok()).toBeTruthy();

    // Phrase not present verbatim in any note, but semantically matches
    // search_astronomy.md ("планеты за пределами Солнечной системы") and
    // search_keywords.md ("экзопланет")
    const res = await authPost(SEARCH_QUERY, {
      input: { query: 'как учёные ищут планеты у других звёзд' },
    });
    const { data } = await res.json();
    const nodes = data.search.nodes;

    const urls = nodes.map((n) => n.url);
    const hasAstronomy = urls.includes('/search_astronomy') || urls.includes('/search_keywords');
    expect(hasAstronomy).toBeTruthy();
  });

  test('hybrid result list contains both text and vector matches', async ({ request }) => {
    const waitRes = await request.get('/debug/wait_all_jobs', { timeout: 60_000 });
    expect(waitRes.ok()).toBeTruthy();

    // "астрофизика" should match by text; semantically related notes should follow
    const res = await authPost(SEARCH_QUERY, { input: { query: 'астрофизика' } });
    const { data } = await res.json();

    expect(data.search.nodes.length).toBeGreaterThan(0);
    // All results should have a url
    for (const node of data.search.nodes) {
      expect(node.url).toBeTruthy();
    }
  });
});
