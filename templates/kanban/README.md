# trip2g Kanban Template

A standalone React app that renders an [obsidian-kanban](https://github.com/mgmeyers/obsidian-kanban) markdown note as an interactive Trello-style board inside a trip2g custom layout.

## What it does

- Parses the obsidian-kanban markdown format (columns + cards, frontmatter, settings block)
- Renders a horizontal drag-and-drop board with **@dnd-kit**
- Drag cards within columns and between columns
- Add, edit (double-click), delete, and checkbox-toggle cards
- Serialises every change back to byte-identical obsidian-kanban markdown
- Saves to trip2g via the `pushNotes` GraphQL mutation

## Install

### 1. Upload the JS bundle

Build or download `dist/kanban.js` and upload it as a vault asset in your trip2g vault settings.

### 2. Add the layout

Drop `_layouts/kanban.html` into your vault's `_layouts/` folder.

Open the file and replace the placeholder:
```html
<!-- BEFORE -->
<script src="__KANBAN_JS_URL__"></script>

<!-- AFTER (using the asset URL assigned by trip2g) -->
<script src="{{ asset('kanban.js') }}"></script>
```

### 3. Set the layout on your kanban note

Add `layout: kanban` to the note's frontmatter. trip2g already adds
`kanban-plugin: basic` if you use the obsidian-kanban plugin — just add:

```yaml
---
kanban-plugin: basic
layout: kanban
---
```

## Dev loop

```bash
cd templates/kanban
npm install
npm run build        # → dist/kanban.js
npm run watch        # incremental rebuild on save
npm test             # smoke-test: format + ops correctness
npx tsc --noEmit     # type-check
```

Point your trip2g preview to the vault and serve `dist/kanban.js` via a
local dev server, then update the `<script src="...">` in the layout to your
local URL (e.g. `http://localhost:4321/kanban.js`).

## Notes

- **Auth**: the app calls `pushNotes` with `credentials: 'include'`. The server
  enforces authorization; a non-admin save returns an `ErrorPayload` which the
  app shows as a toast and the board state reverts on next save attempt.
- **Read-only mode**: set `"editable": false` in the inline bootstrap script
  (or derive it from note meta) to disable all mutations.
- **No Tailwind / no external CSS**: plain CSS variables, dark-mode via
  `prefers-color-scheme`.
- **Bundle**: React, ReactDOM, and @dnd-kit are all inlined into `dist/kanban.js`
  as a self-contained IIFE. No external CDN required.

## TODO

- Markdown rendering: `**bold**`, `` `code` ``, and `[[wikilinks]]` are rendered
  inline. Full block-level markdown (headings, lists inside cards) is not
  supported — card bodies are single-line per the obsidian-kanban format.
- Live-preview reordering during drag (currently cards snap on drop, no
  mid-drag shadow in the destination slot).
- Editable flag derived from note frontmatter/server context rather than
  hardcoded `true`.
