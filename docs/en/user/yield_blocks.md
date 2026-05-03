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

### Related

- [[templates|Custom templates]] — template basics, Jet syntax, `note` and `nvs` variables
- [[templates-best-practices|Template best practices]] — organizing multi-template projects
