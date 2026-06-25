# Rendering Excalidraw drawings

## TL;DR

Excalidraw files **already sync, store, and reach the page endpoint** — the only
missing piece is **rendering**. This is a render-layer feature, not a greenfield
build. `.excalidraw` (and `.canvas`/`.base`) are in the sync whitelist
(`internal/case/pushnotes/resolve.go:27-33`), stored verbatim in
`note_versions.content` as raw files (`internal/mdloader/loader.go:263-299`),
get a URL that keeps the extension (`/board.excalidraw`), and hit the catch-all
page endpoint — where today they short-circuit to a "not supported yet"
placeholder (`internal/case/rendernotepage/endpoint.go:96-103` →
`views.html` `UnsupportedFile`, line 533-534).

The plan: **render the scene to a static SVG/PNG asset via an external render
service** (`EXCALIDRAW_RENDER_URL`). On note load we **enqueue a background
render job** (mirroring `internal/chartdata`: enqueue-on-miss + debounced
reload), POST the scene JSON to the service, get back SVG/PNG, **store it as a
`note_assets` asset on disk/S3** (NOT a SQLite blob), keyed by **scene
content-hash** so identical scenes dedupe and unchanged scenes skip re-render. A
small bookkeeping table maps `scene_hash → asset_id + last_error` (mirrors
`chart_data_cache`, `db/schema.sql:850`). The page shows a **placeholder until
the asset lands**, then live-pull (notebus/SSE) swaps it in.

**Owner decision — no client-side fallback viewer.** If there is nothing to
render with — `EXCALIDRAW_RENDER_URL` is unset, the render errored, or no asset
exists yet and no service is configured — the page just shows the **existing
"not supported yet" placeholder** (the current stub at
`endpoint.go:96-103` → `views.html` `UnsupportedFile`). No heavy `@excalidraw`
React bundle ships to readers, no fallback renderer. This makes Excalidraw
**almost pure-backend**: when a render exists the page emits a plain `<img>` /
inline SVG; the only frontend work is serving that asset (no JS widget).

A **format wrinkle** dominates the parser work: the common Obsidian
Excalidraw-plugin file is `foo.excalidraw.md` — scene JSON embedded in a
markdown wrapper. `filepath.Ext("foo.excalidraw.md") == ".md"`, so trip2g treats
it as **plain markdown today** (renders the wrapper text, not a drawing). Only a
pure `.excalidraw` (JSON) file hits the raw-file path, and it has **no parser**.
The design **targets both**: extract the scene JSON out of the `.excalidraw.md`
wrapper, and parse the pure `.excalidraw` JSON.

This is effectively the **first real consumer** of the planned-but-unimplemented
external-HTTP render mechanism in [template_processors.md](template_processors.md)
(buffered render → POST to an allowlisted host at cache-warm time). We align the
HTTP contract with it but ship a **narrower, dedicated path first** (no template
engine involvement, no Jet `apply_processor`).

---

## What Excalidraw is, and how it'll be used here

Excalidraw is a hand-drawn-style whiteboard: boxes, arrows, freehand strokes,
text, embedded images. The
[Obsidian Excalidraw plugin](https://github.com/zsviczian/obsidian-excalidraw-plugin)
stores a drawing as a **scene**: a JSON document with `elements[]` (shapes),
`appState` (theme, background), and `files{}` (embedded image blobs, base64).

On a published trip2g site a drawing is a **figure**: an author keeps an
architecture sketch or a flow diagram in their vault, links it from a note (or
publishes the file as its own page at `/board.excalidraw`), and a reader sees a
crisp static image — light or dark to match the site theme — with no editor
chrome and no multi-megabyte React bundle.

### The two on-disk formats (critical)

| File | `filepath.Ext` | Routed as | Content |
|------|----------------|-----------|---------|
| `foo.excalidraw.md` | `.md` | **markdown** (full pipeline) | Markdown wrapper with scene JSON inside a code block / `## Drawing` section, often compressed |
| `foo.excalidraw` | `.excalidraw` | **raw file** (`isRawFile`) | Pure scene JSON |

The `.excalidraw.md` form is what the Obsidian plugin produces by default, so
it's the **common case** — and the trap. Because the extension resolves to
`.md`, `isRawFile` (`loader.go:263-266`) returns false and the file runs the
**entire markdown pipeline**: goldmark parses it, the wrapper markdown becomes
HTML, and the reader sees a fenced code block full of JSON (or, if compressed, a
base64 blob), not a drawing.

The `.excalidraw.md` wrapper looks roughly like:

```markdown
---
excalidraw-plugin: parsed
---

# Excalidraw Data
## Text Elements
...
## Drawing
```json
{"type":"excalidraw","version":2,"elements":[...],"appState":{...},"files":{...}}
```
%%
```

The scene lives in the ` ```json ` (or ` ```compressed-json `) block under
`## Drawing`. Compressed scenes are LZ-string–packed base64. The design must
detect this frontmatter/marker and **extract the scene JSON** before rendering;
otherwise it's just markdown.

---

## Current state in trip2g (what already exists vs the stub)

End-to-end, **everything but rendering already works**:

| Stage | Status | Evidence |
|-------|--------|----------|
| Sync whitelist (plugin/CLI/browser) + server `pushNotes` | DONE | `internal/case/pushnotes/resolve.go:27-33` (`.excalidraw` ∈ `allowedExtensins`) |
| Stored verbatim in `note_versions.content` | DONE | `loader.go:registerRawFile` (270-299): `Content: src.Content`, `RawMeta: {}` (no frontmatter), no AST |
| Gets a URL, **keeps extension** | DONE | `loader.go:296` → `Permalink = "/" + src.Path` (e.g. `/board.excalidraw`) |
| Reaches the single page endpoint | DONE | catch-all `rendernotepage.Endpoint` (`endpoint.go:31`) |
| Skipped from Bleve search index | DONE | raw files have no AST/HTML to index |
| **Rendering** | **STUB** | `endpoint.go:96-103` short-circuits to `UnsupportedFileExt` placeholder |

The placeholder, today:

```go
// internal/case/rendernotepage/endpoint.go:96-103
if resp != nil && resp.Note != nil {
    if ext := unsupportedFileExt(resp.Note.Path); ext != "" {  // ".excalidraw"
        dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
        dtCtx.UnsupportedFileExt = ext
        defaulttemplate.WriteRender(ctx, dtCtx)
        return nil, nil
    }
}
```

`unsupportedFileExt()` (endpoint.go:176-186) matches by `strings.HasSuffix` on
`.canvas` / `.base` / `.excalidraw`. The placeholder text is the `UnsupportedFile`
quicktemplate func (`views.html:527-541`, generated `StreamUnsupportedFile` at
`views.html.go:2314-2348`), with the `.excalidraw` branch already present:

```html
{% elseif ctx.UnsupportedFileExt == ".excalidraw" %}
<p>Excalidraw files are not supported yet.</p>
```

Two gaps to note about the short-circuit:

1. **It only sees pure `.excalidraw`.** `foo.excalidraw.md` never reaches it —
   it's been treated as markdown long before (the path doesn't end in
   `.excalidraw`). The renderer must intercept it earlier (at load time / in the
   markdown pipeline), not here.
2. **It runs before paywall/signin** (endpoint.go:96 is above the `PaywallError`
   / `SigninWallError` handling at 115-141). Raw files currently skip access
   control entirely. A real renderer that surfaces drawing content **must
   re-apply** `Free` / `CanReadNote` — see [Access control](#access-control).

Precedents we lean on, all verified present:

- **Canvas parser** `internal/obsidiancanvas.Parse` → `model.NoteView.Canvas`
  (`note.go:284`), invoked in `registerRawFile` (loader.go:286-293). Proven
  end-to-end by the Telegram canvas-bot. Excalidraw gets an analogous parser.
- **Conditional per-note accessor pattern** `HasCharts()` /
  `HasCodeLanguage("mermaid")` (`templateviews/note.go:167-183`): a narrow
  template-facing accessor drives per-note rendering decisions. Excalidraw reuses
  the mechanism for detection only — it ships no widget script.
- **Service+cache pattern** `internal/chartdata` (enqueue-on-miss + 2s debounced
  reload), backed by `chart_data_cache` (`db/schema.sql:850`) and the goqite job
  `internal/case/backjob/refreshchartdata`.
- **Asset storage** `note_assets` (`db/schema.sql:435`, content-hash dedup via
  `unique(absolute_path, sha256_hash)`) served at `/_assets/` by
  `internal/localstorage` or `internal/miniostorage`.
- **Live-pull** `internal/notebus` publishes create/update/remove batches; the
  GraphQL `noteChanges` SSE subscription (`internal/graph/note_changes.go`)
  pushes them to the browser. `chartdata` triggers it via
  `PrepareLiveNotes`/`PrepareLatestNotes`.

---

## Chosen render architecture + data flow

**Render to a static asset at ingest time, served from disk/S3, swapped in live
when ready.** The browser never runs Excalidraw; it gets a plain
`<img>`/inline-SVG pointing at `/_assets/...`. There is **no client-side viewer**
— when there's no asset (no service, render error, or render pending) the page
shows the existing "not supported yet" placeholder.

```
Author saves  board.excalidraw(.md)
        │
        ▼
  noteloader/mdloader  ──► extract scene JSON
        │                   (.excalidraw → raw bytes; .excalidraw.md → unwrap)
        │                   compute scene_hash = sha256(canonical scene JSON)
        ▼
  excalidraw.Service.RenderURL(versionID, scene_hash)
        │
   ┌────┴─────────────── cache HIT (asset exists for scene_hash) ─────────────┐
   │                                                                            │
   ▼ cache MISS                                                                 ▼
  enqueue render job (goqite)                                          return asset URL
        │  (off render thread, like chartdata)
        ▼
  refreshexcalidraw.Resolve
        │  POST scene JSON → EXCALIDRAW_RENDER_URL
        ▼  receive SVG (preferred) / PNG
  store bytes as note_assets asset (content-hash)
  upsert excalidraw_render_cache(scene_hash → asset_id)
        │
        ▼  SaveExcalidrawRender → signalReload (2s debounce)
  PrepareLiveNotes / PrepareLatestNotes  ──► notebus publish ──► SSE noteChanges
        │
        ▼
  browser live-pull refetches page ──► placeholder swapped for <img src="/_assets/...">
```

Render-time output for an excalidraw page — a plain image, no data-island, no JS:

```html
<div class="excalidraw" data-scene-hash="ab12…">
  <img class="excalidraw__img" src="/_assets/na/123/board.svg" alt="…" loading="lazy">
</div>
```

When there's no asset yet, the server emits the existing "not supported yet"
placeholder (the `UnsupportedFile` quicktemplate, `views.html:527-541`) — the
same stub readers see today. This covers two cases identically:

- **Render pending** (service configured, job enqueued): the placeholder shows
  until the job completes.
- **No render possible** (`EXCALIDRAW_RENDER_URL` unset, render errored): the
  placeholder is the final state. No fallback renderer, no download-link
  requirement (linking to the raw `/board.excalidraw` file would be trivial to
  add, but it's not required — the placeholder is enough).

When the background job finishes and the debounced reload fires, the next
render (pushed via SSE live-pull) emits the `<img>` in place of the placeholder.

### Where the hook lives — two entry points

Because of the two formats, there are **two ingest interception points**, both
in the loader, not in `rendernotepage`:

1. **`.excalidraw` (raw)** — extend `registerRawFile` (loader.go:270-299).
   Today it only parses `.canvas`. Add: if path ends `.excalidraw`, parse the
   scene JSON into `nv.Excalidraw` (new field), compute `scene_hash`, ask the
   service for a render URL (enqueue on miss). The pure-`.excalidraw` page still
   reaches `rendernotepage`, so we **replace** the placeholder short-circuit
   (endpoint.go:96-103) with a real "render the excalidraw page" branch that
   emits the `<div class="excalidraw">` (and re-applies access control).
2. **`.excalidraw.md` (markdown)** — intercept in the markdown load path. The
   cleanest hook is a goldmark detector keyed off the `excalidraw-plugin`
   frontmatter (already parsed into `RawMeta`): when present, extract the scene
   from the `## Drawing` block and **replace the note body** with the
   `<div class="excalidraw">` container (the `<img>`, or the placeholder while
   pending) instead of rendering the wrapper. The `HasExcalidraw()` accessor
   rides the same mechanism as charts/mermaid, but here it only drives detection
   — there is no glue script to inject.

---

## Alternatives considered (and why rejected)

| Alternative | Why rejected |
|-------------|--------------|
| **Client-side `@excalidraw` viewer** (default or fallback) | The lib is React + ~megabytes; shipping it on any reader page is heavy and slow, and it can't serve SEO/OG (no static image). **Rejected entirely** — when there's no render service, the page just shows the "not supported yet" placeholder (owner decision: "if there's nothing to render with, just say not supported"). No glue bundle, no React dependency on reader pages. |
| **Render bytes into a SQLite blob** (like chart rows in `chart_data_cache.data_json`) | Rendered SVG/PNG can be 100s of KB–MBs. Storing in the row store bloats the DB, defeats browser/CDN caching (every view re-reads the row + re-serves via the app), and breaks the existing asset-serving fast path. **Assets win** — content-addressed, served by `/_assets/` with normal HTTP caching, dedup for free. (This is the product owner's firm "it IS an asset" decision.) |
| **Server-side headless render inside the Go process** | Pulls a headless Chromium / Node toolchain into the monolith — heavy, fragile, hard to sandbox. Externalizing it behind `EXCALIDRAW_RENDER_URL` keeps the monolith pure-Go and lets the renderer scale/upgrade independently. |
| **Synchronous render on request** | Round-trip latency on every page view; an external service hiccup blocks the page. The chartdata pattern already solved this: render off-thread, serve cache, refresh in background. |
| **Render in the markdown pipeline synchronously at load** | Load already runs on save and on startup for the whole vault; a blocking HTTP-per-drawing would stall boot and saves. Enqueue + debounced reload keeps load fast. |
| **Build on the full `template_processors.md` mechanism now** | That feature is unbuilt and tied to Jet templates + a render cache that doesn't exist (`apply_processor`, buffered-render, status: not implemented). We need a narrow scene→image path, not template post-processing. Align the **HTTP contract + host allowlist** with it; build a dedicated path first (see [Synergy](#synergy-with-template-processors)). |

---

## Backend changes

### New package: `internal/excalidraw` (parser)

Mirror `internal/obsidiancanvas`: pure parsing, no business logic.

```go
// internal/excalidraw/scene.go
package excalidraw

type Scene struct {
    Type     string          `json:"type"`     // "excalidraw"
    Version  int             `json:"version"`
    Elements []Element       `json:"elements"`
    AppState json.RawMessage `json:"appState"`
    Files    map[string]File `json:"files"`    // embedded images, base64
    raw      []byte          // canonical bytes for hashing
}

// ParseRaw decodes a pure .excalidraw JSON file.
func ParseRaw(raw []byte) (*Scene, error)

// ParseMarkdown extracts and decodes the scene JSON embedded in a
// .excalidraw.md wrapper (the ## Drawing code block; handles compressed-json).
func ParseMarkdown(raw []byte) (*Scene, error)

// Hash returns sha256 of the canonical scene JSON — the cache key.
func (s *Scene) Hash() string
```

`ParseMarkdown` handles the wrapper: find the `## Drawing` section, read the
` ```json ` or ` ```compressed-json ` fence, LZ-decompress when needed, then
delegate to the JSON decode. Unit-test both with real plugin output (a fixture
in `docs/demo/`).

### New service package: `internal/excalidraw` render orchestration

Follow the **Service Package Pattern** exactly like `chartdata` (the project's
canonical example, `docs/dev/app_patterns.md`):

- Minimal `Env interface` the package needs; `app` implements it.
- Compile-time check `var _ excalidraw.Env = (*app)(nil)` in
  `cmd/server/excalidraw.go`.
- Type named after the **domain**; embedded **anonymously** in `app` so its
  methods promote (no proxy methods). Provider method named to avoid clashing
  with the embedded field (e.g. type `Excalidraw` → method `ExcalidrawRenderURL`).
- `enqueue-on-miss` + a `reload chan struct{}` with a 2s debounce loop.

```go
// internal/excalidraw/render.go  (mirrors internal/chartdata/chartdata.go)
type Env interface {
    Logger() logger.Logger
    Now() time.Time

    // CachedRender returns the asset URL + asset id for a scene hash;
    // found=false on a miss. lastError is non-empty when the last render failed.
    CachedRender(ctx context.Context, sceneHash string) (assetURL string, lastError string, found bool, err error)
    // EnqueueRender schedules a background render of the scene JSON.
    EnqueueRender(ctx context.Context, versionID int64, sceneHash string, scene []byte) error
    // ReloadExcalidrawNotes re-renders the public note set so the freshly
    // rendered asset URL is surfaced (same chooser as chartdata).
    ReloadExcalidrawNotes(ctx context.Context) error
}

type Excalidraw struct { env Env; logger logger.Logger; reload chan struct{} }

// ExcalidrawRenderURL returns the cached asset URL for a scene, enqueuing a
// render on a miss. Returns "" while pending → renderer emits the placeholder.
func (s *Excalidraw) ExcalidrawRenderURL(versionID int64, sceneHash string, scene []byte) string

// SaveExcalidrawRender / SaveExcalidrawRenderError — job sinks; signal debounced reload.
```

The render job is wired exactly like `refreshchartdata`:

```go
// internal/case/backjob/refreshexcalidraw  (mirror of refreshchartdata)
const JobID = "render_excalidraw"
const QueueID = model.BackgroundDefaultQueue

// Resolve: POST scene JSON → EXCALIDRAW_RENDER_URL, read SVG/PNG, store as
// asset, upsert the cache row. Fetch failures are logged + recorded as
// SaveExcalidrawRenderError and return nil (so goqite does NOT retry forever —
// same rationale as refreshchartdata/resolve.go:38-57).
```

The job's `Resolve` reuses the same HTTP-fetch shape as
`refreshchartdata/resolve.go:59-92` (context timeout, `LimitReader` size cap,
status check) but POSTs the scene and expects image bytes back, then calls
`StoreRenderAsset` (writes via the storage backend's `PutAssetObject`, inserts a
`note_assets` row with the content hash) and upserts the cache row.

### New cache table

Mirror `chart_data_cache` (`db/schema.sql:850`). Key by **scene hash** (not
version) so identical scenes across notes/versions dedupe:

```sql
-- db/schema.sql  (new; remember CLAUDE.md: ASK before creating SQL migrations)
CREATE TABLE excalidraw_render_cache (
  scene_hash    text    not null,
  asset_id      integer not null references note_assets(id) on delete cascade,
  format        text    not null default 'svg',  -- 'svg' | 'png'
  theme         text    not null default '',      -- '' (neutral) | 'light' | 'dark'
  rendered_at   integer not null,
  last_error    text    not null default '',
  last_error_at integer not null default 0,
  primary key (scene_hash, theme)
);
```

`(scene_hash, theme)` PK supports rendering light+dark variants if we choose to
(see open decisions). After editing `queries.sql` run `make sqlc`.

### Model + accessor

- New field on `model.NoteView`: `Excalidraw *excalidraw.Scene` (mirror
  `Canvas *obsidiancanvas.Canvas`, note.go:284) plus the resolved
  `ExcalidrawAssetURL string` (or `""` while pending).
- New narrow accessor on `internal/templateviews/note.go` next to `HasCharts()`
  (note.go:167-183): `func (n *Note) HasExcalidraw() bool`. Per the mermaid doc,
  template-facing code uses the accessor — never `Unwrap()` the raw model.

### Render-hook wiring

Replace the `.excalidraw` placeholder branch in `endpoint.go:96-103`:

```go
// Instead of UnsupportedFileExt for .excalidraw, render the drawing page —
// but re-apply access control first (the old short-circuit skipped paywall).
if resp != nil && resp.Note != nil && strings.HasSuffix(resp.Note.Path, ".excalidraw") {
    if err != nil { /* fall through to paywall/signin handling below */ }
    else {
        dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
        // Note.Excalidraw + Note.ExcalidrawAssetURL drive the template.
        defaulttemplate.WriteRender(ctx, dtCtx)
        return nil, nil
    }
}
```

`.canvas` / `.base` keep using `unsupportedFileExt()`; only `.excalidraw` graduates.

When the asset isn't ready (render pending, errored, or no service), this branch
still renders the page but emits the existing "not supported yet" placeholder
instead of the `<img>` — i.e. it falls back to the same `UnsupportedFile`
output. **No glue script is injected** in `buildDefaultTemplateCtx`
(endpoint.go:498-515): the rendered page is a plain `<img>`, so the conditional
widget-injection list there (HasCharts / mermaid / codeblock) gains no excalidraw
entry. There is no `ExcalidrawFallbackEnabled()` flag.

---

## Frontend changes

**Minimal — and no JavaScript.** Excalidraw is served as a static image, so the
browser runs nothing: no glue bundle, no `@excalidraw` React dependency, no
data-island. This is the big simplification of the owner decision.

- **No JS bundle.** There is no `assets/excalidraw/` directory, no
  `assets/excalidraw.js` / `excalidraw.min.js`, no `esbuild` step, and no
  `//go:embed` / `package.json` entry. The rendered page is just an `<img>` (or
  inline SVG) pointing at `/_assets/...`.
- **CSS only (optional).** Add a small `.excalidraw` block to
  `assets/defaulttemplate/src/index.scss` — container sizing and
  `__img { max-width:100% }` — then `npm run defaulttemplate-css`. When there's
  no asset, the page reuses the existing `UnsupportedFile` placeholder, which
  already has styling, so even this CSS is small.

---

## Access control

The current short-circuit (`endpoint.go:96-103`) runs **above** the
`PaywallError` / `SigninWallError` branches (115-141), so raw files bypass
access control. That's fine for "not supported" placeholders, but a renderer
that **surfaces drawing content (and embedded images)** must respect it.

In the new `.excalidraw` render branch, do **not** short-circuit before access
control. Let `Resolve` (`resolve.go`) run its normal `Free` / `CanReadNote`
checks and return `PaywallError` / `SigninWallError`; only render the drawing on
`err == nil`. For `.excalidraw.md` the note already flows through the normal
markdown path, so paywall already applies — no extra work there.

Because the scene JSON never reaches the browser (only the rendered `<img>`
does), there's no data-island to leak `scene.files{}` base64 blobs into the page
source. The remaining concern is the **rendered asset itself**: a paywalled
drawing must emit the paywall instead of the `<img>`, so the `/_assets/...` URL
of the render isn't exposed on a locked page.

---

## Caching & storage

- **Cache key = scene content-hash** (`sha256` of canonical scene JSON).
  Identical/unchanged scenes dedupe; editing a note that didn't touch the
  drawing reuses the existing asset and **skips the render**.
- **Output = a `note_assets` asset** on disk (`internal/localstorage`, served at
  `/_assets/`) or S3 (`internal/miniostorage`), via `PutAssetObject` +
  `NoteAssetURL`. `note_assets` already dedups on
  `unique(absolute_path, sha256_hash)` (`db/schema.sql:435`), so two identical
  renders collapse to one object.
- **Bookkeeping** = `excalidraw_render_cache(scene_hash, theme → asset_id,
  last_error…)`, mirroring `chart_data_cache`. The service reads it on render
  (cache hit → URL; miss → enqueue), the job upserts it.
- **NOT a SQLite blob.** (Rejected above — bloats the row store, loses
  browser/CDN caching.)
- **Live swap**: the job's `SaveExcalidrawRender` signals the debounced reload
  (`signalReload` → `reloadLoop`, copy `chartdata.go:109-129`), which calls
  `PrepareLiveNotes`/`PrepareLatestNotes`. That publishes a `notebus` batch,
  which the `noteChanges` SSE subscription pushes to open browsers → the
  placeholder page refreshes and shows the `<img>`.

---

## Synergy with template processors

[template_processors.md](template_processors.md) describes the general pattern
we're instancing: **buffered render → POST to an external HTTP service at
cache-warm time → cache the result**, with a **host allowlist** as the SSRF
mitigation, processors only for repo-controlled templates. Its status is **not
implemented** (depends on `template_content_type` + a render cache).

Recommendation: **ship a narrow dedicated excalidraw path first**, but align the
contract so it can later fold into the general mechanism:

- Use a single env var `EXCALIDRAW_RENDER_URL` (one allowlisted host), matching
  the doc's allowlist principle.
- Keep the request/response shape generic (POST text/JSON in, bytes + new
  Content-Type out) so the same `callProcessor`-style helper could back both.
- Don't couple to Jet `apply_processor` or the (unbuilt) template render cache —
  excalidraw renders at note-load time, not template-execute time.

When the general processor mechanism lands, the excalidraw job's HTTP call can
be reimplemented on top of its `callProcessor` helper without changing the cache
table or the asset flow.

---

## Reference render service (external)

We design the **hook + contract + cache**; the renderer is a separate process.

**Request** (from the job):

```
POST $EXCALIDRAW_RENDER_URL
Content-Type: application/json

{ "scene": { ...excalidraw scene JSON... }, "theme": "light", "scale": 2, "format": "svg" }
```

**Response:**

```
200 OK
Content-Type: image/svg+xml         (or image/png)

<svg …>…</svg>
```

Non-200 / unreachable / non-image → the job logs, records
`SaveExcalidrawRenderError`, keeps any stale asset, and returns `nil` (no goqite
retry storm — same rationale as `refreshchartdata/resolve.go:38-45`).

A reference implementation can be a tiny Node service using
`@excalidraw/excalidraw`'s `exportToSvg` / `exportToBlob` in a headless browser
(or `resvg` for SVG→PNG). It's deliberately out of the monolith.

---

## Open decisions (with recommendations)

1. **Fallback when `EXCALIDRAW_RENDER_URL` is unset.** **RESOLVED** (owner
   decision). Show the **existing "not supported yet" placeholder** — the
   current stub at `endpoint.go:96-103` → `views.html` `UnsupportedFile`. No
   client-side viewer, no `@excalidraw` bundle, no `ExcalidrawFallbackEnabled()`
   flag. Linking to the raw `/board.excalidraw` file would be trivial if wanted,
   but it's not required: the placeholder is enough.

2. **SVG vs PNG default.**
   *Recommendation:* **SVG** (crisp at any zoom, smaller for vector-heavy
   drawings, themeable). Fall back to **PNG** only when the scene embeds raster
   images that bloat the SVG. Store `format` per cache row so a service can
   choose per scene.

3. **Theme handling: light+dark variants vs one neutral SVG.**
   *Recommendation:* start with **a single neutral SVG** (light appState,
   transparent background) for v1 — simplest, one render per scene. The
   `(scene_hash, theme)` PK already supports adding `theme=light`/`theme=dark`
   variants later; when added, the server renders both `<img>` sources and the
   template shows the one matching `trip2g_theme` via CSS (no JS, in line with
   the no-bundle decision). Don't render both up front until there's demand.

4. **Where to detect `.excalidraw.md`.**
   *Recommendation:* a goldmark detector keyed on the `excalidraw-plugin`
   frontmatter key (already in `RawMeta`), replacing the note body with the
   container — keeps it inside the existing markdown pipeline rather than adding
   a pre-parse special case. (Decide during implementation whether to do it as a
   goldmark transformer or a pre-render body rewrite.)

5. **SQL migration.** Per `CLAUDE.md`, **ask before creating** the
   `excalidraw_render_cache` migration. (Recorded here, not pre-created.)

---

## Phased implementation plan

When `EXCALIDRAW_RENDER_URL` is unset, the page shows the existing "not
supported yet" placeholder — there is no separate "fallback viewer" phase to
build.

**v1 — minimal, external service required:**
1. `internal/excalidraw` parser: `ParseRaw` + `ParseMarkdown` (+ `Hash`), with
   real-fixture tests.
2. `model.NoteView.Excalidraw` field; parse pure `.excalidraw` in
   `registerRawFile`; detect `.excalidraw.md` in the markdown path.
3. `excalidraw_render_cache` table (ask first) + sqlc queries.
4. `internal/excalidraw` service (chartdata-shaped) + `refreshexcalidraw` job +
   `cmd/server/excalidraw.go` wiring (`var _ excalidraw.Env = (*app)(nil)`,
   embed in `app`).
5. Store rendered bytes as a `note_assets` asset; key cache by scene hash.
6. Replace the `.excalidraw` placeholder branch with a real render branch
   (re-applying access control); emit `<div class="excalidraw"><img></div>` when
   the asset exists, else the existing "not supported yet" placeholder;
   `HasExcalidraw()` accessor; small SCSS.
7. Live-pull swap via debounced reload → notebus → SSE.
8. Demo note + e2e fixture (a `/board.excalidraw` page, mirroring the mermaid
   demo at `docs/demo/mermaid.md` + `e2e/vault.spec.js`).

**v2 — enhancements:**
9. Light/dark variants (`theme` column already in the PK).
10. PNG fallback for raster-heavy scenes; configurable scale.
11. Fold the HTTP call onto the general `template_processors` mechanism once it
    lands.

---

## File touchpoints

| Path | Change |
|------|--------|
| `internal/excalidraw/scene.go` | **new** — `Scene` struct, `ParseRaw`, `ParseMarkdown`, `Hash` |
| `internal/excalidraw/render.go` | **new** — service (Env, `Excalidraw`, `ExcalidrawRenderURL`, save sinks, debounced reload) |
| `internal/excalidraw/*_test.go` | **new** — parser + service tests (moq mocks) |
| `internal/case/backjob/refreshexcalidraw/{job,resolve}.go` | **new** — goqite job: POST scene → render service, store asset, upsert cache (mirror `refreshchartdata`) |
| `cmd/server/excalidraw.go` | **new** — `var _ excalidraw.Env = (*app)(nil)`; `CachedRender`/`EnqueueRender`/`ReloadExcalidrawNotes`/storage glue |
| `cmd/server/main.go` / `boot.go` | wire `excalidraw.New(a)` + embed `*excalidraw.Excalidraw` in `app`; read `EXCALIDRAW_RENDER_URL` |
| `internal/mdloader/loader.go` | `registerRawFile` (270-299): parse `.excalidraw` scene → `nv.Excalidraw`, compute hash, resolve render URL |
| `internal/mdloader/*` | `.excalidraw.md` detector (goldmark) — unwrap scene, emit container, set the per-note signal |
| `internal/model/note.go` | new fields `Excalidraw *excalidraw.Scene`, `ExcalidrawAssetURL string` (near `Canvas`, 284) |
| `internal/templateviews/note.go` | new accessor `HasExcalidraw()` (near `HasCharts()`, 167-183) |
| `internal/case/rendernotepage/endpoint.go` | replace `.excalidraw` placeholder (96-103) with render branch + access control (no glue injection in `buildDefaultTemplateCtx`, 498-515) |
| `internal/defaulttemplate/views.html` | render the `.excalidraw` page (`<div class="excalidraw">` + `<img>`, else the existing `UnsupportedFile` placeholder); regenerate `views.html.go` (`go generate ./internal/defaulttemplate/...`) |
| `db/schema.sql` + `internal/db/queries.sql` | **new** `excalidraw_render_cache` table + queries (ask first; `make sqlc`) |
| `assets/defaulttemplate/src/index.scss` | small `.excalidraw` styles; `npm run defaulttemplate-css` |
| `docs/demo/*.excalidraw(.md)` + `e2e/vault.spec.js` | demo note + e2e fixture |
