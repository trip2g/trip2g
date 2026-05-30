// @ts-check
import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { graphqlSignIn, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const SYSTEM_GRAPHQL = `${APP_URL}/_system/graphql`;

// ── helpers ───────────────────────────────────────────────────────────────────

async function gqlApi(request, apiKey, query, variables = {}) {
  const res = await request.post(SYSTEM_GRAPHQL, {
    headers: { 'Content-Type': 'application/json', 'X-Api-Key': apiKey },
    data: { query, variables },
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

async function gqlAdmin(request, cookie, query, variables = {}) {
  const res = await request.post(SYSTEM_GRAPHQL, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

async function pushAndCommit(request, apiKey, notes) {
  await gqlApi(request, apiKey, `
    mutation PushNotes($input: PushNotesInput!) {
      pushNotes(input: $input) {
        ... on PushNotesPayload { notes { path } }
        ... on ErrorPayload { message }
      }
    }
  `, { input: { updates: notes } });

  await gqlApi(request, apiKey, `mutation { commitNotes { ... on CommitNotesPayload { success } ... on ErrorPayload { message } } }`);
}

async function hideAndCommit(request, apiKey, paths) {
  await gqlApi(request, apiKey, `
    mutation HideNotes($input: HideNotesInput!) {
      hideNotes(input: $input) {
        ... on HideNotesPayload { success }
        ... on ErrorPayload { message }
      }
    }
  `, { input: { paths } });

  await gqlApi(request, apiKey, `mutation { commitNotes { ... on CommitNotesPayload { success } ... on ErrorPayload { message } } }`);
}

async function createRelease(request, cookie, title) {
  const data = await gqlAdmin(request, cookie, `
    mutation CreateRelease($input: CreateReleaseInput!) {
      admin {
        createRelease(input: $input) {
          ... on CreateReleasePayload { release { id } }
          ... on ErrorPayload { message }
        }
      }
    }
  `, { input: { title } });
  const payload = data.admin.createRelease;
  if (payload.message) throw new Error(`createRelease failed: ${payload.message}`);
  return payload.release.id;
}

async function makeReleaseLive(request, cookie, id) {
  const data = await gqlAdmin(request, cookie, `
    mutation MakeReleaseLive($input: MakeReleaseLiveInput!) {
      admin {
        makeReleaseLive(input: $input) {
          ... on MakeReleaseLivePayload { release { id } }
          ... on ErrorPayload { message }
        }
      }
    }
  `, { input: { id } });
  const payload = data.admin.makeReleaseLive;
  if (payload.message) throw new Error(`makeReleaseLive failed: ${payload.message}`);
}

async function createAndActivateRelease(request, cookie, title) {
  const id = await createRelease(request, cookie, title);
  await makeReleaseLive(request, cookie, id);
}

async function setConfigBool(request, cookie, id, value) {
  await gqlAdmin(request, cookie, `
    mutation SetConfig($input: SetConfigBoolValueInput!) {
      admin {
        setConfigBoolValue(input: $input) {
          ... on SetConfigBoolValueSuccess { configValue { id } }
          ... on ErrorPayload { message }
        }
      }
    }
  `, { input: { id, value } });
}

async function unreleasedChanges(request, apiKey, includePatterns = ['**']) {
  const data = await gqlApi(request, apiKey, `
    query UnreleasedChanges($filter: NoteChangesFilter!) {
      unreleasedChanges(filter: $filter) {
        totalCount
        totalStats { addedLines removedLines changedWords }
        nodes {
          path
          changeType
          stats { addedLines removedLines changedWords }
          unifiedDiff
          oldContent
          newContent
        }
      }
    }
  `, { filter: { includePatterns } });
  return data.unreleasedChanges;
}

// ── Part 1: unreleasedChanges API ────────────────────────────────────────────

test.describe('unreleasedChanges', () => {
  let apiKey;
  let adminCookie;

  test.beforeAll(async ({ request }) => {
    const apiKeyPath = path.join(process.cwd(), '.test-api-key');
    apiKey = fs.readFileSync(apiKeyPath, 'utf8').trim();
    const token = await graphqlSignIn(request);
    adminCookie = `${USER_TOKEN_COOKIE_NAME}=${token}`;
  });

  test('returns 0 after creating a live release', async ({ request }) => {
    await pushAndCommit(request, apiKey, [
      { path: 'e2e-uc-base-a.md', content: '# Base A\n\nInitial content.' },
      { path: 'e2e-uc-base-b.md', content: '# Base B\n\nInitial content.' },
    ]);
    await createAndActivateRelease(request, adminCookie, 'e2e-uc-baseline');

    const conn = await unreleasedChanges(request, apiKey);
    const ourNotes = conn.nodes.filter(n => n.path.startsWith('e2e-uc-'));
    expect(ourNotes).toHaveLength(0);
  });

  test('returns ADDED for notes pushed after the live release', async ({ request }) => {
    await pushAndCommit(request, apiKey, [
      { path: 'e2e-uc-new-1.md', content: '# New 1\n\nAdded after release.' },
      { path: 'e2e-uc-new-2.md', content: '# New 2\n\nAdded after release.' },
    ]);

    const conn = await unreleasedChanges(request, apiKey, ['e2e-uc-new-*']);
    expect(conn.totalCount).toBe(2);
    for (const node of conn.nodes) {
      expect(node.changeType).toBe('ADDED');
      expect(node.oldContent).toBeNull();
      expect(node.newContent).toBeTruthy();
    }
  });

  test('returns MODIFIED with non-empty unifiedDiff for updated note', async ({ request }) => {
    await pushAndCommit(request, apiKey, [
      { path: 'e2e-uc-base-a.md', content: '# Base A\n\nModified content.' },
    ]);

    const conn = await unreleasedChanges(request, apiKey, ['e2e-uc-base-a.md']);
    expect(conn.totalCount).toBe(1);
    const node = conn.nodes[0];
    expect(node.changeType).toBe('MODIFIED');
    expect(node.unifiedDiff).toContain('--- released');
    expect(node.unifiedDiff).toContain('+++ latest');
    expect(node.stats.removedLines).toBeGreaterThan(0);
    expect(node.stats.addedLines).toBeGreaterThan(0);
  });

  test('returns REMOVED for hidden note', async ({ request }) => {
    await hideAndCommit(request, apiKey, ['e2e-uc-base-b.md']);

    const conn = await unreleasedChanges(request, apiKey, ['e2e-uc-base-b.md']);
    expect(conn.totalCount).toBe(1);
    const node = conn.nodes[0];
    expect(node.changeType).toBe('REMOVED');
    expect(node.newContent).toBeNull();
    expect(node.oldContent).toBeTruthy();
  });

  test('totalCount resets to 0 after new live release', async ({ request }) => {
    await createAndActivateRelease(request, adminCookie, 'e2e-uc-post-changes');

    const conn = await unreleasedChanges(request, apiKey, ['e2e-uc-*']);
    expect(conn.totalCount).toBe(0);
  });

  test('totalStats aggregates line counts across all nodes', async ({ request }) => {
    await pushAndCommit(request, apiKey, [
      { path: 'e2e-uc-stats-a.md', content: 'line one\nline two\n' },
      { path: 'e2e-uc-stats-b.md', content: 'alpha\nbeta\ngamma\n' },
    ]);

    const conn = await unreleasedChanges(request, apiKey, ['e2e-uc-stats-*']);
    expect(conn.totalCount).toBe(2);
    expect(conn.totalStats.addedLines).toBeGreaterThanOrEqual(2);
    expect(conn.totalStats.changedWords).toBeGreaterThan(0);
  });

  test('glob filter excludes non-matching paths', async ({ request }) => {
    const conn = await unreleasedChanges(request, apiKey, ['e2e-uc-stats-a.md']);
    expect(conn.totalCount).toBe(1);
    expect(conn.nodes[0].path).toBe('e2e-uc-stats-a.md');
  });

  test('requires X-Api-Key auth', async ({ request }) => {
    const res = await request.post(SYSTEM_GRAPHQL, {
      headers: { 'Content-Type': 'application/json' },
      data: {
        query: `query { unreleasedChanges(filter: { includePatterns: ["**"] }) { totalCount } }`,
      },
    });
    const body = await res.json();
    expect(body.errors).toBeTruthy();
  });
});

// ── Part 2: show_draft_versions ───────────────────────────────────────────────

test.describe('show_draft_versions', () => {
  let apiKey;
  let adminCookie;
  const draftNotePath = 'e2e-draft-note.md';
  const draftNoteURL = `${APP_URL}/e2e-draft-note`;

  test.beforeAll(async ({ request }) => {
    const apiKeyPath = path.join(process.cwd(), '.test-api-key');
    apiKey = fs.readFileSync(apiKeyPath, 'utf8').trim();
    const token = await graphqlSignIn(request);
    adminCookie = `${USER_TOKEN_COOKIE_NAME}=${token}`;

    // Ensure show_draft_versions is off at start
    await setConfigBool(request, adminCookie, 'show_draft_versions', false);
  });

  test.afterAll(async ({ request }) => {
    // Restore to default off
    await setConfigBool(request, adminCookie, 'show_draft_versions', false);
  });

  test('setup: push note A, create live release', async ({ request }) => {
    await pushAndCommit(request, apiKey, [
      { path: draftNotePath, content: 'Content version A' },
    ]);
    await createAndActivateRelease(request, adminCookie, 'e2e-draft-baseline');
  });

  test('setup: push note B (latest only, not released)', async ({ request }) => {
    await pushAndCommit(request, apiKey, [
      { path: draftNotePath, content: 'Content version B' },
    ]);
  });

  test('guest sees live version A when show_draft_versions is off', async ({ request }) => {
    const res = await request.get(draftNoteURL);
    const html = await res.text();
    expect(html).toContain('Content version A');
    expect(html).not.toContain('Content version B');
  });

  test('admin also sees live version A when show_draft_versions is off', async ({ request }) => {
    const res = await request.get(draftNoteURL, {
      headers: { Cookie: adminCookie },
    });
    const html = await res.text();
    expect(html).toContain('Content version A');
    expect(html).not.toContain('Content version B');
  });

  test('admin sees latest version B when show_draft_versions is on', async ({ request }) => {
    await setConfigBool(request, adminCookie, 'show_draft_versions', true);

    const res = await request.get(draftNoteURL, {
      headers: { Cookie: adminCookie },
    });
    const html = await res.text();
    expect(html).toContain('Content version B');
    expect(html).not.toContain('Content version A');
  });

  test('guest still sees live version A when show_draft_versions is on', async ({ request }) => {
    const res = await request.get(draftNoteURL);
    const html = await res.text();
    expect(html).toContain('Content version A');
    expect(html).not.toContain('Content version B');
  });
});
