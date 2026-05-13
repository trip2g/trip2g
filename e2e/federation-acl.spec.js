// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn, createPersonalToken, revokePersonalToken } from './helpers/auth.js';

/**
 * Federation ACL E2E Tests
 *
 * Three-tier KB-note access control:
 *   peer  (port 20091) — open, free: true KB-note
 *   peer2 (port 20093) — semi-private, subgraphs: federation-test KB-note
 *   peer3 (port 20095) — admin-only, no free/subgraphs KB-note
 *
 * Three principals:
 *   anonymous  — no auth header
 *   scoped     — non-admin user with federation-test subgraph personal token
 *   admin      — admin personal token
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER2_URL = 'http://localhost:20093';
const PEER3_URL = 'http://localhost:20095';
const HUB_MCP = `${HUB_URL}/_system/mcp`;

const SECRET_HEX_P2 = 'b1c1d1e1f1020304050607080900a1b1c1d1e1f1020304050607080900a1b100';
const SECRET_HEX_P3 = 'c2d2e2f2030405060708090a0b0c0d0e0f1011121314151617181900c2d2e2f2';
const KID_P2 = `e2e-acl-p2-${crypto.randomBytes(4).toString('hex')}`;
const KID_P3 = `e2e-acl-p3-${crypto.randomBytes(4).toString('hex')}`;

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

// MCP call in a fresh cookie-free context.
// Cookie wins over Bearer — fresh context required for personal token tests.
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

test.describe.serial('Federation ACL', () => {
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
    // Sign into all three admins
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    const hubJwt = await graphqlSignIn(hubRequest);
    hubCookie = `trip2g_e2e=${hubJwt}`;

    peer2Request = await playwright.request.newContext({ baseURL: PEER2_URL });
    const peer2Jwt = await graphqlSignIn(peer2Request, 'hello@example.com', '111111', { useCache: false });
    peer2Cookie = `trip2g_e2e_peer2=${peer2Jwt}`;

    peer3Request = await playwright.request.newContext({ baseURL: PEER3_URL });
    const peer3Jwt = await graphqlSignIn(peer3Request, 'hello@example.com', '111111', { useCache: false });
    peer3Cookie = `trip2g_e2e_peer3=${peer3Jwt}`;

    // Create inbound secrets on peer2 and peer3
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

    // Create outbound secrets on hub
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

    // Find federation-test subgraph ID (auto-created when hub loaded federation_kb_semi.md)
    const sgData = await gql(hubRequest, HUB_URL, hubCookie, `
      query { admin { allSubgraphs { nodes { id name } } } }
    `);
    const sg = sgData.admin.allSubgraphs.nodes.find(n => n.name === 'federation-test');
    expect(sg, 'federation-test subgraph must exist — check that hub vault sync ran').toBeTruthy();
    const federationTestSubgraphId = sg.id;

    // Create non-admin user with federation-test subgraph access
    const scopedEmail = `e2e-fed-acl-${crypto.randomBytes(4).toString('hex')}@example.com`;
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

    // Allow subgraph access to propagate
    await new Promise(r => setTimeout(r, 1000));

    // Sign in scoped user (isolated context — must not share admin session)
    scopedCtx = await playwright.request.newContext({ baseURL: HUB_URL });
    const scopedJwt = await graphqlSignIn(scopedCtx, scopedEmail, '111111');
    scopedCookie = `trip2g_e2e=${scopedJwt}`;
    const scopedResult = await createPersonalToken(scopedCtx, HUB_URL, scopedCookie, {
      name: 'e2e-fed-acl-scoped',
    });
    scopedToken = scopedResult.plaintextToken;
    scopedTokenId = scopedResult.id;

    // Admin personal token for authenticated MCP calls
    const adminResult = await createPersonalToken(hubRequest, HUB_URL, hubCookie, {
      name: 'e2e-fed-acl-admin',
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
    // outboundPeer2Id may already be revoked by test 4.1 — ignore error
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

  // ─── Group 1: search visibility ────────────────────────────────────────────

  test('1.1 anonymous search: open KB visible, semi-private and admin-only absent', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('federation_kb');
    expect(text).not.toContain('federation_kb_semi');
    expect(text).not.toContain('federation_kb_private');
  });

  test('1.2 scoped user search: open + semi-private visible, admin-only absent', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('federation_kb');
    expect(text).toContain('federation_kb_semi');
    expect(text).not.toContain('federation_kb_private');
  });

  test('1.3 admin search: all three KB-notes visible', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('federation_kb');
    expect(text).toContain('federation_kb_semi');
    expect(text).toContain('federation_kb_private');
  });

  // ─── Group 2: targeted routing ─────────────────────────────────────────────

  test('2.1 anonymous → peer2: not configured (existence must not leak)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer2', query: 'anything' },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
    expect(text.toLowerCase()).not.toMatch(/access.?denied|unauthorized|forbidden/);
  });

  test('2.2 anonymous → peer3: not configured', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer3', query: 'anything' },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
    expect(text.toLowerCase()).not.toMatch(/access.?denied|unauthorized|forbidden/);
  });

  test('2.3 scoped user → peer2: returns peer2-note content', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer2', query: 'federation ACL testing' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Semi-private peer2 content');
  });

  test('2.4 scoped user → peer3: not configured (admin-only KB inaccessible)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer3', query: 'anything' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
  });

  test('2.5 admin → peer2: returns peer2-note content', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer2', query: 'federation ACL testing' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Semi-private peer2 content');
  });

  test('2.6 admin → peer3: returns peer3-note content', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer3', query: 'federation ACL testing' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Admin-only peer3 content');
  });

  // ─── Group 3: fan-out ──────────────────────────────────────────────────────

  test('3.1 anonymous fan-out: no peer2 or peer3 content', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { query: 'federation ACL testing' },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).not.toContain('Semi-private peer2 content');
    expect(text).not.toContain('Admin-only peer3 content');
  });

  test('3.2 scoped user fan-out: peer2 content present, peer3 absent', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { query: 'federation ACL testing' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Semi-private peer2 content');
    expect(text).not.toContain('Admin-only peer3 content');
  });

  test('3.3 admin fan-out: content from peer2 and peer3 both present', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { query: 'federation ACL testing' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Semi-private peer2 content');
    expect(text).toContain('Admin-only peer3 content');
  });

  // ─── Group 4: revocation + ACL decoupling ─────────────────────────────────

  test('4.1 revoke peer2 outbound secret → federated_search returns revocation error', async ({ playwright }) => {
    const revokeData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
        ... on RevokeFederationSecretPayload { revokedId }
        ... on ErrorPayload { message }
      } } }
    `, { id: outboundPeer2Id });
    expect(revokeData.admin.revokeFederationSecret.revokedId).toBe(outboundPeer2Id);

    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer2', query: 'federation ACL testing' },
    }, adminToken);
    const text = result.result?.content?.[0]?.text ?? result.error?.message ?? '';
    // Hub must not silently downgrade to anonymous — must report revocation
    expect(text.toLowerCase()).toMatch(/revoked|no active|secret/);
    // Must not say "not configured" — that would mean KB-note is inaccessible, but admin can still see it
    expect(text.toLowerCase()).not.toContain('federation is not configured for kb_id');
  });

  test('4.2 after revocation: admin search still shows federation_kb_semi (ACL unchanged)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('federation_kb_semi');
  });

  test('4.3 after revocation: scoped user search still shows federation_kb_semi', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('federation_kb_semi');
  });
});
