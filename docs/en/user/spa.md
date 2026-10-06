---
title: An app on top of trip2g
free: true
lang_redirect: "[[ru/user/spa]]"
---

A custom layout doesn't have to render a page. It can ship a JavaScript app that shows a note as something interactive and writes the changes back. The note stays plain markdown: Obsidian, an agent and the app all edit the same file.

The [[en/user/kanban|Kanban board template]] works this way. It turns an obsidian-kanban note into a drag-and-drop board, and every move is saved back into the note's markdown. This page describes the pattern it follows, so you can build your own app: a checklist, a table editor, a form builder, a dashboard.

New to layouts? Read [[en/user/templates|Templates]] first.

### The three parts

| Part | Where it lives | What it does |
|---|---|---|
| Layout | `_layouts/app.html` in the vault | Puts the note's data into the page, a mount element, and a `<script>` that loads the app |
| App bundle | Next to the layout, or any URL | Reads the data from the page, renders the UI |
| GraphQL API | `/_system/graphql` on your site | Saves the edited markdown back into the note |

A note opts in with `layout: app` in its frontmatter, like any other layout.

### The layout: ship the note in the page

The layout gives the app three things: the note's vault path, its raw markdown, and whether the visitor may edit. `json()` puts them into a `<script type="application/json">` element:

```jet
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ title }}</title>
</head>
<body>
  <div id="app"></div>

  <script type="application/json" id="app-data">
    {{ json(map(
      "path", note.Path(),
      "content", note.ContentString(),
      "editable", currentUser.IsAdmin()
    )) }}
  </script>
  <script src="{{ asset("app.js") }}"></script>
</body>
</html>
```

- `note.ContentString()` is the note's markdown source, frontmatter included. It is the same text the server hashes for conflict checks, so the app can edit it and send it back.
- `json()` escapes `<`, `>` and `&` as `<`…, so markdown that contains `</script>` can't close the element. `JSON.parse` gives back the exact text.
- `currentUser.IsAdmin()` only decides whether the app shows its edit controls. The server checks every save again (see below).
- Calling `currentUser` makes the page personalized: it is not served from the anonymous page cache, so the app always starts from the current version of the note.

The kanban layout uses a different carrier for the same data: the markdown goes into a hidden `<textarea>` and the path into a hidden `<span>`, both printed with plain `{{ … }}`. That is safe too, because output is escaped by default: a note containing `</textarea>` reaches the page as `&lt;/textarea&gt;`, and the browser decodes it back. A `<textarea>` has two quirks, though: the browser normalizes its line endings to `\n` and drops a newline that comes right after the opening tag. The JSON island has neither.

To show trip2g's sign-in button on the app's page, add `{{ defaultTemplate.UserSpaceScripts() }}` to `<head>` and a mount element for it, as the kanban layout does.

### The bundle

Two ways to serve the JavaScript:

- **Next to the layout.** Put `app.js` beside `_layouts/app.html` and load it with `{{ asset("app.js") }}`, as above. The URL carries a cache-busting hash. See [[en/user/yield_blocks#Static assets: asset()|asset()]].
- **From a release.** The kanban layout loads its bundle from a GitHub release URL, `…/releases/latest/download/kanban.js`. Users install one HTML file and get updates without syncing anything.

The bundle can be any framework or none. Kanban is a React app built with esbuild into one self-contained file.

The app reads its data on start:

```js
const data = JSON.parse(
  document.getElementById('app-data').textContent
)
// data.path, data.content, data.editable
```

### Saving: the `updateNotes` mutation

The app writes the note back with the `updateNotes` GraphQL mutation at `/_system/graphql`. A same-origin `fetch` with `credentials: 'include'` sends the visitor's session cookie. A signed-in site admin is allowed to write; anyone else gets an error, so a visitor's browser can't change the note even if the app shows its controls by mistake.

```js
const UPDATE = `mutation ($i: UpdateNotesInput!) {
  updateNotes(input: $i) {
    __typename
    ... on UpdateNotesHashMismatchPayload { actualHash }
    ... on ErrorPayload { message }
  }
}`

async function save(path, content, expectedHash) {
  const change = { upsert: { path, content, expectedHash } }
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      query: UPDATE,
      variables: { i: { changes: [change] } },
    }),
  })
  const body = await res.json()
  if (body.errors) throw new Error(body.errors[0].message)
  return body.data.updateNotes
}
```

`__typename` says how it went:

| `__typename` | Meaning |
|---|---|
| `UpdateNotesSuccessPayload` | Saved |
| `UpdateNotesHashMismatchPayload` | Someone changed the note since the app loaded it; `actualHash` is the current hash |
| `UpdateNotesPatchNotFoundPayload` | A `patch` change didn't find its `find` text, or found it more than once |
| `ErrorPayload` | Refused, with a `message` |

A visitor who isn't the admin doesn't reach a payload at all: the response carries an `errors` entry and no `data.updateNotes`, which `save` above turns into an exception.

**Don't overwrite someone else's edit.** `expectedHash` makes the save conditional: the server saves only if the note still hashes to that value. The hash is SHA-256 of the markdown, in URL-safe base64 with padding:

```js
async function contentHash(text) {
  const bytes = new TextEncoder().encode(text)
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  return btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
}

const hash = await contentHash(data.content)
const result = await save(data.path, newContent, hash)
```

On `UpdateNotesHashMismatchPayload` reload the page, or fetch the new content and apply your change to it. Kanban re-reads the latest version through the admin `noteVersionHistory` and `noteVersion` queries and replays the move.

**Small edits.** Instead of `upsert`, which replaces the whole note, a change can be a `patch`: `{ patch: { path, find, replace, expectedHash } }` replaces one exact piece of text. Kanban toggles a checkbox this way, so text it doesn't model is never rewritten. Both forms, batches and error cases are in [[en/user/update_notes|updateNotes]].

### Live updates

To pick up edits made in Obsidian or by an agent while the app is open, subscribe to `noteChanges` with a filter on the note's path. The subscription runs over server-sent events on the same `/_system/graphql` endpoint and accepts the admin session cookie. Kanban uses it to refresh the board without a reload; its `src/api.ts` is a working client.

### Ready-made GraphQL operations

Copy these into your app. Each one goes to `/_system/graphql` as a `POST` with a JSON body `{ query, variables }`: the operation is `query`, the JSON block under it is `variables`. This helper sends one and returns `data`:

```js
async function gql(query, variables) {
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  const body = await res.json()
  if (body.errors) throw new Error(body.errors[0].message)
  return body.data
}
```

Most results are unions: ask for `__typename` and one `... on` fragment per type, then branch on `__typename`. A refusal the app should show comes back as a payload type such as `ErrorPayload`; a caller with no access gets an `errors` entry instead, which `gql` turns into an exception.

**Who may call what.** The details are in [[en/user/update_notes#Authentication|updateNotes → Authentication]].

| Caller | Sends | Reaches |
|---|---|---|
| Signed-in admin | Session cookie, or `Authorization: Bearer t2g_…` | Everything below, `admin { … }` included, except hiding a note |
| API key | `X-Api-Key: …` | Everything but `admin { … }`; writes only within the key's write patterns |
| Webhook token | `Authorization: Bearer eyJ…` | What an API key reaches, within the token's read and write patterns, except hiding a note |
| Any visitor | Nothing | `search`, and `noteChanges` once signed in |

An app on a page visitors open authenticates with the admin's session cookie, which `credentials: 'include'` sends. Don't put an API key in a bundle: anyone who loads the page can read it.

#### Read a note's markdown

```graphql
query ReadNote($paths: [String!]) {
  notePaths(filter: { paths: $paths }) {
    value
    content
    latestContentHash
  }
}
```

```json
{ "paths": ["boards/todo.md"] }
```

`value` is the vault path and `content` the markdown with its frontmatter. `latestContentHash` is the hash `expectedHash` takes, so the app doesn't have to compute it. A path that doesn't exist is missing from the list.

#### List notes

```graphql
query ListNotes($filter: NotePathsFilter) {
  notePaths(filter: $filter) {
    value
    latestContentHash
    latestNoteView {
      title
      url
    }
  }
}
```

```json
{ "filter": { "like": "boards/%" } }
```

```json
{ "filter": { "frontmatter": [{ "key": "layout", "equals": "app" }] } }
```

`like` is an SQL `LIKE` pattern: `%` matches any run of characters, `_` one character. The filter also takes `search` (full-text search over notes) and `paths`; when several are set, `paths` wins over `search`, and `search` over `like`. `frontmatter` narrows any of them. With no filter you get every path in the vault that isn't hidden.

#### Search

```graphql
query Search($input: SearchInput!) {
  search(input: $input) {
    totalCount
    nodes {
      url
      highlightedTitle
      highlightedContent
      document {
        ... on PublicNote {
          path
          title
        }
      }
    }
  }
}
```

```json
{ "input": { "query": "release plan" } }
```

The site's own search box, open to any visitor. Notes the visitor can't read come last, with a title and a URL but no `document`. An admin searches the latest versions; others search what is published, unless the site shows drafts to everyone.

#### Save a whole note

```graphql
mutation SaveNotes($input: UpdateNotesInput!) {
  updateNotes(input: $input) {
    __typename
    ... on UpdateNotesSuccessPayload {
      updated {
        path
        versionId
      }
    }
    ... on UpdateNotesHashMismatchPayload {
      path
      actualHash
    }
    ... on UpdateNotesPatchNotFoundPayload {
      path
      find
    }
    ... on ErrorPayload {
      message
    }
  }
}
```

```json
{
  "input": {
    "changes": [
      {
        "upsert": {
          "path": "boards/todo.md",
          "content": "# Todo\n\n- [ ] Ship it\n",
          "expectedHash": "latestContentHash from ReadNote"
        }
      }
    ]
  }
}
```

`upsert` replaces the note or creates it. Pass `expectedHash` and the save happens only if nobody changed the note since you read it; otherwise you get `UpdateNotesHashMismatchPayload`. An empty `expectedHash` means "create only": the save fails if the note already exists. Without `expectedHash` the save always overwrites. `updated[].versionId` is the version the save wrote.

#### Change one piece of text

The same `SaveNotes` mutation, with a `patch` change:

```json
{
  "input": {
    "changes": [
      {
        "patch": {
          "path": "boards/todo.md",
          "find": "- [ ] Ship it",
          "replace": "- [x] Ship it",
          "expectedHash": "latestContentHash from ReadNote"
        }
      }
    ]
  }
}
```

`find` must occur in the note exactly once. If it is missing, or appears more than once, the note is left as it is and the result is `UpdateNotesPatchNotFoundPayload`. One call can carry several changes, for several notes; see [[en/user/update_notes|updateNotes]].

#### Hide notes

```graphql
mutation HideNotes($input: HideNotesInput!) {
  hideNotes(input: $input) {
    __typename
    ... on HideNotesPayload {
      success
    }
    ... on ErrorPayload {
      message
    }
  }
}
```

```json
{ "input": { "paths": ["boards/old.md"] } }
```

A hidden note disappears from the site. Saving it again with `updateNotes` brings it back. A `hide` change in `updateNotes`, `{ "hide": { "path": "boards/old.md" } }`, does the same within a batch.

Hiding needs an API key today. Sent with an admin session or a personal token, both `hideNotes` and the `hide` change fail with an `errors` entry; a webhook token gets `ErrorPayload` from `hideNotes` and the same error from the `hide` change.

#### Upload a file

A file goes in as a `multipart/form-data` request, in the [GraphQL multipart request](https://github.com/jaydenseric/graphql-multipart-request-spec) format: an `operations` part with the query and variables, a `map` part that says which variable the file fills, and the file itself.

```graphql
mutation UploadAsset($input: UploadNoteAssetInput!) {
  uploadNoteAsset(input: $input) {
    __typename
    ... on UploadNoteAssetPayload {
      uploadSkipped
    }
    ... on ErrorPayload {
      message
    }
  }
}
```

```js
async function sha256Hex(blob) {
  const bytes = await blob.arrayBuffer()
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

async function upload(file, noteId, path, absolutePath) {
  const input = {
    file: null,
    noteId,
    sha256Hash: await sha256Hex(file),
    path,
    absolutePath,
  }
  const form = new FormData()
  form.append('operations', JSON.stringify({
    query: UPLOAD_ASSET,
    variables: { input },
  }))
  form.append('map', JSON.stringify({ 0: ['variables.input.file'] }))
  form.append('0', file, file.name)
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    body: form,
  })
  const body = await res.json()
  if (body.errors) throw new Error(body.errors[0].message)
  return body.data.uploadNoteAsset
}
```

`UPLOAD_ASSET` is the operation above. Don't set `Content-Type` yourself: the browser adds it with the multipart boundary.

A file belongs to a note version, so save the markdown that links to it first:

1. Save the note with `SaveNotes`, its markdown containing, say, `![Plan](plan.png)`.
2. Take `versionId` from `updated` and pass it as `noteId`.
3. Pass `path` exactly as the markdown writes the link (`plan.png`), and `absolutePath` as the file's path in the vault (`boards/plan.png`).

`sha256Hash` is the file's SHA-256 in hex, not the base64 of `expectedHash`; the server checks it after the upload. A `path` the note doesn't link to is refused with an `ErrorPayload` that lists the links it has. `uploadSkipped: true` means the server already had this file and only linked it to the new version.

#### Watch for changes

```graphql
subscription WatchNotes($filter: NoteChangesFilter!) {
  noteChanges(filter: $filter) {
    changes {
      __typename
      ... on NoteUpsertEvent {
        path
        eventType
        versionId
      }
      ... on NoteHideEvent {
        path
      }
    }
  }
}
```

```json
{ "filter": { "includePatterns": ["boards/**"] } }
```

`includePatterns` (required) and `excludePatterns` are globs: `*` stays within a folder, `**` crosses folders, and a plain path matches one note. One event can carry several changes when one save wrote several notes.

The transport is server-sent events, not WebSocket: a `POST` to `/_system/graphql` with `Accept: text/event-stream`. `EventSource` can't send a `POST`, so read the response stream:

```js
async function watch(query, variables, onData) {
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    },
    body: JSON.stringify({ query, variables }),
  })
  const reader = res.body
    .pipeThrough(new TextDecoderStream())
    .getReader()
  let buffer = ''
  for (;;) {
    const { done, value } = await reader.read()
    if (done) return
    buffer += value
    const frames = buffer.split('\n\n')
    buffer = frames.pop()
    for (const frame of frames) {
      const data = frame.match(/^data: (.*)$/m)
      if (frame.startsWith('event: next') && data) {
        onData(JSON.parse(data[1]).data.noteChanges)
      }
    }
  }
}
```

The server sends `event: next` with a result, `event: complete` at the end, and a `: ping` comment every 30 seconds. The stream can end, for example when the server restarts, so call `watch` again after a pause. Kanban's [src/api.ts](https://github.com/trip2g/kanban_template/blob/main/src/api.ts) does this with a backoff.

#### Read an older version (admin)

Kanban rebuilds a move on top of someone else's edit with these two. Versions come newest first, so `limit: 1` gives the latest.

```graphql
query NoteVersions($filter: AdminNoteVersionHistoryFilter!) {
  admin {
    noteVersionHistory(filter: $filter) {
      nodes {
        versionId
        version
        createdAt
      }
    }
  }
}
```

```json
{ "filter": { "path": "boards/todo.md", "limit": 1 } }
```

```graphql
query NoteVersion($id: Int64!) {
  admin {
    noteVersion(versionId: $id) {
      path
      content
      createdAt
    }
  }
}
```

```json
{ "id": 42 }
```

### Where to find more

The operations above are tested against the schema. For anything else:

- **The schema.** [internal/graph/schema.graphqls](https://github.com/trip2g/trip2g/blob/main/internal/graph/schema.graphqls) lists every query, mutation and subscription with its input and result types. Signed in as admin, open `/_system/graphql` in a browser to explore it in GraphiQL.
- **The Obsidian sync client.** Its operations file, [src/operations.graphql](https://github.com/trip2g/obsidian-sync/blob/master/src/operations.graphql) in [github.com/trip2g/obsidian-sync](https://github.com/trip2g/obsidian-sync), holds the API-key operations it runs: reading notes and their assets, `pushNotes`, `hideNotes`, `uploadNoteAsset`, `commitNotes`. The trip2g repository carries the client as the `obsidian-sync` submodule; GitHub doesn't show a submodule's files under trip2g, so here is [the same file at the commit trip2g pins](https://github.com/trip2g/obsidian-sync/blob/05b385ae943a5f6b5f162c6b3b2ed014d51ee66c/src/operations.graphql).
- **The admin panel.** Every screen keeps its operations in `.graphql` files beside its code, under [assets/ui/](https://github.com/trip2g/trip2g/tree/main/assets/ui): `admin/` for the admin panel, `editor/` for the note editor, `user/` for the reader's side. Find the one you need by the field it calls, with [GitHub code search over those files](https://github.com/search?q=repo%3Atrip2g%2Ftrip2g+path%3Aassets%2Fui+extension%3Agraphql&type=code) (add the field name to the query) or in a clone:

  ```bash
  grep -rl --include='*.graphql' 'noteVersionHistory' assets/ui
  ```

  Anything under `admin { … }` needs a signed-in admin: the session cookie or a personal token, not an API key. That suits an app only admins open, such as an internal dashboard or an editor.

### Build your own

1. Decide the markdown format first. The note must stay readable and editable without your app: a list, a table, headings. Kanban reuses the obsidian-kanban plugin's format, so the same note is a board in Obsidian too.
2. Write a parser and a serializer that round-trip the format exactly, and test that `serialize(parse(text)) === text`. Anything the app doesn't model must survive a save unchanged.
3. Write the layout above, with your own mount element and bundle.
4. Read the data from the page, render it, and save with `updateNotes` and `expectedHash`.
5. Show edit controls only when `editable` is true. The server enforces the rule anyway.

### See also

- [[en/user/kanban|Kanban board template]] and its source, [github.com/trip2g/kanban_template](https://github.com/trip2g/kanban_template)
- [[en/user/update_notes|updateNotes]] — the write API in full
- [[en/user/graphql|GraphQL API]]
- [[en/user/jet-functions#json and writeJson|json() and writeJson()]]
- [[en/user/templates|Templates]]
