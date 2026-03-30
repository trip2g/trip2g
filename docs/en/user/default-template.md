---
title: Default Template
free: true
lang_redirect: "[[ru/user/default-template]]"
---

This site is built with the default template — what you're looking at right now is it in action.

The default template is the built-in page layout that trip2g uses when you don't specify a custom template. It provides a complete publishing setup with header, footer, sidebar navigation, table of contents, backlinks, and a magazine-style grid for displaying related notes.

You control the layout entirely through frontmatter—no HTML coding required.

### What the default template includes

Every page rendered with the default template has these optional sections:

- **Header** — site logo and navigation (pulled from a markdown note)
- **Left sidebar** — table of contents, backlinks, or custom navigation
- **Right sidebar** — outgoing links, custom widgets, or empty
- **Content area** — your note's markdown rendered as HTML
- **Magazine grid** — cards showing related notes (featured, grid, or list layout)
- **Footer** — site footer with columns (pulled from a markdown note)

You choose which sections appear on each page by setting frontmatter properties.

### Controlling layout with frontmatter

#### Header

Reference a markdown note to use as your site header:

```yaml
---
header: [[_nav]]
---
```

The header template extracts:
- The **first image** as the logo
- The **first list** as navigation links

Example header note `_nav.md`:

```markdown
---
title: Navigation
---

![Logo](/logo.png)

- [Home](/)
- [Docs](/docs)
- [About](/about)
```

**To hide the header:** Set `header: false`

#### Footer

Reference a markdown note to use as your site footer:

```yaml
---
footer: [[_footer]]
---
```

The footer can be a simple list or a nested list (which becomes columns):

```markdown
---
title: Footer
---

- [Home](/)
- [Docs](/docs)

### Company

- [About](/about)
- [Contact](/contact)

### Legal

- [Terms](/terms)
- [Privacy](/privacy)
```

Top-level items ("Company", "Legal") become column headings. Items under them become links in that column.

**To hide the footer:** Set `footer: false`

#### Left sidebar

The left sidebar typically shows table of contents and backlinks:

```yaml
---
left_sidebar:
  - TOC
  - Backlinks
---
```

**Available sidebar widgets:**

- `TOC` — interactive table of contents (headings from the current note)
- `Backlinks` or `inlinks` — notes that link to this note
- `outlinks` — links from this note to other notes
- `[[PageName]]` — embed another note by its title
- `path/to/file.md` — embed a note by file path

**Hide the left sidebar:** Set `left_sidebar: false` or omit it

**Auto-load sidebar from file:** If you don't set `left_sidebar` in frontmatter, trip2g automatically loads `_left_sidebar.md` if it exists. This is useful for shared navigation across all pages.

#### Right sidebar

The right sidebar is often used for extra information or links:

```yaml
---
right_sidebar:
  - outlinks
---
```

**Hide the right sidebar:** Set `right_sidebar: false` or omit it

**Auto-load sidebar from file:** If you don't set `right_sidebar` in frontmatter, trip2g automatically loads `_right_sidebar.md` if it exists.

### Content blocks

The `content` property controls which sections appear in the main content area:

```yaml
---
content:
  - self
  - magazine
---
```

**Available content blocks:**

- `self` or `selfcontent` — this note's article with title and body
- `magazine` — grid of related notes (see magazine layout section below)
- `[[PageName]]` — embed another note by title
- `path/to/file.md` — embed a note by file path

**Default behavior:**
- If you don't set `content`, only the note itself is shown (`self`)
- If the note is the site root (no index page), `magazine` is shown by default

### Magazine layout

The magazine displays related notes as cards in a three-tier visual hierarchy:

| Tier | Position | Cards | Style |
|------|----------|-------|-------|
| Featured | First | 1 | Large, full-width |
| Grid | 2nd–5th | 4 | Medium, 4-column grid |
| List | 6th+ | Unlimited | Minimal, vertical list |

Activate the magazine on an index or category page:

```yaml
---
title: Blog
content:
  - magazine
magazine_include_files: "blog/**/*.md"
magazine_sort_property: priority
---
```

#### Magazine properties

**`magazine_include_files`** — Glob pattern for which notes to include

```yaml
magazine_include_files: "blog/*.md"          # All posts in blog/
magazine_include_files: "posts/**/*.md"      # Recursively all .md in posts/
magazine_include_files: "docs/**/README.md"  # All README.md files
```

Default: `**/*.md` (all notes)

**`magazine_exclude_files`** — Glob pattern for which notes to exclude from the magazine

```yaml
magazine_exclude_files: "**/*Telegram.md"       # Exclude Telegram versions by name
magazine_exclude_files: "drafts/**"             # Exclude drafts folder
magazine_exclude_files: "archive/*.md"          # Exclude archived posts
```

Applied after `magazine_include_files` — first notes are included by the include glob, then matches of the exclude glob are removed.

Not set by default — no notes are excluded.

**`magazine_exclude_property`** — Exclude notes that have a specific frontmatter field

```yaml
magazine_exclude_property: telegram_publish_at
```

Notes with this field in their frontmatter will be hidden from the magazine. Useful when Telegram versions of notes live alongside web versions in the same folder.

Not set by default — no notes are excluded by property.

#### Excluding Telegram notes from magazine

When you publish to both web and Telegram, the Telegram versions often sit in the same folder. Two ways to exclude them:

**By filename pattern** — if Telegram notes follow a naming convention (e.g. `My Post. Telegram.md`):

```yaml
---
title: Daily Reports
content:
  - magazine
magazine_include_files: "reports/**/*.md"
magazine_exclude_files: "**/*Telegram.md"
---
```

**By frontmatter property** — if Telegram notes have `telegram_publish_at` (or similar) in frontmatter:

```yaml
---
title: Daily Reports
content:
  - magazine
magazine_include_files: "reports/**/*.md"
magazine_exclude_property: telegram_publish_at
---
```

Both approaches can be combined.

**`magazine_sort_property`** — Sort cards by a custom frontmatter field

```yaml
magazine_sort_property: priority
```

Notes that have this frontmatter field are listed first (sorted by value descending), then the rest sorted by creation date descending.

Example:

```yaml
---
title: Featured Post
priority: 100
---
```

**`magazine_include_property`** — Filter cards: only show notes with a specific frontmatter field

```yaml
magazine_include_property: featured
```

Only notes with `featured: true` (or any truthy value) in their frontmatter will appear.

Example note:

```yaml
---
title: A Great Post
featured: true
---
```

#### Magazine cards

Each magazine card shows:

- **Thumbnail** — the first image in the note (if any)
- **Title** — from frontmatter
- **Description** — from the `description` frontmatter field (or the first paragraph if no description is set)
- **Link** — to the full note

### Complete example

A typical blog index page might look like:

```yaml
---
title: Blog
content:
  - magazine
magazine_include_files: "blog/**/*.md"
magazine_sort_property: featured_priority
magazine_include_property: published
left_sidebar:
  - [[Blog Categories]]
right_sidebar: false
header: [[_nav]]
footer: [[_footer]]
---

Welcome to the blog. Browse recent posts below.
```

Then each blog post:

```yaml
---
title: My First Post
published: true
featured_priority: 100
description: A quick overview of this post
---

Full article content in markdown...
```

### Sidebar as navigation

You can create a reusable sidebar by storing navigation in a separate markdown file:

**`_left_sidebar.md`:**

```markdown
---
title: Docs Navigation
---

### Getting Started

- [Quick Start](/docs/quick-start)
- [Installation](/docs/install)

### Advanced

- [Custom Templates](/docs/templates)
- [API Reference](/docs/api)
```

Every page will auto-load this sidebar (unless you explicitly set `left_sidebar: false` in a note's frontmatter).

Use `###` (h3) headings for section group labels, not bold text (`**text**`). The sidebar UI renders `###` headings as proper section labels; bold text does not receive the same visual treatment.

### Using images in header and footer

**Header logo:**

The first image in the header note becomes the site logo:

```markdown
![Site Logo](/assets/logo.png)

- [Home](/)
- [Docs](/docs)
```

**Footer images:**

Images in footer notes are rendered normally as part of the content.

### Examples

#### Landing page with magazine grid

```yaml
---
title: Home
content:
  - self
  - magazine
magazine_include_files: "**/*.md"
magazine_sort_property: featured_on_home
left_sidebar: false
right_sidebar: false
---

# Welcome

Explore our latest content below.
```

Then mark notes to show on the homepage:

```yaml
---
title: Important Topic
featured_on_home: true
---
```

#### Documentation site

```yaml
---
title: Documentation
content:
  - self
left_sidebar:
  - _sidebar.md
right_sidebar:
  - TOC
  - outlinks
---
```

#### Blog with featured section

```yaml
---
title: Blog
content:
  - magazine
magazine_include_files: "blog/**/*.md"
magazine_sort_property: featured
---
```

Blog posts:

```yaml
---
title: Latest News
featured: 50
description: Summary of recent updates
---
```

### Switching to a custom template

If you need more control than frontmatter provides, you can create a custom template. See [[templates]] for the full template system.

```yaml
---
layout: my-custom-template
title: My Page
---
```

Any note can use either the default template (by omitting `layout`) or a custom one.

### Flexibility through frontmatter patches

Manually adding `header`, `footer`, `left_sidebar`, and `lang` to every note doesn't scale. Frontmatter patches let you set those properties once for an entire folder.

For example, this documentation site is configured like this:

```yaml
# All pages are public
**/*.md → {free: true}

# Russian sidebar for the entire RU section
ru/user/**/*.md → {left_sidebar: "ru/user/_sidebar.md"}

# Russian header and footer
ru/**/*.md → {header: "[[ru/_header]]", footer: "[[ru/_footer]]"}
```

No individual note knows about the template — everything is injected from outside via patches.

→ [[en/user/frontmatter-patches|Full frontmatter patches documentation]]

---

## Telegram post links

If a note has been published to a Telegram channel, the template shows a blue "Read in Telegram" button above the title. Clicking it opens the Telegram post directly.

The button appears automatically in three cases:

### 1. Published via trip2g

When you publish a note to Telegram through trip2g's publishing system, the button appears automatically. If the note was published to multiple channels, each channel gets its own button with the channel name.

No frontmatter needed — trip2g tracks sent messages in the database.

### 2. Imported from Telegram

Notes imported from a Telegram channel already have the link in frontmatter:

```yaml
---
telegram_publish_channel_id: "1234567890"
telegram_publish_message_id: 42
telegram_publish_message_link: https://t.me/c/1234567890/42
---
```

The `telegram_publish_message_link` field is set automatically during import. The button uses this URL directly.

For public channels the link format is `https://t.me/channelname/42` — in this case the button shows the channel name: "Read on @channelname".

### 3. Alternatives (cross-linking to Telegram version)

If you have a web version and a separate Telegram version of the same content, use the `alternatives` frontmatter field:

```yaml
---
title: My Article
alternatives:
  - "[[My Article. Telegram]]"
---
```

The linked note (`My Article. Telegram`) must either have `telegram_publish_message_link` in its frontmatter or be published via trip2g's publishing system. The template resolves the wikilink, extracts the Telegram link, and shows the button on the parent page.

This is useful when you publish a shorter or differently formatted version to Telegram and want to link between the two.

### Priority

The template checks sources in this order:
1. **Database** (published via trip2g) — checked first, supports multiple channels
2. **Frontmatter `telegram_publish_message_link`** — fallback for imported notes
3. **Alternatives** — resolved from linked notes
