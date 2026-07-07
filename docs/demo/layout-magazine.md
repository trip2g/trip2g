---
free: true
title: Your Notes, Magazine-Style
layout: showcase/magazine
tags:
  - layouts
  - showcase
---

One Markdown note, three template functions: the intro becomes this hero lead, each H2 becomes an alternating feature row, and the closing banner is pulled from a separate note.

## Sections drive the design

The layout calls `note.PartialRenderer().Sections(2)` and receives the note split at H2 headings — title and rendered HTML for each. The template decides the geometry: numbers, columns, alternation. The note stays plain prose.

## The chrome is reused, not rebuilt

`defaultTemplate.Styles()` links the standard stylesheet, so this page inherits the site's fonts, colors and dark mode. `defaultTemplate.Header()` drops in the normal site header. A custom layout starts from the design system, not from zero.

## Copy lives in notes

The colored banner at the bottom comes from `_snippet-cta.md` via `nvs.ByWikilink(...)`. Marketing copy, footers, recurring blocks — all editable in Obsidian, no template deploys.
