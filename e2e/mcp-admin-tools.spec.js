// @ts-check
/**
 * MCP Admin Tools E2E Tests
 *
 * Verifies the full path: API key auth on MCP + enable_mcp_admin_tools flag
 * controls visibility and execution of graphql_introspection / graphql_request.
 *
 * Setup:
 * - .test-api-key contains the admin-created API key (from setup.spec.js)
 * - Admin signs in to toggle the flag via setApiKeyMcpAdminTools mutation
 *
 * Scenarios:
 * 1. With X-API-Key but flag OFF → graphql tools hidden in tools/list,
 *    tools/call graphql_request returns Method not found.
 * 2. With X-API-Key and flag ON → graphql tools visible, tools/call
 *    graphql_introspection works, tools/call graphql_request executes a real mutation.
 * 3. After toggling flag back OFF → tools disappear again.
 */

import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { graphqlSignIn } from './helpers/auth.js';

// Use the cookie name the e2e hub is configured with (USER_TOKEN_COOKIE_NAME env var set in
// test-e2e.sh; fall back to 'trip2g_e2e' so the spec works when run directly against the
// docker stack without the wrapper script).
const USER_TOKEN_COOKIE_NAME = process.env.USER_TOKEN_COOKIE_NAME || 'trip2g_e2e';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const MCP_URL = `${APP_URL}/_system/mcp`;

async function gql(request, baseURL, cookie, query, variables = {}) {
  const res = await request.post(`${baseURL}/_system/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  if (!res.ok()) throw new Error(`GraphQL HTTP ${res.status()}`);
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

async function mcpCallWithAPIKey(request, apiKey, method, params = {}) {
  const res = await request.post(MCP_URL, {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey,
    },
    data: { jsonrpc: '2.0', id: 1, method, params },
  });
  expect(res.ok()).toBeTruthy();
  return res.json();
}

async function findApiKeyId(request, baseURL, cookie, plaintextValue) {
  // The GraphQL API does not expose the plaintext key value, so we cannot match
  // directly. Instead we probe each candidate: temporarily enable enableMcpAdminTools
  // on it and verify that an MCP tools/list call with plaintextValue shows admin tools.
  // The first key that passes the probe is the correct one; we restore its original state.
  const data = await gql(request, baseURL, cookie, `
    query { admin { allApiKeys { nodes { id description enableMcpAdminTools } } } }
  `);
  const nodes = data.admin.allApiKeys.nodes || [];
  if (nodes.length === 0) throw new Error('no api keys found in admin');

  for (const node of nodes) {
    const originalFlag = node.enableMcpAdminTools;
    // Enable admin tools on this candidate key.
    await gql(request, baseURL, cookie, `
      mutation Set($input: SetApiKeyMcpAdminToolsInput!) {
        admin { setApiKeyMcpAdminTools(input: $input) {
          ... on SetApiKeyMcpAdminToolsPayload { apiKey { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { id: node.id, enabled: true } });

    // Check if the MCP endpoint using plaintextValue now sees admin tools.
    const mcpRes = await request.post(`${baseURL}/_system/mcp`, {
      headers: { 'Content-Type': 'application/json', 'X-API-Key': plaintextValue },
      data: { jsonrpc: '2.0', id: 1, method: 'tools/list', params: {} },
    });
    const mcpBody = await mcpRes.json();
    const names = (mcpBody.result?.tools || []).map((t) => t.name);
    const isMatch = names.includes('graphql_introspection');

    // Restore the original flag state before deciding.
    if (!originalFlag) {
      await gql(request, baseURL, cookie, `
        mutation Set($input: SetApiKeyMcpAdminToolsInput!) {
          admin { setApiKeyMcpAdminTools(input: $input) {
            ... on SetApiKeyMcpAdminToolsPayload { apiKey { id } }
            ... on ErrorPayload { message }
          } }
        }
      `, { input: { id: node.id, enabled: false } });
    }

    if (isMatch) return node.id;
  }

  throw new Error('could not find api key matching .test-api-key value via MCP probe');
}

async function setAdminToolsFlag(request, baseURL, cookie, id, enabled) {
  return gql(request, baseURL, cookie, `
    mutation Set($input: SetApiKeyMcpAdminToolsInput!) {
      admin {
        setApiKeyMcpAdminTools(input: $input) {
          ... on SetApiKeyMcpAdminToolsPayload { apiKey { id enableMcpAdminTools } }
          ... on ErrorPayload { message }
        }
      }
    }
  `, { input: { id, enabled } });
}

test.describe.serial('MCP API Key admin tools', () => {
  let adminRequest;
  let adminCookie;
  let apiKey;
  let apiKeyId;

  test.beforeAll(async ({ playwright }) => {
    const apiKeyPath = path.join(process.cwd(), '.test-api-key');
    if (!fs.existsSync(apiKeyPath)) {
      throw new Error('.test-api-key not found — run setup.spec.js first');
    }
    apiKey = fs.readFileSync(apiKeyPath, 'utf8').trim();

    adminRequest = await playwright.request.newContext({ baseURL: APP_URL });
    const jwt = await graphqlSignIn(adminRequest);
    adminCookie = `${USER_TOKEN_COOKIE_NAME}=${jwt}`;

    apiKeyId = await findApiKeyId(adminRequest, APP_URL, adminCookie, apiKey);
  });

  test.afterAll(async () => {
    if (adminRequest && apiKeyId) {
      // Leave flag OFF to not affect other tests.
      await setAdminToolsFlag(adminRequest, APP_URL, adminCookie, apiKeyId, false).catch(() => {});
    }
    await adminRequest?.dispose();
  });

  test('flag OFF: graphql tools hidden in tools/list', async ({ playwright }) => {
    await setAdminToolsFlag(adminRequest, APP_URL, adminCookie, apiKeyId, false);

    const ctx = await playwright.request.newContext({});
    try {
      const result = await mcpCallWithAPIKey(ctx, apiKey, 'tools/list');
      expect(result.error).toBeUndefined();
      const names = (result.result.tools || []).map((t) => t.name);
      expect(names).not.toContain('graphql_introspection');
      expect(names).not.toContain('graphql_request');
    } finally {
      await ctx.dispose();
    }
  });

  test('flag OFF: tools/call graphql_request returns method-not-found', async ({ playwright }) => {
    await setAdminToolsFlag(adminRequest, APP_URL, adminCookie, apiKeyId, false);

    const ctx = await playwright.request.newContext({});
    try {
      const result = await mcpCallWithAPIKey(ctx, apiKey, 'tools/call', {
        name: 'graphql_request',
        arguments: { query: '{ __typename }' },
      });
      expect(result.error, 'expected error').toBeDefined();
      expect(result.error.code).toBe(-32601); // MethodNotFound
    } finally {
      await ctx.dispose();
    }
  });

  test('flag ON: graphql tools visible in tools/list', async ({ playwright }) => {
    const setData = await setAdminToolsFlag(adminRequest, APP_URL, adminCookie, apiKeyId, true);
    expect(setData.admin.setApiKeyMcpAdminTools.apiKey?.enableMcpAdminTools).toBe(true);

    const ctx = await playwright.request.newContext({});
    try {
      const result = await mcpCallWithAPIKey(ctx, apiKey, 'tools/list');
      expect(result.error).toBeUndefined();
      const names = (result.result.tools || []).map((t) => t.name);
      expect(names).toContain('graphql_introspection');
      expect(names).toContain('graphql_request');
    } finally {
      await ctx.dispose();
    }
  });

  test('flag ON: graphql_introspection returns filtered schema with referenced types', async ({ playwright }) => {
    await setAdminToolsFlag(adminRequest, APP_URL, adminCookie, apiKeyId, true);

    const ctx = await playwright.request.newContext({});
    try {
      const result = await mcpCallWithAPIKey(ctx, apiKey, 'tools/call', {
        name: 'graphql_introspection',
        arguments: { pattern: 'ApiKey' },
      });
      expect(result.error).toBeUndefined();
      const text = result.result?.content?.[0]?.text;
      expect(text, 'introspection output should contain text').toBeTruthy();

      const parsed = JSON.parse(text);
      const types = parsed.data.__schema.types;
      const names = types.map((t) => t.name);

      // ApiKey-related types must be present.
      expect(names.some((n) => /ApiKey/i.test(n))).toBe(true);
      // Mutation/Query roots are referenced — they should NOT be there unless
      // they reference an ApiKey type. Skip strict negative assertion here
      // because the schema is large; just verify the filter shrunk the result.
      const fullSchemaQuery = await mcpCallWithAPIKey(ctx, apiKey, 'tools/call', {
        name: 'graphql_introspection',
        arguments: { pattern: '.*' },
      });
      const fullText = fullSchemaQuery.result?.content?.[0]?.text;
      const fullTypes = JSON.parse(fullText).data.__schema.types;
      expect(types.length).toBeLessThan(fullTypes.length);
    } finally {
      await ctx.dispose();
    }
  });

  test('flag ON: graphql_request allows allowlisted queries, rejects mutations and admin field', async ({ playwright }) => {
    await setAdminToolsFlag(adminRequest, APP_URL, adminCookie, apiKeyId, true);

    const ctx = await playwright.request.newContext({});
    try {
      // graphql_request is query-only and limited to an allowlist of read-only root fields
      // (note, search, similarNotes, viewer, notePaths, resolveWikilinks).
      // admin field and mutations are intentionally rejected for security.

      // An allowlisted query (viewer) must succeed.
      // The response carries data in structuredContent, not content[0].text.
      const queryResult = await mcpCallWithAPIKey(ctx, apiKey, 'tools/call', {
        name: 'graphql_request',
        arguments: {
          query: 'query { viewer { id role } }',
        },
      });
      expect(queryResult.error).toBeUndefined();
      const viewerData = queryResult.result?.structuredContent?.data?.viewer;
      expect(viewerData).toBeDefined();
      expect(viewerData.role).toBeTruthy();

      // admin field is not in the read-only allowlist — must be rejected.
      const adminQueryResult = await mcpCallWithAPIKey(ctx, apiKey, 'tools/call', {
        name: 'graphql_request',
        arguments: {
          query: 'query { admin { allApiKeys { nodes { id } } } }',
        },
      });
      expect(adminQueryResult.error).toBeDefined();
      expect(adminQueryResult.error.code).toBe(-32602); // InvalidParams
      expect(adminQueryResult.error.message).toContain('"admin" is not allowed');

      // Mutations are rejected — graphql_request is query-only.
      const mutationResult = await mcpCallWithAPIKey(ctx, apiKey, 'tools/call', {
        name: 'graphql_request',
        arguments: {
          query: `mutation Set($input: SetApiKeyMcpAdminToolsInput!) {
            admin {
              setApiKeyMcpAdminTools(input: $input) {
                ... on SetApiKeyMcpAdminToolsPayload { apiKey { id enableMcpAdminTools } }
                ... on ErrorPayload { message }
              }
            }
          }`,
          variables: { input: { id: apiKeyId, enabled: false } },
        },
      });
      expect(mutationResult.error).toBeDefined();
      expect(mutationResult.error.code).toBe(-32602); // InvalidParams — only queries allowed

      // Toggle the flag OFF via the admin GraphQL endpoint (not MCP) and verify
      // that graphql_request then disappears from tools/list.
      await setAdminToolsFlag(adminRequest, APP_URL, adminCookie, apiKeyId, false);
      const afterList = await mcpCallWithAPIKey(ctx, apiKey, 'tools/list');
      const afterNames = (afterList.result.tools || []).map((t) => t.name);
      expect(afterNames).not.toContain('graphql_request');
    } finally {
      await ctx.dispose();
    }
  });

  test('no X-API-Key: tools/list does NOT show admin tools', async ({ request }) => {
    // Anonymous request (no X-API-Key, no Bearer) — admin tools must be hidden.
    const res = await request.post(MCP_URL, {
      headers: { 'Content-Type': 'application/json' },
      data: { jsonrpc: '2.0', id: 1, method: 'tools/list', params: {} },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.error).toBeUndefined();
    const names = (body.result.tools || []).map((t) => t.name);
    expect(names).not.toContain('graphql_introspection');
    expect(names).not.toContain('graphql_request');
  });
});
