---
free: true
title: "yield_blocks: per-page CSS and JS"
lang_redirect: "[[ru/user/yield_blocks]]"
---

`yield_blocks` collects CSS (or JS) from the components used on a page and emits it once, inline. Only the styles for blocks that actually appear on the page are included.

### The problem it solves

When you build a page from reusable components, each component has its own CSS. The naive approach — one global stylesheet — loads everything on every page. `yield_blocks` does the opposite: it scans which components your page uses and writes only their CSS into a single `<style>` tag.

### File layout

Components live alongside your layout files. Each component file defines its blocks:

```
_layouts/
└── my-theme/
    ├── components/
    │   ├── button.html
    │   ├── card.html
    │   └── hero.html
    └── page.html
```

### Component file structure

A component file contains two kinds of blocks: a style block and an HTML block. Both follow a naming convention:

- `_style_blockname` — CSS for the component
- `_js_blockname` — JS for the component (optional)
- `blockname` — the HTML of the component

```html
<!-- components/button.html -->
{{block _style_button()}}
.button { display: inline-flex; padding: 8px 16px; }
.button--primary { background: #0070f3; color: #fff; }
{{end}}

{{block button(label="Click", variant="")}}
<button class="button{{if variant}} button--{{variant}}{{end}}">{{label}}</button>
{{end}}
```

The style block's name starts with `_style_`. The HTML block uses the plain component name. The loader matches them by the shared suffix (`button` in both `_style_button` and `button`).

### Using components in a page

The page template uses `{{yield}}` to call components. No imports needed — the loader discovers component files automatically.

```html
<!-- pages/home.html -->
<html>
<body>
  {{yield hero(title="Welcome")}}
  {{yield card(title="Feature", body="...")}}
  {{yield button(label="Get Started", variant="primary")}}
  <style>{{yield_blocks("_style_")}}</style>
</body>
</html>
```

`yield_blocks("_style_")` scans all `{{yield}}` calls in the page, finds every block whose name starts with `_style_`, and writes their CSS into the tag. The result is a single `<style>` block containing only what this page uses.

Place `<style>{{yield_blocks("_style_")}}</style>` before `</body>`. Browsers apply inline styles regardless of position, but end-of-body is the conventional location.

### Pattern syntax

`yield_blocks` accepts either a prefix string or a regexp:

| Pattern | Matches |
|---------|---------|
| `"_style_"` | All blocks whose name starts with `_style_` |
| `"/_style_.*/"` | Same, as a regexp (wrapped in `/`) |
| `"/_js_.*/"` | All blocks whose name starts with `_js_` |

Use the prefix form for CSS and JS — it is shorter and reads clearly.

For JS, place a `<script>` tag the same way:

```html
<script>{{yield_blocks("/_js_.*/")}}</script>
```

### Transitive dependencies

If `card.html` internally yields `button`, the loader includes `button.html` automatically. You do not need to list dependencies manually — the loader walks the full component graph.

### Block name uniqueness and `@lid` / `@did`

Block names in Jet are **global** across all included files. If `hero.html` and `card.html` both define `{{block hero()}}`, the second definition silently overwrites the first. This is a real risk in multi-component projects where different developers add files independently.

`@lid` solves this by substituting the file path into block names before parsing. The substitution happens automatically — no configuration needed.

| File path | `@lid` value | `@did` value |
|-----------|-------------|-------------|
| `button.html` | `button` | `button` |
| `components/button.html` | `components_button` | `components-button` |
| `ui/nav/header.html` | `ui_nav_header` | `ui-nav-header` |

`@lid` and `@did` are **preprocessor variables** — the layout loader substitutes them before Jet parses the template. `@lid` (**l**odash **id** — underscores) is used in Jet block names. `@did` (**d**ash **id** — hyphens) is used in BEM CSS class names. Both are derived from the file path, not the block name. Use `@@lid` / `@@did` to emit a literal `@lid` / `@did` in JS or CSS.

Use `@lid` in block names and `@did` in CSS class names:

```html
{{block _style_@lid()}}
.@did { display: inline-flex; padding: 8px 20px; }
.@did--primary { background: #0070f3; color: #fff; }
{{end}}

{{block @lid(label="Click", variant="")}}
<button class="@did{{if variant}} @did--{{variant}}{{end}}">{{label}}</button>
{{end}}
```

For `button.html`, this expands to `_style_button`, `.button`, and `button`. For `components/button.html`, it expands to `_style_components_button`, `.components-button`, and `components_button`. The `yield_blocks("_style_")` prefix pattern still matches both, because the prefix is checked against the expanded name.

**Escaping.** If you need the literal string `@lid` in output — for example, as a JavaScript variable — write `@@lid`:

```html
{{block _js_@lid()}}
var @@lid = document.querySelector('.@did-root');
@@lid.addEventListener('click', () => { ... });
{{end}}
```

After substitution (for `button.html`):

```html
{{block _js_button()}}
var @lid = document.querySelector('.button-root');
@lid.addEventListener('click', () => { ... });
{{end}}
```

`@@lid` becomes `@lid` in the output; `@lid` (single `@`) becomes the file-derived value.

### Warnings

Two situations produce a non-fatal warning in the layout preview log:

- **Duplicate block name** — two component files define a block with the same name. The first definition wins; the second is ignored.
- **Invalid regexp** — a pattern wrapped in `/` that is not valid regexp syntax. `yield_blocks` outputs nothing for that call; the rest of the page renders normally.

Neither warning crashes rendering. Check the layout preview log if styles are missing unexpectedly.

### Full example

```
_layouts/theme/
├── components/
│   ├── button.html
│   ├── card.html
│   └── hero.html
└── page.html
```

`components/hero.html`:

```html
{{block _style_hero()}}
.hero { padding: 80px 24px; text-align: center; }
.hero__title { font-size: 2.5rem; font-weight: 700; }
{{end}}

{{block hero(title="")}}
<section class="hero">
  <h1 class="hero__title">{{title}}</h1>
</section>
{{end}}
```

`page.html`:

```html
<!DOCTYPE html>
<html>
<body>
  {{yield hero(title=note.Title())}}
  {{yield card(title="About", body=note.HTMLString())}}
  <style>{{yield_blocks("_style_")}}</style>
</body>
</html>
```

The rendered page gets a `<style>` tag with CSS from `_style_hero` and `_style_card` only — `_style_button` is absent because no `{{yield button(...)}}` appears on this page.

### BEM naming for style blocks

The `_style_blockname` convention pairs naturally with BEM class naming. BEM scopes every class to its block name, so when `yield_blocks` assembles CSS from multiple components onto one page, nothing collides.

A `card` component with BEM classes:

```html
{{block _style_card()}}
.card { border: 1px solid #eee; border-radius: 6px; padding: 16px; }
.card__title { font-weight: bold; }
.card--featured { border-color: #0070f3; }
{{end}}
```

`.card__title` can never conflict with `.hero__title` — the block name is part of every class. See [[en/user/bem|BEM naming in templates]] for the full convention.

### Static assets: `asset()`

A layout can reference static files (JS, CSS, SVG, images) that live next to it using `asset()`:

```html
<script defer src="{{ asset("scripts.js") }}"></script>
<img src="{{ asset("logo.svg") }}" alt="logo">
```

`asset()` resolves the path relative to the layout's own directory. If your layout is at `_layouts/mesh/index.html`, then `{{ asset("scripts.js") }}` serves `_layouts/mesh/scripts.js`.

In production the URL includes a cache-busting hash so browsers pick up updates immediately. In development the hash is omitted for readability.

Any static file can sit next to the layout — JS, SVG, images, fonts. The engine serves them automatically; no build step or extra configuration needed.

**Example layout structure:**

```
_layouts/
└── mesh/
    ├── _blocks.html
    ├── index.html
    ├── scripts.js      ← {{ asset("scripts.js") }}
    └── logo.svg        ← {{ asset("logo.svg") }}
```

Contrast this with `yield_blocks`, which collects inline CSS/JS from component blocks at render time. Use `asset()` when the file is a standalone static asset that does not need template processing.

### Related

- [[templates|Custom templates]] — template basics, Jet syntax, `note` and `nvs` variables
- [[templates-best-practices|Template best practices]] — organizing multi-template projects
- [[en/user/bem|BEM naming in templates]] — naming convention for component CSS classes
