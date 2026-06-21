#!/usr/bin/env zx
// @ts-check
//
// E2E test runner — zx prototype (REPRESENTATIVE SLICE of scripts/test-e2e.sh).
//
// This ports the early/structural phases of the bash runner to google/zx to
// evaluate whether the rewrite is "simpler". It covers, faithfully:
//   - arg parsing (--update-snapshots, --headed/--debug/--ui passthrough)
//   - cleanup of existing test containers
//   - test DB prepare + seed + telegram table wipe
//   - service build/up + health waits (4 app instances + minio)
//   - the Playwright "setup" spec that mints the API key
//   - the first real Playwright group (main tests, --grep-invert)
//
// Phases that are STILL STUBBED (clearly marked `STUB:` below) and left as
// remaining work: the CLI sync suite (test-sync-cli.sh), the three seedvault→peer
// pushes, wait_all_jobs draining, the telegram cron/update/snapshot block, and
// the ordered tail of Playwright specs (screenshots, layoutcss, federation-bidir,
// webhooks, unreleased-changes, show-draft-versions).
//
// Run (once `zx` is a devDep — see scripts/e2e-rewrite-proposal.md):
//   npx zx scripts/test-e2e.mjs [--headed|--debug|--ui] [--update-snapshots]
// Syntax check only (no infra):
//   node --check scripts/test-e2e.mjs
//
// Requires: `npm i -D zx` (zx provides $, fs, chalk, argv, within, cd globally).

import { $, argv, chalk, fs, within, cd, sleep } from 'zx';

// Fail loud, like `set -e`. Print the command we run (like bash -x-ish).
$.verbose = true;

// ---------------------------------------------------------------------------
// Config / env (mirrors the bash exports verbatim)
// ---------------------------------------------------------------------------
const APP_URL = process.env.APP_URL ?? 'http://localhost:20081';
const GIT_API_BASE_PATH = process.env.GIT_API_BASE_PATH ?? '/git';
const ENDPOINT = `${APP_URL}/graphql`;
const DB_PATH = 'tmp/data/test.sqlite3';
const COMPOSE = ['compose', '-f', 'docker-compose.test.yml'];

// These env vars are read by Playwright specs and the sync CLI, so export them
// into the child process environment exactly as the bash script did.
process.env.APP_URL = APP_URL;
process.env.GIT_API_BASE_PATH = GIT_API_BASE_PATH;
process.env.USER_TOKEN_COOKIE_NAME = 'trip2g_e2e';
process.env.ENDPOINT = ENDPOINT;

const UPDATE_SNAPSHOTS =
  argv['update-snapshots'] === true || process.env.UPDATE_SNAPSHOTS === '1';
const ENABLE_TG = process.env.ENABLE_TG === '1';

// Positional UI flag (--headed / --debug / --ui), preserved from bash $1.
const uiFlag = argv.headed
  ? '--headed'
  : argv.debug
    ? '--debug'
    : argv.ui
      ? '--ui'
      : null;

// ---------------------------------------------------------------------------
// Small helpers (replace the bash colour echoes + curl one-liners)
// ---------------------------------------------------------------------------
const ok = (m) => console.log(chalk.green(`✓ ${m}`));
const info = (m) => console.log(m);
const warn = (m) => console.log(chalk.yellow(m));
const fail = (m) => console.log(chalk.red(`✗ ${m}`));

/** Wait for a TCP port via the existing ./scripts/waitfor helper. */
async function waitFor(hostport, label) {
  try {
    await $`./scripts/waitfor ${hostport}`;
  } catch {
    fail(`${label} failed to start`);
    process.exit(1);
  }
}

/** Poll /debug/wait_all_jobs until it returns "ok:" — mirrors wait_all_jobs(). */
async function waitAllJobs() {
  info('⏳ Waiting for all background jobs to complete...');
  const res = await $`curl -s --max-time 300 ${APP_URL}/debug/wait_all_jobs`;
  process.stderr.write(res.stdout);
  if (!res.stdout.startsWith('ok:')) process.exit(1);
  await sleep(2000);
}

// ---------------------------------------------------------------------------
// Phase 1 — cleanup existing test containers
// ---------------------------------------------------------------------------
async function cleanupContainers() {
  info('🧹 Cleaning up existing test containers...');
  const svc = ['app', 'app-peer', 'app-peer2', 'app-peer3', 'minio', 'test-data'];
  // `|| true` equivalents: tolerate "no such container" on a fresh machine.
  await $`docker ${COMPOSE} stop ${svc}`.nothrow();
  await $`docker ${COMPOSE} rm -f ${svc}`.nothrow();
}

// ---------------------------------------------------------------------------
// Phase 2 — prepare DB + seed + wipe telegram tables
// ---------------------------------------------------------------------------
const TG_WIPE_SQL = `
  PRAGMA foreign_keys = OFF;
  delete from telegram_publish_sent_messages;
  delete from telegram_publish_sent_account_messages;
  delete from telegram_publish_note_tags;
  delete from telegram_publish_notes;
  delete from telegram_publish_chats;
  delete from telegram_publish_instant_chats;
  delete from telegram_publish_account_chats;
  delete from telegram_publish_account_instant_chats;
  delete from telegram_publish_tags;
  delete from telegram_accounts;
  delete from tg_user_states;
  delete from tg_user_profiles;
  delete from wait_list_tg_bot_requests;
  delete from tg_attach_codes;
  delete from tg_bot_chat_subgraph_accesses;
  delete from tg_bot_chat_subgraph_invites;
  delete from tg_bot_chats;
  delete from tg_bots;
  PRAGMA foreign_keys = ON;
`;

async function prepareDb() {
  process.env.DB_PATH = DB_PATH;
  info(`🗄️  Preparing test database ${DB_PATH}`);
  await fs.ensureDir('tmp/data');
  // Reproducibility: federation spec assumes a clean DB (see bash comment).
  for (const f of [DB_PATH, `${DB_PATH}-shm`, `${DB_PATH}-wal`]) {
    await fs.remove(f);
  }
  await $`sqlite3 ${DB_PATH} < testdata/e2e_seed.sql`;

  if (ENABLE_TG) {
    await $`go run ./cmd/tge2e -db ${DB_PATH} patch-db`;
    info('🧹 Cleaning up Telegram channels...');
    await $`go run ./cmd/tge2e -db ${DB_PATH} cleanup`;
  } else {
    info('🧹 Removing Telegram accounts and bots (ENABLE_TG not set)...');
    await $`sqlite3 ${DB_PATH} ${TG_WIPE_SQL}`;
  }
}

// ---------------------------------------------------------------------------
// Phase 3 — start services + health waits
// ---------------------------------------------------------------------------
async function startServices() {
  info('🚀 Starting services...');
  // Keep embedding warm between runs (avoid slow model reload).
  await $`docker ${COMPOSE} up -d --no-recreate embedding`.nothrow();
  await $`docker ${COMPOSE} up -d --build app app-peer app-peer2 app-peer3 minio`;

  await waitFor('localhost:20081', 'Services');
  await waitFor('localhost:20091', 'Peer service');
  await waitFor('localhost:20093', 'Peer2 service');
  await waitFor('localhost:20095', 'Peer3 service');
}

// ---------------------------------------------------------------------------
// Phase 4 — Playwright setup spec → mint API key
// ---------------------------------------------------------------------------
async function runSetupSpec() {
  info('🔑 Running setup test (create API key)...');
  try {
    await $`npx playwright test e2e/setup.spec.js`;
  } catch {
    fail('Setup test failed');
    process.exit(1);
  }
  if (!(await fs.pathExists('.test-api-key'))) {
    fail('API key file not found');
    process.exit(1);
  }
  const apiKey = (await fs.readFile('.test-api-key', 'utf8')).trim();
  ok(`API key created: ${apiKey.slice(0, 20)}...`);
  return apiKey;
}

// ---------------------------------------------------------------------------
// Phase 5 — first real Playwright group (main UI tests)
// ---------------------------------------------------------------------------
const MAIN_GREP_INVERT =
  'Setup|Layout CSS|Webhook|Screenshot|Bidirectional Federation';

async function runMainPlaywright() {
  info('🎭 Running main Playwright tests...');
  const args = ['test', '--grep-invert', MAIN_GREP_INVERT];
  if (uiFlag) args.push(uiFlag);
  try {
    await $`npx playwright ${args}`;
  } catch {
    fail('Playwright tests failed');
    info('Run with --ui for interactive debugging: npx zx scripts/test-e2e.mjs --ui');
    process.exit(1);
  }
  ok('Main Playwright tests passed');
}

// ===========================================================================
// STUBS — remaining phases to port (kept as no-ops so the slice stays runnable)
// ===========================================================================
async function stubbedRemainingPhases(apiKey) {
  warn('--- STUB: phases below are NOT yet ported (see proposal doc) ---');
  // STUB: ./scripts/test-sync-cli.sh --api-key $apiKey --endpoint $ENDPOINT [...]
  // STUB: sync_seedvault_to_peer  (port 20091, cookie trip2g_e2e_peer)
  // STUB: sync_seedvault2_to_peer2 (port 20093, cookie trip2g_e2e_peer2)
  // STUB: sync_seedvault3_to_peer3 (port 20095, cookie trip2g_e2e_peer3)
  // STUB: if ENABLE_TG -> run_telegram_cron()
  // STUB: wait_all_jobs()  (available above as waitAllJobs())
  // STUB: npx playwright test e2e/screenshots.spec.js
  // STUB: CSS hot-reload: append to styles.css, sync_vault(), layoutcss.spec.js
  // STUB: ENABLE_TG telegram update + snapshot check block
  // STUB: federation-bidir.spec.js
  // STUB: webhooks.spec.js
  // STUB: RUN_ISOLATED_SPECS=1 unreleased-changes.spec.js
  // STUB: RUN_ISOLATED_SPECS=1 show-draft-versions.spec.js
  void apiKey;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------
async function main() {
  info('🧪 Starting E2E tests...');
  if (UPDATE_SNAPSHOTS) warn('↻ Updating CLI sync golden snapshots');

  await within(async () => {
    // run everything from the repo root regardless of invocation cwd
    cd(new URL('..', import.meta.url).pathname);

    await cleanupContainers();
    await prepareDb();
    await startServices();

    if (argv.manual || process.env.MANUAL === '1') {
      const apiKey = await runSetupSpec();
      warn('🔧 Manual testing mode — services left running.');
      info(`API Key: ${apiKey}`);
      process.exit(0);
    }

    const apiKey = await runSetupSpec();
    await runMainPlaywright();
    await stubbedRemainingPhases(apiKey);
  });

  ok('✅ Slice complete (stubbed phases skipped)');
}

await main();
