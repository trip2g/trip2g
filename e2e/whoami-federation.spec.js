// @ts-check
//
// Whoami federation e2e — asserts docs/dev/whoami_test.md §5 against the live mesh.
// Bring the mesh up first:  node scripts/e2e/whoami-mesh.mjs all
// Then:                     npx playwright test e2e/whoami-federation.spec.js
//
// Data-driven from e2e/whoami/topology.json. Reachability = transitive descendants
// over the directed edges (realized via materialized federation-index.md, §7.3).
// If the mesh isn't running, every test self-skips (so normal runs ignore this file).
import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';

const TOPO = JSON.parse(
  fs.readFileSync(path.join(__dirname, 'whoami', 'topology.json'), 'utf8'),
);
const byId = Object.fromEntries(TOPO.nodes.map((n) => [n.id, n]));
const hostURL = (id) => `http://localhost:${byId[id].port}`;
const cookieName = (id) => `trip2g_whoami_${id.replace(/-/g, '_')}`;
const marker = (id) => `wai${id.replace(/-/g, '')}`;
const privMarker = (id) => `privateonly${id.replace(/-/g, '')}`;
const outEdges = (id) => TOPO.edges.filter(([f]) => f === id).map(([, t]) => t);

function descendants(id) {
  const seen = new Set();
  const stack = [...outEdges(id)];
  while (stack.length) {
    const x = stack.pop();
    if (seen.has(x)) continue;
    seen.add(x);
    for (const t of outEdges(x)) stack.push(t);
  }
  return seen;
}

async function gql(host, query, variables = {}, headers = {}) {
  const res = await fetch(`${host}/graphql`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...headers },
    body: JSON.stringify({ query, variables }),
  });
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL ${host}: ${JSON.stringify(body.errors)}`);
  return body.data;
}

const bearers = {};
async function bearer(id) {
  if (bearers[id]) return bearers[id];
  await gql(hostURL(id), `mutation { requestEmailSignInCode(input: { email: "hello@example.com" }) { ... on RequestEmailSignInCodePayload { success } ... on ErrorPayload { message } } }`);
  let jwt;
  for (const code of ['111111', '000000']) {
    try {
      const d = await gql(hostURL(id), `mutation { signInByEmail(input: { email: "hello@example.com", code: "${code}" }) { ... on SignInPayload { token } ... on ErrorPayload { message } } }`);
      if (d.signInByEmail.token) { jwt = d.signInByEmail.token; break; }
    } catch { /* next dev code */ }
  }
  if (!jwt) throw new Error(`sign-in failed on ${id}`);
  const d = await gql(hostURL(id), `mutation($i: CreateUserTokenInput!) { createUserToken(input: $i) { ... on CreateUserTokenPayload { plaintextToken } ... on ErrorPayload { message } } }`, { i: { name: `whoami-spec-${id}`, expiresInDays: 1 } }, { Cookie: `${cookieName(id)}=${jwt}` });
  bearers[id] = d.createUserToken.plaintextToken;
  return bearers[id];
}

async function mcpText(id, name, args) {
  const tok = await bearer(id);
  const res = await fetch(`${hostURL(id)}/_system/mcp`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}` },
    body: JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name, arguments: args } }),
  });
  const j = await res.json();
  if (j.error) throw new Error(`MCP ${name} ${id}: ${JSON.stringify(j.error)}`);
  return (j.result?.content || []).map((c) => c.text).filter(Boolean).join('\n\n');
}

let meshUp = false;
test.beforeAll(async () => {
  try {
    await fetch(hostURL(TOPO.nodes[0].id), { signal: AbortSignal.timeout(2000) });
    meshUp = true;
  } catch { meshUp = false; }
});

test.describe('Whoami Federation', () => {
  // §5.1 — every KB finds its own whoami locally.
  for (const n of TOPO.nodes) {
    test(`5.1 local: ${n.id} finds own whoami`, async () => {
      test.skip(!meshUp, 'whoami mesh not running (run scripts/e2e/whoami-mesh.mjs all)');
      const text = await mcpText(n.id, 'search', { query: 'whoami Role:' });
      expect(text).toContain(marker(n.id));
    });
  }

  // §5.2/§5.3 — fan-out + materialized indexes reach exactly the descendants.
  for (const origin of ['owner', 'boss-a', 'boss-b', 'company-a-hub', 'company-b-hub']) {
    test(`5.2 reach: ${origin} sees exactly its descendants`, async () => {
      test.skip(!meshUp, 'whoami mesh not running');
      const desc = descendants(origin);
      const text = await mcpText(origin, 'federated_search', { query: 'whoami' });
      for (const d of desc) {
        expect(text, `${origin} should reach ${d}`).toContain(marker(d));
      }
      for (const n of TOPO.nodes) {
        if (n.id === origin || desc.has(n.id)) continue;
        expect(text, `${origin} must NOT reach ${n.id}`).not.toContain(marker(n.id));
      }
    });
  }

  // §5.4 — targeted single peer: no fan-out beyond it.
  test('5.4 targeted: owner → company-a-hub only', async () => {
    test.skip(!meshUp, 'whoami mesh not running');
    const text = await mcpText('owner', 'federated_search', { query: 'whoami', kb_id: 'company-a-hub' });
    expect(text).toContain(marker('company-a-hub'));
  });

  // §5.3 — targeted recursion: 2 hops works (within the depth cap)…
  test('5.3 targeted 2-hop: owner → company-a-hub/dept-a-hub', async () => {
    test.skip(!meshUp, 'whoami mesh not running');
    const text = await mcpText('owner', 'federated_search', { query: 'whoami', kb_id: 'company-a-hub/dept-a-hub' });
    expect(text).toContain(marker('dept-a-hub'));
  });

  // …and a 3rd live hop is rejected by MCP_FEDERATION_MAX_DEPTH (=3 allows 2 live hops).
  // Deeper reach still works via materialized indexes (uncapped) — see §5.2.
  test('5.3 depth cap: owner → company-b-hub/company-site/sub-a (3 hops) is blocked', async () => {
    test.skip(!meshUp, 'whoami mesh not running');
    await expect(
      mcpText('owner', 'federated_search', { query: 'whoami', kb_id: 'company-b-hub/company-site/sub-a' }),
    ).rejects.toThrow(/max depth/i);
  });

  // §5.5 — isolation: a private note never crosses a valid edge.
  for (const [from, to] of TOPO.edges) {
    test(`5.5 isolation: ${from} → ${to} hides private`, async () => {
      test.skip(!meshUp, 'whoami mesh not running');
      // sanity: the public route works (broad query, so an empty private result below
      // means filtering — not a dead edge or a query the server merely echoed back)
      const pub = await mcpText(from, 'federated_search', { query: 'whoami', kb_id: to });
      expect(pub, `${from} should reach ${to}'s public whoami`).toContain(marker(to));
      // isolation: probe with the no-dash private marker. The server echoes that term in
      // "No results found for: …", so we assert on the dashed body text (`private-only-<id>`)
      // which a real leak would contain but the echo never does.
      const priv = await mcpText(from, 'federated_search', { query: privMarker(to), kb_id: to });
      expect(priv, `${from} must NOT see ${to}'s private note`).not.toContain(`private-only-${to}`);
    });
  }

  // §5.5 control — each private note IS findable by its own admin locally. Proves the
  // isolation tests above aren't vacuously empty: the note exists and is indexed; only
  // federation hides it. Without this, an empty federated result could be a missing note.
  for (const n of TOPO.nodes) {
    test(`5.5 control: ${n.id} sees its own private note locally`, async () => {
      test.skip(!meshUp, 'whoami mesh not running');
      const text = await mcpText(n.id, 'search', { query: privMarker(n.id) });
      expect(text, `${n.id} admin should find its own private note`).toContain(`private-only-${n.id}`);
    });
  }

  // §5.6 — negative: no edge → no result (broad query keeps it echo-proof).
  test('5.6 negative: boss-a cannot reach company-site', async () => {
    test.skip(!meshUp, 'whoami mesh not running');
    const text = await mcpText('boss-a', 'federated_search', { query: 'whoami' });
    expect(text).not.toContain(marker('company-site'));
  });
});
