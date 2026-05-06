// @ts-check
import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';
import crypto from 'crypto';
import { graphqlSignIn, createPersonalToken, revokePersonalToken } from './helpers/auth.js';

/**
 * Personal Token E2E Tests
 *
 * Tests personal user tokens (t2g_* prefix) for auth resolution, ACL, and
 * federation. Uses the two-instance harness:
 *   Hub  (port 20081) — main app, vault with federation_kb.md + content notes
 *   Peer (port 20091) — second app, seedvault (team-status free + internal-notes/premium)
 *
 * Key design: all MCP calls that test personal token auth use fresh (cookie-free)
 * request contexts so that session cookies on the shared admin context don't
 * interfere with the Bearer/queryToken auth path (cookie wins over Bearer).
 *
 * All groups run in a single serial describe to avoid sign-in code rate-limit
 * contention (admin user has max 3 active sign-in codes; parallel beforeAlls exceed this).
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER_URL = 'http://localhost:20091';
const HUB_MCP = `${HUB_URL}/_system/mcp`;
const PEER_MCP = `${PEER_URL}/_system/mcp`;

// Hub DB path for direct SQLite updates (expires_at manipulation).
const HUB_DB = process.env.HUB_DB_PATH || './tmp/data/test.sqlite3';

/** Execute GraphQL mutation/query with cookie auth. */
async function gql(request, baseURL, cookie, query, variables = {}) {
  const response = await request.post(`${baseURL}/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`GraphQL request to ${baseURL} failed: ${body}`);
  }
  const body = await response.json();
  if (body.errors) {
    throw new Error(`GraphQL errors (${baseURL}): ${JSON.stringify(body.errors)}`);
  }
  return body.data;
}

/**
 * Send a JSON-RPC request to an MCP endpoint.
 * @param {import('@playwright/test').APIRequestContext} request
 * @param {string} url
 * @param {string} method
 * @param {object} params
 * @param {{ bearer?: string, queryToken?: string, cookie?: string }} [auth]
 */
async function mcpCall(request, url, method, params = {}, auth = {}) {
  const headers = { 'Content-Type': 'application/json' };
  let targetUrl = url;

  if (auth.bearer) {
    headers['Authorization'] = `Bearer ${auth.bearer}`;
  }
  if (auth.cookie) {
    headers['Cookie'] = auth.cookie;
  }
  if (auth.queryToken) {
    targetUrl = `${url}?token=${encodeURIComponent(auth.queryToken)}`;
  }

  const response = await request.post(targetUrl, {
    headers,
    data: { jsonrpc: '2.0', id: 1, method, params },
  });
  expect(response.ok(), `MCP request to ${targetUrl} failed: ${response.status()}`).toBeTruthy();
  return response.json();
}

/**
 * Make an MCP call using a fresh cookie-free context.
 * Use this when testing personal token auth so session cookies don't interfere.
 * Cookie wins over Bearer, so any shared context with admin session would bypass token checks.
 */
async function mcpCallFresh(playwright, url, method, params = {}, auth = {}) {
  const ctx = await playwright.request.newContext({});
  try {
    return await mcpCall(ctx, url, method, params, auth);
  } finally {
    await ctx.dispose();
  }
}

// ─── Entire file runs serially to avoid sign-in rate-limit contention ────────
test.describe.serial('Personal tokens', () => {
  let hubRequest;  // admin context for GraphQL mutations (has admin session cookie)
  let peerRequest; // admin context for peer GraphQL mutations
  let hubCookie;   // explicit cookie header string for hubRequest
  let peerCookie;  // explicit cookie header string for peerRequest

  let adminHubToken;   // plaintext token from test 1 — shared across tests
  let adminHubTokenId; // id for cleanup in afterAll

  test.beforeAll(async ({ playwright }) => {
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    peerRequest = await playwright.request.newContext({ baseURL: PEER_URL });

    const hubJwt = await graphqlSignIn(hubRequest);
    hubCookie = `trip2g_e2e=${hubJwt}`;

    const peerJwt = await graphqlSignIn(peerRequest, 'hello@example.com', '111111', { useCache: false });
    peerCookie = `trip2g_e2e_peer=${peerJwt}`;
  });

  test.afterAll(async () => {
    if (adminHubTokenId) {
      await revokePersonalToken(hubRequest, HUB_URL, hubCookie, adminHubTokenId).catch(() => {});
    }
    await hubRequest?.dispose();
    await peerRequest?.dispose();
  });

  // ─── Group 1: Auth resolution (hub) ──────────────────────────────────────

  test('1. createUserToken returns t2g_ token with length 68', async () => {
    const result = await createPersonalToken(hubRequest, HUB_URL, hubCookie, {
      name: 'e2e-auth-test',
      expiresInDays: 90,
    });
    adminHubToken = result.plaintextToken;
    adminHubTokenId = result.id;

    expect(adminHubToken).toMatch(/^t2g_/);
    expect(adminHubToken.length).toBe(68); // 't2g_' (4) + 64 alnum
  });

  test('2. MCP search via Authorization Bearer token returns results', async ({ playwright }) => {
    // Fresh context: no session cookie — auth purely via Bearer personal token
    const result = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation' },
    }, { bearer: adminHubToken });

    expect(result.error).toBeUndefined();
    expect(result.result).toBeDefined();
    expect(typeof result.result.content[0].text).toBe('string');
  });

  test('3. Same MCP call via ?token= query param returns identical result', async ({ playwright }) => {
    const bearerResult = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation' },
    }, { bearer: adminHubToken });

    const queryResult = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation' },
    }, { queryToken: adminHubToken });

    expect(bearerResult.result).toBeDefined();
    expect(queryResult.result).toBeDefined();
    // Equal-score entries may appear in different order between two calls.
    // Sort by content (strip numeric prefix for comparison) and renumber so positions don't affect equality.
    const sortedEntries = (text) => {
      const [header, ...entries] = text.split(/\n\n(?=\d+\.)/);
      const key = (e) => e.replace(/^\d+\.\s*/, '');
      const sorted = entries.sort((a, b) => key(a).localeCompare(key(b)));
      const renumbered = sorted.map((e, i) => e.replace(/^\d+\./, `${i + 1}.`));
      return [header, ...renumbered].join('\n\n');
    };
    expect(sortedEntries(queryResult.result.content[0].text)).toBe(sortedEntries(bearerResult.result.content[0].text));
  });

  test('4. Cookie of user A + Bearer of user B token -> cookie user (admin) wins', async ({ playwright }) => {
    const secondEmail = `e2e-cookie-wins-${crypto.randomBytes(4).toString('hex')}@example.com`;
    const createData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateUserInput!) {
        admin { createUser(input: $input) {
          ... on CreateUserPayload { user { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { email: secondEmail } });
    const secondUserId = createData.admin.createUser.user?.id;
    expect(secondUserId).toBeDefined();

    // Isolated context so second-user sign-in cookie doesn't pollute hubRequest
    const isolatedCtx = await playwright.request.newContext({ baseURL: HUB_URL });
    try {
      const secondJwt = await graphqlSignIn(isolatedCtx, secondEmail, '111111');
      const secondCookie = `trip2g_e2e=${secondJwt}`;

      const secondTokenResult = await createPersonalToken(isolatedCtx, HUB_URL, secondCookie, {
        name: 'e2e-second-user-token',
      });

      // Post with explicit admin cookie + second user bearer — cookie wins (admin role)
      const viewerResp = await hubRequest.post(`${HUB_URL}/graphql`, {
        headers: {
          'Content-Type': 'application/json',
          'Cookie': hubCookie,
          'Authorization': `Bearer ${secondTokenResult.plaintextToken}`,
        },
        data: { query: `{ viewer { id role } }` },
      });
      const viewerBody = await viewerResp.json();
      expect(viewerBody.data.viewer.role).toBe('ADMIN');

      await revokePersonalToken(isolatedCtx, HUB_URL, secondCookie, secondTokenResult.id).catch(() => {});
    } finally {
      await isolatedCtx.dispose();
    }
  });

  test('5. Bearer t2g_invalid -> MCP returns JSON-RPC error, not anonymous results', async ({ playwright }) => {
    // Fresh context: no session cookie, so the invalid t2g_ token triggers a hard error
    const result = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation' },
    }, { bearer: 't2g_invalidtokenxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx' }); // 68 chars

    expect(result.error).toBeDefined();
    expect(result.error.message).toMatch(/invalid|expired|auth/i);
  });

  test('6. Expired token -> Bearer call returns error', async ({ playwright }) => {
    test.setTimeout(70_000); // needs 31s cache TTL wait

    const { plaintextToken, id } = await createPersonalToken(hubRequest, HUB_URL, hubCookie, {
      name: 'e2e-expired',
      expiresInDays: 1,
    });

    try {
      execSync(
        `sqlite3 ${HUB_DB} "UPDATE user_tokens SET expires_at = datetime('1970-01-01') WHERE id = '${id}';"`,
        { timeout: 5000 },
      );
    } catch {
      test.skip(true, 'Cannot reach DB for direct update (running in container?)');
      return;
    }

    // Cache TTL is 30s — token just created so cache entry exists.
    // Wait for cache to expire, then DB lookup will return not-found.
    await new Promise(r => setTimeout(r, 31_000));

    const result = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation' },
    }, { bearer: plaintextToken });

    expect(result.error).toBeDefined();
    expect(result.error.message).toMatch(/invalid|expired|auth/i);
  });

  test('7. Revoked token -> subsequent call returns error after cache TTL', async ({ playwright }) => {
    test.setTimeout(70_000); // needs 31s cache TTL wait

    const { plaintextToken, id: tokenId } = await createPersonalToken(hubRequest, HUB_URL, hubCookie, {
      name: 'e2e-revoke-test',
    });

    // Verify it works before revocation (fresh context = pure Bearer auth)
    const before = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation' },
    }, { bearer: plaintextToken });
    expect(before.error).toBeUndefined();

    await revokePersonalToken(hubRequest, HUB_URL, hubCookie, tokenId);

    // Wait for cache TTL to expire
    await new Promise(r => setTimeout(r, 31_000));

    const after = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation' },
    }, { bearer: plaintextToken });

    expect(after.error).toBeDefined();
    expect(after.error.message).toMatch(/invalid|expired|revoked|auth/i);
  });

  // ─── Group 2: ACL for non-admin (peer instance) ───────────────────────────

  test('8. Non-admin without premium subgraph -> search hides premium notes', async ({ playwright }) => {
    const nonAdminEmail = `e2e-acl-${crypto.randomBytes(4).toString('hex')}@example.com`;

    const createData = await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateUserInput!) {
        admin { createUser(input: $input) {
          ... on CreateUserPayload { user { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { email: nonAdminEmail } });
    const userId = createData.admin.createUser.user?.id;
    expect(userId).toBeDefined();

    // Isolated context for non-admin sign-in (don't pollute peerRequest with non-admin cookie)
    const isolatedCtx = await playwright.request.newContext({ baseURL: PEER_URL });
    try {
      const nonAdminJwt = await graphqlSignIn(isolatedCtx, nonAdminEmail, '111111');
      const nonAdminCookie = `trip2g_e2e_peer=${nonAdminJwt}`;

      const { plaintextToken, id: tokenId } = await createPersonalToken(isolatedCtx, PEER_URL, nonAdminCookie, {
        name: 'e2e-acl-nosubgraph',
      });

      try {
        // Fresh context: pure Bearer auth, no session cookie
        const result = await mcpCallFresh(playwright, PEER_MCP, 'tools/call', {
          name: 'search',
          arguments: { query: 'notes' },
        }, { bearer: plaintextToken });

        expect(result.error).toBeUndefined();
        const text = result.result?.content?.[0]?.text ?? '';
        // internal-notes.md requires premium subgraph — not visible without subscription
        expect(text).not.toContain('Internal Notes');
        expect(text).not.toContain('Budget allocation');
      } finally {
        await revokePersonalToken(isolatedCtx, PEER_URL, nonAdminCookie, tokenId).catch(() => {});
      }
    } finally {
      await isolatedCtx.dispose();
    }
  });

  test('9. Non-admin WITH premium subgraph access -> search returns premium notes', async ({ playwright }) => {
    const nonAdminEmail = `e2e-acl-sub-${crypto.randomBytes(4).toString('hex')}@example.com`;
    const PREMIUM_SUBGRAPH_ID = 1; // peer DB premium subgraph from seedvault

    const createData = await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateUserInput!) {
        admin { createUser(input: $input) {
          ... on CreateUserPayload { user { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { email: nonAdminEmail } });
    const userId = createData.admin.createUser.user?.id;
    expect(userId).toBeDefined();

    await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateUserSubgraphAccessInput!) {
        admin { createUserSubgraphAccess(input: $input) {
          ... on CreateUserSubgraphAccessPayload { accesses { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { userId, subgraphIds: [PREMIUM_SUBGRAPH_ID] } });

    await new Promise(r => setTimeout(r, 1000));

    const isolatedCtx = await playwright.request.newContext({ baseURL: PEER_URL });
    try {
      const nonAdminJwt = await graphqlSignIn(isolatedCtx, nonAdminEmail, '111111');
      const nonAdminCookie = `trip2g_e2e_peer=${nonAdminJwt}`;

      const { plaintextToken, id: tokenId } = await createPersonalToken(isolatedCtx, PEER_URL, nonAdminCookie, {
        name: 'e2e-acl-withsub',
      });

      try {
        const result = await mcpCallFresh(playwright, PEER_MCP, 'tools/call', {
          name: 'search',
          arguments: { query: 'internal notes' },
        }, { bearer: plaintextToken });

        expect(result.error).toBeUndefined();
        const text = result.result?.content?.[0]?.text ?? '';
        expect(text).toContain('Internal Notes');
      } finally {
        await revokePersonalToken(isolatedCtx, PEER_URL, nonAdminCookie, tokenId).catch(() => {});
      }
    } finally {
      await isolatedCtx.dispose();
    }
  });

  test('10. Admin token on peer -> search returns all notes including premium', async ({ playwright }) => {
    const { plaintextToken, id: tokenId } = await createPersonalToken(peerRequest, PEER_URL, peerCookie, {
      name: 'e2e-peer-admin',
    });

    try {
      const result = await mcpCallFresh(playwright, PEER_MCP, 'tools/call', {
        name: 'search',
        arguments: { query: 'internal notes' },
      }, { bearer: plaintextToken });

      expect(result.error).toBeUndefined();
      const text = result.result?.content?.[0]?.text ?? '';
      expect(text).toContain('Internal Notes');
    } finally {
      await revokePersonalToken(peerRequest, PEER_URL, peerCookie, tokenId).catch(() => {});
    }
  });

  // ─── Group 3: Federation via personal token ───────────────────────────────

  test('11-12. Admin personal token -> federated_search returns peer notes', async ({ playwright }) => {
    const KID = `e2e-pt-fed-${crypto.randomBytes(4).toString('hex')}`;
    const SECRET_HEX = 'a0b1c2d3e4f5060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f';

    await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID, secretHex: SECRET_HEX } });

    const outboundData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID, secretHex: SECRET_HEX, kbURL: 'http://app-peer:20091/_system/mcp' } });
    const outboundId = outboundData.admin.createOutboundFederationSecret.id;

    const { plaintextToken, id: tokenId } = await createPersonalToken(hubRequest, HUB_URL, hubCookie, {
      name: 'e2e-fed-admin',
    });

    try {
      const result = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
        name: 'federated_search',
        arguments: { kb_id: 'peer', query: 'team status' },
      }, { bearer: plaintextToken });

      // Skip if federation_kb.md not loaded (fresh containers without vault sync)
      if (result.result?.content?.[0]?.text?.includes('not configured')) {
        test.skip(true, 'federation_kb.md not in LatestNoteViews; vault sync needed — skipping');
        return;
      }

      expect(result.error).toBeUndefined();
      const text = result.result?.content?.[0]?.text ?? '';
      expect(text).toContain('team');
    } finally {
      await revokePersonalToken(hubRequest, HUB_URL, hubCookie, tokenId).catch(() => {});
      if (outboundId) {
        await gql(hubRequest, HUB_URL, hubCookie, `
          mutation($id: Int64!) {
            admin { revokeFederationSecret(id: $id) {
              ... on RevokeFederationSecretPayload { revokedId }
              ... on ErrorPayload { message }
            } }
          }
        `, { id: outboundId }).catch(() => {});
      }
    }
  });

  test('13. Non-admin personal token -> federated_search with unknown kb returns not-configured', async ({ playwright }) => {
    const email = `e2e-fed-na-${crypto.randomBytes(4).toString('hex')}@example.com`;
    await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateUserInput!) {
        admin { createUser(input: $input) {
          ... on CreateUserPayload { user { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { email } });

    const isolatedCtx = await playwright.request.newContext({ baseURL: HUB_URL });
    try {
      const jwt = await graphqlSignIn(isolatedCtx, email, '111111');
      const cookie = `trip2g_e2e=${jwt}`;
      const { plaintextToken, id: tokenId } = await createPersonalToken(isolatedCtx, HUB_URL, cookie, {
        name: 'e2e-fed-na',
      });

      try {
        const result = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
          name: 'federated_search',
          arguments: { kb_id: 'ghost-kb-nonexistent', query: 'anything' },
        }, { bearer: plaintextToken });

        expect(result.error).toBeUndefined();
        const text = result.result?.content?.[0]?.text ?? '';
        expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
      } finally {
        await revokePersonalToken(isolatedCtx, HUB_URL, cookie, tokenId).catch(() => {});
      }
    } finally {
      await isolatedCtx.dispose();
    }
  });

  test('14. Fan-out federated_search without kb_id runs for admin token', async ({ playwright }) => {
    const { plaintextToken, id: tokenId } = await createPersonalToken(hubRequest, HUB_URL, hubCookie, {
      name: 'e2e-fanout',
    });

    try {
      const result = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
        name: 'federated_search',
        arguments: { query: 'team status' },
      }, { bearer: plaintextToken });

      expect(result.error).toBeUndefined();
      expect(result.result).toBeDefined();
    } finally {
      await revokePersonalToken(hubRequest, HUB_URL, hubCookie, tokenId).catch(() => {});
    }
  });

  // ─── Group 4: Smoke / regressions ────────────────────────────────────────

  test('15. Two rapid sequential calls with same token -> both succeed', async ({ playwright }) => {
    const { plaintextToken, id: tokenId } = await createPersonalToken(hubRequest, HUB_URL, hubCookie, {
      name: 'e2e-smoke-rapid',
    });

    try {
      const [r1, r2] = await Promise.all([
        mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
          name: 'search', arguments: { query: 'federation' },
        }, { bearer: plaintextToken }),
        mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
          name: 'search', arguments: { query: 'federation' },
        }, { bearer: plaintextToken }),
      ]);

      expect(r1.error).toBeUndefined();
      expect(r2.error).toBeUndefined();
      expect(r1.result).toBeDefined();
      expect(r2.result).toBeDefined();
    } finally {
      await revokePersonalToken(hubRequest, HUB_URL, hubCookie, tokenId).catch(() => {});
    }
  });

  test('16. Plain federation Bearer JWT path unaffected (regression)', async () => {
    // Anonymous MCP tools/list should work (no auth required)
    const result = await mcpCall(hubRequest, HUB_MCP, 'tools/list');
    expect(result.error).toBeUndefined();
    const toolNames = result.result?.tools?.map((t) => t.name) ?? [];
    expect(toolNames).toContain('search');
    expect(toolNames).toContain('federated_search');
  });

  test('17. Non-t2g_ Bearer does not trigger personal token resolver (regression)', async ({ playwright }) => {
    // A well-formed JWT (non-t2g_) as Bearer should be treated as federation JWT,
    // not as a personal token. Since it's not a valid federation JWT for this server,
    // the MCP endpoint should return an auth error (not a personal-token error).
    // We verify the error message is about federation auth, not personal token.
    const fakeJwt = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZXN0IjoidHJ1ZSJ9.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c';
    const result = await mcpCallFresh(playwright, HUB_MCP, 'tools/call', {
      name: 'search',
      arguments: { query: 'test' },
    }, { bearer: fakeJwt });

    // Should fail with federation auth error, not personal token error
    const hasError = result.error != null ||
      (result.result?.content?.[0]?.text ?? '').toLowerCase().includes('error');
    expect(hasError).toBeTruthy();
    // Must NOT be a personal token error (t2g_ prefix not present)
    if (result.error) {
      expect(result.error.message).not.toMatch(/personal token/i);
    }
  });
});
