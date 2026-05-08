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
import { graphqlSignIn, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const MCP_URL = `${APP_URL}/_system/mcp`;

async function gql(request, baseURL, cookie, query, variables = {}) {
  const res = await request.post(`${baseURL}/graphql`, {
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
  // The list returns api keys with the description we created in setup.
  // We don't have a direct lookup-by-value, so we fetch all and pick
  // the one with the e2e setup description.
  const data = await gql(request, baseURL, cookie, `
    query { admin { allApiKeys { nodes { id description enableMcpAdminTools } } } }
  `);
  const nodes = data.admin.allApiKeys.nodes || [];
  if (nodes.length === 0) throw new Error('no api keys found in admin');
  // Setup creates a single key for E2E; if there are multiple, prefer one
  // whose description includes "e2e" or "test", fall back to first.
  const preferred = nodes.find((k) => /e2e|test/i.test(k.description));
  return preferred ? preferred.id : nodes[0].id;
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

  test('flag ON: graphql_request executes admin mutation (toggle flag via MCP)', async ({ playwright }) => {
    await setAdminToolsFlag(adminRequest, APP_URL, adminCookie, apiKeyId, true);

    const ctx = await playwright.request.newContext({});
    try {
      // Read current flag via graphql_request (sanity that admin queries work).
      const queryResult = await mcpCallWithAPIKey(ctx, apiKey, 'tools/call', {
        name: 'graphql_request',
        arguments: {
          query: 'query { admin { allApiKeys { nodes { id enableMcpAdminTools } } } }',
        },
      });
      expect(queryResult.error).toBeUndefined();
      const queryText = queryResult.result?.content?.[0]?.text;
      expect(queryText).toContain('"enableMcpAdminTools":true');

      // Toggle flag OFF via MCP graphql_request — proves write access.
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
      expect(mutationResult.error).toBeUndefined();
      const mutationText = mutationResult.result?.content?.[0]?.text;
      expect(mutationText).toContain('"enableMcpAdminTools":false');

      // After toggle, MCP should now hide graphql tools again — but the
      // current request is already dispatched. Send a fresh tools/list.
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
