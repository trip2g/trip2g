---
home_position: 46
title: "Theming the default template"
free: true
lang_redirect: "[[ru/user/themes]]"
---

You theme the default template by pasting a `<style>` block that overrides Pico CSS variables into the admin **HTML Injection** field. No rebuild, no fork — the override lands in `<head>` after the stylesheet, so your values win.

The default template is a custom build of [Pico CSS v2.1.1](https://picocss.com). Its whole look — colors, spacing, fonts, radius — comes from `--pico-*` custom properties. Change the variables and the whole site re-themes at once.

> 🎨 **Try it live:** the [[en/user/theme-editor|Theme editor]] page renders an interactive editor right on this site — drag the Pico variables with a live preview, and (signed in as admin) Save writes the theme for you.

The editor is just a custom layout (`_layouts/theme_editor.html`) with the save wired in: it calls the `createHtmlInjection` / `updateHtmlInjection` GraphQL mutations, gated by an admin check (`currentUser.IsAdmin()` in the layout, re-verified on the server), so an admin's Save lands straight in the site's HTML injection. That is the general pattern — a custom template can grow into any tool that talks to the API, a Markdown editor included, the same way the [[en/user/kanban|kanban board]] does.

Install the editor on your own site from [trip2g/theme_editor_template](https://github.com/trip2g/theme_editor_template): `curl -fsSL https://github.com/trip2g/theme_editor_template/releases/latest/download/theme_editor.html -o _layouts/theme_editor.html`, then add a note with `layout: theme_editor`.

## Where to paste a theme

Admin → **SEO & URLs** → **HTML Injections** → **+ Add**.

Fill the form:

- **Placement** — `Head` (loads inside `<head>`, so CSS overrides beat the base stylesheet)
- **HTML Content** — your `<style>…</style>` block
- **Description** — a name for yourself, e.g. `Dark purple theme`
- **Position** — order among injections (lower runs first); leave `0`
- **Active From / Active To** — optional date window (handy for seasonal themes); leave empty for always-on

Save. The injection is **global** — it applies to every page of the instance. There is no per-note theme.

> [!warning] Injected HTML is raw and unescaped
> Whatever you paste renders verbatim into the page. A stray `<style>` or `<script>` runs. Paste CSS you wrote or trust.

## The Pico variables you can override

These are the real variables the template reads. Redefine any of them on `:root`.

**Colors**

| Variable | Controls |
|---|---|
| `--pico-primary` | Accent color — links, active nav, highlights |
| `--pico-primary-hover` | Accent on hover |
| `--pico-primary-underline` | Link underline color |
| `--pico-background-color` | Page background |
| `--pico-color` | Body text color |
| `--pico-contrast` | High-contrast text (header nav) |
| `--pico-muted-color` | Secondary/dimmed text |
| `--pico-muted-border-color` | Borders, dividers |
| `--pico-card-background-color` | Card / magazine tile background |
| `--pico-card-sectioning-background-color` | Card header/footer background |
| `--pico-h1-color` … `--pico-h6-color` | Heading colors |
| `--pico-code-background-color`, `--pico-code-color` | Inline code |
| `--pico-mark-background-color`, `--pico-mark-color` | `==highlight==` |
| `--pico-blockquote-border-color` | Blockquote left bar |

**Type & shape**

| Variable | Controls |
|---|---|
| `--pico-font-family-sans-serif` | Main font stack |
| `--pico-font-family-monospace` | Code font stack |
| `--pico-font-size` | Base font size |
| `--pico-line-height` | Line height |
| `--pico-font-weight` | Base weight |
| `--pico-border-radius` | Corner rounding |
| `--pico-spacing` | Base spacing unit |
| `--pico-typography-spacing-vertical` | Gap between paragraphs |

**trip2g-specific variables**

| Variable | Default | Controls |
|---|---|---|
| `--site-max-width` | `1440px` | Max width of header, content, footer |
| `--header-h` | `56px` | Sticky header height |

## Light and dark

The template ships light and dark modes and remembers the reader's choice. It sets `data-theme="light"` or `data-theme="dark"` on the `<html>` element. Pico's default palette is azure (`#0172ad` light, `#01aaff` dark).

Theme each mode separately:

- `:root` — applies to both modes
- `:root[data-theme="light"]` — light mode only
- `:root[data-theme="dark"]` — dark mode only

## Ready-to-paste example: purple theme

Paste this into an `Head` injection. It re-accents both modes and darkens the dark background.

```html
<style>
:root {
  --pico-primary: #7c3aed;
  --pico-primary-hover: #6d28d9;
  --pico-primary-underline: rgba(124, 58, 237, 0.45);
  --pico-border-radius: 0.75rem;
  --pico-font-family-sans-serif:
    "Inter", system-ui, -apple-system, sans-serif;
  --site-max-width: 1100px;
}

:root[data-theme="light"] {
  --pico-background-color: #faf9fc;
  --pico-primary: #7c3aed;
  --pico-primary-hover: #6d28d9;
}

:root[data-theme="dark"] {
  --pico-background-color: #0f0f16;
  --pico-card-background-color: #191922;
  --pico-color: #e6e6ee;
  --pico-primary: #a78bfa;
  --pico-primary-hover: #c4b5fd;
}
</style>
```

Change the six hex values and you have a different brand. Everything else — cards, nav, callouts, code — recolors from those variables.

> [!bug] Antipattern: styling elements directly
> Don't write `a { color: purple !important }` or restyle `.content h1` by hand. You'll fight the base stylesheet with `!important` and break dark mode. Override the **variables** instead — one change, both modes, no `!important`.

## Custom classes you can target

For tweaks beyond colors, the template uses stable BEM classes. Target these in your injected CSS:

| Class | Element |
|---|---|
| `.site-header`, `.site-header__logo`, `.site-header__nav` | Top header |
| `.site-footer`, `.site-footer__column-title` | Footer |
| `.layout`, `.layout__main`, `.layout__sidebar` | Page grid |
| `.content`, `.content__title`, `.content__body` | Note content area |
| `.magazine`, `.magazine-item`, `.magazine-item__title` | Related-notes grid |
| `.sidebar-nav` | Sidebar navigation |
| `.callout`, `.callout--<type>` | Obsidian callouts (`--note`, `--warning`, `--tip`, …) |

Callouts expose per-type `--callout-color` and `--callout-bg` variables, so you can recolor one callout type without touching the rest:

```html
<style>
.callout--tip { --callout-color: #7c3aed; --callout-bg: rgba(124, 58, 237, 0.1); }
</style>
```

## Load a custom font

Pico reads whatever font stack you set. Import the file, then point the variable at it:

```html
<style>
@import url("https://fonts.googleapis.com/css2?family=Manrope:wght@400;700&display=swap");
:root { --pico-font-family-sans-serif: "Manrope", system-ui, sans-serif; }
</style>
```

Import from a `Head` injection so the font is available before first paint.

## Design note: a live theme editor (idea, not built)

There is no visual theme editor today. You edit the CSS by hand.

It is buildable, though, with pieces that already exist. A theme editor could ship as a **custom layout** — the same mechanism the [[en/user/kanban|Kanban board]] rides on: a file in `_layouts/`, selected with `layout:` frontmatter. That layout would render color pickers, live-preview the page against the chosen `--pico-*` values, and on save call the `updateHtmlInjection` GraphQL mutation to write the generated `<style>` block into the same `Head` injection you fill in by hand today.

Everything it needs is in place: custom layouts, the admin HTML-injection store, and a GraphQL mutation to write to it. What's missing is the editor UI itself. Until someone builds it, this page is the manual path — and the manual path is only a `<style>` block.

## See also

- [[en/user/default-template|Default template]] — structure, header, footer, sidebars
- [[en/user/bem|BEM naming in templates]] — the class names you can target
- [[en/user/yield_blocks|yield_blocks]] — per-page CSS/JS instead of global
- [[en/user/templates|Custom templates]] — replace the template entirely
