// @ts-check
/**
 * Fleet Kanban Vertical Slice — E2E headline test (Task C9).
 *
 * DevMode trip2g + real cmd/fleet + deterministic stub LLM.
 * Scenario: seed boards/sprint.md with a @status:doing card and roles/triage.md;
 * start the fleet; trigger a user edit that fires the change webhook; wait for the
 * agent to triage the card (@status:doing → @status:doing @triaged); assert:
 *   - exactly one fleet delivery fired (depth guard prevents re-trigger)
 *   - board content contains @status:doing @triaged
 *   - note versions >= 2 (user edit + agent patch)
 *
 * Consistency fix vs. plan draft: the plan seeded the card as @status:todo while
 * the stub/role targeted @status:doing, so the find would never match.  Here the
 * card is seeded as @status:doing from the start; the user edit re-saves the board
 * triggering the webhook; the stub finds @status:doing and replaces it.
 */

import { test, expect } from '@playwright/test';
import { spawn } from 'child_process';
import { unlinkSync } from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { signInAsAdmin } from './helpers/auth.js';
import { startStubLLM } from './helpers/stub-llm.js';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const GRAPHQL_URL = `${APP_URL}/_system/graphql`;
// REPO_ROOT is the worktree root — where go run ./cmd/fleet will be executed.
const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

// FLEET_CALLBACK_HOST controls the callback URL the fleet registers with trip2g.
// Pure-host run: 127.0.0.1 (default). Docker compose run (app in container,
// fleet on host): host.docker.internal — the app container resolves this to the
// host gateway via extra_hosts in docker-compose.test.yml.
const FLEET_CALLBACK_HOST = process.env.FLEET_CALLBACK_HOST || '127.0.0.1';

const BOARD_PATH = 'boards/sprint.md';
const ROLE_PATH = 'roles/triage.md';

// ---------------------------------------------------------------------------
// GraphQL helpers
// ---------------------------------------------------------------------------

/** Admin cookie auth (for queries/mutations under admin { ... }). */
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

/** API key auth (for updateNotes / notePaths which require X-Api-Key). */
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

/** Poll /debug/wait_all_jobs until all background jobs are drained. */
async function waitJobs(request) {
  const r = await request.get(`${APP_URL}/debug/wait_all_jobs`, { timeout: 60000 });
  expect(r.ok()).toBeTruthy();
}

// ---------------------------------------------------------------------------
// Test suite
// ---------------------------------------------------------------------------

test.describe.serial('Fleet kanban vertical slice', () => {
  // go build + fleet reconcile + agent triage: needs more than the default 30 s.
  test.setTimeout(120000);
  /** @type {string} Admin session cookie (trip2g_e2e=...). */
  let cookie;
  /** @type {string} Full-admin API key value (for X-Api-Key header). */
  let apiKey;
  /** @type {{ server: import('http').Server, port: number, calls: () => number }} */
  let stub;
  /** @type {import('child_process').ChildProcess} */
  let fleetProc;
  /** @type {string} Path to the pre-built fleet binary. */
  let fleetBin;

  // Seeded board content — card is already @status:doing so the stub's find
  // matches immediately when the fleet fires on the user edit.
  const boardSeed = '---\nlayout: kanban\n---\n\n## Doing\n- Fix login bug @status:doing\n';

  // Role note with Jet template body — exercises for_each templating end-to-end.
  // for_each: changed_files renders the body once per changed file with
  // change_file bound to that file's changeInfo. The stub LLM ignores the
  // rendered instruction text; what matters is that renderInstruction succeeds
  // (no Jet parse/exec error) and the handler invokes agentruntime.Run once.
  const roleSeed = [
    '---',
    'model: stub',
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

  test.beforeAll(async ({ browser, request }) => {
    // Extend this hook's timeout to cover go build + fleet reconcile.
    test.setTimeout(120000);
    // signInAsAdmin needs a Page, not a Browser — create one and close after.
    const page = await browser.newPage();
    cookie = await signInAsAdmin(page);
    await page.close();

    // -----------------------------------------------------------------------
    // Create a full-admin API key with MCP admin tools enabled.
    // The schema has no enableMcpAdminTools on CreateApiKeyInput; it requires
    // two separate calls: createApiKey → setApiKeyMcpAdminTools.
    // createApiKey returns value (the plaintext key) and apiKey.id.
    // -----------------------------------------------------------------------
    const keyData = await gqlAdmin(request, cookie, `
      mutation {
        admin {
          createApiKey(input: { description: "fleet-e2e" }) {
            ... on CreateApiKeyPayload { value apiKey { id } }
            ... on ErrorPayload { message }
          }
        }
      }
    `);
    apiKey = keyData.admin.createApiKey.value;
    const apiKeyId = keyData.admin.createApiKey.apiKey.id;

    // Enable MCP admin tools so the fleet can use the admin lane (/_system/mcp).
    await gqlAdmin(request, cookie, `
      mutation Set($input: SetApiKeyMcpAdminToolsInput!) {
        admin {
          setApiKeyMcpAdminTools(input: $input) {
            ... on SetApiKeyMcpAdminToolsPayload { apiKey { id } }
            ... on ErrorPayload { message }
          }
        }
      }
    `, { input: { id: apiKeyId, enabled: true } });

    // -----------------------------------------------------------------------
    // Seed the board and role notes.  The fleet isn't running yet so no
    // delivery is created here — seeding just establishes the initial versions.
    // -----------------------------------------------------------------------
    const upsertResult = await gqlApi(request, apiKey, `
      mutation Up($input: UpdateNotesInput!) {
        updateNotes(input: $input) {
          ... on UpdateNotesSuccessPayload { paths }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: {
        changes: [
          { upsert: { path: BOARD_PATH, content: boardSeed } },
          { upsert: { path: ROLE_PATH, content: roleSeed } },
        ],
      },
    });
    // Fail fast if the seed itself is rejected (e.g. scope enforcement error).
    expect(upsertResult.updateNotes.message).toBeFalsy();
    await waitJobs(request);

    // -----------------------------------------------------------------------
    // Pre-build the fleet binary so `go run` cold-compile time (~10-30 s) does
    // not eat into the reconcile poll budget.
    // -----------------------------------------------------------------------
    fleetBin = `/tmp/fleet-e2e-${process.pid}`;
    await new Promise((resolve, reject) => {
      const proc = spawn('go', ['build', '-o', fleetBin, './cmd/fleet'], {
        cwd: REPO_ROOT,
        stdio: 'pipe',
      });
      proc.stderr?.pipe(process.stderr);
      proc.on('close', (code) => {
        if (code !== 0) reject(new Error(`go build ./cmd/fleet exited ${code}`));
        else resolve();
      });
    });

    // -----------------------------------------------------------------------
    // Start the deterministic stub LLM.
    // Call 1 → patch_note({path, find: '@status:doing', replace: '@status:doing @triaged'})
    // Call 2+ → finish({answer: 'triaged'})
    // The find string IS present in the seeded board content.
    // -----------------------------------------------------------------------
    stub = await startStubLLM({
      path: BOARD_PATH,
      find: '@status:doing',
      replace: '@status:doing @triaged',
    });

    // -----------------------------------------------------------------------
    // Start cmd/fleet pointed at the DevMode server.
    // DevMode has SSRF guard off so loopback callbacks (127.0.0.1:9099) are allowed.
    // stdio: 'pipe' keeps fleet output out of the test runner unless debugging.
    // -----------------------------------------------------------------------
    fleetProc = spawn(
      fleetBin,
      [
        '-fleet-id', 'e2e',
        '-listen', '127.0.0.1:9099',
        '-callback-url', `http://${FLEET_CALLBACK_HOST}:9099`,
        '-trip2g-url', APP_URL,
        '-admin-api-key', apiKey,
        '-fleet-secret', 'e2e-secret',
        '-llm-base-url', `http://127.0.0.1:${stub.port}/v1`,
        '-llm-api-key', 'x',
        '-default-model', 'stub',
        '-agents-folder', 'roles/',
        '-poll-seconds', '2',
      ],
      { cwd: REPO_ROOT, stdio: 'pipe' },
    );
    fleetProc.stderr?.pipe(process.stderr);

    // -----------------------------------------------------------------------
    // Wait for the fleet to reconcile: it must have registered a change webhook
    // whose description matches the role note (fleet:<id>:<path>#<ver> format).
    // -----------------------------------------------------------------------
    await expect
      .poll(
        async () => {
          const d = await gqlAdmin(request, cookie, `
            query { admin { allChangeWebhooks { nodes { description } } } }
          `);
          return d.admin.allChangeWebhooks.nodes.some((n) =>
            n.description.startsWith('fleet:e2e:roles/triage.md#'),
          );
        },
        { timeout: 30000, intervals: [1000] },
      )
      .toBeTruthy();
  });

  test.afterAll(async () => {
    if (fleetProc) fleetProc.kill('SIGTERM');
    if (stub) stub.server.close();
    if (fleetBin) try { unlinkSync(fleetBin); } catch {}
  });

  test('user edit → agent triage → one delivery, board updated, no re-trigger', async ({ request }) => {
    // -----------------------------------------------------------------------
    // Simulate the user editing the board (e.g. adding another card while the
    // @status:doing card stays in place).  This updateNotes triggers the fleet's
    // change webhook.  The @status:doing string remains in the content so the
    // stub's find matches when the agent reads the board.
    // -----------------------------------------------------------------------
    const boardWithUserEdit =
      boardSeed + '- Write tests @status:todo\n';

    const editResult = await gqlApi(request, apiKey, `
      mutation Up($input: UpdateNotesInput!) {
        updateNotes(input: $input) {
          ... on UpdateNotesSuccessPayload { paths }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: {
        changes: [{ upsert: { path: BOARD_PATH, content: boardWithUserEdit } }],
      },
    });
    expect(editResult.updateNotes.message).toBeFalsy();
    await waitJobs(request);

    // -----------------------------------------------------------------------
    // Poll until the fleet agent writes back its triage patch.
    // -----------------------------------------------------------------------
    await expect
      .poll(
        async () => {
          const d = await gqlApi(request, apiKey, `
            query BoardContent($f: NotePathsFilter) { notePaths(filter: $f) { content } }
          `, { f: { paths: [BOARD_PATH] } });
          return d.notePaths[0]?.content ?? '';
        },
        { timeout: 60000, intervals: [1000] },
      )
      .toContain('@status:doing @triaged');

    // -----------------------------------------------------------------------
    // Assert exactly one fleet delivery (depth guard prevents a second run).
    // The fleet webhook description starts with 'fleet:e2e:roles/triage.md#'.
    // -----------------------------------------------------------------------
    const wh = await gqlAdmin(request, cookie, `
      query { admin { allChangeWebhooks { nodes { id description } } } }
    `);
    const fleetWebhook = wh.admin.allChangeWebhooks.nodes.find((n) =>
      n.description.startsWith('fleet:e2e:roles/triage.md#'),
    );
    expect(fleetWebhook, 'fleet webhook must be registered').toBeTruthy();

    const deliveries = await gqlAdmin(request, cookie, `
      query D($f: AdminChangeWebhookDeliveriesFilterInput!) {
        admin { changeWebhookDeliveries(filter: $f) { nodes { id status } } }
      }
    `, { f: { webhookId: fleetWebhook.id } });
    expect(
      deliveries.admin.changeWebhookDeliveries.nodes.length,
      'exactly one delivery — depth=1 agent write must not re-trigger',
    ).toBe(1);

    // -----------------------------------------------------------------------
    // Final content assertion (re-render oracle).
    // -----------------------------------------------------------------------
    const final = await gqlApi(request, apiKey, `
      query BoardContent($f: NotePathsFilter) { notePaths(filter: $f) { content } }
    `, { f: { paths: [BOARD_PATH] } });
    expect(final.notePaths[0].content).toContain('@status:doing @triaged');

    // -----------------------------------------------------------------------
    // Note version count: seed (v1) + user edit (v2) + agent patch (v3) = >=2
    // post-seed versions.  A soft >=2 is used since the exact seed version
    // count is environment-dependent.
    // -----------------------------------------------------------------------
    const vhist = await gqlAdmin(request, cookie, `
      query VH($f: AdminNoteVersionHistoryFilter!) {
        admin { noteVersionHistory(filter: $f) { totalCount nodes { versionId version } } }
      }
    `, { f: { path: BOARD_PATH } });
    expect(
      vhist.admin.noteVersionHistory.totalCount,
      'at least the user edit + agent patch must exist as note versions',
    ).toBeGreaterThanOrEqual(2);

    // TODO(C9): once AdminNoteVersionMeta exposes created_by_delivery_kind /
    // created_by_delivery_id (Task A2/M2 fleet attribution columns), assert
    // that the last version is attributed to the fleet delivery:
    //   const agentVersion = vhist.admin.noteVersionHistory.nodes.at(-1);
    //   expect(agentVersion.createdByDeliveryKind).toBe('change');
    //   expect(agentVersion.createdByDeliveryId).toBe(deliveries.admin.changeWebhookDeliveries.nodes[0].id);
    // These fields are in note_versions DB table (M2 migration) but are not
    // yet projected in the GraphQL schema (AdminNoteVersionMeta type).
  });
});
