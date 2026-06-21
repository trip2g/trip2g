// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn, createPersonalToken, revokePersonalToken } from './helpers/auth.js';

/**
 * Federated GraphQL Request E2E Tests
 *
 * Covers the `federated_graphql_request` MCP tool:
 *   hub forwards a read-only GraphQL query to a federation peer;
 *   the peer executes it scoped to the inbound identity's AllowedSubgraphs,
 *   NEVER as admin; query-only + strict root-field allowlist
 *   (note, search, similarNotes, viewer — NOT notePaths/resolveWikilinks/admin).
 *   Off by default behind MCP_FEDERATED_GRAPHQL=true.
 *
 * Three-principal setup mirrors federation-acl.spec.js exactly:
 *   peer  (port 20091) — open, free: team-status.md
 *   peer2 (port 20093) — semi-private, subgraphs: federation-test KB-note
 *   peer3 (port 20095) — admin-only, no free/subgraphs KB-note
 *
 * Principals:
 *   anonymous  — no auth header
 *   scoped     — non-admin user with federation-test subgraph personal token
 *   admin      — admin personal token
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER2_URL = 'http://localhost:20093';
const PEER3_URL = 'http://localhost:20095';
const HUB_MCP = `${HUB_URL}/_system/mcp`;

// Deterministic secrets — different from federation-acl.spec.js to avoid kid collisions.
const SECRET_HEX_P2 = 'd3e3f3030405060708090a0b0c0d0e0f101112131415161718191ad3e3f30304';
const SECRET_HEX_P3 = 'e4f4040506070809000102030405060708090a0b0c0d0e0f1011e4f404050607';
const KID_P2 = `e2e-gql-p2-${crypto.randomBytes(4).toString('hex')}`;
const KID_P3 = `e2e-gql-p3-${crypto.randomBytes(4).toString('hex')}`;

async function gql(request, baseURL, cookie, query, variables = {}) {
  const response = await request.post(`${baseURL}/_system/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`GraphQL ${baseURL} failed: ${body}`);
  }
  const body = await response.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

// MCP call in a fresh cookie-free context (cookie wins over Bearer — fresh context required).
async function mcpFresh(playwright, method, params = {}, token = null) {
  const ctx = await playwright.request.newContext({});
  try {
    const headers = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    const response = await ctx.post(HUB_MCP, {
      headers,
      data: { jsonrpc: '2.0', id: 1, method, params },
    });
    expect(response.ok(), `MCP ${method} status ${response.status()}`).toBeTruthy();
    return response.json();
  } finally {
    await ctx.dispose();
  }
}

test.describe.serial('Federated GraphQL Request', () => {
  let hubRequest;
  let peer2Request;
  let peer3Request;
  let hubCookie;
  let peer2Cookie;
  let peer3Cookie;

  let adminToken;
  let adminTokenId;
  let scopedToken;
  let scopedTokenId;
  let scopedCookie;
  let scopedCtx;

  let outboundPeer2Id;
  let outboundPeer3Id;

  test.beforeAll(async ({ playwright }) => {
    // Sign into all three admins.
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    const hubJwt = await graphqlSignIn(hubRequest);
    hubCookie = `trip2g_e2e=${hubJwt}`;

    peer2Request = await playwright.request.newContext({ baseURL: PEER2_URL });
    const peer2Jwt = await graphqlSignIn(peer2Request, 'hello@example.com', '111111', { useCache: false });
    peer2Cookie = `trip2g_e2e_peer2=${peer2Jwt}`;

    peer3Request = await playwright.request.newContext({ baseURL: PEER3_URL });
    const peer3Jwt = await graphqlSignIn(peer3Request, 'hello@example.com', '111111', { useCache: false });
    peer3Cookie = `trip2g_e2e_peer3=${peer3Jwt}`;

    // Create inbound secrets on peer2 and peer3.
    await gql(peer2Request, PEER2_URL, peer2Cookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_P2, secretHex: SECRET_HEX_P2 } });

    await gql(peer3Request, PEER3_URL, peer3Cookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_P3, secretHex: SECRET_HEX_P3 } });

    // Create outbound secrets on hub pointing to peer2 and peer3.
    const p2Out = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_P2, secretHex: SECRET_HEX_P2, kbURL: 'http://app-peer2:20093/_system/mcp' } });
    outboundPeer2Id = p2Out.admin.createOutboundFederationSecret.id;
    expect(outboundPeer2Id, 'outbound peer2 secret must be created').toBeTruthy();

    const p3Out = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_P3, secretHex: SECRET_HEX_P3, kbURL: 'http://app-peer3:20095/_system/mcp' } });
    outboundPeer3Id = p3Out.admin.createOutboundFederationSecret.id;
    expect(outboundPeer3Id, 'outbound peer3 secret must be created').toBeTruthy();

    // Find federation-test subgraph ID (auto-created when hub loaded federation_kb_semi.md).
    const sgData = await gql(hubRequest, HUB_URL, hubCookie, `
      query { admin { allSubgraphs { nodes { id name } } } }
    `);
    const sg = sgData.admin.allSubgraphs.nodes.find(n => n.name === 'federation-test');
    expect(sg, 'federation-test subgraph must exist — check that hub vault sync ran').toBeTruthy();
    const federationTestSubgraphId = sg.id;

    // Create non-admin user with federation-test subgraph access.
    const scopedEmail = `e2e-fed-gql-${crypto.randomBytes(4).toString('hex')}@example.com`;
    const createUserData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateUserInput!) {
        admin { createUser(input: $input) {
          ... on CreateUserPayload { user { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { email: scopedEmail } });
    const scopedUserId = createUserData.admin.createUser.user?.id;
    expect(scopedUserId, 'scoped user must be created').toBeTruthy();

    await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateUserSubgraphAccessInput!) {
        admin { createUserSubgraphAccess(input: $input) {
          ... on CreateUserSubgraphAccessPayload { accesses { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { userId: scopedUserId, subgraphIds: [federationTestSubgraphId] } });

    // Allow subgraph access to propagate.
    await new Promise(r => setTimeout(r, 1000));

    // Sign in scoped user (isolated context — must not share admin session).
    scopedCtx = await playwright.request.newContext({ baseURL: HUB_URL });
    const scopedJwt = await graphqlSignIn(scopedCtx, scopedEmail, '111111');
    scopedCookie = `trip2g_e2e=${scopedJwt}`;
    const scopedResult = await createPersonalToken(scopedCtx, HUB_URL, scopedCookie, {
      name: 'e2e-fed-gql-scoped',
    });
    scopedToken = scopedResult.plaintextToken;
    scopedTokenId = scopedResult.id;

    // Admin personal token for authenticated MCP calls.
    const adminResult = await createPersonalToken(hubRequest, HUB_URL, hubCookie, {
      name: 'e2e-fed-gql-admin',
    });
    adminToken = adminResult.plaintextToken;
    adminTokenId = adminResult.id;
  });

  test.afterAll(async () => {
    if (adminTokenId) {
      await revokePersonalToken(hubRequest, HUB_URL, hubCookie, adminTokenId).catch(() => {});
    }
    if (scopedTokenId && scopedCtx) {
      await revokePersonalToken(scopedCtx, HUB_URL, scopedCookie, scopedTokenId).catch(() => {});
    }
    if (outboundPeer2Id) {
      await gql(hubRequest, HUB_URL, hubCookie, `
        mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
          ... on RevokeFederationSecretPayload { revokedId }
          ... on ErrorPayload { message }
        } } }
      `, { id: outboundPeer2Id }).catch(() => {});
    }
    if (outboundPeer3Id) {
      await gql(hubRequest, HUB_URL, hubCookie, `
        mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
          ... on RevokeFederationSecretPayload { revokedId }
          ... on ErrorPayload { message }
        } } }
      `, { id: outboundPeer3Id }).catch(() => {});
    }
    await scopedCtx?.dispose();
    await hubRequest?.dispose();
    await peer2Request?.dispose();
    await peer3Request?.dispose();
  });

  // ─── Group 1: tool discovery ────────────────────────────────────────────────

  test('1.1 tools/list includes federated_graphql_request when flag is on', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/list');
    expect(result.error).toBeUndefined();
    const toolNames = (result.result?.tools ?? []).map(t => t.name);
    expect(toolNames).toContain('federated_graphql_request');
  });

  // ─── Group 2: functional forwarding ────────────────────────────────────────

  test('2.1 valid search query forwarded to open peer returns free note content', async ({ playwright }) => {
    // peer2 has peer2-note.md (free: true) with "Semi-private peer2 content".
    // Use admin token so the outbound secret is used and the scoped response is irrelevant here.
    // Data lands in structuredContent (single-copy design); content[0].text is the stub "structured result".
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: '{ search(input: { query: "federation ACL testing" }) { nodes { highlightedTitle highlightedContent url } } }',
      },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const nodes = result.result?.structuredContent?.data?.search?.nodes ?? [];
    expect(nodes.length).toBeGreaterThan(0);
    // peer2-note.md has title "Peer2 Note"
    const titles = nodes.map(n => n.highlightedTitle ?? '');
    expect(titles.join(' ').toLowerCase()).toContain('peer2');
  });

  // ─── Group 3: scope parity (security) ──────────────────────────────────────

  test('3.1 anonymous principal: search on peer2 returns not-configured (no grant)', async ({ playwright }) => {
    // Anonymous caller has no outbound secret configured for peer2 visible to them.
    // federated_graphql_request must behave like federated_search for anonymous:
    // existence must not leak — should return "not configured".
    // For "not configured" responses, the message lands in content[0].text (not structuredContent).
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: '{ search(input: { query: "federation knowledge base" }) { nodes { highlightedTitle url } } }',
      },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
    expect(text.toLowerCase()).not.toMatch(/access.?denied|unauthorized|forbidden/);
  });

  test('3.2 anonymous principal: no semi-private content in search results', async ({ playwright }) => {
    // Even if routed to a peer, anonymous must never see semi-private content.
    // For anonymous, peer3 returns "not configured" — no peer3 data in either channel.
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer3',
        query: '{ search(input: { query: "federation knowledge base" }) { nodes { highlightedTitle url } } }',
      },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    // peer3 is admin-only — anonymous must see not-configured, not peer3 content.
    expect(text).not.toContain('Admin-only peer3 content');
    expect(text).not.toContain('peer3-note');
    // Also ensure structuredContent carries no peer3 node data
    const nodes = result.result?.structuredContent?.data?.search?.nodes ?? [];
    expect(nodes).toHaveLength(0);
  });

  test('3.3 scoped token: search on granted peer2 returns semi-private content', async ({ playwright }) => {
    // Scoped user has federation-test subgraph → should see peer2 content.
    // Data lands in structuredContent; content[0].text is the stub "structured result".
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: '{ search(input: { query: "federation ACL testing" }) { nodes { highlightedTitle highlightedContent url } } }',
      },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const nodes = result.result?.structuredContent?.data?.search?.nodes ?? [];
    expect(nodes.length).toBeGreaterThan(0);
    // peer2-note.md has title "Peer2 Note" — visible to scoped user via federation-test subgraph.
    const titles = nodes.map(n => n.highlightedTitle ?? '');
    expect(titles.join(' ').toLowerCase()).toContain('peer2');
  });

  test('3.4 scoped token: search on admin-only peer3 returns not-configured', async ({ playwright }) => {
    // Scoped user does not have access to peer3 (admin-only).
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer3',
        query: '{ search(input: { query: "federation ACL testing" }) { nodes { highlightedTitle url } } }',
      },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
    expect(text).not.toContain('Admin-only peer3 content');
  });

  test('3.5 admin token: search on peer3 returns admin-only content', async ({ playwright }) => {
    // Admin sees all — peer3 content must be present.
    // Data lands in structuredContent; content[0].text is the stub "structured result".
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer3',
        query: '{ search(input: { query: "federation ACL testing" }) { nodes { highlightedTitle highlightedContent url } } }',
      },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const nodes = result.result?.structuredContent?.data?.search?.nodes ?? [];
    expect(nodes.length).toBeGreaterThan(0);
    // peer3-note.md has title "Peer3 Note" — admin-only content visible to admin.
    const titles = nodes.map(n => n.highlightedTitle ?? '');
    expect(titles.join(' ').toLowerCase()).toContain('peer3');
  });

  test('3.6 scoped token does not see admin-only peer3 content that admin sees', async ({ playwright }) => {
    // Parity assertion: same query, different principals.
    // Data lands in structuredContent; content[0].text is the stub "structured result".
    const adminResult = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer3',
        query: '{ search(input: { query: "federation ACL testing" }) { nodes { highlightedTitle highlightedContent url } } }',
      },
    }, adminToken);
    const adminNodes = adminResult.result?.structuredContent?.data?.search?.nodes ?? [];
    expect(adminNodes.length).toBeGreaterThan(0);
    const adminTitles = adminNodes.map(n => n.highlightedTitle ?? '');
    // Admin must see peer3 content
    expect(adminTitles.join(' ').toLowerCase()).toContain('peer3');

    const scopedResult = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer3',
        query: '{ search(input: { query: "federation ACL testing" }) { nodes { highlightedTitle highlightedContent url } } }',
      },
    }, scopedToken);
    // Scoped user has no access to peer3 — must get "not configured" response with no node data.
    // The "not configured" message may mention "peer3" as the kb_id — that's fine.
    // What must NOT appear is peer3 note content (title "Peer3 Note").
    const scopedNodes = scopedResult.result?.structuredContent?.data?.search?.nodes ?? [];
    expect(scopedNodes).toHaveLength(0);
    const scopedTitles = scopedNodes.map(n => n.highlightedTitle ?? '');
    expect(scopedTitles).not.toContain('Peer3 Note');
    // Admin content sentinel must not leak
    const scopedText = scopedResult.result?.content?.[0]?.text ?? '';
    expect(scopedText).not.toContain('Admin-only peer3 content');
  });

  // ─── Group 4: mutation rejected ────────────────────────────────────────────

  test('4.1 mutation is rejected before forwarding', async ({ playwright }) => {
    // Any mutation operation must be rejected at the hub — never forwarded to the peer.
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: 'mutation { admin { __typename } }',
      },
    }, adminToken);
    // Must be a JSON-RPC error (InvalidParams), not a successful tool call.
    expect(result.error).toBeDefined();
    expect(result.error.code).toBe(-32602); // ErrCodeInvalidParams
    expect(result.error.message.toLowerCase()).toMatch(/query rejected.*only quer/);
  });

  test('4.2 mutation with createNote field is rejected', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: 'mutation CreateSomething { admin { __typename } }',
      },
    }, adminToken);
    expect(result.error).toBeDefined();
    expect(result.error.code).toBe(-32602);
    expect(result.error.message.toLowerCase()).toContain('query rejected');
  });

  // ─── Group 5: root-field allowlist ─────────────────────────────────────────

  test('5.1 notePaths root field is rejected (not in federated allowlist)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: '{ notePaths }',
      },
    }, adminToken);
    expect(result.error).toBeDefined();
    expect(result.error.code).toBe(-32602);
    expect(result.error.message).toContain('"notePaths" is not allowed');
  });

  test('5.2 admin root field is rejected (not in federated allowlist)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: '{ admin { __typename } }',
      },
    }, adminToken);
    expect(result.error).toBeDefined();
    expect(result.error.code).toBe(-32602);
    expect(result.error.message).toContain('"admin" is not allowed');
  });

  test('5.3 resolveWikilinks root field is rejected (not in federated allowlist)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: '{ resolveWikilinks(filter: { paths: ["x"] }) { path href } }',
      },
    }, adminToken);
    expect(result.error).toBeDefined();
    expect(result.error.code).toBe(-32602);
    expect(result.error.message).toContain('"resolveWikilinks" is not allowed');
  });

  // ─── Group 6: introspection blocked ────────────────────────────────────────

  test('6.1 __schema introspection is rejected (not in allowlist)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: '{ __schema { queryType { name } } }',
      },
    }, adminToken);
    expect(result.error).toBeDefined();
    expect(result.error.code).toBe(-32602);
    expect(result.error.message).toContain('"__schema" is not allowed');
  });

  test('6.2 __type introspection is rejected (not in allowlist)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_graphql_request',
      arguments: {
        kb_id: 'peer2',
        query: '{ __type(name: "Query") { name } }',
      },
    }, adminToken);
    expect(result.error).toBeDefined();
    expect(result.error.code).toBe(-32602);
    expect(result.error.message).toContain('"__type" is not allowed');
  });
});
