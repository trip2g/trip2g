# Mermaid diagrams & conditional widget loading

Mermaid renders ` ```mermaid ` fenced code blocks as diagrams, like Obsidian.
Implementing it introduced a general mechanism — **per-note, backend-decided
widget script loading** — that any future per-language client widget reuses.

## Two parts

1. **Conditional widget loading** (backend) — the server knows, at render time,
   which client widgets a page needs, and emits exactly those `<script>` tags.
2. **The mermaid bundle** (frontend) — a tiny glue that lazily loads the heavy
   mermaid library, mirroring the datachart/echarts split.

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

## The mermaid bundle

`assets/mermaid/` mirrors `assets/chart/`:

- `src/index.ts` → `assets/mermaid.js` — tiny glue. Finds
  `pre > code.language-mermaid`, converts each to `<div class="mermaid">` with the
  decoded text, lazily loads `/assets/mermaid.min.js` (only if a block exists),
  then `mermaid.initialize({ startOnLoad: false, theme: dark ? 'dark' : 'default',
  securityLevel: 'strict' })` + `mermaid.run({ nodes })`.
- `src/lib.ts` → `assets/mermaid.min.js` — wraps the ESM mermaid package in an
  IIFE that sets `window.mermaid`, mirroring the prebuilt `echarts.min.js` global.
  ~3 MB; loaded lazily and only on pages with a diagram.
- `esbuild.browser.mjs` builds both outputs.

Theme follows `document.documentElement.classList.contains('dark')`, set by the
inline theme script in `views.html`.

### Pan/zoom

`src/panzoom.ts` makes each rendered SVG pan/zoomable — zero-dep, Pointer
Events based (svg-pan-zoom needs Hammer.js for pinch, so a small custom
implementation is lighter):

- Touch: pinch-to-zoom + one-finger drag to pan (drag only once zoomed —
  at fit the container keeps `touch-action: pan-y` so page scroll works).
- Desktop: + / − / reset / fullscreen buttons overlaid top-right, plus
  ctrl/⌘+wheel zoom (plain wheel keeps scrolling the page).
- Fullscreen is a CSS `position: fixed` overlay (portable, unlike
  `requestFullscreen` on iOS); Escape or the button exits.
- Default state is untouched fit-to-width; the CSS is injected by the glue so
  nothing is added to the main bundle. Theme re-render rebinds instead of
  stacking listeners.

## Build

```
npm run mermaid        # builds assets/mermaid.js + assets/mermaid.min.js
```

Both outputs are committed artifacts (like `chart.js` / `echarts.min.js`).
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
| `assets/mermaid/` | glue + lib bundle sources |
| `assets/mermaid.js`, `assets/mermaid.min.js` | built artifacts |
| `assets/embed.go` | embeds the artifacts |
| `docs/demo/mermaid.md` | demo note + e2e fixture (`/mermaid`) |
| `e2e/vault.spec.js` | "Mermaid Diagrams" tests |

## Tests

- `internal/model/note_codelang_test.go` — language set extraction.
- `e2e/vault.spec.js` "Mermaid Diagrams" — diagrams render to SVG; `mermaid.js`
  loads on `/mermaid` but not on a diagram-free page.
