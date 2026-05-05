# Domain-Aware Note URLs

**Date:** 2026-05-04
**Status:** Approved

## Problem

Two places in the app return note URLs but neither accounts for custom domain routing:

1. **GraphQL `NoteView`** has `permalink` (path only, e.g. `/my/note`) but no full `url` field.
2. **MCP search results** build `URL = env.PublicURL() + note.Permalink`, always using the main domain even if the note is served on a custom domain.

`NoteViews.ResolveURL` exists but is an unimplemented stub returning only the path.

**Goal:** Both GraphQL and MCP return the full URL, preferring the first custom domain route if one exists, falling back to `publicURL + permalink`.

## Shared Infrastructure (model layer)

Add a reverse route index to `NoteViews`:

```go
// In NoteViews struct
customDomainRoutes map[int64]struct{ Host, Path string }
```

Populated in `RegisterNoteRoutes`: for each route with non-empty host, store the first one per `note.PathID`.

Implement `ResolveFullURL` (replacing the current stub):

```go
func (nv *NoteViews) ResolveFullURL(note *NoteView, publicURL string) string {
    if r, ok := nv.customDomainRoutes[note.PathID]; ok {
        scheme := schemeFrom(publicURL) // parses scheme from publicURL
        return scheme + "://" + r.Host + r.Path
    }
    return publicURL + note.Permalink
}
```

## GraphQL

- Add `url: String! @goField(forceResolver: true)` to `NoteView` in `schema.graphqls`
- Run `make gqlgen` to regenerate resolver stub
- Add `NoteURL(*model.NoteView) string` to the GraphQL resolver `Env` interface
- App implements: `return a.noteViews.ResolveFullURL(note, a.config.PublicURL)`
- Resolver: `return r.env.NoteURL(obj), nil`

## MCP

- Add `NoteURL(*model.NoteView) string` to MCP `Env` interface
- Same app implementation as GraphQL
- Change `buildSearchPayload`, `buildSimilarPayload`, `searchResultItemFromNote` to accept `noteURL func(*model.NoteView) string` instead of `publicURL string`
- Call site: `buildSearchPayload(query, results, env.NoteURL, env.LatestNoteChunks())`

## E2E Tests

### Seed note

Add one `.md` file to the seedvault with:
```
---
route: customdomain.test/mcp-url-test
---
```

Check if the existing multidomain seedvault notes already provide a suitable note; if not, add a dedicated one (e.g. `mcp-url-test.md`).

### `e2e/note-url.spec.js` (GraphQL)

- Query `notePaths { latestNoteView { url permalink } }`
- Assert: note with `route: customdomain.test/mcp-url-test` returns `url` starting with `http://customdomain.test/`
- Assert: regular notes return `url` starting with `APP_URL`

### `e2e/mcp-url.spec.js` (MCP)

- `tools/call` → `search` with a query matching the custom-domain note
- Assert: result `url` starts with `http://customdomain.test/`
- Assert: result `url` for a regular note starts with `APP_URL`

## Key Decisions

- **First custom domain wins**: if a note has multiple custom domain routes, the first one registered is used
- **Scheme** is derived from `publicURL` (so `http://` in dev, `https://` in prod)
- **Both GraphQL and MCP** use the same `NoteURL` method on `app`, backed by the same `ResolveFullURL` logic
- `NoteViews.ResolveFullURL` is the single source of truth; the old stub `ResolveURL` is replaced
