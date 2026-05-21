# Obsidian Bases — Integration Research

> Researched 2026-05-21. Bases is a core Obsidian feature released in 1.9.0 (public: 1.9.10, 2025-08-18).

## What is Bases

A `.base` file is **plain YAML** that defines a query view over the vault's frontmatter index. It doesn't store data — it reads existing `.md` frontmatter. Conceptually similar to Dataview tables but built-in, faster, and interactively editable.

```yaml
# example: blog.base
filters:
  - file.inFolder("blog")
views:
  - type: table
    name: "Posts"
    groupBy: { property: status, direction: DESC }
    order: [file.ctime]
    filters:
      - 'status != "draft"'
formulas:
  slug: 'file.name.toLowerCase().replace(" ", "-")'
```

### Top-level YAML schema

| Key | Purpose |
|-----|---------|
| `filters` | Global filter narrowing the note set |
| `formulas` | Computed properties (expression language) |
| `properties` | Display-name overrides per column |
| `summaries` | Custom aggregation expressions |
| `views` | List of view definitions (table, map, …) |

### Available properties

**File-system (always available, read-only):**

| Property | Type | Description |
|----------|------|-------------|
| `file.name` | String | Filename without extension |
| `file.path` | String | Full vault-relative path |
| `file.folder` | String | Parent folder |
| `file.tags` | List | All tags (content + frontmatter) |
| `file.links` | List | Internal wikilinks |
| `file.backlinks` | List | Backlinks |
| `file.ctime` / `file.mtime` | Date | Created / modified |
| `file.size` | Number | Bytes |

**Note (frontmatter):** any YAML key accessible as `note.fieldname` or just `fieldname`.

**Filter functions:** `hasTag()`, `hasLink()`, `inFolder()`, `if()`, `contains()`, `startsWith()`, `now()`, `date()` — and arithmetic/boolean operators.

## Server-side feasibility

`.base` files are **plain YAML** — standard Go YAML parser handles them. There is no external Bases engine; it runs only inside Obsidian. Obsidian Publish does not render Bases yet.

To execute Bases queries server-side, trip2g would implement a subset of the filter expression language in Go. For navigation use cases only basic functions are needed: `inFolder`, `hasTag`, `hasLink`, property comparisons.

## Integration plan

### Phase 1 — sync + metadata reading (small)

| Step | Location | Notes |
|------|----------|-------|
| Add `.base` to extension allowlist | `internal/case/pushnotes/resolve.go` | One-line change |
| Parse `.base` YAML on sync | `internal/mdloader/loader.go` | Reuse existing YAML parser |
| Store in `notes` table with `type: base` frontmatter | existing schema | No migration needed |
| Expose to Jet templates as a `BaseView` | `templateviews` | Read-only, metadata only |

### Phase 2 — filter engine (medium)

Implement a `internal/bases/` package that evaluates `filters` + `order` against the note index. Powers navigation lists and collection pages.

### Phase 3 — full query execution (large, future)

Formulas, summaries, `groupBy`, multiple view types. Needs more complete expression language parser.

## Use cases for trip2g

### 1. Navigation as a Base query

A user creates `sidebar-nav.base` in their vault:

```yaml
filters:
  - file.inFolder("docs")
  - 'published == true'
views:
  - type: table
    order: [note.order, file.name]
```

Trip2g reads the YAML, executes filters against its note index, returns an ordered list for Jet templates. Replaces fragile `magazine_include_files` frontmatter with a first-class Obsidian-native config that users can edit visually.

### 2. Structured collections for templates

A Base defines a "blog", "portfolio", or "docs index" — the author configures it once in Obsidian, trip2g materializes it on the website. Cleaner than the current `magazine_*` frontmatter approach.

### 3. Default template navigation

`header: blog.base` instead of (or alongside) `header: [[Navigation]]` — the base provides both structure and sort order without needing a hand-maintained navigation note.

## Current constraints

- Obsidian plugin (`obsidian-sync/src/main.ts`) already handles arbitrary file content — no plugin changes needed to start syncing `.base` files.
- Extension allowlist in `internal/case/pushnotes/resolve.go` must be updated (currently only `.md`, `.html`, `.html.json`).
- No external Bases library exists — filter execution requires custom Go implementation.
- Obsidian Publish does not support Bases — trip2g would be ahead of the official tooling.
- `.base` format had a breaking change in Obsidian 1.9.2 (restructured `properties` section) — parse against the current schema.

## References

- [Introduction to Bases — Obsidian Help](https://obsidian.md/help/bases)
- [Bases syntax (full schema)](https://obsidian.md/help/bases/syntax)
- [Obsidian 1.9.0 changelog](https://obsidian.md/changelog/2025-05-21-desktop-v1.9.0/)
- [Dataview vs Bases comparison](https://obsidian.rocks/dataview-vs-datacore-vs-obsidian-bases/)
