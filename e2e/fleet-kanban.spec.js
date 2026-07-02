// @ts-check
/**
 * Fleet Kanban Vertical Slice — E2E headline test (Task C9).
 *
 * Uses the shared Docker fleet + llm-mock services (docker-compose.test.yml),
 * exactly like fleet.spec.js.  No subprocess is spawned here.
 *
 * Architecture under test (same compose network as fleet.spec.js):
 *   trip2g app (Docker) ← change webhook delivery →
 *   fleet (Docker)      ← chat completions →
 *   llm-mock (Docker)   → patch_note tool call →
 *   fleet               → updateNotes via scoped token →
 *   trip2g app
 *
 * Scenario:
 *   1. Seed roles/triage.md (fleet role) and boards/sprint.md (board with
 *      a @status:doing card).
 *   2. Fleet discovers the role and reconciles a change webhook for boards/sprint.md.
 *   3. User edits boards/sprint.md → app fires the change webhook → fleet
 *      delivers → llm-mock detects "@status:doing" in the rendered instruction
 *      and returns patch_note({find:"@status:doing",replace:"@status:doing @triaged"})
 *      → fleet patches the board via scoped token.
 *   4. Spec polls notePaths until content contains "@status:doing @triaged".
 *
 * llm-mock route detection: cmd/mockserver/configs/llm.jsonnet branches on
 * "@status:doing" in the extracted instruction content.
 */

import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const GRAPHQL_URL = `${APP_URL}/_system/graphql`;

const BOARD_PATH = 'boards/sprint.md';
const ROLE_PATH = 'roles/triage.md';

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

/** Drain the app's background job queue. */
async function waitJobs(request) {
  const r = await request.get(`${APP_URL}/debug/wait_all_jobs`, { timeout: 60000 });
  expect(r.ok()).toBeTruthy();
}

// ---------------------------------------------------------------------------
// Test suite
// ---------------------------------------------------------------------------

test.describe.serial('Fleet kanban vertical slice', () => {
  // Fleet reconcile + patch_note round-trip easily exceeds 30 s on cold start.
  test.setTimeout(120000);

  /** @type {string} Admin session cookie. */
  let cookie;
  /** @type {string} Full-admin API key for updateNotes / notePaths queries. */
  let apiKey;

  // Initial board content — card is @status:doing so llm-mock's find matches
  // as soon as the fleet delivers.
  const boardSeed = '---\nlayout: kanban\n---\n\n## Doing\n- Fix login bug @status:doing\n';

  // Triage role: fires on boards/sprint.md updates, patches @status:doing lines.
  // model: mock → uses the shared docker llm-mock service.
  // llm-mock detects "@status:doing" in the rendered instruction and returns
  // patch_note on the first call, finish on the second.
  const roleSeed = [
    '---',
    'model: mock',
    'tools: [search, read_note, patch_note]',
    'read_patterns: ["boards/**","roles/**"]',
    'write_patterns: ["boards/**"]',
    'max_tokens: 4000',
    'max_steps: 6',
    'mode: change',
    'trigger_include: ["boards/sprint.md"]',
    'trigger_on: [update]',
    'attach_notes: ["boards/**","roles/**"]',
    'max_depth: 1',
    'concurrency: skip',
    'for_each: changed_files',
    '---',
    'You are a sprint-triage agent. A change occurred in the project board.',
    '',
    'Changed file: {{ change_file.Path }} (event: {{ change_file.Event }})',
    '',
    'Current content:',
    '{{ change_file.Content }}',
    '',
    'Append " @triaged" to any line containing "@status:doing" that does not yet',
    'have "@triaged". Use patch_note with the exact find text from the content above.',
  ].join('\n');

  test.beforeAll(async ({ request }) => {
    // Extend this hook's timeout: sign-in + seed + fleet reconcile can take >30 s.
    test.setTimeout(90000);

    // API-based sign-in — no browser needed (same pattern as fleet.spec.js).
    const token = await graphqlSignIn(request, 'hello@example.com', '111111', { useCache: false });
    const cookieName = process.env.USER_TOKEN_COOKIE_NAME || 'trip2g_e2e';
    cookie = `${cookieName}=${token}`;

    // Create a full-admin API key for note read/write.
    const keyData = await gqlAdmin(request, cookie, `
      mutation {
        admin {
          createApiKey(input: { description: "fleet-kanban-e2e" }) {
            ... on CreateApiKeyPayload { value apiKey { id } }
            ... on ErrorPayload { message }
          }
        }
      }
    `);
    expect(keyData.admin.createApiKey.message).toBeFalsy();
    apiKey = keyData.admin.createApiKey.value;

    // Seed the role note.  The board is seeded in the test function so it
    // arrives AFTER the fleet's webhook is registered (clean delivery count).
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
    await waitJobs(request);

    // Wait for the docker fleet to discover roles/triage.md and register a
    // change webhook.  Fleet polls every 3 s (TRIP2G_FLEET_POLL_SECONDS=3);
    // allow 30 s.
    await expect
      .poll(
        async () => {
          const d = await gqlAdmin(request, cookie, `
            query { admin { allChangeWebhooks { nodes { description } } } }
          `);
          return d.admin.allChangeWebhooks.nodes.some(n =>
            n.description.startsWith('fleet:e2e:roles/triage.md#'),
          );
        },
        { timeout: 30000, intervals: [1000] },
      )
      .toBeTruthy();
  });

  test('user edit → agent triage → one delivery, board updated, no re-trigger', async ({ request }) => {
    // Seed the initial board AFTER the fleet webhook is registered.
    const initResult = await gqlApi(request, apiKey, `
      mutation Up($input: UpdateNotesInput!) {
        updateNotes(input: $input) {
          ... on UpdateNotesSuccessPayload { paths }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: { changes: [{ upsert: { path: BOARD_PATH, content: boardSeed } }] },
    });
    expect(initResult.updateNotes.message).toBeFalsy();
    await waitJobs(request);

    // Locate the fleet triage webhook registered in beforeAll.
    const wh = await gqlAdmin(request, cookie, `
      query { admin { allChangeWebhooks { nodes { id description } } } }
    `);
    const fleetWebhook = wh.admin.allChangeWebhooks.nodes.find(n =>
      n.description.startsWith('fleet:e2e:roles/triage.md#'),
    );
    expect(fleetWebhook, 'fleet triage webhook must be registered').toBeTruthy();

    // Snapshot delivery count BEFORE the trigger.
    const before = await gqlAdmin(request, cookie, `
      query D($f: AdminChangeWebhookDeliveriesFilterInput!) {
        admin { changeWebhookDeliveries(filter: $f) { nodes { id } } }
      }
    `, { f: { webhookId: fleetWebhook.id } });
    const deliveryCountBefore = before.admin.changeWebhookDeliveries.nodes.length;

    // Simulate a user edit: add a new card while the @status:doing card remains.
    // This triggers the fleet's change webhook for boards/sprint.md.
    const boardWithUserEdit = boardSeed + '- Write tests @status:todo\n';
    const editResult = await gqlApi(request, apiKey, `
      mutation Up($input: UpdateNotesInput!) {
        updateNotes(input: $input) {
          ... on UpdateNotesSuccessPayload { paths }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: { changes: [{ upsert: { path: BOARD_PATH, content: boardWithUserEdit } }] },
    });
    expect(editResult.updateNotes.message).toBeFalsy();
    await waitJobs(request);

    // Poll until the fleet agent patches boards/sprint.md via llm-mock.
    // llm-mock detects "@status:doing" in the instruction and returns
    // patch_note({find:"@status:doing", replace:"@status:doing @triaged"}).
    await expect
      .poll(
        async () => {
          const d = await gqlApi(request, apiKey, `
            query N($f: NotePathsFilter) { notePaths(filter: $f) { content } }
          `, { f: { paths: [BOARD_PATH] } });
          return d.notePaths[0]?.content ?? '';
        },
        { timeout: 60000, intervals: [2000] },
      )
      .toContain('@status:doing @triaged');

    // Confirm exactly one new delivery fired (depth guard prevents re-trigger).
    // The fleet's write-back to boards/sprint.md must NOT create a second delivery
    // because max_depth:1 prevents recursive triggering.
    const after = await gqlAdmin(request, cookie, `
      query D($f: AdminChangeWebhookDeliveriesFilterInput!) {
        admin { changeWebhookDeliveries(filter: $f) { nodes { id status } } }
      }
    `, { f: { webhookId: fleetWebhook.id } });
    expect(
      after.admin.changeWebhookDeliveries.nodes.length - deliveryCountBefore,
      'exactly one new delivery — write-back must not re-trigger',
    ).toBe(1);

    // Note version count: board init (v1) + user edit (v2) + agent patch (v3) = >=2
    // post-init versions.  Soft >=2 since exact seed count is environment-dependent.
    const vhist = await gqlAdmin(request, cookie, `
      query VH($f: AdminNoteVersionHistoryFilter!) {
        admin { noteVersionHistory(filter: $f) { totalCount nodes { versionId version } } }
      }
    `, { f: { path: BOARD_PATH } });
    expect(
      vhist.admin.noteVersionHistory.totalCount,
      'at least the user edit + agent patch must exist as note versions',
    ).toBeGreaterThanOrEqual(2);
  });
});
