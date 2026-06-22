# trip2g-sync watch mode + live-follow URL flag + docs

**Date:** 2026-06-22
**Status:** Design — pending review

## TL;DR

Turn `trip2g-sync.mjs` into an optional long-running daemon (`--watch`) that keeps a
local vault folder and a trip2g instance in continuous two-way sync **without Obsidian**:
remote→local via the existing `noteChanges` SSE subscription, local→remote via a filesystem
watcher with auto-push. Pair it with a URL flag that activates the already-built (but
undocumented) frontend "live-follow" mode, so an agent can hand the user a link and the
browser auto-navigates to each note as the agent edits it — watch the agent walk the vault
like a movie. Document both, plus the headless container flow, in `agent-memory.md`.

## Motivation

`agent-memory.md` is evolving toward a flow where an agent can run a trip2g server
container, pull the onboarding-vault (which already bundles `trip2g-sync.mjs`), and run a
sidecar sync container — getting continuous sync with **no Obsidian app involved**. Today
the CLI does only one-shot sync; its README suggests piping external `fswatch`. We want
first-class watch built in. Separately, the frontend can already auto-navigate to changed
pages, but there is no link-based way to turn it on and no docs for it.

## Scope

Three bounded pieces in one effort:

- **A. CLI watch mode** — `obsidian-sync` (TypeScript → bundled `trip2g-sync.mjs`).
- **B. Frontend live-follow URL flag** — `assets/ui` ($mol).
- **C. Docs** — `docs/en/user` + `docs/ru/user`.

Out of scope: changing one-shot sync semantics; backend changes (the `noteChanges`
subscription and SSE transport already exist and are reused as-is).

---

## A. CLI watch mode

### Existing pieces reused

- `LivePullConnection.ts` — pure-`fetch` + `response.body.getReader()` SSE client for the
  `noteChanges` subscription. Self-reconnecting (exponential backoff), 60s health-check.
  **Zero dependencies.** Used today by the Obsidian plugin; not yet wired into the CLI.
- `classifySync` / `filterPlan` / `executePlan` (`src/sync/*`) — the one-shot sync engine.
- `readDataJson()` (`src/sync/cli/cmd.ts`) — reads `.obsidian/plugins/trip2g/data.json`.

### Behavior

`trip2g-sync --watch` (alias `-w`) starts a long-running foreground daemon:

```
trip2g-sync --watch --folder <vault> [--api-key <k>] [--api-url <gql>]
                    [--include <glob>] [--exclude <glob>] [--conflict-resolution <mode>]

  1. Startup reconcile — run one-shot sync once (two-way) so both sides align.
  2. SSE follower (remote→local) — LivePullConnection over the CLI's GraphQL
     endpoint; on each noteChanges batch, pull + write the changed paths.
  3. FS watcher (local→remote) — chokidar if importable, else fs.watch; on a
     debounced batch (~500 ms) of events, classify the affected paths and auto-push.
  4. SIGINT/SIGTERM — disconnect SSE, finish/flush any in-flight push, exit 0.
```

### Decisions

- **`--watch` implies two-way.** The SSE follower writes files locally (a pull), so watch is
  inherently bidirectional. The existing "prefix not allowed with `--two-way`" guard carries over.
- **Auto-push on local change**, governed by `--conflict-resolution` (default `local` =
  keep local, push). Conflicts (changed both sides) are pushed per the chosen mode; we do
  not silently overwrite beyond what one-shot sync already does.
- **FS watcher dependency strategy:** `await import('chokidar')`; if it resolves, use it
  (robust editor-atomic-save / rename handling). If it throws, fall back to
  `fs.watch({ recursive: true })` (Node ≥20 on Linux; works on macOS/Windows) and print a
  one-line warning that watching is best-effort without chokidar.
- **Safety net everywhere:** hash-based `classifySync` reconciles. A missed, duplicated, or
  self-triggered event (our own push echoing back over SSE) classifies as `unchanged` →
  no-op. No need to track "files I just pushed."

### Config: `data.json` is the source of truth

`data.json` already carries `apiUrl`, `apiKey`, `twoWaySync`, and (in the `SyncDir` type)
`livePullIncludePatterns` / `livePullExcludePatterns`. The CLI's `readDataJson()` currently
surfaces only `apiUrl`/`apiKey`.

- **Extend `readDataJson()`** to also surface `livePullIncludePatterns` /
  `livePullExcludePatterns`.
- **SSE filter precedence:** `--include`/`--exclude` flags → `data.json` livePull patterns →
  default `["**"]`.
- **Default when `--watch` is explicit but no patterns anywhere:** subscribe to `["**"]`
  (follow everything). This deliberately diverges from the plugin's "empty patterns =
  live-pull off" — for a CLI daemon, an explicit `--watch` means "yes, follow." Documented
  as such.
- `data.json` is authorable two ways: by **Obsidian** (plugin settings UI writes it), or by
  the **agent** writing the JSON file directly. Either way the daemon auto-discovers config.

### URL reconciliation note

`LivePullConnection` appends `/_system/graphql` to a **base** site URL, whereas the CLI's
`--api-url` is already the full GraphQL endpoint. Watch mode must POST the subscription to
the CLI's endpoint directly (not re-append the path). Adapt `LivePullConnection` to accept a
full endpoint, or wrap it with the correct URL.

### Container / daemon requirements

- Long-running foreground process suitable for `docker run` (no detach; non-zero exit on
  fatal error).
- Line-oriented logs to stdout so `docker logs` is readable (already the style).
- The watch feature ships inside the onboarding-vault zip automatically — `esbuild.cli.mjs`
  already propagates the bundle to `onboarding-vault/.obsidian/plugins/trip2g/trip2g-sync.mjs`.

### Build

After editing TS sources, rebuild via `npm run build:cli` (esbuild → `dist/trip2g-sync.mjs`,
auto-propagated to the vault plugin dirs). Tests: extend the `vitest` suite under
`src/sync/` for the new watch orchestration where unit-testable (debounce batching, filter
precedence, signal handling are the candidates; SSE/FS are integration-level).

---

## B. Frontend live-follow URL flag

### Existing pieces reused

The live-follow feature is **already built**:

- `assets/ui/user/live/live.view.ts` — subscribes to `noteChanges` (`includePatterns:
  ['**/*.md']`). When `follow_enabled()` is on, on a change it sets
  `location.href = <changed note permalink>` → the browser navigates to the just-edited
  note. When `reload_enabled()` is on, it reloads the current page and highlights the changed
  selectors.
- `$trip2g_user_live_follow_toggler` — a UI checkbox storing the flag in
  `$mol_state_local` under key `trip2g_live_follow`.

### What we add

A **URL-arg entry point** so the flag can be enabled by link (no manual checkbox):

- Read a dedicated `$mol_state_arg` flag and OR it into `follow_enabled()` in
  `live.view.ts`. Use a dedicated arg name (e.g. `live_follow`) — **not** `search`, which is
  already the search arg and would collide.
- Confirm the exact hashbang idiom against existing `$mol_state_arg` usage in `assets/ui`
  before finalizing the arg name and link form (the user's `?#!search=...` sketch was
  illustrative; the real link will be `…/?#!live_follow=on` or equivalent once verified).
- When the URL flag turns follow on, the existing subscription + navigation logic does the
  rest. Optionally persist it into `$mol_state_local` so it survives the auto-navigation
  (otherwise each navigation drops the arg). **Open implementation detail:** the flag must
  survive `location.href` navigation — either re-append the arg on navigate or write it to
  `$mol_state_local` on activation. Decide during implementation; `$mol_state_local`
  persistence is the simpler path.

### Result

An agent gives the user one link. The user opens it, and the browser rides along behind the
agent, hopping to each note as it's edited — the "watch the agent walk the vault" experience.

---

## C. Docs

`agent-memory.md` documents only one-shot sync. `live-editing.md` documents live DOM-patch
updates but **not** live-follow or the reload toggle. The CLI watch mode is brand new.

Add/extend (bilingual — EN + RU pairs per project convention):

1. **`agent-memory.md`** — a new section for the headless, Obsidian-free flow: run the
   server container, pull the onboarding-vault, run `trip2g-sync --watch` as a sidecar, and
   (optionally) open the live-follow link to watch the agent edit notes in real time.
   Include the "agent may also install Obsidian" alternative for `data.json` authoring.
2. **`live-editing.md`** (or a sibling) — document live-follow (auto-navigate / cinema mode)
   and the reload toggle, including the URL flag to enable follow by link.
3. **Cross-links** — from `agent-memory.md` and `live-editing.md` to each other and to the
   watch-mode CLI reference (`local-quickstart.md`).
4. **CLI reference** — add `--watch` and the new `--include`/`--exclude` flags to the
   `trip2g-sync` help text and to `local-quickstart.md`.

Follow the project doc style: answer-first TL;DR lead, concrete examples + antipatterns,
scannable structure.

---

## Risks & open items

- **`fs.watch` reliability** without chokidar (missed events on some platforms / editor
  atomic saves). Mitigated by hash-classify reconciliation; documented as best-effort.
- **Live-follow flag surviving navigation** (B) — needs the `$mol_state_local` persistence
  vs. re-append decision at implementation time.
- **`$mol_state_arg` idiom** (B) — exact arg name/link form to be verified against existing
  `assets/ui` usage.
- **Self-echo loops** — covered by hash-classify, but worth an explicit test: push a file,
  receive own SSE event, assert no re-push.
```
