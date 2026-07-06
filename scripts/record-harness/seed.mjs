#!/usr/bin/env node
// seed.mjs — bring up docker-compose.record.yml and seed it deterministically.
//
//   node scripts/record-harness/seed.mjs            # up --build + seed + verify
//   SKIP_UP=1 node scripts/record-harness/seed.mjs  # seed a running stack only
//
// What it produces (see docs/dev/record_harness.md):
//   landing http://localhost:21080 — repo docs/ vault (same excludes as
//     scripts/sync-docs-prod.sh) + overrides: header "try free →" CTA,
//     en/user/cloud pointing at the local dashboard stand-in, /simplecloud page.
//   space   http://localhost:21090 — EMPTY (onboarding state) + a DETERMINISTIC
//     admin api key (SPACE_API_KEY below) for the rig's preflight/stage/gate.
//
// Re-runnable: every step is idempotent (INSERT-if-missing, hideNotes cleanup).
import { execFileSync } from "node:child_process";
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(HERE, "../..");

const LANDING = "http://localhost:21080";
const SPACE = "http://localhost:21090";
const LANDING_COOKIE = "trip2g_record_landing";
const EMAIL = "hello@example.com";
const DEV_CODE = "111111";

// Deterministic space api key (64 alnum chars, matching GenerateAPIKey shape).
// Local-only harness credential — NOT a secret. The rig config
// (autoproducer v4-config.record.json) carries the same value.
const SPACE_API_KEY =
  "recordharness0recordharness0recordharness0recordharness0record0";

const SYNC_CLI = path.join(ROOT, "obsidian-sync/dist/trip2g-sync.mjs");
const DATA_DIR = path.join(ROOT, "tmp/record-data");

const log = (msg) => console.log(`== ${msg}`);
const run = (cmd, args, opts = {}) =>
  execFileSync(cmd, args, { stdio: "inherit", cwd: ROOT, ...opts });

async function gql(base, query, variables, headers = {}) {
  const res = await fetch(`${base}/_system/graphql`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...headers },
    body: JSON.stringify({ query, variables }),
  });
  if (!res.ok) throw new Error(`${base} graphql HTTP ${res.status}`);
  const body = await res.json();
  if (body.errors) throw new Error(`${base} graphql: ${JSON.stringify(body.errors).slice(0, 300)}`);
  return body.data;
}

async function waitFor(url, tries = 120) {
  for (let i = 0; i < tries; i++) {
    try {
      const res = await fetch(url);
      if (res.status < 500) return;
    } catch {}
    await new Promise((r) => setTimeout(r, 2000));
  }
  throw new Error(`timeout waiting for ${url}`);
}

async function signIn(base) {
  await gql(base, `mutation { requestEmailSignInCode(input: { email: ${JSON.stringify(EMAIL)} }) { ... on RequestEmailSignInCodePayload { success } } }`);
  const data = await gql(base, `mutation { signInByEmail(input: { email: ${JSON.stringify(EMAIL)}, code: "${DEV_CODE}" }) { ... on SignInPayload { token } ... on ErrorPayload { message } } }`);
  const token = data.signInByEmail?.token;
  if (!token) throw new Error(`${base}: sign-in failed: ${JSON.stringify(data)}`);
  return token;
}

async function pushNotes(base, apiKey, updates) {
  const data = await gql(
    base,
    `mutation($u:[PushNoteInput!]!){ pushNotes(input:{updates:$u}){ __typename ... on ErrorPayload { message } } }`,
    { u: updates },
    { "X-Api-Key": apiKey },
  );
  if (data.pushNotes?.__typename === "ErrorPayload") {
    throw new Error(`${base} pushNotes: ${data.pushNotes.message}`);
  }
}

async function hideNotes(base, apiKey, paths) {
  await gql(
    base,
    `mutation($p:[String!]!){ hideNotes(input:{paths:$p}) { ... on HideNotesPayload { success } ... on ErrorPayload { message } } }`,
    { p: paths },
    { "X-Api-Key": apiKey },
  );
}

// ---------- 0. compose up ----------
// Data dir MUST exist host-owned before compose creates it root-owned, and the
// containers run as the host uid (RECORD_UID/GID) so seed.mjs can write the DB.
fs.mkdirSync(DATA_DIR, { recursive: true });
const composeEnv = {
  ...process.env,
  RECORD_UID: String(process.getuid?.() ?? 1000),
  RECORD_GID: String(process.getgid?.() ?? 1000),
};
if (!process.env.SKIP_UP) {
  log("docker compose up --build (landing + space + minio)");
  run("docker", ["compose", "-f", "docker-compose.record.yml", "up", "-d", "--build"], { env: composeEnv });
}
log("waiting for landing + space");
await waitFor(`${LANDING}/`);
await waitFor(`${SPACE}/`);

// ---------- 1. landing: admin key for the seeder ----------
log("landing: sign in (dev code) + create seeder api key");
const landingToken = await signIn(LANDING);
const created = await gql(
  LANDING,
  `mutation($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) { ... on ErrorPayload { message } ... on CreateApiKeyPayload { value } } } }`,
  { input: { description: "record-harness seeder" } },
  { Cookie: `${LANDING_COOKIE}=${landingToken}` },
);
const landingKey = created.admin?.createApiKey?.value;
if (!landingKey) throw new Error(`landing createApiKey failed: ${JSON.stringify(created)}`);

// ---------- 2. landing: push the docs vault (same excludes as prod seed) ----------
if (!fs.existsSync(SYNC_CLI)) {
  log("building obsidian-sync CLI (dist missing)");
  run("npm", ["run", "build:cli"], { cwd: path.join(ROOT, "obsidian-sync") });
}
fs.mkdirSync(DATA_DIR, { recursive: true });
log("landing: sync docs/ vault (this is the same content trip2g.com is re-seeded from)");
run("node", [
  SYNC_CLI,
  "--folder", path.join(ROOT, "docs"),
  "--api-url", `${LANDING}/_system/graphql`,
  "--api-key", landingKey,
  "--state-file", path.join(DATA_DIR, "landing-sync-state.json"),
  "--conflict-resolution", "local",
  "--exclude", "demo",
  "--exclude", "dev",
  "--exclude", "marketing",
  "--exclude", "superpowers",
  "--exclude", "demo-video",
]);

// ---------- 3. landing: record-harness overrides ----------
// (a) header CTA "try free →" -> /en/user/cloud. This is the exact element the
// scenario clicks (segLand) — it was deployed ad-hoc on prod 2026-07-03 and
// later clobbered by the CI re-seed because it never landed in the repo.
// Inserted here as a seed-time overlay so the repo docs stay untouched.
log("landing: overrides (try-free CTA, local cloud page, /simplecloud stand-in)");
const barPath = path.join(ROOT, "docs/_layouts/mesh/bar.html");
const startAnchor = `<a href="/en/user/getting_started" class="@did__nav-link">start →</a>`;
let bar = fs.readFileSync(barPath, "utf8");
if (!bar.includes(startAnchor)) {
  throw new Error(`bar.html drifted: EN "start →" anchor not found — update seed.mjs override`);
}
bar = bar.replace(
  startAnchor,
  `<a href="/en/user/cloud" class="@did__nav-link">try free →</a>\n    ${startAnchor}`,
);

// (b) en/user/cloud.md: point "Open the cloud" at the LOCAL dashboard stand-in.
// The path contains "simplecloud" on purpose — record.mjs asserts
// a.href.includes('simplecloud') on this link.
const cloudPath = path.join(ROOT, "docs/en/user/cloud.md");
let cloud = fs.readFileSync(cloudPath, "utf8");
if (!cloud.includes("https://simplecloud.2pub.me")) {
  throw new Error(`en/user/cloud.md drifted: simplecloud link not found — update seed.mjs override`);
}
cloud = cloud.replaceAll("https://simplecloud.2pub.me", `${LANDING}/simplecloud`);
// visible link TEXT too — the video must not show the prod domain over a local URL
cloud = cloud.replaceAll("simplecloud.2pub.me", "localhost:21080/simplecloud");

// (c) /simplecloud dashboard stand-in with the "Open as Admin" link.
const dash = fs
  .readFileSync(path.join(HERE, "overrides/simplecloud/_index.md"), "utf8")
  .replaceAll("{{SPACE}}", SPACE);

await pushNotes(LANDING, landingKey, [
  { path: "_layouts/mesh/bar.html", content: bar },
  { path: "en/user/cloud.md", content: cloud },
  { path: "simplecloud/_index.md", content: dash },
]);

// ---------- 4. space: deterministic admin api key ----------
log("space: sign in (asserts the boot-created owner/admin)");
await signIn(SPACE);
const hash = crypto.createHash("sha256").update(SPACE_API_KEY).digest("hex");
log("space: insert deterministic api key (idempotent)");
run("sqlite3", [
  path.join(DATA_DIR, "space.sqlite3"),
  `PRAGMA busy_timeout=5000;
   INSERT INTO api_keys (value, created_by, description)
   SELECT '${hash}', user_id, 'record-harness (deterministic)'
   FROM admins
   WHERE NOT EXISTS (SELECT 1 FROM api_keys WHERE value = '${hash}')
   LIMIT 1;`,
]);

// ---------- 5. space: verify the key + restore the onboarding empty state ----------
log("space: probe push via the deterministic key, then hide (back to onboarding)");
await pushNotes(SPACE, SPACE_API_KEY, [
  { path: "Record Harness Probe.md", content: "---\nfree: true\n---\nrecord-harness seed probe — safe to delete\n" },
]);
const probe = await fetch(`${SPACE}/record_harness_probe`, { headers: { "Cache-Control": "no-store" } });
if (probe.status !== 200) throw new Error(`space probe page -> HTTP ${probe.status} (key or render broken)`);
await hideNotes(SPACE, SPACE_API_KEY, ["Record Harness Probe.md"]);
const home = await (await fetch(`${SPACE}/`, { headers: { "Cache-Control": "no-store" } })).text();
if (!home.includes("data-onboarding")) {
  throw new Error("space is NOT back in the onboarding empty state after hideNotes");
}

// ---------- summary ----------
console.log(`
record harness READY
  landing    ${LANDING}            (try free → header CTA -> /en/user/cloud)
  dashboard  ${LANDING}/simplecloud (stand-in; "Open as Admin" -> space)
  space      ${SPACE}            (onboarding empty state)
  space key  ${SPACE_API_KEY}
  sign-in    ${EMAIL} / dev code ${DEV_CODE} (both instances)

one-time in the recording browser profile: sign in as admin on ${SPACE}
(dev code ${DEV_CODE}) so the Welcome page shows "Download archive" —
mirrors the operational simplecloud login that was never on camera.
`);
