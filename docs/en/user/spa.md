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

### How your app is authorised: the session cookie {#auth}

The app has no login of its own. It borrows the visitor's trip2g session:

1. **The visitor signs in with the button in the standard header.** The layout puts trip2g's header on the page, see [[en/user/spa#chrome|Wrap your app in the standard site chrome]]. After a successful sign-in the page reloads.
2. **The server sets a session cookie.** It is `HttpOnly` (page JavaScript can't read it), `SameSite=Lax`, and belongs to the host the visitor signed in on.
3. **The app's own requests carry it.** A `fetch('/_system/graphql', …)` from the page goes to the same origin, and the browser attaches the cookie by itself. The server reads it and treats the request as that user's: a signed-in admin may read and write notes.

So the reloaded page is rendered for the signed-in user: `currentUser.IsAdmin()` is now `true` for the admin, and the app's `editable` flag with it.

**Do:**

- Call the API with a relative URL, `/_system/graphql`, from a page the site itself serves: the layout and its `asset()` files, or a bundle that layout loads.
- Leave `fetch` credentials at the default (`same-origin`) or set `credentials: 'include'`, as the helpers on this page do; for a same-origin request the two are the same. Never set `credentials: 'omit'`.
- Send operations as `POST` with `Content-Type: application/json`; a file upload goes as `multipart/form-data`. The endpoint runs no operation sent another way.
- Decide whether to show edit controls from `currentUser.IsAdmin()` in the layout, passed to the app as a flag such as `editable`.

**Don't:**

- **Don't put an API key or a personal token in the page.** Anyone who opens the page can read them. The cookie is the credential.
- **Don't build a login screen.** Signing in is the header's job; after it the page reloads with the session in place.
- **Don't call the trip2g UI's internal GraphQL client.** The user-space bundle that draws the sign-in button is a $mol app. It exposes no JavaScript API to other code, and reads its input from `window.__trip2g_settings`, which isn't an API either. Its client and names can change with any UI rebuild. Use your own `fetch`, such as the `gql()` helper in [[en/user/spa#Ready-made GraphQL operations|Ready-made GraphQL operations]].
- **Don't serve the app from another origin,** such as a separate dev server. The browser doesn't send the `SameSite=Lax` cookie with a cross-site `POST`, and the server allows cross-origin requests only from the Obsidian plugin.

**No CSRF token, no extra header.** The endpoint asks for nothing beyond the cookie: no CSRF token and no custom header. A form on another site can't use the session, because the browser leaves the `SameSite=Lax` cookie off a cross-site `POST`.

**A visitor who isn't signed in** reaches the page and the app normally. `currentUser.IsAdmin()` is `false`, so `editable` is `false`: render the note read-only and point to the header's sign-in button, for example with a "Sign in to edit" line. On the API, `search` still works; a write returns an `errors` entry. A reader who is signed in but isn't the admin gets the same refusal on writes. The table in [[en/user/spa#Ready-made GraphQL operations|Ready-made GraphQL operations]] lists who reaches what.

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


### Wrap your app in the standard site chrome {#chrome}

The layout above is a bare page. To give the app the site's header, with its sign-in button, and its footer, print the default template's parts around the mount element:

```jet
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ title }}</title>
  {{ defaultTemplate.Styles() }}
  {{ defaultTemplate.UserSpaceScripts() }}
</head>
<body>
  {{ defaultTemplate.Header() }}
  <main id="app"></main>
  <script type="application/json" id="app-data">
    {{ json(map(
      "path", note.Path(),
      "content", note.ContentString(),
      "editable", currentUser.IsAdmin()
    )) }}
  </script>
  <script src="{{ asset("app.js") }}"></script>
  {{ defaultTemplate.Footer() }}
</body>
</html>
```

| Call | Prints |
|---|---|
| `defaultTemplate.Styles()` | `<link>` tags for the default template's stylesheet, so the header and footer look as they do on other pages |
| `defaultTemplate.UserSpaceScripts()` | A `<script>` that sets `window.__trip2g_settings`, then the user-space bundle. The bundle draws the sign-in button and the search box in the header. The settings come only on the first call |
| `defaultTemplate.Header()` | The site header: logo, navigation, the sign-in button and the search box |
| `defaultTemplate.Footer()` | The site footer |

`Header()` and `Footer()` print nothing when the page has no header or footer note. The note comes from the page's `header:` / `footer:` frontmatter, a matching layout section, or a `_header` / `_footer` note in the vault: see [[en/user/default-template#Functional notes: header, footer, and sidebars|Functional notes]]. Without a header there is no sign-in button either. To keep one, put the bundle's mount element where you want the button, with `UserSpaceScripts()` still in `<head>`:

```html
<div mol_view_root="$trip2g_user_space"></div>
```

In the layout preview (`/_system/renderlayout`) all four calls print nothing. The full list of what a layout gets is in [[en/user/jet-functions#Variables in every custom layout|Jet functions → Variables in every custom layout]].

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
| Signed-in admin | Session cookie, or `Authorization: Bearer t2g_…` | Everything below, `admin { … }` included |
| API key | `X-Api-Key: …` | Everything but `admin { … }`; writes only within the key's write patterns |
| Webhook token | `Authorization: Bearer eyJ…` | What an API key reaches, within the token's read and write patterns, except `hideNotes`, `pushNotes` and `commitNotes` |
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

The hide is recorded against an admin: the signed-in admin, the admin who created the API key, or the admin who created the webhook that issued the token. A webhook token gets `ErrorPayload` from `hideNotes`; it hides with the `hide` change, within its write patterns.

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

### Checklist

- [ ] The note has `layout: app`, and the layout prints `defaultTemplate.Styles()` and `defaultTemplate.UserSpaceScripts()` in `<head>`, `Header()` before the app and `Footer()` after it.
- [ ] The page has a header note, or the layout has its own `$trip2g_user_space` mount element: otherwise there is no sign-in button.
- [ ] The app calls `/_system/graphql` by a relative URL, as `POST` with `Content-Type: application/json`, with default credentials or `credentials: 'include'`.
- [ ] No API key or token in the page, no login screen of the app's own.
- [ ] No calls into the trip2g UI's bundle or `window.__trip2g_settings`: the app has its own `fetch` helper.
- [ ] Edit controls show only when the layout's `currentUser.IsAdmin()` says so; a visitor sees the note read-only and the header's sign-in button.
- [ ] Every save passes `expectedHash` and handles `UpdateNotesHashMismatchPayload` and an `errors` entry.

### See also

- [[en/user/kanban|Kanban board template]] and its source, [github.com/trip2g/kanban_template](https://github.com/trip2g/kanban_template)
- [[en/user/update_notes|updateNotes]] — the write API in full
- [[en/user/graphql|GraphQL API]]
- [[en/user/jet-functions#json and writeJson|json() and writeJson()]]
- [[en/user/templates|Templates]]
