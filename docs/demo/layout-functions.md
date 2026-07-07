---
free: true
title: Custom Layout Cookbook
---

**TL;DR.** A custom layout is one [Jet](https://github.com/CloudyKit/jet) template file at `_layouts/<name>.html`, assigned per note with `layout: <name>` frontmatter. Inside it you get the current note (`note`), every note in the vault (`nvs`), the standard site chrome (`defaultTemplate`), and AST-level access to the note's sections. Live examples: [[layout-sections]], [[layout-include]], [[layout-magazine]].

## Setup

1. Put a Jet template in your vault: `_layouts/showcase/sections.html`
2. Point a note at it:

```yaml
---
layout: showcase/sections
---
```

Auto-escaping is off: layouts are trusted vault content, so `{{ note.HTMLString() }}` outputs raw HTML directly.

## The current note: `note.*`

Methods need parentheses.

| Snippet | Returns |
|---|---|
| `{{ note.Title() }}` | note title |
| `{{ note.HTMLString() }}` | full rendered HTML body |
| `{{ note.ContentString() }}` | raw markdown source |
| `{{ note.Path() }}` / `{{ note.Permalink() }}` | vault path / URL path |
| `{{ note.Description() }}` | SEO description |
| `{{ note.Author() }}`, `{{ note.Tags() }}`, `{{ note.CreatedAt() }}`, `{{ note.UpdatedAt() }}` | frontmatter meta |
| `{{ note.ReadingTime() }}` | minutes, estimated |
| `{{ note.TOC() }}` | table-of-contents headings |
| `{{ note.M().GetString("key", "default") }}` | any frontmatter field (also `GetInt`, `GetBool`, `GetStrings`, `Raw()`) |
| `{{ note.Lang() }}` | note language (also `LangName()`, `LangAlternative("ru")`) |

## AST sections: `note.PartialRenderer()`

Splits the note at headings — build your own markup around each piece.

```jet
{{ range i, sec := note.PartialRenderer().Sections(2) }}
  <section class="card">
    <h2>{{ sec.TitleHTML }}</h2>
    {{ sec.ContentHTML }}
  </section>
{{ end }}
```

| Snippet | Returns |
|---|---|
| `.Sections(2)` | list of sections split at H2; each has `.Title`, `.TitleHTML`, `.ContentHTML` |
| `sec.Sections(3)` | nested split — subsections inside a section |
| `.Section("Pricing")` | one section found by heading title (nil if absent) |
| `.Introduce()` | everything before the first heading (the lead) |
| `.FirstList()` / `.Lists()` | markdown lists as structured items (`.Text`, `.URL`, `.Children`) |
| `.FirstImageURL()` | URL of the note's first image |

## Other notes: `nvs.*`

Pull any note in the vault into the page.

```jet
{{ pulled := nvs.ByWikilink("_snippet-cta") }}
{{ if pulled }}{{ pulled.HTMLString() }}{{ end }}
```

| Snippet | Returns |
|---|---|
| `nvs.ByPath("/_snippet-cta.md")` | note by exact vault path |
| `nvs.ByWikilink("target")` | Obsidian-style resolution (basename lookup) |
| `nvs.ByPermalink("/about")` | note by URL |
| `nvs.BackLinks(note)` / `nvs.OutLinks(note)` | notes linking here / linked from here |
| `nvs.List()` | all notes |
| `nvs.ByGlob("blog/*.md")` | query builder: chain `.SortBy("created_at")` / `.SortByMeta("date")`, `.Desc()`/`.Asc()`, `.Limit(n)`, `.Offset(n)`, end with `.All()`, `.First()` or `.Last()` |

A recent-posts block in three lines:

```jet
{{ range i, post := nvs.ByGlob("blog/*.md").SortByMeta("date").Desc().Limit(5).All() }}
  <a href="{{ post.PermalinkEncoded() }}">{{ post.Title() }}</a>
{{ end }}
```

Every returned note is the same view as `note` — call `.Title()`, `.HTMLString()`, even `.PartialRenderer()` on it.

## Site chrome: `defaultTemplate.*`

Reuse the standard design instead of rebuilding it.

| Snippet | Emits |
|---|---|
| `{{ defaultTemplate.Styles() }}` | standard stylesheet `<link>`s — Pico variables, typography, dark mode |
| `{{ defaultTemplate.Header() }}` | the normal site header |
| `{{ defaultTemplate.Footer() }}` | the normal site footer |
| `{{ defaultTemplate.UserSpaceScripts() }}` | sign-in / search widget scripts |

`{{ currentUser.IsAdmin() }}` gates admin-only markup.

## Layout-file plumbing

| Snippet | Does |
|---|---|
| `{{ include "components/header.html" }}` | inline another template file |
| `{{ import "blocks" }}` + `{{ yield my_block() }}` | reusable named blocks |
| `{{ asset("styles.css") }}` | URL of a file next to the layout |
| `{{ yield_blocks("prefix") }}` | collect CSS blocks from used components |

## Gotchas

- **`range` with one variable gives the index, not the value.** `{{ range x := items }}` makes `x` an int. Always use two variables: `{{ range i, item := items }}`.
- **`_` is not a valid range variable** — use a named one even if unused: `{{ range i, item := items }}`.
