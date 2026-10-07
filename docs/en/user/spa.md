---
title: An app on top of trip2g
free: true
lang_redirect: "[[ru/user/spa]]"
---

A custom layout doesn't have to render a page. It can ship a JavaScript app that shows a note as something interactive and writes the changes back. The note stays plain markdown: Obsidian, an agent and the app all edit the same file.

Think of it as an interactive view app: a note rendered as an app, with markdown files as its database. One note is the app's home, and the app can read and write many more.

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

- `title` is a variable every custom layout gets: the note's title run through the site's title template, ready for `<title>`. With the default template, `%s`, it is the title itself. For the bare title elsewhere on the page, use `note.Title()`.
- `note.ContentString()` is the note's markdown source, frontmatter included. It is the same text the server hashes for conflict checks, so the app can edit it and send it back.
- `json()` escapes `<`, `>` and `&` as `<`…, so markdown that contains `</script>` can't close the element. `JSON.parse` gives back the exact text.
- `currentUser.IsAdmin()` only decides whether the app shows its edit controls. The server checks every save again (see below).
- Calling `currentUser` makes the page personalized: it is not served from the anonymous page cache, so the app always starts from the current version of the note.

The kanban layout uses a different carrier for the same data: the markdown goes into a hidden `<textarea>` and the path into a hidden `<span>`, both printed with plain `{{ … }}`. That is safe too, because output is escaped by default: a note containing `</textarea>` reaches the page as `&lt;/textarea&gt;`, and the browser decodes it back. A `<textarea>` has two quirks, though: the browser normalizes its line endings to `\n` and drops a newline that comes right after the opening tag. The JSON island has neither.

### Configure the app from the frontmatter {#config}

The note's frontmatter is a handy place for the app's settings. Whoever edits the note, in Obsidian or by an agent, changes them along with the content, and one bundle serves many notes, each with its own settings:

```yaml
---
title: Todo
layout: app
columns: [Todo, Doing, Done]
wip_limit: 3
show_archive: false
---
```

The layout reads the keys with `note.M()` and adds them to the data block as `config`:

```jet
<script type="application/json" id="app-data">
  {{ json(map(
    "path", note.Path(),
    "content", note.ContentString(),
    "editable", currentUser.IsAdmin(),
    "config", map(
      "columns", note.M().GetStrings("columns"),
      "wipLimit", note.M().GetInt("wip_limit", 0),
      "showArchive", note.M().GetBool("show_archive", true)
    )
  )) }}
</script>
```

The app finds them in `data.config`:

```js
const { columns, wipLimit, showArchive } = data.config
```

Each getter returns its default when the key is missing or holds the wrong type, so a note without settings still opens; `GetStrings` returns an empty list. To hand the app the whole frontmatter instead, use `"config", note.M().Raw()`. The getters are in [[en/user/templates#What's available in a custom template|Templates]]. The settings are also part of `content`, so a save that keeps the frontmatter keeps them.

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
| `defaultTemplate.UserSpaceScripts()` | A `<script>` with the bundle's settings, then the user-space bundle. The bundle draws the sign-in button and the search box in the header. The settings come only on the first call |
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

### Saving: the `updateNotes` mutation {#save}

The app writes the note back with the `updateNotes` GraphQL mutation at `/_system/graphql`. A signed-in site admin is allowed to write; anyone else gets an error, so a visitor's browser can't change the note even if the app shows its controls by mistake.

Every operation is the same `POST` with a JSON body `{ query, variables }`, so one small helper covers them all. `makeRequest` takes an operation and returns a function of its variables:

```js
const makeRequest = (query) => (variables) =>
  fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
    .then((res) => res.json())
    .then((body) => {
      if (body.errors) throw new Error(body.errors[0].message)
      return body.data
    })

const updateNotes = makeRequest(`mutation ($input: UpdateNotesInput!) {
  updateNotes(input: $input) {
    __typename
    ... on UpdateNotesSuccessPayload { updated { path versionId } }
    ... on UpdateNotesHashMismatchPayload { path actualHash }
    ... on UpdateNotesPatchNotFoundPayload { path find }
    ... on ErrorPayload { message }
  }
}`)

const change = { upsert: { path: data.path, content: newContent, expectedHash } }
const { updateNotes: result } = await updateNotes({ input: { changes: [change] } })
```

`newContent` is the edited markdown, and `expectedHash` is explained below. The request is same-origin, so the browser attaches the session cookie; `credentials: 'include'` only says so explicitly. `__typename` says how the save went:

| `__typename` | Meaning |
|---|---|
| `UpdateNotesSuccessPayload` | Saved; `updated` lists each note with the `versionId` the save wrote |
| `UpdateNotesHashMismatchPayload` | Nothing was written: the note no longer matches `expectedHash`; `actualHash` is its current hash |
| `UpdateNotesPatchNotFoundPayload` | A `patch` change didn't find its `find` text, or found it more than once |
| `ErrorPayload` | Refused, with a `message` |

A visitor who isn't the admin doesn't reach a payload at all: the response carries an `errors` entry and no `data`, which `makeRequest` turns into an exception.

#### `expectedHash`: a save applies once, to the text it was made from

`expectedHash` is a field of the `upsert` and `patch` changes of `updateNotes`. It ties a change to the version of the note it was made from. Before writing, the server hashes the note's current markdown and compares:

- **It matches:** the change is written.
- **It doesn't:** nothing is written, and the result is `UpdateNotesHashMismatchPayload` with the note's `path` and its current `actualHash`.

That makes a change apply at most once. The same save sent twice, by a retry after a timeout, a double click or a second open tab, carries the same `expectedHash`; the first one changes the note, so the second no longer matches and is refused instead of applied again. The same check keeps the app from overwriting an edit someone made in Obsidian or by an agent in between.

What to pass is the hash of the markdown the app started from: `latestContentHash` from the `ReadNote` query below, or the hash of `data.content` the layout shipped. The hash is SHA-256 of the markdown, in URL-safe base64 with padding:

```js
async function contentHash(text) {
  const bytes = new TextEncoder().encode(text)
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  return btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
}

const expectedHash = await contentHash(data.content)
```

A successful save doesn't return the new hash. The note now holds exactly the markdown an `upsert` sent, so the next save's `expectedHash` is `contentHash(newContent)`; after a `patch`, read `latestContentHash` again. Two values are special: an empty `expectedHash` means "create only" and fails on a note that exists, and a change without `expectedHash` is written unconditionally.

On `UpdateNotesHashMismatchPayload` reload the page, or fetch the new content and apply your change to it. Kanban re-reads the latest version through the admin `noteVersionHistory` and `noteVersion` queries and replays the move.

**Small edits.** Instead of `upsert`, which replaces the whole note, a change can be a `patch`: `{ patch: { path, find, replace, expectedHash } }` replaces one exact piece of text. Kanban toggles a checkbox this way, so text it doesn't model is never rewritten. Both forms, batches and error cases are in [[en/user/update_notes|updateNotes]].

### Live updates {#live}

To pick up edits made in Obsidian or by an agent while the app is open, subscribe to `noteChanges` with a filter on the note's path. The endpoint serves subscriptions over server-sent events only, on the same `/_system/graphql`: a `POST` with `Accept: text/event-stream`. It has no WebSocket transport, so a `graphql-ws` client won't connect. `EventSource` can't send a `POST`, so the helper reads the response stream itself:

```js
const makeSubscription = (query) => async (variables, onData) => {
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
      const line = frame.match(/^data: (.*)$/m)
      if (!frame.startsWith('event: next') || !line) continue
      const body = JSON.parse(line[1])
      if (body.errors) throw new Error(body.errors[0].message)
      onData(body.data)
    }
  }
}

const watchNotes = makeSubscription(`subscription ($filter: NoteChangesFilter!) {
  noteChanges(filter: $filter) {
    changes {
      __typename
      ... on NoteUpsertEvent { path versionId }
      ... on NoteHideEvent { path }
    }
  }
}`)

async function follow(path, onChange) {
  for (;;) {
    await watchNotes({ filter: { includePatterns: [path] } }, onChange)
      .catch(() => {})
    await new Promise((resolve) => setTimeout(resolve, 5000))
  }
}

const ownVersions = new Set()

follow(data.path, ({ noteChanges }) => {
  const foreign = noteChanges.changes
    .some((change) => !ownVersions.has(change.versionId))
  if (foreign) location.reload()
})
```

The server sends `event: next` with a result, `event: complete` at the end, and a `: ping` comment every 30 seconds. The stream can end, for example when the server restarts, so `follow` subscribes again after a pause.

Your own save comes back as an event too. Add each `versionId` from the save's `updated` to `ownVersions`, and the app skips its own echo. `location.reload()` is the simplest reaction to someone else's edit. To keep the app's state, re-read the note with `ReadNote` and merge instead, as Kanban's [src/api.ts](https://github.com/trip2g/kanban_template/blob/main/src/api.ts) does. The subscription needs a signed-in visitor: the admin gets every matching note, another signed-in reader only the notes they may read. The filter's globs are explained under [[en/user/spa#Watch for changes|Watch for changes]].

### Ready-made GraphQL operations

Copy these into your app. Each one goes to `/_system/graphql` as a `POST` with a JSON body `{ query, variables }`: the operation is `query`, the JSON block under it is `variables`. Wrap an operation with `makeRequest` from [[en/user/spa#save|Saving]] and call it with the variables. Here `READ_NOTE` holds the `ReadNote` operation below:

```js
const readNote = makeRequest(READ_NOTE)
const { notePaths } = await readNote({ paths: [data.path] })
```

Most results are unions: ask for `__typename` and one `... on` fragment per type, then branch on `__typename`. A refusal the app should show comes back as a payload type such as `ErrorPayload`; a caller with no access gets an `errors` entry instead, which `makeRequest` turns into an exception.

**Who may call what.** The details are in [[en/user/update_notes#Authentication|updateNotes → Authentication]].

| Caller | Sends | Reaches |
|---|---|---|
| Signed-in admin | Session cookie, or `Authorization: Bearer t2g_…` | Everything below, `admin { … }` included |
| API key | `X-Api-Key: …` | Everything but `admin { … }`; writes only within the key's write patterns |
| Webhook token | `Authorization: Bearer eyJ…` | What an API key reaches, within the token's read and write patterns, except `hideNotes`, `pushNotes` and `commitNotes` |
| Any visitor | Nothing | `search`, `submitForm`, and `noteChanges` once signed in |

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

The client that runs this subscription, with reconnects, is in [[en/user/spa#live|Live updates]].

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

### API calls and admin calls {#api-vs-admin}

The endpoint carries two kinds of operations, and they differ in what they promise:

| | API calls | Admin calls |
|---|---|---|
| Where they sit | At the top level of the operation: `notePaths`, `search`, `updateNotes`, `noteChanges` | Inside `admin { … }`, the `AdminQuery` and `AdminMutation` types in the schema |
| Who relies on them | The Obsidian plugin, agents with an API key, apps like this one | trip2g's own admin panel |
| Stability | The surface those clients are built on | Fewer guarantees: they change with the admin panel and may change in any release |
| Who may call | See the table in [[en/user/spa#Ready-made GraphQL operations\|Ready-made GraphQL operations]] | A signed-in admin: the session cookie or a personal token |

An app may use admin calls; Kanban reads version history with them. Keep them to features only admins see, and check them again after you upgrade trip2g. Everything else on this page, apart from the section marked "(admin)", is API.

The API surface is documented in [[en/user/update_notes|updateNotes]], [[en/user/graphql|GraphQL API]] and the operations above, and the Obsidian plugin's operations file below lists what it runs.

### An app over the whole knowledge base: MCP from the browser {#mcp}

An app doesn't have to be about the note it is on. [MCP Graph Walk](https://trip2g.com/search_visualizer) shows a model searching and reading this documentation site live, every call a step on an animated map. Its page is one note, `docs/search_visualizer.md` with `layout: search_visualizer`, and the layout, [docs/_layouts/search_visualizer.html](https://github.com/trip2g/trip2g/blob/main/docs/_layouts/search_visualizer.html), is the whole app. It:

- calls `/_system/mcp` from the browser with JSON-RPC `tools/call`: `search`, `note_html`, `expand`, `similar` and their federated versions;
- reads the knowledge graph of the whole site and writes nothing;
- runs full-screen with its own header, without `defaultTemplate.Header()`;
- talks to a model with an API key the visitor pastes. The page keeps the key in the browser's `localStorage` and sends it only to the model provider; trip2g never sees it.

| | The note is | The app talks to |
|---|---|---|
| Kanban | The data: the app edits the note it is on | GraphQL: `updateNotes`, `noteChanges` |
| Graph walk | The app's home: the data is the whole site | MCP: `search`, `note_html` |

**MCP or GraphQL.** Use MCP to search and read across the base: search results with snippets and a `match_id` per hit, one section read by its `toc_path`, a note's table of contents walked with `expand`, related notes with `similar`, and search across connected bases. It is the same tool surface agents use, so a page that shows what an agent would find is built on it. The tools are described in [[en/user/mcp|MCP Server]]. Use GraphQL for everything else: writes (`updateNotes`), live updates (`noteChanges`), typed queries that return exactly the fields you ask for, and `viewer`.

**A call.** The endpoint is stateless, so a `tools/call` works without an `initialize` first. Send `initialize` only to read the site's instructions for agents, and `tools/list` to get each tool's input schema; the graph walk sends both because it hands the schemas to the model.

```js
let rpcId = 0

async function callTool(name, args) {
  const res = await fetch('/_system/mcp', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: ++rpcId,
      method: 'tools/call',
      params: { name, arguments: args },
    }),
  })
  const body = await res.json()
  if (body.error) throw new Error(body.error.message)
  if (body.result.isError) throw new Error(body.result.content[0].text)
  return body.result
}

const found = await callTool('search', { query: 'release plan', limit: 5 })
const hits = found.structuredContent.results
const section = await callTool('note_html', { path: hits[0].note_path })
const html = section.content.map((part) => part.text).join('\n')
```

Each hit in `structuredContent.results` has `title`, `note_path`, `url` and `matches`; `content` holds the same answer as text, and for `note_html` that text is the note's HTML. The request must be a `POST` with `Content-Type: application/json`; anything else gets HTTP 415. Leave `Accept` at the browser's default: `Accept: application/json` alone is refused with HTTP 400, because the endpoint wants `application/json, text/event-stream` or `*/*`.

**Access.** MCP reads the same session cookie as GraphQL. The request goes to the same origin, the browser attaches the cookie, and the server checks it before it runs a tool:

- **A visitor who isn't signed in** finds and reads only the notes open to everyone. Search leaves the rest out entirely, unlike GraphQL `search`, which lists them last without a `document`.
- **A signed-in reader** also reaches the notes of the subgraphs they have access to: the same notes they could open on the site.
- **A signed-in admin** reaches every note.
- **A note the visitor can't read** answers `note_html` with the error `Note not found`, the same as a note that doesn't exist.
- **Nothing the browser sends unlocks a write.** The tools that run GraphQL as admin, `graphql_request` and `graphql_introspection`, need an API key with MCP admin tools turned on; a session cookie, even the admin's, doesn't reach them. From a page, MCP is read-only.

Call it from a page on the same site. The server sends CORS headers only to the Obsidian plugin, so a page on another origin can't read the answers.

### Your own API next to trip2g, with trip2g as the access check {#own-api}

An app may need more than notes: a report built from another database, a call to a paid service, a job that runs for a minute. Put that in an API of your own and let trip2g decide who may call it. The API asks trip2g on the user's behalf, so it keeps no users, no passwords and no access rules of its own.

**Mount it on the same host.** The session cookie, `trip2g_token`, is `HttpOnly`, `Secure`, `SameSite=Lax` with `Path=/`, and has no `Domain` attribute. Without `Domain` the browser sends it only to the exact host that set it: not to a subdomain, not to a parent domain. So the API lives under a path on the site's own host, not on `api.example.com`.

**Under `/_system/extra/`.** This prefix is reserved for the site owner: trip2g serves nothing under it and never will, and a test in trip2g's router fails if one of its routes starts with it. Mount each API at `/_system/extra/<name>`, such as `https://notes.example.com/_system/extra/report`, and route `/_system/extra/` to your APIs in the reverse proxy in front of trip2g; everything else keeps going to trip2g. Two nearby choices are not safe:

- **`/api/…`** is also a note path: a note `api/report.md` is served at `/api/report`, and the proxy rule would hide it.
- **`/_system/<name>`** is where trip2g's own routes live: `/_system/graphql`, `/_system/mcp`, `/_system/admin`, `/_system/auth/…` and more. A later release may add one with your name.

No note is ever served under `/_system/`, so the prefix can't hide content either. A request to `/_system/extra/…` that reaches trip2g, because no proxy rule took it, gets the site's "Page not found" page with HTTP 404, for `GET` and `POST` alike.

**The flow.**

1. The app calls `fetch('/_system/extra/report', …)`. The request is same-origin, so the browser attaches the cookie.
2. The API takes the request's `Cookie` header and sends it unchanged to trip2g's `/_system/graphql` over the internal network, with a query that asks who this is.
3. trip2g answers as it would answer the browser. The API decides from that answer, and refuses whenever trip2g refused.

The layout gives the app the note's id, adding `"pathId", note.PathID()` to the data block from [[en/user/spa#The layout: ship the note in the page|the layout]], and the app sends it:

```js
const res = await fetch('/_system/extra/report', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ pathId: data.pathId }),
})
if (!res.ok) throw new Error(`report: ${res.status}`)
const report = await res.json()
```

The API, here in Node 18 or later, asks trip2g who the caller is:

```js
import { createServer } from 'node:http'

const askTrip2g = (query) => async (cookie, variables) => {
  const res = await fetch(process.env.TRIP2G_GRAPHQL, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    body: JSON.stringify({ query, variables }),
  })
  const body = await res.json()
  return body.errors ? null : body.data
}

const whoIs = askTrip2g(`query { viewer { role user { email } } }`)

createServer(async (req, res) => {
  if (req.method !== 'POST') return res.writeHead(405).end()
  const cookie = req.headers.cookie ?? ''
  const who = await whoIs(cookie)
  if (!who || who.viewer.role === 'GUEST') return res.writeHead(401).end()
  let raw = ''
  for await (const chunk of req) raw += chunk
  const { pathId } = JSON.parse(raw)
  res.writeHead(200, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(await buildReport(pathId, who.viewer.user)))
}).listen(8080)
```

`TRIP2G_GRAPHQL` is trip2g's GraphQL endpoint at its internal address, and `buildReport` is the API's own work. `viewer.role` is `GUEST`, `USER` or `ADMIN`. A visitor who isn't signed in, or whose cookie has expired, gets `GUEST` and `user: null`; a banned user gets an `errors` entry, which `askTrip2g` turns into `null`. The schema has no user id: `user.email` is what identifies the person, and it is `null` for an account signed in without an email.

**Stronger: check the resource, not just the person.** Before serving data about a note, ask trip2g for that note on the user's behalf. The public `note` query answers only when the user may open the note on the site; otherwise it returns an `errors` entry (`Need auth`, `Need subscription`, `Sign in required`, `page not found`) and no note. Refuse whenever trip2g did:

```js
const canRead = askTrip2g(`query ($pathId: Int64!) {
  note(input: { pathId: $pathId, referer: "" }) { pathId }
}`)

const allowed = await canRead(cookie, { pathId })
if (!allowed?.note) return res.writeHead(403).end()
```

Now the access rules live in one place, the vault's subgraphs and paywalls, and the API can't drift from them. Serve exactly the `pathId` you checked. A successful check by a signed-in reader counts as a view of the note in their reading history, as opening the page would.

**Rules.**

- **The forwarded cookie is the user's whole session.** It works for every trip2g call that user could make, writes included for an admin. Send it only to trip2g; never log it, store it or pass it to another service. Signing out only deletes the cookie from the browser: a copy someone kept stays valid until it expires, 30 days by default.
- **Refuse `GET` on anything that changes state.** `SameSite=Lax` keeps the cookie off a cross-site `POST`, but the browser still sends it with a top-level `GET` navigation from another site. An endpoint that only reads may answer `GET`.
- **Every API call costs a trip2g call.** You can cache trip2g's answer for a few seconds, keyed by the cookie value. Then a ban or a revoked subscription reaches the API only when the cached answer expires.
- **Call trip2g at its internal address,** not the public domain. The cookie then doesn't cross the internet a second time, and the call doesn't go back through the proxy.

trip2g asks nothing more of a call from another process: no `Origin`, no `Referer`, no CSRF token or custom header. It reads the cookie from the `Cookie` header of any `POST` with `Content-Type: application/json`, the same as from a browser. The `Secure` flag only tells browsers not to send the cookie over plain HTTP; the server accepts it either way, so a plain-HTTP call on the internal network works.

### Accept data without a backend: forms {#forms}

An app that collects something, such as an order, a sign-up or feedback, needs no server of its own to keep it. A trip2g form already stores the submissions, decides who may send one and keeps spam out. Declare the form in the note's frontmatter as [[en/user/forms|Forms in notes]] describes, and the app sends it with the `submitForm` mutation:

- **Fields from the definition.** The layout prints `note.FormSpecJSON()` into a `<script type="application/json" id="form-spec">`, and the app builds its inputs from it: names, types, `required`, limits. It also carries `note_version_id`, which the submit sends as `noteVersionId`. See [[en/user/forms#Custom layout|Forms → Custom layout]].
- **The submit** goes to `/_system/graphql` like every other call on this page. The fields and every result are in [[en/user/forms#Submitting via GraphQL|Forms → Submitting via GraphQL]].
- **Spam protection.** Cloudflare Turnstile is on for every form unless it sets `turnstile: false`. Without a valid token the result is `TurnstileRequiredPayload` with the site's `siteKey`: the app shows the widget and sends the same input again with the token.
- **Submissions** are listed in the admin panel under Forms, and each one sends the site's admins an email. An admin reads them over GraphQL with `admin { formSubmits … }`, an admin call: [[en/user/forms#Reading submissions|Forms → Reading submissions]].

```js
const submitForm = makeRequest(`mutation ($input: SubmitFormInput!) {
  submitForm(input: $input) {
    __typename
    ... on SubmitFormPayload { submitId }
    ... on FormSubmitDeniedPayload { reason }
    ... on TurnstileRequiredPayload { siteKey }
    ... on ErrorPayload { message }
  }
}`)

const spec = JSON.parse(document.getElementById('form-spec').textContent)

async function send(fields, turnstileToken) {
  const input = { noteVersionId: spec.note_version_id, formId: '', fields, turnstileToken }
  const { submitForm: result } = await submitForm({ input })
  if (result.__typename === 'TurnstileRequiredPayload') {
    turnstile.render('#captcha', {
      sitekey: result.siteKey,
      callback: (token) => send(fields, token),
    })
  }
  return result
}

await send([{ name: 'email', stringValue: 'alice@example.com' }])
```

`turnstile` comes from Cloudflare's script, `<script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script>`, in the layout, and `#captcha` is an empty element for the widget. `formId` is `""` for a single `form:`, or the key of one form under `forms:`.

**Who may submit.** The server checks two things, in this order:

1. **May the visitor read the note?** A note the visitor can't open answers `ErrorPayload` with `form_not_found`, as if it had no form.
2. **`can_submit`.** Omitted or `guest`: anyone who passed the first check, signed in or not. `admin`: only a signed-in admin; anyone else gets `FormSubmitDeniedPayload` with `reason: "admin_required"`. `paid_user` isn't enforced yet and refuses everyone, the admin included, with `reason: "not_implemented"`. A value trip2g doesn't know, such as `user`, counts as `guest` and opens the form to everyone.

**What that means for members.** No `can_submit` value means "signed-in members only". The note's own access is what narrows a form to them: put the form on a note without `free: true` in a subgraph marked Require sign-in, and any signed-in user may submit while a guest gets `form_not_found`; in a paid subgraph, only those with access may. A form on a note open to everyone takes submissions from everyone. So a shop with member accounts can't make a form members-only with `can_submit` yet; it has to rely on the note's access, see [[en/user/subgraphs|Subgraphs]]. When the submitter is signed in, the submission keeps their account, readable as `user` in `formSubmits`.

### For example: a small shop {#shop-example}

How the pieces on this page could fit together for a shop. This is an illustration, not a shop feature: trip2g has no catalogue, cart or checkout of its own.

- **Products are notes,** one per product, with the price and stock in the frontmatter: `price: 1200`, `stock: 5`.
- **The storefront is a layout** that lists them with `nvs.ByGlob("shop/*.md").Public()` and reads each one's `M().GetInt("price", 0)`; see [[en/user/jet-functions|Jet functions]].
- **The cart lives in the browser,** in `localStorage`, or in an API of your own at `/_system/extra/cart` that asks trip2g's `viewer` who the shopper is, as in [[en/user/spa#own-api|Your own API]].
- **The order is a trip2g form,** not your API: a checkout note declares a `form:` with the buyer's contacts and a text field for the cart, and the app submits it as in [[en/user/spa#forms|Forms]]. It arrives in the admin panel and the admins' mail.

Nothing here changes the stock or checks the price: the order holds whatever the browser sent, and the stock goes down when someone edits the product note.

### Where to find more

The operations above are tested against the schema. For anything else:

- **The schema.** [internal/graph/schema.graphqls](https://github.com/trip2g/trip2g/blob/main/internal/graph/schema.graphqls) lists every query, mutation and subscription with its input and result types. Signed in as admin, open `/_system/graphql` in a browser to explore it in GraphiQL.
- **The Obsidian sync client.** Its operations file, [src/operations.graphql](https://github.com/trip2g/obsidian-sync/blob/master/src/operations.graphql) in [github.com/trip2g/obsidian-sync](https://github.com/trip2g/obsidian-sync), holds the API-key operations it runs: reading notes and their assets, `pushNotes`, `hideNotes`, `uploadNoteAsset`, `commitNotes`. The trip2g repository carries the client as the `obsidian-sync` submodule; GitHub doesn't show a submodule's files under trip2g, so here is [the same file at the commit trip2g pins](https://github.com/trip2g/obsidian-sync/blob/05b385ae943a5f6b5f162c6b3b2ed014d51ee66c/src/operations.graphql).
- **The admin panel.** Every screen keeps its operations in `.graphql` files beside its code, under [assets/ui/](https://github.com/trip2g/trip2g/tree/main/assets/ui): `admin/` for the admin panel, `editor/` for the note editor, `user/` for the reader's side. Find the one you need by the field it calls, with [GitHub code search over those files](https://github.com/search?q=repo%3Atrip2g%2Ftrip2g+path%3Aassets%2Fui+extension%3Agraphql&type=code) (add the field name to the query) or in a clone:

  ```bash
  grep -rl --include='*.graphql' 'noteVersionHistory' assets/ui
  ```

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
- [ ] Edit controls show only when the layout's `currentUser.IsAdmin()` says so; a visitor sees the note read-only and the header's sign-in button.
- [ ] Every save passes `expectedHash` and handles `UpdateNotesHashMismatchPayload` and an `errors` entry.
- [ ] MCP calls go to `/_system/mcp` by a relative URL, as `POST` with `Content-Type: application/json` and the browser's default `Accept`, and handle both a JSON-RPC `error` and a result with `isError`.
- [ ] An API of your own sits under `/_system/extra/` on the site's host, not on a subdomain, `/api/` or another `/_system/` path; it sends the `Cookie` header only to trip2g at its internal address, never logs or stores it, and refuses `GET` on endpoints that change anything.
- [ ] Data the app collects goes through a trip2g form: the app handles `TurnstileRequiredPayload`, `FormSubmitDeniedPayload` and `ErrorPayload`, and a form meant for members sits on a note only they can read.

### See also

- [[en/user/kanban|Kanban board template]] and its source, [github.com/trip2g/kanban_template](https://github.com/trip2g/kanban_template)
- [[en/user/update_notes|updateNotes]] — the write API in full
- [[en/user/graphql|GraphQL API]]
- [[en/user/mcp|MCP Server]] — the tools `/_system/mcp` offers
- [[en/user/forms|Forms in notes]] — form fields, `can_submit`, `turnstile` and reading submissions
- [[en/user/subgraphs|Subgraphs]] — who may read a note, and so who may submit its form
- [MCP Graph Walk](https://trip2g.com/search_visualizer) and its layout, [docs/_layouts/search_visualizer.html](https://github.com/trip2g/trip2g/blob/main/docs/_layouts/search_visualizer.html)
- [[en/user/jet-functions#json and writeJson|json() and writeJson()]]
- [[en/user/templates|Templates]]
