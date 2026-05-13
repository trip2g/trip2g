// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn } from './helpers/auth.js';

/**
 * Bidirectional Federation E2E Tests
 *
 * Verifies peer ↔ hub bidirectional federation:
 * - Peer has hub-kb.md pointing to hub's MCP endpoint (http://app:20081/_system/mcp)
 * - Hub→peer (direction A) and peer→hub (direction B) secrets exchanged
 *
 * Key scenario: peer asks hub to search peer ("link back to yourself"):
 *   federated_search(kb_id="hub/peer") → peer→hub→peer.Search → peer's own content
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER_URL = 'http://localhost:20091';
const HUB_MCP = `${HUB_URL}/_system/mcp`;
const PEER_MCP = `${PEER_URL}/_system/mcp`;

// Deterministic 64-hex secrets (32 bytes each).
const SECRET_A = 'f0e1d2c3b4a5968778695a4b3c2d1e0ff0e1d2c3b4a5968778695a4b3c2d1e0f'; // hub→peer
const SECRET_B = '1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f800'; // peer→hub
const KID_A = `e2e-bidir-a-${crypto.randomBytes(4).toString('hex')}`;
const KID_B = `e2e-bidir-b-${crypto.randomBytes(4).toString('hex')}`;

async function gql(request, baseURL, cookie, query, variables = {}) {
  const response = await request.post(`${baseURL}/_system/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  if (!response.ok()) throw new Error(`GraphQL ${baseURL} failed: ${await response.text()}`);
  const body = await response.json();
  if (body.errors) throw new Error(`GraphQL errors (${baseURL}): ${JSON.stringify(body.errors)}`);
  return body.data;
}

async function mcpCall(request, url, name, args = {}) {
  const response = await request.post(url, {
    headers: { 'Content-Type': 'application/json' },
    data: { jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name, arguments: args } },
  });
  expect(response.ok(), `MCP ${url} status ${response.status()}`).toBeTruthy();
  return response.json();
}

test.describe.serial('Bidirectional Federation', () => {
  let hubRequest;
  let peerRequest;
  let hubCookie;
  let peerCookie;
  let outboundAId; // hub→peer outbound secret id (direction A)
  let outboundBId; // peer→hub outbound secret id (direction B)

  test.beforeAll(async ({ playwright }) => {
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    const hubJwt = await graphqlSignIn(hubRequest);
    hubCookie = `trip2g_e2e=${hubJwt}`;

    peerRequest = await playwright.request.newContext({ baseURL: PEER_URL });
    const peerJwt = await graphqlSignIn(peerRequest, 'hello@example.com', '111111', { useCache: false });
    peerCookie = `trip2g_e2e_peer=${peerJwt}`;

    const revokeQ = `mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
      ... on RevokeFederationSecretPayload { revokedId }
      ... on ErrorPayload { message }
    } } }`;

    // Direction A: hub→peer (hub calls peer's MCP)
    await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_A, secretHex: SECRET_A } });

    const outA = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_A, secretHex: SECRET_A, kbURL: 'http://app-peer:20091/_system/mcp' } });
    outboundAId = outA.admin.createOutboundFederationSecret.id;
    expect(outboundAId, 'hub→peer outbound secret must be created').toBeTruthy();

    // Direction B: peer→hub (peer calls hub's MCP, using docker-internal URL matching hub-kb.md)
    await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_B, secretHex: SECRET_B } });

    const outB = await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_B, secretHex: SECRET_B, kbURL: 'http://app:20081/_system/mcp' } });
    outboundBId = outB.admin.createOutboundFederationSecret.id;
    expect(outboundBId, 'peer→hub outbound secret must be created').toBeTruthy();
  });

  test.afterAll(async () => {
    const revokeQ = `mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
      ... on RevokeFederationSecretPayload { revokedId }
      ... on ErrorPayload { message }
    } } }`;
    if (outboundAId) await gql(hubRequest, HUB_URL, hubCookie, revokeQ, { id: outboundAId }).catch(() => {});
    if (outboundBId) await gql(peerRequest, PEER_URL, peerCookie, revokeQ, { id: outboundBId }).catch(() => {});
    await hubRequest?.dispose();
    await peerRequest?.dispose();
  });

  test('peer fan-out includes hub content', async () => {
    const result = await mcpCall(peerRequest, PEER_MCP, 'federated_search', {
      query: 'federation knowledge base',
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    // Fan-out labels results from each KB with [kb_id]
    expect(text).toContain('[hub]');
    // federation_kb is the hub note that links to the peer — always present in federation queries
    expect(text).toContain('federation_kb');
  });

  test('peer → hub direct: returns hub content', async () => {
    const result = await mcpCall(peerRequest, PEER_MCP, 'federated_search', {
      query: 'team status',
      kb_id: 'hub',
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toContain('team status');
    // Hub's public URL contains :20081
    expect(text).toContain('20081');
  });

  test('peer → hub/peer: peer gets its own content via hub round-trip', async () => {
    // Round-trip: peer asks hub to search peer (peer gets "a link back to itself")
    const result = await mcpCall(peerRequest, PEER_MCP, 'federated_search', {
      query: 'team status',
      kb_id: 'hub/peer',
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';

    // Peer's own content returned via two-hop chain
    expect(text.toLowerCase()).toContain('team status');
    // URL comes from peer (app-peer hostname set in PUBLIC_URL env)
    expect(text).toContain('app-peer');
  });

  test('hub → peer/hub: hub gets its own content via peer round-trip', async () => {
    const result = await mcpCall(hubRequest, HUB_MCP, 'federated_search', {
      query: 'team status',
      kb_id: 'peer/hub',
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toContain('team status');
    // URL comes from hub
    expect(text).toContain('20081');
  });

  test('depth header at FederationMaxDepth is rejected', async () => {
    // Default FederationMaxDepth is 3. Sending depth=3 must be rejected.
    const response = await hubRequest.post(HUB_MCP, {
      headers: {
        'Content-Type': 'application/json',
        'X-MCP-Federation-Depth': '3',
      },
      data: { jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name: 'search', arguments: { query: 'test' } } },
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    const errorMsg = body.error?.message ?? '';
    expect(errorMsg.toLowerCase()).toContain('max depth');
  });

  test('revoke peer→hub outbound: targeted federated_search to hub returns revocation error', async () => {
    const revokeQ = `mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
      ... on RevokeFederationSecretPayload { revokedId }
      ... on ErrorPayload { message }
    } } }`;
    const revokeData = await gql(peerRequest, PEER_URL, peerCookie, revokeQ, { id: outboundBId });
    expect(revokeData.admin.revokeFederationSecret.revokedId).toBe(outboundBId);
    outboundBId = null; // already revoked, skip afterAll cleanup

    // Targeted call: peer must not silently downgrade to anonymous access for a previously-configured KB
    const result = await mcpCall(peerRequest, PEER_MCP, 'federated_search', {
      query: 'federation knowledge base',
      kb_id: 'hub',
    });
    const text = result.result?.content?.[0]?.text ?? result.error?.message ?? '';
    // Accept any auth-failure signal: revoked secret, no active secret, unknown KID, or auth error
    expect(text.toLowerCase()).toMatch(/revoked|no active|secret|auth/);
  });
});
