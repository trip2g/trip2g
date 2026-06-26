# Design: `trip2g-preview` (watch) + memcli instance isolation

**Status:** design / spec — not yet implemented.
**Date:** 2026-06-26.

**TL;DR.** Two small, independent improvements for the layout developer (верстальщик).
**(1)** A standalone node tool `trip2g-preview` — a rewrite of `scripts/renderlayout.py`
with a `--watch` mode: edit a Jet layout, it re-POSTs to `/_system/renderlayout` on
save, and a browser parked on `?live` reloads automatically. It targets **any** trip2g
server (local memcli by default via `data.json`, or a remote/staging server via flags).
**(2)** memcli gains `--name` so several local instances can run side by side instead of
fighting over the single hardcoded container name. The two features do not depend on each
other and ship as separate units.

This deliberately keeps concerns unmixed: previewing layouts is a layout-developer concern
and lives in its own tool; memcli stays "local trip2g server + agent memory".

---

## Non-goals (explicitly out of scope)

- The admin JSON-layout builder (`/_system/layouts/render`, `assets/ui/admin/layout/editor/`) — left as-is. JSON layouts stay; the owner may finish the editor later. This work neither uses nor removes it.
- GitHub components as themes — dropped for now; idea recorded separately in [github_themes.md](./github_themes.md).
- Agent/MCP affordances (`memory_preview` tool, JSON-layout composition preview) — deferred; nail the human loop first.
- CSS/JS sync from `_layouts/` and real-page auto-reload (Loop B server dev-mode) — separate future work.
- A machine-readable BEM component catalog — the real unlock for "an agent assembles sites from BEM components", but a larger separate sub-project. Noted as the next direction, not built here.

---

## Feature 1 — `trip2g-preview`

A standalone, dependency-free node ESM tool that supersedes `scripts/renderlayout.py`. Because memcli/this tooling is not used publicly yet, `renderlayout.py` is **removed** and its doc/skill references updated to point at `trip2g-preview`.

### Location & shape

- File: `scripts/trip2g-preview.mjs`, `#!/usr/bin/env node`, plain ESM, no build step, no third-party deps (uses built-in `fetch`, `node:fs`, `node:crypto` only).
- Run: `node scripts/trip2g-preview.mjs …` (or directly if executable).
- A sibling test file `scripts/trip2g-preview.test.mjs` run via `node --test`.

### CLI surface (arg names kept 1:1 with renderlayout.py)

```
trip2g-preview [--watch]
  (--layout-file <path> | --layout-path <serverpath>) [--layout-src <inline>]
  [--note-path <p> | --note-file <p> | --note-src <text>]
  [--api-url <url>] [--api-key <key>] [--folder <vault>] [--open] [--fetch]
```

New vs renderlayout.py: `--watch`, `--open`. Everything else is identical so existing muscle memory and docs carry over.

### Config resolution — `resolveTarget()`

Priority for `apiUrl` / `apiKey`:

1. `--api-url` / `--api-key` flags
2. env `TRIP2G_API_URL` / `TRIP2G_API_KEY`
3. `.obsidian/plugins/trip2g/data.json` (`syncDirs[0].apiUrl` / `.apiKey`), found by walking up from `--folder` (default cwd) — same logic as renderlayout.py's `find_config`.

`apiUrl` is the server base (e.g. `http://localhost:24081`); requests go to `${base}/_system/renderlayout`. No API key resolvable → exit 1 with: `no API key — run \`memcli up\` or pass --api-key`.

### Payload builder — `buildPayload()`

Pure function producing the [`RenderLayoutInput`](../en/user/renderlayout.md) body:

```json
{ "layout": { "path": "<layout-path>", "src": "<optional inline/file contents>" },
  "note":   { "path": "<note-path>" } | { "src": "<note markdown>" } }
```

Rules (mirroring renderlayout.py): `--layout-file` implies `layout.path = "/" + file` (unless `--layout-path` given) and `layout.src = <file contents>`; note is `path` xor `src` xor a file; if no note flag, omit `note` (server uses a stub).

### Behavior

**One-shot (no `--watch`):**
1. `resolveTarget()` → `buildPayload()` → POST `${base}/_system/renderlayout` with `X-API-Key`.
2. Print `warnings.layout` entries to stderr; print the full `${base}${previewURL}` to stdout.
3. `--fetch` → GET the preview and print HTML to stdout instead of the URL.
4. Exit 1 if `warnings.layout` non-empty or the response carries `error` (parity with renderlayout.py).

**Watch (`--watch`):**
1. Require `--layout-file` (need a local file to watch); else exit 1 with a clear message.
2. Initial render (POST). Print the live URL prominently: `${base}/_system/renderlayout?live`. `--open` → `xdg-open`/`open`/`start` it.
3. `fs.watch(layoutFile)` (watch the parent dir too, since many editors replace-on-save); debounce ~200 ms; on change re-read the file, re-POST.
4. Each render prints one status line: `HH:MM:SS  ok` or `HH:MM:SS  N warning(s)` plus the warning text. The server's `?live` long-poll reloads the browser on each new version — no extra wiring.
5. `SIGINT` (Ctrl-C) → clean exit.

### Error handling

| Case | Behavior |
|------|----------|
| No API key | exit 1 + hint |
| `--watch` without `--layout-file` | exit 1 + hint |
| POST non-2xx / body `{error}` | print server error to stderr, exit 1 (one-shot); in watch, log and keep watching |
| Jet warnings | stderr; one-shot exits 1, watch keeps going |

### Doc / reference updates

- Rewrite the "CLI tool" and "Live preview with file watcher" sections of `docs/en/user/renderlayout.md` (+ `docs/ru/user/renderlayout.md`) to use `trip2g-preview --watch` instead of `renderlayout.py` + `watchexec`.
- Update `docs/skills/check_templates.md` reference.
- Update `docs/dev/local_design_iteration.md` Loop A to use `trip2g-preview`.
- Remove `scripts/renderlayout.py`.

---

## Feature 2 — memcli instance isolation (`--name`)

Today `cli/memcli/src/cli.ts` uses `const CONTAINER_NAME = 'trip2g-memory'` everywhere, so a second `up` collides. Make the container name resolvable while keeping the default byte-for-byte identical.

### Changes

- `Flags` gains `name?: string`; `parseArgs` parses `--name <id>`.
- New `containerName(flags)`:
  - `flags.name` → `trip2g-memory-${sanitize(flags.name)}` (sanitize to docker-safe `[a-zA-Z0-9_.-]`)
  - else → `trip2g-memory` (unchanged)
- Replace the `CONTAINER_NAME` const at every use site — `cmdUp` (docker run `--name`, the "already running" `ps --filter`, start/skip log), `cmdDown` (stop/rm), `cmdStatus` (`ps --filter`), `cmdLogs` (`docker logs`), `cmdKey` if it touches the container.
- Port: `--port` stays (default 24081). Before `docker run`, `cmdUp` checks whether the port is already bound; if so, exit with: `port <p> busy — pass --port for this instance`. (No auto-pick: keep the printed URL predictable.)
- Per-vault state (`<vault>/.trip2g-memory/`, `data.json`) is already isolated by `--folder`; no change.
- Help text + `docs/en/user/memcli.md` (+ ru) document: a second instance needs both `--name` and `--port`, and `down`/`status`/`logs`/`key` take the same `--name`.

---

## Testing

**`trip2g-preview`** (`node --test`, pure functions, no network):
- `resolveTarget()` precedence: flags > env > data.json; missing-key error path.
- `buildPayload()`: every layout×note input combination yields the documented body.
- arg parsing recognizes `--watch`, `--open`, `--fetch`, `--layout-*`, `--note-*`.

**memcli** (extend `cli/memcli/src/cli.test.ts`):
- `containerName(flags)`: default vs `--name foo` → `trip2g-memory-foo`; sanitization.
- `parseArgs` recognizes `--name`.

The thin fs.watch / SIGINT / browser-open glue is verified manually (the same way the Loop A/B demos were verified live), not unit-tested.

---

## Future direction (context, not scope)

The aspiration is an agent composing sites from BEM components. The render/preview loop
(this spec) and the JSON-layout composition format already exist; the missing piece is a
**discoverable component catalog** — blocks, their args, and a per-block preview — so an
agent (or a человек) can browse and assemble. That is the next sub-project; this work is
its human-facing foundation.
