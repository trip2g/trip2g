# TODO: remove the old server-rendered layout/page path

The note page now renders through `defaulttemplate` (`defaulttemplate.WriteRender`).
Several older server-rendered cases and templates are partially or fully
superseded. This doc records what can be removed, what must stay, and the
cascade for each — so the cleanup can be done later as its own change.

> Status: investigation only. Nothing removed yet. Verify each "candidate"
> against live traffic / external links before deleting.

## Verdicts

| Target | Verdict | Why |
|--------|---------|-----|
| `internal/case/rendersearchpage` | **Remove (candidate)** | GET `/search` server-rendered page. The user-facing search uses the GraphQL `search()` query (`assets/ui/user/search/search.view.ts`), not this page. Only reference is the router registration. |
| `internal/case/admin/renderlayoutpreview` | **Remove (candidate)** | Registered endpoint only. No internal callers, not referenced by GraphQL resolvers. No telegram/tg code inside. |
| `internal/case/rendernotepage/view.html` | **Trim** | Of ~350 lines only `LayoutError` is still used (`endpoint.go:291`). The rest (`Note`, `NoteContent`, `Sidebar`, `ReadingMeta`, `ReadingComplexityStars`, `LatestBanner`, `TurboNote`) has zero callers — superseded by `defaulttemplate`. |
| `internal/case/renderlayout` (`Handle` + `views.html`) | **Remove after `rendersearchpage`** | `renderlayout.Handle` has exactly one caller: `rendersearchpage`. Once that is gone, `Handle` / `WriteBeginLayout` / `views.html(.go)` are dead. **Keep the `Env`/`Params` types** — still used by `render404`, `rendernotepage` (`buildDefaultTemplateCtx`), and admin previews. |
| `internal/case/admin/renderpreview` | **KEEP — do NOT remove** | Active admin live-preview, NOT telegram. Wired into: GraphQL resolver (`schema.resolvers.go:1293` → `renderpreview.Resolve`), `PreviewBuffer` (`cmd/server/main.go` `previewBuffer`, `PreviewBuffer()`), and its own config section (`appconfig/config.go` `RenderPreview`). |

Note: the original assumption that `renderpreview` / `renderlayoutpreview` "were
for Telegram" is not supported — neither package references telegram/tg/bot.
`renderpreview` is the admin editor preview; `renderlayoutpreview` is an
admin layout preview endpoint with no remaining callers.

## Removal steps (when doing the cleanup)

1. **rendersearchpage**
   - Delete `internal/case/rendersearchpage/`.
   - Regenerate the router: `go generate ./internal/router/...` (registration list is built by `internal/router/gencmd`).
   - Before deleting: confirm nothing links to `GET /search` (sitemap, templates, external SEO). The search UI itself is unaffected (GraphQL).

2. **renderlayout Handle + template** (only after step 1)
   - Remove `Handle`, `WriteBeginLayout`/`WriteFinishLayout`/TG-layout funcs, and `internal/case/renderlayout/views.html` + `views.html.go`.
   - Keep `render.go`'s `Env`, `Params`, `HrefLang` types (still imported elsewhere).

3. **renderlayoutpreview**
   - Delete `internal/case/admin/renderlayoutpreview/`.
   - Regenerate the router.
   - Before deleting: confirm the admin UI does not call its endpoint path.

4. **rendernotepage/view.html trim**
   - Reduce `view.html` to just the `LayoutError` func (drop `Note`, `NoteContent`, `Sidebar`, `ReadingMeta`, `ReadingComplexityStars`, `LatestBanner`, `TurboNote` and their `Stream*`/`Write*`).
   - Regenerate: `go generate ./internal/case/rendernotepage/...`.
   - Alternative: move `LayoutError` into `defaulttemplate` and delete `view.html` entirely.

## Verification after cleanup

- `go build ./...` and `go vet ./...` clean.
- `go generate ./internal/router/...` produces no diff beyond the removed entries.
- Note page, 404 page, admin editor preview, and admin layout list still render.
- Search still works from the UI (GraphQL path).
