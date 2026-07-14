# RSS

RSS is entirely template-driven: a Jet layout (`_layouts/rss.html`) + a feed note (`layout: rss`, `content_type: application/rss+xml; charset=utf-8`). No Go RSS-specific package exists anymore.

## The one Go primitive: `NoteQuery.Public()`

`internal/templateviews/query.go` — `NoteQuery.Public()` restricts `.All()` results to notes readable by anonymous visitors (`model.NoteView.IsPubliclyReadable()`: `Free`, not a system note, not in a `RequireSignin` subgraph). Filtering happens before offset/limit so pagination can't be used to probe hidden notes.

`model.NoteView.IsPubliclyReadable()` (`internal/model/note.go`) is the single source of truth for "can an unauthenticated visitor see this note" — used by `.Public()`, by `internal/assetindex` (deciding public vs access-checked asset serving), and by any future anonymous-facing feature.

## The default `_layouts/rss.html`

```jet
{{ range i, n := nvs.ByGlob(note.M().GetString("rss_glob", "**")).Public().SortBy("created_at").Desc().Limit(note.M().GetInt("rss_limit", 20)).All() }}
```

Notes:
- `html()` (Jet built-in) escapes into XML numeric refs — used for all text fields including `content:encoded`, avoiding the `]]>`-inside-CDATA problem.
- No `<?xml?>` declaration (avoids a preamble-injection edge case).
- `range i, n := ...` — Jet's single-var `range x := ...` binds the loop index, not the value; the two-var form is required to get the note.

Ships in both `onboarding-vault/` (new-vault default) and `docs/demo/` (e2e fixture), each with a matching `feed.md`.

## What was removed

The old `internal/rssfeed/` package (curated-links model: walk a note's markdown AST, each link → one RSS item) and the `<permalink>.rss.xml` middleware (`cmd/server/routing.go:handleRSSFeed`) are gone, along with the `EnableRSS` config flag (`enable_rss` in the generic `config_bool_values`/`configregistry` system — registry-only, no schema/migration involved). No redirect shim; old `.rss.xml` URLs 404.
