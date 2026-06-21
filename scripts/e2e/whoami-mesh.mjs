// Whoami federation mesh — json-driven linker for the whoami e2e.
//
// Single source of truth: e2e/whoami/topology.json (mirrors docs/dev/whoami_test.md).
// This script GENERATES the compose stack, SEEDS notes, WIRES the federation edges,
// and MATERIALIZES federation-index.md bottom-up — so a 1-level fan-out surfaces
// grandchildren via the indexes (see whoami_test.md §7.3). No new deps: plain Node
// + fetch, same style as scripts/bench-pushnotes.mjs.
//
// Usage:
//   node scripts/e2e/whoami-mesh.mjs gen          # write docker-compose.whoami.yml
//   node scripts/e2e/whoami-mesh.mjs up           # build image + compose up --wait
//   node scripts/e2e/whoami-mesh.mjs seed         # push whoami/private/kb notes
//   node scripts/e2e/whoami-mesh.mjs wire         # create inbound+outbound secrets
//   node scripts/e2e/whoami-mesh.mjs materialize  # build federation-index.md bottom-up
//   node scripts/e2e/whoami-mesh.mjs all          # up + seed + wire + materialize
//   node scripts/e2e/whoami-mesh.mjs down         # compose down -v
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const TOPO = JSON.parse(fs.readFileSync(path.join(ROOT, 'e2e/whoami/topology.json'), 'utf8'));
const COMPOSE_FILE = path.join(ROOT, 'docker-compose.whoami.yml');
const IMAGE = 'trip2g-whoami:local';
const PROJECT = 'trip2g_whoami_env';
const MINIO_PORT = 29010;

// ── derived topology helpers ────────────────────────────────────────────────
const byId = Object.fromEntries(TOPO.nodes.map((n) => [n.id, n]));
const service = (id) => `whoami-${id}`;
const cookieName = (id) => `trip2g_whoami_${id.replace(/-/g, '_')}`;
const internalURL = (id) => `http://${service(id)}:${byId[id].port}`;
const mcpURL = (id) => `${internalURL(id)}/_system/mcp`;
const hostURL = (id) => `http://localhost:${byId[id].port}`;
const outEdges = (id) => TOPO.edges.filter(([f]) => f === id).map(([, t]) => t);
const nodash = (id) => id.replace(/-/g, '');
const log = (m) => console.log(`· ${m}`);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// ── GraphQL / MCP over fetch (retries while a node is warming up) ────────────
async function gql(host, query, variables = {}, headers = {}) {
  let lastErr;
  for (let attempt = 0; attempt < 40; attempt++) {
    try {
      const res = await fetch(`${host}/graphql`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...headers },
        body: JSON.stringify({ query, variables }),
      });
      const body = await res.json();
      if (body.errors) throw new Error(`GraphQL ${host}: ${JSON.stringify(body.errors)}`);
      return body.data;
    } catch (e) {
      lastErr = e;
      const transient = e.cause?.code === 'ECONNREFUSED' || String(e).includes('fetch failed');
      if (transient) { await sleep(1000); continue; }
      throw e;
    }
  }
  throw lastErr;
}

async function mcp(host, name, args, bearer) {
  const headers = { 'Content-Type': 'application/json' };
  if (bearer) headers['Authorization'] = `Bearer ${bearer}`;
  const res = await fetch(`${host}/_system/mcp`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name, arguments: args } }),
  });
  if (!res.ok) throw new Error(`MCP ${name} ${host}: HTTP ${res.status}`);
  return res.json();
}

// ── per-node session (jwt → cookie, api key, bearer token), memoised ─────────
const sessions = {};
async function session(id) {
  if (sessions[id]) return sessions[id];
  await gql(hostURL(id), `mutation { requestEmailSignInCode(input: { email: "hello@example.com" }) { ... on RequestEmailSignInCodePayload { success } ... on ErrorPayload { message } } }`);
  let jwt;
  for (const code of ['111111', '000000']) {
    try {
      const d = await gql(hostURL(id), `mutation { signInByEmail(input: { email: "hello@example.com", code: "${code}" }) { ... on SignInPayload { token } ... on ErrorPayload { message } } }`);
      if (d.signInByEmail.token) { jwt = d.signInByEmail.token; break; }
    } catch { /* try next dev code */ }
  }
  if (!jwt) throw new Error(`sign-in failed on ${id}`);
  sessions[id] = { jwt, cookie: `${cookieName(id)}=${jwt}` };
  return sessions[id];
}

async function apiKey(id) {
  const s = await session(id);
  if (s.apiKey) return s.apiKey;
  const d = await gql(hostURL(id), `mutation { admin { createApiKey(input: { description: "whoami-e2e" }) { ... on CreateApiKeyPayload { value } ... on ErrorPayload { message } } } }`, {}, { Cookie: s.cookie });
  const v = d.admin.createApiKey.value;
  if (!v) throw new Error(`createApiKey failed on ${id}: ${JSON.stringify(d)}`);
  s.apiKey = v;
  return v;
}

async function bearer(id) {
  const s = await session(id);
  if (s.bearer) return s.bearer;
  const d = await gql(hostURL(id), `mutation($i: CreateUserTokenInput!) { createUserToken(input: $i) { ... on CreateUserTokenPayload { plaintextToken } ... on ErrorPayload { message } } }`, { i: { name: `whoami-e2e-${id}`, expiresInDays: 1 } }, { Cookie: s.cookie });
  const t = d.createUserToken.plaintextToken;
  if (!t) throw new Error(`createUserToken failed on ${id}: ${JSON.stringify(d)}`);
  s.bearer = t;
  return t;
}

async function push(id, updates) {
  const key = await apiKey(id);
  const d = await gql(hostURL(id), `mutation Push($input: PushNotesInput!) { pushNotes(input: $input) { __typename ... on ErrorPayload { message } } }`, { input: { updates } }, { 'X-API-Key': key });
  if (d.pushNotes.__typename === 'ErrorPayload') throw new Error(`pushNotes ${id}: ${d.pushNotes.message}`);
}

// ── note bodies ─────────────────────────────────────────────────────────────
function whoamiNote(id) {
  const nb = outEdges(id);
  return [
    '---', `subgraph: ${TOPO.publicSubgraph}`, `title: whoami ${id}`, '---', '',
    '# whoami', '',
    `wai${nodash(id)} whoami Role: ${id} KB: ${id}`,
    `Neighbours: ${nb.length ? nb.join(', ') : '(leaf)'}`, '',
  ].join('\n');
}
function privateNote(id) {
  return [
    '---', `subgraph: ${TOPO.privateSubgraph}`, `title: private ${id}`, '---', '',
    '# private', '', `privateonly${nodash(id)} private-only-${id}`, '',
  ].join('\n');
}
function indexNote(id, body = '') {
  return [
    '---', `subgraph: ${TOPO.publicSubgraph}`, `title: federation-index ${id}`, '---', '',
    '# federation-index', '', `Hub: ${id}`, '', body, '',
  ].join('\n');
}
function kbNote(from, to) {
  return [
    '---', `mcp_federation_kb_url: ${mcpURL(to)}`, `mcp_federation_kb_id: ${to}`,
    'free: true', `title: kb ${to}`, '---', '', `# kb ${to}`, '',
    `Federation route ${from} to ${to}.`, '',
  ].join('\n');
}

// ── phases ──────────────────────────────────────────────────────────────────
async function seed() {
  for (const n of TOPO.nodes) {
    const updates = [
      { path: 'whoami.md', content: whoamiNote(n.id) },
      { path: 'private.md', content: privateNote(n.id) },
    ];
    if (n.hub) updates.push({ path: 'federation-index.md', content: indexNote(n.id) });
    for (const to of outEdges(n.id)) updates.push({ path: `kb/${to}.md`, content: kbNote(n.id, to) });
    await push(n.id, updates);
    log(`seeded ${n.id}: ${updates.map((u) => u.path).join(', ')}`);
  }
}

async function findSubgraphId(id, name) {
  const s = await session(id);
  const d = await gql(hostURL(id), `query { admin { allSubgraphs { nodes { id name } } } }`, {}, { Cookie: s.cookie });
  return d.admin.allSubgraphs.nodes.find((x) => x.name === name)?.id;
}

async function wire() {
  for (const [from, to] of TOPO.edges) {
    const kid = `whoami-${from}-${to}-${crypto.randomBytes(3).toString('hex')}`;
    const secretHex = crypto.randomBytes(32).toString('hex');
    const toS = await session(to);
    // inbound secret on the target, scoped to the public subgraph only
    const inb = await gql(hostURL(to), `mutation($i: CreateInboundFederationSecretInput!) { admin { createInboundFederationSecret(input: $i) { ... on CreateInboundFederationSecretPayload { id } ... on ErrorPayload { message } } } }`, { i: { kid, secretHex } }, { Cookie: toS.cookie });
    if (inb.admin.createInboundFederationSecret.message) throw new Error(`inbound ${to}: ${inb.admin.createInboundFederationSecret.message}`);
    const pubId = await findSubgraphId(to, TOPO.publicSubgraph);
    if (!pubId) throw new Error(`no '${TOPO.publicSubgraph}' subgraph on ${to} — was whoami.md seeded first?`);
    const add = await gql(hostURL(to), `mutation($i: AddFederationSecretSubgraphInput!) { admin { addFederationSecretSubgraph(input: $i) { ... on AddFederationSecretSubgraphPayload { success } ... on ErrorPayload { message } } } }`, { i: { kid, subgraphID: pubId } }, { Cookie: toS.cookie });
    if (add.admin.addFederationSecretSubgraph.message) throw new Error(`scope ${to}: ${add.admin.addFederationSecretSubgraph.message}`);
    // outbound secret on the source
    const fromS = await session(from);
    const out = await gql(hostURL(from), `mutation($i: CreateOutboundFederationSecretInput!) { admin { createOutboundFederationSecret(input: $i) { ... on CreateOutboundFederationSecretPayload { id } ... on ErrorPayload { message } } } }`, { i: { kid, secretHex, kbURL: mcpURL(to) } }, { Cookie: fromS.cookie });
    if (out.admin.createOutboundFederationSecret.message) throw new Error(`outbound ${from}: ${out.admin.createOutboundFederationSecret.message}`);
    log(`wired ${from} → ${to}`);
  }
}

// Build hubs bottom-up: a parent's index must be (re)built after its children's.
function hubBuildOrder() {
  const memo = {};
  const depth = (id) => {
    if (id in memo) return memo[id];
    const outs = outEdges(id);
    memo[id] = outs.length ? 1 + Math.max(...outs.map(depth)) : 0;
    return memo[id];
  };
  return TOPO.nodes.filter((n) => n.hub).sort((a, b) => depth(a.id) - depth(b.id));
}

async function materialize() {
  for (const n of hubBuildOrder()) {
    const tok = await bearer(n.id);
    const res = await mcp(hostURL(n.id), 'federated_search', { query: 'whoami' }, tok);
    const text = (res.result?.content || []).map((c) => c.text).filter(Boolean).join('\n');
    // Compact + snippet-safe: one early line listing every reachable marker, so a
    // parent's 1-level search *snippet* of this index carries the full reach (§7.3).
    // Markers propagate bottom-up because each child's index is itself this one line.
    const markers = [...new Set(text.match(/wai[a-z0-9]+/g) || [])].sort();
    await push(n.id, [{ path: 'federation-index.md', content: indexNote(n.id, `whoami reach: ${markers.join(' ')}`) }]);
    log(`materialized federation-index for ${n.id} (${markers.length} markers: ${markers.join(' ')})`);
  }
}

// ── compose generation (JSON is valid YAML; docker compose reads it) ─────────
function composeDoc() {
  const services = {
    minio: {
      image: 'minio/minio:latest',
      ports: [`${MINIO_PORT}:${MINIO_PORT}`, `${MINIO_PORT + 1}:${MINIO_PORT + 1}`],
      environment: { MINIO_ROOT_USER: 'testuser', MINIO_ROOT_PASSWORD: 'testpassword' },
      command: `server /data --console-address ":${MINIO_PORT + 1}" --address ":${MINIO_PORT}"`,
      healthcheck: { test: ['CMD', 'curl', '-f', `http://localhost:${MINIO_PORT}/minio/health/live`], interval: '5s', timeout: '5s', retries: 5 },
    },
  };
  for (const n of TOPO.nodes) {
    const p = n.port;
    const ip = n.port + 1;
    services[service(n.id)] = {
      image: IMAGE,
      depends_on: { minio: { condition: 'service_healthy' } },
      ports: [`${p}:${p}`],
      environment: [
        `LISTEN_ADDR=0.0.0.0:${p}`,
        `INTERNAL_LISTEN_ADDR=:${ip}`,
        `DB_FILE=/data/${n.id}.sqlite3`,
        'DEV=true',
        'LOG_LEVEL=info',
        'OWNER_EMAIL=hello@example.com',
        'SHUTDOWN_GRACE_PERIOD=1ms',
        'SHUTDOWN_TIMEOUT=1ms',
        `PUBLIC_URL=${internalURL(n.id)}`,
        'FEATURES={"vector_search":{"enabled":false}}',
        `MINIO_ENDPOINT=minio:${MINIO_PORT}`,
        'MINIO_ACCESS_KEY_ID=testuser',
        'MINIO_SECRET_KEY=testpassword',
        'MINIO_BUCKET=whoami-bucket',
        `MINIO_PREFIX=${n.id}/`,
        'MINIO_USE_SSL=false',
        `JWT_SECRET=whoami-${n.id}-secret`,
        `USER_TOKEN_COOKIE_NAME=${cookieName(n.id)}`,
        `GIT_API_REPO_PATH=/data/git-${n.id}`,
        'GIT_API_BASE_PATH=/git',
        'RESEND_API_KEY=test-key',
        'MAIL_FROM=test@example.com',
        'MCP_FEDERATION_MAX_DEPTH=3',
        'MCP_FEDERATED_GRAPHQL=true',
      ],
      volumes: ['whoami-data:/data'],
      healthcheck: {
        test: ['CMD', 'wget', '-q', '--spider', `http://localhost:${ip}/healthz`],
        interval: '5s',
        timeout: '5s',
        retries: 20,
        start_period: '10s',
      },
    };
  }
  return { name: PROJECT, services, volumes: { 'whoami-data': {} } };
}

function gen() {
  fs.writeFileSync(COMPOSE_FILE, JSON.stringify(composeDoc(), null, 2) + '\n');
  log(`wrote ${path.relative(ROOT, COMPOSE_FILE)} (${TOPO.nodes.length} nodes + minio)`);
}

function sh(cmd, args) {
  execFileSync(cmd, args, { stdio: 'inherit', cwd: ROOT });
}

function up() {
  gen();
  log('building image…');
  sh('docker', ['build', '-t', IMAGE, '.']);
  log('compose up --wait…');
  sh('docker', ['compose', '-f', COMPOSE_FILE, 'up', '-d', '--wait']);
}

function down() {
  sh('docker', ['compose', '-f', COMPOSE_FILE, 'down', '-v']);
}

// ── entrypoint ──────────────────────────────────────────────────────────────
const phases = {
  gen: async () => gen(),
  up: async () => up(),
  seed: async () => seed(),
  wire: async () => wire(),
  materialize: async () => materialize(),
  down: async () => down(),
  all: async () => { up(); await seed(); await wire(); await materialize(); },
};

const cmd = process.argv[2] || 'all';
const run = phases[cmd];
if (!run) {
  console.error(`unknown command "${cmd}". one of: ${Object.keys(phases).join(', ')}`);
  process.exit(2);
}
run()
  .then(() => log(`✓ ${cmd} done`))
  .catch((e) => { console.error(`✗ ${cmd} failed:`, e.message || e); process.exit(1); });
