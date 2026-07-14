# Mesh Layout

> **For AI agents:** Read this file to generate a complete landing page. All components, parameters, CSS variables, and the exact file structure are documented below. To produce a working landing: create a Markdown note + one HTML layout file. No build step needed.



The mesh theme for trip2g. Used on the main landing pages (`/` and `/ru`).

## Where the landing copy lives

The landing dogfoods trip2g's own content primitives: the words are **markdown notes**, not
hardcoded HTML. The visible copy lives in hidden section notes (the `_` prefix keeps them out of
listings/search and off standalone URLs):

| Note | Rendered into |
|------|---------------|
| `_index_hero.md` / `ru/_index_hero.md` | hero left column (`hero.html`) |
| `_index_getting_started.md` / `ru/_index_getting_started.md` | "0 → live site" steps (`how.html`) |
| `_index_capabilities.md` / `ru/_index_capabilities.md` | 6-capability grid (`capabilities.html`) |
| `_index_payoff.md` / `ru/_index_payoff.md` | federation payoff band (`network.html`) |

The layout pulls each note's rendered body with `{{ yield mesh_section(path="…") }}`, which calls
`nvs.ByPath(path).HTMLString()`. The layout only supplies structure and CSS; edit the markdown
to change the words. Page `<title>`, meta description, and `og:image` come from the `_index.md`
frontmatter (`title`, `description`, `og_image`), read in `index_layout`.

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

All components live in this directory. Each file defines **one** `@lid` block plus a shared `_style_@lid` CSS block. Import the file and yield the block by its expanded name.

**One block serves both languages.** There are no `@lid_ru` twins. The block declares its
translatable strings as parameters with **English defaults**; `index.html` yields it bare
(`{{ yield mesh_bar() }}`), and `ru_index.html` overrides the strings at the call site
(`{{ yield mesh_bar(lang="ru", docs_label="Документация", …) }}`). See
[Localization](#localization) below.

| File | Block | Description |
|------|-------|-------------|
| `bar.html` | `mesh_bar` | Top navigation bar with ⌘K MCP hint modal |
| `hero.html` | `mesh_hero` | Type-led hero: copy from the `_index_hero` note + CTA row |
| `how.html` | `mesh_how` | "0 → live site" steps from the `_index_getting_started` note |
| `capabilities.html` | `mesh_capabilities` | 6-card capability grid, copy from the `_index_capabilities` note |
| `network.html` | `mesh_network` | Federation payoff (`_index_payoff` note) + animated graph and trace frames |
| `privacy.html` | `mesh_privacy` | Data privacy section with SVG diagram |
| `philo.html` | `mesh_philo` | Philosophy blurb |
| `matrix.html` | `mesh_matrix` | Red/blue pill matrix section |
| `cases.html` | `mesh_cases` | 4 topology use-case cards |
| `roadmap.html` | `mesh_roadmap` | Shipped / in-progress / planned columns |
| `try_now.html` | `mesh_try_now` | Try-now section with prompt box and steps |
| `newsletter.html` | `mesh_newsletter` | Newsletter signup |
| `docs_list.html` | `mesh_docs_list` | JS-populated `tree docs/` list |
| `pricing.html` | `mesh_pricing` | 3 pricing cards |
| `community.html` | `mesh_community` | Closed-community CTA (Russian only; no English twin — see Localization) |
| `foot.html` | `mesh_foot` | Footer with CTA and coda |

## Localization

The landing renders in two languages from **one block per component**. Three mechanisms,
in order of preference:

1. **`lang` parameter.** Locale-dependent hrefs compose from it: `href="/{{ lang }}/user"`
   in plain HTML, or `href="/"+lang+"/user"` inside a `{{ yield }}` argument. `index.html`
   leaves `lang` at its `"en"` default; `ru_index.html` passes `lang="ru"`. A component
   only declares `lang` when something actually derives from it (hrefs, asset names); pure
   label/note components skip it.
2. **String parameters with English defaults.** Every short label — nav text, headings,
   CTA labels, section-header sides — is a block parameter defaulting to its English text.
   `ru_index.html` overrides each with the Russian string. Non-mechanical hrefs (e.g.
   `/en/user/getting_started` vs `/ru/user/nachalo_rabotyi`) are parameters too.
3. **Markdown notes for paragraph copy.** Where a component's copy is prose with no
   load-bearing element classes (hero, how, capabilities, network), it lives in a hidden
   `_index_*.md` note pair (see [Where the landing copy lives](#where-the-landing-copy-lives))
   and the block takes the note path as a parameter (`note="ru/_index_hero.md"`).

**When copy stays in the template as parameters, not a note:** if the markup carries BEM
element classes or structure the Markdown→HTML round-trip can't reproduce (`<dl>`, `<pre>`
with token spans, `<kbd>`, styled inline links, ASCII art, status columns), moving it to a
note would drop those classes and change the page. Keep it in the template and parametrize
the strings instead.

**Genuinely different structure** (not just text) between EN and RU → a small
`{{ if lang == "ru" }}…{{ end }}` around only the differing fragment, as a documented
exception. `community.html` is Russian-only (no English counterpart), so it is a single
`@lid` block with hardcoded Russian copy, yielded as `mesh_community()` from `ru_index.html`.

## Shared blocks (`_blocks.html`)

Parameterized blocks available everywhere after `{{ import "_blocks" }}`:

| Block | Parameters | Description |
|-------|-----------|-------------|
| `index_layout` | — | Full HTML page wrapper (`<html>`, `<head>`, `<body>`, global CSS, JS). Title/description/og:image read from the page note frontmatter (`title`, `description`, `og_image`). |
| `section_header` | `lhs`, `rhs` | Section heading with left and right labels |
| `mesh_section` | `path` | Transcludes a hidden section note's rendered body by vault path via `nvs.ByPath(path).HTMLString()`. The landing copy lives in `_index_*.md` notes; the layout supplies only structure and style. |
| `how_step` | `num`, `title` | Numbered step with `{{ yield content }}` |
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

{{ block @lid(title="Title", body="Body text.") }}
<section class="@did">
  <h2 class="@did__title">{{ title }}</h2>
  <p class="@did__body">{{ body }}</p>
</section>
{{ end }}
```

One block, English defaults. Save as `_layouts/mesh/mycomponent.html`, import it in the
page, and yield `mesh_mycomponent()`. The Russian page overrides the strings:
`{{ yield mesh_mycomponent(title="Заголовок", body="Текст.") }}`.

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
{{ block _style_@lid() }}          ← CSS for this component, scoped to .@did
  .@did { ... }
  .@did__nav { ... }
  .@did__nav-link--key { ... }
{{ end }}

{{ block @lid(lang="en", …) }}     ← one block, English defaults; RU overrides at call site
  <header class="@did">
    <nav class="@did__nav">...</nav>
  </header>
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

1. **One file = one component = one block.** CSS and a single bilingual `@lid` block (English defaults + `lang`/string parameters the Russian page overrides) live in one `.html` file.
2. **`@lid` for block names, `@did` for CSS classes.** Preprocessor variables — no manual renaming when copying components.
3. **`yield_blocks` collects CSS automatically.** Place `<style>{{ yield_blocks("_style_") }}</style>` once at end of body. Only CSS for used components is emitted.
4. **Modifier = BEM mixin.** Pass `modifier="parent-block__child"` to inject a component into a parent's layout context without breaking its own BEM scope.
5. **Auto-import.** No `{{ import "button" }}` needed. The loader resolves transitive dependencies from `{{ yield }}` calls.
