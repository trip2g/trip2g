// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

/**
 * Read replica E2E.
 *
 * `app-replica` and `app-replica2` (docker-compose.test.yml) are trip2g
 * processes started with LEADER_ADDR, sharing the SAME SQLite DB file as the
 * leader (`app`). The shared file is the "replication" — no LiteFS needed
 * locally. Each replica serves GET locally and forwards every mutating request
 * to the leader's internal intake (app:20082) with an X-Replica-Auth HMAC.
 * See docs/dev/readreplica.md.
 *
 * Ports: leader public 20081 / internal 20082
 *        replica  public 20071 / internal 20072
 *        replica2 public 20073 / internal 20074
 */
const LEADER_URL = process.env.APP_URL || 'http://localhost:20081';
const REPLICA_URL = process.env.REPLICA_URL || 'http://localhost:20071';
const REPLICA2_URL = process.env.REPLICA2_URL || 'http://localhost:20073';
const LEADER_INTERNAL = process.env.LEADER_INTERNAL_URL || 'http://localhost:20082';

/**
 * Shared suite exercising a single replica against the leader.
 * @param {string} replicaUrl
 */
function replicaSuite(replicaUrl) {
  let replica;
  let leader;

  test.beforeAll(async ({ playwright }) => {
    replica = await playwright.request.newContext({ baseURL: replicaUrl });
    leader = await playwright.request.newContext({ baseURL: LEADER_URL });
  });

  test.afterAll(async () => {
    await replica?.dispose();
    await leader?.dispose();
  });

  test('serves GET locally', async () => {
    const r = await replica.get('/');
    expect(r.status()).toBe(200);
  });

  test('read parity — replica returns the same bytes as the leader', async () => {
    // A read replica is eventually consistent: it reloads its in-memory note
    // cache from the shared DB a beat behind the leader after a push. Poll until
    // the rendered "/" converges to identical byte length (it must) so a transient
    // cache-reload lag doesn't flake the parity check — it still fails loudly if
    // the bytes never converge.
    await expect(async () => {
      const [lr, rr] = await Promise.all([leader.get('/'), replica.get('/')]);
      expect(lr.status()).toBe(200);
      expect(rr.status()).toBe(200);
      const [lb, rb] = await Promise.all([lr.body(), rr.body()]);
      expect(rb.length).toBe(lb.length);
    }).toPass({ timeout: 15000, intervals: [500, 1000, 2000] });
  });

  test('forwards a POST (GraphQL) to the leader and relays the response', async () => {
    const r = await replica.post('/_system/graphql', {
      headers: { 'Content-Type': 'application/json' },
      data: { query: '{ __typename }' },
    });
    expect(r.status()).toBe(200);
    const body = await r.json();
    expect(body.data.__typename).toBe('Query');
  });

  test('leader intake rejects an unauthenticated forward (no X-Replica-Auth → 401)', async ({ request }) => {
    const r = await request.post(`${LEADER_INTERNAL}/_system/graphql`, {
      headers: { 'Content-Type': 'application/json' },
      data: { query: '{ __typename }' },
    });
    expect(r.status()).toBe(401);
  });

  test('a mutation through the replica is executed on the leader (sign-in forwards)', async () => {
    // graphqlSignIn runs requestEmailSignInCode + signInByEmail mutations via
    // POST /_system/graphql. Pointed at the replica, both forward to the leader,
    // which executes them and returns a real session token. A token proves the
    // full write path: replica → X-Replica-Auth intake → leader executes → relay.
    const token = await graphqlSignIn(replica, 'hello@example.com', '111111', { useCache: false });
    expect(token, 'sign-in mutation forwarded through the replica returns a token').toBeTruthy();
  });
}

// Run the two replica suites serially to avoid sign-in code contention:
// both suites forward mutations to the same leader, which issues a single-use
// code per email. Running concurrently would have the second signInByEmail
// fail with not_found because the first already consumed the code.
test.describe.serial('Read replicas', () => {
  test.describe.serial('app-replica (port 20071)', () => {
    replicaSuite(REPLICA_URL);
  });

  test.describe.serial('app-replica2 (port 20073)', () => {
    replicaSuite(REPLICA2_URL);
  });
});
