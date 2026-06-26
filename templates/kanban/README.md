# trip2g Kanban Template

A drop-in [trip2g](https://github.com/trip2g/trip2g) layout that renders an
[obsidian-kanban](https://github.com/mgmeyers/obsidian-kanban) note as an
interactive, Trello-style board on your site — editable both on the website
**and** locally in Obsidian (it round-trips to the exact same markdown).

Built as a self-contained React + [@dnd-kit](https://github.com/clauderic/dnd-kit)
app, served by a custom Jet layout. Not part of trip2g core — install it per-vault.

## Install (curl & mkdir)

In the root of your trip2g vault:

```bash
mkdir -p _layouts
curl -L -o _layouts/kanban.html \
  https://github.com/trip2g/kanban_template/releases/latest/download/kanban.html
```

Sync the vault (the obsidian-sync plugin uploads `_layouts/*.html`), then add
`layout: kanban` to any obsidian-kanban note's frontmatter:

```yaml
---

kanban-plugin: basic
layout: kanban

---
```

That's it. The JS bundle loads automatically from the release — there is nothing
else to upload. Open the note's page and the board appears.

> The bundle (`kanban.js`) is **not** copied into your vault: the sync uploads
> `.md`/`.html`, not `.js`, so the layout loads the bundle from the GitHub release
> via `/releases/latest/download/`. It always serves the newest release, so you
> never re-curl to update.

## What it does

- Parses the obsidian-kanban markdown format (columns, cards, frontmatter, the
  `%% kanban:settings %%` block) and renders a horizontal drag-and-drop board.
- Drag cards within and between columns; add, double-click-to-edit, delete, and
  checkbox-toggle cards.
- Renders inline markdown in cards: `**bold**`, `` `code` ``, and `[[wikilinks]]`.
  Clicking a wikilink opens the linked note in a slide-in preview drawer (iframe);
  ⌘/Ctrl/middle-click opens it in a new tab.
- **Saves in place** via the `updateNotes` GraphQL mutation: a single-line edit
  (toggle/edit/delete/add) becomes a surgical `find`/`replace` patch; a move
  rewrites the affected lines via `upsert`. Optimistic concurrency via
  `expectedHash`. **Content the parser doesn't model — blockquotes, multi-line
  notes, the archive, the settings block — is preserved byte-for-byte.**
- On a hash mismatch (someone else edited the note) the board reloads to the
  latest; on an auth failure the change reverts and a toast explains why.

## Customise / rebuild

The release ships the prebuilt `kanban.js`. To change the board (styling,
behaviour) fork this repo and rebuild:

```bash
npm install
npm run build        # → dist/kanban.js  (self-contained IIFE: React + @dnd-kit inlined)
npm run watch        # incremental rebuild on save
npm test             # smoke tests: byte-perfect format round-trip + ops reducers
npx tsc --noEmit     # type-check
```

For a local dev loop, serve `dist/kanban.js` from any static server and change the
`<script src="…">` URL in `_layouts/kanban.html` to point at it (cross-origin
classic scripts are fine; the GraphQL `fetch` stays same-origin to your instance).

## How it works

- **`_layouts/kanban.html`** — a Jet layout. It puts the note path in a JSON
  island and the raw markdown in a hidden `<textarea>` (Jet has no JSON/base64
  filter; `<textarea>` is escapable raw text so the browser restores the exact
  bytes), assembles `window.__trip2g_kanban = {path, content, editable}`, and loads
  the bundle.
- **`src/`** — `format.ts` (byte-perfect parse/serialize, the single source of
  truth), `ops.ts` (pure reducers + surgical line patching), `Board.tsx`
  (@dnd-kit board, preview drawer, debounced save), `api.ts` (`updateNotes` +
  sha256 hashing), `markdown.tsx`, `styles.css` (shadcn-style tokens, light/dark).

## Limitations / follow-ups

- The board UI always renders editable; non-admin saves are rejected server-side
  (revert + toast) rather than hidden. Gating the edit affordances on viewer role
  is a follow-up.
- Card bodies are single-line per the obsidian-kanban format; block-level markdown
  inside a card (lists, headings) is not rendered.
- Air-gapped instances that cannot reach github.com need to self-host the bundle
  (serve `dist/kanban.js` from your own host and edit the `<script src>` URL).
