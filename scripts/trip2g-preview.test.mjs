// scripts/trip2g-preview.test.mjs
import { test } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { parseArgs, findConfig, resolveTarget, buildPayload } from './trip2g-preview.mjs';

test('parseArgs reads flags and values', () => {
  const f = parseArgs(['--watch', '--layout-file', 'a.html', '--note-path', '/n', '--open']);
  assert.equal(f.watch, true);
  assert.equal(f.open, true);
  assert.equal(f.layoutFile, 'a.html');
  assert.equal(f.notePath, '/n');
});

test('parseArgs rejects unknown flags', () => {
  assert.throws(() => parseArgs(['--nope']), /unknown flag/);
});

test('buildPayload: file-derived path + src + note path', () => {
  const p = buildPayload({ layoutPath: '/_layouts/a.html', layoutSrc: '<h1>x</h1>', notePath: '/n' });
  assert.deepEqual(p, { layout: { path: '/_layouts/a.html', src: '<h1>x</h1>' }, note: { path: '/n' } });
});

test('buildPayload: no note → note omitted', () => {
  const p = buildPayload({ layoutPath: '/a.html', layoutSrc: null, notePath: null, noteSrc: null });
  assert.deepEqual(p, { layout: { path: '/a.html' } });
});

test('buildPayload: missing layout path throws', () => {
  assert.throws(() => buildPayload({ layoutPath: null }), /layout path/);
});

test('resolveTarget precedence: flags > env > data.json', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'tp-'));
  const plug = path.join(dir, '.obsidian', 'plugins', 'trip2g');
  fs.mkdirSync(plug, { recursive: true });
  fs.writeFileSync(path.join(plug, 'data.json'),
    JSON.stringify({ syncDirs: [{ apiUrl: 'http://cfg:1/', apiKey: 'CFG' }] }));

  // data.json only
  let t = resolveTarget({ folder: dir }, {}, dir);
  assert.equal(t.apiUrl, 'http://cfg:1'); // trailing slash trimmed
  assert.equal(t.apiKey, 'CFG');

  // env overrides data.json
  t = resolveTarget({ folder: dir }, { TRIP2G_API_KEY: 'ENV' }, dir);
  assert.equal(t.apiKey, 'ENV');

  // flag overrides env
  t = resolveTarget({ folder: dir, apiKey: 'FLAG' }, { TRIP2G_API_KEY: 'ENV' }, dir);
  assert.equal(t.apiKey, 'FLAG');
});

test('findConfig walks up to data.json', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'tp-'));
  const sub = path.join(dir, 'a', 'b');
  fs.mkdirSync(sub, { recursive: true });
  const plug = path.join(dir, '.obsidian', 'plugins', 'trip2g');
  fs.mkdirSync(plug, { recursive: true });
  fs.writeFileSync(path.join(plug, 'data.json'),
    JSON.stringify({ syncDirs: [{ apiUrl: 'http://x', apiKey: 'K' }] }));
  assert.equal(findConfig(sub).apiKey, 'K');
});
