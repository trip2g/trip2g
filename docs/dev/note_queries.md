# Named note queries (`noteQuery`) — MVP

**Summary:** a note with `type: note-query` and one ```` ```jsonnet ```` block is a query. `noteQuery(input: {query, params})` runs it: the program gets `notes` (every note the caller can read) and `params` (JSON from the caller), and its result comes back as a JSON string. The vault author writes the code; a visitor only passes params. This is the answer to "an SPA needs frontmatter queries" without letting visitors run their own jsonnet or see raw frontmatter.

## Contract

```graphql
noteQuery(input: { query: "[[by_tag]]", params: "{\"tag\": \"go\"}" }) {
  ... on NoteQueryPayload { json }
  ... on ErrorPayload { message }
}
```

- `query` is a vault path (`queries/by_tag.md`, with or without `.md`) or a wikilink target, resolved with `NoteViews.ResolveWikilinkTarget`.
- Inside the block: `notes` — `[{path, title, permalink, meta}]` in path order, `meta` after frontmatter patches; `params` — the parsed `params`, `{}` when omitted, at most 16 KB.
- Notes come from `LiveNoteViews()`, what the public site serves (latest only when `ShowDraftVersions` is on). An unreleased note never reaches a query, and an unreleased edit to a query note doesn't run until release.
- `notes` leaves out system notes (`_…`) and files without a markdown AST (`.canvas`, `.base`, `.excalidraw`).
- A NaN or ±Inf in frontmatter (YAML allows them, JSON doesn't) becomes `null`, so one odd note can't break every query.
- The result is compacted JSON. Any JSON value is allowed.

Example query note:

````markdown
---
type: note-query
free: true
---

```jsonnet
local tagged = [
  n for n in notes
  if std.member(std.get(n.meta, 'tags', []), params.tag)
];
[{title: n.title, url: n.permalink} for n in tagged]
```
````

## Access

Two checks, both through `CanReadNote` (`internal/case/canreadnote`):

1. **Who may run a query:** whoever can read the query note. The author sets it like any note's access: `free`, subgraphs, sign-in. A query the caller can't read answers `query not found`, the same as a missing one.
2. **What the program sees:** only notes the caller can read. A public query run by a guest never sees paid notes, so a mistake in the program's output can't leak their frontmatter or titles.

The program decides which fields leave the server. Raw frontmatter is never returned unless a query returns it.

## Errors (`ErrorPayload`)

`query not found`, `<path> is not a note-query`, `a note-query needs exactly one jsonnet block`, `params must be JSON`, `params is larger than 16 KB`, and jsonnet errors as `<path>: RUNTIME ERROR: …` (first line only).

## Cost

`BenchmarkResolveDocs`: one query over the repo's `docs/` vault (857 notes) takes ~12 ms and ~11 MB per call, access checks mocked. Every call rebuilds `notes` and evaluates the program; nothing is cached. For a signed-in reader who isn't an admin, `CanReadNote` lists their active subgraphs from the database **once per note**, which this benchmark doesn't include.

## Not in the MVP

- A result cache keyed by (query note version, params, the caller's access set, vault version).
- One subgraph lookup per call instead of per note.
- `links` / `backlinks` in `notes`, and the note body.
- A time limit for a query program. `import` is blocked and `MaxStack` stops deep recursion, but a wide loop runs on the request goroutine, and the request context doesn't stop it. A public query whose loops depend on `params` (`std.range(1, params.n)`) lets a guest burn CPU; write queries that don't.
- An admin reading the latest version (`?version=latest`, as pages allow).
- User docs, and an admin-only `filterNotes` that runs caller-supplied jsonnet (raised in discussion; `notePaths(filter: {frontmatter: [...]})` already filters by `exists`/`equals` with an API key).
