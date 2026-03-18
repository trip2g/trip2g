---
home_position: 1
title: Publishing notes
free: true
lang_redirect: "[[ru/user/Свойства заметок]]"
---

Properties in your note's frontmatter control how it appears on your site.

### How to add properties

Three ways to open the properties editor in Obsidian:

- Press **Cmd/Ctrl + ;**
- Right-click the file tab → **Add file property**
- Type `---` at the very start of a file

Properties are stored as YAML at the top of the file:

```yaml
---
title: My note
free: true
---
```

### Property types

Obsidian supports several data types. Using the correct type matters — for example, `telegram_publish_at` must be **Date & time**, not Date.

| Type | Example | Description |
|------|---------|-------------|
| Text | `title: My page` | A text string |
| Checkbox | `free: true` | True/false toggle |
| List | `tags: [news, updates]` | Multiple values |
| Date & time | `2025-01-15T09:00` | Date with time |

### Core properties

**free** (Checkbox)

Makes the note publicly visible. Without this property, only logged-in users (admins and subscribers) can read the page.

```yaml
free: true
```

**title** (Text)

The page heading. Defaults to the filename if not set.

```yaml
title: Getting started
```

**description** (Text)

SEO description and social preview text shown when the link is shared.

```yaml
description: A step-by-step guide to setting up trip2g
```

**slug** (Text)

A custom URL path that replaces the auto-generated URL derived from the filename.

```yaml
```

Result: `yoursite.com/getting-started` instead of `yoursite.com/My_Note_Title`

**subgraphs** (Text or List)

Attaches the note to a course or section with its own sidebar.

```yaml
subgraphs: premium
```

### Telegram properties

**telegram_publish_at** (Date & time)

Schedules the note as a Telegram post. Select type **Date & time** — not plain Date — or the scheduler will not recognize the value.

```yaml
telegram_publish_at: 2025-01-15T09:00
```

**telegram_publish_tags** (List)

Tags that link the note to Telegram channels. The note publishes to every channel whose tags match.

```yaml
telegram_publish_tags:
  - news
  - announcements
```

### What gets synced

The plugin uploads all notes in the configured sync folder. Use the **Publish fields** setting in the plugin to filter by property: entering `publish` means only notes with `publish: true` are uploaded. Leave it blank to sync everything.

### Keyboard shortcuts

- **Cmd/Ctrl + ;** — add a property
- **Tab / Shift+Tab** — move between properties
- **Cmd/Ctrl + Backspace** — delete a property

### Markdown formatting

Notes are written in Markdown. The following elements are supported:

| Element | Syntax |
|---------|--------|
| Headings | `# H1`, `## H2`, `### H3` |
| Bold | `**bold**` |
| Italic | `*italic*` |
| Strikethrough | `~~strikethrough~~` |
| Inline code | `` `code` `` |
| Code block | ` ```language ``` ` (with syntax highlighting) |
| Link | `[text](https://example.com)` |
| Wikilink | `[[Note name]]` |
| Image | `![alt](path/to/image.png)` |
| Blockquote | `> quote` |
| Unordered list | `- item` |
| Ordered list | `1. item` |
| Table | `\| Col \| Col \|` |
| Checkbox | `- [ ] task` / `- [x] done` |
| Horizontal rule | `---` |

Wikilinks (`[[Note name]]`) are the core navigation feature — they create links between notes that resolve to URLs on your published site.

### Next steps

- [[en/user/telegram]] — schedule posts to a Telegram channel using note properties
- [[en/user/monetization]] — restrict content to paying subscribers
- [[en/user/frontmatter-patches]] — apply properties to entire folders at once, without editing each note
- [[en/user/templates]] — control page layout with custom templates
