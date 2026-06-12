// Benchmark driver for pushNotes lock-time measurements.
//
// Pushes N synthetic notes in one mutation (initial push = one long write
// transaction), fires a single concurrent 1-note "probe" push mid-flight to
// measure how long a victim writer waits for the lock, then measures the
// median 1-note incremental push and the median 10%-of-vault batch push at
// vault size N (the renderer caches unchanged notes, so batch size matters).
// Reads RSS from the internal /metrics endpoint and peak RSS from
// /proc/<pid>/status.
//
// Env: APP_URL, INTERNAL_URL, SERVER_PID, N (note count), CORES (label).
// Output: one JSON line on stdout.
import fs from 'node:fs';

const APP_URL = process.env.APP_URL || 'http://localhost:28081';
const INTERNAL_URL = process.env.INTERNAL_URL || 'http://localhost:28082';
const SERVER_PID = process.env.SERVER_PID;
const N = parseInt(process.env.N || '10', 10);
const CORES = process.env.CORES || '?';
// VECTOR=1: vector search enabled on the server — after pushes, wait for
// embedding jobs to finish and reload so the in-memory vector index is
// populated before measuring RSS.
const VECTOR = process.env.VECTOR === '1';

async function gql(query, variables, headers = {}) {
  const res = await fetch(`${APP_URL}/graphql`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...headers },
    body: JSON.stringify({ query, variables }),
  });
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

async function signIn() {
  await gql(`mutation { requestEmailSignInCode(input: { email: "hello@example.com" }) { ... on RequestEmailSignInCodePayload { success } } }`);
  for (const code of ['111111', '000000']) {
    try {
      const d = await gql(`mutation { signInByEmail(input: { email: "hello@example.com", code: "${code}" }) { ... on SignInPayload { token } ... on ErrorPayload { message } } }`);
      if (d.signInByEmail.token) return d.signInByEmail.token;
    } catch { /* try next code */ }
  }
  throw new Error('sign-in failed with dev codes');
}

async function createApiKey(token) {
  const d = await gql(
    `mutation { admin { createApiKey(input: { description: "bench" }) { ... on CreateApiKeyPayload { value } ... on ErrorPayload { message } } } }`,
    {},
    { Cookie: `trip2g_token=${token}` },
  );
  const v = d.admin.createApiKey.value;
  if (!v) throw new Error('createApiKey failed: ' + JSON.stringify(d));
  return v;
}

function noteContent(i, rev = 0) {
  return [
    '---', 'free: true', `title: Bench note ${i}`, '---', '',
    `# Bench note ${i}`, '',
    `Revision ${rev}. This is a synthetic note used by the pushNotes benchmark.`,
    'It contains a couple of paragraphs, a [[bench/note_0]] wikilink, a list and',
    'a code block to be representative of real content.', '',
    '- point one', '- point two', '- point three', '',
    '```js', `const x = ${i};`, 'console.log(x);', '```', '',
    'Closing paragraph with some more text to give the renderer real work.',
  ].join('\n');
}

async function push(apiKey, updates) {
  const t0 = performance.now();
  let error = null;
  try {
    const d = await gql(
      `mutation Push($input: PushNotesInput!) { pushNotes(input: $input) { __typename ... on ErrorPayload { message } } }`,
      { input: { updates } },
      { 'X-API-Key': apiKey },
    );
    if (d.pushNotes.__typename === 'ErrorPayload') error = d.pushNotes.message;
  } catch (e) {
    error = String(e.message || e);
  }
  return { ms: performance.now() - t0, error };
}

function readMetric(text, name) {
  const m = text.match(new RegExp(`^${name}(?:{[^}]*})? ([0-9.e+]+)$`, 'm'));
  return m ? parseFloat(m[1]) : null;
}

async function rssMB() {
  try {
    const text = await (await fetch(`${INTERNAL_URL}/metrics`)).text();
    const rss = readMetric(text, 'process_resident_memory_bytes');
    return rss ? Math.round(rss / 1024 / 1024) : null;
  } catch { return null; }
}

function peakRssMB() {
  if (!SERVER_PID) return null;
  try {
    const status = fs.readFileSync(`/proc/${SERVER_PID}/status`, 'utf8');
    const m = status.match(/VmHWM:\s+(\d+) kB/);
    return m ? Math.round(parseInt(m[1], 10) / 1024) : null;
  } catch { return null; }
}

const median = (xs) => xs.slice().sort((a, b) => a - b)[Math.floor(xs.length / 2)];

async function main() {
  const token = await signIn();
  const apiKey = await createApiKey(token);

  // Initial push of N notes in one mutation = one write transaction.
  const updates = Array.from({ length: N }, (_, i) => ({ path: `bench/note_${i}.md`, content: noteContent(i) }));

  const bigPush = push(apiKey, updates);

  // Probe: a 1-note push racing the big one — measures victim wait + its own work.
  await new Promise((r) => setTimeout(r, 150));
  const probe = await push(apiKey, [{ path: 'probe/probe.md', content: noteContent(99999) }]);
  const big = await bigPush;

  // Incremental: typical small sync at vault size N (median of 3).
  const incr1 = [];
  for (let rev = 1; rev <= 3; rev++) {
    incr1.push(await push(apiKey, [{ path: 'bench/note_0.md', content: noteContent(0, rev) }]));
  }

  // Batch incremental: 10% of the vault changed in one push (median of 3).
  const tenPercent = Math.max(1, Math.round(N / 10));
  const incr10 = [];
  for (let rev = 4; rev <= 6; rev++) {
    const batch = Array.from({ length: tenPercent }, (_, i) => ({ path: `bench/note_${i}.md`, content: noteContent(i, rev) }));
    incr10.push(await push(apiKey, batch));
  }

  // Vector mode: wait for embedding jobs, then push one more revision so the
  // reload picks the fresh embeddings up into the in-memory index.
  let embedMs = null;
  if (VECTOR) {
    const t0 = performance.now();
    // wait_all_jobs times out server-side after 5 minutes; loop until drained.
    // Each iteration covers up to 5 min × (10 jobs / 100 ms poll) = 3 000 jobs.
    for (;;) {
      const res = await fetch(`${APP_URL}/debug/wait_all_jobs?interval=500`, { signal: AbortSignal.timeout(600_000) });
      if (res.ok) break;
      const text = await res.text();
      if (!text.includes('jobs still pending after')) throw new Error(`wait_all_jobs failed: ${text}`);
      // Server hit its 5-minute window; jobs still running — retry immediately.
    }
    embedMs = Math.round(performance.now() - t0);
    await push(apiKey, [{ path: 'bench/note_0.md', content: noteContent(0, 7) }]);
    await fetch(`${APP_URL}/debug/wait_all_jobs?interval=500`, { signal: AbortSignal.timeout(300_000) });
    await push(apiKey, [{ path: 'bench/note_0.md', content: noteContent(0, 8) }]);
  }

  const result = {
    cores: CORES,
    notes: N,
    vector: VECTOR,
    embed_wait_ms: embedMs,
    initial_push_ms: Math.round(big.ms),
    initial_push_error: big.error,
    probe_ms: Math.round(probe.ms),
    probe_error: probe.error,
    incr_1_ms: Math.round(median(incr1.map((r) => r.ms))),
    incr_1_errors: incr1.map((r) => r.error).filter(Boolean),
    incr_10p_ms: Math.round(median(incr10.map((r) => r.ms))),
    incr_10p_notes: tenPercent,
    incr_10p_errors: incr10.map((r) => r.error).filter(Boolean),
    rss_mb: await rssMB(),
    peak_rss_mb: peakRssMB(),
  };
  console.log(JSON.stringify(result));
}

main().catch((e) => { console.error(e); process.exit(1); });
