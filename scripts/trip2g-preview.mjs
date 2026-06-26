#!/usr/bin/env node
// trip2g-preview — render a Jet layout against a note via /_system/renderlayout,
// with optional --watch live reload. Standalone, zero dependencies.
// Supersedes scripts/renderlayout.py.

import fs from 'node:fs';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { pathToFileURL } from 'node:url';

export function parseArgs(argv) {
  const f = {
    watch: false, open: false, fetch: false,
    layoutFile: null, layoutPath: null, layoutSrc: null,
    notePath: null, noteFile: null, noteSrc: null,
    apiUrl: null, apiKey: null, folder: null,
  };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    switch (a) {
      case '--watch': f.watch = true; break;
      case '--open': f.open = true; break;
      case '--fetch': f.fetch = true; break;
      case '--layout-file': f.layoutFile = argv[++i]; break;
      case '--layout-path': f.layoutPath = argv[++i]; break;
      case '--layout-src': f.layoutSrc = argv[++i]; break;
      case '--note-path': f.notePath = argv[++i]; break;
      case '--note-file': f.noteFile = argv[++i]; break;
      case '--note-src': f.noteSrc = argv[++i]; break;
      case '--api-url': f.apiUrl = argv[++i]; break;
      case '--api-key': f.apiKey = argv[++i]; break;
      case '--folder': f.folder = argv[++i]; break;
      default:
        if (a.startsWith('-')) throw new Error(`unknown flag: ${a}`);
    }
  }
  return f;
}

export function findConfig(startDir) {
  let dir = path.resolve(startDir || '.');
  for (;;) {
    const candidate = path.join(dir, '.obsidian', 'plugins', 'trip2g', 'data.json');
    if (fs.existsSync(candidate)) {
      try {
        const data = JSON.parse(fs.readFileSync(candidate, 'utf8'));
        const sd = (data.syncDirs || [])[0];
        if (sd) return { apiUrl: sd.apiUrl || null, apiKey: sd.apiKey || null };
      } catch { /* ignore malformed */ }
    }
    const parent = path.dirname(dir);
    if (parent === dir) return {};
    dir = parent;
  }
}

export function resolveTarget(flags, env, startDir) {
  const cfg = findConfig(flags.folder || startDir || '.');
  const apiUrl = (flags.apiUrl || env.TRIP2G_API_URL || cfg.apiUrl || 'http://localhost:8081')
    .replace(/\/+$/, '');
  const apiKey = flags.apiKey || env.TRIP2G_API_KEY || cfg.apiKey || '';
  return { apiUrl, apiKey };
}

export function buildPayload({ layoutPath, layoutSrc, notePath, noteSrc }) {
  if (!layoutPath) throw new Error('layout path required (--layout-path or --layout-file)');
  const layout = { path: layoutPath };
  if (layoutSrc != null) layout.src = layoutSrc;
  let note;
  if (notePath) note = { path: notePath };
  else if (noteSrc != null) note = { src: noteSrc };
  const payload = { layout };
  if (note) payload.note = note;
  return payload;
}

// Read files referenced by flags and derive the layout path.
export function resolveInputs(flags) {
  let layoutPath = flags.layoutPath;
  let layoutSrc = flags.layoutSrc;
  if (flags.layoutFile) {
    layoutPath = layoutPath || '/' + flags.layoutFile.replace(/^\/+/, '');
    layoutSrc = fs.readFileSync(flags.layoutFile, 'utf8');
  }
  let noteSrc = flags.noteSrc;
  if (flags.noteFile) noteSrc = fs.readFileSync(flags.noteFile, 'utf8');
  return { layoutPath, layoutSrc, notePath: flags.notePath, noteSrc };
}

export async function postRender(apiUrl, apiKey, payload) {
  const res = await fetch(`${apiUrl}/_system/renderlayout`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', 'x-api-key': apiKey },
    body: JSON.stringify(payload),
  });
  let body = {};
  try { body = await res.json(); } catch { /* non-JSON */ }
  return { status: res.status, body };
}

function printWarnings(body) {
  const warns = (body.warnings && body.warnings.layout) || [];
  if (warns.length) {
    process.stderr.write('WARNINGS:\n');
    for (const w of warns) process.stderr.write(`  ${w}\n`);
  }
  return warns.length;
}

async function renderOnce(apiUrl, apiKey, flags) {
  const payload = buildPayload(resolveInputs(flags));
  const { status, body } = await postRender(apiUrl, apiKey, payload);
  if (body.error) { process.stderr.write(`ERROR: ${body.error}\n`); return { ok: false, body }; }
  if (status >= 400) { process.stderr.write(`ERROR: HTTP ${status}\n`); return { ok: false, body }; }
  const nWarn = printWarnings(body);
  const url = `${apiUrl}${body.previewURL || ''}`;
  return { ok: nWarn === 0, url, body };
}

function openInBrowser(url) {
  const cmd = process.platform === 'darwin' ? 'open'
    : process.platform === 'win32' ? 'start' : 'xdg-open';
  try { spawn(cmd, [url], { stdio: 'ignore', detached: true }).unref(); } catch { /* ignore */ }
}

async function runWatch(apiUrl, apiKey, flags) {
  if (!flags.layoutFile) {
    process.stderr.write('Error: --watch needs --layout-file.\n');
    process.exit(1);
  }
  const liveUrl = `${apiUrl}/_system/renderlayout?live`;
  const stamp = () => new Date().toTimeString().slice(0, 8);
  const render = async () => {
    try {
      const r = await renderOnce(apiUrl, apiKey, flags);
      process.stdout.write(`${stamp()}  ${r.ok ? 'ok' : 'warnings'}\n`);
    } catch (e) { process.stdout.write(`${stamp()}  error: ${e.message}\n`); }
  };
  await render();
  process.stdout.write(`live: ${liveUrl}\n`);
  if (flags.open) openInBrowser(liveUrl);

  let timer = null;
  const onChange = () => { if (timer) clearTimeout(timer); timer = setTimeout(render, 200); };
  // Watch the file; also watch its directory (editors replace-on-save).
  fs.watch(flags.layoutFile, onChange);
  fs.watch(path.dirname(path.resolve(flags.layoutFile)), (_e, fn) => {
    if (fn && path.basename(flags.layoutFile) === fn) onChange();
  });
  process.on('SIGINT', () => { process.stdout.write('\nstopped.\n'); process.exit(0); });
}

async function main() {
  const flags = parseArgs(process.argv.slice(2));
  const { apiUrl, apiKey } = resolveTarget(flags, process.env, '.');
  if (!apiKey) {
    process.stderr.write('Error: no API key — run `memcli up` or pass --api-key.\n');
    process.exit(1);
  }
  if (flags.watch) { await runWatch(apiUrl, apiKey, flags); return; }
  // one-shot
  const r = await renderOnce(apiUrl, apiKey, flags);
  if (flags.fetch && r.url) {
    const html = await (await fetch(r.url)).text();
    process.stdout.write(html + '\n');
  } else if (r.url) {
    process.stdout.write(r.url + '\n');
  }
  process.exit(r.ok ? 0 : 1);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((e) => { process.stderr.write(`Error: ${e.message}\n`); process.exit(1); });
}
