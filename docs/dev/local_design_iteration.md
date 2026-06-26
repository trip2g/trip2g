---
title: "Local Design Iteration"
slug: local_design_iteration
free: true
---

# Local Design Iteration on Jet Layouts

**TL;DR.** You can edit a Jet layout on your machine and watch the result update almost instantly, two ways. **Loop A** — POST the template to `/_system/renderlayout` and keep `?live` open; it long-polls and reloads on every render, nothing touches your vault. **Loop B** — run `memcli up` (or `trip2g-sync --watch`) so the bidirectional watcher pushes `_layouts/*.html` to a local server, and the **real published page** re-renders on save. One caveat: the watcher syncs `.html`/`.html.json` templates but **not** standalone `.css`/`.js` — put layout styles in an inline `<style>` (the site-wide default theme CSS is a separate, binary-embedded pipeline).

This guide ties together three pieces that already exist but were never connected: the [renderlayout preview endpoint][renderlayout], the memcli `--watch` sidecar, and the Jet/[JSON-layout][json_layouts] design surface.

---

## Which loop do I want?

| | Loop A — transient preview | Loop B — real site |
|---|---|---|
| **You see** | the layout rendered against one note, in an in-memory buffer | the actual page on your local site, served like production |
| **Writes to vault?** | no — nothing is uploaded | yes — the template is pushed to the server |
| **Best for** | fast template/Jet-logic iteration, catching compile errors, sharing a quick preview link | final look-and-feel, navigation, cross-note queries (`nvs`), "does it work in situ" |
| **Trigger** | `?live` long-poll reloads on each POST | edit a file → watcher pushes → refresh (or `?#!live_follow=1`) |
| **Auth** | admin session or API key | the watcher's API key |
| **Entry point** | `/_system/renderlayout` ([doc][renderlayout]) | `memcli up` / `trip2g-sync --watch` |

Use **A** while you're shaping the template, **B** to confirm it on the real page.

---

## What syncs (and what doesn't)

The sync watcher's local scan only picks up these extensions
(`obsidian-sync/src/sync/cli/env.ts`, the `getLocalFiles` filter):

```
.md  .html  .html.json  .canvas  .base  .excalidraw
```

So:

- **Jet layouts sync.** `_layouts/*.html` and `*.html.json` are *always publishable* — they go to the server even without a `free:`/publish field in frontmatter (`obsidian-sync/src/sync/utils.ts`, `isAlwaysPublishable`).
- **CSS/JS do not sync.** A standalone `style.css` next to your layout is ignored by the local scan. Put layout-scoped styles in an inline `<style>` block inside the `.html`, or reference an already-published asset with `{{ asset("…") }}`.
- **The default theme is a different world.** The site-wide look (`assets/defaulttemplate/src/index.scss` → embedded `defaulttemplate.css`) is compiled into the Go binary. Changing it needs `npm run defaulttemplate-css` **and a rebuild** — it is not part of either live loop. Loops A/B are for *custom Jet layouts*, not the built-in theme.

> Heads-up: the `$mol` admin JSON-layout editor (`/_system/layouts/render`, `assets/ui/admin/layout/editor/`) is a separate, currently-incomplete builder — JSON layouts stay, and it may be finished later. For the file-based live loop in this guide, the endpoint to use is `/_system/renderlayout` (the editor's `/_system/layouts/render` is a different thing).

---

## Loop A — transient preview (`/_system/renderlayout`)

Render any Jet template against a note without uploading anything. Full reference and CLI live in [renderlayout][renderlayout]; the short version:

```bash
# POST the template + a note → get a preview link + Jet warnings
curl -s -X POST http://localhost:8081/_system/renderlayout \
  -H "X-API-Key: $API_KEY" -H 'Content-Type: application/json' \
  -d '{
    "layout": { "path": "/_layouts/article.html",
                "src":  "<h1>{{ note.Title() }}</h1>{{ note.HTMLString() | unsafe }}" },
    "note":   { "src": "# Hello\n\nbody" }
  }'
# → {"previewID":"…","previewURL":"/_system/renderlayout?preview_id=…","warnings":{...}}
```

Keep the live view open in a browser — it injects a long-poll script and reloads the instant a new render lands:

```
http://localhost:8081/_system/renderlayout?live
```

Use `--watch` for a built-in loop that re-POSTs on every file save:

```bash
node scripts/trip2g-preview.mjs --watch \
  --layout-file _layouts/article.html --note-path posts/hello.md
# open the printed ?live URL; it reloads on every save
```

Errors come back in `warnings.layout` as text (e.g. `runtime: … can't use NoSuchMethod as field name …`), so an agent can fix the template without opening a browser. Implementation: `internal/case/admin/renderpreview/endpoint.go` (POST, `?live`, `?longpolling`, `?preview_id`).

---

## Loop B — the real site via memcli `--watch`

Here you edit a layout file and the **actual page** updates, because the bidirectional watcher pushes the template to a running server. `memcli up` is the one-command harness: it boots a local trip2g (Docker, local storage), mints an admin key, and starts `trip2g-sync --watch`. See [memcli][memcli] for the full command surface.

### Worked example

Point memcli at a vault and boot it:

```bash
node cli/memcli/dist/memcli.js up --folder ./site-vault
# memory live — web: http://localhost:24081  read/write .md in ./site-vault
```

Add a Jet layout and a note that uses it (paths relative to the vault):

```html
<!-- site-vault/_layouts/demo/page.html -->
<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>{{ note.Title() }}</title>
<style>
  body { font-family: system-ui, sans-serif; max-width: 40rem; margin: 3rem auto; }
  .hero { background: #0b5; color: #fff; padding: 2rem; border-radius: 12px; }
</style></head>
<body>
  <div class="hero"><h1>{{ note.Title() }}</h1></div>
  <main>{{ note.HTMLString() | unsafe }}</main>
</body></html>
```

```markdown
<!-- site-vault/landing.md -->
---
free: true
route: /landing
layout: demo/page
---
# Sales Landing

Body copy rendered inside the custom Jet layout.
```

`layout: demo/page` resolves to `_layouts/demo/page.html`. The watcher pushes both files within ~500 ms of the save, and the page is live:

```bash
curl -s http://localhost:24081/landing      # rendered through your Jet layout
```

Now **edit `_layouts/demo/page.html`** — change `#0b5` to `#a04bd6`, save. The watcher logs `📤 pushing 1 local change(s)`, and the same URL re-renders with the new color. No rebuild, no manual upload.

For a hands-free view, open the page in cinema mode — it follows whichever note changes on the server:

```
http://localhost:24081/landing?#!live_follow=1
```

### Without memcli

If you already have a server running (e.g. `make air` on `:8081`, or any deployed instance), skip the container and run the watcher directly against the vault:

```bash
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder ./site-vault \
  --api-url http://localhost:8081/_system/graphql \
  --api-key "$API_KEY"
```

Same loop: save a `_layouts/*.html`, the watcher pushes it, the real page updates.

> Single-instance note: `memcli up` always names its container `trip2g-memory` (hardcoded). To run a second isolated instance alongside it, drive `docker run` + `trip2g-sync --watch` yourself with a different name and port, or just reuse the running one.

---

## Putting it together

A tight design session usually mixes both:

1. **Shape the template with Loop A.** Iterate Jet logic fast against one representative note; read `warnings` until the render is clean.
2. **Confirm with Loop B.** Drop the layout into the vault, point a note at it, refresh the real page (or use `?#!live_follow=1`) to check navigation, `nvs` queries, and surrounding chrome.
3. Keep layout styles inline; if you need site-wide theme changes, that's the separate `npm run defaulttemplate-css` + rebuild path, not this loop.

## Related

- [[renderlayout]] — the preview endpoint reference + `scripts/trip2g-preview.mjs` CLI
- [[json_layouts]] — `.html.json` block format that compiles to Jet
- [[layouts]] — the `templateviews` API (`note`, `nvs`, `Meta`) available in templates
- [[jet_ast_details]] — Jet v6 block/`yield_blocks` internals
- [[memcli]] — the local server + sync harness
- [[json_layouts]] / `assets/ui/admin/layout/editor/` — the JSON-layout format and its (in-progress) admin editor

[renderlayout]: ../en/user/renderlayout.md
[json_layouts]: ./json_layouts.md
[memcli]: ../en/user/memcli.md
