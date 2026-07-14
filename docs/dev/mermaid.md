# Mermaid diagrams & conditional widget loading

Mermaid renders ` ```mermaid ` fenced code blocks as diagrams, like Obsidian.
Implementing it introduced a general mechanism — **per-note, backend-decided
widget script loading** — that any future per-language client widget reuses.

Diagrams render with one of two engines by type — beautiful-mermaid for
flowchart/state/sequence/class/ER/xychart, mermaid.js as the fallback for
everything else — both enhanced with pan/zoom/fullscreen/export from
`@mostlylucid/mermaid-enhancements`. See "The mermaid bundle — dual renderer"
below.

## Two parts

1. **Conditional widget loading** (backend) — the server knows, at render time,
   which client widgets a page needs, and emits exactly those `<script>` tags.
2. **The mermaid bundle** (frontend) — a tiny glue that routes each diagram to
   the right engine and lazily loads it, mirroring the datachart/echarts split.

No `mermaid` HTML is produced server-side: goldmark already renders the fenced
block as `<pre><code class="language-mermaid">…</code></pre>`. The glue takes it
from there in the browser.

## Conditional widget loading

### Why backend-decided, not a client loader

A client-side loader (one `loader.js` that inspects the DOM and injects the
widgets it finds) adds a request waterfall: HTML → run loader → inject glue →
download glue → … The browser's preload scanner can't see dynamically injected
scripts. Since the server already parses every note at render time, it knows
which widgets are needed and can put the `<script defer>` tags straight into the
initial HTML — the preload scanner fetches them immediately, in parallel, with
no JS-execution gate.

### How it works

- `model.NoteView.CodeLanguages` is a `map[string]bool` set of the languages of
  every fenced code block in the note, lowercased. It is built by
  `extractCodeLanguages()` during the AST walk (`internal/model/note_codelang.go`),
  called alongside `extractCharts()` from `note.go`. `HasCodeLanguage(lang)` is
  the nil-safe lookup.
- `templateviews.Note` exposes narrow accessors `HasCharts()` and
  `HasCodeLanguage(lang)` that delegate to the model. Use these — do **not**
  `Unwrap()` the raw `*model.NoteView` into template-facing code (the wrapper
  exists to keep model system fields out of the templater).
- `rendernotepage.buildDefaultTemplateCtx` (`endpoint.go`) appends widget script
  URLs to `JSURLs` based on the note:
  - `chart.js`   when `note.HasCharts()`
  - `mermaid.js` when `note.HasCodeLanguage("mermaid")`
- `app.UserJSURLs()` now returns only the **core** bootstrap scripts
  (`defaulttemplate.js`, the $mol user app); widgets are appended per note.
- `app.AssetURL(path)` exposes the cache-busting hashing (`assetURL`) so each
  widget URL carries its own content hash — granular cache busting, no shared
  hash.

### Adding a new per-language widget

1. Build a glue bundle under `assets/<name>/` (see below) → `assets/<name>.js`.
2. Add it to the `//go:embed` list in `assets/embed.go`.
3. In `buildDefaultTemplateCtx`, append it when the feature is present, e.g.
   `if note.HasCodeLanguage("plantuml") { jsURLs = append(jsURLs, env.AssetURL("/assets/plantuml.js")) }`.

That's the whole wiring — no client loader, no template changes.

## The mermaid bundle — dual renderer

`assets/mermaid/` mirrors `assets/chart/`, but renders each diagram with one of
two engines depending on its type:

- **[beautiful-mermaid](https://github.com/lukilabs/beautiful-mermaid)** — a
  synchronous, from-scratch renderer (`renderMermaidSVG(text, options) → string`,
  no async, no DOM deps) for 6 diagram types: flowchart (`graph`/`flowchart`),
  state (`stateDiagram`/`stateDiagram-v2`), sequence (`sequenceDiagram`), class
  (`classDiagram`), ER (`erDiagram`), and XY charts (`xychart-beta`). Prettier
  output than mermaid.js's default theme.
- **mermaid.js** — the reference implementation, kept as the fallback for every
  diagram type beautiful-mermaid doesn't support (`gantt`, `pie`, `gitGraph`,
  `mindmap`, `timeline`, `journey`, `sankey`, `C4Context`, `quadrantChart`, …)
  and for any beautiful-mermaid parse failure (caught and retried on this path).

`src/index.ts` → `assets/mermaid.js` (the glue) does the routing: it finds
`pre > code.language-mermaid`, converts each to `<div class="mermaid">`, and
sorts blocks into the two engines by detecting the diagram type from the first
non-empty, non-`%%`-comment line's leading keyword (`firstKeyword()` /
`BEAUTIFUL_KEYWORDS` in `index.ts`). Each engine is its own lazy chunk, loaded
only if a block on the page needs it:

- `src/lib.ts` → `assets/mermaid.min.js` — mermaid.js wrapped in an IIFE that
  sets `window.mermaid` (unchanged from before). ~3 MB minified.
- `src/lib-beautiful.ts` → `assets/beautifulmermaid.min.js` — beautiful-mermaid
  wrapped the same way as `window.beautifulMermaid`. ~1.5 MB minified (pulls in
  ELK.js for flowchart layout) — too heavy to fold into the glue eagerly, so it
  gets its own lazy chunk rather than joining `mermaid.js`.

`@mostlylucid/mermaid-enhancements` (pan/zoom/export, see below) *is* bundled
eagerly into `mermaid.js` — tree-shaken down to ~48 KB minified / ~15 KB
gzipped, small enough not to need a lazy chunk of its own.

Theme follows `document.documentElement.classList.contains('dark')` /
`[data-theme]`, set by the inline theme script in `views.html`. beautiful-mermaid
diagrams re-theme for free: their colors are passed as `var(--pico-*)` CSS
custom properties on the `<svg>` (`beautifulThemeOptions()` in `index.ts`), so
Pico's own light/dark cascade repaints them with no re-render. mermaid.js
diagrams still re-render on theme change (`onThemeChange` + `renderAll()`),
same as before.

beautiful-mermaid also unconditionally embeds a Google Fonts `@import` in every
SVG's `<style>` block for whatever `font` name is in effect (default `Inter`) —
a CDN call this self-hosted-only site must not make. `stripGoogleFontsImport()`
removes it from the returned SVG string before it's injected; the real font is
applied afterward via our own CSS (`.mermaid-wrapper .mermaid svg text`).

### Pan/zoom, fullscreen, export

Both engines' output is enhanced with
[`@mostlylucid/mermaid-enhancements`](https://github.com/scottgal/mostlylucidweb/tree/main/mostlylucid-mermaid)
(`enhanceMermaidDiagrams()`, called after each render) — pan (drag), zoom
(buttons + ctrl/⌘+wheel), a fullscreen lightbox, and PNG/SVG export, built on
`svg-pan-zoom` + `html-to-image`. This replaced an earlier hand-rolled
`panzoom.ts` whose drag panning didn't actually work; the community library's
drag/zoom/fullscreen was verified live (real pointer drag pans the diagram,
confirmed via a transform-matrix check before/after).

The library auto-wraps any `.mermaid[data-processed="true"]` element it finds
in `.mermaid-wrapper` + a controls toolbar — mermaid.js sets
`data-processed="true"` itself; the beautiful-mermaid path sets it manually
after injecting the SVG so both engines get identical treatment. Its own
stylesheet + Boxicons-font toolbar would be a second self-hosted asset, so
`index.ts` injects an equivalent stylesheet instead (`ENH_CSS`): same class
names, colors mapped to `var(--pico-*)`, icons as plain glyph characters
(no icon font, no CDN).

`.mermaid-wrapper` is bounded to `max-height: 80vh`. Combined with the
library's `fit: true` / `center: true` initial pan-zoom, a large diagram fits
that viewport-bounded box on load instead of shrinking to unreadable size —
and the same fit-to-box math fixes wide-but-short diagrams (previously a
sliver): scale is bounded by whichever dimension is tighter, so a wide diagram
fits by width and sits centered rather than stretched to fill the height. Full
detail beyond fit level is one zoom/pan or fullscreen away.

## Build

```
npm run mermaid        # builds assets/mermaid.js + mermaid.min.js + beautifulmermaid.min.js
```

All three outputs are committed artifacts (like `chart.js` / `echarts.min.js`).
`npm run build` (tsc + vite) does not build the widget bundles — run the script
after editing `assets/mermaid/src/*`.

## Files

| File | Role |
|------|------|
| `internal/model/note_codelang.go` | `CodeLanguages` set + `HasCodeLanguage` |
| `internal/model/note.go` | `CodeLanguages` field; calls `extractCodeLanguages()` |
| `internal/templateviews/note.go` | `HasCharts()` / `HasCodeLanguage()` accessors |
| `internal/case/rendernotepage/endpoint.go` | appends widget scripts per note |
| `cmd/server/main.go` | core-only `UserJSURLs()`, `AssetURL()` |
| `assets/mermaid/` | glue + lib bundle sources (`index.ts`, `lib.ts`, `lib-beautiful.ts`) |
| `assets/mermaid.js`, `assets/mermaid.min.js`, `assets/beautifulmermaid.min.js` | built artifacts |
| `assets/embed.go` | embeds the artifacts |
| `docs/demo/mermaid.md` | demo note + e2e fixture (`/mermaid`) |
| `e2e/vault.spec.js` | "Mermaid Diagrams" tests |

## Tests

- `internal/model/note_codelang_test.go` — language set extraction.
- `e2e/vault.spec.js` "Mermaid Diagrams" — diagrams render to SVG; `mermaid.js`
  loads on `/mermaid` but not on a diagram-free page.
