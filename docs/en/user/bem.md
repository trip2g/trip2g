---
free: true
title: "BEM naming in templates"
lang_redirect: "[[ru/user/bem]]"
---

BEM is a CSS naming convention that keeps component styles from leaking into each other. In trip2g templates, BEM is the recommended way to name the classes inside your `_style_*` blocks.

### What BEM is

BEM stands for Block, Element, Modifier. Every class name is one of three things:

- **Block** — a standalone component: `.card`, `.hero`, `.button`
- **Element** — a part of a block, joined with `__`: `.card__title`, `.card__image`
- **Modifier** — a variant of a block or element, joined with `--`: `.card--featured`, `.button--primary`

That's the entire system. No nesting in the names, no context-dependent selectors.

Key rules:

- **Flat element names** — never encode DOM depth in the class name: `.card__body` not `.card__content__body`
- **No global modifiers** — `.card--hidden` not `.hidden` (global classes collide across components)
- **Mixin** — one element can carry two block classes: `class="button nav__item"` applies both blocks' styles

### Why trip2g templates use BEM

Template components are loaded independently and their CSS is assembled per-page by `yield_blocks`. Without a naming convention, two components can easily define `.title` or `.wrapper` and overwrite each other.

BEM prevents that by making every class name unique to its block. `.card__title` and `.hero__title` cannot collide, even if both blocks appear on the same page.

### Naming convention in component files

Each component file in `_layouts/your-theme/components/` follows this pattern:

```
_style_blockname   ← the CSS block for yield_blocks to collect
blockname          ← the HTML block that yield renders
```

The block name in the CSS classes matches the component name. If the component is `card`, all its classes start with `card`:

```html
{{block _style_card()}}
.card { border: 1px solid #eee; border-radius: 6px; padding: 16px; }
.card__title { font-size: 1.25rem; font-weight: bold; margin-bottom: 8px; }
.card__body { color: #555; line-height: 1.5; }
.card--featured { border-color: #0070f3; }
{{end}}

{{block card(title="", body="", featured=false)}}
<div class="card{{if featured}} card--featured{{end}}">
  <h2 class="card__title">{{title}}</h2>
  <p class="card__body">{{body}}</p>
</div>
{{end}}
```

The style block is named `_style_card` — the `_style_` prefix tells `yield_blocks` to collect it. The HTML block is named `card` — the suffix matches, so the loader pairs them automatically.

### The `_style_blockname` convention

The `_style_` prefix is not arbitrary. `yield_blocks("_style_")` finds every block whose name starts with `_style_` and writes their CSS into one `<style>` tag. Naming your CSS block `_style_card` instead of `card_css` or `styles_card` keeps it discoverable by the loader with no extra configuration.

Consistent BEM class naming inside those blocks is what prevents collisions between components once their CSS lands on the same page.

### A fuller example

Three components on one page, all using BEM:

```html
{{block _style_hero()}}
.hero { padding: 80px 24px; text-align: center; background: #f5f5f5; }
.hero__title { font-size: 2.5rem; font-weight: 700; }
.hero__subtitle { color: #666; margin-top: 8px; }
{{end}}

{{block _style_button()}}
.button { display: inline-flex; padding: 10px 20px; border-radius: 4px; }
.button--primary { background: #0070f3; color: #fff; }
.button--ghost { border: 1px solid #0070f3; color: #0070f3; }
{{end}}

{{block _style_card()}}
.card { border: 1px solid #eee; border-radius: 6px; padding: 16px; }
.card__title { font-weight: bold; }
.card--featured { border-color: #0070f3; }
{{end}}
```

`yield_blocks("_style_")` collects all three into a single `<style>` tag. Because every class is scoped to its block name, there are no conflicts.

### What to avoid

Avoid generic class names inside component CSS blocks — they will conflict the moment two components share a page:

```html
{{block _style_card()}}
/* Bad: .title, .body, .wrapper are too generic */
.title { font-weight: bold; }
.wrapper { padding: 16px; }

/* Good: scoped to the block */
.card__title { font-weight: bold; }
.card { padding: 16px; }
{{end}}
```

### Using `$fileID` to guarantee unique block names

Block names in Jet are global. If two component files both define a block called `hero`, one silently overwrites the other. This can happen in any multi-component project — BEM scopes CSS class names, but it does not scope Jet block names.

`$fileID` solves this at the block-name level. Before parsing, the engine replaces `$fileID` in block names with a sanitized version of the file path:

- `components/button.html` → `$fileID` = `components_button`
- `ui/nav/header.html` → `$fileID` = `ui_nav_header`

The recommended pattern combines BEM class names with `$fileID` block names:

```html
{{block _style_$fileID()}}
.button { display: inline-flex; padding: 8px 20px; }
.button--primary { background: #0070f3; color: #fff; }
{{end}}

{{block $fileID(label="Click", variant="")}}
<button class="button{{if variant}} button--{{variant}}{{end}}">{{label}}</button>
{{end}}
```

BEM keeps CSS classes unique across the page. `$fileID` keeps Jet block names unique across all component files. Use both together in every component file.

### Related

- [[en/user/yield_blocks|yield_blocks: per-page CSS and JS]] — how style blocks are collected and emitted, and the full `$fileID` reference
- [[en/user/templates|Custom templates]] — template basics and Jet syntax
