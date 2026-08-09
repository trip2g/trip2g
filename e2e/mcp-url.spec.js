// @ts-check
/**
 * E2E tests for MCP search URL resolution.
 * A note with `route: customdomain.test/mcp-url-test` must appear in
 * search results with url = http://customdomain.test/mcp-url-test.
 * Regular notes must have urls starting with APP_URL.
 *
 * JSON field names (from MCP types.go):
 *   SearchResultPayload: { query, results[] }
 *   SearchResultItem:    { note_path, url, title, href, note_id, score }
 *   Response:            { result: { content[], structuredContent } }
 */
import { test, expect } from '@playwright/test';
import { graphqlSignIn, createPersonalToken, revokePersonalToken, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:8081';
const MCP_URL = `${APP_URL}/_system/mcp`;

async function mcpCall(request, method, params = {}, token) {
  const res = await request.post(MCP_URL, {
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json, text/event-stream',
      Authorization: `Bearer ${token}`,
    },
    data: { jsonrpc: '2.0', id: 1, method, params },
  });
  expect(res.ok()).toBeTruthy();
  return res.json();
}

// Returns SearchResultPayload: { query, results: [{ note_path, url, ... }] }
async function mcpSearch(request, query, token) {
  const resp = await mcpCall(request, 'tools/call', {
    name: 'search',
    arguments: { query },
  }, token);
  expect(resp.error).toBeUndefined();
  return resp.result?.structuredContent;
}

test.describe.serial('MCP search URL resolution', () => {
  let adminRequest;
  let adminCookie;
  let token;
  let tokenId;

  test.beforeAll(async ({ playwright }) => {
    adminRequest = await playwright.request.newContext({ baseURL: APP_URL });
    const jwt = await graphqlSignIn(adminRequest);
    adminCookie = `${USER_TOKEN_COOKIE_NAME}=${jwt}`;

    const result = await createPersonalToken(adminRequest, APP_URL, adminCookie, {
      name: 'e2e-mcp-url',
      expiresInDays: 1,
    });
    token = result.plaintextToken;
    tokenId = result.id;

    // Initialize MCP session
    await mcpCall(adminRequest, 'initialize', {
      protocolVersion: '2024-11-05',
      capabilities: {},
      clientInfo: { name: 'e2e-test', version: '1' },
    }, token);
  });

  test.afterAll(async () => {
    if (tokenId) {
      await revokePersonalToken(adminRequest, APP_URL, adminCookie, tokenId).catch(() => {});
    }
    await adminRequest?.dispose();
  });

  test('custom domain note appears with custom domain URL', async () => {
    const payload = await mcpSearch(adminRequest, 'xyzzy-mcp-url-test-sentinel', token);
    expect(payload).not.toBeNull();
    expect(payload.results?.length).toBeGreaterThan(0);

    const customNote = payload.results.find(r => r.note_path === 'mcp-url-test.md');
    expect(customNote).toBeDefined();
    expect(customNote.url).toBe('http://customdomain.test/mcp-url-test');
  });

  test('regular note appears with main domain URL', async () => {
    const payload = await mcpSearch(adminRequest, 'Weekly team status', token);
    expect(payload).not.toBeNull();
    expect(payload.results?.length).toBeGreaterThan(0);

    const regularNote = payload.results.find(r => r.note_path === 'team-status.md');
    expect(regularNote).toBeDefined();
    expect(regularNote.url).toMatch(new RegExp(`^${APP_URL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}/`));
  });
});
