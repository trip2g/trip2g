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
const HUB_GRAPHQL = `${HUB_URL}/graphql`;
const PEER_GRAPHQL = `${PEER_URL}/graphql`;
const HUB_MCP = `${HUB_URL}/_system/mcp`;

// Deterministic secret for reproducible test runs (64 hex chars = 32 bytes).
const SECRET_HEX = 'a0b0c0d0e0f001020304050607080900010203040506070809a0b0c0d0e0f000';
// Unique kid per run to avoid collisions with previous runs.
const KID = `e2e-${crypto.randomBytes(4).toString('hex')}`;

/** Execute GraphQL mutation/query with cookie auth. */
async function gql(request, baseURL, cookie, query, variables = {}) {
  const response = await request.post(`${baseURL}/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  expect(response.ok(), `GraphQL request to ${baseURL} failed: ${response.status()}`).toBeTruthy();
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

    // Sign into peer (different cookie name)
    peerRequest = await playwright.request.newContext({ baseURL: PEER_URL });
    const peerToken = await graphqlSignIn(peerRequest);
    peerCookie = `trip2g_e2e_peer=${peerToken}`;
  });

  test.afterAll(async () => {
    await hubRequest?.dispose();
    await peerRequest?.dispose();
  });

  test('bootstrap federation secrets', async () => {
    // 1. Create inbound secret on peer
    const inboundData = await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid secretHex }
          ... on ErrorPayload { message }
        }
      }
    `, { input: { kid: KID, secretHex: SECRET_HEX } });

    const inbound = inboundData.createInboundFederationSecret;
    expect(inbound.kid).toBe(KID);
    expect(inbound.secretHex).toBe(SECRET_HEX);

    // 2. Create outbound secret on hub pointing to peer's MCP endpoint
    const outboundData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: {
        kid: KID,
        secretHex: SECRET_HEX,
        kbURL: 'http://app-peer:20091/_system/mcp',
      },
    });

    const outbound = outboundData.createOutboundFederationSecret;
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

  test('revoke outbound secret blocks federation', async () => {
    // Revoke the outbound secret on the hub
    const revokeData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($id: Int64!) {
        revokeFederationSecret(id: $id) {
          ... on RevokeFederationSecretPayload { revokedId }
          ... on ErrorPayload { message }
        }
      }
    `, { id: outboundSecretId });

    expect(revokeData.revokeFederationSecret.revokedId).toBe(outboundSecretId);

    // Federated search should now fail or return no results
    const result = await mcpCall(hubRequest, HUB_MCP, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer', query: 'team status' },
    });

    expect(result.result).toBeDefined();
    const text = result.result.content[0].text;
    // After revocation, expect either auth failure or "not configured"
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)|auth|revoke|no.*secret/);
  });
});
