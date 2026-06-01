# Unreleased Changes API

## Why

trip2g already has a release system: notes have versioned content (`note_versions`), and a
release (`releases` + `release_note_versions`) snapshots the currently-published state.
When `is_live = true`, that release determines what site visitors see.

Until now, this was a human-driven workflow: an admin edits notes in Obsidian, then manually
creates a release via the admin panel. The gap between "latest edited versions" and "live
release" was invisible — you had no way to know what had accumulated since the last publish
without eyeballing the admin UI.

With AI agents able to read and write notes via API key, this gap becomes a first-class
concept. An agent can:

1. **Monitor** — poll for unreleased changes and see what the author edited since the last
   publish, including line/word-level diffs against the released text.
2. **Curate** — review the diff, make its own edits (fix typos, add links, reformat), then
   decide the content is ready.
3. **Publish** — call the existing `createRelease` mutation. Divergence resets to zero.

The feature being designed here is step 1: a single read-only GraphQL query,
`unreleasedChanges`, that exposes this gap with LLM-friendly diff representations.

The agent already has everything it needs for steps 2 and 3 (`pushNotes`/`commitNotes` and
`createRelease` exist and are API-key-accessible). No new mutations are required.

---

## Schema

```graphql
enum NoteChangeType {
  ADDED     # exists in latest, not in live release (or no live release yet)
  MODIFIED  # in both; versionIds differ
  REMOVED   # was in live release, now hidden/deleted from latest
}

type UnreleasedChangeStats {
  addedLines:    Int!
  removedLines:  Int!
  changedWords:  Int!  # word-level count across all hunks
}

type UnreleasedChange {
  path:            String!
  pathId:          Int64!
  title:           String!          # from LatestNoteViews (empty for removed)
  changeType:      NoteChangeType!
  liveVersionId:   Int64            # null when changeType=added
  latestVersionId: Int64            # null when changeType=removed
  stats:           UnreleasedChangeStats! @goField(forceResolver: true)

  # Diff representations — both computed lazily via forceResolver
  unifiedDiff: String!  @goField(forceResolver: true)
  # git-style unified diff on raw markdown; primary format for LLMs
  # ADDED: old side is null; REMOVED: new side is null; diff treats null as ""

  wordDiff: String!     @goField(forceResolver: true)
  # inline word-level diff using {+added+} / {-removed-} markers

  # Raw source, available without resolver
  oldContent: String    # released markdown; null when changeType=ADDED
  newContent: String    # latest markdown;  null when changeType=REMOVED
}

type UnreleasedChangesConnection {
  totalCount:       Int!
  totalStats:       UnreleasedChangeStats! @goField(forceResolver: true)  # lazy: sum of per-node stats
  nodes:            [UnreleasedChange!]!
}

# Added to Query root:
type Query {
  """
  X-Api-Key header required.
  Returns notes whose latest version differs from the current live release.
  If no live release exists, all notes are considered unreleased (changeType=added).
  Supports the same glob filter as noteChanges.
  """
  unreleasedChanges(filter: NoteChangesFilter!): UnreleasedChangesConnection!
}
```

`NoteChangesFilter` (existing, reused):
```graphql
input NoteChangesFilter {
  includePatterns: [String!]!
  excludePatterns: [String!]
}
```

---

## Data model

```
note_paths           version_count → current note_version.id (latest)
note_versions        (path_id, version) — all historical content
releases             is_live bool — at most one live at a time
release_note_versions release_id + note_version_id junction
```

Divergence = notes where `note_paths.version_count` version ≠ the version pinned by the
current live release (or no entry in that release).

Query logic:

1. Find the live release: `SELECT id FROM releases WHERE is_live = true LIMIT 1`.
2. Load its pinned versions: `SELECT path_id, note_version_id FROM release_note_versions WHERE release_id = ?`.
3. Compare against `LatestNoteViews()` (in-memory, already loaded):
   - in latest, not in live → `added`
   - in both, different `version_id` → `modified`
   - in live, not in latest → `removed`
4. Apply `NoteChangesFilter` (doublestar glob on `path`) to include/exclude notes.
5. For each result, load `note_versions.content` for both sides (single query with `WHERE id IN (...)`).

---

## Diff libraries

**Unified diff + word diff** — use `github.com/sergi/go-diff`:

- `dmp.DiffMain(old, new, true)` → word-level diff slices
- Convert to unified format via `dmp.PatchMake` + `dmp.PatchToText`, or use the built-in
  pretty-printer for inline markers
- Both outputs from a single diff computation (compute once, format twice)

This library is battle-tested, pure Go, no CGO, and already the standard choice for Go diffs.

---

## Implementation plan

### 1. Go module: `internal/case/getunreleasedchanges/`

```
resolve.go     — Env interface + Resolve(ctx, env, filter) function
resolve_test.go
mocks_test.go  — generated via moq
```

**Env interface:**

```go
type Env interface {
    LiveRelease(ctx context.Context) (*db.Release, error)
    ReleaseNoteVersions(ctx context.Context, releaseID int64) ([]db.ReleaseNoteVersion, error)
    NoteVersionsByIDs(ctx context.Context, ids []int64) ([]db.NoteVersion, error)
    LatestNoteViews() *appmodel.NoteViews
}
```

`Resolve` returns `[]model.UnreleasedChange` (the resolver builds the connection wrapper).

### 2. DB queries (`internal/db/queries.read.sql`)

Two new queries (then `make sqlc`):

```sql
-- name: LiveRelease :one
SELECT * FROM releases WHERE is_live = true LIMIT 1;

-- name: ReleaseNoteVersions :many
SELECT path_id, note_version_id FROM release_note_versions
JOIN note_versions ON note_versions.id = release_note_versions.note_version_id
WHERE release_id = ?;

-- name: NoteVersionsByIDs :many
SELECT * FROM note_versions WHERE id IN (/*SLICE:ids*/);
```

(`ReleaseNoteVersions` likely already exists — check before adding.)

### 3. GraphQL schema changes (`internal/graph/schema.graphqls`)

Add `NoteChangeType`, `UnreleasedChangeStats`, `UnreleasedChange`,
`UnreleasedChangesConnection` types, and the `unreleasedChanges` field on `Query`.

Then `make gqlgen` to regenerate.

### 4. Resolvers (`internal/graph/schema.resolvers.go` + new file)

- `Query.UnreleasedChanges` — auth check via `checkapikey.Resolve(ctx, env, "unreleased_changes")`,
  call `getunreleasedchanges.Resolve`, wrap in connection.
- `UnreleasedChange.UnifiedDiff` — compute via `go-diff` on `oldContent`/`newContent`.
- `UnreleasedChange.WordDiff` — format the same diff as inline markers.
- `UnreleasedChange.Stats` is computed eagerly in `Resolve` (cheap, part of the diff result).

`totalStats` in the connection is a sum over all nodes' stats — computed in the resolver,
not a lazy field.

### 5. Add `go-diff` dependency

```sh
go get github.com/sergi/go-diff/diffmatchpatch
```

### 6. Wire Env

Add `LiveRelease` and `ReleaseNoteVersions` to `app.App` (or confirm they exist).
`NoteVersionsByIDs` needs a batch-load DB method.

### 7. Tests

Table-driven tests for `getunreleasedchanges.Resolve`:
- No live release → all latest notes are `added`
- Live release with identical versions → empty result
- Modified note → correct `modified` with expected diff content
- Removed note (in release, not in latest) → `removed`
- Glob filter excludes paths correctly

---

## Out of scope / future ideas

- **Change attribution** — `note_versions` does not store `created_by`. Changes from
  Obsidian, admin UI, checkbox toggles, external API clients, and other agents all appear
  as one undifferentiated stream. The agent works with the final diff and figures it out
  from context (e.g. `- [ ]` → `- [x]` is a recognizable checkbox state change).
  If source tracking becomes important, add a `created_by_api_key_id` column to
  `note_versions` and expose it on `UnreleasedChange`.

- **SSE divergence stream** — push notification when new unreleased changes land, so the
  agent doesn't need to poll. Low priority: the existing `noteChanges` subscription already
  signals when edits happen.

- **Per-note version history diff** — diff version N vs N-1 independent of releases, for
  a finer-grained edit history view.

---

## Agent workflow example

```graphql
# 1. Overview — cheap
query {
  unreleasedChanges(filter: { includePatterns: ["**"] }) {
    totalCount
    totalStats { addedLines removedLines changedWords }
    nodes { path changeType stats { addedLines removedLines } }
  }
}

# 2. Deep read on specific notes
query {
  unreleasedChanges(filter: { includePatterns: ["posts/**"] }) {
    nodes {
      path title changeType
      unifiedDiff   # or wordDiff — request only what the agent needs
    }
  }
}

# 3. Publish (existing mutation, no changes)
mutation {
  adminMutation {
    createRelease(input: { title: "2026-05-29 agent curated" }) {
      ... on CreateReleasePayload { release { id } }
    }
  }
}
```
