/**
 * trip2g-memory — one verb to give an agent a minimal self-hosted memory.
 *
 * Boots a trip2g server (local filesystem storage, no S3/DEV/email/git) and
 * a trip2g-sync --watch sidecar, then mints an admin API key via HAT.
 *
 * Usage:
 *   trip2g-memory [up] --folder <vault> [options]
 *   trip2g-memory down
 *   trip2g-memory status
 *   trip2g-memory logs
 *   trip2g-memory key
 */

import crypto from 'node:crypto';
import { spawnSync, spawn } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { print } from 'graphql';
import { CreateApiKeyDocument, DisableApiKeyDocument } from './generated/graphql.ts';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const CONTAINER_NAME = 'trip2g-memory';
const DEFAULT_PORT = 24081;
const DEFAULT_IMAGE = 'ghcr.io/trip2g/trip2g:latest';
const DEFAULT_EMAIL = 'memory@local';
const READY_TIMEOUT_MS = 60_000;
const READY_POLL_MS = 500;

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface Flags {
  dryRun: boolean;
  help: boolean;
  folder: string | null;
  port: number;
  email: string;
  image: string;
  publicUrl: string | null;
  noHub: boolean;
  hubUrl: string;
}

export interface ServerEnv {
  LISTEN_ADDR: string;
  INTERNAL_LISTEN_ADDR: string;
  DB_FILE: string;
  OWNER_EMAIL: string;
  PUBLIC_URL: string;
  JWT_SECRET: string;
  STORAGE_BACKEND: string;
  STORAGE_LOCAL_DIR: string;
}

export interface DataJson {
  syncDirs: Array<{
    path: string;
    apiUrl: string;
    apiKey: string;
    twoWaySync: boolean;
  }>;
}

const DEFAULT_HUB_URL = 'https://trip2g.com/_system/mcp';

// ---------------------------------------------------------------------------
// Pure helpers (exported for testing)
// ---------------------------------------------------------------------------

/**
 * Build a base64url-encoded string without padding.
 */
function base64url(buf: Buffer | string): string {
  const b = Buffer.isBuffer(buf) ? buf : Buffer.from(buf);
  return b.toString('base64url');
}

/**
 * Sign a HAT JWT with HMAC-SHA256.
 * Mirrors the logic in internal/hotauthtoken/ and admin_hat.go.
 *
 * @param secret - the JWT_SECRET / HOT_AUTH_TOKEN_SECRET
 * @param email  - user email to embed in payload
 * @returns compact JWT string (header.payload.signature)
 */
export function signHatJwt(secret: string, email: string): string {
  const headerB64 = base64url(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = {
    e: email,
    ae: true,
    exp: Math.floor(Date.now() / 1000) + 300,
  };
  const payloadB64 = base64url(JSON.stringify(payload));
  const signingInput = `${headerB64}.${payloadB64}`;
  const sig = crypto.createHmac('sha256', secret).update(signingInput).digest();
  return `${signingInput}.${base64url(sig)}`;
}

/**
 * Build the hub federation note content.
 * Frontmatter wires mcp_federation_kb_url so the local trip2g instance
 * federates outbound to the given hub MCP endpoint.
 */
export function buildHubNote(hubUrl: string): string {
  const host = new URL(hubUrl).hostname;
  return `---
mcp_federation_kb_url: ${hubUrl}
mcp_federation_kb_id: ${host}
---

This memory federates to **${host}** via MCP. An agent that connects to this instance's \`/_system/mcp\` can pull ${host}'s published docs (including the trip2g memory docs) without leaving its own memory scope.

To disable: delete this file or run \`memcli up\` with \`--no-hub\`.
`.trimStart();
}

/**
 * Parse process.argv (or a provided array) into { cmd, flags }.
 * Subcommands: up (default), down, status, logs, key.
 */
export function parseArgs(argv: string[]): { cmd: string; flags: Flags } {
  const SUBCOMMANDS = new Set(['up', 'down', 'status', 'logs', 'key']);
  const flags: Flags = {
    dryRun: false,
    help: false,
    folder: null,
    port: DEFAULT_PORT,
    email: DEFAULT_EMAIL,
    image: DEFAULT_IMAGE,
    publicUrl: null,
    noHub: false,
    hubUrl: DEFAULT_HUB_URL,
  };

  let cmd = 'up';
  let i = 0;

  if (argv.length > 0 && !argv[0].startsWith('-')) {
    if (SUBCOMMANDS.has(argv[0])) {
      cmd = argv[0];
      i = 1;
    }
  }

  while (i < argv.length) {
    const arg = argv[i];
    if (arg === '--dry-run') {
      flags.dryRun = true;
    } else if (arg === '--help' || arg === '-h') {
      flags.help = true;
    } else if (arg === '--folder') {
      flags.folder = argv[++i];
    } else if (arg === '--port') {
      flags.port = parseInt(argv[++i], 10);
    } else if (arg === '--email') {
      flags.email = argv[++i];
    } else if (arg === '--image') {
      flags.image = argv[++i];
    } else if (arg === '--public-url') {
      flags.publicUrl = argv[++i];
    } else if (arg === '--no-hub') {
      flags.noHub = true;
    } else if (arg === '--hub-url') {
      flags.hubUrl = argv[++i];
    }
    i++;
  }

  return { cmd, flags };
}

/**
 * Build the environment variable map for the trip2g Docker container.
 * Asserts no DEV/RESEND/SMTP/GIT keys are present.
 * Uses local filesystem storage (no MinIO required).
 */
export function buildServerEnv(opts: {
  port: number;
  iport: number;
  email: string;
  secret: string;
}): ServerEnv {
  const { port, iport, email, secret } = opts;
  const env: ServerEnv = {
    LISTEN_ADDR: `0.0.0.0:${port}`,
    INTERNAL_LISTEN_ADDR: `:${iport}`,
    DB_FILE: '/data/local.sqlite3',
    OWNER_EMAIL: email,
    PUBLIC_URL: `http://localhost:${port}`,
    JWT_SECRET: secret,
    STORAGE_BACKEND: 'local',
    STORAGE_LOCAL_DIR: '/data/storage',
  };

  // Safety assertion: must not leak DEV/email/git keys
  const forbidden = ['DEV', 'RESEND_API_KEY', 'SMTP_PASSWORD', 'GIT_API_REPO_PATH'];
  for (const k of forbidden) {
    if (k in env) {
      throw new Error(`buildServerEnv: forbidden key ${k} must not be in server env`);
    }
  }

  return env;
}

/**
 * Build the docker run argv array (everything after "docker run").
 */
export function buildDockerRunArgs(opts: {
  port: number;
  iport: number;
  email: string;
  secret: string;
  stateDir: string;
  image: string;
}): string[] {
  const { port, iport, stateDir, image } = opts;
  const env = buildServerEnv(opts);

  const args: string[] = [
    '-d',
    '--name', CONTAINER_NAME,
    // Loopback bind — only reachable from localhost
    '-p', `127.0.0.1:${port}:${port}`,
    '-p', `127.0.0.1:${iport}:${iport}`,
    '-v', `${stateDir}/data:/data`,
  ];

  for (const [k, v] of Object.entries(env)) {
    args.push('-e', `${k}=${v}`);
  }

  args.push(image);
  return args;
}

/**
 * Build the Obsidian plugin config JSON object.
 */
export function buildDataJson(vault: string, publicUrl: string, apiKey: string): DataJson {
  return {
    syncDirs: [
      {
        path: vault,
        apiUrl: publicUrl,
        apiKey,
        twoWaySync: true,
      },
    ],
  };
}

// ---------------------------------------------------------------------------
// GraphQL over HTTP
// ---------------------------------------------------------------------------

/**
 * Execute a typed GraphQL document over HTTP with Bearer auth.
 * Throws if the response is not ok or the data is missing.
 */
async function gqlRequest<TData, TVariables>(
  endpoint: string,
  bearerToken: string,
  document: import('@graphql-typed-document-node/core').TypedDocumentNode<TData, TVariables>,
  variables: TVariables,
): Promise<TData> {
  const res = await fetch(endpoint, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      'authorization': `Bearer ${bearerToken}`,
    },
    body: JSON.stringify({ query: print(document), variables }),
  });

  if (!res.ok) {
    throw new Error(`GraphQL request failed: ${res.status} ${await res.text()}`);
  }

  const body = (await res.json()) as { data?: TData; errors?: unknown[] };
  if (body.errors && body.errors.length > 0) {
    throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  }
  if (body.data === undefined) {
    throw new Error(`GraphQL response missing data: ${JSON.stringify(body)}`);
  }
  return body.data;
}

// ---------------------------------------------------------------------------
// Side-effectful helpers
// ---------------------------------------------------------------------------

function readEnvFile(envFile: string): Record<string, string> {
  if (!fs.existsSync(envFile)) return {};
  const lines = fs.readFileSync(envFile, 'utf8').split('\n');
  const out: Record<string, string> = {};
  for (const line of lines) {
    const eq = line.indexOf('=');
    if (eq < 0) continue;
    out[line.slice(0, eq).trim()] = line.slice(eq + 1).trim();
  }
  return out;
}

function writeEnvFile(envFile: string, data: Record<string, string>): void {
  const content = Object.entries(data).map(([k, v]) => `${k}=${v}`).join('\n') + '\n';
  fs.writeFileSync(envFile, content, { mode: 0o600 });
  fs.chmodSync(envFile, 0o600);
}

async function waitReady(url: string, timeoutMs: number, pollMs: number): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(url, { signal: AbortSignal.timeout(2000) });
      if (res.status === 200) return;
    } catch {
      // Not yet ready — keep polling
    }
    await new Promise<void>((r) => setTimeout(r, pollMs));
  }
  throw new Error(`Timed out waiting for ${url} to return 200 after ${timeoutMs}ms`);
}

/** Obtain a Bearer token via HAT flow. */
async function hatAuth(publicUrl: string, secret: string, email: string): Promise<string> {
  const jwt = signHatJwt(secret, email);
  const hatRes = await fetch(`${publicUrl}/_system/hat`, {
    method: 'POST',
    redirect: 'manual',
    headers: { 'content-type': 'application/x-www-form-urlencoded' },
    body: `token=${encodeURIComponent(jwt)}`,
  });

  const setCookie = hatRes.headers.get('set-cookie') || '';
  const match = setCookie.match(/trip2g_token=([^;]+)/);
  if (!match) {
    throw new Error(
      `HAT response did not set trip2g_token cookie. status=${hatRes.status} set-cookie=${setCookie}`,
    );
  }
  return match[1];
}

/** Mint API key via HAT flow. Returns { apiKey, apiKeyId }. */
async function mintApiKey(
  publicUrl: string,
  secret: string,
  email: string,
): Promise<{ apiKey: string; apiKeyId: number | null }> {
  const token = await hatAuth(publicUrl, secret, email);
  const gqlUrl = `${publicUrl}/_system/graphql`;

  const data = await gqlRequest(gqlUrl, token, CreateApiKeyDocument, {
    description: 'agent-memory',
  });

  const result = data.admin.createApiKey;
  if (result.__typename === 'ErrorPayload') {
    throw new Error(`createApiKey error: ${result.message}`);
  }
  if (result.__typename !== 'CreateApiKeyPayload') {
    throw new Error(`Unknown createApiKey result type: ${result.__typename}`);
  }

  return {
    apiKey: result.value,
    apiKeyId: result.apiKey?.id ?? null,
  };
}

function writeDataJson(vault: string, publicUrl: string, apiKey: string): string {
  const pluginDir = path.join(vault, '.obsidian', 'plugins', 'trip2g');
  fs.mkdirSync(pluginDir, { recursive: true });
  const dataFile = path.join(pluginDir, 'data.json');
  fs.writeFileSync(dataFile, JSON.stringify(buildDataJson(vault, publicUrl, apiKey), null, 2));
  return dataFile;
}

function isPidAlive(pid: number, cmdlinePattern: RegExp): boolean {
  try {
    process.kill(pid, 0);
    try {
      const cmdline = fs.readFileSync(`/proc/${pid}/cmdline`, 'utf8').replace(/\0/g, ' ');
      return cmdlinePattern.test(cmdline);
    } catch {
      // /proc not available (non-Linux) — trust the signal
      return true;
    }
  } catch {
    return false;
  }
}

function repoRoot(): string {
  // bundle lives at cli/memcli/dist/memcli.js → ../../../ = repo root
  const scriptDir = path.dirname(new URL(import.meta.url).pathname);
  return path.resolve(scriptDir, '..', '..', '..');
}

// ---------------------------------------------------------------------------
// Subcommand handlers
// ---------------------------------------------------------------------------

function printHelp(): void {
  console.log(`
memcli — minimal self-hosted agent memory backed by trip2g

USAGE
  memcli [SUBCOMMAND] [FLAGS]

SUBCOMMANDS
  up (default)  Boot trip2g server + sync watcher, mint API key
  down          Stop the server and watcher
  status        Show container and watcher status
  logs          Stream container logs
  key           Re-mint the API key (rewrites data.json)

FLAGS (up)
  --folder <path>      Vault directory (required)
  --port <n>           Public port (default: ${DEFAULT_PORT})
  --email <addr>       Owner email (default: ${DEFAULT_EMAIL})
  --image <ref>        Docker image (default: ${DEFAULT_IMAGE})
  --public-url <url>   Override PUBLIC_URL (default: http://localhost:<port>)
  --no-hub             Skip writing the federation hub note (hub.md)
  --hub-url <url>      Override hub MCP endpoint (default: ${DEFAULT_HUB_URL})

GLOBAL FLAGS
  --dry-run    Print commands without executing
  --help       Show this help

NOTES
  State is stored in <vault>/.trip2g-memory/ (env file, data dir, PID file).
  The Docker image MUST include local-storage backend support (feat/filestorage).
`.trim());
}

async function cmdUp(flags: Flags, dryRun: boolean): Promise<void> {
  const { folder, port, email, image } = flags;
  if (!folder) {
    console.error('Error: --folder <vault> is required for `up`');
    process.exit(1);
  }

  const vault = path.resolve(folder);
  const stateDir = path.join(vault, '.trip2g-memory');
  const envFile = path.join(stateDir, 'env');
  const pidFile = path.join(stateDir, 'watch.pid');
  const iport = port + 1;
  const publicUrl = flags.publicUrl || `http://localhost:${port}`;

  const containerRunning = (() => {
    try {
      const out = spawnSync('docker', ['ps', '-q', '--filter', `name=${CONTAINER_NAME}`], {
        encoding: 'utf8',
      });
      return (out.stdout || '').trim().length > 0;
    } catch {
      return false;
    }
  })();

  let watcherAlive = false;
  if (fs.existsSync(pidFile)) {
    const pid = parseInt(fs.readFileSync(pidFile, 'utf8').trim(), 10);
    if (!isNaN(pid)) {
      watcherAlive = isPidAlive(pid, /trip2g-sync/);
    }
  }

  if (containerRunning && watcherAlive && !dryRun) {
    console.log(`trip2g-memory is already up. Web: ${publicUrl}`);
    return;
  }

  if (!dryRun) {
    fs.mkdirSync(stateDir, { recursive: true, mode: 0o700 });
    fs.chmodSync(stateDir, 0o700);
  }

  let secret: string;
  const existingEnv = dryRun ? {} : readEnvFile(envFile);
  if (existingEnv.JWT_SECRET) {
    secret = existingEnv.JWT_SECRET;
    console.log('Reusing existing JWT_SECRET from state dir.');
  } else {
    secret = crypto.randomBytes(32).toString('hex');
    if (!dryRun) {
      writeEnvFile(envFile, { JWT_SECRET: secret });
      console.log('Generated new JWT_SECRET and wrote to state dir.');
    } else {
      console.log('[dry-run] Would generate new JWT_SECRET and write to', envFile);
    }
  }

  const dockerArgs = buildDockerRunArgs({ port, iport, email, secret, stateDir, image });

  if (dryRun) {
    console.log(`[dry-run] docker run ${dockerArgs.join(' ')}`);
  } else if (!containerRunning) {
    console.log(`Starting container ${CONTAINER_NAME} (image: ${image})...`);
    console.log('NOTE: image must include feat/filestorage local-storage backend.');
    spawnSync('docker', ['run', ...dockerArgs], { encoding: 'utf8' });
  } else {
    console.log(`Container ${CONTAINER_NAME} already running, skipping docker run.`);
  }

  const readyzUrl = `http://localhost:${iport}/readyz`;
  if (dryRun) {
    console.log(`[dry-run] Would poll ${readyzUrl} until 200 (timeout ${READY_TIMEOUT_MS}ms)`);
  } else {
    console.log(`Waiting for server to be ready at ${readyzUrl} ...`);
    await waitReady(readyzUrl, READY_TIMEOUT_MS, READY_POLL_MS);
    console.log('Server is ready.');
  }

  let apiKey: string;
  if (dryRun) {
    console.log(
      `[dry-run] Would POST ${publicUrl}/_system/hat with HS256 JWT (email=${email}, ae=true, exp=now+300)`,
    );
    console.log(`[dry-run] Would POST ${publicUrl}/_system/graphql mutation createApiKey`);
    apiKey = '<api-key-would-be-minted>';
  } else {
    console.log('Minting admin API key via HAT...');
    const result = await mintApiKey(publicUrl, secret, email);
    apiKey = result.apiKey;
    console.log('API key minted.');

    // Persist key + id for re-mint / disable flow
    const stateJson = path.join(stateDir, 'state.json');
    let state: Record<string, unknown> = {};
    try {
      state = JSON.parse(fs.readFileSync(stateJson, 'utf8')) as Record<string, unknown>;
    } catch {
      // fresh state
    }
    state.apiKey = apiKey;
    if (result.apiKeyId != null) state.apiKeyId = result.apiKeyId;
    fs.writeFileSync(stateJson, JSON.stringify(state, null, 2), { mode: 0o600 });
  }

  if (dryRun) {
    const pluginDir = path.join(vault, '.obsidian', 'plugins', 'trip2g', 'data.json');
    console.log(`[dry-run] Would write ${pluginDir}`);
    console.log(
      '[dry-run] Content:',
      JSON.stringify(buildDataJson(vault, publicUrl, '<api-key>'), null, 2),
    );
  } else {
    const dataFile = writeDataJson(vault, publicUrl, apiKey);
    console.log(`Wrote plugin config to ${dataFile}`);
  }

  // Hub federation note
  const hubNotePath = path.join(vault, 'hub.md');
  if (!flags.noHub) {
    if (dryRun) {
      console.log(
        `[dry-run] would write hub note ${hubNotePath} (federates → ${flags.hubUrl})`,
      );
    } else if (fs.existsSync(hubNotePath)) {
      console.log(`Hub note already present at ${hubNotePath}, skipping.`);
    } else {
      fs.writeFileSync(hubNotePath, buildHubNote(flags.hubUrl), 'utf8');
      console.log(`Wrote hub note to ${hubNotePath} (federates → ${flags.hubUrl})`);
    }
  }

  const repo = repoRoot();
  const syncMjs = path.join(repo, 'obsidian-sync', 'dist', 'trip2g-sync.mjs');
  const watcherArgs = [
    syncMjs,
    '--watch',
    '--folder', vault,
    '--api-url', `${publicUrl}/_system/graphql`,
    '--api-key', apiKey,
  ];

  if (dryRun) {
    console.log(`[dry-run] spawn node ${watcherArgs.join(' ')} (detached, PID → ${pidFile})`);
  } else if (!watcherAlive) {
    console.log('Starting sync watcher...');
    const child = spawn(process.execPath, watcherArgs, {
      detached: true,
      stdio: 'ignore',
    });
    child.unref();
    fs.writeFileSync(pidFile, String(child.pid));
    console.log(`Watcher started (PID ${child.pid}), config written to ${pidFile}`);
  } else {
    console.log('Watcher already running.');
  }

  console.log('');
  console.log(`memory live — web: ${publicUrl}  read/write .md in ${vault}`);
}

function cmdDown(dryRun: boolean): void {
  if (dryRun) {
    console.log(`[dry-run] docker stop ${CONTAINER_NAME}`);
    console.log(`[dry-run] docker rm ${CONTAINER_NAME}`);
  } else {
    try {
      spawnSync('docker', ['stop', CONTAINER_NAME], { encoding: 'utf8' });
    } catch {
      // ignore
    }
    try {
      spawnSync('docker', ['rm', CONTAINER_NAME], { encoding: 'utf8' });
    } catch {
      // ignore
    }
    console.log(`Container ${CONTAINER_NAME} stopped and removed.`);
  }
}

function cmdStatus(): void {
  console.log('=== Container ===');
  const out = spawnSync(
    'docker',
    [
      'ps',
      '-a',
      '--filter',
      `name=${CONTAINER_NAME}`,
      '--format',
      'table {{.Names}}\t{{.Status}}\t{{.Ports}}',
    ],
    { encoding: 'utf8' },
  );
  console.log(out.stdout || '(no output)');
}

function cmdLogs(): void {
  const result = spawnSync('docker', ['logs', CONTAINER_NAME], {
    encoding: 'utf8',
    stdio: 'inherit',
  });
  if (result.error) throw result.error;
}

async function cmdKey(flags: Flags, dryRun: boolean): Promise<void> {
  const { folder, port, email } = flags;
  if (!folder) {
    console.error('Error: --folder <vault> is required for `key`');
    process.exit(1);
  }

  const vault = path.resolve(folder);
  const stateDir = path.join(vault, '.trip2g-memory');
  const envFile = path.join(stateDir, 'env');
  const stateJson = path.join(stateDir, 'state.json');
  const publicUrl = flags.publicUrl || `http://localhost:${port}`;

  const existingEnv = readEnvFile(envFile);
  if (!existingEnv.JWT_SECRET) {
    console.error('No existing JWT_SECRET found. Run `up` first.');
    process.exit(1);
  }
  const secret = existingEnv.JWT_SECRET;

  // Read prior state before minting so we can disable the old key afterwards.
  let prevState: Record<string, unknown> = {};
  try {
    prevState = JSON.parse(fs.readFileSync(stateJson, 'utf8')) as Record<string, unknown>;
  } catch {
    // no prior state
  }
  const priorApiKeyId =
    typeof prevState.apiKeyId === 'number' ? prevState.apiKeyId : null;

  if (dryRun) {
    console.log(`[dry-run] Would mint new API key via HAT for ${email}`);
    if (priorApiKeyId != null) {
      console.log(`[dry-run] Would disable prior API key id=${priorApiKeyId}`);
    }
    console.log(
      `[dry-run] Would rewrite ${path.join(vault, '.obsidian', 'plugins', 'trip2g', 'data.json')}`,
    );
    return;
  }

  console.log('Minting new API key...');
  const { apiKey, apiKeyId } = await mintApiKey(publicUrl, secret, email);

  // Persist new key + id before disabling old key (so state is never left empty).
  prevState.apiKey = apiKey;
  if (apiKeyId != null) prevState.apiKeyId = apiKeyId;
  fs.writeFileSync(stateJson, JSON.stringify(prevState, null, 2), { mode: 0o600 });

  writeDataJson(vault, publicUrl, apiKey);
  console.log('New API key written to data.json.');

  if (priorApiKeyId != null) {
    try {
      const gqlUrl = `${publicUrl}/_system/graphql`;
      const token = await hatAuth(publicUrl, secret, email);
      const data = await gqlRequest(gqlUrl, token, DisableApiKeyDocument, {
        id: priorApiKeyId,
      });
      const disResult = data.admin.disableApiKey;
      if (disResult.__typename === 'ErrorPayload') {
        console.warn(
          `Warning: could not disable prior API key id=${priorApiKeyId}: ${disResult.message}`,
        );
      } else {
        console.log(`Prior API key id=${priorApiKeyId} disabled.`);
      }
    } catch (err) {
      console.warn(
        `Warning: failed to disable prior API key id=${priorApiKeyId}: ${(err as Error).message}`,
      );
    }
  }
}

// ---------------------------------------------------------------------------
// Main entry point (only runs when executed directly, not when imported)
// ---------------------------------------------------------------------------

// Guard: only run when this file is the direct entry point.
// Covers: `node src/cli.ts`, `node dist/trip2g-memory.js`, and `trip2g-memory` bin.
// When imported by tests (node --test src/cli.test.ts), process.argv[1] points to
// the test runner and import.meta.url differs.
const _mainUrl = import.meta.url;
const _argv1Url = process.argv[1]
  ? (() => {
      try {
        return new URL(
          process.argv[1].startsWith('/') ? 'file://' + process.argv[1] : process.argv[1],
        ).href;
      } catch {
        return '';
      }
    })()
  : '';

if (_mainUrl === _argv1Url) {
  const argv = process.argv.slice(2);
  const { cmd, flags } = parseArgs(argv);

  if (flags.help) {
    printHelp();
    process.exit(0);
  }

  (async () => {
    try {
      switch (cmd) {
        case 'up':
          await cmdUp(flags, flags.dryRun);
          break;
        case 'down':
          cmdDown(flags.dryRun);
          break;
        case 'status':
          cmdStatus();
          break;
        case 'logs':
          cmdLogs();
          break;
        case 'key':
          await cmdKey(flags, flags.dryRun);
          break;
        default:
          console.error(`Unknown subcommand: ${cmd}`);
          printHelp();
          process.exit(1);
      }
    } catch (err) {
      console.error('Error:', (err as Error).message);
      process.exit(1);
    }
  })();
}
