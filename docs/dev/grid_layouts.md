# Grid Page Layouts (design)

**TL;DR.** Replace bespoke page layouts (the `magazine`, the fixed header/sidebars/main/footer chrome, the `wide`/`sidebars` flags) with **one declarative layout DSL**: a tree of typed nodes — `grid` (track sizes + a `grid-template-areas` matrix), `flex`, and `slot` (content). It maps **1:1 to CSS Grid/Flexbox**, so the renderer is a thin emitter. The page itself becomes a configurable grid; `magazine` becomes a preset; custom layouts (kanban) become a slot. Not built — design only.

Related: [[layouts]] (Layouts API), [[json_layouts]] (JSON layouts format), [[default_template]] (current page chrome + `wide`).

## The problem

A natural first instinct for a layout description is **nested arrays**:

```
[[col1, col2], [row2]]
```

This is isomorphic to a **flex tree** — each level alternates row/column. It works for simple nesting, but it cannot express **grid**, because a bare array has nowhere to put the two things grids need:

- **track sizes** — "col1 is `2fr`, col2 is `320px`"
- **2D adjacency** — an item spanning cells, or named regions

Flex nesting is 1D-per-level with no per-node metadata; grid is 2D with explicit tracks. So nested arrays fundamentally top out at "flexbox you can't size."

## The design: typed nodes + `grid-template-areas`

Make every layout node a **tagged object** instead of a bare array. Three node kinds:

```yaml
# flex node — 1D, sized children
flex: { dir: row, gap: 16, items: [ { node: …, grow?: 1, basis?: 240 }, … ] }

# grid node — 2D, the important one
grid:
  cols: [2fr, 1fr, 320px]
  rows: [auto, 1fr]
  gap: 16
  areas:
    - [hero, hero, side]
    - [a,    b,    side]
  slots:
    hero: { note: "featured" }
    a:    { query: "recent", limit: 1 }
    b:    { query: "recent", skip: 1, limit: 1 }
    side: { widget: toc }

# leaf — content reference
slot: { note: "…" }   # or { query: … } | { widget: … } | { layout: kanban }
```

`areas` is a **2D matrix of region names = CSS `grid-template-areas`** — repeating a name *is* the span. `cols`/`rows` are the track sizes. This maps **1:1 to CSS Grid**, so a grid node renders as a single element:

```html
<div style="display:grid;
            grid-template-columns: 2fr 1fr 320px;
            grid-template-areas: 'hero hero side' 'a b side';
            gap: 16px">
  <div style="grid-area:hero">…featured…</div>
  <div style="grid-area:a">…</div>
  <div style="grid-area:b">…</div>
  <div style="grid-area:side">…toc…</div>
</div>
```

The renderer is a thin recursive emitter (Jet/quicktemplate): grid → the div above; flex → `display:flex` + per-child `flex`; slot → resolve the content ref (note / query / widget / nested layout). No new engine.

## What it collapses into one model

1. **The page is already a grid.** `.layout` is `left / main / right`, with `.site-header` above and the footer below. So the `width:` / `sidebars:` keys (see [[default_template]]) become **page-grid track config**: `sidebars: none` drops the side tracks; `width: full` sets the outer track to `100vw` instead of `1440px`. They stop being ad-hoc flags.
2. **`magazine` becomes a preset** — "featured area spanning + a `query` filling the rest" is just a grid layout filled by data. The bespoke `Magazine`/`MagazineCard` quicktemplate funcs go away.
3. **Custom layouts (kanban, etc.) become a `slot`** in the page grid — they get the chrome (header/footer/login) for free instead of re-implementing it (cf. the `content_layout` idea).

## Where it lives

trip2g already has **JSON layouts** ([[json_layouts]]) + the **Layouts API** ([[layouts]]) + a Jet renderer. This is the natural vehicle: add `grid` / `flex` / `slot` node types to the JSON-layout schema and a recursive render func. The page chrome (header / footer / user-space corner) stays as today, but the *body* between them is a rendered layout tree rather than a fixed `note.HTMLString()` slot.

## Scope / open questions

- **Responsive.** Grid track lists need breakpoints (a `cols@md` style override, or per-breakpoint area matrices). Mobile usually collapses to a single column.
- **Migration.** `wide`/`sidebars` and `magazine` need a compatibility shim (translate the old flags to the new page-grid preset) so existing vaults don't break.
- **Content refs in slots.** Define the slot grammar: `note` (by path), `query` (a NoteQuery → N cards), `widget` (toc/backlinks/…), `layout` (a nested layout / custom layout like kanban), `html` (raw).
- **Authoring.** Frontmatter YAML for simple cases; the JSON-layout files for reusable named layouts. Possibly a visual builder later.
- **Sizing units.** Support `fr`, `px`, `%`, `ch`, `minmax()`, `auto`, `min-content` — i.e. pass CSS track values through, with light validation.

This is load-bearing architecture (schema + renderer + refactoring the page body onto a configurable grid). High leverage — it folds `wide`/`sidebars`/`magazine`/custom-layouts into one model — but it should get a full plan before code.
