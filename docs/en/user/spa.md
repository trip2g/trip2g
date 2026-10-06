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
| `UpdateNotesPatchNotFoundPayload` | A `patch` change didn't find its `find` text |
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
