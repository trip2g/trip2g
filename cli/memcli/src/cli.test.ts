/**
 * Tests for trip2g-memory pure helpers.
 * Run with: node --experimental-strip-types --test src/cli.test.ts
 * Zero npm dependencies — uses node:test + node:assert only.
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import {
  signHatJwt,
  parseArgs,
  buildServerEnv,
  buildDataJson,
  buildDockerRunArgs,
  buildHubNote,
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

test('buildServerEnv: STORAGE_BACKEND is "local"', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x' });
  assert.equal(env.STORAGE_BACKEND, 'local');
});

test('buildServerEnv: STORAGE_LOCAL_DIR is set', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x' });
  assert.ok(env.STORAGE_LOCAL_DIR, 'STORAGE_LOCAL_DIR must be non-empty');
});

test('buildServerEnv: contains LISTEN_ADDR with correct port', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x' });
  assert.ok(env.LISTEN_ADDR.includes('24081'), `LISTEN_ADDR should include port 24081, got: ${env.LISTEN_ADDR}`);
});

test('buildServerEnv: contains INTERNAL_LISTEN_ADDR with iport', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x' });
  assert.ok(env.INTERNAL_LISTEN_ADDR.includes('24082'), `INTERNAL_LISTEN_ADDR should include 24082, got: ${env.INTERNAL_LISTEN_ADDR}`);
});

test('buildServerEnv: contains JWT_SECRET', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'hunter2' });
  assert.equal(env.JWT_SECRET, 'hunter2');
});

test('buildServerEnv: contains OWNER_EMAIL', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'owner@test.com', secret: 'x' });
  assert.equal(env.OWNER_EMAIL, 'owner@test.com');
});

test('buildServerEnv: contains DB_FILE', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x' });
  assert.ok(env.DB_FILE, 'DB_FILE must be non-empty');
});

test('buildServerEnv: contains PUBLIC_URL with correct port', () => {
  const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x' });
  assert.ok(env.PUBLIC_URL.includes('24081'), `PUBLIC_URL should include port 24081, got: ${env.PUBLIC_URL}`);
});

for (const key of FORBIDDEN_KEYS) {
  test(`buildServerEnv: does NOT contain ${key}`, () => {
    const env = buildServerEnv({ port: 24081, iport: 24082, email: 'a@b.com', secret: 'x' }) as unknown as Record<string, unknown>;
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
