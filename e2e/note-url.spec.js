// @ts-check
/**
 * E2E tests for NoteView.url field — verifies domain-aware URL resolution
 * in GraphQL. A note with `route: customdomain.test/mcp-url-test` must return
 * a url starting with http://customdomain.test/, not APP_URL.
 */
import { test, expect } from '@playwright/test';
import { graphqlSignIn, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:8081';

const NOTE_URL_QUERY = `
  query {
    notePaths(filter: { like: "mcp-url-test.md" }) {
      latestNoteView {
        path
        permalink
        url
      }
    }
  }
`;

const REGULAR_NOTE_QUERY = `
  query {
    notePaths(filter: { like: "team-status.md" }) {
      latestNoteView {
        path
        permalink
        url
      }
    }
  }
`;

test.describe('NoteView.url field', () => {
  let cookie;

  test.beforeAll(async ({ request }) => {
    const jwt = await graphqlSignIn(request);
    cookie = `${USER_TOKEN_COOKIE_NAME}=${jwt}`;
  });

  test('custom domain note url uses the custom domain', async ({ request }) => {
    const res = await request.post(`${APP_URL}/graphql`, {
      headers: { 'Content-Type': 'application/json', Cookie: cookie },
      data: { query: NOTE_URL_QUERY },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.errors).toBeUndefined();

    const notePaths = body.data.notePaths;
    expect(notePaths.length).toBeGreaterThan(0);

    const noteView = notePaths[0].latestNoteView;
    expect(noteView).not.toBeNull();
    expect(noteView.url).toMatch(/^http:\/\/customdomain\.test\//);
    expect(noteView.url).toBe('http://customdomain.test/mcp-url-test');
  });

  test('regular note url uses APP_URL', async ({ request }) => {
    const res = await request.post(`${APP_URL}/graphql`, {
      headers: { 'Content-Type': 'application/json', Cookie: cookie },
      data: { query: REGULAR_NOTE_QUERY },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.errors).toBeUndefined();

    const notePaths = body.data.notePaths;
    expect(notePaths.length).toBeGreaterThan(0);

    const noteView = notePaths[0].latestNoteView;
    expect(noteView).not.toBeNull();
    expect(noteView.url).toMatch(new RegExp(`^${APP_URL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}/`));
  });
});
