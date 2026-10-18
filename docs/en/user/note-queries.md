---
title: Note queries
free: true
lang: en
lang_redirect: "[[ru/user/note-queries]]"
---

A page built with JavaScript often needs a list of notes: posts with a tag, the latest five articles, tasks grouped by status. trip2g keeps that data in the notes' frontmatter, but doesn't hand raw frontmatter to visitors. A note query is the bridge: you write a small [[en/user/jsonnet|Jsonnet]] program in a note, and the page calls it by name and gets back exactly the JSON the program builds.

The code lives in your vault, so you decide what leaves the server. A visitor only passes parameters, such as a tag.

### Quick start

1\. Create a note `queries/by_tag.md` with `type: note-query` and one `jsonnet` block:

````
---
type: note-query
free: true
---

Blog posts with a tag, newest first.

```jsonnet
local tagged = [
  n for n in notes
  if std.startsWith(n.path, 'blog/')
  && std.member(std.get(n.meta, 'tags', []), params.tag)
];
local byDate = std.sort(tagged, function(n) std.get(n.meta, 'date', ''));

[
  { title: n.title, url: n.permalink, date: n.meta.date }
  for n in std.reverse(byDate)
]
```
````

2\. Sync the vault, then call the query from your page. `makeRequest` is the small `fetch` helper from [[en/user/spa#save|An app on top of trip2g]]:

```js
const noteQuery = makeRequest(`query ($input: NoteQueryInput!) {
  noteQuery(input: $input) {
    __typename
    ... on NoteQueryPayload { json }
    ... on ErrorPayload { message }
  }
}`)

const data = await noteQuery({
  input: { query: '[[by_tag]]', params: JSON.stringify({ tag: 'go' }) },
})
const posts = JSON.parse(data.noteQuery.json)
// [{ title: 'Go on the web', url: '/blog/go_on_the_web', date: '2026-10-10' }, …]
```

`query` names the query note: a path from the vault root such as `queries/by_tag.md`, or a wikilink such as `[[by_tag]]`. A bare name is found anywhere in the vault; when several notes share it, the one with the shortest path wins. `params` is a JSON string, up to 16 KB; without it the query gets `{}`. The result comes back as a JSON string in `json`.

### What the query sees

| Name | What it holds |
|---|---|
| `notes` | a list of notes, sorted by path: `{ path, title, permalink, meta }` |
| `params` | the parsed `params`, `{}` when none were passed |

`meta` is the note's frontmatter after [[en/user/frontmatter-patches|frontmatter patches]]. System notes (paths starting with `_`) and files that aren't markdown notes (canvas, base, excalidraw) are left out.

The program can return any JSON value: a list, an object, a number.

### Who can run it, and what it sees

Two rules keep a query from leaking anything:

1. **A visitor can run a query only if they can read the query note.** Set its access like any other note: `free: true` for everyone, a [[en/user/subgraphs|subgraph]] for subscribers. A query the visitor can't read answers `query not found`, the same as one that doesn't exist.
2. **The program sees only notes the visitor can read.** A guest running a public query never gets paid notes in `notes`, so the query can't show their titles or fields by mistake. An admin running the same query sees everything.

With the examples below, a guest asking `by_tag` for `go` gets two posts; an admin gets a third, members-only one.

The query note is published like any other note, so readers who can run the query can also read its code.

### Examples

These queries come from a demo vault that the trip2g test suite runs on every build.

#### Posts with a tag, newest first

Params: `{"tag": "go"}`.

```jsonnet
local tagged = [
  n for n in notes
  if std.startsWith(n.path, 'blog/')
  && std.member(std.get(n.meta, 'tags', []), params.tag)
];
local byDate = std.sort(tagged, function(n) std.get(n.meta, 'date', ''));

[
  { title: n.title, url: n.permalink, date: n.meta.date }
  for n in std.reverse(byDate)
]
```

#### The latest posts

Params: `{"limit": 2}`; without them, three posts.

```jsonnet
local limit = std.get(params, 'limit', 3);
local posts = [n for n in notes if std.startsWith(n.path, 'blog/')];
local byDate = std.sort(posts, function(n) std.get(n.meta, 'date', ''));

[n.title for n in std.reverse(byDate)][:limit]
```

A slice, `list[:limit]`, keeps the first items.

#### Counts for a dashboard

How many tasks are in each status. The result is an object, such as `{"done": 1, "todo": 2}`:

```jsonnet
local tasks = [n for n in notes if std.startsWith(n.path, 'tasks/')];
local status(n) = std.get(n.meta, 'status', 'none');

{
  [s]: std.length([n for n in tasks if status(n) == s])
  for s in std.set([status(n) for n in tasks])
}
```

`{ [key]: value for … }` builds an object with computed keys, and `std.set` removes duplicates and sorts.

### Errors

A failed call returns `ErrorPayload` with a message:

| Message | Why |
|---|---|
| `query not found` | no such note, or the visitor can't read it |
| `<path> is not a note-query` | the note has no `type: note-query` |
| `<path>: a note-query needs exactly one jsonnet block` | zero or several blocks |
| `params must be JSON`, `params is larger than 16 KB` | bad `params` |
| `<path>: RUNTIME ERROR: …` | the program failed; the first line of the Jsonnet error |

### Note queries or a template

A [[en/user/templates|custom template]] can also list notes: `nvs.ByGlob("blog/*.md")` and `SortByMeta` build the list on the server while the page renders, and `json()` hands it to the page. That list is fixed when the page renders. Use a note query when the page needs a different answer for each visitor input, like a tag picked from a menu.

### Limits

- **Every call runs the program again.** There is no cache yet; over a vault of about 850 notes a call takes around 12 ms. For a signed-in visitor who isn't an admin, checking which notes they can read costs a database lookup per note. Both are tracked in [issue #441](https://github.com/trip2g/trip2g/issues/441).
- **A query runs until it finishes.** Jsonnet can't be stopped halfway, so don't write loops whose size comes from `params`, such as `std.range(1, params.n)`: any visitor could make them huge.
- **Watching cost.** The instance's metrics have `trip2g_note_query_duration_seconds` per query note, and a call that takes a second or more is logged as `slow note query`.
