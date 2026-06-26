# Rendering Obsidian Kanban boards (kanban-plugin)

**TL;DR.** The popular Obsidian **Kanban** plugin (`mgmeyers/obsidian-kanban`)
stores a board **as plain markdown** — `kanban-plugin: basic` frontmatter,
`## Column` headings, `- [ ] card` checklist items, and a trailing
`%% kanban:settings {…} %%` block. Because the board *is* markdown, trip2g
**already round-trips it** (Obsidian → `pushNotes` → DB → `noteChanges` SSE →
Obsidian). So this is **not** a sync feature — it is a **render + edit feature**:
detect the board note, render it as an interactive board inside the **default
template**, and serialize admin edits back to the **exact same markdown** so the
real Kanban plugin still parses it. This is a **render-layer feature, not a
greenfield build.**

**Locked decisions.**
1. **Render home — default template content-type.** A board note renders inside
   the default template (not a bare Jet layout), so it inherits the existing
   user-space corner (`$trip2g_user_space`: sign-in, admin link, editor) and the
   conditional-widget loader. Login "in the corner" comes for free.
2. **Board UI — a `$trip2g_kanban` $mol component** using `$mol_drag`. The default
   template already loads the $mol bundle (it hydrates `$trip2g_user_space`), so a
   new `mol_view_root="$trip2g_kanban"` hydrates on the same page.
3. **Parse/serialize — TS only.** Rendering is client-side, so the board
   parser/serializer lives entirely in TypeScript (single source of truth, no
   Go ↔ JS drift). Go only detects the frontmatter key and emits a data island.
4. **Edit scope — cards CRUD + drag, admin-only.** Add / edit-text / delete cards
   and drag cards between columns. No column add/rename/reorder this cut. The
   settings block and per-card metadata are **preserved verbatim, never edited**.

---

## 1. What a Kanban board is, on disk

A board is one markdown note. Real plugin output (whitespace is significant —
blank lines inside the frontmatter fence, **3 blank lines** between columns, a
`**Complete**` marker in the Done column, and a **fenced** JSON settings block):

````markdown
---

kanban-plugin: basic

---

## Todo

- [ ] Write the parser
- [ ] Wire the data island


## Done

**Complete**
- [x] Research the format



%% kanban:settings
```
{"kanban-plugin":"basic","lane-width":270}
```
%%
````

Format rules the TS parser/serializer must honor:

- **Board marker** — `kanban-plugin` key in frontmatter (`basic`). Preserve the
  whole frontmatter verbatim.
- **Columns (lists)** — each `## Heading` starts a column; the heading text is the
  column title.
- **Cards** — `- [ ] text` (open) / `- [x] text` (done) checklist items under a
  column. Card text is markdown (may contain wikilinks, inline formatting, tags).
- **Settings block** — the trailing `%% kanban:settings … %%` HTML-comment block
  holds the plugin's per-board config (lane width, archive, etc.). **Opaque to us:
  carried through byte-for-byte.**
- **Per-card metadata** — the plugin may append metadata to a card line (dates,
  `@@{…}`, etc.). Also opaque: preserve the trailing remainder of each card line.

**Fidelity rule (locked).** Our serializer targets **the plugin's own output**, not
a hand-written ideal. We capture a *real exported board* as the golden fixture and
lock `serialize(parse(md)) === md` byte-identity against it. If our output and the
plugin's ever diverge, the plugin wins on the next Obsidian edit and we re-derive —
so the only correctness bar is "the plugin still recognizes what we wrote."

---

## 2. Architecture

```
Obsidian Kanban note (board.md, raw markdown)
        │  sync (existing)
        ▼
note_versions.content  ──►  default template render path
        │                         │  note.M().Has("kanban-plugin")? → board
        │                         ▼
        │                   emit  <div mol_view_root="$trip2g_kanban">
        │                         + window.__trip2g_kanban = {path, versionId,
        │                                                     editable, markdown}
        │                         ▼
        │                   $trip2g_kanban ($mol)  ── format.ts parse → Board
        │                         │  render columns/cards, $mol_drag, CRUD (admin)
        │                         ▼
        │                   format.ts serialize → markdown
        │                         │  pushNotes({path, content}) via graphql_request
        ▼                         ▼
note_versions (new version)  ◄────┘
        │  noteChanges SSE (existing)
        ▼
sync watcher rewrites board.md  ──►  real Kanban plugin re-parses it
```

### 2.1 Detection + data island (Go, minimal)

In the default template (`internal/defaulttemplate/views.html`, then
`go generate ./internal/defaulttemplate/...`):

- A board is a note whose frontmatter has `kanban-plugin` (`ctx.Note.M().Has("kanban-plugin")`).
  Add a small `ctx`-level helper (e.g. `IsKanban()`) to keep `views.html` readable.
- For a board note, **replace** the normal markdown content render with:
  - a `<div mol_view_root="$trip2g_kanban">` mount, and
  - a **JSON data island** the component reads —
    `<script type="application/json" id="kanban-data">{%z= kanbanJSON %}</script>` —
    carrying `{path, versionId, editable, markdown}`. This mirrors the **form-spec
    precedent** (`views.html:324`, `<script id="form-spec" type="application/json">`)
    and uses `%z` JSON-safe escaping so card markdown containing `</script>` is safe.
    (The paywall's `window.__trip2g_paywall` global,
    `assets/ui/user/paywall/page/page.ts`, is the same idea; the island is preferred
    here because it embeds arbitrary note markdown.)
  - `editable` = `ctx.UserToken.IsAdmin()`.
  - `markdown` = the raw note content (`ctx.Note.ContentString()`).
- Keep the server-rendered list HTML **inside the mount** as a no-JS fallback; the
  component clears it on hydration.

No Go parser, no `note.Kanban()`, no new model fields. Detection is a frontmatter
key check.

### 2.2 TS format module — `assets/ui/kanban/format.ts`

Pure functions, no DOM, unit-tested:

```ts
type Card  = { text: string; checked: boolean; trailer: string }  // trailer = preserved metadata
type List  = { title: string; cards: Card[] }
type Board = { frontmatter: string; lists: List[]; settings: string } // settings/frontmatter raw

function parse(md: string): Board
function serialize(b: Board): string
```

- `serialize(parse(md)) === md` byte-identity on the golden fixture(s).
- Round-trip after a mutation (move/add/edit/delete a card) still produces
  plugin-parseable markdown.

### 2.3 `$trip2g_kanban` $mol component — `assets/ui/kanban/`

`kanban.view.tree` (structure) + `kanban.view.ts` (behavior), per the project's
$mol conventions (symlink `assets/ui/` ← `../mam/trip2g/`).

- **Load.** Read `#kanban-data` (`JSON.parse(getElementById('kanban-data').textContent)`);
  `format.parse(markdown)` → board.
- **Render.** Columns as drop zones, cards as draggable rows. `$mol_drag` provides
  the drag primitive (see the mol drag demo). Card text rendered as markdown/plain.
- **Edit (admin only, `editable === true`).** Add card, edit card text, delete card,
  drag a card to another column. Each mutation updates the in-memory board.
- **Save.** Debounced: `format.serialize(board)` → `pushNotes({ input: { updates:
  [{ path, content }] } })` via `$trip2g_graphql_request` (reuse the editor's
  `EditorPushNotes` mutation shape, `assets/ui/editor/pane/pane.view.ts:13`). On
  success, bump a **baseline version id** so the `noteChanges` SSE self-echo is
  ignored (same mechanism as `pane.view.ts:250`).
- **Read-only.** When `editable === false`, render a static board (no drag/CRUD).

### 2.4 Round-trip (free, existing infra)

- **Site → Obsidian.** Save → `pushNotes` → new `note_versions` row → `noteChanges`
  SSE → the memcli instance's sync watcher rewrites `board.md` → Obsidian's Kanban
  plugin re-parses it (format preserved).
- **Obsidian → site.** Edit the board in Obsidian → plugin writes markdown → sync
  watcher `pushNotes` → DB → next site render shows it.

---

## 3. Test environment (scratch, not committed)

A gitignored scratch vault drives local iteration and proves the round-trip in
*real* Obsidian.

- `kanban-vault/`
  - `board.md` — a sample board in `kanban-plugin: basic` format.
  - `.obsidian/` — with the real **Kanban** community plugin installed, so we can
    open the same vault in Obsidian and confirm our serialized output parses.
  - seed files (`index.md`, etc.) as memcli expects.
- Build a local dev image (the repo's normal image build, tagged e.g. `trip2g:dev`).
- Run an **isolated** instance:

  ```
  memcli up --folder ./kanban-vault --name kanban --port 24181 --image trip2g:dev
  ```

  This boots a separate container (`trip2g-memory-kanban`) with its own DB and a
  `trip2g-sync --watch` daemon that mirrors DB ↔ vault within ~500 ms.
- Iterate: open `localhost:24181`, sign in via the user-space corner, edit the
  board → watch `kanban-vault/board.md` change; edit `board.md` (or in Obsidian) →
  watch the site update.

---

## 4. Verification

| Claim | Evidence |
|-------|----------|
| Format is faithful | `format.ts` round-trip unit tests pass against a **real exported** board fixture |
| Board renders | `localhost:24181` board page screenshot (chrome-devtools) |
| Admin gating | logged-out page shows read-only board; signed-in admin sees drag/CRUD |
| Site→Obsidian | edit a card on site → `kanban-vault/board.md` is rewritten and the Kanban plugin still parses it (frontmatter + settings block intact) |
| Obsidian→site | add `- [ ] card` in the vault → site shows it after sync |

---

## 5. Delivery & non-goals

- **Branch/worktree.** Implement on `feat/kanban-board` in an isolated worktree;
  code delegated to the executor (Sonnet); orchestrator verifies; deliver a PR.
- **Build commands touched.** `go generate ./internal/defaulttemplate/...` (after
  `views.html`); the $mol/Vite app build for the new component; possibly
  `npm run graphqlgen` is **not** needed (reusing the existing `pushNotes`).
- **Non-goals (YAGNI).** Column add/rename/reorder; editing card due-dates/tags;
  server-side board HTML for SEO; multi-board indexes. The settings block and card
  metadata are preserved, never edited.

---

## 6. Open risks

- **Plugin output drift.** Mitigated by golden-fixturing a real export (§1).
- **Auth refresh after sign-in.** `editable` is server-rendered, so a freshly
  signed-in admin may need one reload before edit affordances appear. Acceptable
  for v1; a later pass can read live auth state from the `$trip2g_user_space` store.
- **Bundle size.** The component joins the always-loaded $mol bundle. Fine for v1;
  lazy-load via `$mol_import` later if it matters.
