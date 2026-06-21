# Git ↔ Obsidian-plugin coexistence — design

**Date:** 2026-06-20
**Area:** `internal/gitapi`, `internal/case/pushnotes`, `cmd/server`
**Status:** approved, ready for implementation plan

## Problem

The same notes (`note_paths` / `note_versions`) are written by two independent paths
that don't know about each other:

1. **Obsidian plugin / CLI** → GraphQL `pushNotes` → `InsertNote`. Never touches the git repo.
2. **Git push** → `git-receive-pack` updates the bare repo → `ApplyChanges` reads **all**
   files from `HEAD`, hashes each, skips those already matching the DB, pushes the rest
   through the same `pushNotes` use case.

Nothing ever writes DB changes back into the git repo. Consequences:

- A plugin edit never reaches git. Git clients `pull` stale content.
- That client edits and `git push`. `ApplyChanges` sees the stale git content's hash ≠ the
  DB's new hash, so it pushes the **stale git version as a new version — silently
  overwriting the plugin's edit.** Last-writer-wins, no merge, no warning.

### Other defects found in `internal/gitapi/api.go`

1. **Git deletions are ignored.** `getAllFiles` only lists files still present in `HEAD`; a
   file deleted in a commit is never hidden in the DB. The plugin has `hideNotes`; the git
   path has no equivalent.
2. **`getChangedFiles` was never implemented** (`ApplyChanges` has the diff logic commented
   out with `// TODO: fix logic for first push`), so every push re-scans **all** files. This
   is also *why* deletions can't be detected — there is no old→new diff.
3. **`uploadRepo` runs synchronously** in the push handler (`// todo: run in background`);
   every push blocks on a full-repo tar+gzip+upload.
4. **Concurrency holes:** the global `api.mu` is held for the *entire* request, including
   slow `git-upload-pack` clone streaming (clones block pushes and each other); the daily
   cronjob calls `ApplyChanges` **without** taking `api.mu`; and the plugin's `pushNotes`
   takes no git lock at all — so plugin-push and git-apply can interleave on the DB.
5. **Inconsistent failure recovery:** `receive-pack` updates refs *before* `ApplyChanges` /
   `uploadRepo` run. If those fail, the client gets a 500 but the repo already advanced, and
   the object-storage snapshot is stale → on restart `downloadRepo` restores divergent state.
6. **Assets diverge too** — plugin-uploaded assets are never written into git.
7. **Test coverage is effectively zero** — only `filterDotFiles` is tested.

## Decisions

- **Source of truth:** DB is canonical. The bare repo is a server-maintained mirror.
- **Materialization:** lazy — rebuild the git ref from the DB only when a git client
  connects, and only when the tree actually changed (idempotent, no empty commits).
- **Conflict semantics:** standard git non-fast-forward rejection. Because materialize keeps
  `HEAD` mirroring the DB, a stale push is rejected natively; the user runs `git pull`
  (gets plugin changes from the mirror), merges, pushes again. No silent data loss, almost
  no custom conflict code.
- **No `EnableGit` flag.** Git is already gated by the git binaries being present
  (`gitAPI != nil`) and by token auth — lazy materialization costs nothing until a valid
  client connects, so a runtime flag would be redundant.
- **Storage stays the whole-repo `tar.gz` snapshot.** Incremental storage is out of scope
  (target is ≤1k notes; the snapshot is cheap at that scale).
- **In scope:** materializer, diff-based apply with deletions, shared locking, fast-forward
  enforcement, robust push transaction, asset mirroring, tests.

## Architecture

```
            pushNotes (plugin / CLI / TG import / automation)
                       │  writes
                       ▼
                 ┌───────────┐        materialize (lazy, on git access)
                 │    DB     │ ───────────────────────────────► bare repo (mirror)
                 │ (canon.)  │ ◄───────────────────────────────  HEAD == DB tree
                 └───────────┘        apply (diff old→new HEAD)
                       ▲                         │
                       └─────────────────────────┘
                              git push
```

### Data flow on a git request

```
HandleRequest (git enabled, auth ok):
  ── lock ──────────────────────────────────────────────
  materialize()                # DB → git, idempotent
  oldHEAD = rev-parse master
  ── unlock ────────────────────────────────────────────
  stream upload-pack / receive-pack   # NOT under lock (slow byte transfer)
  ── lock ──────────────────────────────────────────────   (push only)
  newHEAD = rev-parse master
  applyDiff(oldHEAD, newHEAD)  # git → DB, with deletions
    └─ on error: update-ref master oldHEAD   # rollback, repo never diverges
  ── unlock ────────────────────────────────────────────
  go uploadRepo()              # snapshot off the critical path
```

## Components

### 1. Materializer — `materialize(ctx)`
Plumbing on the bare repo with a temporary `GIT_INDEX_FILE` (no working tree):

```
for each latest note (path → raw content) and asset (path → bytes) from DB:
    blob = git hash-object -w --stdin            # same content → same sha (idempotent)
    git update-index --add --cacheinfo 100644,<blob>,<path>
tree = git write-tree
if tree == HEAD^{tree}:  return                   # nothing changed → no commit
commit = git commit-tree tree -p HEAD -m "server sync"
git update-ref refs/heads/<master> commit
```

- Rebuilds the full index from the DB each call (simple; fine at ≤1k notes).
- Idempotent at the tree level — steady state writes nothing.
- Note content is the **raw markdown source** from `note_versions`, not rendered HTML.

### 2. Diff-based apply — `applyDiff(oldHEAD, newHEAD)`
Replaces the "always list all files" path and `getChangedFiles` TODO.

```
git diff --name-status oldHEAD newHEAD
  A/M *.md|*.html  → PushNotes (hash-skip if it already matches DB — breaks the loop)
  D    *           → HideNotes (new Env method)
```

First push (no `oldHEAD`): diff against the empty tree so every file counts as added.

### 3. Locking
A single mutex guards **only** the DB/ref-mutating critical sections (`materialize`,
`applyDiff`), never the byte-streaming of clone/fetch. The plugin `pushNotes` path acquires
the **same** mutex (so plugin-push and git-apply cannot interleave); the daily cron goes
through it too. Restructure `HandleRequest` so the lock is taken twice (around materialize,
around apply) rather than held for the whole request.

### 4. Robust push transaction
- Capture `oldHEAD` before `receive-pack`. If `applyDiff` fails afterward, roll
  `refs/heads/<master>` back to `oldHEAD` so the repo never diverges from the DB; the client
  sees a clean rejection.
- `uploadRepo` moves into a background goroutine; a lagging snapshot is harmless (only used
  for restore-on-restart, and it reflects committed state).

### 5. Asset mirroring
`materialize` also writes asset blobs into git at their repo paths, fetched by hash from
object storage. Idempotent by hash (skip when the git blob already matches), so steady state
is cheap. Without this, git clients pulling get notes but not plugin-uploaded images.

### 6. Fast-forward enforcement
At repo init, set `receive.denyNonFastForwards = true` (in addition to the existing
branch-restriction `pre-receive` hook). This is what makes a stale push fail natively.

### New `Env` methods (`internal/gitapi`)
- latest raw note contents for materialize (`path → content`, visible notes only);
- asset bytes (or a reader) by hash/path for asset mirroring;
- `HideNotes(ctx, paths)` for git deletions.

## Error handling

- **Apply fails after ref advanced** → roll the ref back to `oldHEAD`, return error; repo
  stays equal to the DB.
- **Materialize fails** → abort the request with an error before serving; HEAD is unchanged,
  so the mirror is never partially written (commit is the last, atomic step via `update-ref`).
- **Snapshot upload fails** (background) → log only; the next successful push re-uploads.
- **Validation/business errors from `PushNotes`** keep the existing `ErrorPayload` contract.

## Testing

Two layers. Go unit tests cover the internals (mocked `Env`, local temp repo); a Playwright
e2e spec covers the real cross-path coexistence over the smart-HTTP protocol — which the unit
tests can't, since they never exercise the HTTP endpoints or a genuine plugin-vs-git race.

### Go unit tests (`internal/gitapi`)
Table-driven, with a real temporary git repo plus a moq'd `Env`:

- materialize idempotency (second call with unchanged DB makes no commit);
- materialize-on-change creates exactly one commit and `HEAD` tree matches the DB;
- diff detects add / modify / delete;
- deletion → `HideNotes`;
- hash-skip prevents the materialize↔apply loop (push of server-materialized content is a no-op);
- non-fast-forward push is rejected;
- asset blob is written into git and is idempotent by hash;
- ref rollback on apply failure (repo equals `oldHEAD` afterward);
- concurrency: a plugin `pushNotes` and a git apply do not interleave (shared lock).

### E2e (`e2e/gitsync.spec.js`, Playwright)
Setup: create a git token via the `createGitToken` GraphQL mutation (API-key auth), then drive
the real `git` CLI (`child_process`) against `http://localhost:8081/_system/git` using
`http://user:<token>@…` basic auth. The plugin side is simulated with the existing GraphQL
`pushNotes` / `notePaths` calls (the same pattern as `updatenotes.spec.js`). Scenarios:

- **plugin → git:** push a note via GraphQL, then `git clone` and assert the file is present
  with current content (mirror is materialized on access);
- **git → plugin:** `git push` a new/edited note, then query `notePaths` and assert the DB has it;
- **non-fast-forward:** clone, let a GraphQL `pushNotes` change the same file, then `git push`
  from the stale clone and assert it is rejected; after `git pull` + push it succeeds;
- **git deletion:** `git rm` + push, then assert the note is hidden in the DB;
- **assets:** GraphQL-upload a note with an image, `git clone`, assert the image blob is present.

## Out of scope

- Incremental / packed object storage (keep `tar.gz`).
- Server-side per-file merge (relying on git's fast-forward rejection instead).
- Multi-instance coordination of the single `repo.tar.gz` blob (single-instance assumption,
  unchanged from today).
