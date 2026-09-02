// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn } from './helpers/auth.js';

/**
 * MCP Federation E2E Tests
 *
 * Tests federated search/similar/note_html across two trip2g instances:
 * - Hub (port 20081) — main app with docs/demo content
 * - Peer (port 20091) — second app with testdata/seedvault content
 *
 * Federation secrets are bootstrapped in beforeAll using deterministic hex keys
 * and the createInboundFederationSecret(secretHex:) parameter.
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER_URL = 'http://localhost:20091';
const HUB_GRAPHQL = `${HUB_URL}/_system/graphql`;
const PEER_GRAPHQL = `${PEER_URL}/_system/graphql`;
const HUB_MCP = `${HUB_URL}/_system/mcp`;

// Deterministic secret for reproducible test runs (64 hex chars = 32 bytes).
const SECRET_HEX = 'a0b0c0d0e0f001020304050607080900010203040506070809a0b0c0d0e0f000';
// Unique kid per run to avoid collisions with previous runs.
const KID = `e2e-${crypto.randomBytes(4).toString('hex')}`;

/** Execute GraphQL mutation/query with cookie auth. */
async function gql(request, baseURL, cookie, query, variables = {}) {
  const response = await request.post(`${baseURL}/_system/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });

  if (!response.ok()) {
    const failedBody = await response.text()
    throw new Error(`GraphQL request to ${baseURL} failed: ${failedBody}, variables: ${JSON.stringify(variables)}`);
  }

  const body = await response.json();
  if (body.errors) {
    throw new Error(`GraphQL errors (${baseURL}): ${JSON.stringify(body.errors)}`);
  }

  return body.data;
}

/** Send a JSON-RPC request to an MCP endpoint. */
async function mcpCall(request, url, method, params = {}) {
  const response = await request.post(url, {
    headers: { 'Content-Type': 'application/json' },
    data: { jsonrpc: '2.0', id: 1, method, params },
  });
  expect(response.ok(), `MCP request to ${url} failed: ${response.status()}`).toBeTruthy();
  return response.json();
}

test.describe.serial('Federation', () => {
  let hubCookie;
  let peerCookie;
  let hubRequest;
  let peerRequest;
  let outboundSecretId;

  test.beforeAll(async ({ playwright }) => {
    // Sign into hub
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    const hubToken = await graphqlSignIn(hubRequest);
    hubCookie = `trip2g_e2e=${hubToken}`;

    // Sign into peer (different cookie name, different JWT secret — skip cache)
    peerRequest = await playwright.request.newContext({ baseURL: PEER_URL });
    const peerToken = await graphqlSignIn(peerRequest, 'hello@example.com', '111111', { useCache: false });
    peerCookie = `trip2g_e2e_peer=${peerToken}`;
  });

  test.afterAll(async () => {
    await hubRequest?.dispose();
    await peerRequest?.dispose();
  });

  test.beforeAll(async ({ playwright }) => {
    // 1. Create inbound secret on peer
    const inboundData = await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin {
          createInboundFederationSecret(input: $input) {
            ... on CreateInboundFederationSecretPayload { id kid secretHex }
            ... on ErrorPayload { message }
          }
        }
      }
    `, { input: { kid: KID, secretHex: SECRET_HEX } });

    console.log(inboundData);

    const inbound = inboundData.admin.createInboundFederationSecret;
    expect(inbound.kid).toBe(KID);
    expect(inbound.secretHex).toBe(SECRET_HEX);

    // 2. Create outbound secret on hub pointing to peer's MCP endpoint
    const outboundData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin {
          createOutboundFederationSecret(input: $input) {
            ... on CreateOutboundFederationSecretPayload { id kid }
            ... on ErrorPayload { message }
          }
        }
      }
    `, {
      input: {
        kid: KID,
        secretHex: SECRET_HEX,
        kbURL: 'http://app-peer:20091/_system/mcp',
      },
    });

    const outbound = outboundData.admin.createOutboundFederationSecret;
    expect(outbound.kid).toBe(KID);
    outboundSecretId = outbound.id;
  });

  test('tools/list includes federated tools', async () => {
    const result = await mcpCall(hubRequest, HUB_MCP, 'tools/list');
    const toolNames = result.result.tools.map((t) => t.name);

    expect(toolNames).toContain('search');
    expect(toolNames).toContain('similar');
    expect(toolNames).toContain('note_html');
    expect(toolNames).toContain('federated_search');
    expect(toolNames).toContain('federated_similar');
    expect(toolNames).toContain('federated_note_html');
  });

  test('hub local search returns federation_kb note', async () => {
    const result = await mcpCall(hubRequest, HUB_MCP, 'tools/call', {
      name: 'search',
      arguments: { query: 'peer federation' },
    });

    const text = result.result.content[0].text;
    expect(text).toContain('federation_kb');
    expect(text).toContain('peer');
  });

  test('federated_search returns peer notes', async () => {
    const result = await mcpCall(hubRequest, HUB_MCP, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer', query: 'team status' },
    });

    expect(result.result).toBeDefined();
    const text = result.result.content[0].text;
    expect(text).toContain('team');
  });

  test('federated_search with unknown kb_id returns structured error', async () => {
    const result = await mcpCall(hubRequest, HUB_MCP, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'ghost-kb', query: 'anything' },
    });

    // Should be a tool result, not a JSON-RPC error
    expect(result.error).toBeUndefined();
    expect(result.result).toBeDefined();
    const text = result.result.content[0].text;
    // The handler returns a "not configured" message or kb-not-found
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
  });

  test('federated_search behind the peer names both the peer bases and the hub bases', async () => {
    // The peer answers "not configured" in its own frame; the hub rewrites it
    // into the caller's and appends its own directly-connected bases, so the
    // caller is not left believing the peer's list is everything there is.
    const result = await mcpCall(hubRequest, HUB_MCP, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer/ghost-kb', query: 'anything' },
    });

    expect(result.error).toBeUndefined();
    const text = result.result.content[0].text;
    expect(text).toContain('Federation is not configured for kb_id "peer/ghost-kb"');
    expect(text).toContain('hub "peer" has no base "ghost-kb"');
    expect(text).toContain('Bases connected directly to this hub: peer');

    const status = result.result.structuredContent;
    expect(status.status).toBe('federation_not_configured');
    expect(status.kb_id).toBe('peer/ghost-kb');
    expect(status.hub).toBe('peer');
    // The peer's own list in the caller's frame: the seed vault federates the
    // hub back as "hub", so it arrives as peer/hub. An anonymous caller sees
    // only the hub's free KB-note (peer), not the subgraph-gated peer2/peer3.
    expect(status.connected_kb_ids).toEqual(['peer/hub']);
    expect(text).toContain('Bases connected under "peer": peer/hub');
    expect(status.local_kb_ids).toEqual(['peer']);
    expect(status.message).toBe(text);
  });

  test('federated_expand of a leaf section returns it through the peer', async () => {
    const args = { kb_id: 'peer', path: 'expand-steps.md', toc_path: ['Boot the server'] };
    const [expanded, viaNoteHTML] = await Promise.all([
      mcpCall(hubRequest, HUB_MCP, 'tools/call', { name: 'federated_expand', arguments: args }),
      mcpCall(hubRequest, HUB_MCP, 'tools/call', { name: 'federated_note_html', arguments: args }),
    ]);

    expect(expanded.error).toBeUndefined();
    const text = expanded.result.content[0].text;
    expect(text).toContain('expand-leaf-sentinel-boot');
    expect(text).not.toContain('expand-leaf-sentinel-logs');
    expect(text).toBe(viaNoteHTML.result.content[0].text);

    const payload = expanded.result.structuredContent;
    expect(payload.note_path).toBe('expand-steps.md');
    expect(payload.children ?? []).toHaveLength(0);
    expect(payload.section_html).toContain('expand-leaf-sentinel-boot');

    // A section with subsections still lists them instead of its body.
    const parent = await mcpCall(hubRequest, HUB_MCP, 'tools/call', {
      name: 'federated_expand',
      arguments: { kb_id: 'peer', path: 'expand-steps.md', toc_path: ['Verify the install'] },
    });
    expect(parent.result.structuredContent.children.map((c) => c.title)).toEqual(['Check the server logs']);
    expect(parent.result.content[0].text).not.toContain('expand-leaf-sentinel-logs');
  });

  test('revoke outbound secret blocks federation', async () => {
    console.log({ id: outboundSecretId })
    // Revoke the outbound secret on the hub
    const revokeData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($id: Int64!) {
        admin {
          revokeFederationSecret(id: $id) {
            ... on RevokeFederationSecretPayload { revokedId }
            ... on ErrorPayload { message }
          }
        }
      }
    `, { id: outboundSecretId });

    expect(revokeData.admin.revokeFederationSecret.revokedId).toBe(outboundSecretId);

    // Federated search should now fail or return no results
    const result = await mcpCall(hubRequest, HUB_MCP, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer', query: 'team status' },
    });

    const text = result.result?.content?.[0]?.text ?? result.error?.message ?? '';
    // After revocation, the hub must not silently downgrade this configured private peer to anonymous access.
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)|auth|revoke|no.*secret/);
    expect(text.toLowerCase()).not.toContain('team status');
    expect(text.toLowerCase()).not.toContain('internal notes');
  });
});
