# Writer Protocol

The trip2g writer protocol is the complete API surface for writing notes. Three GraphQL mutations cover all write paths: `pushNotes` for bulk blind sync, `updateNotes` for fine-grained batched changes, and `hideNotes` for deletion.

| Mutation | Kind | Concurrency | Auto-commits | Used by |
|---|---|---|---|---|
| `pushNotes` | Full content, blind upsert | Write-locked, no per-note check | Optional (`skipCommit`) | Obsidian plugin, memcli sidecar |
| `updateNotes` | Per-change: upsert / patch / hide | Per-note `expectedHash` | Always (upsert/patch) | Kanban board, AI agents (direct), in-browser editors |
| `hideNotes` | Soft-delete batch | None | Immediate | Obsidian plugin (server-only paths) |
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

Blind bulk upsert. No per-note concurrency check. Accepts all content types the vault can store.

```graphql
input PushNoteInput {
  path: String!
  content: String!
}

input PushNotesInput {
  skipCommit: Boolean
  updates: [PushNoteInput!]!
}

type PushNotesPayload {
  notes: [PushedNote!]!    # all notes currently indexed
  updated: [PushedNote!]!  # only the notes just pushed
}

union PushNotesOrErrorPayload = PushNotesPayload | ErrorPayload
```

**Accepted file extensions:** `.md`, `.html`, `.html.json`, `.canvas`, `.base`, `.excalidraw`.

**`skipCommit` flag:** when absent or `false`, `HandleLatestNotesAfterSave` runs immediately — webhooks fire, the in-memory index refreshes, SSE subscribers are notified. When `true`, the path IDs are saved to an uncommitted staging table and nothing further happens until `commitNotes` is called. This lets a client batch many pushes (e.g., first sync of a large vault) and commit atomically.

The resolver holds a process-wide write mutex for the duration of the call. Concurrent `pushNotes` calls are serialised.

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

Fine-grained batch of per-note changes. Each element in `changes` is a tagged union with exactly one field set: `upsert`, `patch`, or `hide`.

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

input NoteChangeHideInput {
  path: String!
}

input NoteChangeInput {
  upsert: NoteChangeUpsertInput
  patch: NoteChangePatchInput
  hide: NoteChangeHideInput
}

input UpdateNotesInput {
  changes: [NoteChangeInput!]!
}
```

**Payloads:**

```graphql
type UpdateNotesSuccessPayload {
  paths: [String!]!  # paths that were touched (upsert, patch, and hide)
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

The mutation processes `changes` in order. The first change that returns a non-success payload aborts the batch and returns that payload immediately. Only content changes (upsert and patch) call `HandleLatestNotesAfterSave` afterward; hide-only batches still reload the in-memory index but skip webhook processing.

**Note:** the resolver does not hold the note write mutex. Do not include the same `path` more than once in a single `updateNotes` call — the second operation would read stale in-memory content.

### Change kinds

**upsert** — full content replacement. Creates the note if it doesn't exist; overwrites it if it does. `expectedHash` controls concurrency (see below).

**patch** — exact-string find/replace within the note's current content. The server reads the in-memory content, calls `strings.Index(content, find)`, substitutes one occurrence, and saves. Rules:
- `find` must appear exactly once; zero occurrences → `UpdateNotesPatchNotFoundPayload`; two or more occurrences → `UpdateNotesPatchNotFoundPayload` (the `find` field echoes the ambiguous string back to the caller)
- The note must already exist; patching a non-existent path → `ErrorPayload { message: "note not found: {path}" }`
- `find == replace` after substitution → `InsertNote` detects identical content and skips creating a new version (idempotent)
- `replace` may be empty string (deletes the found substring)
- `find` is a literal string, not a regular expression

**hide** — soft-deletes the path (sets `hidden_at` / `hidden_by`). Does NOT trigger webhooks from `updateNotes` (see code comment in `internal/case/updatenotes/resolve.go`); use the standalone `hideNotes` mutation when webhook notification is required.

---

## hideNotes

Standalone soft-delete. Triggers webhooks and publishes to the `noteChanges` SSE bus.

```graphql
input HideNotesInput {
  paths: [String!]!
}

type HideNotesPayload {
  success: Boolean!
}

union HideNotesOrErrorPayload = HideNotesPayload | ErrorPayload
```

---

## expectedHash: Optimistic Concurrency

`expectedHash` is available on `upsert` and `patch` changes. It is a Go pointer (`*string`), so the GraphQL field is truly optional: omitting it and setting it to `null` are equivalent.

| `expectedHash` value | Behaviour |
|---|---|
| absent / `null` | Blind write — no concurrency check, last writer wins |
| `""` (empty string) | **Create-only sentinel** — succeeds iff the note does not yet exist |
| Non-empty string | Optimistic update — applies iff `sha256(currentContent)` matches this value |

**Create-only sentinel detail:** when a note doesn't exist, the server computes `actualHash = ""` (empty string, because there is no content to hash). Setting `expectedHash: ""` makes the equality check `"" == ""` pass, so the note is created. If the note already exists, `actualHash` is a non-empty hash and the check fails, returning `UpdateNotesHashMismatchPayload { path, actualHash }` with the current hash. This is the race-proof "fail-if-exists" primitive.

This behaviour is enforced in `internal/case/updatenotes/resolve.go` and tested by `TestResolve_UpsertCreateOnly` in `resolve_test.go`.

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

Agents that return changes synchronously in their HTTP response body use the **webhook agent response format** (`internal/webhookutil/agentresponse.go`). This is a separate code path from the GraphQL mutations: the server parses `changes` and calls `InsertNote` directly. The current format supports only full content replacement:

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

This response format does not yet support `patch` or `hide` operations. For those, agents should use the shortapitoken to call `updateNotes` directly via `/_system/graphql`.

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
| File content detected as binary | `ErrorPayload` (MIME check on first 512 bytes) |
| Shortapitoken with `wp: []` (empty write patterns) | HTTP 403 for any write |
| `updateNotes` hide | Does NOT fire webhooks; use standalone `hideNotes` for that |
