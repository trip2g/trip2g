// @ts-check
import { test, expect } from '@playwright/test';
import { signInAsAdmin } from './helpers/auth.js';

/**
 * Background Queue E2E Tests
 *
 * Uses debug_sleep_job — a devMode-only job that sleeps N ms then writes to
 * an in-memory log accessible via /debug/completed_jobs.
 *
 * Covers:
 * 1. Jobs complete and appear in the completed log
 * 2. Stopped queue accumulates pending jobs without running them
 * 3. Clear removes all pending jobs
 * 4. Start resumes processing after stop
 */

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const GRAPHQL_URL = `${APP_URL}/_system/graphql`;
const QUEUE_ID = 'global_jobs';

// ── helpers ──────────────────────────────────────────────────────────────────

async function graphqlAdmin(request, cookie, query, variables = {}) {
  const res = await request.post(GRAPHQL_URL, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

async function spawnJobs(request, { count = 1, durationMs = 0, tag = '' } = {}) {
  const res = await request.post(
    `${APP_URL}/debug/spawn_jobs?count=${count}&durationMs=${durationMs}&tag=${encodeURIComponent(tag)}`,
  );
  expect(res.ok()).toBeTruthy();
}

async function completedJobs(request) {
  const res = await request.get(`${APP_URL}/debug/completed_jobs`);
  expect(res.ok()).toBeTruthy();
  return /** @type {Array<{tag:string,durationMs:number,completedAt:string}>} */ (await res.json());
}

async function clearCompletedJobs(request) {
  await request.delete(`${APP_URL}/debug/completed_jobs`);
}

async function waitAllJobs(request) {
  const res = await request.get(`${APP_URL}/debug/wait_all_jobs?interval=500`, { timeout: 120_000 });
  expect(res.ok()).toBeTruthy();
}

async function getQueueStats(request, cookie) {
  const data = await graphqlAdmin(request, cookie, `
    query GetQueueStats($id: String!) {
      admin {
        backgroundQueue(id: $id) {
          pendingCount
          stopped
        }
      }
    }
  `, { id: QUEUE_ID });
  return data.admin.backgroundQueue;
}

async function stopQueue(request, cookie) {
  await graphqlAdmin(request, cookie, `
    mutation StopQueue($input: StopBackgroundQueueInput!) {
      admin {
        stopBackgroundQueue(input: $input) {
          ... on StopBackgroundQueuePayload { queues { id stopped } }
          ... on ErrorPayload { message }
        }
      }
    }
  `, { input: { id: QUEUE_ID } });
}

async function startQueue(request, cookie) {
  await graphqlAdmin(request, cookie, `
    mutation StartQueue($input: StartBackgroundQueueInput!) {
      admin {
        startBackgroundQueue(input: $input) {
          ... on StartBackgroundQueuePayload { queues { id stopped } }
          ... on ErrorPayload { message }
        }
      }
    }
  `, { input: { id: QUEUE_ID } });
}

async function clearQueue(request, cookie) {
  await graphqlAdmin(request, cookie, `
    mutation ClearQueue($input: ClearBackgroundQueueInput!) {
      admin {
        clearBackgroundQueue(input: $input) {
          ... on ClearBackgroundQueuePayload { deletedCount }
          ... on ErrorPayload { message }
        }
      }
    }
  `, { input: { id: QUEUE_ID } });
}

// ── tests ─────────────────────────────────────────────────────────────────────

test.describe.serial('background queue', () => {
  let cookie;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    cookie = await signInAsAdmin(page);
    await page.close();
  });

  test.beforeEach(async ({ request }) => {
    // Clean slate: stop, wipe all pending/failed jobs, restart, clear log.
    await stopQueue(request, cookie);
    await clearQueue(request, cookie);
    await startQueue(request, cookie);
    await clearCompletedJobs(request);
  });

  test.afterEach(async ({ request }) => {
    // Leave queue running so other tests/server aren't affected.
    await startQueue(request, cookie);
  });

  test('spawned jobs complete and appear in the log', async ({ request }) => {
    const tag = `run-${Date.now()}`;
    await spawnJobs(request, { count: 5, durationMs: 200, tag });
    await waitAllJobs(request);

    const log = await completedJobs(request);
    const mine = log.filter(r => r.tag === tag);
    expect(mine).toHaveLength(5);
    for (const r of mine) {
      expect(r.durationMs).toBe(200);
      expect(new Date(r.completedAt).getTime()).toBeGreaterThan(0);
    }
  });

  test('single job with unique tag appears in log', async ({ request }) => {
    const tag = `single-${Date.now()}`;
    await spawnJobs(request, { count: 1, durationMs: 500, tag });
    await waitAllJobs(request);

    const log = await completedJobs(request);
    expect(log.find(r => r.tag === tag)).toBeDefined();
  });

  test('stopped queue accumulates pending jobs without running them', async ({ request }) => {
    await stopQueue(request, cookie);

    const statsBefore = await getQueueStats(request, cookie);
    expect(statsBefore.stopped).toBe(true);

    await spawnJobs(request, { count: 50, durationMs: 100, tag: 'pending-test' });

    // Give the runner a moment — jobs must NOT execute while stopped.
    await new Promise(r => setTimeout(r, 500));

    const statsAfter = await getQueueStats(request, cookie);
    expect(statsAfter.pendingCount).toBeGreaterThanOrEqual(50);

    const log = await completedJobs(request);
    expect(log.filter(r => r.tag === 'pending-test')).toHaveLength(0);
  });

  test('clear removes all pending jobs from a stopped queue', async ({ request }) => {
    await stopQueue(request, cookie);
    await spawnJobs(request, { count: 50, durationMs: 0, tag: 'clear-test' });

    const statsBefore = await getQueueStats(request, cookie);
    expect(statsBefore.pendingCount).toBeGreaterThanOrEqual(50);

    await clearQueue(request, cookie);

    const statsAfter = await getQueueStats(request, cookie);
    expect(statsAfter.pendingCount).toBe(0);
  });

  test('start resumes processing after stop', async ({ request }) => {
    const tag = `resume-${Date.now()}`;

    await stopQueue(request, cookie);
    await spawnJobs(request, { count: 10, durationMs: 100, tag });

    const statsStopped = await getQueueStats(request, cookie);
    expect(statsStopped.pendingCount).toBeGreaterThanOrEqual(10);

    await startQueue(request, cookie);
    await waitAllJobs(request);

    const log = await completedJobs(request);
    expect(log.filter(r => r.tag === tag)).toHaveLength(10);
  });
});
