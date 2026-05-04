# Mesh Layout

> **For AI agents:** Read this file to generate a complete landing page. All components, parameters, CSS variables, and the exact file structure are documented below. To produce a working landing: create a Markdown note + one HTML layout file. No build step needed.



The mesh theme for trip2g. Used on the main landing pages (`/` and `/ru`).

## How to create a page

1. Create a Markdown note with a `layout` field in frontmatter:

```markdown
---
layout: mesh/my-page
title: My Page
free: true
---
```

2. Create the layout file at `_layouts/mesh/my-page.html`:

```html
{{ import "_blocks" }}
{{ import "bar" }}
{{ import "foot" }}

{{ yield index_layout() content }}
  {{ yield mesh_bar() }}

  <!-- your content here -->

  {{ yield mesh_foot() }}
  <style>{{ yield_blocks("_style_mesh_") }}</style>
{{ end }}
```

That's it. The layout file controls the HTML structure; the note provides the title and any custom frontmatter.

## Components

All components live in this directory. Each file defines a `@lid` (EN) and `@lid_ru` (RU) block plus a shared `_style_@lid` CSS block. Import the file and yield the block by its expanded name.

| File | Block (EN) | Block (RU) | Description |
|------|-----------|-----------|-------------|
| `bar.html` | `mesh_bar` | `mesh_bar_ru` | Top navigation bar with ⌘K MCP hint modal |
| `hero.html` | `mesh_hero` | `mesh_hero_ru` | Hero section with animated graph and trace frames |
| `privacy.html` | `mesh_privacy` | `mesh_privacy_ru` | Data privacy section with SVG diagram |
| `philo.html` | `mesh_philo` | `mesh_philo_ru` | Philosophy blurb |
| `matrix.html` | `mesh_matrix` | `mesh_matrix_ru` | Red/blue pill matrix section |
| `try_now.html` | `mesh_try_now` | `mesh_try_now_ru` | Try-now section with prompt box and steps |
| `newsletter.html` | `mesh_newsletter` | `mesh_newsletter_ru` | Newsletter signup |
| `trusted_by.html` | `mesh_trusted_by` | `mesh_trusted_by_ru` | Trusted-by channel grid |
| `testimonials.html` | `mesh_testimonials` | `mesh_testimonials_ru` | Testimonials grid |
| `foot.html` | `mesh_foot` | `mesh_foot_ru` | Footer with CTA and coda |

## Shared blocks (`_blocks.html`)

Parameterized blocks available everywhere after `{{ import "_blocks" }}`:

| Block | Parameters | Description |
|-------|-----------|-------------|
| `index_layout` | — | Full HTML page wrapper (`<html>`, `<head>`, `<body>`, global CSS, JS) |
| `section_header` | `lhs`, `rhs` | Section heading with left and right labels |
| `how_step` | `num`, `title` | Numbered step with `{{ yield content }}` |
| `compat_item` | `title`, `desc`, `state`, `state_class` | Compatibility table row with `{{ yield content }}` for icon |
| `roadmap_col` | `title` | Roadmap column with `{{ yield content }}` |
| `pricing_card` | `tag`, `title`, `price`, `period`, `cta_href`, `cta_class`, `cta_text`, `cls` | Pricing card with `{{ yield content }}` for feature list |

## CSS variables

Defined in `_blocks.html` `<style>`, available in all component CSS:

| Variable | Default | |
|----------|---------|--|
| `--bg` | `#0e0e0c` | Background |
| `--fg` | `#e8e6df` | Foreground / text |
| `--accent` | `oklch(0.78 0.18 145)` | Green accent |
| `--muted` | `#5a5a52` | Muted text |
| `--rule` | `#1e1e1a` | Border / rule color |
| `--mono` | `"JetBrains Mono", monospace` | Monospace font |
| `--sans` | `"Inter", sans-serif` | Sans font |

## Placeholder reference

When writing a new component file, use these placeholders — they are expanded at load time based on the file path:

| Placeholder | Full name | Expands to | Used for |
|-------------|-----------|-----------|---------|
| `@lid` | **l**odash **id** | `mesh_bar` (underscores) | Jet block names |
| `@did` | **d**ash **id** | `mesh-bar` (hyphens) | BEM CSS class names |
| `@@lid` | escape | literal `@lid` | In JS/CSS where `@lid` should stay as-is |
| `@@did` | escape | literal `@did` | In JS/CSS where `@did` should stay as-is |

## New component template

```html
{{ block _style_@lid() }}
.@did {  }
.@did__title {  }
.@did__body {  }
{{ end }}

{{ block @lid() }}
<section class="@did">
  <h2 class="@did__title">Title</h2>
  <p class="@did__body">Body text.</p>
</section>
{{ end }}

{{ block @lid_ru() }}
<section class="@did">
  <h2 class="@did__title">Заголовок</h2>
  <p class="@did__body">Текст.</p>
</section>
{{ end }}
```

Save as `_layouts/mesh/mycomponent.html`, import it in the page, and yield `mesh_mycomponent()`.

## Agent instructions: generating a landing page

To generate a landing page from a prompt, produce two files.

### File 1: the note (`docs/<slug>.md`)

```markdown
---
layout: mesh/<slug>
title: <Page Title>
free: true
---
```

No body content needed unless the layout uses `{{ note.Body() }}`.

### File 2: the layout (`docs/_layouts/mesh/<slug>.html`)

Pick the components you need from the table above, import them, yield them in order:

```html
{{ import "_blocks" }}
{{ import "bar" }}
{{ import "hero" }}
{{ import "try_now" }}
{{ import "foot" }}

{{ yield index_layout() content }}
  {{ yield mesh_bar() }}
  {{ yield mesh_hero() }}
  {{ yield mesh_try_now() }}
  {{ yield mesh_foot() }}
  <style>{{ yield_blocks("_style_mesh_") }}</style>
{{ end }}
```

You can also add custom sections inline between yields using the shared blocks from `_blocks.html`:

```html
{{ yield section_header(lhs="how it works", rhs="3 steps") }}
<div class="how">
  {{ yield how_step(num="01", title="Step one") content }}
    <p>Description of step one.</p>
  {{ end }}
</div>
```

### How BEM components work here

Each component file (`bar.html`, `hero.html`, etc.) is a self-contained BEM block:

```
{{ block _style_@lid() }}   ← CSS for this component, scoped to .@did
  .@did { ... }
  .@did__nav { ... }
  .@did__nav-link--key { ... }
{{ end }}

{{ block @lid() }}           ← EN HTML
  <header class="@did">
    <nav class="@did__nav">...</nav>
  </header>
{{ end }}

{{ block @lid_ru() }}        ← RU HTML (same structure, Russian text)
  ...
{{ end }}
```

At load time:
- `@lid` → `mesh_bar` (underscores) — used as the Jet block name
- `@did` → `mesh-bar` (hyphens) — used as the BEM CSS class name

`{{ yield_blocks("_style_mesh_") }}` at the end of the page collects CSS from all used components and emits it once in a `<style>` tag. Unused components contribute no CSS.

### Creating a custom component

1. Create `docs/_layouts/mesh/myblock.html`:

```html
{{ block _style_@lid() }}
.@did {
  padding: 64px 32px;
  text-align: center;
}
.@did__title {
  font-size: 2rem;
  font-weight: 700;
}
.@did__body {
  color: var(--muted);
  max-width: 600px;
  margin: 16px auto 0;
}
{{ end }}

{{ block @lid(title="", body="") }}
<section class="@did">
  <h2 class="@did__title">{{ title }}</h2>
  <p class="@did__body">{{ body }}</p>
</section>
{{ end }}
```

2. Import and yield in the layout:

```html
{{ import "myblock" }}
...
{{ yield mesh_myblock(title="Hello", body="World") }}
```

BEM rules to follow:
- Block: `.@did` — the component root
- Elements: `.@did__name` — parts of the block (flat, no nesting: `.@did__el` not `.@did__parent__el`)
- Modifiers: `.@did__el--state` — variants (always used with the base class)
- No global classes, no tag selectors, no IDs in CSS
