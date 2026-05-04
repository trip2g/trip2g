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

{{ yield index_layout() content }}
  {{ yield mesh_bar() }}

  <!-- your content here -->

  {{ yield mesh_foot() }}
  <style>{{ yield_blocks("_style_mesh_") }}</style>
{{ end }}
```

That's it. Only `_blocks.html` needs an explicit import — it contains `index_layout` and shared blocks that are not auto-discovered. All component files (`bar.html`, `foot.html`, etc.) are imported automatically by the loader based on `{{ yield }}` calls.

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

Pick the components you need from the table above, yield them in order. Only `_blocks.html` needs an explicit import:

```html
{{ import "_blocks" }}

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

2. Yield in the layout — no import needed, the loader auto-discovers it:

```html
{{ yield mesh_myblock(title="Hello", body="World") }}
```

BEM rules to follow:
- Block: `.@did` — the component root
- Elements: `.@did__name` — parts of the block (flat, no nesting: `.@did__el` not `.@did__parent__el`)
- Modifiers: `.@did__el--state` — variants (always used with the base class)
- No global classes, no tag selectors, no IDs in CSS

## Building your own design system for one-shot landings

The mesh layout is a working example of a component-based design system inside trip2g. You can fork this approach for your own themes.

### What a design system looks like here

```
_layouts/
└── mytheme/
    ├── _blocks.html        ← page shell (HTML, head, CSS variables, shared blocks)
    ├── README.md           ← this file — component catalog for agents and humans
    ├── button.html         ← shared primitive: @lid/@did, variant, modifier
    ├── nav.html            ← navigation component
    ├── hero.html           ← hero section
    ├── features.html       ← features grid
    ├── pricing.html        ← pricing cards
    ├── footer.html         ← footer
    └── index.html          ← page: imports + yields + yield_blocks
```

### Step 1: define your design tokens in `_blocks.html`

```html
{{ block page_shell() }}
<!doctype html>
<html>
<head>
  <style>
    :root {
      --bg: #ffffff;
      --fg: #111111;
      --accent: #3b82f6;
      --muted: #6b7280;
      --mono: 'JetBrains Mono', monospace;
      --sans: 'Inter', sans-serif;
    }
  </style>
</head>
<body>
  {{ yield content }}
  <style>{{ yield_blocks("_style_") }}</style>
</body>
</html>
{{ end }}
```

### Step 2: create atomic components first

Start with primitives that everything else uses:

- `button.html` — links and buttons with variants
- `tag.html` — labels, badges
- `input.html` — form inputs

Then build section components that compose the primitives:

```html
{{ block @lid(title="", cta_label="", cta_href="") }}
<section class="@did">
  <h1 class="@did__title">{{ title }}</h1>
  {{yield mesh_button(label=cta_label, href=cta_href, variant="primary")}}
</section>
{{ end }}
```

### Step 3: write a README.md for your theme

Document every component with its block name and parameters. This file is what agents read to generate landings. Include:

- Component table (name, block, parameters, description)
- CSS variable reference
- `@lid`/`@did` placeholder explanation
- A minimal page template agents can copy

### Step 4: generate a landing with one prompt

Give the agent:
1. This README (or your theme's README)
2. A prompt: _"Create a landing for a SaaS tool that does X. Use components: hero, features (3 items), pricing (2 tiers), footer."_

The agent produces two files:
- `docs/mypage.md` — frontmatter with `layout: mytheme/mypage`
- `docs/_layouts/mytheme/mypage.html` — imports + yields in order

No build step. No config. Reload the vault and the page is live.

### Key principles

1. **One file = one component.** CSS, HTML (EN), HTML (RU) all in one `.html` file.
2. **`@lid` for block names, `@did` for CSS classes.** Preprocessor variables — no manual renaming when copying components.
3. **`yield_blocks` collects CSS automatically.** Place `<style>{{ yield_blocks("_style_") }}</style>` once at end of body. Only CSS for used components is emitted.
4. **Modifier = BEM mixin.** Pass `modifier="parent-block__child"` to inject a component into a parent's layout context without breaking its own BEM scope.
5. **Auto-import.** No `{{ import "button" }}` needed. The loader resolves transitive dependencies from `{{ yield }}` calls.
