# Static export mode: one-shot generator (Mode A) + incremental static cache (Mode B)

**TL;DR:** Both modes are feasible, and they are the same machinery. The anonymous
page cache (`internal/pagecache` + `internal/case/rendernotepage/pagecache.go`)
already answers the hard question — *which pages are safe to serve without the Go
app* — with battle-tested gates. Mode A = drive the real fasthttp handler in-process
over an enumerated URL list and write bodies to disk (`trip2g export --out dir`).
Mode B = run that same export after every `commitNotes` with an atomic directory
swap, and let nginx `try_files` the result. Do **not** attempt precise per-page
invalidation first: today's invalidation model is deliberately "any commit → rebuild
everything" (`ClearPageCache()` on every reload), and cross-note dependencies (Jet
layouts query the whole corpus) make a precise graph unsound. Recommended MVP:
Mode A (~3–5 days), then Mode B as "Mode A on commit + swap" (+2–3 days).

## 1. How a page is produced today

The expensive work is already done at **load time**, not request time:

- `noteloader.Load` (`internal/noteloader/loader.go:140`) builds the full in-memory
  corpus: markdown → HTML per note (`note.HTML`, per-domain `note.DomainHTML`),
  wikilink resolution, backlinks (`note.InLinks`), the bleve search index, layouts,
  the sitemap bytes (`model.NoteViews.Sitemap`, `note.go:325`) and per-custom-domain
  sitemaps (`DomainSitemaps`, `note.go:339`).
- Per **request**, `rendernotepage.Endpoint.Handle` (`internal/case/rendernotepage/endpoint.go:33`)
  assembles a `defaulttemplate.Ctx` or Jet layout vars and executes the template.
  Per-request extras: telegram post links (DB, `resolve.go:335`), HTML injections
  (DB), similar notes (in-memory vectors, `endpoint.go:689`), per-user view
  recording (authenticated only), title/OG/hreflang assembly.
- The **anonymous page cache** (`internal/pagecache/pagecache.go`) stores the final
  pre-gzipped 200 HTML keyed by `{Path, Host, NoteVersionID, ConfigEpoch, UILang}`,
  filled at request time (`fillPageCache`, `pagecache.go:259`), served early
  (`serveCachedPageEarly`, `pagecache.go:125`), and flushed wholesale on every note
  reload (`cmd/server/main.go:547,567`).

`cacheDecision` (`rendernotepage/pagecache.go:55`) is the canonical list of what
makes a page non-uniform: authenticated viewer, `?version=` preview, personalized
Jet layout (`layout.Personalized`), non-whitelisted UI lang. Export reuses exactly
these gates.

## 2. Everything that makes a page dynamic today

| Feature | Where | Static verdict |
|---|---|---|
| Sign-in wall / paywall / non-free notes | `resolve.go:291-327` | **Exclude** from export (same as cache: `serveCachedPageEarly` gates at `pagecache.go:190-197`) |
| live vs latest corpora, `?version=` admin preview, draft banner | `resolve.go:210-224`, `checkLatestBanner` | Export **live** only (or latest when `ShowDraftVersions`) — mirrors the cache's source selection (`pagecache.go:147-151`) |
| Search widget | `views.html:421` `$trip2g_user_search` → GraphQL `siteSearch` (bleve + vector, `internal/case/sitesearch`) | Degradable: hide in export mode, or later ship a client-side index (pagefind). Mode B: proxy `/_system/graphql` to the app — search keeps working |
| Similar notes block | `endpoint.go:689` | Baked at render time — static-friendly (stale until re-export) |
| datachart `url:` sources | `internal/chartdata` — background queue fetches, stores in DB, injected at load; `chart.js` renders client-side | Baked HTML + static JS asset — works; refreshes only on re-export |
| SSE `noteChanges` subscription | `graph/schema.resolvers.go:3401`, SSE bypass in `server.go:149` | Mode A: gone (acceptable). Mode B: proxy to app |
| Lang negotiation: `redirectToRightLang` (cookie + Accept-Language, `endpoint.go:308`), `?setlang` cookie (`endpoint.go:275`), embedded `ui_lang` (`views.html:58`) | per-request | **Hard.** Export one variant per page with site-default `ui_lang`; hub pages lose auto-redirect (degrade to a small inline JS redirect using `navigator.language`). nginx `map $http_accept_language` is possible but not MVP |
| Redirects: alternate-permalink 301 (`endpoint.go:84`), `note.Redirect` 302, `redirectManager` (`server.go:109`) | per-request | Emit an nginx `map`/include with all redirects at export time, or meta-refresh stub pages |
| sitemap.xml / per-domain sitemaps | precomputed bytes, `routing.go:165` | Trivially exportable |
| RSS `*.rss.xml` | generated from in-memory corpus, `routing.go:128` | Exportable per publicly-readable note |
| `content_type` notes (robots.txt, llms.txt) | `endpoint.go:229` | Exportable as raw files with correct extension/MIME (nginx types or explicit config) |
| Embedded assets `/assets/*` | embedded FS, cache-busted `AssetURL` (`cmd/server/assets.go:126,201`) | Copy to disk once per binary version |
| Note assets, local storage `/_assets/*` | `internal/localstorage` — stable non-expiring URLs | Copy directory |
| Note assets, **minio presigned URLs** | `miniostorage/storage.go:182`; URLs baked into note HTML, expire → `OnURLExpiring` triggers full reload (`assets.go:26`) | **Hard blocker for Mode A** on minio-backed sites: exported HTML rots when URLs expire. Require local storage, or add a stable proxy-URL mode |
| HTML injections, site config | DB, `ConfigEpoch` bumps | Baked; config change must trigger re-export (Mode B) or is stale (Mode A) |
| GraphQL (all POST), MCP, gitapi, OAuth/OIDC, payments webhooks, tg webhooks, purchase/hat/tg-auth tokens, admin SPA (`/admin`, `/_system/admin`), admin previews, federation topology, 404 tracking | `server.go`, `routing.go:200-266`, `router/endpoints_gen.go` | Dynamic-only. Mode A: dropped. Mode B: proxied to the app |
| Custom domains | `RouteMap[host]` (`note.go:331`), strict isolation (`resolve.go:519`) | Export one tree per host; nginx `server` block per domain |

## 3. Route inventory (classification)

From `cmd/server/server.go` handler order + middlewares (`routing.go:200`) +
`router/endpoints_gen.go`:

- **Static-exportable:** `rendernotepage` catch-all GET (anonymous-readable subset),
  `/sitemap.xml`, `*.rss.xml`, `content_type` notes, `render404` (as `404.html`),
  `/assets/*`, `/_assets/*`.
- **Degradable:** search widget (hide / client index / proxy), lang redirects (JS
  snippet), `hat`/`setlang` query params (no-op without app), 404 tracking (lost).
- **Dynamic-required:** GraphQL `/_system/graphql` (+ `/graphql` compat incl. SSE),
  MCP, gitapi smart-HTTP, OAuth/OIDC start+callback, payment webhooks (`processnowpaymentsipn`,
  `processpatreonwebhook`), telegram webhooks, `signinbyhat`, `/admin` + `/_system/admin`,
  admin render previews, `downloadonboardingvault`, `federationtopology`, obsidian
  sync (pushNotes etc. — all GraphQL POST).

## 4. Mode A — `trip2g export --out <dir>`

**Approach: in-process request replay.** Boot the app in read-only warmup (Block A
only — same code path a read replica uses, `main.go:481`), then instead of
`startServer()`, iterate the URL list and invoke the already-built fasthttp handler
with a synthetic anonymous `fasthttp.RequestCtx` (`Accept-Encoding: gzip` off, no
cookies), writing each 200 body to `<out>/<path>/index.html` (or raw filename for
`content_type` notes). Exit 0 on success, non-zero if any page 5xxs. This reuses
the entire pipeline — templates, layouts, redirects, content types — with zero
duplicate rendering logic. Precedent for off-request rendering exists
(`noteloader/smoke_render.go`, `renderpreview`), but replaying through the real
handler is strictly more faithful.

**URL enumeration** (all in memory after `loadAllNotes`):
- main host: `NoteViews.Map` (permalinks) + `RouteMap[""]` aliases; root `/`
  (magazine fallback, `resolve.go:246`).
- per custom domain: `RouteMap[host]` keys → separate output tree `<out>/<host>/`.
- skip: paywalled (`!note.Free`), `RequireSignin` subgraphs, `/_` system paths,
  personalized layouts — exactly the `serveCachedPageEarly` gate list.
- extras: `/sitemap.xml` (+ per-domain), `<path>.rss.xml` for `rssfeed.IsPubliclyReadable`
  notes, `404.html`, `/assets/` dump, `/_assets/` copy (local storage).
- redirects (alternate permalinks, `note.Redirect`, redirect manager, lang hubs) →
  emit `redirects.map` for nginx include.

**Baked-in decisions (MVP):** single `ui_lang` = site default; search box hidden via
an export flag on `defaulttemplate.Ctx` (or left pointing at a dead endpoint —
prefer hiding); telegram links/injections/similar-notes frozen at export time;
minio storage refused with a clear error (`--allow-expiring-assets` to override).

**Effort: ~3–5 days** (CLI plumbing + enumerator + synthetic-request writer + nginx
snippet emitter + tests). No changes to the render path itself.

## 5. Mode B — incremental static cache behind nginx

**Approach: Mode A triggered on every reload, atomic swap.** Subscribe to the same
signal that clears the page cache today — `PrepareLatestNotes`/`PrepareLiveNotes`
(`main.go:532,561`), which every `commitNotes` (`internal/case/commitnotes/resolve.go:49`)
and live-notes save funnels through. On each reload (debounced a few seconds):
export to `<dir>.tmp`, `rename` over a `current` symlink. nginx:

```nginx
location / {
    root /var/lib/trip2g/static/current;
    try_files $uri $uri/index.html @app;
}
location @app          { proxy_pass http://127.0.0.1:8081; }
location /_system/     { proxy_pass http://127.0.0.1:8081; }  # GraphQL, MCP, admin
location = /graphql    { proxy_pass http://127.0.0.1:8081; }
# all POST → app (nginx: if ($request_method = POST) { proxy_pass ... } or limit_except)
include /var/lib/trip2g/static/current/redirects.map.conf;
```

**Auth-gated pages bypass the cache structurally:** paywalled/signin pages are never
exported, so `try_files` misses and falls through to `@app`, which runs the normal
authenticated pipeline. Same for `?version=`, `?hat=`, `?setlang=` — nginx must be
configured to fall through to `@app` whenever a query string is present on a note
URL (one `if ($args)` guard), since static files can't vary on query params.
Signed-in users get *stale-free but slower* pages only if we additionally bypass
static for requests carrying the `trip2g_token` cookie — recommended, one more
nginx guard.

**Why not precise per-page invalidation:** there is no sound note→pages graph.
Jet layouts receive the whole corpus (`nvs` var, `resolve.go:385`) and can query it
arbitrarily; sidebars/headers/footers/glob widgets (`defaulttemplate/widgets.go:120`),
`LayoutSections`, the magazine root, backlink lists, hreflang groups and sitemaps
all fan out. `note.InLinks` covers only wikilink backlinks. The app itself already
concluded this: reload flushes the *entire* page cache with an explicit comment
("flush the whole anonymous page cache rather than reasoning about which keys
moved", `main.go:545`). The read-replica lesson applies verbatim: a cache without a
provable invalidation path fails silently. Full re-export per commit is honest;
with pre-rendered HTML in memory, exporting even thousands of pages is seconds of
template execution, and the debounce absorbs push bursts. Precision (dirty-set =
changed notes + their `InLinks` + hub/index pages + sitemap/RSS, everything else
weekly) is a later optimization behind a flag, only if export time actually hurts.

**Risks:** staleness window between commit and swap-complete (seconds — fine);
config/injection changes must also trigger export (hook `ConfigEpoch` bump); minio
URL expiry forces periodic re-export anyway (the `OnURLExpiring` reload can drive
it); disk churn on large vaults (two full trees during swap); nginx config drift
(ship a generated snippet, not hand-edited instructions).

**Effort on top of Mode A: ~2–3 days** (trigger wiring + debounce + atomic swap +
nginx template + docs). Precise invalidation would be weeks and is not recommended.

## 6. The 3 hardest dynamic features to degrade

1. **Language negotiation.** `ui_lang` is embedded server-side per request and hub
   pages 302 by cookie/Accept-Language. Static export must pick one language per
   URL; auto-routing degrades to an inline JS snippet or is lost. (Mode B softens
   this: only anonymous no-cookie requests hit static.)
2. **Search.** The widget is a mol component speaking GraphQL to bleve/vector
   lanes. Mode A: hide it or invest in a client-side index. Mode B: proxy GraphQL —
   search survives untouched.
3. **Minio presigned asset URLs.** Baked into note HTML with an expiry; the running
   app re-renders on `OnURLExpiring`, a static tree cannot. Mode A must require
   local asset storage (or a new stable-proxy-URL asset mode); Mode B must re-export
   on the expiring callback.

## 7. Recommendation

Build **Mode A first**; Mode B is a thin loop around it. Shared machinery: one
`internal/staticexport` package (enumerator + synthetic-request renderer + writers +
nginx snippet emitter), a `--export` CLI mode using the read-only boot path, and a
reload-triggered runner for Mode B. MVP scope: local-storage sites, single default
UI lang, search hidden, redirects via nginx map, full re-export per commit.
Total: roughly one week for both modes.
