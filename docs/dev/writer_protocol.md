# Writer Protocol

The trip2g writer protocol is the complete API surface for writing notes. Three GraphQL mutations cover all write paths: `pushNotes` for bulk sync, `updateNotes` for fine-grained batched changes, and `hideNotes` for deletion.

| Mutation | Kind | Concurrency | Auto-commits | Used by |
|---|---|---|---|---|
| `pushNotes` | Full content, upsert | Write-locked; optional per-note `expectedHash` | Optional (`skipCommit`) | Obsidian plugin, memcli sidecar |
| `updateNotes` | Per-change: upsert / patch / meta / hide | Per-note `expectedHash` | Always (upsert/patch/meta) | Kanban board, AI agents (direct), in-browser editors |
| `hideNotes` | Soft-delete batch | Optional `expectedHash` per `notes` entry (`paths` carries none) | Immediate | Obsidian plugin (server-only paths) |
| `commitNotes` | Flush staged `pushNotes` batch | None | Yes | Obsidian plugin (explicit commit flow) |

---

## Transport and Auth

**Endpoint:** `POST /_system/graphql`
**Content-Type:** `application/json`

Auth is checked by `checkapikey.Resolve` (called from each mutation resolver). Two credential sources are accepted:

| Credential | Where | Depth | Write-pattern restriction |
|---|---|---|---|
| `X-API-Key: {key}` | Permanent API key from admin panel | 0 | None |
| `Authorization: Bearer {token}` | Short-lived JWT from webhook `api_token` field | From JWT `d` claim | `wp` claim (glob array) |

Admin session cookie is also accepted for browser-based tools (e.g., the kanban board uses `credentials: 'include'`).

**Write-pattern enforcement (shortapitoken only):** every path in every write call is matched against the JWT's `wp` (write patterns) claim. A path not covered by any pattern returns HTTP 403 for the entire request. Permanent API keys have no path restrictions.

---

## pushNotes

Bulk upsert. Accepts all content types the vault can store. A note without `expectedHash` is written unconditionally; a note with one is written only if the server still holds that hash (see [expectedHash](#expectedhash-optimistic-concurrency)).

```graphql
input PushNoteInput {
  path: String!
  content: String!
  expectedHash: String     # optional; see the expectedHash section
}

input PushNotesInput {
  skipCommit: Boolean
  updates: [PushNoteInput!]!
}

type NoteHashMismatch {
  path: String!
  actualHash: String!      # "" when the note does not exist or is hidden
}

type PushNotesPayload {
  notes: [PushedNote!]!    # all notes currently indexed
  updated: [PushedNote!]!  # only the notes just pushed
  hashMismatches: [NoteHashMismatch!]!  # notes expectedHash refused
}

union PushNotesOrErrorPayload = PushNotesPayload | ErrorPayload
```

**Accepted file extensions:** `.md`, `.html`, `.html.json`, `.canvas`, `.base`, `.excalidraw`.

**`skipCommit` flag:** when absent or `false`, `HandleLatestNotesAfterSave` runs immediately — webhooks fire, the in-memory index refreshes, SSE subscribers are notified. When `true`, the path IDs are saved to an uncommitted staging table and nothing further happens until `commitNotes` is called. This lets a client batch many pushes (e.g., first sync of a large vault) and commit atomically.

The resolver holds a process-wide write mutex for the duration of the call. Concurrent `pushNotes` calls are serialised.

**Per-note `expectedHash`:** a mismatch refuses only that note. The rest of the batch is written, and the refused note is listed in `hashMismatches` with the hash the server holds now. It is not in `updated`, and none of its side effects (webhooks, SSE, uncommitted staging) happen. Validation still applies to the whole batch first: an unsupported extension, binary content or a path listed twice returns `ErrorPayload` and writes nothing.

**`commitNotes`** (no input) flushes all paths staged with `skipCommit: true`:

```graphql
type CommitNotesPayload {
  success: Boolean!
  updated: [PushedNote!]!
}
```

`commitNotes` calls `HandleLatestNotesAfterSave` for all staged path IDs, then clears the staging table.

**Example — bulk sync:**

```graphql
mutation PushNotes($input: PushNotesInput!) {
  pushNotes(input: $input) {
    ... on PushNotesPayload {
      updated { id path url warnings { level message } }
    }
    ... on ErrorPayload { message }
  }
}
```

```json
{
  "input": {
    "updates": [
      { "path": "blog/hello.md", "content": "# Hello\n\nFirst post." },
      { "path": "blog/about.md", "content": "# About\n\nThis is me." }
    ]
  }
}
```

---

## updateNotes

Fine-grained batch of per-note changes. Each element in `changes` is a tagged union with exactly one field set: `upsert`, `patch`, `meta`, or `hide`.

```graphql
input NoteChangeUpsertInput {
  path: String!
  content: String!
  expectedHash: String  # see expectedHash section below
}

input NoteChangePatchInput {
  path: String!
  find: String!         # exact substring to locate (must appear exactly once)
  replace: String!      # replacement string
  expectedHash: String
}

input NoteChangeMetaInput {
  path: String!
  set: Map              # top-level frontmatter keys to add or replace
  unset: [String!]      # top-level frontmatter keys to remove
  expectedHash: String
}

input NoteChangeHideInput {
  path: String!
}

input NoteChangeInput {
  upsert: NoteChangeUpsertInput
  patch: NoteChangePatchInput
  hide: NoteChangeHideInput
  meta: NoteChangeMetaInput
}

input UpdateNotesInput {
  changes: [NoteChangeInput!]!
}
```

**Payloads:**

```graphql
type UpdateNotesSuccessPayload {
  paths: [String!]!  # paths that were touched (upsert, patch, meta, and hide)
}

type UpdateNotesHashMismatchPayload {
  path: String!
  actualHash: String!  # current server hash; use to rebase and retry
}

type UpdateNotesPatchNotFoundPayload {
  path: String!
  find: String!  # the string that was not found or was ambiguous
}

union UpdateNotesOrErrorPayload =
    UpdateNotesSuccessPayload
  | UpdateNotesHashMismatchPayload
  | UpdateNotesPatchNotFoundPayload
  | ErrorPayload
```

The mutation processes `changes` in order. The first change that returns a non-success payload aborts the batch and returns that payload immediately. Only content changes (upsert, patch and meta) call `HandleLatestNotesAfterSave` afterward; hide-only batches still reload the in-memory index but skip webhook processing.

**Note:** the resolver does not hold the note write mutex. Do not include the same `path` more than once in a single `updateNotes` call — the second operation would read stale in-memory content.

### Change kinds

**upsert** — full content replacement. Creates the note if it doesn't exist; overwrites it if it does. `expectedHash` controls concurrency (see below).

**patch** — exact-string find/replace within the note's current content. The server reads the in-memory content, calls `strings.Index(content, find)`, substitutes one occurrence, and saves. Rules:
- `find` must appear exactly once; zero occurrences → `UpdateNotesPatchNotFoundPayload`; two or more occurrences → `UpdateNotesPatchNotFoundPayload` (the `find` field echoes the ambiguous string back to the caller)
- The note must already exist; patching a non-existent path → `ErrorPayload { message: "note not found: {path}" }`
- `find == replace` after substitution → `InsertNote` detects identical content and skips creating a new version (idempotent)
- `replace` may be empty string (deletes the found substring)
- `find` is a literal string, not a regular expression

**meta** — sets and unsets top-level frontmatter keys of an existing note by splicing text (`internal/frontmatteredit`): every untouched key line, comment and the body come out byte for byte. Rules:
- The note must already exist; a missing path → `ErrorPayload { message: "note not found: {path}" }`
- `set` and `unset` cannot both be empty, a key cannot be in both, and a key cannot be blank or contain a control character
- Frontmatter that cannot be edited key by key (invalid YAML, a `{...}` flow mapping, indented keys, aliases, a duplicate key, no closing fence) → `ErrorPayload`
- The result is parsed back with yaml.v3 and with yaml.v2 (what goldmark-meta reads). Each parser must see exactly the expected keys, each untouched key must read as that parser read it before the edit, and each set key must hold the value sent, compared with the value itself (numbers by worth, NaN equal to NaN) rather than with what the encoder wrote; otherwise nothing is saved → `ErrorPayload`
- Setting the scalar a key already holds keeps the line, so `InsertNote` sees identical content and skips the version

**hide** — soft-deletes the path (sets `hidden_at` / `hidden_by`). Does NOT trigger webhooks from `updateNotes` (see code comment in `internal/case/updatenotes/resolve.go`); use the standalone `hideNotes` mutation when webhook notification is required.

---

## hideNotes

Standalone soft-delete. Triggers webhooks and publishes to the `noteChanges` SSE bus.

```graphql
input HideNoteInput {
  path: String!
  expectedHash: String     # optional; see the expectedHash section
}

input HideNotesInput {
  paths: [String!]! = []        # hidden unconditionally
  notes: [HideNoteInput!]! = [] # hidden per expectedHash
}

type HideNotesPayload {
  success: Boolean!
  hashMismatches: [NoteHashMismatch!]!  # notes entries expectedHash refused
}

union HideNotesOrErrorPayload = HideNotesPayload | ErrorPayload
```

`paths` and `notes` are both optional and are processed in that order. The hash is per `notes` entry; `paths` carries none, so a path in `paths` behaves like a `notes` entry without `expectedHash`. A path may appear only once across `paths` and `notes` together: a repeat returns `ErrorPayload` naming it, and nothing is hidden. A refused path stays visible, is listed in `hashMismatches`, and fires no webhook or SSE event. `success` is `true` whenever the request was processed, mismatches included. A delete that saw an old hash is the case this guards: if the note was edited after the client listed it, the hide is refused instead of hiding the edit.

---

## expectedHash: Optimistic Concurrency

`expectedHash` is available on `updateNotes`' `upsert`, `patch` and `meta` changes, on each `pushNotes` update and on each `hideNotes` `notes` entry. It is a Go pointer (`*string`), so the GraphQL field is truly optional: omitting it and setting it to `null` are equivalent.

| `expectedHash` value | Behaviour |
|---|---|
| absent / `null` | Blind write — no concurrency check, last writer wins |
| `""` (empty string) | **Create-only sentinel** — succeeds iff the note does not yet exist |
| Non-empty string | Optimistic update — applies iff `sha256(currentContent)` matches this value |

**Create-only sentinel detail:** when a note doesn't exist, the server computes `actualHash = ""` (empty string, because there is no content to hash). Setting `expectedHash: ""` makes the equality check `"" == ""` pass, so the note is created. If the note already exists, `actualHash` is a non-empty hash and the check fails, returning `UpdateNotesHashMismatchPayload { path, actualHash }` with the current hash. This is the race-proof "fail-if-exists" primitive.

This behaviour is enforced in `internal/case/updatenotes/resolve.go` and tested by `TestResolve_UpsertCreateOnly` in `resolve_test.go`.

### pushNotes and hideNotes

The values mean the same, with three differences from `updateNotes`:

- **Per note, not per batch.** `updateNotes` stops the whole batch at the first mismatch and returns `UpdateNotesHashMismatchPayload`. `pushNotes` and `hideNotes` skip only the refused note and list it in `hashMismatches`.
- **Checked against the database.** The current hash is `note_paths.latest_content_hash` read inside the request transaction (`LatestNoteContentHash` in `cmd/server/notes.go`), not the in-memory note views `updateNotes` reads.
- **A hidden note counts as absent.** Its `actualHash` is `""`, so `expectedHash: ""` on `pushNotes` writes it and brings it back, and `expectedHash: ""` on `hideNotes` hides nothing new.

For `hideNotes`, `""` matches only a note that is absent or already hidden, so it is a no-op: no `hidden_by`/`hidden_at` re-stamp, no webhook, no SSE event, no reload. The useful value is the hash the client last saw.

**Why the check cannot race.** Every GraphQL mutation runs in one request transaction on the single write connection, opened with `BEGIN IMMEDIATE` (see `internal/graph/handler.go` and [sqlite.md](sqlite.md)). A second writer waits for that transaction to commit, so nothing lands between the hash read and the write, and of two pushes that quote the same hash, the second sees the first one's hash and is refused. `TestPushNotesExpectedHashConcurrentWrites` in `cmd/server/note_expected_hash_test.go` runs exactly this sequence.

A client that listed hashes with `notePaths { value latestContentHash }` sends the hash it classified against: the last-synced hash for a plain push or delete, the remote hash it saw when it resolves a conflict in favour of the local copy. Classify against the unfiltered listing (`notePaths` with no filter): it holds only visible notes, so a hidden note is absent from it, matching the `""` the server reports for it. `notePaths(filter: { paths })` still returns a hidden note, with the last hash it had before it was hidden, so it is not the listing to classify against.

### Three upsert modes

| `expectedHash` | Mode | Semantics |
|---|---|---|
| absent / `null` | Blind upsert | Create or overwrite unconditionally |
| `""` | Create-only | Create iff absent; existing note → `HashMismatch` |
| Non-empty hash | Optimistic update | Overwrite iff current content matches this hash |

**Hash algorithm:** SHA-256 of the note's raw byte content, encoded as base64url with padding.

Go (server, `internal/case/updatenotes/resolve.go` and `internal/case/insertnote/resolve.go`):

```go
base64.URLEncoding.EncodeToString(sha256.Sum256([]byte(content)))
```

JavaScript (client, `templates/kanban/src/api.ts`):

```js
const buf = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(content))
const hash = btoa(String.fromCharCode(...new Uint8Array(buf)))
  .replace(/\+/g, '-')
  .replace(/\//g, '_')
// btoa produces standard base64 with '=' padding; the replacements yield base64url
// This is identical to Go's base64.URLEncoding (which also keeps '=' padding)
```

---

## Result Payloads and Client Handling

| Payload | `__typename` | When | Client action |
|---|---|---|---|
| `UpdateNotesSuccessPayload` | Success | All changes applied | `paths` lists every path that was touched |
| `UpdateNotesHashMismatchPayload` | Conflict | `expectedHash` mismatch | Re-read `actualHash`, rebase change onto current content, retry |
| `UpdateNotesPatchNotFoundPayload` | Find failed | `find` string absent or appeared more than once | Choose a more specific string; surface error to caller |
| `ErrorPayload` | Validation / system | Note not found, write denied, etc. | `message` is human-readable |
| `PushNotesPayload.hashMismatches`, `HideNotesPayload.hashMismatches` | Per-note conflict | `expectedHash` mismatch on one note | The other notes were applied. Treat each listed path as a conflict against `actualHash` |

**Rebase loop (HashMismatch):** fetch the current content of the note via `admin { noteVersionHistory }` then `noteVersion { content }`, recompute the desired change against the new content, set `expectedHash` to `actualHash` from the payload, and retry. The kanban template implements this in `templates/kanban/src/api.ts`.

---

## Who Uses What

### Obsidian sync plugin (`trip2g_obsidian_plugin/main.ts`)

Calls `pushNotes` with all changed notes (no `skipCommit`), then calls `hideNotes` for paths that exist on the server but were deleted locally. Does not use `updateNotes`. Auth: `X-API-Key` header, URL `{apiUrl}/graphql`.

### memcli (`cli/memcli`)

The `up` command runs `trip2g-sync --watch` as a background process, which drives the same Obsidian sync plugin code path (`pushNotes`). The `daily` and `log` subcommands write Markdown files directly to the vault folder on disk; the watcher picks up the changes and syncs via `pushNotes`.

### Kanban board (`templates/kanban/src/api.ts`)

Uses `updateNotes` exclusively — upsert for card creation and full-board saves, patch for in-place card edits (title, status changes). Auth: admin session cookie (`credentials: 'include'`). Endpoint: `/_system/graphql`. Implements the full rebase-on-HashMismatch loop.

### Web editor (`assets/ui/editor/pane/pane.view.ts`)

Uses `updateNotes` upsert with `expectedHash: ""` for the "create new file" flow as the server-side race guard against creating over an existing note. A client-side path-existence check provides the fast UX pre-reject; the empty-hash sentinel is the authoritative server guarantee behind it.

### AI agents (via change webhooks)

When a change webhook fires with `pass_api_key: true`, the delivery payload includes `api_token` (a shortapitoken JWT with scoped `read_patterns` / `write_patterns`). Agents can use this token to call `updateNotes` or `pushNotes` directly for programmatic writes.

Agents that return changes synchronously in their HTTP response body use the **webhook agent response format** (`internal/webhookutil/agentresponse.go`). This is a separate code path from the GraphQL mutations: the server parses `changes` and calls `InsertNote` directly. A change without `kind` (or `"kind": "write"`) replaces the full content:

```json
{
  "status": "ok",
  "message": "Linted 3 files",
  "changes": [
    {
      "path": "blog/post.md",
      "content": "# Fixed post\n\nCorrected content.",
      "expected_hash": "base64url-sha256-of-previous-content"
    }
  ]
}
```

`"kind": "patch"` with `find` / `replace` and `"kind": "meta"` with `set` / `unset` follow the updateNotes patch and meta rules; `expected_hash` applies to every kind. The format has no `hide`; for that, agents use the shortapitoken to call `updateNotes` directly via `/_system/graphql`.

---

## Practical Recipes

### Create a note (fail if it already exists)

```graphql
mutation CreateNote($i: UpdateNotesInput!) {
  updateNotes(input: $i) {
    __typename
    ... on UpdateNotesSuccessPayload { paths }
    ... on UpdateNotesHashMismatchPayload { path actualHash }
    ... on ErrorPayload { message }
  }
}
```

```json
{
  "i": {
    "changes": [{
      "upsert": {
        "path": "inbox/new-idea.md",
        "content": "---\nfree: true\n---\n\n# New Idea\n\nContent here.",
        "expectedHash": ""
      }
    }]
  }
}
```

Returns `HashMismatchPayload` (with `actualHash` set to the real hash) if the note already exists.

### Edit one line (patch)

```json
{
  "i": {
    "changes": [{
      "patch": {
        "path": "todo/sprint.md",
        "find": "- [ ] Deploy v2.0",
        "replace": "- [x] Deploy v2.0"
      }
    }]
  }
}
```

### Anchor-marker append (sequential inserts via sentinel)

Keep a `$INBOX$` sentinel in the note. Each insert prepends the new entry above the sentinel and re-places the sentinel so the next call finds it:

```json
{
  "i": {
    "changes": [{
      "patch": {
        "path": "inbox.md",
        "find": "$INBOX$",
        "replace": "## 2026-06-28 09:15\nNew thought.\n\n$INBOX$"
      }
    }]
  }
}
```

Successive patches accumulate above the sentinel, which stays in place as the insertion point.

### Bulk sync (Obsidian plugin path)

```json
{
  "input": {
    "updates": [
      { "path": "notes/a.md", "content": "..." },
      { "path": "notes/b.md", "content": "..." }
    ]
  }
}
```

Omit `skipCommit` (or set `false`) to fire webhooks immediately. Use `skipCommit: true` + a subsequent `commitNotes` call to batch the entire vault load as a single atomic publish.

### Rebase on HashMismatch

```js
let result = await updateNotes([patchChange(path, find, replace, expectedHash)])
if ('hashMismatch' in result) {
  const fetched = await fetchNoteContent(path)         // re-read from server
  const newContent = applyEdit(fetched.content, ...)   // rebase the edit
  const newHash = await sha256Base64(fetched.content)
  result = await updateNotes([upsertChange(path, newContent, newHash)])
}
```

The kanban template's `updateNotes` function in `templates/kanban/src/api.ts` implements exactly this pattern.

### Delete (hide)

```graphql
mutation HideNotes($input: HideNotesInput!) {
  hideNotes(input: $input) {
    ... on HideNotesPayload { success }
    ... on ErrorPayload { message }
  }
}
```

```json
{ "input": { "paths": ["drafts/abandoned.md"] } }
```

To hide only if nobody edited the note since you read it, pass its hash and ask for the mismatches:

```graphql
mutation HideNotes($input: HideNotesInput!) {
  hideNotes(input: $input) {
    ... on HideNotesPayload { success hashMismatches { path actualHash } }
    ... on ErrorPayload { message }
  }
}
```

```json
{ "input": { "notes": [{ "path": "drafts/abandoned.md", "expectedHash": "<hash you read>" }] } }
```

Use `hideNotes` (not `updateNotes` hide) when you need webhooks to fire on removal.

---

## Edge Cases and Constraints

| Situation | Behaviour |
|---|---|
| `patch` on a note that doesn't exist | `ErrorPayload { message: "note not found: {path}" }` |
| `patch` with zero occurrences of `find` | `UpdateNotesPatchNotFoundPayload` |
| `patch` with two or more occurrences of `find` | `UpdateNotesPatchNotFoundPayload` (same payload; use a more specific string) |
| `upsert` with `expectedHash: ""` on an existing note | `UpdateNotesHashMismatchPayload { actualHash: "<real hash>" }` |
| Content identical to current version | `InsertNote` skips creating a new version; `paths` still includes the path in the success payload |
| Same path appears twice in one `updateNotes` call | Second change reads stale in-memory content — callers must avoid this |
| `pushNotes` with unsupported extension | `ErrorPayload` |
| `pushNotes` update with a stale `expectedHash` | That note is skipped and listed in `hashMismatches`; the rest of the batch is written |
| `hideNotes` entry with a stale `expectedHash` | That path stays visible and is listed in `hashMismatches`; the rest are hidden |
| Same path twice in one `pushNotes` call | `ErrorPayload` naming the path; nothing is written |
| Same path twice in one `hideNotes` call, within or across `paths` and `notes` | `ErrorPayload` naming the path; nothing is hidden |
| File content detected as binary | `ErrorPayload` (MIME check on first 512 bytes) |
| Shortapitoken with `wp: []` (empty write patterns) | HTTP 403 for any write |
| `updateNotes` hide | Does NOT fire webhooks; use standalone `hideNotes` for that |
