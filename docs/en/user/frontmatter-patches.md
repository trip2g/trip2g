---
title: Frontmatter Patches
free: true
lang_redirect: "[[ru/user/frontmatter_patches]]"
---

You have 200 notes in `blog/` and every one needs `free: true`. Or you want all notes in `docs/` served from `docs.mysite.com`. Editing them by hand is pointless — write one rule and it applies to the whole folder.

A frontmatter patch works like this: you specify a path pattern and an expression that adds or changes note properties. The system applies the patch at load time, before rendering. Your notes stay clean — their frontmatter is never modified on disk.

### How this documentation site is configured

The trip2g docs you are reading right now use frontmatter patches to manage the multilingual structure. Here are the actual rules running on this site:

| Pattern | Expression | What it does |
|---------|-----------|--------------|
| `**/*.md` | `{lang: "en"}` | Sets English as the default language for all notes (priority −1, runs first) |
| `ru/**/*.md` | `{lang: "ru"}` | Overrides language to Russian for the `ru/` section (priority 0, runs after) |
| `**/*.md` | `{free: true}` | Makes all documentation pages publicly accessible |
| `ru/user/**/*.md` | `{left_sidebar: "ru/user/_sidebar.md"}` | Attaches the Russian sidebar to all RU user docs |
| `ru/**/*.md` | `{header: "[[ru/_header]]", footer: "[[ru/_footer]]"}` | Switches header and footer to their Russian versions |
| `ru/thoughts/**/*.md` | `{left_sidebar: "[[ru/thoughts/_sidebar]]"}` | Attaches the Russian philosophy sidebar |
| `en/thoughts/**/*.md` | `{left_sidebar: "[[en/thoughts/_sidebar]]"}` | Attaches the English philosophy sidebar |

The language cascade is worth noting: the first rule sets `lang: "en"` at priority −1 for everything. The second rule then sets `lang: "ru"` at priority 0 for `ru/**` — overriding only what it needs to. No note needs `lang` in its own frontmatter.

The [[en/user/default-template|default template]] reads `lang`, `left_sidebar`, `header`, and `footer` from the note's properties to decide what to render. Patches inject those properties at load time without touching the source files.

### Who needs this

- You have a free section and a paid section, and setting `free` on every note manually is tedious
- You want to assign a layout to a whole folder instead of each note individually
- You need to attach a subdomain to a section via [[en/user/advanced|custom routes]]

### Creating a patch

Patches are created in the admin panel under **Notes & Content**. Each patch has three parts:

1. **Path patterns** — which notes to match. Supports `*` (one folder level) and `**` (recursive)
2. **Expression** — what to add or change in the properties
3. **Priority** — order of application when you have multiple patches

### Examples

**Make all notes in `blog/` free:**

```
pattern: blog/*
expression: { free: true }
```

Every note in `blog/` gets `free: true`, even if its frontmatter doesn't have it.

**Assign a layout to a section:**

```
pattern: blog/*
expression: { layout: "blog_layout" }
```

All notes in `blog/` use the `blog_layout` template.

**Attach a folder to a subdomain:**

```
pattern: docs/**
expression: { route: "docs.mysite.com" }
```

All notes in `docs/` and subfolders automatically appear at `docs.mysite.com`. See [[en/user/advanced|custom domains]] for more on routes.

**Add a suffix to titles:**

```
pattern: *
expression: meta + { title: meta.title + " — My Site" }
```

`meta` contains the note's current properties. The expression takes the existing title and appends the site name.

**Conditional patch — set a layout only if not already set:**

```
pattern: *
expression: if std.objectHas(meta, "layout") then {} else { layout: "default" }
```

If the note already has a `layout`, the patch does nothing. Otherwise sets `default`.

### Priorities and chains

Patches apply in order: lower priority number first, then higher. Each patch sees the result of the previous ones.

| Priority | Pattern | Expression |
|----------|---------|-----------|
| 0 | `*` | `{ layout: "default" }` |
| 10 | `blog/*` | `{ layout: "blog_layout" }` |

For `blog/post.md`: rule 0 sets `layout: "default"`, then rule 10 overwrites it with `blog_layout`. All other notes keep `default`.

### Patterns

| Symbol | What it matches |
|--------|----------------|
| `*` | Any characters except `/`. Does not match hidden files (starting with `.`) |
| `**` | Any number of nested folders, including zero |
| `?` | Exactly one character, except `/` |
| `[abc]` | One character from the set |
| `{foo,bar}` | One of the alternatives |

**Include and exclude:** each patch has include patterns (what to match) and exclude patterns (what to skip). A note is patched if it matches at least one include and no exclude patterns.

```
include: ["blog/**"]
exclude: ["blog/drafts/*"]
```

### Expressions (Jsonnet)

Expressions are written in [Jsonnet](https://jsonnet.org/) — a language that extends JSON with variables, conditions, and functions. Simple cases look like plain JSON.

**Available variables:**

| Variable | Type | Contents |
|----------|------|---------|
| `meta` | object | Current note properties (after previous patches in the chain) |
| `path` | string | File path, e.g. `"blog/my-post.md"` |

The expression must return an object `{}`. Its contents are merged into the note properties. Existing keys are overwritten.

**Common patterns:**

```jsonnet
// Simple object — most common case
{ free: true, layout: "blog" }

// Read current properties with meta
meta + { title: meta.title + " — My Site" }

// Conditional
if std.objectHas(meta, "layout") then {} else { layout: "default" }

// Path-based logic
if std.startsWith(path, "premium/")
then { free: false }
else {}
```

### Vault-based patches

Instead of creating patches in the admin panel, you can define them as markdown files directly in your vault. The patch rules live next to your content, under version control, editable in Obsidian.

#### File format

Create a markdown file anywhere in your vault. Give it a frontmatter with `type: frontmatter-patch`, the patterns you want to match, and write the Jsonnet expression in a fenced code block in the body:

```markdown
---
type: frontmatter-patch
include:
  - blog/*
  - articles/**
exclude:
  - blog/premium/*
priority: 10
---

Makes all blog posts free, except the premium section.

```jsonnet
{ free: true }
```
```

**Frontmatter fields:**

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | yes | — | Must be `"frontmatter-patch"` |
| `include` | yes | — | Glob patterns for notes to match |
| `exclude` | no | `[]` | Glob patterns for notes to skip |
| `priority` | no | `0` | Lower numbers apply first (within vault patches) |

There is no `enabled` field. To disable a patch, delete or rename the file.

#### Visibility

Files with a `_` prefix — such as `_rules.md` or `_publish-rules.md` — are hidden from the published site but still work as patches. Files without the prefix appear as regular notes on the site and also work as patches. This lets you write the patch description in plain prose, link to it from other notes, and keep it visible as documentation.

#### Multiple patches in one file

Each fenced `jsonnet` block in the body becomes a separate patch. All blocks in a single file share the same `include`, `exclude`, and `priority` from the frontmatter:

```markdown
---
type: frontmatter-patch
include:
  - docs/**
priority: 5
---

Two rules for the docs section:

```jsonnet
{ layout: "doc" }
```

```jsonnet
{ free: true }
```
```

This creates two patches, both with `priority: 5` and `include: ["docs/**"]`.

#### Order of application

Admin panel patches apply first, then vault patches. Within each group, patches apply in priority order. This means vault patches can override admin panel rules.

#### Complete example

`docs/_publish-rules.md`:

```markdown
---
type: frontmatter-patch
include:
  - docs/**
  - guides/**
exclude:
  - docs/draft/*
priority: 20
---

Publishing rules for docs and guides.

All docs become free. Guides get the "guide" layout by default.
Draft notes are excluded.

```jsonnet
{ free: true }
```

```jsonnet
if std.startsWith(path, "guides/")
then { layout: "guide" }
else {}
```
```

#### Errors

If a patch file has problems, the site keeps working. The note displays a warning so you can see and fix the issue.

| Situation | What happens |
|-----------|-------------|
| No jsonnet block in the body | Warning on the note; file still renders normally |
| `include` patterns missing | Warning on the note |
| Invalid glob pattern | Warning on the note |
| Invalid Jsonnet syntax | Warning on the note |

The warning appears when you open the note in the published site. Other patches continue to apply normally.

### Troubleshooting

**Patch doesn't apply** — check your pattern: `blog/*` won't match `blog/drafts/post.md`. Use `blog/**` for nested folders.

**Expression error** — if the expression fails for a specific note (e.g., accessing a missing field), the patch is skipped only for that note. Other notes process normally; the site keeps working.

**Two patches conflict** — the one with higher priority (larger number) wins. At equal priority, the one created later wins.

**Accessing a missing field** — `meta.tags` fails if the note has no `tags`. Wrap in a check: `if std.objectHas(meta, "tags") then meta.tags else []`.
