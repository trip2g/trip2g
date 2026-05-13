// @ts-check
/**
 * MCP ACL and ?method= E2E Tests
 *
 * Covers two things:
 *
 * 1. ACL bug: handleDynamicMethod ignores subgraph access control.
 *    A note with mcp_method: premium_guide + subgraphs: premium must NOT be
 *    readable by anonymous users or non-subscribers via tools/call.
 *    Currently FAILS (content is returned without any access check).
 *
 * 2. ?method= feature: initialize with ?method=wiki must use the note with
 *    mcp_method: wiki as the main instruction instead of mcp_method: initialize.
 *    Currently FAILS (feature not implemented).
 *
 * Seed notes (docs/demo/):
 *   _mcp_premium_guide.md  — mcp_method: premium_guide, subgraphs: premium
 *   _mcp_wiki.md           — mcp_method: wiki, free: true
 */

import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn, createPersonalToken, revokePersonalToken, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

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

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const MCP_URL = `${APP_URL}/_system/mcp`;

async function mcpCall(request, url, method, params = {}, auth = {}) {
  const headers = { 'Content-Type': 'application/json' };
  let targetUrl = url;

  if (auth.bearer) headers['Authorization'] = `Bearer ${auth.bearer}`;
  if (auth.cookie) headers['Cookie'] = auth.cookie;
  if (auth.queryToken) targetUrl = `${url}${url.includes('?') ? '&' : '?'}token=${encodeURIComponent(auth.queryToken)}`;

  const res = await request.post(targetUrl, {
    headers,
    data: { jsonrpc: '2.0', id: 1, method, params },
  });
  expect(res.ok()).toBeTruthy();
  return res.json();
}

test.describe.serial('MCP ACL and ?method=', () => {
  let adminRequest;
  let adminCookie;
  let adminToken;
  let adminTokenId;

  test.beforeAll(async ({ playwright }) => {
    adminRequest = await playwright.request.newContext({ baseURL: APP_URL });
    const jwt = await graphqlSignIn(adminRequest);
    adminCookie = `${USER_TOKEN_COOKIE_NAME}=${jwt}`;

    const result = await createPersonalToken(adminRequest, APP_URL, adminCookie, {
      name: 'e2e-mcp-acl',
      expiresInDays: 1,
    });
    adminToken = result.plaintextToken;
    adminTokenId = result.id;
  });

  test.afterAll(async () => {
    if (adminTokenId) {
      await revokePersonalToken(adminRequest, APP_URL, adminCookie, adminTokenId).catch(() => {});
    }
    await adminRequest?.dispose();
  });

  // ── ACL bug: dynamic method ignores subgraph restriction ─────────────────

  test('anonymous user cannot call premium dynamic method', async ({ request }) => {
    const result = await mcpCall(request, MCP_URL, 'tools/call', {
      name: 'premium_guide',
      arguments: {},
    });
    // Must return error, not premium content
    expect(result.error, 'expected error for anonymous premium_guide call').toBeDefined();
    expect(result.result).toBeUndefined();
  });

  test('non-subscriber token cannot call premium dynamic method', async ({ playwright }) => {
    const nonAdminEmail = `e2e-acl-${crypto.randomBytes(4).toString('hex')}@example.com`;

    // Create a fresh non-admin user via admin GraphQL
    const createData = await gql(adminRequest, APP_URL, adminCookie, `
      mutation($input: CreateUserInput!) {
        admin { createUser(input: $input) {
          ... on CreateUserPayload { user { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { email: nonAdminEmail } });
    expect(createData.admin.createUser.user?.id, 'user creation failed').toBeDefined();

    const ctx = await playwright.request.newContext({ baseURL: APP_URL });
    try {
      const jwt = await graphqlSignIn(ctx, nonAdminEmail, '111111', { useCache: false });
      const cookie = `${USER_TOKEN_COOKIE_NAME}=${jwt}`;
      const { plaintextToken, id: tokenId } = await createPersonalToken(ctx, APP_URL, cookie, {
        name: 'e2e-acl-nopremium',
      });
      try {
        const freshCtx = await playwright.request.newContext({});
        try {
          const result = await mcpCall(freshCtx, MCP_URL, 'tools/call', {
            name: 'premium_guide',
            arguments: {},
          }, { bearer: plaintextToken });
          expect(result.error, 'expected error for non-subscriber premium_guide call').toBeDefined();
          expect(result.result).toBeUndefined();
        } finally {
          await freshCtx.dispose();
        }
      } finally {
        await revokePersonalToken(ctx, APP_URL, cookie, tokenId).catch(() => {});
      }
    } finally {
      await ctx.dispose();
    }
  });

  test('admin token can call premium dynamic method', async ({ playwright }) => {
    const ctx = await playwright.request.newContext({});
    try {
      const result = await mcpCall(ctx, MCP_URL, 'tools/call', {
        name: 'premium_guide',
        arguments: {},
      }, { bearer: adminToken });
      expect(result.error).toBeUndefined();
      expect(result.result?.content?.[0]?.text).toContain('Premium Knowledge Base Guide');
    } finally {
      await ctx.dispose();
    }
  });

  // ── ?method= feature: initialize uses named entry point ──────────────────

  test('initialize without ?method= returns default instructions', async ({ request }) => {
    const result = await mcpCall(request, MCP_URL, 'initialize', {});
    expect(result.error).toBeUndefined();
    // Default initialize note is the Marcus Aurelius one
    expect(result.result.instructions).toContain('Marcus Aurelius');
  });

  test('initialize with ?method=wiki returns wiki instructions', async ({ request }) => {
    const result = await mcpCall(request, `${MCP_URL}?method=wiki`, 'initialize', {});
    expect(result.error).toBeUndefined();
    expect(result.result.instructions).toContain('Wiki Knowledge Base Instructions');
    // Must NOT contain default initialize instructions
    expect(result.result.instructions).not.toContain('Marcus Aurelius');
  });

  test('initialize with ?method=nonexistent returns method not found error', async ({ request }) => {
    const result = await mcpCall(request, `${MCP_URL}?method=nonexistent`, 'initialize', {});
    expect(result.error).toBeDefined();
    expect(result.error.code).toBe(-32601); // ErrCodeMethodNotFound
  });

  test('initialize with ?method=premium_guide blocked for anonymous user', async ({ request }) => {
    const result = await mcpCall(request, `${MCP_URL}?method=premium_guide`, 'initialize', {});
    expect(result.error).toBeDefined();
  });

  test('initialize with ?method=premium_guide returns instructions for admin', async ({ playwright }) => {
    const ctx = await playwright.request.newContext({});
    try {
      const result = await mcpCall(ctx, `${MCP_URL}?method=premium_guide`, 'initialize', {}, { bearer: adminToken });
      expect(result.error).toBeUndefined();
      expect(result.result.instructions).toContain('Premium Knowledge Base Guide');
    } finally {
      await ctx.dispose();
    }
  });
});
