# layout.json (`*.html.json`) — audit verdict

**TL;DR: CUT.** Zero production usage (the only two `.html.json` files in existence are demo test fixtures), the visual editor that was the format's whole reason to exist was never started on the frontend, and the "assemble pages from blocks" value already ships today via plain Jet `.html` layouts (`{{ block }}` / `{{ yield }}`). The cut is ~1 day of deletion work and fully reversible from git history. If the grid-layouts DSL ever gets built, it deserves a fresh schema, not this one.

## 1. Inventory — what exists

Backend (complete, tested, working):

| Piece | Where | State |
|---|---|---|
| JSON → Jet converter | `internal/layoutloader/json_layout.go` (425 loc) + `json_layout_test.go` (617 loc) | done, well tested |
| `.html.json` load path | `internal/noteloader/loader.go:202-225` (ext detection), `internal/layoutloader/loader.go:91,215,426` | done |
| Sync/push allowlist | `internal/case/pushnotes/resolve.go:115-122`; obsidian-sync `isAlwaysPublishable` (see `docs/dev/local_design_iteration.md`) | done |
| Block metadata for the editor palette | `arg_type` parsing in `layoutloader/loader.go`; GraphQL `admin.layoutBlocks` (`internal/graph/schema.graphqls:1253-1281`, resolver `schema.resolvers.go:2061`) | done, **no consumer** |
| JSON live-preview endpoint | `POST /_system/layouts/render` — `internal/case/admin/renderlayoutpreview/` | done, **no callers**; already marked "Remove (candidate)" in `docs/dev/todo_remove_old_layout.md` |
| doclint support | `internal/doclint/fsenv.go:45-68` | done |
| Original content round-trip for sync | `internal/model/layout.go:17,45`, `internal/graph/note_path_content.go:32` | done |

Frontend (the actual product): **nothing**. `LayoutBlock` types appear only in generated GraphQL types (`assets/ui/graphql/queries.ts:1414,2776`); no component in `assets/ui/` queries `layoutBlocks` or renders any editor. The drag-and-drop tree, property panel, palette, partial preview, and save mutation from `docs/dev/json_layouts.md` are all unchecked TODOs.

## 2. Usage reality

- **Two** `.html.json` files exist across all vaults, both demo fixtures: `docs/demo/_layouts/json-test.html.json` and `json-include-missing.html.json` (worktree/tmp copies of the same two).
- Every real layout is a plain Jet `.html`: `docs/_layouts/mesh/*` (the site chrome/landing), `docs/_layouts/kanban.html`, `docs/demo/_layouts/*` (showcase, yield-blocks demo, components), the kanban_template repo. **No page anyone looks at depends on `.html.json` today.**
- The block-assembly integration the owner remembers as "neat" is real — but it lives in the Jet `.html` layer (`{{ block card(...) }}` in `docs/demo/_layouts/components/`, `custom/blocks.html`, `yield-blocks-demo.html`). It survives a JSON cut untouched.
- Bug class: the live-pull layout-wipe data-loss hit `_layouts/*.html.json` files. Cutting the format removes that instance, though the sync empty-overwrite asymmetry itself is on the obsidian-sync side and should be fixed regardless.

## 3. Effort to finish the original vision

Remaining, per `json_layouts.md` status lists: `/_system/layouts/render` partial render (`node_path`), tree UI component, drag & drop, block palette from GraphQL, typed property panel, live preview wiring, save mutation, `else` branches for `if`/`range`. All frontend work in $mol, which has no existing drag-and-drop tree precedent. Realistic estimate: **10–20 person-days for a usable MVP editor**, plus ongoing maintenance — for a feature with zero current demand and a product focus that has moved elsewhere (fleetbox).

And it would be built on the wrong schema: `docs/dev/grid_layouts.md` (the newer design) plans typed `grid`/`flex`/`slot` nodes that subsume magazine/wide/sidebars. An editor built now on the block-call-only JSON schema would need rework the moment grid layouts land.

## 4. Verdict: CUT

**Why not KEEP/FINISH:** the format's only value over `.html` Jet is machine-editability for a visual editor. No editor, no value — a human authoring `.html.json` by hand is strictly worse off than writing the Jet directly. Finishing costs 10–20 days against zero demand.

**Why not FREEZE:** freezing costs nothing today, but keeps a third template format threaded through eight subsystems (noteloader, layoutloader, pushnotes allowlist, GraphQL, doclint, obsidian-sync, writer protocol, docs) — every future change to any of them pays a small `.html.json` tax, and the docs keep promising an editor that will never come. For a solo-maintained codebase, surface area is the real cost.

**Steelman for FREEZE (the one real tension):** `grid_layouts.md` explicitly names JSON layouts as "the natural vehicle" for the grid DSL, and the converter is finished and tested. If the grid DSL is genuinely planned within ~6 months, FREEZE (delete only `renderlayoutpreview` + `layoutBlocks`, keep the converter) is defensible. But the grid design itself says "full plan before code," the node types (`grid`/`flex`/`slot` with track sizes and area matrices) share almost nothing with today's block-call schema beyond "it's JSON," and git history preserves the converter anyway. Betting kept code on an unplanned design is how zombie features accumulate.

## 5. What cutting removes, and migration

Delete list (roughly one day, mostly mechanical):

1. `internal/layoutloader/json_layout.go` + test; `.html.json` branches in `layoutloader/loader.go` (`:91,215,426`) and `noteloader/loader.go` (`:202-203,225`).
2. `internal/case/admin/renderlayoutpreview/` + router regen (already slated by `todo_remove_old_layout.md`).
3. GraphQL: `admin.layoutBlocks`, `LayoutBlock*` types (`schema.graphqls:1253-1281`), resolver, `arg_type` parsing in `layoutloader` — unless the palette API is wanted for something else, nothing consumes it.
4. `.html.json` in `pushnotes/resolve.go:115-122` allowlist and error message; obsidian-sync `isAlwaysPublishable` + extension lists; `doclint/fsenv.go`; `note_path_content.go` original-content branch (keep for `.html` if shared).
5. The two demo fixtures; docs: retire `json_layouts.md` (move to a "removed features" note), update `writer_protocol.md`, `local_design_iteration.md`, `obsidian_bases.md` mentions.

Migration for users: **none needed** — no known vault has a real `.html.json`. Safety valve: on push, reject `.html.json` with a clear "format removed, use Jet .html layouts" error rather than silently ignoring.

## 6. Relation to grid_layouts

Grid layouts is a different, better-scoped idea (page chrome as a configurable CSS grid) and does **not** depend on this format — it needs a schema designed around tracks/areas/slots. If/when it gets its full plan, start that schema clean; the JSON→Jet emitter pattern (the useful lesson from `json_layout.go`) is recoverable from git in minutes. Cutting now does not close that door.
