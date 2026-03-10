---
title: "Default Template"
slug: default_template
free: true
---

The default template is the unified page rendering system for trip2g. It replaces the previous fragmented rendering approach (`renderlayout`, `rendernotepage`) with a single, composable architecture.

## Overview

`internal/defaulttemplate` handles all page rendering:

- **Single entry point**: `defaulttemplate.Render(ctx)` for all page types
- **Unified Ctx**: All rendering data flows through a single `Ctx` struct
- **Composable layouts**: Header, sidebars, content, and footer are independently configurable via frontmatter
- **Rich widgets**: Table of Contents, backlinks, outlinks, and embedded content
- **Magazine layout**: Multi-tier card system for content discovery
- **Responsive design**: Mobile-friendly with CSS breakpoints at 1024px and 640px

## Ctx Fields

The `Ctx` struct holds all data needed to render a complete page:

| Field | Type | Purpose |
|-------|------|---------|
| `Note` | `*templateviews.Note` | The main note being rendered |
| `Notes` | `*templateviews.NVS` | Note lookup service for sidebars and magazine |
| `Title` | `string` | Page `<title>` |
| `JSURLs` | `[]string` | Script URLs to load before `</body>` |
| `CSSURLs` | `[]string` | External stylesheet URLs |
| `DevMode` | `string` | Development mode flag (passed to JS) |
| `MetaDescription` | `*string` | SEO meta description |
| `MetaRobots` | `string` | Robots meta tag value |
| `OGTags` | `map[string]string` | Open Graph tags (`og:image`, `og:type`, etc.) |
| `HTMLInjections` | `map[string][]db.HtmlInjection` | Custom HTML to inject into `<head>` and end of `<body>` |
| `HrefLangs` | `[]HrefLang` | Alternate language links for SEO |
| `HTMLLang` | `string` | Language attribute for `<html>` tag |
| `OnboardingMode` | `bool` | Render onboarding screen instead of note |
| `PaywallError` | `*PaywallError` | If set, render paywall page |
| `UserToken` | `*usertoken.Data` | Current user (check via `.IsAdmin()`) |
| `Lang` | `string` | Current language code (`"en"`, `"ru"`, etc.) |
| `IsAdmin` | `bool` | (Deprecated) Use `UserToken.IsAdmin()` instead |

## File Structure

All rendering logic lives in `internal/defaulttemplate/views.html`. The quicktemplate generator creates `views.html.go` with these functions:

| Function | Purpose |
|----------|---------|
| `Render(ctx)` | Entry point: routes to `PayWall`, `Onboarding`, or `NotePage` |
| `NotePage(ctx)` | Main page layout with header, sidebars, content, footer |
| `SiteHeader(ctx, note)` | Renders `<header>` from header note (logo + nav) |
| `SiteFooter(ctx, note)` | Renders `<footer>` from footer note |
| `NoteSidebar(ctx, position, widgets)` | Left/right sidebar with widgets |
| `TOCWidget(ctx)` | Table of Contents widget |
| `InLinksWidget(ctx)` | Backlinks widget |
| `OutLinksWidget(ctx)` | Outlinks widget (TBD) |
| `NoteContent(ctx)` | Main content area: routes to self-content, magazine, or embedded notes |
| `SelfContent(ctx)` | Article: `<h1>` + note HTML body |
| `Magazine(ctx)` | Magazine layout: featured, grid, and list tiers |
| `MagazineCard(ctx, item)` | Single card in magazine (featured/small/list) |
| `PayWall(ctx)` | Paywall page (shown when `ctx.PaywallError` is set) |
| `Onboarding(ctx)` | Onboarding page (shown when `ctx.OnboardingMode` is true) |

## Header

The site header is rendered from a note specified in frontmatter via `header: [[Navigation]]`.

### Logo

Extracted from the header note:
```
Logo URL = note.FirstImageURL()  // first ![alt](url) in the note
```

If a logo exists, it's rendered as:
```html
<a class="site-header__logo" href="/">
  <img src="/logo.png" alt="Logo">
</a>
```

### Navigation

Extracted from the header note:
```
Nav HTML = note.FirstListHTML()  // first <ul> in rendered HTML
```

The HTML is inserted as-is into the nav element:
```html
<nav class="site-header__nav">{nav HTML}</nav>
```

### Multilevel Menu Support

The nav list can be nested with arbitrary depth:
```markdown
# Navigation

- [Home](/)/
- [Docs](/docs)
  - [Getting Started](/docs/getting-started)
  - [API Docs](/docs/api)
    - [Notes](/docs/api/notes)
    - [Users](/docs/api/users)
- [About](/about)
```

Each list level renders as a nested `<ul>` inside the previous `<li>`. CSS handles the indentation and visual layout.

## Sidebar

Sidebars appear left and right of the main content and contain widgets. Configure via frontmatter:

```yaml
left_sidebar:
  - TOC
  - inlinks
  - [[Custom Content]]

right_sidebar:
  - outlinks
```

### Widget Types

| Widget | ID | Purpose |
|--------|----|---------
| `TOC` | `toc` | Table of Contents (JavaScript-enhanced) |
| `InLinks`, `Backlinks` | `inlinks` | Notes linking to this note |
| `OutLinks` | `outlinks` | Links from this note (reserved) |
| Content | `[[Title]]` or `path.md` | Embed another note or file |

### Configuration

- `left_sidebar: false` — disable left sidebar
- `left_sidebar: null` — no sidebar (default)
- `left_sidebar: [TOC, inlinks]` — list of widgets

Each widget renders into a `<div class="widget">` with title and content.

## Magazine

The magazine layout displays multiple notes in three tiers:

| Tier | Indices | Visual | Use Case |
|------|---------|--------|----------|
| Featured | 0 | Large card, full-width | Top/hero item |
| Grid | 1–4 | Smaller cards, auto-fill grid | Secondary items |
| List | 5+ | Minimal cards, vertical list | Extended browsing |

### Activating Magazine

In a note's frontmatter, set:
```yaml
content:
  - magazine
```

Or make a note the index without explicit `content` (automatic magazine for `_index.md`).

### Configuration

| Setting | Default | Purpose |
|---------|---------|---------|
| `magazine_property` | `"magazine_priority"` | Frontmatter key used for sorting |
| `magazine_include_files` | `"**/*.md"` | Glob pattern for notes to include |

### Example

`_index.md`:
```yaml
---
content:
  - magazine
magazine_property: priority
magazine_include_files: "posts/**/*.md"
---
```

`posts/post1.md`:
```yaml
---
title: "My Post"
priority: 100
---
```

The magazine will load all notes matching `posts/**/*.md`, sort by `priority` descending, and display the highest-priority notes as featured/grid/list.

### Image Extraction

Each magazine card displays an image from `note.FirstImageURL()`:
- Searches the note's embedded images (markdown `![](url)`)
- Returns the first match, or empty string

The featured card shows images at 280px height; grid cards at 200px.

## Footer

The site footer is rendered from a note specified in frontmatter via `footer: [[Footer]]`.

### Content

The footer displays the note's `FirstListHTML()` if it exists; otherwise, the full HTML body.

### List Structure for Columns

If the footer note contains a list, it renders as columns:
```markdown
# Footer

- [Product](/)
  - [Features](/features)
  - [Pricing](/pricing)
- [Company](/company)
  - [About](/about)
  - [Contact](/contact)
```

Top-level items become column headers; nested items become column links.

## Content Types

The `content` frontmatter key controls what appears in the main area. It's a list; each item renders in order.

```yaml
content:
  - selfcontent
  - [[Related Articles]]
  - magazine
```

### SelfContent

Renders the note's own article with `<h1>` title + HTML body.

### Magazine

Renders a magazine layout (see above).

### WikiLink

Syntax: `[[Title]]`. Resolves via note permalink and embeds the linked note's HTML.

### File

Syntax: `"path/to/file.md"`. Resolves via file path and embeds that note's HTML.

## CSS and PicoCSS

The default template uses **PicoCSS** v2.1.1 as the base stylesheet, with custom BEM classes for layout.

### Base Classes

| Class | Element |
|-------|---------|
| `.site-header` | Top navigation bar |
| `.layout` | Main 3-column grid (left sidebar, main, right sidebar) |
| `.layout__sidebar--left/right` | Sidebar containers |
| `.layout__main` | Main content area |
| `.widget` | Sidebar widget container |
| `.widget__title` | Widget heading |
| `.widget__list` | Widget link list |
| `.content__title` | Article `<h1>` |
| `.content__body` | Article content |
| `.magazine` | Magazine container |
| `.magazine__grid` | 4-column grid tier (indices 1–4) |
| `.magazine__list` | List tier (indices 5+) |
| `.card` | Magazine card |
| `.card--featured` | Featured tier card |
| `.card--small` | Grid tier card |
| `.card--list` | List tier card |
| `.site-footer` | Footer container |

### Responsive Breakpoints

| Breakpoint | Change |
|-----------|--------|
| 1024px | Right sidebar hidden, left sidebar remains |
| 640px | Both sidebars hidden, single-column layout |

### Dark Mode

PicoCSS supports dark mode via the `data-theme` attribute:
```html
<html data-theme="dark">
```

Light is default. CSS variables (e.g., `var(--pico-primary)`, `var(--pico-background-color)`) automatically adapt.

### Rebuilding CSS

The default template CSS is embedded in the binary via `//go:embed defaulttemplate.css` in `embed.go`.

To rebuild after editing `assets/defaulttemplate/src/index.scss`:

```bash
npm run defaulttemplate-css
```

This compiles SCSS to CSS and places it at `internal/defaulttemplate/defaulttemplate.css`.

## Table of Contents JavaScript

The TOC widget is enhanced by JavaScript at runtime. It reads JSON data embedded in the template and builds an interactive nav with active-heading tracking.

### How It Works

1. **Server-side (template)**: TOC items are serialized to JSON and embedded in a `<script type="application/json" class="widget__data">` tag.

   ```html
   <script type="application/json" class="widget__data">
     [
       {"Text": "Introduction", "Level": 1, "ID": "introduction"},
       {"Text": "Getting Started", "Level": 1, "ID": "getting-started"},
       {"Text": "Installation", "Level": 2, "ID": "installation"}
     ]
   </script>
   ```

2. **Client-side (JS)**: The script parses the JSON, renders links into `#dt-toc-nav`, and attaches an IntersectionObserver.

3. **IntersectionObserver**: Tracks which heading is currently in view (80% from bottom of viewport) and marks it `.active` in the TOC.

### Data Format

Each TOC item has:
```typescript
interface TOCItem {
  Text: string;      // heading text
  Level: number;     // h1=1, h2=2, h3=3, etc.
  ID: string;        // heading id attribute
}
```

### CSS Styling

Links are indented based on level:
```scss
#dt-toc-nav a {
  padding-left: calc(var(--data-level, 1) * 0.75rem);

  &:hover, &.active {
    color: var(--pico-primary);
  }
}
```

The `--data-level` CSS variable is set via JavaScript on each link.

### Rebuilding TOC

The TOC JavaScript is compiled from `assets/toc/src/index.ts` to `assets/toc/toc.js` using esbuild:

```bash
npm run toc
```

The compiled script is then embedded in `assets/embed.go` and served at `/assets/toc/toc.js`.

## Frontmatter Reference

All frontmatter keys supported in notes:

| Key | Type | Default | Purpose |
|-----|------|---------|---------|
| `header` | `[[Title]]` or `false` | `false` | Reference to header note |
| `footer` | `[[Title]]` or `false` | `false` | Reference to footer note |
| `left_sidebar` | `[widget, ...]` or `false` | `false` | Left sidebar widgets |
| `right_sidebar` | `[widget, ...]` or `false` | `false` | Right sidebar widgets |
| `content` | `[type, ...]` | `[selfcontent]` | Content blocks to render |
| `magazine_property` | `string` | `"magazine_priority"` | Sort key for magazine |
| `magazine_include_files` | `string` | `"**/*.md"` | Glob for magazine notes |
| `title` | `string` | — | Page title (required) |
| `description` | `string` | — | SEO meta description |
| `free` | `bool` | `false` | Accessible without paywall |

### Widget Values

For `left_sidebar` and `right_sidebar`:

- `"TOC"` or `"toc"` — Table of Contents
- `"InLinks"`, `"Backlinks"`, `"inlinks"` — Backlinks widget
- `"OutLinks"`, `"outlinks"` — Outlinks widget
- `"[[Title]]"` — Embed note by title
- `"path/to/file.md"` — Embed note by file path

### Content Values

For `content`:

- `"selfcontent"` or `"self"` — This note's article
- `"magazine"` — Magazine layout (filtered by `magazine_property`)
- `"[[Title]]"` — Embed note by title
- `"path/to/file.md"` — Embed note by file path

## Ctx Methods

The `Ctx` struct provides helper methods for parsing frontmatter:

```go
// SidebarWidgets returns widgets for "left" or "right".
// Returns nil if not configured or set to false.
func (ctx *Ctx) SidebarWidgets(position string) []WidgetRef

// ContentRefs returns the list of content blocks.
func (ctx *Ctx) ContentRefs() []ContentRef

// HeaderRef / FooterRef — note reference or ContentRefNone
func (ctx *Ctx) HeaderRef() ContentRef
func (ctx *Ctx) FooterRef() ContentRef

// MagazineProperty / MagazineIncludeFiles — with defaults
func (ctx *Ctx) MagazineProperty() string      // default: "magazine_priority"
func (ctx *Ctx) MagazineIncludeFiles() string  // default: "**/*.md"

// MagazineItems returns sorted magazine items with visual tiers.
func (ctx *Ctx) MagazineItems() []MagazineItem
```

## Magazine Items

The `MagazineItem` struct represents a single card in the magazine:

```go
type MagazineItemSize int
const (
    MagazineItemFeatured  // index 0
    MagazineItemSmall     // index 1-4
    MagazineItemList      // index 5+
)

type MagazineItem struct {
    Note     *templateviews.Note
    Size     MagazineItemSize
    ImageURL string
}
```

Size is determined automatically based on the item's index in the sorted list.

## Internationalization

The template supports multiple languages via the `ctx.Lang` field and built-in translation keys.

### Supported Languages

Currently:
- `en` — English
- `ru` — Russian

### Translation Keys

Common strings used in the template:

- `widget.toc.title` — "Table of Contents"
- `widget.inlinks.title` — "Backlinks"
- `widget.outlinks.title` — "Related Links"

### Adding a Language

Create a new locale file at `internal/defaulttemplate/locales/{lang}.json`:

```json
{
  "widget.toc.title": "Table of Contents",
  "widget.inlinks.title": "Backlinks"
}
```

Then embed it in `embed.go` and use it in the template via `ctx.T("widget.toc.title")`.

## Examples

### Magazine Index

`_index.md`:
```yaml
---
content:
  - magazine
magazine_property: priority
---
```

With posts:
```yaml
---
title: "My Post"
priority: 100
---
Content...
```

Result: Magazine layout sorted by priority, featured + grid + list.

### Article with TOC and Backlinks

`docs/article.md`:
```yaml
---
title: "Understanding Layouts"
left_sidebar:
  - TOC
  - inlinks
content:
  - selfcontent
---

## Section 1
...
## Section 2
...
```

Result: Two-column layout with TOC on left, article on right.

### Page with Custom Header/Footer

`guide.md`:
```yaml
---
title: "Complete Guide"
header: [[Navigation]]
footer: [[Site Footer]]
content:
  - selfcontent
---
```

Result: Header at top, footer at bottom, full-width article in between.

## See Also

- [Layouts API](./layouts.md) — `templateviews` API for templates
- [Note Rendering](./note_rendering.md) — HTML generation details
