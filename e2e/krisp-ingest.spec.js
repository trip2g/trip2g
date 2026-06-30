// @ts-check
/**
 * Krisp cron ingest e2e — deterministic, no LLM, no real data.
 *
 * Architecture under test:
 *   trip2g app (Docker) ← cron webhook delivery →
 *   fleet (Docker)      ← code executor (python3) →
 *   krisp-mock (Docker) → synthetic meeting list + block trees →
 *   fleet               → updateNotes (transcripts/**) via scoped token →
 *   trip2g app
 *
 * Scenario:
 *   1. Seed roles/krisp/transcript-ingest.md (cron role, executor:code).
 *   2. Fleet discovers the role, reconciles a cron webhook registered against
 *      the app (poll_seconds=3, so within ~3 s).
 *   3. Admin fires triggerCronWebhook → fleet delivers → python3 calls krisp-mock,
 *      builds transcript notes, prints {"changes":[...],"answer":"ingested 3"}.
 *   4. Fleet writes transcripts/<id>.md back via scoped token.
 *   5. Spec polls notePaths for the first meeting's transcript until present,
 *      then asserts content matches the krisp-mock synthetic data.
 *
 * krisp-mock, fleet, and the app run as Docker compose services
 * (see docker-compose.test.yml). APP_URL is the host-published app port.
 */

import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { graphqlSignIn } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const GRAPHQL_URL = `${APP_URL}/_system/graphql`;

// Role stored as a note at this path in the trip2g vault.
const ROLE_PATH = 'roles/krisp/transcript-ingest.md';

// First synthetic meeting ID from cmd/krispmock/main.go (meetingID1).
const MEETING_ID_1 = 'aabbccddeeff00112233445566778800';
const TRANSCRIPT_PATH_1 = `transcripts/${MEETING_ID_1}.md`;

// ---------------------------------------------------------------------------
// GraphQL helpers (same pattern as fleet.spec.js)
// ---------------------------------------------------------------------------

/** Admin cookie auth. */
async function gqlAdmin(request, cookie, query, variables = {}) {
  const r = await request.post(GRAPHQL_URL, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  expect(r.ok()).toBeTruthy();
  const j = await r.json();
  if (j.errors) throw new Error(`GraphQL errors: ${JSON.stringify(j.errors)}`);
  return j.data;
}

/** API key auth. */
async function gqlApi(request, apiKey, query, variables = {}) {
  const r = await request.post(GRAPHQL_URL, {
    headers: { 'Content-Type': 'application/json', 'X-Api-Key': apiKey },
    data: { query, variables },
  });
  expect(r.ok()).toBeTruthy();
  const j = await r.json();
  if (j.errors) throw new Error(`GraphQL errors: ${JSON.stringify(j.errors)}`);
  return j.data;
}

// ---------------------------------------------------------------------------
// Role content — read the SHIPPED role-note verbatim so the e2e exercises the
// exact file we ship. An inline copy silently drifts from the doc (it already
// had: double-quoted Python vs the doc's single quotes).
// ---------------------------------------------------------------------------

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const roleSeed = fs.readFileSync(
  path.join(REPO_ROOT, 'docs/fleet/krisp/roles/transcript-ingest.md'),
  'utf8',
);

// ---------------------------------------------------------------------------
// Test suite
// ---------------------------------------------------------------------------

test.describe.serial('Krisp cron ingest (dockerized)', () => {
  // Role seed + 30 s fleet reconcile + python execution + write-back.
  test.setTimeout(120000);

  /** @type {string} Admin session cookie. */
  let cookie;
  /** @type {string} Full-admin API key for note reads. */
  let apiKey;

  test.beforeAll({ timeout: 90000 }, async ({ request }) => {
    const token = await graphqlSignIn(request, 'hello@example.com', '111111', { useCache: false });
    const cookieName = process.env.USER_TOKEN_COOKIE_NAME || 'trip2g_e2e';
    cookie = `${cookieName}=${token}`;

    // Create a full-admin API key for note queries.
    const keyData = await gqlAdmin(request, cookie, `
      mutation {
        admin {
          createApiKey(input: { description: "krisp-ingest-e2e" }) {
            ... on CreateApiKeyPayload { value apiKey { id } }
            ... on ErrorPayload { message }
          }
        }
      }
    `);
    expect(keyData.admin.createApiKey.message).toBeFalsy();
    apiKey = keyData.admin.createApiKey.value;

    // Seed the cron ingest role note.
    const seedResult = await gqlApi(request, apiKey, `
      mutation Up($input: UpdateNotesInput!) {
        updateNotes(input: $input) {
          ... on UpdateNotesSuccessPayload { paths }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: {
        changes: [
          { upsert: { path: ROLE_PATH, content: roleSeed } },
        ],
      },
    });
    expect(seedResult.updateNotes.message).toBeFalsy();

    // Wait for the fleet to discover the role and register a cron webhook.
    // Fleet polls every 3 s (TRIP2G_FLEET_POLL_SECONDS=3 in compose); allow 30 s.
    await expect
      .poll(
        async () => {
          const d = await gqlAdmin(request, cookie, `
            query { admin { allCronWebhooks { nodes { description } } } }
          `);
          return d.admin.allCronWebhooks.nodes.some(n =>
            n.description.startsWith('fleetcron:e2e:' + ROLE_PATH + '#'),
          );
        },
        { timeout: 30000, intervals: [1000] },
      )
      .toBeTruthy();
  });

  test('triggerCronWebhook → python ingest → transcripts/<id>.md written with meeting content', async ({ request }) => {
    // Find the cron webhook registered by fleet.
    const wh = await gqlAdmin(request, cookie, `
      query { admin { allCronWebhooks { nodes { id description } } } }
    `);
    const cronWebhook = wh.admin.allCronWebhooks.nodes.find(n =>
      n.description.startsWith('fleetcron:e2e:' + ROLE_PATH + '#'),
    );
    expect(cronWebhook, 'fleet cron webhook must be registered').toBeTruthy();

    // Manually fire the cron webhook. This enqueues a background delivery job.
    const triggerData = await gqlAdmin(request, cookie, `
      mutation Trigger($input: TriggerCronWebhookInput!) {
        admin {
          triggerCronWebhook(input: $input) {
            ... on TriggerCronWebhookPayload { deliveryId }
            ... on ErrorPayload { message }
          }
        }
      }
    `, { input: { cronWebhookId: cronWebhook.id } });
    expect(triggerData.admin.triggerCronWebhook.message).toBeFalsy();

    // Poll until the fleet's python ingest writes transcripts/<meetingID1>.md.
    // The delivery is async: background job → fleet → python → updateNotes.
    // 90 s covers delivery queue latency + python execution + write-back.
    await expect
      .poll(
        async () => {
          const d = await gqlApi(request, apiKey, `
            query N($f: NotePathsFilter) { notePaths(filter: $f) { content } }
          `, { f: { paths: [TRANSCRIPT_PATH_1] } });
          return (d.notePaths[0]?.content ?? '').length > 0;
        },
        { timeout: 90000, intervals: [2000] },
      )
      .toBeTruthy();

    // Assert transcript content matches the krisp-mock synthetic data for meeting 1.
    // meetingID1 speakers: Alice Mock (idx 1), Bob Mock (idx 2).
    // First utterance: "Good morning, let us get started with the Q1 planning."
    const d = await gqlApi(request, apiKey, `
      query N($f: NotePathsFilter) { notePaths(filter: $f) { content } }
    `, { f: { paths: [TRANSCRIPT_PATH_1] } });
    const content = d.notePaths[0]?.content ?? '';

    expect(content).toContain('Team Sync Q1 Planning');
    expect(content).toContain('Alice Mock');
    expect(content).toContain('Good morning, let us get started with the Q1 planning.');
    expect(content).toContain('Bob Mock');
  });
});
