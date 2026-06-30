// @ts-check
/**
 * Fleet LLM pipeline — E2E deterministic smoke test.
 *
 * Architecture under test:
 *   trip2g app (Docker) ← change webhook delivery →
 *   fleet (Docker)      ← chat completions →
 *   llm-mock (Docker)   → write_note tool call →
 *   fleet               → updateNotes via scoped token →
 *   trip2g app
 *
 * Scenario:
 *   1. Seed roles/transcript-agent.md (fleet role) and transcripts/sample.md.
 *   2. Fleet discovers the role, reconciles a change webhook registered against
 *      the app (poll_seconds=3, so within ~3 s).
 *   3. User updates transcripts/sample.md → app fires the change webhook → fleet
 *      delivers → llm-mock returns write_note({path:"segments/sample.md",
 *      content:"processed: Begin."}) → fleet writes it back via scoped token.
 *   4. Spec polls notePaths for segments/sample.md until content contains
 *      "processed: ", then asserts.
 *
 * Fleet and llm-mock run as Docker compose services (see docker-compose.test.yml).
 * No subprocess is spawned here. APP_URL is the host-published app port.
 */

import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const GRAPHQL_URL = `${APP_URL}/_system/graphql`;

const TRANSCRIPT_PATH = 'transcripts/sample.md';
const ROLE_PATH = 'roles/transcript-agent.md';
const SEGMENT_PATH = 'segments/sample.md';

// ---------------------------------------------------------------------------
// GraphQL helpers (same pattern as fleet-kanban.spec.js)
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

test.describe.serial('Fleet LLM pipeline (dockerized)', () => {
  // Sign-in + seed + 30 s fleet-reconcile poll easily exceeds the default 30 s.
  test.setTimeout(120000);

  /** @type {string} Admin session cookie (trip2g_e2e=...). */
  let cookie;
  /** @type {string} Full-admin API key for updateNotes / notePaths queries. */
  let apiKey;

  // Transcript note seeded before the trigger.
  const transcriptSeed = [
    '# Sample Transcript',
    '',
    'Speaker A: Hello.',
    'Speaker B: World.',
  ].join('\n') + '\n';

  // Role note: triggers on transcript updates, writes a processed segment.
  // write_note is the only tool needed — the llm-mock returns it on the first
  // LLM call and finish on the second, so no read_note/search is exercised.
  const roleSeed = [
    '---',
    'model: mock',
    'tools: [write_note]',
    'read_patterns: ["transcripts/**"]',
    'write_patterns: ["segments/**"]',
    'mode: change',
    'trigger_on: [create, update]',
    'trigger_include: ["transcripts/**"]',
    'max_depth: 1',
    'for_each: changed_files',
    '---',
    'Process the transcript at {{ change_file.Path }}:',
    '',
    '{{ change_file.Content }}',
    '',
    'Write your analysis to segments/sample.md.',
  ].join('\n');

  test.beforeAll({ timeout: 90000 }, async ({ request }) => {
    // Use the API-based sign-in (faster than browser UI, no locator waiting).
    // graphqlSignIn returns a raw JWT; construct the cookie header from it.
    const token = await graphqlSignIn(request, 'hello@example.com', '111111', { useCache: false });
    const cookieName = process.env.USER_TOKEN_COOKIE_NAME || 'trip2g_e2e';
    cookie = `${cookieName}=${token}`;

    // Create a full-admin API key for note read/write.
    const keyData = await gqlAdmin(request, cookie, `
      mutation {
        admin {
          createApiKey(input: { description: "fleet-llm-e2e" }) {
            ... on CreateApiKeyPayload { value apiKey { id } }
            ... on ErrorPayload { message }
          }
        }
      }
    `);
    expect(keyData.admin.createApiKey.message).toBeFalsy();
    apiKey = keyData.admin.createApiKey.value;

    // Seed only the role note.  The transcript is seeded in the test function
    // so that it appears AFTER the fleet's webhook is registered — keeping the
    // delivery count clean (one trigger, one delivery).
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

    // Wait for the fleet to discover roles/transcript-agent.md and register a
    // change webhook whose description matches fleet:<id>:<role-path>#<ver>.
    // Fleet polls every 3 s (TRIP2G_POLL_SECONDS=3 in compose); allow 30 s.
    await expect
      .poll(
        async () => {
          const d = await gqlAdmin(request, cookie, `
            query { admin { allChangeWebhooks { nodes { description } } } }
          `);
          return d.admin.allChangeWebhooks.nodes.some(n =>
            n.description.startsWith('fleet:e2e:roles/transcript-agent.md#'),
          );
        },
        { timeout: 30000, intervals: [1000] },
      )
      .toBeTruthy();
  });

  test('transcript update → llm-mock write_note → segments/sample.md contains "processed: "', async ({ request }) => {
    // Locate the fleet webhook (registered in beforeAll).
    const wh = await gqlAdmin(request, cookie, `
      query { admin { allChangeWebhooks { nodes { id description } } } }
    `);
    const fleetWebhook = wh.admin.allChangeWebhooks.nodes.find(n =>
      n.description.startsWith('fleet:e2e:roles/transcript-agent.md#'),
    );
    expect(fleetWebhook, 'fleet webhook must be registered').toBeTruthy();

    // Snapshot delivery count BEFORE the trigger so repeated runs don't
    // accumulate stale history into the assertion.
    const before = await gqlAdmin(request, cookie, `
      query D($f: AdminChangeWebhookDeliveriesFilterInput!) {
        admin { changeWebhookDeliveries(filter: $f) { nodes { id } } }
      }
    `, { f: { webhookId: fleetWebhook.id } });
    const deliveryCountBefore = before.admin.changeWebhookDeliveries.nodes.length;

    // Use a unique run marker so the poll below waits for FRESH content
    // rather than stale content from a previous test run.
    const runId = String(Date.now());
    const updated = transcriptSeed + `\nRun: ${runId}\n`;

    const editResult = await gqlApi(request, apiKey, `
      mutation Up($input: UpdateNotesInput!) {
        updateNotes(input: $input) {
          ... on UpdateNotesSuccessPayload { paths }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: { changes: [{ upsert: { path: TRANSCRIPT_PATH, content: updated } }] },
    });
    expect(editResult.updateNotes.message).toBeFalsy();
    await waitJobs(request);

    // Poll until the fleet agent writes segments/sample.md via the mock LLM.
    // llm-mock sees the runId in the last user message and writes:
    //   write_note({path:"segments/sample.md", content:"processed: <msg>"})
    // Poll checks for BOTH markers so stale content from previous runs is skipped.
    await expect
      .poll(
        async () => {
          const d = await gqlApi(request, apiKey, `
            query N($f: NotePathsFilter) { notePaths(filter: $f) { content } }
          `, { f: { paths: [SEGMENT_PATH] } });
          const content = d.notePaths[0]?.content ?? '';
          return content.includes('processed: ') && content.includes(runId);
        },
        { timeout: 60000, intervals: [2000] },
      )
      .toBeTruthy();

    // Confirm at least one new delivery fired for this run.
    // Depth guard (max_depth:1) + trigger_include:["transcripts/**"] ensures
    // the write-back to segments/ does NOT re-trigger the webhook.
    const after = await gqlAdmin(request, cookie, `
      query D($f: AdminChangeWebhookDeliveriesFilterInput!) {
        admin { changeWebhookDeliveries(filter: $f) { nodes { id status } } }
      }
    `, { f: { webhookId: fleetWebhook.id } });
    const deliveryCountAfter = after.admin.changeWebhookDeliveries.nodes.length;

    expect(
      deliveryCountAfter - deliveryCountBefore,
      'exactly one new delivery this run — write-back does not re-trigger',
    ).toBe(1);
  });
});
