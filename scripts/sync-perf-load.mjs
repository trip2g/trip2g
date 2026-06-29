#!/usr/bin/env node
// @ts-check
//
// Sustained sync LOAD generator — run it by hand while you profile the Go server
// (`make bench` / pprof) and watch the obsidian-sync plugin react in Obsidian.
//
// Unlike scripts/syncperf/bench.mjs (which times ONE sync), this keeps writing at
// a steady rate so you can observe live behaviour under continuous churn:
//   - api  mode: push notes to the server  → exercises the plugin's LIVE-PULL
//                (the open vault pulls each change; watch for lag / echo loops)
//   - disk mode: rewrite files in the vault → exercises the plugin's AUTO-SYNC push
//                (the plugin batches+pushes them; edit your own note meanwhile and
//                 see if your save is starved behind the load)
//
// Usage:
//   node scripts/sync-perf-load.mjs                         # api, 10/s, 20 files, 60s, :8081
//   node scripts/sync-perf-load.mjs --mode disk --vault docs
//   node scripts/sync-perf-load.mjs --rate 25 --files 50 --duration 0   # 0 = until Ctrl+C
//   node scripts/sync-perf-load.mjs --cleanup-only          # remove a previous run's notes/files
//
// Flags:
//   --mode api|disk     load style (default: api)
//   --app-url URL       server (default: http://localhost:8081)
//   --vault PATH        vault root for disk mode + data.json key fallback (default: docs)
//   --dir NAME          subfolder for the synthetic notes (default: loadtest)
//   --rate N            writes per second (default: 10)
//   --files N           number of distinct files cycled (default: 20)
//   --duration SEC      run length; 0 = until Ctrl+C (default: 60)
//   --key KEY           api key to use (default: mint a dedicated dev key — see below)
//   --cleanup           delete the synthetic notes/files when the run finishes
//   --cleanup-only      just clean up a previous run and exit
//
// Auth (api mode): by default this mints its OWN dedicated API key via the dev
// sign-in flow (hello@example.com + dev code, DEV server only) — independent of the
// Obsidian plugin's key in data.json, which rotates under load and causes spurious
// "invalid token" errors. Pass --key to override, or it falls back to the vault's
// data.json key.
//
// Pair with the Go profiler (separate terminal), e.g.:
//   make bench DUR=30s RATE=200 URL=/en/user/kanban_demo     # vegeta + CPU pprof
//   # or: curl 'http://localhost:6060/debug/pprof/profile?seconds=30' -o /tmp/cpu.prof
//
// Tip: keep an eye on the plugin's status bar / the server log while this runs.

import fs from "node:fs";
import path from "node:path";

const args = process.argv.slice(2);
const argVal = (f, d) => { const i = args.indexOf(f); return i >= 0 ? args[i + 1] : d; };
const has = (f) => args.includes(f);

const MODE = argVal("--mode", "api");
const APP_URL = argVal("--app-url", "http://localhost:8081").replace(/\/+$/, "");
const VAULT = path.resolve(argVal("--vault", "docs"));
const DIR = argVal("--dir", "loadtest");
const RATE = Math.max(0.1, parseFloat(argVal("--rate", "10")));
const FILES = Math.max(1, parseInt(argVal("--files", "20"), 10));
const DURATION = parseInt(argVal("--duration", "60"), 10); // seconds; 0 = forever
const CLEANUP = has("--cleanup");
const CLEANUP_ONLY = has("--cleanup-only");

const period = 1000 / RATE;
const notePath = (n) => `${DIR}/${n}.md`;
const noteBody = (n, i) =>
  `---\nfree: true\ntitle: Load ${n}\n---\n\n# Load ${n}\n\n- [ ] iter ${i} @ ${new Date().toISOString()}\n\n` +
  `Synthetic note from sync-perf-load. Rev ${i}. Padding line one. Padding line two.\n`;

async function gql(query, variables, headers = {}) {
  const res = await fetch(`${APP_URL}/graphql`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...headers },
    body: JSON.stringify({ query, variables }),
  });
  const body = await res.json();
  if (body.errors) throw new Error(JSON.stringify(body.errors));
  return body.data;
}

async function devSignInKey() {
  await gql(`mutation { requestEmailSignInCode(input: { email: "hello@example.com" }) { __typename } }`).catch(() => {});
  let token = null;
  for (const code of ["111111", "000000"]) {
    try {
      const d = await gql(`mutation { signInByEmail(input: { email: "hello@example.com", code: "${code}" }) { ... on SignInPayload { token } } }`);
      if (d.signInByEmail?.token) { token = d.signInByEmail.token; break; }
    } catch { /* next */ }
  }
  if (!token) throw new Error("dev sign-in failed");
  const d = await gql(
    `mutation { admin { createApiKey(input: { description: "sync-perf-load" }) { ... on CreateApiKeyPayload { value } ... on ErrorPayload { message } } } }`,
    {}, { Cookie: `trip2g_token=${token}` });
  const v = d.admin?.createApiKey?.value;
  if (!v) throw new Error("createApiKey failed: " + JSON.stringify(d));
  return v;
}

function dataJsonKey() {
  try {
    const p = path.join(VAULT, ".obsidian/plugins/trip2g/data.json");
    const j = JSON.parse(fs.readFileSync(p, "utf8"));
    return j.syncDirs?.[0]?.apiKey || null;
  } catch { return null; }
}

async function getApiKey() {
  const explicit = argVal("--key", null);
  if (explicit) return explicit;
  try { return await devSignInKey(); }
  catch (e) {
    const k = dataJsonKey();
    if (k) { console.warn(`! dev sign-in failed (${e.message}); using vault data.json key (may rotate under load)`); return k; }
    throw new Error(`no API key: dev sign-in failed and no data.json key in ${VAULT}`);
  }
}

const PUSH = `mutation Push($input: PushNotesInput!) { pushNotes(input: $input) { __typename ... on ErrorPayload { message } } }`;
const HIDE = `mutation Hide($input: HideNotesInput!) { hideNotes(input: $input) { __typename } }`;

// ── metrics ──────────────────────────────────────────────────────────────────
let sent = 0, ok = 0, err = 0, inflight = 0;
let winLat = [], winOk = 0, winErr = 0;
const errSamples = new Map();
function pct(a, p) { if (!a.length) return 0; const s = [...a].sort((x, y) => x - y); return Math.round(s[Math.min(s.length - 1, Math.floor(p / 100 * s.length))]); }

let reporter = null, ticker = null, stopping = false;
const t0 = Date.now();

function report() {
  const t = ((Date.now() - t0) / 1000).toFixed(0);
  const avg = winLat.length ? Math.round(winLat.reduce((a, b) => a + b, 0) / winLat.length) : 0;
  const latStr = MODE === "api" ? ` lat avg=${avg}ms p50=${pct(winLat, 50)}ms p95=${pct(winLat, 95)}ms` : "";
  console.log(`[t=${t}s] ${MODE}: ${winOk}/s ok${latStr} err=${winErr} inflight=${inflight} | total ok=${ok} err=${err}`);
  if (winErr && errSamples.size) { for (const [m, c] of errSamples) console.log(`    err ×${c}: ${m.slice(0, 100)}`); errSamples.clear(); }
  winLat = []; winOk = 0; winErr = 0;
}

async function finish(apiKey) {
  if (stopping) return; stopping = true;
  clearInterval(ticker); clearInterval(reporter);
  while (inflight > 0) await new Promise((r) => setTimeout(r, 50));
  const dur = (Date.now() - t0) / 1000;
  console.log(`\n── done ── ${sent} writes in ${dur.toFixed(1)}s = ${(sent / dur).toFixed(1)}/s | ok=${ok} err=${err}`);
  if (CLEANUP || CLEANUP_ONLY) await cleanup(apiKey);
  process.exit(err > 0 && ok === 0 ? 1 : 0);
}

async function cleanup(apiKey) {
  const paths = Array.from({ length: FILES }, (_, i) => notePath(i + 1));
  if (MODE === "disk") {
    let n = 0;
    for (const rel of paths) { const f = path.join(VAULT, rel); if (fs.existsSync(f)) { fs.rmSync(f); n++; } }
    const d = path.join(VAULT, DIR); try { fs.rmdirSync(d); } catch {}
    console.log(`cleaned ${n} local files under ${DIR}/`);
  } else {
    try { await gql(HIDE, { input: { paths } }, { "X-API-Key": apiKey }); console.log(`hid ${paths.length} notes under ${DIR}/`); }
    catch (e) { console.warn(`cleanup hideNotes failed: ${e.message}`); }
  }
}

// ── load loops ───────────────────────────────────────────────────────────────
async function runApi(apiKey) {
  let i = 0;
  ticker = setInterval(() => {
    if (stopping) return;
    i++; const n = (i % FILES) + 1;
    sent++; inflight++;
    const start = performance.now();
    gql(PUSH, { input: { updates: [{ path: notePath(n), content: noteBody(n, i) }] } }, { "X-API-Key": apiKey })
      .then((d) => {
        const lat = performance.now() - start;
        if (d.pushNotes.__typename === "ErrorPayload") { err++; winErr++; errSamples.set(d.pushNotes.message, (errSamples.get(d.pushNotes.message) || 0) + 1); }
        else { ok++; winOk++; winLat.push(lat); }
      })
      .catch((e) => { err++; winErr++; errSamples.set(e.message, (errSamples.get(e.message) || 0) + 1); })
      .finally(() => { inflight--; });
  }, period);
}

function runDisk() {
  const base = path.join(VAULT, DIR);
  fs.mkdirSync(base, { recursive: true });
  for (let n = 1; n <= FILES; n++) fs.writeFileSync(path.join(base, `${n}.md`), noteBody(n, 0)); // seed so edits fire "modify"
  let i = 0;
  ticker = setInterval(() => {
    if (stopping) return;
    i++; const n = (i % FILES) + 1;
    try { fs.writeFileSync(path.join(base, `${n}.md`), noteBody(n, i)); sent++; ok++; winOk++; }
    catch (e) { sent++; err++; winErr++; errSamples.set(e.message, (errSamples.get(e.message) || 0) + 1); }
  }, period);
}

// ── main ─────────────────────────────────────────────────────────────────────
(async () => {
  let apiKey = null;
  if (MODE === "api" || CLEANUP_ONLY && MODE === "api") apiKey = await getApiKey();

  if (CLEANUP_ONLY) { await cleanup(apiKey); process.exit(0); }

  console.log(`sync-perf-load: mode=${MODE} target=${MODE === "api" ? APP_URL : VAULT + "/" + DIR} rate=${RATE}/s files=${FILES} duration=${DURATION || "∞"}s`);
  if (MODE === "api") console.log(`  api key: ...${apiKey.slice(-6)} (dedicated; cleanup with --cleanup or --cleanup-only)`);
  console.log(`  Ctrl+C to stop. Watch the plugin in Obsidian; profile the server in another terminal (make bench / pprof).\n`);

  process.on("SIGINT", () => { console.log("\nstopping…"); finish(apiKey); });
  reporter = setInterval(report, 1000);
  if (MODE === "api") await runApi(apiKey); else runDisk();
  if (DURATION > 0) setTimeout(() => finish(apiKey), DURATION * 1000);
})().catch((e) => { console.error("fatal:", e.message); process.exit(1); });
