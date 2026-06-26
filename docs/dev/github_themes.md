# Idea: GitHub repos as site themes

**Status:** idea / parked — **not planned** right now. Recorded so it is not lost.
The owner currently judges it unnecessary; revisit after the component-catalog sub-project.

**TL;DR.** Let a vault adopt a *theme* (layouts + BEM blocks + styles) by pointing at a
GitHub repository, instead of only carrying its own local `_layouts/`. A site could "wear"
a published theme repo, pin it to a ref, and override individual blocks locally. This is one
plausible delivery mechanism for the bigger goal — reusable, shareable component sets that an
agent (or a человек) assembles sites from.

---

## Why it was considered

- **Reuse / sharing.** Themes (a coherent set of Jet layouts + BEM blocks + CSS) could be
  published once and reused across many vaults/sites.
- **Foundation for the BEM dream.** "An agent assembles sites from BEM components" needs a
  *source* of components. A GitHub repo is a natural, versioned, public source.
- **Separation of concerns.** Content (the vault's markdown) stays separate from
  presentation (the theme repo).

## Why it is parked

- Currently judged **not needed** for the near-term goal (make the layout-developer loop
  convenient — see [[preview_tool]] and [[local_design_iteration]]).
- Prerequisites aren't in place: CSS/JS still don't sync, and there is no machine-readable
  component catalog yet. A theme system is more valuable once those exist.
- Real cost/risk: fetching, caching, pinning, and **trusting** arbitrary remote repos
  (custom layouts render with no HTML-escaping — see `renderpreview` notes), plus precedence
  rules against the built-in default template.

## Rough sketch (if revisited — nothing here is decided)

- A vault declares a theme, e.g. frontmatter/config `theme_repo: github.com/org/trip2g-theme-x`
  pinned to a tag/commit.
- On sync/build, trip2g materialises the repo's `_layouts/**/*.html`, `*.html.json`, and
  assets and exposes them as layouts for that vault — the same surface as the locally-synced,
  always-publishable `_layouts/*.html` (`obsidian-sync/src/sync/utils.ts`,
  `internal/layoutloader`).
- Precedence: vault-local `_layouts/` override the theme repo's files of the same path.
- Likely reuses the existing git plumbing rather than a new fetcher — trip2g already keeps a
  DB-canonical lazy git mirror (`internal/gitapi`); a theme could be pulled through similar
  machinery.

## Open questions

- Per-vault themes vs a global/instance theme registry?
- Trust model for third-party repos (what can a theme execute? asset/script scope?).
- How a theme coexists with / overrides the built-in default template (quicktemplate + SCSS)?
- Update cadence — pinned ref only, or follow a branch with a refresh job?
- Is a "theme" a full set of page layouts, or just a block library that local layouts import?

## Related

- [[preview_tool]] — the layout-developer preview loop (the near-term work)
- [[local_design_iteration]] — how layouts/BEM are iterated locally today
- [[json_layouts]], [[layouts]], [[jet_ast_details]] — the layout/Jet/BEM surface a theme would ship
- `internal/gitapi` — existing git mirror that a theme fetch could reuse
