# BEM Blocks as Jet Templates

Each BEM block is a self-contained Jet template that owns its HTML, CSS, and JS.
Styles and scripts are collected during rendering and flushed once at the end of the page.

## Concept

Traditional approach: one global CSS file, one global JS file. Every page loads everything.

BEM Blocks approach: each block declares its own styles and scripts inline.
The rendering context deduplicates by block name — no matter how many times a block is
included on a page, its CSS and JS are written to the page exactly once.

No build step. No bundler. No class name collisions. Only what is used gets included.

## Block Structure

A block template contains three optional sections:

```html
{{block button(label="Click", variant="")}}

{{inline_css}}
.button {
  display: inline-flex;
  align-items: center;
  padding: 8px 16px;
  border: none;
  cursor: pointer;
}
.button--primary {
  background: #0070f3;
  color: #fff;
}
{{end}}

{{inline_js}}
document.querySelectorAll('.button').forEach(btn => {
  btn.addEventListener('click', e => e.currentTarget.classList.toggle('button--active'))
})
{{end}}

<button class="button{{if variant}} button--{{variant}}{{end}}">{{label}}</button>

{{end}}
```

Rules:
- `inline_css` and `inline_js` sections are optional — include only what the block needs.
- CSS uses only BEM class selectors (see `bem.md`).
- JS is self-contained: uses `querySelectorAll`, no globals, no dependencies on load order.

## Page Template

```html
<html>
<head>
  <meta charset="utf-8">
  <title>{{.Title}}</title>
</head>
<body>

  {{import "blocks/header.html"}}
  {{yield header()}}

  {{import "blocks/button.html"}}
  {{yield button(label="Subscribe", variant="primary")}}
  {{yield button(label="Cancel")}}

  {{import "blocks/card.html"}}
  {{yield card(title="Hello")}}

  {{render_inline_css}}
  {{render_inline_js}}
</body>
</html>
```

`button` is included twice — its CSS and JS appear in the page once.

`render_inline_css` and `render_inline_js` are placed before `</body>`.
Inline `<style>` tags are valid anywhere in the document; browsers apply them regardless of
position. Inline `<script>` tags at end of body is the standard pattern anyway.

## How It Works

### Registration functions

`inline_css` and `inline_js` are Jet global functions backed by the render context:

```go
// Simplified
type InlineCollector struct {
    css map[string]string // block name → css text
    js  map[string]string // block name → js text
}
```

When a block calls `{{inline_css}}...{{end}}`, the function receives the block name
(from the enclosing `BlockNode`) and the text content, then stores it in the map.
Subsequent calls with the same key are ignored — first registration wins.

### Render functions

`render_inline_css` outputs:

```html
<style>
/* collected CSS from all used blocks */
</style>
```

`render_inline_js` outputs:

```html
<script>
/* collected JS from all used blocks */
</script>
```

### Static vs dynamic CSS

At template parse time, `blockFinder` in `layoutloader` can walk the AST of each
`inline_css` block and check whether all nodes are `*jet.TextNode`. If so, the CSS is
static and can be extracted at load time — no runtime overhead for those blocks.

If the block contains Jet expressions (e.g. a CSS custom property derived from a
template variable), it falls back to runtime collection.

## One Block, One Responsibility

A block owns both its CSS and its JS. This means you cannot import a block and opt out
of its JS while keeping its CSS (or vice versa).

If you need that, split the block:

```
blocks/button.html          — HTML + CSS only
blocks/button-interactive.html — extends button, adds JS
```

This is intentional: splitting is a signal that the block has two distinct responsibilities.

## Performance

### Per request cost

| Operation | Cost |
|-----------|------|
| CSS/JS registration | Map write — O(1) per block |
| Deduplication | Map key check — O(1) |
| render_inline_css/js | Single string join over N used blocks |
| Body buffering | Not needed — `render_inline_*` are at end of body |

No body buffering is required because the render functions are placed after all block
yields. The page streams normally; styles and scripts are appended at the end.

### Comparison

WordPress renders a page by:
- executing dozens of PHP hooks and filters
- running 30–100+ SQL queries per request
- concatenating CSS/JS from 10–30 plugin files
- often serving 500–2000 KB of unused CSS

BEM Blocks approach:
- template is pre-compiled and cached in memory
- CSS/JS collection is pure in-memory map operations
- output contains only CSS and JS for blocks actually used on that page
- zero SQL queries, zero file reads during render

A page using 15 blocks collects 15 map writes and outputs one `<style>` and one
`<script>` tag. The overhead is immeasurable compared to a full page render.

### Caching

If all blocks on a given layout have static CSS (no Jet expressions inside `inline_css`),
the entire collected stylesheet can be cached at layout compile time — the per-request
cost drops to a single string write.

## yield_blocks

`yield_blocks` is the user-facing global function that implements the collection and
emission described above. It accepts a prefix string or a regexp pattern and emits CSS
(or JS) only for the blocks that were actually yielded on the current page.

```html
<style>{{yield_blocks("_style_")}}</style>
<script>{{yield_blocks("/_js_.*/")}}</script>
```

The naming convention it relies on:

- `_style_blockname` — CSS block for a component
- `_js_blockname` — JS block for a component

The loader auto-discovers component files (no `{{import}}` needed) and resolves
transitive dependencies: if `card.html` yields `button`, `button.html` is included
automatically.

User-facing documentation: [yield_blocks user guide](docs/en/user/yield_blocks.md)

### Limitations and rules

**Block parameters are nil.** `yield_blocks` calls each collected block via `rt.YieldBlock(name, nil)` — no arguments are passed. All block parameters default to their zero/empty values. A CSS block with `{{ if theme }}.box--{{theme}}{{ end }}` will never emit the conditional part when called this way. Rule: CSS blocks must be self-contained and must not branch on their own parameters.

**Global functions are available.** Globals registered via `AddGlobalFunc` (e.g. `asset`, `note`) are accessible inside CSS blocks called by `yield_blocks`, exactly as in any other block. No special handling is needed.

**Do not yield sibling CSS blocks manually.** If a CSS block calls `{{ yield _style_other() }}` internally, and `_style_other` also matches the `yield_blocks` pattern, the output will contain that CSS twice — once from the explicit yield and once from `yield_blocks` collecting it independently. Let `yield_blocks` handle collection; CSS blocks must not yield each other.

**Placement in `<head>` is safe.** Because block names are resolved by static analysis (third-pass wire) before any rendering occurs, `yield_blocks` can be placed in `<head>` rather than at the end of `<body>`. This is the recommended placement for CSS to avoid flash of unstyled content (FOUC):

```html
<head>
  <style>{{ yield_blocks("_style_mesh_") }}</style>
</head>
```
