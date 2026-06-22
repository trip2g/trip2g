/**
 * Tests for trip2g-memory pure helpers.
 * Run with: node --experimental-strip-types --test src/cli.test.ts
 * Zero npm dependencies — uses node:test + node:assert only.
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {
  signHatJwt,
  parseArgs,
  buildServerEnv,
  buildDataJson,
  buildDockerRunArgs,
  buildHubNote,
  hubSlug,
  _stampBlock,
  buildDailyEntry,
  appendDaily,
  appendLog,
  ensureDailyIndex,
  shouldRunMcp,
  buildToolList,
} from './cli.ts';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function decodeBase64url(s: string): Buffer {
  const padded = s.replace(/-/g, '+').replace(/_/g, '/');
  const remainder = padded.length % 4;
  const pad = remainder === 0 ? '' : '='.repeat(4 - remainder);
  return Buffer.from(padded + pad, 'base64');
}

function hmacBase64url(secret: string, signingInput: string): string {
  const sig = crypto.createHmac('sha256', secret).update(signingInput).digest();
  return sig.toString('base64url');
}

// ---------------------------------------------------------------------------
// signHatJwt
// ---------------------------------------------------------------------------

test('signHatJwt: produces a three-segment JWT string', () => {
  const jwt = signHatJwt('mysecret', 'admin@example.com');
  const parts = jwt.split('.');
  assert.equal(parts.length, 3, 'JWT must have 3 segments');
});

test('signHatJwt: header is {alg:"HS256",typ:"JWT"}', () => {
  const jwt = signHatJwt('mysecret', 'admin@example.com');
  const header = JSON.parse(decodeBase64url(jwt.split('.')[0]).toString('utf8')) as Record<string, unknown>;
  assert.equal(header.alg, 'HS256');
  assert.equal(header.typ, 'JWT');
});

test('signHatJwt: payload has correct e, ae=true, and exp in ~5 min', () => {
  const before = Math.floor(Date.now() / 1000);
  const jwt = signHatJwt('mysecret', 'admin@example.com');
  const after = Math.floor(Date.now() / 1000);
  const payload = JSON.parse(decodeBase64url(jwt.split('.')[1]).toString('utf8')) as Record<string, unknown>;

  assert.equal(payload.e, 'admin@example.com');
  assert.equal(payload.ae, true);
  assert.ok(typeof payload.exp === 'number', 'exp must be a number');
  const exp = payload.exp as number;
  assert.ok(exp >= before + 298, `exp ${exp} too small (before+298=${before + 298})`);
  assert.ok(exp <= after + 302, `exp ${exp} too large (after+302=${after + 302})`);
});

test('signHatJwt: signature verifies with the same secret', () => {
  const secret = 'test-secret-value';
  const jwt = signHatJwt(secret, 'admin@example.com');
  const parts = jwt.split('.');
  const signingInput = parts[0] + '.' + parts[1];
  const expectedSig = hmacBase64url(secret, signingInput);
  assert.equal(parts[2], expectedSig, 'Signature must match recomputed HMAC-SHA256');
});

test('signHatJwt: different secrets produce different signatures', () => {
  const j1 = signHatJwt('secret-A', 'user@x.com');
  const j2 = signHatJwt('secret-B', 'user@x.com');
  assert.notEqual(j1.split('.')[2], j2.split('.')[2]);
});

// ---------------------------------------------------------------------------
// parseArgs
// ---------------------------------------------------------------------------

test('parseArgs: default subcommand is "up"', () => {
  const { cmd } = parseArgs([]);
  assert.equal(cmd, 'up');
});

test('parseArgs: "up" subcommand', () => {
  const { cmd } = parseArgs(['up']);
  assert.equal(cmd, 'up');
});

test('parseArgs: "down" subcommand', () => {
  const { cmd } = parseArgs(['down']);
  assert.equal(cmd, 'down');
});

test('parseArgs: "status" subcommand', () => {
  const { cmd } = parseArgs(['status']);
  assert.equal(cmd, 'status');
});

test('parseArgs: "logs" subcommand', () => {
  const { cmd } = parseArgs(['logs']);
  assert.equal(cmd, 'logs');
});

test('parseArgs: "key" subcommand', () => {
  const { cmd } = parseArgs(['key']);
  assert.equal(cmd, 'key');
});

test('parseArgs: --dry-run flag', () => {
  const { flags } = parseArgs(['up', '--dry-run']);
  assert.equal(flags.dryRun, true);
});

test('parseArgs: --help flag', () => {
  const { flags } = parseArgs(['--help']);
  assert.equal(flags.help, true);
});

test('parseArgs: --folder flag', () => {
  const { flags } = parseArgs(['up', '--folder', '/tmp/vault']);
  assert.equal(flags.folder, '/tmp/vault');
});

test('parseArgs: --port flag', () => {
  const { flags } = parseArgs(['up', '--port', '9999']);
  assert.equal(flags.port, 9999);
});

test('parseArgs: --email flag', () => {
  const { flags } = parseArgs(['up', '--email', 'me@example.com']);
  assert.equal(flags.email, 'me@example.com');
});

test('parseArgs: --image flag', () => {
  const { flags } = parseArgs(['up', '--image', 'trip2g:local']);
  assert.equal(flags.image, 'trip2g:local');
});

test('parseArgs: --public-url flag', () => {
  const { flags } = parseArgs(['up', '--public-url', 'https://example.com']);
  assert.equal(flags.publicUrl, 'https://example.com');
});

// ---------------------------------------------------------------------------
// buildServerEnv
// ---------------------------------------------------------------------------

const FORBIDDEN_KEYS = ['DEV', 'RESEND_API_KEY', 'SMTP_PASSWORD', 'GIT_API_REPO_PATH'];

const TEST_ENC_KEY = crypto.randomBytes(16).toString('hex'); // 32 ASCII chars = 32 bytes

test('buildServerEnv: STORAGE_BACKEND is "local"', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.equal(env.STORAGE_BACKEND, 'local');
});

test('buildServerEnv: STORAGE_LOCAL_DIR is set', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.ok(env.STORAGE_LOCAL_DIR, 'STORAGE_LOCAL_DIR must be non-empty');
});

test('buildServerEnv: contains LISTEN_ADDR with correct port', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.ok(env.LISTEN_ADDR.includes('24081'), `LISTEN_ADDR should include port 24081, got: ${env.LISTEN_ADDR}`);
});

test('buildServerEnv: contains INTERNAL_LISTEN_ADDR with iport', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.ok(env.INTERNAL_LISTEN_ADDR.includes('24082'), `INTERNAL_LISTEN_ADDR should include 24082, got: ${env.INTERNAL_LISTEN_ADDR}`);
});

test('buildServerEnv: contains JWT_SECRET', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'hunter2', encryptionKey: TEST_ENC_KEY });
  assert.equal(env.JWT_SECRET, 'hunter2');
});

test('buildServerEnv: contains OWNER_EMAIL', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'owner@test.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.equal(env.OWNER_EMAIL, 'owner@test.com');
});

test('buildServerEnv: contains DB_FILE', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.ok(env.DB_FILE, 'DB_FILE must be non-empty');
});

test('buildServerEnv: contains PUBLIC_URL with correct port', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.ok(env.PUBLIC_URL.includes('24081'), `PUBLIC_URL should include port 24081, got: ${env.PUBLIC_URL}`);
});

test('buildServerEnv: contains DATA_ENCRYPTION_KEY matching the provided value', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.equal(env.DATA_ENCRYPTION_KEY, TEST_ENC_KEY, 'DATA_ENCRYPTION_KEY must match provided value');
});

test('buildServerEnv: DATA_ENCRYPTION_KEY is exactly 32 bytes (chars)', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.equal(
    Buffer.byteLength(env.DATA_ENCRYPTION_KEY, 'utf8'),
    32,
    `DATA_ENCRYPTION_KEY must be 32 bytes, got ${Buffer.byteLength(env.DATA_ENCRYPTION_KEY, 'utf8')}`,
  );
});

test('buildServerEnv: DATA_ENCRYPTION_KEY is not the default placeholder', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY });
  assert.notEqual(env.DATA_ENCRYPTION_KEY, 'please-change-me-to-32-byte-key!', 'DATA_ENCRYPTION_KEY must not be the default value');
});

for (const key of FORBIDDEN_KEYS) {
  test(`buildServerEnv: does NOT contain ${key}`, () => {
    const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x', encryptionKey: TEST_ENC_KEY }) as unknown as Record<string, unknown>;
    assert.equal(env[key], undefined, `${key} must not be present in server env`);
  });
}

// ---------------------------------------------------------------------------
// buildDockerRunArgs
// ---------------------------------------------------------------------------

test('buildDockerRunArgs: binds to 127.0.0.1 (loopback only)', () => {
  const args = buildDockerRunArgs({
    port: 24081,
    iport: 24082,
    email: 'a@b.com',
    secret: 'x',
    encryptionKey: TEST_ENC_KEY,
    stateDir: '/tmp/state',
    image: 'ghcr.io/trip2g/trip2g:latest',
  });
  const portArg = args.find((a) => a.startsWith('127.0.0.1:'));
  assert.ok(portArg, `Expected a 127.0.0.1:<port>:<port> -p argument, args: ${JSON.stringify(args)}`);
});

test('buildDockerRunArgs: includes -d flag', () => {
  const args = buildDockerRunArgs({
    port: 24081,
    iport: 24082,
    email: 'a@b.com',
    secret: 'x',
    encryptionKey: TEST_ENC_KEY,
    stateDir: '/tmp/state',
    image: 'ghcr.io/trip2g/trip2g:latest',
  });
  assert.ok(args.includes('-d'), 'Must include -d for detached mode');
});

test('buildDockerRunArgs: includes --name trip2g-memory', () => {
  const args = buildDockerRunArgs({
    port: 24081,
    iport: 24082,
    email: 'a@b.com',
    secret: 'x',
    encryptionKey: TEST_ENC_KEY,
    stateDir: '/tmp/state',
    image: 'ghcr.io/trip2g/trip2g:latest',
  });
  const nameIdx = args.indexOf('--name');
  assert.ok(nameIdx >= 0, 'Must have --name flag');
  assert.equal(args[nameIdx + 1], 'trip2g-memory');
});

test('buildDockerRunArgs: image is last argument', () => {
  const image = 'ghcr.io/trip2g/trip2g:latest';
  const args = buildDockerRunArgs({
    port: 24081,
    iport: 24082,
    email: 'a@b.com',
    secret: 'x',
    encryptionKey: TEST_ENC_KEY,
    stateDir: '/tmp/state',
    image,
  });
  assert.equal(args[args.length - 1], image, 'Image must be the last argument');
});

test('buildDockerRunArgs: volume mounts stateDir/data to /data', () => {
  const stateDir = '/my/state';
  const args = buildDockerRunArgs({
    port: 24081,
    iport: 24082,
    email: 'a@b.com',
    secret: 'x',
    encryptionKey: TEST_ENC_KEY,
    stateDir,
    image: 'trip2g:local',
  });
  const vIdx = args.indexOf('-v');
  assert.ok(vIdx >= 0, 'Must have -v flag');
  const vol = args[vIdx + 1];
  assert.ok(vol.startsWith(stateDir + '/data:'), `Volume must start with ${stateDir}/data:, got: ${vol}`);
});

test('buildDockerRunArgs: does not include DEV=true in env args', () => {
  const args = buildDockerRunArgs({
    port: 24081,
    iport: 24082,
    email: 'a@b.com',
    secret: 'x',
    encryptionKey: TEST_ENC_KEY,
    stateDir: '/tmp/state',
    image: 'trip2g:local',
  });
  const envArgs = args.filter((_, i) => args[i - 1] === '-e');
  const hasDev = envArgs.some((v) => v.startsWith('DEV='));
  assert.ok(!hasDev, `Must not pass DEV=... env arg, got: ${JSON.stringify(envArgs)}`);
});

// ---------------------------------------------------------------------------
// buildDataJson
// ---------------------------------------------------------------------------

test('buildDataJson: returns object with syncDirs array', () => {
  const obj = buildDataJson('/vault', 'http://localhost:24081', 'my-api-key');
  assert.ok(Array.isArray(obj.syncDirs), 'syncDirs must be an array');
  assert.equal(obj.syncDirs.length, 1);
});

test('buildDataJson: syncDirs[0] has correct path', () => {
  const obj = buildDataJson('/vault', 'http://localhost:24081', 'my-api-key');
  assert.equal(obj.syncDirs[0].path, '/vault');
});

test('buildDataJson: syncDirs[0] has apiUrl', () => {
  const obj = buildDataJson('/vault', 'http://localhost:24081', 'my-api-key');
  assert.ok(typeof obj.syncDirs[0].apiUrl === 'string' && obj.syncDirs[0].apiUrl.length > 0);
});

test('buildDataJson: syncDirs[0] has apiKey', () => {
  const obj = buildDataJson('/vault', 'http://localhost:24081', 'my-api-key');
  assert.equal(obj.syncDirs[0].apiKey, 'my-api-key');
});

test('buildDataJson: syncDirs[0] has twoWaySync=true', () => {
  const obj = buildDataJson('/vault', 'http://localhost:24081', 'my-api-key');
  assert.equal(obj.syncDirs[0].twoWaySync, true);
});

// ---------------------------------------------------------------------------
// buildHubNote
// ---------------------------------------------------------------------------

test('buildHubNote: contains mcp_federation_kb_url with the given url', () => {
  const note = buildHubNote('https://trip2g.com/_system/mcp');
  assert.ok(
    note.includes('mcp_federation_kb_url: https://trip2g.com/_system/mcp'),
    `Expected mcp_federation_kb_url in note, got:\n${note}`,
  );
});

test('buildHubNote: includes "free: true" (required for the federation scan to recognize the KB-note)', () => {
  const note = buildHubNote('https://trip2g.com/_system/mcp');
  assert.ok(note.includes('free: true'), `Expected "free: true" in hub note, got:\n${note}`);
});

test('buildHubNote: contains mcp_federation_kb_id derived from hostname', () => {
  const note = buildHubNote('https://trip2g.com/_system/mcp');
  assert.ok(
    note.includes('mcp_federation_kb_id: trip2g.com'),
    `Expected mcp_federation_kb_id: trip2g.com in note, got:\n${note}`,
  );
});

test('buildHubNote: uses correct hostname for custom hub url', () => {
  const note = buildHubNote('https://custom.example.org/_system/mcp');
  assert.ok(note.includes('mcp_federation_kb_id: custom.example.org'));
  assert.ok(note.includes('mcp_federation_kb_url: https://custom.example.org/_system/mcp'));
});

test('buildHubNote: starts with YAML frontmatter delimiter', () => {
  const note = buildHubNote('https://trip2g.com/_system/mcp');
  assert.ok(note.startsWith('---\n'), 'Note must start with ---');
});

// ---------------------------------------------------------------------------
// parseArgs — hub flags
// ---------------------------------------------------------------------------

test('parseArgs: --no-hub sets noHub=true', () => {
  const { flags } = parseArgs(['up', '--no-hub']);
  assert.equal(flags.noHub, true);
});

test('parseArgs: noHub defaults to false', () => {
  const { flags } = parseArgs(['up']);
  assert.equal(flags.noHub, false);
});

test('parseArgs: --hub-url sets hubUrl', () => {
  const { flags } = parseArgs(['up', '--hub-url', 'https://example.com/_system/mcp']);
  assert.equal(flags.hubUrl, 'https://example.com/_system/mcp');
});

test('parseArgs: hubUrl defaults to trip2g.com endpoint', () => {
  const { flags } = parseArgs(['up']);
  assert.equal(flags.hubUrl, 'https://trip2g.com/_system/mcp');
});

// ---------------------------------------------------------------------------
// _stampBlock
// ---------------------------------------------------------------------------

test('_stampBlock: first line is prefixed with HH:MM timestamp', () => {
  const now = new Date(2024, 0, 15, 9, 5); // 09:05 local
  const result = _stampBlock('hello', now);
  assert.match(result, /^\d\d:\d\d hello$/);
});

test('_stampBlock: timestamp matches /^\\d\\d:\\d\\d/', () => {
  const now = new Date();
  const result = _stampBlock('msg', now);
  assert.match(result, /^\d\d:\d\d /);
});

test('_stampBlock: only first line gets the timestamp', () => {
  const now = new Date(2024, 0, 15, 14, 30);
  const result = _stampBlock('line1\nline2\nline3', now);
  const lines = result.split('\n');
  assert.match(lines[0], /^\d\d:\d\d line1$/);
  assert.equal(lines[1], 'line2');
  assert.equal(lines[2], 'line3');
});

test('_stampBlock: literal \\\\n (backslash-n) is normalized to real newline', () => {
  const now = new Date(2024, 0, 15, 14, 30);
  const result = _stampBlock('first\\nsecond', now);
  const lines = result.split('\n');
  assert.equal(lines.length, 2);
  assert.match(lines[0], /^\d\d:\d\d first$/);
  assert.equal(lines[1], 'second');
});

// ---------------------------------------------------------------------------
// appendDaily
// ---------------------------------------------------------------------------

function makeTmpVault(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'memcli-test-'));
}

test('appendDaily: creates note with frontmatter, header, and entry (no time on first)', () => {
  const vault = makeTmpVault();
  try {
    const now = new Date(2024, 5, 10, 8, 0); // 2024-06-10 08:00 local
    appendDaily(vault, 'first entry', now);
    const localDay = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    const note = path.join(vault, 'daily', `${localDay}.md`);
    assert.ok(fs.existsSync(note), 'note file must exist');
    const content = fs.readFileSync(note, 'utf8');
    assert.ok(content.startsWith('---\ntimezone:'), 'must start with frontmatter');
    assert.ok(content.includes(`# ${localDay}`), 'must contain date header');
    // First entry of the day: no HH:MM prefix
    assert.ok(content.includes('first entry'), 'must contain entry text');
    assert.doesNotMatch(content, /\d\d:\d\d first entry/, 'first entry must NOT have a timestamp prefix');
  } finally {
    fs.rmSync(vault, { recursive: true });
  }
});

test('appendDaily: second same-day entry gets HH:MM prefix', () => {
  const vault = makeTmpVault();
  try {
    const now = new Date(2024, 5, 10, 8, 0);
    appendDaily(vault, 'first entry', now);
    const now2 = new Date(2024, 5, 10, 14, 30);
    appendDaily(vault, 'second entry', now2);
    const localDay = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    const note = path.join(vault, 'daily', `${localDay}.md`);
    const content = fs.readFileSync(note, 'utf8');
    assert.match(content, /\d\d:\d\d second entry/, 'second entry must have HH:MM prefix');
  } finally {
    fs.rmSync(vault, { recursive: true });
  }
});

test('appendDaily: appends a second entry without re-creating frontmatter', () => {
  const vault = makeTmpVault();
  try {
    const now = new Date(2024, 5, 10, 8, 0);
    appendDaily(vault, 'entry one', now);
    const now2 = new Date(2024, 5, 10, 9, 15);
    appendDaily(vault, 'entry two', now2);
    const localDay = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    const note = path.join(vault, 'daily', `${localDay}.md`);
    const content = fs.readFileSync(note, 'utf8');
    // Only one frontmatter block
    assert.equal((content.match(/^---$/gm) ?? []).length, 2, 'exactly one frontmatter block');
    assert.match(content, /entry one/);
    assert.match(content, /entry two/);
  } finally {
    fs.rmSync(vault, { recursive: true });
  }
});

test('appendDaily: maintains _index.md with date link', () => {
  const vault = makeTmpVault();
  try {
    const now = new Date(2024, 5, 10, 8, 0);
    appendDaily(vault, 'hello', now);
    const localDay = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    const idx = path.join(vault, 'daily', '_index.md');
    assert.ok(fs.existsSync(idx), '_index.md must exist');
    const content = fs.readFileSync(idx, 'utf8');
    assert.ok(content.includes(`[[${localDay}]]`), `_index.md must contain [[${localDay}]]`);
  } finally {
    fs.rmSync(vault, { recursive: true });
  }
});

test('ensureDailyIndex: does not duplicate link on second call', () => {
  const vault = makeTmpVault();
  try {
    fs.mkdirSync(path.join(vault, 'daily'), { recursive: true });
    ensureDailyIndex(vault, '2024-06-10');
    ensureDailyIndex(vault, '2024-06-10');
    const content = fs.readFileSync(path.join(vault, 'daily', '_index.md'), 'utf8');
    const count = (content.match(/\[\[2024-06-10\]\]/g) ?? []).length;
    assert.equal(count, 1, 'link must appear exactly once');
  } finally {
    fs.rmSync(vault, { recursive: true });
  }
});

// ---------------------------------------------------------------------------
// appendLog
// ---------------------------------------------------------------------------

test('appendLog: creates ### [[date]] section and entry (no time on first)', () => {
  const vault = makeTmpVault();
  try {
    const now = new Date(2024, 5, 10, 10, 30);
    appendLog(vault, 'work', 'did something', now);
    const localDay = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    const note = path.join(vault, 'work.md');
    assert.ok(fs.existsSync(note), 'log note must exist');
    const content = fs.readFileSync(note, 'utf8');
    assert.ok(content.includes(`### [[${localDay}]]`), 'must have date section header');
    // First entry under this day's section: no HH:MM prefix
    assert.ok(content.includes('did something'), 'must contain entry text');
    assert.doesNotMatch(content, /\d\d:\d\d did something/, 'first entry must NOT have a timestamp prefix');
  } finally {
    fs.rmSync(vault, { recursive: true });
  }
});

test('appendLog: second same-day entry gets HH:MM prefix', () => {
  const vault = makeTmpVault();
  try {
    const now = new Date(2024, 5, 10, 10, 30);
    appendLog(vault, 'work', 'first thought', now);
    const now2 = new Date(2024, 5, 10, 14, 30);
    appendLog(vault, 'work', 'second thought', now2);
    const localDay = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    const content = fs.readFileSync(path.join(vault, 'work.md'), 'utf8');
    assert.ok(content.includes('first thought'), 'must contain first entry text');
    assert.doesNotMatch(content, /\d\d:\d\d first thought/, 'first entry must NOT have a timestamp');
    assert.match(content, /\d\d:\d\d second thought/, 'second entry must have HH:MM prefix');
  } finally {
    fs.rmSync(vault, { recursive: true });
  }
});

test('appendLog: appends under existing ### [[date]] section', () => {
  const vault = makeTmpVault();
  try {
    const now = new Date(2024, 5, 10, 10, 30);
    appendLog(vault, 'work', 'first', now);
    const now2 = new Date(2024, 5, 10, 11, 0);
    appendLog(vault, 'work', 'second', now2);
    const localDay = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    const content = fs.readFileSync(path.join(vault, 'work.md'), 'utf8');
    // Only one date section
    const sectionCount = (content.match(new RegExp(`### \\[\\[${localDay}\\]\\]`, 'g')) ?? []).length;
    assert.equal(sectionCount, 1, 'date section must appear exactly once');
    assert.match(content, /first/);
    assert.match(content, /second/);
  } finally {
    fs.rmSync(vault, { recursive: true });
  }
});

test('appendLog: adds .md extension if missing from file arg', () => {
  const vault = makeTmpVault();
  try {
    const now = new Date(2024, 5, 10, 10, 30);
    appendLog(vault, 'notes', 'entry', now); // no .md
    assert.ok(fs.existsSync(path.join(vault, 'notes.md')));
  } finally {
    fs.rmSync(vault, { recursive: true });
  }
});

// ---------------------------------------------------------------------------
// parseArgs — daily / log subcommands
// ---------------------------------------------------------------------------

test('parseArgs: "daily" subcommand is recognized', () => {
  const { cmd } = parseArgs(['daily', 'some text']);
  assert.equal(cmd, 'daily');
});

test('parseArgs: "log" subcommand is recognized', () => {
  const { cmd } = parseArgs(['log', 'myfile', 'some text']);
  assert.equal(cmd, 'log');
});

test('parseArgs: daily positional arg is captured', () => {
  const { positional } = parseArgs(['daily', 'my note text']);
  assert.equal(positional[0], 'my note text');
});

test('parseArgs: log positional args are captured', () => {
  const { positional } = parseArgs(['log', 'journal', 'did work']);
  assert.equal(positional[0], 'journal');
  assert.equal(positional[1], 'did work');
});

test('parseArgs: --context flag sets context', () => {
  const { flags } = parseArgs(['daily', 'x', '--context', '5']);
  assert.equal(flags.context, 5);
});

test('parseArgs: context defaults to 15', () => {
  const { flags } = parseArgs(['daily', 'x']);
  assert.equal(flags.context, 15);
});

// ---------------------------------------------------------------------------
// shouldRunMcp — detection logic
// ---------------------------------------------------------------------------

test('shouldRunMcp: explicit known subcommand → false (never MCP)', () => {
  for (const cmd of ['up', 'down', 'status', 'logs', 'key', 'daily', 'log']) {
    assert.equal(shouldRunMcp([cmd], true), false, `${cmd} should not trigger MCP`);
    assert.equal(shouldRunMcp([cmd], false), false, `${cmd} piped should not trigger MCP`);
  }
});

test('shouldRunMcp: explicit "mcp" arg → true regardless of TTY', () => {
  assert.equal(shouldRunMcp(['mcp'], true), true);
  assert.equal(shouldRunMcp(['mcp'], false), true);
});

test('shouldRunMcp: piped stdin + no subcommand → true', () => {
  assert.equal(shouldRunMcp([], false), true);
});

test('shouldRunMcp: TTY + no subcommand → false', () => {
  assert.equal(shouldRunMcp([], true), false);
});

test('shouldRunMcp: piped stdin + flags only (no subcommand) → true', () => {
  assert.equal(shouldRunMcp(['--folder', '/tmp/v'], false), true);
});

// ---------------------------------------------------------------------------
// buildToolList — tool registry
// ---------------------------------------------------------------------------

test('buildToolList: returns exactly 8 tools', () => {
  const tools = buildToolList();
  assert.equal(tools.length, 8);
});

test('buildToolList: all expected tool names are present', () => {
  const names = new Set(buildToolList().map((t) => t.name));
  const expected = ['memory_up', 'memory_down', 'memory_status', 'memory_logs', 'memory_key', 'memory_daily', 'memory_log', 'memory_bind_hub'];
  for (const name of expected) {
    assert.ok(names.has(name), `Expected tool ${name} to be in list`);
  }
});

test('buildToolList: every tool has a non-empty description', () => {
  for (const tool of buildToolList()) {
    assert.ok(tool.description.length > 0, `Tool ${tool.name} must have a description`);
  }
});

test('buildToolList: every tool has an inputSchema with type=object', () => {
  for (const tool of buildToolList()) {
    assert.equal(
      (tool.inputSchema as { type?: string }).type,
      'object',
      `Tool ${tool.name} inputSchema.type must be "object"`,
    );
  }
});

test('buildToolList: memory_daily requires text', () => {
  const daily = buildToolList().find((t) => t.name === 'memory_daily');
  assert.ok(daily, 'memory_daily must exist');
  const required = (daily!.inputSchema as { required?: string[] }).required ?? [];
  assert.ok(required.includes('text'), 'memory_daily must require text');
});

test('buildToolList: memory_log requires file and text', () => {
  const log = buildToolList().find((t) => t.name === 'memory_log');
  assert.ok(log, 'memory_log must exist');
  const required = (log!.inputSchema as { required?: string[] }).required ?? [];
  assert.ok(required.includes('file'), 'memory_log must require file');
  assert.ok(required.includes('text'), 'memory_log must require text');
});

test('buildToolList: memory_bind_hub is present with required url', () => {
  const tool = buildToolList().find((t) => t.name === 'memory_bind_hub');
  assert.ok(tool, 'memory_bind_hub must be in the tool list');
  const required = (tool!.inputSchema as { required?: string[] }).required ?? [];
  assert.ok(required.includes('url'), 'memory_bind_hub must require url');
});

// ---------------------------------------------------------------------------
// buildHubNote — custom id
// ---------------------------------------------------------------------------

test('buildHubNote: custom id sets mcp_federation_kb_id to the provided id', () => {
  const note = buildHubNote('https://demo.lahab.cc/_system/mcp', 'my-team');
  assert.ok(
    note.includes('mcp_federation_kb_id: my-team'),
    `Expected mcp_federation_kb_id: my-team, got:\n${note}`,
  );
});

test('buildHubNote: custom id with free: true still present', () => {
  const note = buildHubNote('https://demo.lahab.cc/_system/mcp', 'my-team');
  assert.ok(note.includes('free: true'), 'free: true must be present even with custom id');
});

test('buildHubNote: without id defaults to hostname', () => {
  const note = buildHubNote('https://demo.lahab.cc/_system/mcp');
  assert.ok(note.includes('mcp_federation_kb_id: demo.lahab.cc'));
});

// ---------------------------------------------------------------------------
// hubSlug
// ---------------------------------------------------------------------------

test('hubSlug: returns hostname for standard url', () => {
  assert.equal(hubSlug('https://demo.lahab.cc/_system/mcp'), 'demo.lahab.cc');
});

test('hubSlug: returns hostname for trip2g.com', () => {
  assert.equal(hubSlug('https://trip2g.com/_system/mcp'), 'trip2g.com');
});

// ---------------------------------------------------------------------------
// parseArgs — hub subcommand
// ---------------------------------------------------------------------------

test('parseArgs: "hub" subcommand is recognized', () => {
  const { cmd } = parseArgs(['hub', 'https://demo.lahab.cc/_system/mcp']);
  assert.equal(cmd, 'hub');
});

test('parseArgs: hub url is captured as positional[0]', () => {
  const { positional } = parseArgs(['hub', 'https://demo.lahab.cc/_system/mcp']);
  assert.equal(positional[0], 'https://demo.lahab.cc/_system/mcp');
});

test('parseArgs: hub --id flag sets flags.id', () => {
  const { flags } = parseArgs(['hub', 'https://demo.lahab.cc/_system/mcp', '--id', 'my-team']);
  assert.equal(flags.id, 'my-team');
});

test('parseArgs: flags.id defaults to null', () => {
  const { flags } = parseArgs(['hub', 'https://demo.lahab.cc/_system/mcp']);
  assert.equal(flags.id, null);
});
