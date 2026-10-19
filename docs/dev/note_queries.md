# Named note queries (`noteQuery`) — MVP

**Summary:** a note with `type: note-query` and one ```` ```jsonnet ```` block is a query. `noteQuery(input: {query, params})` runs it: the program gets `notes` (every note the caller can read) and `params` (JSON from the caller), and its result comes back as a JSON value in `result`. The vault author writes the code; a visitor only passes params. This is the answer to "an SPA needs frontmatter queries" without letting visitors run their own jsonnet or see raw frontmatter.

## Contract

```graphql
noteQuery(input: { query: "[[by_tag]]", params: { tag: "go" } }) {
  ... on NoteQueryPayload { result rows { data note { title url html } } }
  ... on ErrorPayload { message }
}
```

- `query` is a vault path (`queries/by_tag.md`, with or without `.md`) or a wikilink target, resolved with `NoteViews.ResolveWikilinkTarget`.
- Inside the block: `currentUser` (see Access); `notes` — `[{path, path_id, version_id, title, permalink, meta}]` in path order, `meta` after frontmatter patches; `params` — the caller's `params` (a `JSON` scalar, any value), `{}` when omitted, at most 16 KB once encoded.
- Notes come from `LatestNoteViews()`. Releases (live versions) are not in use, so latest is what the site serves. Several readers pick live or latest by different rules; that is tracked in https://github.com/trip2g/trip2g/issues/438.
- `notes` leaves out system notes (`_…`) and files without a markdown AST (`.canvas`, `.base`, `.excalidraw`) — `queryable()` in `resolve.go`. `search: false` notes stay in: the flag is about discoverability, not access, and the note's `meta` carries `search: false`, so a program filters on it when it wants to. (Commit 951d08fc had excluded them, like search and embeddings; that was reverted.)
- A NaN or ±Inf in frontmatter (YAML allows them, JSON doesn't) becomes `null`, so one odd note can't break every query.
- The result comes back in `result`, typed with the `JSON` scalar (gqlgen `graphql.Any`): any JSON value, sent as is, without string-wrapping.

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

## Rows

`rows: [NoteQueryRow!]` splits a list result into `{data, note}`; null when the result is not a list (`runnotequery.Rows`, which decodes `result` into `[]json.RawMessage`, so `data` is the item's exact JSON). `note: PublicNote` is resolved per row by `rowNotes.RowNote` (`rows.go`), by precedence: `version_id` (that version), else `path_id` (latest), else `path` (vault path). A present key that names nothing gives null; it does not fall through to the next key. A non-integer id or a non-object item gives null too.

- **Lazy.** `Resolve` puts a `rowNotes` on the payload (`NoteQueryPayload.Notes`, a `goExtraField` holding the `model.NoteQueryRowNotes` interface), which does nothing until a row's `note` is resolved. The first lookup resolves the viewer (`canreadnote.NewReader`) and builds version / path-id indexes over the same `NoteViews` snapshot the program ran on; every row after it reuses them (`sync.Once`, gqlgen resolves rows concurrently). A request that doesn't select `note` pays nothing.
- **Second wall.** The row's note goes through the reader again, independently of what the program saw. Without it `all_notes: true` (or a program returning ids from `params`) would hand out notes the viewer can't read. System notes and non-markdown files are refused like in `notes`.
- **Versions.** Only the snapshot's versions resolve: a `version_id` that is no longer the note's latest gives null. Within one call the program and the rows share the snapshot, so ids the program returns always resolve.

## In Jet templates

`runnotequery.JetFunc` is exposed as `noteQuery(query, params?)` in page renders (`rendernotepage/endpoint.go`) and the admin layout preview, running with the request's viewer. It returns the result as `[]any` / `map[string]any`; a failing query panics the render with its message (shown to admins). `runnotequery.JetRowsFunc` is `noteQueryRows(query, params?)`: the same call, whose result must be a list (else the render stops), as `[]*JetRow` with `Data()` (the item as maps and lists) and `Note()` (the `templateviews.Note` for the page's domain, or an untyped nil), with the same lookup and access re-check as `rows`. The load-time smoke render (`noteloader/smoke_render.go`) renders each layout with up to 10 notes that use it, guest-readable ones (`IsAnonymouslyReadable`, the page-render gate) first, and runs the real query (`noteQuery` and `noteQueryRows` alike) over the notes being loaded (metrics not recorded). A guest-readable note runs it as a guest, so a layout reading an object result renders and a members-only query on a public page warns. Any other note runs it with an admin token: only a signed-in viewer opens that page, so a guest's `query not found` there is noise, while a missing or broken query still warns. Each distinct (query, params) runs once per layout and viewer, on every load. The anonymous page cache is keyed and served to guests only, so a guest's query result is never served to a signed-in viewer.

## Access

Two checks, both through `CanReadNote` (`internal/case/canreadnote`):

1. **Who may run a query:** whoever can read the query note. The author sets it like any note's access: `free`, subgraphs, sign-in. A query the caller can't read answers `query not found`, the same as a missing one.
2. **What the program sees:** only notes the caller can read. A public query run by a guest never sees paid notes, so a mistake in the program's output can't leak their frontmatter or titles.

`all_notes: true` in the query note's frontmatter is an explicit opt-out of rule 2: `notes` holds every note (still without system and non-markdown files), each with `readable` (the `CanReadNote` result). The author filters on `readable`. The flag sits in the query note, so it shows up in review.

Every query also gets `currentUser` — `{id, role, isAdmin, signedIn}` from `CurrentUserToken` (no email: it isn't in the token). A guest has null `id` and `role`. A result that reads `id` is per user, which the cache in #441 has to account for.

The program decides which fields leave the server. Raw frontmatter is never returned unless a query returns it.

## Errors (`ErrorPayload`)

`query not found`, `<path> is not a note-query`, `a note-query needs exactly one jsonnet block`, `params must be JSON`, `params is larger than 16 KB`, and jsonnet errors, first line only: a static error as `<path>:line:col Unknown variable: …` (the snippet is named after the note; first-line columns include the prelude), a runtime one as `<path>: RUNTIME ERROR: …`.

## Cost

`BenchmarkResolveDocs`: one query over the repo's `docs/` vault (857 notes) takes ~12 ms and ~11 MB per call, access checks mocked. Every call rebuilds `notes` and evaluates the program; nothing is cached. The viewer is resolved once per call (`canreadnote.NewReader`): an admin reads everything, a guest reads anonymously readable notes, a signed-in reader's active subgraphs are listed once; scoped tokens and federated identities go through `canreadnote.Resolve` per note, which reads no DB for them. `noteQuery` costs 10 in the GraphQL complexity limit of 30 (`internal/graph/handler.go`), so one request runs at most two queries.

## Observability

A query is the instance admin's own code, so trip2g doesn't sandbox it beyond `MaxStack`; it makes slow queries visible instead.

- `trip2g_note_query_duration_seconds{query, result}` — histogram per query note. `query` is the note's path once the caller may see it, else `unknown` (a visitor's string never becomes a label). `result` is `ok`, `error` (an `ErrorPayload`), `not_found` or `failed` (a system error). Buckets run from 1 ms to ~16 s.
- A call that takes a second or more is logged as `slow note query` with the query, result, seconds and params size (`cmd/server/note_query.go`).
- The generic GraphQL metrics already cover the field: `trip2g_graphql_resolver_duration_seconds{object="Query",field="noteQuery"}`, and `trip2g_graphql_resolver_started_total` minus `…_done_total` shows calls still running — the sign of a runaway query.

go-jsonnet can't be cancelled (no context, step limit or settable hook; upstream issue google/go-jsonnet#729), so a stuck query holds its goroutine until it finishes.

## Not in the MVP

- A result cache keyed by access class: https://github.com/trip2g/trip2g/issues/441.
- `links` / `backlinks` in `notes`, and the note body.
- A time limit or concurrency cap for query programs (see Observability). A public query whose loops depend on `params` (`std.range(1, params.n)`) lets a guest burn CPU; write queries that don't.
- An admin-only `filterNotes` that runs caller-supplied jsonnet (raised in discussion; `notePaths(filter: {frontmatter: [...]})` already filters by `exists`/`equals` with an API key).
