---
title: "AI Agent Instructions: Documentation Site on trip2g"
description: "Step-by-step instructions for an AI agent to set up a documentation site on trip2g. Covers vault structure, public access, header with logo, and navigation."
free: true
lang_redirect: "[[ru/user/ai-agent-docs-setup]]"
---

This document is an instruction set for an AI agent. Reading it, the agent knows exactly what to create to turn an Obsidian vault into a working documentation site on trip2g.

The agent runs inside an extracted vault folder. The goal: create the right files and tell the user what to do in the web interface.

---

## 1. Root `_index.md`

The vault must contain `_index.md` at its root. This is the site's home page — trip2g serves it at `/`.

Create `_index.md` in the vault root:

```markdown
---
title: Documentation
description: Project documentation.
content:
  - self
  - magazine
magazine_include_files: "**/*.md"
---

Welcome. Choose a section from the navigation or use search.
```

The file must be named exactly `_index.md`. Without the underscore it is a different file and will not serve as the home page.

---

## 2. Vault structure

Create a minimal file layout:

```
_index.md          — home page
_header.md         — site header (logo + navigation)
_footer.md         — footer (optional)
_left_sidebar.md   — left sidebar with navigation
logo.png           — logo file (if available)

docs/
  getting-started.md
  installation.md
  configuration.md
```

Files prefixed with `_` are hidden from the site's listing pages but still perform their function. Files without `_` are visible to visitors as regular notes.

---

## 3. Notes are hidden by default

All notes are visible only to the administrator by default. To make them public, the user must create a frontmatter patch in the web interface.

Tell the user to do the following:

1. Open the site admin panel
2. Go to **Notes & Content** → **Patches**
3. Click **+ Add**
4. Fill in the form:
   - **Include:** `**/*.md`
   - **Description:** `all free`
   - **Expression (Jsonnet):** `{ free: true }`
5. Click **Submit**

After this, all notes become publicly accessible. To hide specific notes, add `free: false` to their frontmatter.

For more on patches: [[en/user/frontmatter-patches|Frontmatter patches]].

---

## 4. Site header with logo

Create `_header.md` in the vault root. Trip2g automatically uses it as the header on every page — no additional configuration needed in individual notes.

The header extracts from the file:
- **First image** — becomes the site logo
- **First list** — becomes the navigation links

Example `_header.md` with a logo:

```markdown
![[logo.png]]

- [Home](/)
- [Documentation](/docs/getting-started)
- [About](/about)
```

If there is no logo, omit the image line:

```markdown
- [Home](/)
- [Documentation](/docs/getting-started)
- [About](/about)
```

The logo file (`logo.png`) must be present in the vault — alongside `_header.md` or in an `assets/` folder. The Obsidian link `![[logo.png]]` resolves it automatically.

---

## 5. Functional navigation notes

Trip2g automatically picks up four special files from the vault root:

| File | Role |
|------|------|
| `_header.md` | Header: logo + navigation |
| `_footer.md` | Footer |
| `_left_sidebar.md` | Left sidebar |
| `_right_sidebar.md` | Right sidebar |

You do not need all four — create only what the site requires.

**Example `_left_sidebar.md`** for a documentation site:

```markdown
### Getting Started

- [Introduction](/docs/getting-started)
- [Installation](/docs/installation)
- [Configuration](/docs/configuration)

### Reference

- [API](/docs/api)
- [FAQ](/docs/faq)
```

Use `###` for group headings, not bold text. The sidebar UI renders `###` as proper section labels.

**Example `_footer.md`**:

```markdown
- [Home](/)
- [Documentation](/docs/getting-started)

### Project

- [GitHub](https://github.com/yourproject)
- [Changelog](/changelog)
```

For more on functional notes and glob sections for different site areas: [[en/user/default-template|Default template]].

---

## 6. Content notes

Each documentation note is a plain markdown file. Frontmatter is minimal:

```markdown
---
title: Installation
description: How to install and run the project.
---

## Requirements

- Node.js 18+
- ...

## Steps
```

The `title` field is required — it is used as the page heading and in magazine cards. The `description` field is used for SEO and card previews.

---

## 7. Wikilinks / Internal links

This is an Obsidian vault. All internal links must use `[[wikilink]]` syntax, not markdown links like `[text](path)`.

Rules:

- `[[Note name]]` — links by note title, no extension or path needed
- `[[Note name|Display text]]` — with custom display text
- `![[logo.png]]` — embeds an image
- When two notes share the same filename in different folders, Obsidian resolves by the shortest unique path from the root. To avoid ambiguity, be explicit: `[[docs/installation]]` rather than `[[installation]]`
- Folder depth does not matter — `[[installation]]` finds `docs/installation.md` from anywhere in the vault

Markdown links `[text](path)` are also supported, but only for external URLs and in-page anchor links (`#section`). For navigation between vault notes, use `[[wikilinks]]` exclusively.

---

## 8. Multilingual sites

Trip2g supports multilingual documentation sites: parallel sections for each language, a language switcher, and automatic hreflang tags for SEO. See [[en/user/multilingual|Multilingual sites]] for details.

---

## Final checklist

Verify the vault contains:

- [ ] `_index.md` at the root with `content: [self, magazine]`
- [ ] `_header.md` with logo and navigation links
- [ ] `_left_sidebar.md` with documentation sections
- [ ] Content notes under `docs/`
- [ ] The `{ free: true }` patch configured in the web interface

After syncing the vault, the site is ready.
