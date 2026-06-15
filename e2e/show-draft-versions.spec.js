// @ts-check
import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { graphqlSignIn, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

// This spec flips the global `show_draft_versions` config, which switches public
// serving from "latest committed" to the live-release snapshot. That side effect
// is global on the shared server, so this spec MUST run dead last (see
// test-e2e.sh) — any later spec that pushes notes and expects them served would
// 404. The afterAll restores the default (true), but keeping it last is the
// belt-and-suspenders guarantee.
test.describe.configure({ mode: 'serial' });

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

async function getLiveReleaseId(request, cookie) {
  const data = await gqlAdmin(request, cookie, `
    query { admin { allReleases { nodes { id isLive } } } }
  `);
  const live = data.admin.allReleases.nodes.find(r => r.isLive);
  return live ? live.id : null;
}

// ── show_draft_versions ───────────────────────────────────────────────────────

test.describe('show_draft_versions', () => {
  test.describe.configure({ mode: 'serial' });
  let apiKey;
  let adminCookie;
  let originalLiveReleaseId;
  // Note: the server canonicalizes URL separators to underscores
  // (normalizeURLPart maps every non-alphanumeric char to "_"), so a file named
  // with hyphens is served at the underscore URL. Keep path and URL aligned.
  const draftNotePath = 'e2e_draft_note.md';
  const draftNoteURL = `${APP_URL}/e2e_draft_note`;

  test.beforeAll(async ({ request }) => {
    const apiKeyPath = path.join(process.cwd(), '.test-api-key');
    apiKey = fs.readFileSync(apiKeyPath, 'utf8').trim();
    const token = await graphqlSignIn(request);
    adminCookie = `${USER_TOKEN_COOKIE_NAME}=${token}`;
    originalLiveReleaseId = await getLiveReleaseId(request, adminCookie);
    await setConfigBool(request, adminCookie, 'show_draft_versions', false);
  });

  test.afterAll(async ({ request }) => {
    // Restore the default (true): with it on, public serving falls back to
    // "latest committed" instead of a live-release snapshot. Leaving it false
    // would make every later-pushed note 404. We intentionally restore the
    // default rather than the captured value because the default is the only
    // state in which freshly pushed notes are served.
    await setConfigBool(request, adminCookie, 'show_draft_versions', true);
    if (originalLiveReleaseId) {
      await makeReleaseLive(request, adminCookie, originalLiveReleaseId);
    }
  });

  test('setup: push note A, create live release', async ({ request }) => {
    await pushAndCommit(request, apiKey, [
      { path: draftNotePath, content: '---\nfree: true\n---\n\nContent version A' },
    ]);
    await createAndActivateRelease(request, adminCookie, 'e2e-draft-baseline');
  });

  test('setup: push note B (latest only, not released)', async ({ request }) => {
    await pushAndCommit(request, apiKey, [
      { path: draftNotePath, content: '---\nfree: true\n---\n\nContent version B' },
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

  test('guest also sees draft version B when show_draft_versions is on', async ({ request }) => {
    // show_draft_versions promotes the latest (unreleased) version to the default
    // view for everyone, guests included — not just admins. With it on, a guest
    // sees version B (latest), same as the admin above.
    const res = await request.get(draftNoteURL);
    const html = await res.text();
    expect(html).toContain('Content version B');
    expect(html).not.toContain('Content version A');
  });
});
