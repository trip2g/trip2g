# Changelog

All notable releases of trip2g. Newest on top. Older tags (`v0.3.1` and below) live in
git history only and were not recorded here.

## v0.4.0 — 2026-05-21

### Features
- **updateNotes mutation**: atomic find/replace across notes via PathMap, with e2e and bilingual user docs.
- **Forms admin**: submit-processing flow (`markFormSubmitProcessed`, `processed` fields, resolvers), `can_submit` enforcement, `success_url`, bilingual docs and e2e.
- **Layout smoke-render** (`noteloader`): each parsed layout is executed at load time against up to 10 notes that use it via frontmatter `layout:`. Jet runtime errors and panics become `NoteWarning` entries on the layout, so CLI / `pushNotes` flows surface them without needing a browser request.
- **Template debugging**: `Meta.Debug()` and a global `debug()` helper with method reflection.
- **renderlayout.py CLI**: standalone script to render a layout against a note + new `check_templates` skill doc.

### Fixes
- `layoutloader`: nil-guard for `YieldNode.Parameters` in `yieldBlocksUsageFinder`.
- `layoutloader`: normalize layout ID in preview to match production format.
- `renderlayout` preview: autoimport, `yield_blocks` wiring, `htmlInjections`.
- `renderpreview`: parse YAML frontmatter from `note.src`; preview/production diff documented.
- `templateviews.GetStrings`: returns empty slice instead of `nil`.

### Docs & chore
- Forms dev reference + roadmap, templates wikilinks, BEM skill, Jet debug section.
- Lint passes in `updatenotes`, `layoutloader`, `noteloader`.
