---
free: true
title: "Sections Layout: One Card per Chapter"
layout: showcase/sections
---

This note is ordinary Markdown, but its layout iterates the note's **AST sections**: every H2 below becomes a numbered card, and H3 subsections show up as chips. The text you are reading is the "intro" — everything before the first heading.

## Write in Obsidian

Nothing changes in your writing flow. Headings, lists, links — the layout only decides how the pieces are arranged on the page.

### Plain Markdown

No shortcodes, no page builders.

### Wikilinks work

Links like [[callouts]] resolve the same way they do in Obsidian.

## The layout splits the note

The template calls `note.PartialRenderer().Sections(2)` and gets a list of sections, each with a title and rendered HTML body. It wraps each one in its own markup — here, a card with a number.

## Restyle without rewriting

Change the layout file and every note using it is re-dressed instantly. The content never moves.

### One file

A layout is a single Jet template in `_layouts/`.

### Assigned per note

One line of frontmatter: `layout: showcase/sections`.
