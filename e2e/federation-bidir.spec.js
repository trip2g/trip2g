// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn } from './helpers/auth.js';
import {
  createApiKey, disableApiKey, fedGql, hideNotes, revokeLiveOutboundSecrets, upsertKBNote,
} from './helpers/federation.js';

/**
 * Bidirectional Federation E2E Tests
 *
 * Verifies peer ↔ hub bidirectional federation:
 * - Peer has hub-kb.md pointing to hub's MCP endpoint (http://app:20081/_system/mcp)
 * - Hub→peer (direction A) and peer→hub (direction B) secrets exchanged
 *
 * Key scenario: peer asks hub to search peer ("link back to yourself"):
 *   federated_search(kb_id="hub/<KID_A>") → peer→hub→peer.Search → peer's own content
 *
 * Hub→peer goes through the peer's container name and a hub KB-note of this
 * spec's own (kb_id = KID_A), since the seed KB-note "peer" belongs to
 * federation.spec.js. That note is gated by a subgraph granted to KID_B, so the
 * peer reaches it and anonymous callers of the other specs do not see it.
 * Peer→hub uses the peer's seed KB-note "hub": this spec owns its URL.
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER_URL = 'http://localhost:20091';
const HUB_MCP = `${HUB_URL}/_system/mcp`;
const PEER_MCP = `${PEER_URL}/_system/mcp`;
const PEER_KB_URL = 'http://trip2g-test-app-peer:20091/_system/mcp';
const HUB_KB_URL = 'http://app:20081/_system/mcp';
const HUB_KB_NOTE = 'e2e_federation_bidir_peer_kb.md';

// Deterministic 64-hex secrets (32 bytes each).
const SECRET_A = 'f0e1d2c3b4a5968778695a4b3c2d1e0ff0e1d2c3b4a5968778695a4b3c2d1e0f'; // hub→peer
const SECRET_B = '1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f800'; // peer→hub
const KID_A = `e2e-bidir-a-${crypto.randomBytes(4).toString('hex')}`;
const KID_B = `e2e-bidir-b-${crypto.randomBytes(4).toString('hex')}`;
const PEER_SUBGRAPH = 'e2e-bidir-peer';

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

async function mcpCall(request, url, name, args = {}, extraHeaders = {}) {
  const response = await request.post(url, {
    headers: { 'Content-Type': 'application/json', ...extraHeaders },
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
  let hubApiKey;
  let hubApiKeyId;

  test.beforeAll(async ({ playwright }) => {
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    const hubJwt = await graphqlSignIn(hubRequest);
    hubCookie = `trip2g_e2e=${hubJwt}`;

    peerRequest = await playwright.request.newContext({ baseURL: PEER_URL });
    const peerJwt = await graphqlSignIn(peerRequest, 'hello@example.com', '111111', { useCache: false });
    peerCookie = `trip2g_e2e_peer=${peerJwt}`;

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
    `, { input: { kid: KID_A, secretHex: SECRET_A, kbURL: PEER_KB_URL } });
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

    await revokeLiveOutboundSecrets(peerRequest, PEER_URL, peerCookie, HUB_KB_URL);
    const outB = await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_B, secretHex: SECRET_B, kbURL: HUB_KB_URL } });
    outboundBId = outB.admin.createOutboundFederationSecret.id;
    expect(outboundBId, 'peer→hub outbound secret must be created').toBeTruthy();

    ({ value: hubApiKey, id: hubApiKeyId } = await createApiKey(hubRequest, HUB_URL, hubCookie, `e2e federation bidir ${KID_A}`));
    await upsertKBNote(hubRequest, HUB_URL, hubApiKey, {
      path: HUB_KB_NOTE, kbURL: PEER_KB_URL, kbID: KID_A, frontmatter: [`subgraphs: ${PEER_SUBGRAPH}`],
    });
    const subgraphs = await gql(hubRequest, HUB_URL, hubCookie, `query { admin { allSubgraphs { nodes { id name } } } }`);
    const peerSubgraphId = subgraphs.admin.allSubgraphs.nodes.find((n) => n.name === PEER_SUBGRAPH)?.id;
    expect(peerSubgraphId, `${PEER_SUBGRAPH} subgraph must exist on the hub`).toBeTruthy();
    const grant = await fedGql(hubRequest, HUB_URL, { Cookie: hubCookie }, `
      mutation($input: AddFederationSecretSubgraphInput!) {
        admin { addFederationSecretSubgraph(input: $input) {
          ... on AddFederationSecretSubgraphPayload { success }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_B, subgraphID: peerSubgraphId } });
    expect(grant.admin.addFederationSecretSubgraph.success).toBe(true);
  });

  test.afterAll(async () => {
    const revokeQ = `mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
      ... on RevokeFederationSecretPayload { revokedId }
      ... on ErrorPayload { message }
    } } }`;
    if (hubApiKey) {
      await hideNotes(hubRequest, HUB_URL, hubApiKey, [HUB_KB_NOTE]).catch(() => {});
      await disableApiKey(hubRequest, HUB_URL, hubCookie, hubApiKeyId).catch(() => {});
    }
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
      kb_id: `hub/${KID_A}`,
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';

    // Peer's own content returned via two-hop chain
    expect(text.toLowerCase()).toContain('team status');
    // URL comes from peer (app-peer hostname set in PUBLIC_URL env)
    expect(text).toContain('app-peer');
  });

  test('hub → peer/hub: hub gets its own content via peer round-trip', async () => {
    // The hub KB-note is not free, so the hub-side caller is the admin.
    const result = await mcpCall(hubRequest, HUB_MCP, 'federated_search', {
      query: 'team status',
      kb_id: `${KID_A}/hub`,
    }, { Cookie: hubCookie });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toContain('team status');
    // URL comes from hub
    expect(text).toContain('20081');
  });

  test('depth header at FederationMaxDepth is allowed (inclusive limit)', async () => {
    // Default FederationMaxDepth is 3. depth=3 reaches the limit but is still allowed.
    const result = await mcpCall(hubRequest, HUB_MCP, 'search', { query: 'test' }, { 'X-MCP-Federation-Depth': '3' });
    expect(result.error).toBeUndefined();
  });

  test('depth header past FederationMaxDepth is rejected', async () => {
    // depth=4 is one past the limit and must be rejected.
    const result = await mcpCall(hubRequest, HUB_MCP, 'search', { query: 'test' }, { 'X-MCP-Federation-Depth': '4' });
    expect(result.error?.message?.toLowerCase()).toContain('max depth');
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
