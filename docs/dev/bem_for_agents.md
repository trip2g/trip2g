# BEM Blocks for AI Agents

## Vision

Split the layout authoring workflow into two distinct phases:

1. **Design phase** — AI agent creates or selects BEM blocks (components with HTML + CSS + JS)
2. **Assembly phase** — AI agent composes a page from existing blocks without reading their source

The block registry is the bridge. An agent composing a page only needs the block catalog:
name, parameters, description. It never needs to read the block implementation.

## Block Registry

`layoutloader` already builds a block registry at load time via `blockFinder`:

```go
Layouts.Blocks.ByName     map[string]model.LayoutBlock
Layouts.Blocks.ByFullName map[string]model.LayoutBlock  // sourceID#blockName
```

Each `model.LayoutBlock` carries:
- `Name` — BEM block name (`hero`, `button`, `card`)
- `Params` — typed parameters with defaults and descriptions (via `arg_type` directive)
- `SourceID` — which layout file defines this block
- `HasContent` — whether the block accepts `{{yield content}}`

This is enough for an agent to compose a page. Example agent prompt:

```
Available blocks: hero(title, subtitle), button(label, variant="primary"), card(title, body)
Build a landing page with a hero, three cards, and a CTA button.
```

The agent outputs a JSON layout or a Jet template using only block names and params.

## JSON Layouts

`docs/dev/json_layouts.md` describes the existing JSON layout format that lets you compose
pages from blocks without writing HTML:

```json
{
  "blocks": [
    { "name": "hero", "params": { "title": "Welcome", "subtitle": "..." } },
    { "name": "card", "params": { "title": "Feature 1", "body": "..." } }
  ]
}
```

This is already the agent-friendly interface. The agent does not need to know HTML.

## Inline CSS/JS (defer_block)

Each BEM block optionally declares its CSS and JS. The layoutloader extracts this at parse
time. `flush_blocks "style"` and `flush_blocks "js"` emit the collected assets once per page.

This means the agent can compose pages from blocks and get correct CSS/JS automatically —
it never has to think about asset management.

## Name Collision Warnings

Since blocks are global across all imported layout files, the registry warns at load time
when two files define a block with the same name:

```
NoteWarning: block "hero" defined in both layout/hero.html and layout/alt-hero.html
```

The last definition wins (consistent with `ByName` last-wins semantics). The warning lets
the agent or template author know to rename one of them.

## Workflow

### Design phase (agent creates blocks)

```
Agent input:  "Create a hero block with title, subtitle, CTA button, and background image"
Agent output: layout/hero.html  (HTML structure + inline CSS + inline JS)
```

The agent writes a self-contained BEM block file. The block is immediately available in the
registry after the layout is reloaded.

### Assembly phase (agent composes pages)

```
Agent input:  block catalog + "Build a SaaS landing page"
Agent output: JSON layout or Jet template referencing existing blocks
```

The agent picks blocks from the catalog, supplies parameters, and produces a composition.
No HTML knowledge required. No CSS/JS knowledge required.

## API Exposure

To make the block catalog available to agents, the GraphQL API should expose:

```graphql
type LayoutBlock {
  name: String!
  sourceID: String!
  params: [LayoutBlockParam!]!
  hasContent: Boolean!
  staticCSS: String    # extracted at load time if CSS is static
  staticJS: String
}

type Query {
  layoutBlocks(siteID: ID!): [LayoutBlock!]!
}
```

An agent queries `layoutBlocks` once per site, caches the result, and uses it for all
subsequent assembly tasks.

## Relation to JSON Layouts

JSON layouts (`docs/dev/json_layouts.md`) are already designed for programmatic composition.
HTML-based Jet templates with `{{yield blockName(params...)}}` calls are more flexible but
require knowledge of Jet syntax.

For agents, JSON layouts are the preferred interface — declarative, schema-validated,
no template syntax. HTML templates are better for human authors who want fine-grained
layout control.

Both formats use the same block registry. An agent can start with JSON layouts and graduate
to HTML templates as its capabilities grow.

## Summary

| Phase | Who | Interface | Output |
|-------|-----|-----------|--------|
| Design | Agent / human | File editor | `layout/block-name.html` |
| Assembly | Agent | Block catalog API | JSON layout or Jet template |
| Render | trip2g | Jet template engine | HTML page with deduped CSS/JS |
