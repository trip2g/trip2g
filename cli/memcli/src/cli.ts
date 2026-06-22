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
 *   trip2g-memory daily "<text>" [--folder <vault>] [--context <n>]
 *   trip2g-memory log <file> "<text>" [--folder <vault>] [--context <n>]
 *   trip2g-memory mcp   (or pipe stdin to run as MCP stdio server)
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
  folder: string;
  port: number;
  email: string;
  image: string;
  publicUrl: string | null;
  noHub: boolean;
  hubUrl: string;
  context: number;
}

export interface ServerEnv {
  LISTEN_ADDR: string;
  INTERNAL_LISTEN_ADDR: string;
  DB_FILE: string;
  OWNER_EMAIL: string;
  PUBLIC_URL: string;
  JWT_SECRET: string;
  DATA_ENCRYPTION_KEY: string;
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

/** Result returned by every runXxx() handler: collected output + error flag. */
export interface CommandResult {
  text: string;
  isError: boolean;
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
 * federates outbound to the given hub MCP endpoint. `free: true` marks the
 * KB-note as openly readable — REQUIRED for the federation scan to recognize
 * it (without it `accessibleKBNotes` excludes the note and federated_* tools
 * report "Federation is not configured").
 */
export function buildHubNote(hubUrl: string): string {
  const host = new URL(hubUrl).hostname;
  return `---
free: true
mcp_federation_kb_url: ${hubUrl}
mcp_federation_kb_id: ${host}
---

# Your trip2g memory

**trip2g** turns a folder of Markdown notes into living memory: you write notes, an agent reads and searches them as persistent long-term memory, and the same notes can be published as a website with subscriptions and a Telegram channel. It is a self-hosted Go + SQLite app.

## What you can do here
- **Remember** — write Markdown notes into this vault (\`memcli daily "…"\`, \`memcli log <topic> "…"\`, or just create \`.md\` files).
- **Recall** — grep/read the vault, or query this instance's \`/_system/mcp\` tools (\`search\`, \`expand\`).
- **Search the wider trip2g knowledge base** — this memory federates to **${host}**. Use the **\`federated_search\`** MCP tool to search ${host} and pull matching docs back, without leaving your own memory.

## Docs
- Long-term memory for agents — https://trip2g.com/en/user/agent_memory
- Local MCP server & tools — https://trip2g.com/en/user/mcp
- Federation — https://trip2g.com/en/user/federation
- How trip2g works — https://trip2g.com/en/user/protocol
- Self-hosting — https://trip2g.com/en/user/selfhosted

To disable federation: delete this file or run \`memcli up\` with \`--no-hub\`.
`.trimStart();
}

/**
 * Parse process.argv (or a provided array) into { cmd, flags, positional }.
 * Subcommands: up (default), down, status, logs, key, daily, log, mcp.
 *
 * daily: positional[0] = text
 * log:   positional[0] = file, positional[1] = text
 */
export function parseArgs(argv: string[]): { cmd: string; flags: Flags; positional: string[] } {
  const SUBCOMMANDS = new Set(['up', 'down', 'status', 'logs', 'key', 'daily', 'log', 'mcp']);
  const flags: Flags = {
    dryRun: false,
    help: false,
    folder: './memory-vault',
    port: DEFAULT_PORT,
    email: DEFAULT_EMAIL,
    image: DEFAULT_IMAGE,
    publicUrl: null,
    noHub: false,
    hubUrl: DEFAULT_HUB_URL,
    context: 15,
  };

  let cmd = 'up';
  let i = 0;
  const positional: string[] = [];

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
    } else if (arg === '--context') {
      flags.context = parseInt(argv[++i], 10);
    } else if (!arg.startsWith('-')) {
      positional.push(arg);
    }
    i++;
  }

  return { cmd, flags, positional };
}

/**
 * Determine whether to start the MCP stdio server.
 *
 * Rules:
 * - If argv[0] is a known non-mcp subcommand → false (run CLI as normal)
 * - If --help / -h is anywhere in argv → false (show help via CLI path)
 * - If argv[0] === 'mcp' → true
 * - If stdin is not a TTY (piped) and no non-mcp subcommand → true
 * - Otherwise → false (interactive TTY, show help / default behavior)
 */
export function shouldRunMcp(argv: string[], isTty: boolean): boolean {
  const KNOWN_CLI_CMDS = new Set(['up', 'down', 'status', 'logs', 'key', 'daily', 'log']);
  const first = argv[0];
  if (first !== undefined && KNOWN_CLI_CMDS.has(first)) return false;
  if (argv.includes('--help') || argv.includes('-h')) return false;
  if (first === 'mcp') return true;
  if (!isTty) return true;
  return false;
}

/**
 * Metadata for a single MCP tool exposed by this server.
 */
export interface ToolDef {
  name: string;
  description: string;
  inputSchema: Record<string, unknown>;
}

/**
 * Return the static list of MCP tools this server exposes.
 * Pure function: no side-effects, suitable for unit testing.
 */
export function buildToolList(): ToolDef[] {
  return [
    {
      name: 'memory_up',
      description:
        'Start the local trip2g memory: boots a server (local filesystem storage, no S3 or dev mode), ' +
        'mints an admin key via HAT, starts a two-way sync watcher, and drops a hub note federating to trip2g.com. ' +
        'Idempotent — safe to call if already running. Call once before remembering/recalling. ' +
        'Arg: folder (vault path, default ./memory-vault).',
      inputSchema: {
        type: 'object',
        properties: {
          folder: { type: 'string', description: 'Vault directory path (default: ./memory-vault)' },
          port: { type: 'number', description: 'Public port (default: 24081)' },
          email: { type: 'string', description: 'Owner email (default: memory@local)' },
          image: { type: 'string', description: 'Docker image ref (default: ghcr.io/trip2g/trip2g:latest)' },
          noHub: { type: 'boolean', description: 'Skip writing the federation hub note' },
          hubUrl: { type: 'string', description: 'Override hub MCP endpoint URL' },
          publicUrl: { type: 'string', description: 'Override PUBLIC_URL for the server' },
        },
        required: [],
      },
    },
    {
      name: 'memory_down',
      description: 'Stop the memory server container and the sync watcher for a vault. Arg: folder.',
      inputSchema: {
        type: 'object',
        properties: {
          folder: { type: 'string', description: 'Vault directory path (default: ./memory-vault)' },
        },
        required: [],
      },
    },
    {
      name: 'memory_status',
      description: 'Report whether the memory server and sync watcher are running for a vault. Arg: folder.',
      inputSchema: {
        type: 'object',
        properties: {
          folder: { type: 'string', description: 'Vault directory path (default: ./memory-vault)' },
        },
        required: [],
      },
    },
    {
      name: 'memory_logs',
      description: 'Show recent server logs (docker logs snapshot) for diagnostics. Arg: folder.',
      inputSchema: {
        type: 'object',
        properties: {
          folder: { type: 'string', description: 'Vault directory path (default: ./memory-vault)' },
        },
        required: [],
      },
    },
    {
      name: 'memory_key',
      description:
        'Rotate the admin API key (mints a new one, disables the previous). ' +
        'Use if the key leaked or you need a fresh one. Arg: folder.',
      inputSchema: {
        type: 'object',
        properties: {
          folder: { type: 'string', description: 'Vault directory path (default: ./memory-vault)' },
          port: { type: 'number', description: 'Public port (default: 24081)' },
          email: { type: 'string', description: 'Owner email (default: memory@local)' },
          publicUrl: { type: 'string', description: 'Override PUBLIC_URL' },
        },
        required: [],
      },
    },
    {
      name: 'memory_daily',
      description:
        "Append a thought to TODAY'S daily note — the day's general working space for capturing thoughts as they come. " +
        'The first entry of the day is plain; later same-day entries are timestamped HH:MM. ' +
        "Use this to 'remember' something now. " +
        'Args: text (the thought), folder, context (lines of the note to echo back).',
      inputSchema: {
        type: 'object',
        properties: {
          text: { type: 'string', description: 'The thought to append (use \\n for newlines)' },
          folder: { type: 'string', description: 'Vault directory path (default: ./memory-vault)' },
          context: { type: 'number', description: 'Lines of note context to return after write (default: 15)' },
        },
        required: ['text'],
      },
    },
    {
      name: 'memory_log',
      description:
        "Append a thought to a specific note's running log, under a '### [[today]]' day header — " +
        'an append-only journal of how ONE idea/topic evolves over days. ' +
        'First entry under a new day is plain; later same-day entries get HH:MM. ' +
        "Use this (not memory_daily) when you're tracking the evolution of a specific named topic over time. " +
        'Args: file (note name without .md), text, folder, context.',
      inputSchema: {
        type: 'object',
        properties: {
          file: { type: 'string', description: 'Note name without .md extension (e.g. "work", "project-ideas")' },
          text: { type: 'string', description: 'The thought to append (use \\n for newlines)' },
          folder: { type: 'string', description: 'Vault directory path (default: ./memory-vault)' },
          context: { type: 'number', description: 'Lines of note context to return after write (default: 15)' },
        },
        required: ['file', 'text'],
      },
    },
  ];
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
  encryptionKey: string;
}): ServerEnv {
  const { port, iport, email, secret, encryptionKey } = opts;
  const env: ServerEnv = {
    LISTEN_ADDR: `0.0.0.0:${port}`,
    INTERNAL_LISTEN_ADDR: `:${iport}`,
    DB_FILE: '/data/local.sqlite3',
    OWNER_EMAIL: email,
    PUBLIC_URL: `http://localhost:${port}`,
    JWT_SECRET: secret,
    DATA_ENCRYPTION_KEY: encryptionKey,
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
  encryptionKey: string;
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
// Daily / Log helpers (exported for unit tests)
// ---------------------------------------------------------------------------

/**
 * Resolve the timezone label for the daily-note frontmatter.
 * Uses TZ env var if set, then Intl to detect the system zone, then UTC.
 */
function _tzLabel(): string {
  const tz = process.env.TZ?.trim();
  if (tz) return tz;
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
  } catch {
    return 'UTC';
  }
}

/**
 * Prefix the first line of text with "HH:MM " from the given Date.
 * Normalizes a literal backslash-n ("\\n") in the text to a real newline
 * before splitting, so agents can pass multi-line text through a shell.
 */
export function _stampBlock(text: string, now: Date): string {
  const normalized = text.replace(/\\n/g, '\n');
  const lines = normalized.split('\n');
  const hhmm = now.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', hour12: false });
  lines[0] = `${hhmm} ${lines[0]}`;
  return lines.join('\n');
}

/**
 * Normalize text (backslash-n → real newline) without adding a timestamp.
 * Used for the first entry of a day, where no HH:MM prefix is written.
 */
export function _plainBlock(text: string): string {
  return text.replace(/\\n/g, '\n');
}

/**
 * Write content to path atomically: write to a sibling temp file, then rename.
 * Creates parent directories as needed.
 */
export function atomicWrite(filePath: string, content: string): void {
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });
  const tmp = path.join(dir, `.memcli-${process.pid}-${Date.now()}.tmp`);
  try {
    fs.writeFileSync(tmp, content, 'utf8');
    fs.renameSync(tmp, filePath);
  } catch (err) {
    try { fs.unlinkSync(tmp); } catch { /* ignore */ }
    throw err;
  }
}

/**
 * Return the last `n` lines of a file as a string.
 * Returns empty string if the file does not exist or n <= 0.
 */
function tailLines(filePath: string, n: number): string {
  if (n <= 0) return '';
  if (!fs.existsSync(filePath)) return '';
  const lines = fs.readFileSync(filePath, 'utf8').split('\n');
  return lines.slice(-n).join('\n');
}

/**
 * Build the initial content for a new daily note.
 * Includes frontmatter, navigation header, and the first stamped entry.
 */
export function buildDailyEntry(day: string, tz: string, stampedEntry: string): string {
  const frontmatter = `---\ntimezone: ${tz}\n---\n`;
  const header = `- [[_index|Home]]\n- [[daily/_index|Daily]]\n\n# ${day}\n`;
  return `${frontmatter}${header}\n${stampedEntry}\n`;
}

/**
 * Ensure vault/daily/_index.md exists and contains a `- [[<day>]]` link
 * under the `# Daily` heading. Port of _ensure_daily_index.
 */
export function ensureDailyIndex(vault: string, day: string): void {
  const idx = path.join(vault, 'daily', '_index.md');
  if (!fs.existsSync(idx)) {
    atomicWrite(idx, '- [[_index|Home]]\n- [[daily/_index|Daily]]\n\n# Daily\n');
  }
  const text = fs.readFileSync(idx, 'utf8');
  if (text.includes(`[[${day}]]`)) return;

  const lines = text.split('\n');
  const out: string[] = [];
  let inserted = false;
  for (const line of lines) {
    out.push(line);
    if (!inserted && line.trim() === '# Daily') {
      out.push(`- [[${day}]]`);
      inserted = true;
    }
  }
  if (!inserted) out.push(`- [[${day}]]`);
  // Remove trailing empty line added by split, then rejoin
  atomicWrite(idx, out.join('\n') + '\n');
}

/**
 * Append a timestamped entry to vault/daily/<day>.md (creating the note if needed).
 * Also maintains vault/daily/_index.md.
 * Note: sync is NOT called here — the running trip2g-sync --watch daemon picks up the change.
 *
 * @returns path to the note file
 */
export function appendDaily(vault: string, text: string, now: Date): string {
  // Use local date to match the user's timezone
  const localDay = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, '0'),
    String(now.getDate()).padStart(2, '0'),
  ].join('-');
  const note = path.join(vault, 'daily', `${localDay}.md`);
  if (!fs.existsSync(note)) {
    // First entry of the day: no HH:MM prefix
    const entry = _plainBlock(text);
    atomicWrite(note, buildDailyEntry(localDay, _tzLabel(), entry));
  } else {
    // Subsequent entries: add HH:MM prefix
    const entry = _stampBlock(text, now);
    const existing = fs.readFileSync(note, 'utf8').replace(/\n+$/, '');
    atomicWrite(note, `${existing}\n\n${entry}\n`);
  }
  ensureDailyIndex(vault, localDay);
  return note;
}

/**
 * Append a timestamped entry under today's `### [[<day>]]` section in vault/<file>.md.
 * Creates the section at end of file if absent.
 * Note: sync is NOT called here — the running trip2g-sync --watch daemon picks up the change.
 *
 * @returns path to the note file
 */
export function appendLog(vault: string, file: string, text: string, now: Date): string {
  const localDay = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, '0'),
    String(now.getDate()).padStart(2, '0'),
  ].join('-');
  const fileMd = file.endsWith('.md') ? file : `${file}.md`;
  const note = path.join(vault, fileMd);
  const header = `### [[${localDay}]]`;

  const existing = fs.existsSync(note) ? fs.readFileSync(note, 'utf8') : '';
  if (existing.split('\n').includes(header)) {
    // Section already exists: subsequent entry, add HH:MM prefix
    const entry = _stampBlock(text, now);
    const base = existing.replace(/\n+$/, '');
    atomicWrite(note, `${base}\n\n${entry}\n`);
  } else {
    // First entry under this day's section: no HH:MM prefix
    const entry = _plainBlock(text);
    const base = existing.replace(/\n+$/, '');
    const prefix = base ? `${base}\n\n` : '';
    atomicWrite(note, `${prefix}${header}\n\n${entry}\n`);
  }
  return note;
}

// ---------------------------------------------------------------------------
// Subcommand runner functions (return CommandResult, no process.exit)
// ---------------------------------------------------------------------------

/** Capture lines written by fn() into a string. */
function captureLines(fn: () => void): string {
  const lines: string[] = [];
  const origLog = console.log;
  const origErr = console.error;
  const origWarn = console.warn;
  console.log = (...args: unknown[]) => lines.push(args.map(String).join(' '));
  console.error = (...args: unknown[]) => lines.push(args.map(String).join(' '));
  console.warn = (...args: unknown[]) => lines.push(args.map(String).join(' '));
  try {
    fn();
  } finally {
    console.log = origLog;
    console.error = origErr;
    console.warn = origWarn;
  }
  return lines.join('\n');
}

async function captureAsync(fn: () => Promise<void>): Promise<string> {
  const lines: string[] = [];
  const origLog = console.log;
  const origErr = console.error;
  const origWarn = console.warn;
  console.log = (...args: unknown[]) => lines.push(args.map(String).join(' '));
  console.error = (...args: unknown[]) => lines.push(args.map(String).join(' '));
  console.warn = (...args: unknown[]) => lines.push(args.map(String).join(' '));
  try {
    await fn();
  } finally {
    console.log = origLog;
    console.error = origErr;
    console.warn = origWarn;
  }
  return lines.join('\n');
}

export async function runUp(flags: Flags): Promise<CommandResult> {
  try {
    const text = await captureAsync(() => cmdUp(flags, flags.dryRun));
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${(err as Error).message}`, isError: true };
  }
}

export function runDown(flags: Flags): CommandResult {
  try {
    const text = captureLines(() => cmdDown(flags.dryRun, flags.folder));
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${(err as Error).message}`, isError: true };
  }
}

export function runStatus(): CommandResult {
  try {
    const text = captureLines(() => cmdStatus());
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${(err as Error).message}`, isError: true };
  }
}

export function runLogs(): CommandResult {
  // For MCP mode: capture output via spawnSync with pipe instead of inherit
  try {
    const result = spawnSync('docker', ['logs', CONTAINER_NAME], {
      encoding: 'utf8',
    });
    if (result.error) throw result.error;
    const text = [result.stdout, result.stderr].filter(Boolean).join('\n');
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${(err as Error).message}`, isError: true };
  }
}

export async function runKey(flags: Flags): Promise<CommandResult> {
  try {
    const text = await captureAsync(() => cmdKey(flags, flags.dryRun));
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${(err as Error).message}`, isError: true };
  }
}

export function runDaily(vault: string, text: string, context: number): CommandResult {
  if (!text) {
    return { text: 'Error: `daily` requires a text argument', isError: true };
  }
  try {
    const note = appendDaily(path.resolve(vault), text, new Date());
    return { text: tailLines(note, context), isError: false };
  } catch (err) {
    return { text: `Error: ${(err as Error).message}`, isError: true };
  }
}

export function runLog(vault: string, file: string, text: string, context: number): CommandResult {
  if (!file || !text) {
    return { text: 'Error: `log` requires <file> and <text> arguments', isError: true };
  }
  try {
    const note = appendLog(path.resolve(vault), file, text, new Date());
    return { text: tailLines(note, context), isError: false };
  } catch (err) {
    return { text: `Error: ${(err as Error).message}`, isError: true };
  }
}

// ---------------------------------------------------------------------------
// Legacy command wrappers (print + exit — used by the CLI path only)
// ---------------------------------------------------------------------------

function cmdDaily(vault: string, text: string, context: number): void {
  if (!text) {
    console.error('Error: `daily` requires a text argument');
    process.exit(1);
  }
  const note = appendDaily(path.resolve(vault), text, new Date());
  console.log(tailLines(note, context));
}

function cmdLog(vault: string, file: string, text: string, context: number): void {
  if (!file || !text) {
    console.error('Error: `log` requires <file> and <text> arguments');
    process.exit(1);
  }
  const note = appendLog(path.resolve(vault), file, text, new Date());
  console.log(tailLines(note, context));
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
  daily "<text>"         Append a HH:MM entry to today's daily note
  log <file> "<text>"    Append a HH:MM entry under today's ### [[date]] section
  mcp           Run as MCP stdio server (also auto-detected when stdin is piped)

FLAGS (up)
  --folder <path>      Vault directory (default: ./memory-vault)
  --port <n>           Public port (default: ${DEFAULT_PORT})
  --email <addr>       Owner email (default: ${DEFAULT_EMAIL})
  --image <ref>        Docker image (default: ${DEFAULT_IMAGE})
  --public-url <url>   Override PUBLIC_URL (default: http://localhost:<port>)
  --no-hub             Skip writing the federation hub note (hub.md)
  --hub-url <url>      Override hub MCP endpoint (default: ${DEFAULT_HUB_URL})

FLAGS (daily / log)
  --folder <path>      Vault directory (default: ./memory-vault)
  --context <n>        Lines of note context to print after write (default: 15)

GLOBAL FLAGS
  --dry-run    Print commands without executing
  --help       Show this help

NOTES
  State is stored in <vault>/.trip2g-memory/ (env file, data dir, PID file).
  The Docker image MUST include local-storage backend support (feat/filestorage).
  daily/log do not call sync — the running trip2g-sync --watch daemon picks up changes.
  Pipe stdin or use the mcp subcommand to run as an MCP stdio JSON-RPC server.
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
  let encryptionKey: string;
  const existingEnv = dryRun ? {} : readEnvFile(envFile);
  const needsWrite = !existingEnv.JWT_SECRET || !existingEnv.DATA_ENCRYPTION_KEY;
  if (existingEnv.JWT_SECRET) {
    secret = existingEnv.JWT_SECRET;
    console.log('Reusing existing JWT_SECRET from state dir.');
  } else {
    secret = crypto.randomBytes(32).toString('hex');
  }
  if (existingEnv.DATA_ENCRYPTION_KEY) {
    encryptionKey = existingEnv.DATA_ENCRYPTION_KEY;
    console.log('Reusing existing DATA_ENCRYPTION_KEY from state dir.');
  } else {
    // AES-256 requires a 32-byte key; hex-encoding 16 random bytes yields 32 ASCII chars
    encryptionKey = crypto.randomBytes(16).toString('hex');
  }
  if (!dryRun && needsWrite) {
    writeEnvFile(envFile, { JWT_SECRET: secret, DATA_ENCRYPTION_KEY: encryptionKey });
    console.log('Generated secrets and wrote to state dir.');
  } else if (dryRun && needsWrite) {
    console.log('[dry-run] Would generate JWT_SECRET and DATA_ENCRYPTION_KEY and write to', envFile);
  }

  const dockerArgs = buildDockerRunArgs({ port, iport, email, secret, encryptionKey, stateDir, image });

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

function cmdDown(dryRun: boolean, folder?: string): void {
  if (dryRun) {
    console.log(`[dry-run] docker stop ${CONTAINER_NAME}`);
    console.log(`[dry-run] docker rm ${CONTAINER_NAME}`);
    if (folder) {
      const pidFile = path.join(path.resolve(folder), '.trip2g-memory', 'watch.pid');
      console.log(`[dry-run] Would kill watcher PID from ${pidFile} and remove pid file`);
    }
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

    // Kill the detached watcher process if a pid file exists
    if (folder) {
      const pidFile = path.join(path.resolve(folder), '.trip2g-memory', 'watch.pid');
      if (fs.existsSync(pidFile)) {
        const raw = fs.readFileSync(pidFile, 'utf8').trim();
        const pid = parseInt(raw, 10);
        if (!isNaN(pid) && isPidAlive(pid, /trip2g-sync/)) {
          try {
            process.kill(pid, 'SIGTERM');
            console.log(`Watcher process (PID ${pid}) terminated.`);
          } catch {
            // ignore if already gone
          }
        }
        try {
          fs.unlinkSync(pidFile);
        } catch {
          // ignore
        }
      }
    }
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
// MCP stdio server (hand-rolled JSON-RPC 2.0 over stdio, zero external deps)
// ---------------------------------------------------------------------------

interface JsonRpcRequest {
  jsonrpc: '2.0';
  id?: string | number | null;
  method: string;
  params?: unknown;
}

interface JsonRpcResponse {
  jsonrpc: '2.0';
  id: string | number | null;
  result?: unknown;
  error?: { code: number; message: string };
}

function mcpSend(resp: JsonRpcResponse): void {
  process.stdout.write(JSON.stringify(resp) + '\n');
}

function mcpError(id: string | number | null, code: number, message: string): void {
  mcpSend({ jsonrpc: '2.0', id, error: { code, message } });
}

function defaultFlags(): Flags {
  return {
    dryRun: false,
    help: false,
    folder: './memory-vault',
    port: DEFAULT_PORT,
    email: DEFAULT_EMAIL,
    image: DEFAULT_IMAGE,
    publicUrl: null,
    noHub: false,
    hubUrl: DEFAULT_HUB_URL,
    context: 15,
  };
}

async function dispatchMcpTool(
  name: string,
  args: Record<string, unknown>,
): Promise<{ content: Array<{ type: 'text'; text: string }>; isError?: boolean }> {
  function flagsFrom(a: Record<string, unknown>): Flags {
    const f = defaultFlags();
    if (typeof a.folder === 'string') f.folder = a.folder;
    if (typeof a.port === 'number') f.port = a.port;
    if (typeof a.email === 'string') f.email = a.email;
    if (typeof a.image === 'string') f.image = a.image;
    if (typeof a.publicUrl === 'string') f.publicUrl = a.publicUrl;
    if (typeof a.noHub === 'boolean') f.noHub = a.noHub;
    if (typeof a.hubUrl === 'string') f.hubUrl = a.hubUrl;
    return f;
  }

  let result: CommandResult;
  switch (name) {
    case 'memory_up':
      result = await runUp(flagsFrom(args));
      break;
    case 'memory_down':
      result = runDown(flagsFrom(args));
      break;
    case 'memory_status':
      result = runStatus();
      break;
    case 'memory_logs':
      result = runLogs();
      break;
    case 'memory_key':
      result = await runKey(flagsFrom(args));
      break;
    case 'memory_daily': {
      const folder = typeof args.folder === 'string' ? args.folder : './memory-vault';
      const context = typeof args.context === 'number' ? args.context : 15;
      const text = typeof args.text === 'string' ? args.text : '';
      result = runDaily(folder, text, context);
      break;
    }
    case 'memory_log': {
      const folder = typeof args.folder === 'string' ? args.folder : './memory-vault';
      const context = typeof args.context === 'number' ? args.context : 15;
      const file = typeof args.file === 'string' ? args.file : '';
      const text = typeof args.text === 'string' ? args.text : '';
      result = runLog(folder, file, text, context);
      break;
    }
    default:
      return {
        content: [{ type: 'text', text: `Unknown tool: ${name}` }],
        isError: true,
      };
  }

  return {
    content: [{ type: 'text', text: result.text }],
    ...(result.isError ? { isError: true } : {}),
  };
}

async function startMcpServer(): Promise<void> {
  const { createInterface } = await import('node:readline');

  const rl = createInterface({ input: process.stdin, crlfDelay: Infinity });

  for await (const line of rl) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    let msg: JsonRpcRequest;
    try {
      msg = JSON.parse(trimmed) as JsonRpcRequest;
    } catch {
      // Can't recover id from malformed JSON — send parse error with null id
      mcpError(null, -32700, 'Parse error');
      continue;
    }

    const id = msg.id ?? null;

    // Notifications (no id field) → no response
    if (!('id' in msg)) continue;

    const method = msg.method;

    if (method === 'initialize') {
      mcpSend({
        jsonrpc: '2.0',
        id,
        result: {
          protocolVersion: '2024-11-05',
          capabilities: { tools: {} },
          serverInfo: { name: 'memcli', version: '0.1.0' },
        },
      });
    } else if (method === 'tools/list') {
      mcpSend({ jsonrpc: '2.0', id, result: { tools: buildToolList() } });
    } else if (method === 'tools/call') {
      const params = (msg.params ?? {}) as Record<string, unknown>;
      const toolName = typeof params.name === 'string' ? params.name : '';
      const toolArgs = (params.arguments ?? {}) as Record<string, unknown>;
      try {
        const toolResult = await dispatchMcpTool(toolName, toolArgs);
        mcpSend({ jsonrpc: '2.0', id, result: toolResult });
      } catch (err) {
        mcpSend({
          jsonrpc: '2.0',
          id,
          result: {
            content: [{ type: 'text', text: `Error: ${(err as Error).message}` }],
            isError: true,
          },
        });
      }
    } else {
      mcpError(id, -32601, 'Method not found');
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
  const isTty = process.stdin.isTTY === true;

  // MCP mode: piped stdin with no explicit subcommand, or explicit `mcp` subcommand
  if (shouldRunMcp(argv, isTty)) {
    startMcpServer().catch((err) => {
      process.stderr.write(`MCP server error: ${(err as Error).message}\n`);
      process.exit(1);
    });
  } else {
    const { cmd, flags, positional } = parseArgs(argv);

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
            cmdDown(flags.dryRun, flags.folder);
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
          case 'daily':
            cmdDaily(flags.folder, positional[0] ?? '', flags.context);
            break;
          case 'log':
            cmdLog(flags.folder, positional[0] ?? '', positional[1] ?? '', flags.context);
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
}
