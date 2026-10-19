---
title: Note queries
free: true
lang: en
lang_redirect: "[[ru/user/note-queries]]"
---

A site often needs a selection of notes: posts with a tag, the five latest articles, tasks counted by status. A note query is that selection logic, written once as a small [[en/user/jsonnet|Jsonnet]] program in a note. A template calls it with `noteQuery()` while the page renders, and a JavaScript page calls it through the API. Both get exactly the JSON the program builds.

The data comes from the notes' frontmatter, but raw frontmatter never leaves the server: the code lives in your vault, so you decide what is returned. A visitor only passes parameters, such as a tag.

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
    ... on NoteQueryPayload { rawResult }
    ... on ErrorPayload { message }
  }
}`)

const data = await noteQuery({
  input: { query: '[[by_tag]]', params: { tag: 'go' } },
})
const posts = data.noteQuery.rawResult
// [{ title: 'Go on the web', url: '/blog/go_on_the_web', date: '2026-10-10' }, …]
```

`query` names the query note: a path from the vault root such as `queries/by_tag.md`, or a wikilink such as `[[by_tag]]`. A bare name is found anywhere in the vault; when several notes share it, the one with the shortest path wins. `params` is any JSON value, up to 16 KB once encoded; without it the query gets `{}`. The result comes back in `rawResult` as a JSON value, so there is nothing to parse.

### What the query sees

| Name | What it holds |
|---|---|
| `notes` | a list of notes, sorted by path: `{ path, title, permalink, meta }` |
| `params` | the parsed `params`, `{}` when none were passed |
| `currentUser` | who is asking: `{ id, role, isAdmin, signedIn }`; a guest has `null` for `id` and `role` |

`meta` is the note's frontmatter after [[en/user/frontmatter-patches|frontmatter patches]]. With `all_notes: true` each note also has `readable`. System notes (paths starting with `_`), notes hidden from [[en/user/search|search]] with `search: false`, and files that aren't markdown notes (canvas, base, excalidraw) are left out.

The program can return any JSON value: a list of notes, an object of counts, a single number. A query is not limited to listing notes; it can aggregate them.

### Who can run it, and what it sees

Two rules keep a query from leaking anything:

1. **A visitor can run a query only if they can read the query note.** Set its access like any other note: `free: true` for everyone, a [[en/user/subgraphs|subgraph]] for subscribers. A query the visitor can't read answers `query not found`, the same as one that doesn't exist.
2. **The program sees only notes the visitor can read.** A guest running a public query never gets paid notes in `notes`, so the query can't show their titles or fields by mistake. An admin running the same query sees everything.

With the examples below, a guest asking `by_tag` for `go` gets two posts; an admin gets a third, members-only one.

The query note is published like any other note, so readers who can run the query can also read its code.

#### Seeing every note: `all_notes: true`

Sometimes a page should show more than the visitor can open: "3 open posts, 1 for members". Add `all_notes: true` to the query note's frontmatter. Then `notes` holds every note, and each has `readable`: `true` when the visitor can open it, worked out by the server with the same rules as the site. The program now decides what each visitor gets, so filter on `readable` before returning anything a visitor shouldn't see. Without the flag `notes` only has readable notes.

```jsonnet
local blog = [n for n in notes if std.startsWith(n.path, 'blog/')];

{
  open: [n.title for n in blog if n.readable],
  locked: std.length([n for n in blog if !n.readable]),
  signedIn: currentUser.signedIn,
}
```

A guest gets the titles of open posts and only the number of the locked one:

```json
{
  "open": ["Go on the web", "Hello, Go", "Web basics"],
  "locked": 1,
  "signedIn": false
}
```

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

#### Notes per folder

How many notes the visitor can read in each top-level folder. A guest gets `{"blog": 3, "queries": 5, "tasks": 3}`; an admin, who also sees the members-only post, gets `{"blog": 4, "queries": 5, "tasks": 3}`:

```jsonnet
local folder(n) =
  local parts = std.split(n.path, '/');
  if std.length(parts) > 1 then parts[0] else '(root)';

{
  [f]: std.length([n for n in notes if folder(n) == f])
  for f in std.set([folder(n) for n in notes])
}
```

The counts follow the visitor's access, so the same query gives each reader the numbers they are allowed to see.

#### Other aggregations

The same pattern, a key per group and a value computed from the notes in it, covers:

- **notes per tag** for a tag cloud, grouping by each item of `meta.tags`;
- **an archive by month**, grouping by the first seven characters of `meta.date`;
- **the newest note in each folder**, sorting each group and taking the first;
- **a total of a number field**, such as hours estimated across tasks, with `std.foldl`;
- **a completeness report**: the paths of notes that lack a field such as `description`;
- **one call for a whole dashboard**: an object with several of these sections.

### Errors

A failed call returns `ErrorPayload` with a message:

| Message | Why |
|---|---|
| `query not found` | no such note, or the visitor can't read it |
| `<path> is not a note-query` | the note has no `type: note-query` |
| `<path>: a note-query needs exactly one jsonnet block` | zero or several blocks |
| `params must be JSON`, `params is larger than 16 KB` | bad `params` |
| `<path>:3:5-9 Unknown variable: …` | the code doesn't compile: a typo, an unknown name, a missing bracket; the location is a line and columns in the block, columns on the first line counted from a hidden header |
| `<path>: RUNTIME ERROR: …` | the program failed; the first line of the Jsonnet error |

### In a template

A [[en/user/templates|custom template]] calls the same query with `noteQuery(query, params)`. It runs with the access of whoever views the page and returns the result as lists and maps, so you can range over it:

```jet
{{ range _, p := noteQuery("[[by_tag]]", map("tag", "go")) }}
  <a href="{{ p.url }}">{{ p.title }}</a>
{{ end }}
```

`params` is optional; build it with `map(…)`. Range with two variables, as everywhere in Jet: with one, `range` gives the index. A failing query stops the render; an admin sees its message. The template preview in the admin panel runs queries too, while the check a layout gets on sync sees an empty list.

A guest and a subscriber viewing the same page can get different lists. Pages a guest sees are cached for guests only, so a subscriber never gets a guest's list.

### Note queries or a template

A template can also list notes without a query: `nvs.ByGlob("blog/*.md")` and `SortByMeta` build the list on the server while the page renders, and `json()` hands it to the page. That list is fixed when the page renders. Use `noteQuery()` in a template for aggregates that are clumsy in Jet, or when a JavaScript page and a template should share one query; call it from JavaScript when the answer depends on visitor input, like a tag picked from a menu.

### Limits

- **Every call runs the program again.** There is no cache yet; over a vault of about 850 notes a call takes around 12 ms. For a signed-in visitor who isn't an admin, checking which notes they can read costs a database lookup per note. Both are tracked in [issue #441](https://github.com/trip2g/trip2g/issues/441).
- **A query runs until it finishes.** Jsonnet can't be stopped halfway, so don't write loops whose size comes from `params`, such as `std.range(1, params.n)`: any visitor could make them huge.
- **Watching cost.** The instance's metrics have `trip2g_note_query_duration_seconds` per query note, and a call that takes a second or more is logged as `slow note query`.
