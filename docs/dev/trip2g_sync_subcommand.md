# `trip2g sync` subcommand — REJECTED

**Status: REJECTED (2026-06-29).** A built-in Go `trip2g sync [--watch] <dir>` was considered and rejected: two existing mechanisms already do its job, and both do it better. Use the existing **`obsidian-sync` CLI** (JS, grown from the Obsidian plugin) for FS→instance sync, and **`git push`** for bulk/versioned/CI sync.

## What was proposed

A built-in `trip2g sync --watch <dir> --url <instance> --api-key <key>` that watches a filesystem notes/docs directory (fsnotify) and pushes changed markdown to a trip2g instance via the `pushNotes` GraphQL mutation — a Go replacement for the JS watch-sync that grew out of the Obsidian plugin.

## Why rejected

**1. We already have a sync CLI — `obsidian-sync`.** A JS sync CLI grown from the Obsidian plugin already does filesystem→instance sync today (`cd docs && obsidian-sync`; it auto-discovers the API key from `docs/.obsidian/plugins`). It handles the hard parts a Go port would have to re-implement:
- binary **asset upload** (presigned-S3 `uploadNoteAsset`),
- **deletes** (`hideNotes`),
- conflict handling, initial full sync, `.obsidian/` ignores.

`pushNotes` carries text only (`{path, content}`); markdown-only Go parity is quick, but full parity duplicates a working, mature CLI for low ROI.

**2. git is already a bidirectional sync transport.** trip2g's git integration goes both ways:
- **DB → git** (mirror): `internal/gitapi/materialize.go` rebuilds refs from the DB.
- **git → DB** (ingest): `internal/gitapi/apply.go` `ApplyGitChanges` → `PushNotes`; driven by the `applygitchanges` cron; a **receive-pack** (`git push`) path exists.

`docs/dev/graphql.md` lists the sync sources as *"API, **git push**, editor"*; `docs/dev/refactor.md`: *"Push в git → webhook → обновление БД."* So **`git push` to the trip2g remote IS sync** — and it brings versioning, offline, branching, and merge for free. A fsnotify watcher offers none of that.

**3. A Go watcher would be worse than both** — inferior to the JS plugin for live editing, inferior to git for versioned/bulk/CI sync. It occupies the weakest middle.

## What to use instead

| Need | Use |
|------|-----|
| FS → instance sync (incl. assets, deletes, live editing) | the existing **`obsidian-sync` CLI** (JS, from the Obsidian plugin) |
| Headless / CI / bulk / versioned sync | `git push` to the trip2g git remote (ingested via `ApplyGitChanges`) |
| A one-liner convenience | just `git add && git commit && git push` — not worth a subcommand |

## Notes / related

- Known perf caveat of the git/push path: each push currently triggers a full note reload under a global write mutex (see `docs/dev/obsidian_sync_2026-06-21.md`). That's a perf item to improve on the existing path — **not** a reason to build a parallel Go sync.
- Accepted companion feature: **`trip2g lint <dir>`** (DB-free doc linter) — see `docs/dev/trip2g_lint.md`. Both were analysed together as a "trip2g local FS CLI"; `lint` is built, `sync` is not.
- Code: `internal/gitapi/{materialize,apply}.go`, `internal/case/cronjob/applygitchanges/`, `docs/dev/obsidian_sync.md`.
