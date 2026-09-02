# Rendering Obsidian Canvas (.canvas)

**TL;DR.** `.canvas` files are **already ingested end-to-end** — they sync, store
as raw text in `note_versions.content`, get a URL (today the extension URL
`/board.canvas`), are parsed into `model.NoteView.Canvas` at load time, and reach
the page endpoint. The **only** missing piece is rendering: today the endpoint
short-circuits raw files to a "Canvas files are not supported yet" placeholder.
This is a **render-layer feature, not a greenfield build.**

**URL decision (locked).** `board.canvas` is served at the **pretty URL `/board`**
(strip `.canvas`, derive the permalink the same way markdown notes do), with
`/board.canvas` kept as a working **alias** so existing links still resolve.
Canonical = the pretty URL. **Collision rule:** if a markdown note already owns
`/board` (e.g. `board.md` + `board.canvas`), the **markdown/content note wins** the
pretty URL; the canvas stays reachable at `/board.canvas` and a load-time warning is
emitted. See §7 and decision #1.

The plan: serialize the parsed canvas into a `<script type="application/json">`
data-island next to a `<div>` container (the chart/mermaid widget pattern), and
ship a tiny vanilla-TS glue bundle (`assets/canvas/`) that draws a faithful
**spatial board** with hand-rolled pan/zoom — no heavy graph library on reader
pages. Two prerequisites gate everything: (1) **extend the parser to keep node
geometry** (`x/y/width/height`) and edge sides/colors — the JSON has them but the
current `Node` struct drops them; (2) **re-apply access control** — the raw-file
short-circuit currently runs before the paywall/signin checks, so a renderer that
surfaces note content must re-add `Free`/`CanReadNote`.

---

## 1. What Canvas is, and how it'll be used on a trip2g site

Obsidian Canvas is an infinite spatial whiteboard. The `.canvas` file is JSON
([JSON Canvas spec](https://jsoncanvas.org)):

- **nodes** — each has `id`, `type` (`text` | `file` | `link` | `group`),
  spatial geometry `x`, `y`, `width`, `height`, an optional `color`, and
  type-specific payload:
  - `text` → markdown `text`
  - `file` → `file` (a vault-relative path to another note or an image)
  - `link` → `url` (external URL)
  - `group` → a labelled bounding box (`label`) that visually contains others
- **edges** — directed connectors: `id`, `fromNode`, `toNode`, optional
  `fromSide`/`toSide` (`top`|`right`|`bottom`|`left`), `label`, `color`.

On a published trip2g site a canvas becomes a **visual board page**: the author
arranges notes, images, links and free text spatially, and the reader pans/zooms
around it. `file` nodes embed the *rendered* target note inline (reusing the same
embed machinery that powers `![[note]]`), so a canvas works as a curated
"map of content" or a visual landing page.

## 2. Current state in trip2g — ingested, parsed, but not rendered

Everything up to rendering already works:

| Stage | Where | Status |
|-------|-------|--------|
| Sync whitelist (push) | `internal/case/pushnotes/resolve.go:27-32` (`allowedExtensins`) | ✅ `.canvas` allowed |
| Raw-file ingest | `internal/mdloader/loader.go:263` `isRawFile`, `:270` `registerRawFile` | ✅ stores `Content` verbatim, `RawMeta={}`, `Permalink="/"+path` (keeps `.canvas`) |
| Canvas parse at load | `loader.go:286-292` → `obsidiancanvas.Parse` → `nv.Canvas` | ✅ parsed into `model.NoteView.Canvas` (`note.go:282-284`) |
| Reaches page endpoint | `internal/case/rendernotepage/endpoint.go:96-103` | ✅ but short-circuits to placeholder |
| **Render** | `views.html` `UnsupportedFile` func (`:527-541`) | ❌ **stub only** |

The parser (`internal/obsidiancanvas/canvas.go`) is **already proven end-to-end**
by the Telegram canvas-bot (`internal/case/handletgcanvasupdate`), which walks the
graph (`Entry()`, `EdgesFrom()`, `Node()`) and resolves `file` nodes to rendered
HTML via `renderFileNode` → `RenderNoteHTML` (`render.go:41-57`).

### The critical parser gap: geometry is dropped

The current `Node` struct (`canvas.go:13-20`) keeps **only**
`id/type/text/file/url/color`. The `.canvas` JSON carries `x/y/width/height` on
every node and `fromSide/toSide/color` on every edge — **`encoding/json` silently
discards them** because there are no struct fields. The Telegram bot never needed
geometry (it renders a linear menu), but a spatial board cannot exist without it.
**This is the single most important prerequisite.** See §5.1 (Step Zero).

### The stub today

```html
{# views.html:527-541 #}
{% func UnsupportedFile(ctx *Ctx) %}
  ...
  {% if ctx.UnsupportedFileExt == ".canvas" %}
  <p>Canvas files are not supported yet.</p>
  ...
{% endfunc %}
```

Reached from `endpoint.go:96-103`:

```go
if ext := unsupportedFileExt(resp.Note.Path); ext != "" {
    dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
    dtCtx.UnsupportedFileExt = ext
    defaulttemplate.WriteRender(ctx, dtCtx)
    return nil, nil
}
```

## 3. Chosen render architecture: JSON data-island + client board

We follow the **chart/mermaid widget pattern** (see `docs/dev/mermaid.md`): the
server emits a static container plus a JSON data-island; a tiny glue bundle
renders it client-side, lazy-loading nothing heavy.

Concretely, the server produces (inside the page `<main>`):

```html
<div class="canvas-board" data-canvas></div>
<script class="canvas-board__data" type="application/json">
{ "nodes": [ {"id":"a","type":"text","x":0,"y":0,"width":250,"height":120,
              "color":"4","text":"<p>rendered <em>markdown</em></p>"},
             {"id":"b","type":"file","x":400,"y":0,"width":300,"height":200,
              "url":"/some/note","html":"<rendered note HTML>"},
             {"id":"c","type":"file","x":0,"y":300,"width":200,"height":200,
              "img":"/_assets/abcd1234.png"} ],
  "edges": [ {"from":"a","to":"b","fromSide":"right","toSide":"left",
              "label":"see also","color":"6"} ] }
</script>
<noscript>...SSR fallback list...</noscript>  <!-- crawler/no-JS fallback -->
```

The glue (`assets/canvas/`) reads the island, builds absolutely-positioned
node `<div>`s inside a transform-wrapped layer, draws edges as SVG paths, and
wires pan/zoom/fit-to-screen.

### Data flow

```
.canvas file (synced)
   │  loader.go:registerRawFile → obsidiancanvas.Parse (geometry now kept)
   ▼
model.NoteView.Canvas  ── x/y/w/h, sides, colors
   │
   ▼  rendernotepage.Resolve  (RE-APPLY Free / CanReadNote — see §8)
   │
   ▼  endpoint.go  (replace the short-circuit branch)
   │     canvasdata.Build(canvas, nvs, env)   ← NEW: pre-resolve file nodes
   │       • text  → render markdown to HTML
   │       • file  → GetByPath(node.File) → note.HTML  (access-checked)
   │              or image → AssetReplaces[...].URL  (/_assets/)
   │       • link  → url passthrough
   ▼
views.html  CanvasBoard(ctx)  →  <div data-canvas> + JSON island + <noscript>
   │  + buildDefaultTemplateCtx appends /assets/canvas.js when note.HasCanvas()
   ▼
browser:  assets/canvas/src/index.ts
   │  parse island → position nodes → draw edges → pan/zoom/fit
   ▼
spatial board (theme-aware via window.trip2g_theme_listeners)
```

**Why this approach.** It matches an existing, documented seam, keeps SSR HTML
crawlable, makes the heavy interactivity progressive enhancement, and reuses the
proven embed/asset machinery for `file` nodes. The visual board only needs the
client when geometry-driven layout is wanted; the `<noscript>` ordered list still
conveys the content.

## 4. Alternatives considered (and why rejected for the primary path)

| Alternative | Idea | Verdict |
|-------------|------|---------|
| **(2) Mermaid-transpile "lite"** | Convert canvas → a mermaid `flowchart` and reuse the existing mermaid widget. | Loses geometry/grouping/spatial intent; mermaid auto-layouts. Keep as an **optional fallback** for geometry-less or trivially-linear canvases (a frontmatter opt-in), not the default. |
| **(3) Interactive graph lib (cytoscape)** | Pull cytoscape onto reader pages for layout + interaction. | **Rejected.** Heavy (~hundreds of KB), it's force-layout-oriented (we already have exact coordinates), and it's currently **admin-only** (`assets/ui/admin/noteview/graph`). Do not ship it to readers. |
| **(4) Pre-rendered raster image** | Server rasterizes the board to PNG. | Needs a headless renderer; not interactive; no embedded live note HTML. **Only** worth it later for OG/thumbnail images. |

## 5. Backend changes

### 5.1 Step Zero — extend the parser (additive, non-breaking)

`internal/obsidiancanvas/canvas.go` — add geometry fields. **Additive only**, so
the Telegram consumer (`handletgcanvasupdate`) keeps compiling unchanged:

```go
type Node struct {
    ID     string `json:"id"`
    Type   string `json:"type"`
    Text   string `json:"text,omitempty"`
    File   string `json:"file,omitempty"`
    URL    string `json:"url,omitempty"`
    Color  string `json:"color,omitempty"`
    // NEW — JSON Canvas geometry, currently dropped:
    X      int    `json:"x"`
    Y      int    `json:"y"`
    Width  int    `json:"width"`
    Height int    `json:"height"`
    Label  string `json:"label,omitempty"` // for type="group"
    Subpath string `json:"subpath,omitempty"` // file#heading anchor
}

type Edge struct {
    ID       string `json:"id"`
    FromNode string `json:"fromNode"`
    ToNode   string `json:"toNode"`
    Label    string `json:"label,omitempty"`
    // NEW:
    FromSide string `json:"fromSide,omitempty"` // top|right|bottom|left
    ToSide   string `json:"toSide,omitempty"`
    Color    string `json:"color,omitempty"`
}
```

No change to `Parse`, the index maps, or any existing method — pure data
widening. Verify the bot still builds (`go test ./internal/case/handletgcanvasupdate/...`).

### 5.2 New package: `internal/canvasview` (build the data island)

A pure builder (like `templateviews`, no IO state) that turns a parsed
`*obsidiancanvas.Canvas` + the note set into a serializable island. Pre-resolving
`file` nodes here mirrors the bot's `renderFileNode` (`render.go:41-57`):

```go
package canvasview

type Node struct {
    ID, Type           string
    X, Y, Width, Height int
    Color              string `json:",omitempty"`
    Text               string `json:",omitempty"` // rendered HTML for text nodes
    HTML               string `json:",omitempty"` // resolved note HTML for file nodes
    URL                string `json:",omitempty"` // permalink (file→note) or external (link)
    Img                string `json:",omitempty"` // /_assets/ URL for image file nodes
    Label              string `json:",omitempty"` // group label
}

// Build resolves file nodes to permalink + rendered HTML (or image URL),
// renders text nodes' markdown, and serializes geometry + edges.
// canRead gates embedded note content (access control, §8).
func Build(c *obsidiancanvas.Canvas, nvs *model.NoteViews,
           canRead func(*model.NoteView) bool, renderMD func(string) string) Board
```

File-node resolution reuses the existing seam exactly:
- `nvs.GetByPath(node.File)` (the same lookup `renderEmbed` uses,
  `link_renderer.go:219`); fall back to a `PathMap` suffix match like the bot's
  `findNoteByFile` (`render.go:59-73`) for vault-prefix paths.
- If the target is a note: take `note.Permalink` + `note.HTML` (already rendered
  in memory — same as `renderEmbed` wrapping `note.HTML`, `link_renderer.go:245`).
  **Gate on `canRead`**: if the embedded note is paywalled and the viewer can't
  read it, emit only its title/permalink, not the HTML.
- If the target is an image (`image.IsMediaExtension`): use
  `note.AssetReplaces[node.File].URL` (the `/_assets/` or S3 URL,
  `model.NoteAssetReplace.URL`, `note.go:46-54`).

Text-node markdown: render via the shared markdown→HTML path so `**bold**`,
links, etc. work. (v1 may ship plain-escaped text and upgrade to full markdown in
v1.1 — see phasing.)

### 5.3 Wire the render hook (`endpoint.go`)

Replace the unconditional placeholder branch (`endpoint.go:96-103`) so `.canvas`
renders the board while `.base`/`.excalidraw` keep the stub:

```go
if resp != nil && resp.Note != nil {
    if ext := unsupportedFileExt(resp.Note.Path); ext != "" {
        // RE-APPLY access control here — the short-circuit ran before paywall (§8).
        dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
        if ext == ".canvas" && resp.Note.Canvas != nil {
            dtCtx.Canvas = canvasview.Build(resp.Note.Canvas, resp.Notes.Unwrap(),
                                            env.canReadFn(...), env.renderMD)
        } else {
            dtCtx.UnsupportedFileExt = ext // .base / .excalidraw stub
        }
        defaulttemplate.WriteRender(ctx, dtCtx)
        return nil, nil
    }
}
```

### 5.4 Accessor + Ctx field

- `internal/templateviews/note.go` — add a narrow accessor next to `HasCharts()`
  (`note.go:169`) / `HasCodeLanguage()` (`:176`):

  ```go
  // HasCanvas reports whether the note is a parsed .canvas board.
  func (n *Note) HasCanvas() bool { return n.nv.Canvas != nil }
  ```
  Use this in `buildDefaultTemplateCtx` to conditionally append `canvas.js`
  (§6 / mirrors mermaid wiring at `endpoint.go:505-515`). Do **not** `Unwrap()`
  the raw `*model.NoteView` into template code.

- `internal/defaulttemplate/template.go` — add `Canvas *canvasview.Board` to the
  `Ctx` struct (alongside `UnsupportedFileExt`, `template.go:71`).

### 5.5 Template func (`views.html`)

Add a `CanvasBoard(ctx *Ctx)` quicktemplate func and branch to it where the
`UnsupportedFile` branch lives (`views.html:92-93`). It emits the container, the
JSON island, and the `<noscript>` SSR fallback (an ordered list of node
text/links — see §3). **Quicktemplate is `{%s %}`/`{%= %}`, not Go templates** —
after editing, run `go generate ./internal/defaulttemplate/...` and commit the
regenerated `views.html.go` (the generated `StreamUnsupportedFile` lives at
`views.html.go:2314-2348`; the new func will sit next to it).

## 6. Frontend changes

### 6.1 Glue bundle `assets/canvas/` (mirror `assets/mermaid/`)

- `src/index.ts` → `assets/canvas.js` — the **only** bundle (no heavy lib, so no
  `lib.ts`/`canvas.min.js` split). Responsibilities:
  1. Find `[data-canvas]` and its sibling `script.canvas-board__data`; bail if
     absent (cheap no-op on non-canvas pages — same guard as mermaid's
     `if (blocks.length === 0) return`).
  2. Compute the bounding box of all node geometries; create a transform layer.
  3. For each node, create an absolutely-positioned `<div>` at `x/y/width/height`,
     inject `text`/`html` (sanitized server-side already), apply the canvas
     color palette class.
  4. Draw edges as SVG `<path>`s between node sides (`fromSide`/`toSide`), with
     optional `label` and `color`.
  5. **Hand-rolled pan/zoom** (pointer drag to pan, wheel/pinch to zoom, a
     "fit-to-screen" button). No external dependency — keep the bundle tiny.
  6. Theme: register a `window.trip2g_theme_listeners` callback (exactly like
     `assets/mermaid/src/index.ts:56-61`) to recolor on light/dark toggle.

### 6.2 Build wiring

- `assets/canvas/esbuild.browser.mjs` — copy `assets/mermaid/esbuild.browser.mjs`
  (IIFE, `target: es2020`, `minify`), but **one** output (`canvas.js`).
- `package.json` — add a per-widget script next to the others
  (`"mermaid"`/`"chart"`/`"codeblock"`, `package.json:18-20`):

  ```json
  "canvas": "node assets/canvas/esbuild.browser.mjs"
  ```

- `assets/embed.go` — add `canvas.js` to the `//go:embed` list
  (`embed.go:8`, alongside `mermaid.js`). The built `canvas.js` is a committed
  artifact, like `chart.js`/`mermaid.js`.

- `buildDefaultTemplateCtx` (`endpoint.go:505-515`) — append the glue per note:

  ```go
  if note.HasCanvas() {
      jsURLs = append(jsURLs, env.AssetURL("/assets/canvas.js"))
  }
  ```
  `AssetURL` adds the content-hash cache-bust (`cmd/server/assets.go:128-160`,
  `?h=<sha256[:8]>`).

### 6.3 CSS

Canvas board styles go in `assets/defaulttemplate/src/index.scss` (the board
container, node cards, the color palette, edge SVG, controls). Compile with
`npm run defaulttemplate-css` (`package.json:15`). Keep node colors mapped from
the JSON Canvas 1–6 palette to CSS custom properties so the theme listener can
swap them.

## 7. URL & routing — pretty `/board`, with the extension URL as an alias

**Decision (locked): a `.canvas` is served at the pretty URL** (`board.canvas` →
`/board`), not at the extension URL. The extension URL `/board.canvas` is kept as a
working alias so existing links don't break. Canonical = the pretty URL.

### How permalinks are derived and looked up today

- **Raw files** (`registerRawFile`, `loader.go:270-299`) set
  `Permalink = "/" + src.Path` **with the extension** (`loader.go:296`) — so today a
  canvas lands at `/board.canvas`.
- **Markdown notes** strip `.md` and transliterate via
  `NoteView.PreparePermalink` (`note.go:439-493`): it removes the `.md` suffix
  (`:450-452`), normalizes each path segment (`normalizeURLPart`), sets
  `Permalink`/`PermalinkOriginal`, and records `AlternatePermalinks` (other
  transliteration variants) for 301 redirects.
- **Registration** (`NoteViews.RegisterNote`, `note.go:1338-1362`) writes the note
  into `nv.Map` under `Permalink`, `PermalinkOriginal`, and each
  `AlternatePermalinks` value. `nv.Map` is **last-writer-wins** (the existing
  alternate-collision comment at `:1344-1351` confirms it). `PathMap` is keyed by
  `note.Path`.
- **Lookup** at request time: `resolveNote` (`resolve.go:507-527`) tries
  `GetByRoute` (custom-domain / alias routes) and then falls back to
  `GetByPath(path)` — which, despite its name, reads `nv.Map[path]` (the **permalink**
  map, `note.go:1315-1322`). So a URL resolves to a note iff that exact string is a
  key in `nv.Map`.

### The change

For `.canvas`, derive the permalink **the same way markdown notes do** instead of the
raw `"/" + path`:

1. **Strip `.canvas` and transliterate.** Extend `PreparePermalink`'s extension-strip
   (`note.go:450-452`, currently `.md`-only) to also strip `.canvas`, then have
   `registerRawFile` call `PreparePermalink(...)` for canvas files (reusing the
   markdown derivation) rather than setting `Permalink = "/" + src.Path`. Result:
   `board.canvas` → `Permalink = /board`.
2. **Register the extension URL as an alias.** Keep `/board.canvas` reachable by also
   registering it in `nv.Map` (an extra alias key, alongside the derived `/board`), so
   existing links to the extension URL still resolve. Both URLs work; canonical is the
   pretty URL.

This is an alias, not the existing `AlternatePermalinks` 301 mechanism: an
`AlternatePermalinks` entry triggers a 301 to the canonical URL (`endpoint.go:69-77`),
whereas here both URLs should **serve**. (A canonical-link or optional 301 from the
extension URL to `/board` can be layered on later if desired.)

### Collision rule (deterministic, non-clobbering)

If a markdown note and a canvas derive the **same** pretty permalink — `board.md` and
`board.canvas` both → `/board` — the **markdown/content note wins** the pretty URL; the
canvas stays reachable only at `/board.canvas`; emit a load-time warning.

This falls out of the existing registration order and `nv.Map` semantics — no special
casing of the lookup is required:

- Raw files register **first** (Step 1a, `loader.go:145-154`), markdown notes register
  **later** (Step 3, `loader.go:214-231`), both through `RegisterNote`.
- `nv.Map` is last-writer-wins, so the later `board.md` registration overwrites
  `nv.Map["/board"]` — the **markdown note ends up owning `/board`**, exactly the
  desired precedence.
- The canvas's separately-registered alias `nv.Map["/board.canvas"]` is **not**
  touched by the markdown note, so the canvas remains reachable there.

To make this robust rather than incidental, `registerRawFile` (or the load step) should
**detect the collision explicitly** — check whether the derived pretty permalink already
maps to a content note (or is claimed later) — and, on collision, **not** register the
pretty key for the canvas (only the `.canvas` alias), plus emit a warning via
`ldr.log.Warn` (same pattern as the existing `"canvas parse failed"` warning,
`loader.go:289`) or `page.AddWarning` (`loader.go:222`). Do not rely solely on
registration order for correctness; treat last-writer-wins as the tiebreaker, the
explicit check as the guarantee.

## 8. Access control — closing the short-circuit gap

**This is a real gap, not a theoretical one.** The raw-file short-circuit
(`endpoint.go:96-103`) runs **before** the onboarding/signin/paywall branches
(`:105-150`), and `Resolve` itself never runs the `Free`/`CanReadNote` checks for
the canvas page in a way the short-circuit honors — today a `.canvas` URL renders
the (contentless) placeholder regardless of `err`, so it doesn't matter. Once the
renderer **surfaces content** (the canvas's own structure and, crucially,
embedded note HTML for `file` nodes), it must re-apply access control:

1. **The canvas page itself** — honor the same checks `Resolve` produces for
   regular notes: `SigninWallError` (`resolve.go:279-285`), `Free` + guest →
   paywall (`:288-290`), `CanReadNote` (`:309-316`). In the short-circuit, if
   `err` is a `*SigninWallError`/`*PaywallError`, render those walls instead of
   the board (reuse the existing branches, or move the short-circuit *after*
   them).
2. **Embedded `file` nodes** — `canvasview.Build` must check `CanReadNote` per
   embedded note and **omit HTML** for notes the viewer can't read (emit
   title + permalink only). A canvas must not become a paywall bypass.

Recommendation: **move the canvas short-circuit below the paywall/signin
handling** so the page-level checks are reused verbatim, and pass a
`canRead func(*NoteView) bool` into `canvasview.Build` for per-embed gating.

## 9. Caching / storage

v1 needs **no new cache**. The canvas is parsed once at note-load
(`registerRawFile`), embedded note HTML is already rendered in memory, and asset
URLs come from existing `AssetReplaces`. The data island is built per request but
the inputs are all in-memory — cheap.

If profiling later shows the per-request `canvasview.Build` (markdown rendering of
text nodes + HTML assembly) is hot, adopt the **chartdata service pattern**
(`internal/chartdata`): a minimal `Env` interface embedded anonymously in `app`
(`var _ chartdata.Env = (*app)(nil)`), a cache keyed by `version_id`, rebuilt on
note reload. Not needed for v1; record as a future option.

## 10. Open decisions (with recommendations)

Decision #1 is **resolved** (recorded below); #2–#5 remain open.

| # | Decision | Recommendation |
|---|----------|----------------|
| 1 | **Pretty URL `/board` vs extension URL `/board.canvas`** | **DECIDED — pretty URL.** Serve at `/board` (derive the permalink like markdown notes do, not the raw `"/"+path` at `loader.go:296`); keep `/board.canvas` as a working **alias**; canonical = pretty URL. **Collision rule:** if a markdown note also maps to `/board`, the **content note wins** `/board` and the canvas stays at `/board.canvas` (load-time warning). This is **v1 wiring**, not a deferral. Full mechanics in §7. |
| 2 | **Embedding `![[board.canvas]]` inside a note** | Currently skipped — `renderEmbed` bails when `note.HTML` is empty (`link_renderer.go:235-238`), and raw files have empty `HTML`. **Defer.** Later: give canvas notes a small server-rendered HTML preview (title + node count, or a static thumbnail) so embeds resolve. |
| 3 | **Text-node markdown** | Render full markdown via the shared path. v1 may ship escaped-plaintext to de-risk, upgrade in v1.1. **Recommend full markdown** if the shared renderer is easy to call from `canvasview`. |
| 4 | **Mermaid-transpile fallback** | Offer as an **opt-in** (`canvas_mode: flowchart` frontmatter) for simple/geometry-less canvases. Not default. |
| 5 | **Group nodes** | Render as a labelled bounding box behind contained nodes (z-index below, semi-transparent fill). Low effort, high fidelity. Include in v1 if time allows, else v1.1. |

## 11. Phased implementation plan

**v1 — minimal faithful board**
1. **Step Zero:** widen `obsidiancanvas.Node`/`Edge` with geometry/sides/colors
   (additive); confirm `handletgcanvasupdate` tests pass.
2. **Pretty-URL wiring (§7):** extend `PreparePermalink` to also strip `.canvas`
   (`note.go:450-452`) and have `registerRawFile` derive the canvas permalink via
   `PreparePermalink` instead of `"/"+path` (`loader.go:296`); register `/board.canvas`
   as an alias; add the collision check (content note wins `/board`, canvas keeps
   `/board.canvas`, load-time warning).
3. `internal/canvasview`: `Build` — geometry + edges + `file`-node resolution
   (note HTML / image URL) + `text` nodes (escaped or markdown) + `link` URLs.
4. `endpoint.go`: replace the `.canvas` short-circuit with board build; **move it
   below paywall/signin** and re-apply access control (§8).
5. `templateviews.HasCanvas()` accessor; `Ctx.Canvas` field.
6. `views.html` `CanvasBoard` func + `<noscript>` SSR fallback → `go generate`.
7. `assets/canvas/` glue (positioned nodes, SVG edges, pan/zoom/fit, theme
   listener) + esbuild + `package.json` script + `embed.go` + conditional
   `canvas.js` in `buildDefaultTemplateCtx`.
8. SCSS for board/nodes/edges/palette.
9. Demo canvas in `docs/demo/` + e2e test (board renders at the pretty URL,
   `/board.canvas` alias still resolves, `canvas.js` loads only on canvas pages —
   mirror the mermaid e2e in `e2e/vault.spec.js`).

**v1.1 — enhancements**
- Full markdown in text nodes (if v1 shipped plaintext).
- Group-node bounding boxes (decision #5).
- Optional 301 from the `/board.canvas` alias to the canonical `/board` (decision #1).
- Per-embed access-control polish + paywalled-embed placeholders.

**v2 — optional**
- Mermaid-transpile `flowchart` fallback mode (decision #4).
- Canvas embed previews so `![[board.canvas]]` resolves (decision #2).
- Server-rendered raster for OG/thumbnail images (alternative #4).
- `canvasview` caching via the chartdata service pattern if profiling demands it.

## 12. File touchpoints

| Path | Change |
|------|--------|
| `internal/obsidiancanvas/canvas.go` | **Step Zero:** add `X/Y/Width/Height/Label/Subpath` to `Node`, `FromSide/ToSide/Color` to `Edge` (additive) |
| `internal/model/note.go` | **Pretty URL (§7):** extend `PreparePermalink`'s extension-strip (`:450-452`, today `.md`-only) to also strip `.canvas` so the canvas permalink derives like a markdown note |
| `internal/mdloader/loader.go` | **Pretty URL (§7):** in `registerRawFile` (`:270-299`) derive the canvas permalink via `PreparePermalink` instead of `"/"+src.Path` (`:296`); also register the `/board.canvas` **alias** in `nv.Map`; add the collision check (content note wins `/board`, canvas keeps `/board.canvas`) + load-time warning (`ldr.log.Warn`, cf. `:289`) |
| `internal/canvasview/` (new) | `Build` — serialize geometry+edges, pre-resolve `file` nodes (note HTML / `/_assets/` image URL), render text nodes; access-gated |
| `internal/case/rendernotepage/endpoint.go` | Replace `.canvas` short-circuit (`:96-103`) with board build; move below paywall/signin; append `canvas.js` in `buildDefaultTemplateCtx` (`:505-515`) |
| `internal/case/rendernotepage/resolve.go` | Ensure `Free`/`CanReadNote` apply to canvas pages (reuse `:279-316`); pass `canRead` to `canvasview.Build`. No change needed to `resolveNote`'s lookup order (`:507-527`): both `/board` and `/board.canvas` resolve via `GetByPath` once registered in `nv.Map` |
| `internal/templateviews/note.go` | `HasCanvas()` accessor (next to `HasCharts()`/`HasCodeLanguage()`, `:169-185`) |
| `internal/defaulttemplate/template.go` | `Ctx.Canvas *canvasview.Board` field (near `:71`) |
| `internal/defaulttemplate/views.html` | New `CanvasBoard(ctx)` func + branch (replacing/alongside `:527-541`); `<noscript>` fallback |
| `internal/defaulttemplate/views.html.go` | Regenerated via `go generate ./internal/defaulttemplate/...` — commit together |
| `assets/canvas/src/index.ts` (new) | Glue: positioned nodes, SVG edges, pan/zoom/fit, theme listener |
| `assets/canvas/esbuild.browser.mjs` (new) | Single-output esbuild config (copy mermaid's) |
| `assets/canvas.js` (new, built) | Committed artifact |
| `assets/embed.go` | Add `canvas.js` to `//go:embed` (`:8`) |
| `package.json` | Add `"canvas": "node assets/canvas/esbuild.browser.mjs"` script (`:18-20`) |
| `assets/defaulttemplate/src/index.scss` | Board/node/edge/palette styles → `npm run defaulttemplate-css` |
| `docs/demo/*.canvas` + `e2e/vault.spec.js` | Demo canvas + e2e (board renders at the pretty `/board` URL; `/board.canvas` alias still resolves; `canvas.js` loads only on canvas pages) |
