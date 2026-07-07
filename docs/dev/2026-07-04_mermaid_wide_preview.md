# Mermaid + wide: agent-published full-page diagram previews

**TL;DR: yes, everything needed is already shipped and live-verified.** An agent
calls `updateNotes` (upsert) with a note containing `wide: true` + `free: true`
frontmatter and a ` ```mermaid ` fence, and hands back the note URL. A 42-node
flowchart rendered full-page on a local instance: fills the whole main column,
no sidebars, no overflow. One real limitation: mermaid scales the SVG *down* to
fit — no pan/zoom, so a huge diagram gets small text instead of a scrollbar.

## Author recipe

````markdown
---
free: true
wide: true
title: My architecture diagram
---

# Diagram

```mermaid
graph TD
  A --> B
```
````

- ` ```mermaid ` fence — same syntax as Obsidian. Server-side, goldmark leaves
  it as `<pre><code class="language-mermaid">`; client glue converts and renders
  (`docs/dev/mermaid.md`).
- `wide: true` — drops the sidebars and the 65ch reading cap so the diagram
  gets the full main column. `Ctx.IsWide()` in
  `internal/defaulttemplate/template.go:118`, class `layout__main--wide`
  (`assets/defaulttemplate/src/index.scss:406`), sidebar suppression in
  `views.html:114-120`. The comment even names "big mermaid diagrams" as a use
  case — this IS the intended mechanism, no other "full-width" knob exists.
- `free: true` — makes the page public (no paywall/auth), so the preview URL
  works for anyone.

Widget loading is per-note and backend-decided: `mermaid.js` is emitted only
when the note has a mermaid fence (`note.HasCodeLanguage("mermaid")` in
`internal/case/rendernotepage/endpoint.go`); the heavy `mermaid.min.js` (~3 MB)
is then lazy-loaded client-side. Verified live: the rendered page had exactly
one `/assets/mermaid.js?h=…` script and the `layout__main--wide` class.

## Live render finding

Published a 42-node / ~45-edge `graph TD` architecture flowchart with
`wide: true` to a local instance and screenshotted at 1920×1080:

- SVG filled the wide main column (1398px of a 1728px inner viewport),
  `wide: true` confirmed active, **no horizontal overflow**.
- Mermaid's default `useMaxWidth` behavior scaled the diagram to fit: intrinsic
  width was 2793px, rendered at 1398px (~50%). At this size all 42 node labels
  were still legible on desktop.
- Without `wide: true` the same diagram would be capped at 65ch (~700px) —
  half the space again — so the flag matters exactly as assumed.

## Agent → preview flow (verified end-to-end)

1. Auth: an API key (admin → `createApiKey`), sent as `X-Api-Key` header to
   `POST /graphql`.
2. Mutation: `updateNotes` with an `upsert` change — `path` + full markdown
   `content` (frontmatter included). Returns `UpdateNotesSuccessPayload` with
   the saved path and `versionId`.
3. URL: the permalink is the path normalized to underscores
   (`mermaid-wide-test.md` → `/mermaid_wide_test`,
   `internal/model/note.go` PreparePermalink). To hand back a deterministic
   URL, the agent should set `slug: /my-preview` (used verbatim, becomes the
   permalink) or just apply the same normalization to its chosen path.
4. Cleanup: `updateNotes` with a `hide` change removes the page (verified:
   404 after hide).

The fleet runtime's scoped write lane goes through the same `updateNotes`
mutation, so a fleet agent can do this without any new plumbing.

## Gaps / limitations

- **No pan/zoom controls.** Mermaid fits-to-width; a 100+ node diagram will
  render with unreadably small text rather than scroll. If that becomes a real
  need, the glue (`assets/mermaid/src/index.ts`) could add a pan/zoom wrapper
  (e.g. svg-pan-zoom) or set `useMaxWidth: false` + `overflow-x: auto` — not
  built today.
- `securityLevel: 'strict'` in the glue — no click handlers / HTML labels in
  diagrams.
- Permalink guessing: default path→URL normalization converts every
  non-alphanumeric to `_`; agents should use `slug:` for exact URLs.
- New notes appear in the sitemap and (depending on layout) nav — a "preview"
  is a real public page, not an unlisted one. No draft/unlisted mode exists;
  `hide` is the cleanup.
