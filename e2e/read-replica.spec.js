// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

/**
 * Read replica E2E.
 *
 * `app-replica` (docker-compose.test.yml) is a second trip2g process started with
 * LEADER_ADDR, sharing the SAME SQLite DB file as the leader (`app`). The shared
 * file is the "replication" — no LiteFS needed locally. The replica serves GET
 * locally and forwards every mutating request to the leader's internal intake
 * (app:20082) with an X-Replica-Auth HMAC. See docs/dev/readreplica.md.
 *
 * Ports: leader public 20081 / internal 20082, replica public 20071 / internal 20072.
 */
const LEADER_URL = process.env.APP_URL || 'http://localhost:20081';
const REPLICA_URL = process.env.REPLICA_URL || 'http://localhost:20071';
const LEADER_INTERNAL = process.env.LEADER_INTERNAL_URL || 'http://localhost:20082';

test.describe.serial('Read replica', () => {
  let replica;
  let leader;

  test.beforeAll(async ({ playwright }) => {
    replica = await playwright.request.newContext({ baseURL: REPLICA_URL });
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
    const [lr, rr] = await Promise.all([leader.get('/'), replica.get('/')]);
    expect(lr.status()).toBe(200);
    expect(rr.status()).toBe(200);
    const [lb, rb] = await Promise.all([lr.body(), rr.body()]);
    expect(rb.length).toBe(lb.length);
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
});
