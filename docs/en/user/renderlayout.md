# Layout Preview Endpoint

`/_system/renderlayout` renders any Jet layout template without uploading files to the server. Nothing is saved — rendered HTML lives only in a small in-memory ring buffer. Use it for debugging template changes and sharing quick previews before any content reaches the production vault.

## Who is it for?

- **Developers** editing layout files locally before deploying
- **AI agents** testing Jet template edits, reading compile/runtime errors from the response, and sharing intermediate preview links with the human they are working with
- **Anyone** who wants to see how a note will look with a different layout

## Quick start

```bash
curl -X POST https://yoursite.com/_system/renderlayout \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "layout": {
      "path": "_layouts/article.html",
      "src": "<html><body><h1>{{ note.Title() }}</h1>{{ note.HTMLString() }}</body></html>"
    },
    "note": { "src": "# Hello\n\nThis is a test." }
  }'
```

Response:
```json
{
  "previewID":  "a3f9b2c1",
  "previewURL": "/_system/renderlayout?preview_id=a3f9b2c1",
  "warnings":   {"layout": [], "note": [], "files": []}
}
```

Open `previewURL` in a browser to see the rendered result.

---

## POST `/_system/renderlayout`

**Auth:** `X-API-Key` header or admin session cookie.

### Request body

```json
{
  "layout": {
    "path": "required — server path of the entry-point template",
    "src":  "optional — inline Jet HTML; replaces the server file at path"
  },
  "note": {
    "path": "optional — server note path (e.g. posts/hello.md)",
    "src":  "optional — inline markdown (alternative to path)"
  },
  "overrideFiles": [
    {"path": "some/_layouts/component.html", "src": "override content for included files"}
  ]
}
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `layout.path` | **yes** | Server path of the layout. Identifies which template to start rendering from. |
| `layout.src` | no | Inline Jet HTML. When provided, replaces the server file at `layout.path` for this render only. |
| `note.path` | no | Path of an existing server note. Used as the `note` template variable. |
| `note.src` | no | Inline markdown. Rendered and used as the note. |
| `overrideFiles` | no | Array of `{path, src}` objects. Overrides any server layout file with the given path when Jet resolves `include`/`extends`. |

`note.path` and `note.src` are mutually exclusive. If neither is provided, the template receives a stub note with empty content.

### Response

```json
{
  "previewID":  "a3f9b2c1",
  "previewURL": "/_system/renderlayout?preview_id=a3f9b2c1",
  "warnings":   {
    "layout": ["compile: line 5: undefined variable 'note.Tags'"],
    "note":   [],
    "files":  []
  }
}
```

- `warnings` is an object with three keys: `layout`, `note`, and `files`. Each contains an array of Jet compilation and runtime errors for that scope. All arrays empty = clean render.
- A render with warnings still produces an entry in the buffer (partial HTML is stored).
- `previewID` is valid until the ring buffer wraps (default: 10 entries).

---

## GET `/_system/renderlayout`

| URL | Auth | Returns |
|-----|------|---------|
| `?preview_id=xxx` | none | Stored HTML for that ID |
| (no params) | API key / admin | Latest rendered HTML |
| `?live` | API key / admin | Latest HTML + auto-reload script |
| `?longpolling&since=N` | none | `{action, version}` after ≤ 30 s |

### Auto-reload (`?live`)

Open `?live` in a browser. The page injects a polling script that calls `?longpolling` and reloads automatically when a new render arrives. Combine with a local file watcher to get live preview as you edit.

---

## Body combinations and use cases

### 1 — Test a server layout with an existing note

```json
{
  "layout": {"path": "_layouts/article.html"},
  "note":   {"path": "posts/hello.md"}
}
```

Useful to verify that an already-deployed layout renders a specific note correctly.

### 2 — Test local layout edits (not yet uploaded)

```json
{
  "layout": {
    "path": "_layouts/article.html",
    "src":  "...modified Jet HTML..."
  },
  "note": {"path": "posts/hello.md"}
}
```

The server's `_layouts/article.html` is replaced by your local `src` for this render. All `include`/`extends` targets still resolve from the server.

### 3 — Fully inline render (no server files required)

```json
{
  "layout": {
    "path": "_layouts/test.html",
    "src":  "<html><body><h1>{{ note.Title() }}</h1>{{ note.HTMLString() }}</body></html>"
  },
  "note": {"src": "# My Title\n\nParagraph."}
}
```

Completely self-contained. Good for AI agents testing a template from scratch.

### 4 — Override a component (e.g. `_blocks.html`)

```json
{
  "layout": {"path": "docs/_layouts/index.html"},
  "note":   {"path": "docs/intro.md"},
  "overrideFiles": [
    {"path": "docs/_layouts/_blocks.html", "src": "...new blocks content..."}
  ]
}
```

`docs/_layouts/index.html` runs on the server as-is. When it tries to `include "docs/_layouts/_blocks.html"`, the server version is replaced by your local `overrideFiles` entry.

### 5 — Test frontmatter and layout property resolution

```json
{
  "layout": {"path": "_layouts/article.html"},
  "note":   {
    "path": "some/note.md",
    "src":  "---\nlayout: article\ntags: [go, testing]\n---\n# Frontmatter Test"
  }
}
```

Server note's real path is used for context lookup; `src` replaces its content and frontmatter.

### 6 — Catch Jet errors before deploying

```json
{
  "layout": {
    "path": "_layouts/article.html",
    "src":  "{{ note.NonExistentMethod() }}"
  },
  "note": {"src": "# Test"}
}
```

The `warnings.layout` array in the response will contain the Jet error. The AI agent or developer can read it directly without opening a browser.

---

## CLI tool

The repo ships `scripts/renderlayout.py` — a wrapper that reads the API key automatically from `.obsidian/plugins/trip2g/data.json`:

```bash
# run from your vault directory (where .obsidian/ lives)
python3 ../scripts/renderlayout.py \
  --layout-file _layouts/mesh/index.html \
  --note-src "# Hello"

# → http://localhost:8081/_system/renderlayout?preview_id=abc123

# fetch rendered HTML directly
python3 ../scripts/renderlayout.py \
  --layout-src "{{ note.M().Debug() }}" --layout-path "/_debug.html" \
  --note-path /my-note \
  --fetch
```

Args: `--layout-path`, `--layout-file`, `--layout-src`, `--note-path`, `--note-file`, `--note-src`, `--fetch`.
Warnings and errors go to stderr; exit code 1 on failure.

For agent-specific instructions and debugging tips (including the `debug()` template function), see `docs/skills/check_templates.md`.

---

## Rendering a single BEM component

To preview one component in isolation, explicitly import its file and yield the HTML and style blocks.
See [[en/user/bem]] for BEM naming conventions and `@lid`/`@did`.

```bash
python3 ../scripts/renderlayout.py \
  --layout-path "/_layouts/mesh/_preview.html" \
  --layout-src '{{ import "_blocks" }}{{ import "bar" }}{{ import "button" }}<style>{{ yield _style_mesh_bar() }}{{ yield _style_mesh_button() }}</style>{{ yield mesh_bar() }}' \
  --note-src "hello"
```

**Caveats:**

- Import paths must be **relative** — `"bar"` not `"/_layouts/mesh/bar"`. Paths are resolved from the normalized layout ID: `/_layouts/mesh/_preview.html` → `/mesh/_preview`, so `"bar"` → `/mesh/bar`.
- **`yield_blocks()` returns empty in preview** — the block wiring phase is skipped. Yield each style block directly: `{{ yield _style_myblock() }}`.
- Component dependencies (e.g. `button` used inside `bar`) must be imported manually.
- `GET /_system/renderlayout` without params always returns the **latest** render — keep it open in a browser while iterating.

---

## AI agent workflow

An agent working on a layout alongside a human can share intermediate results instantly — no deploy, no upload to the vault:

1. Agent edits a layout or note locally.
2. Agent POSTs the changes to `/_system/renderlayout`. Nothing is written to the server vault.
3. Response includes `previewURL` and `warnings`.
4. If all `warnings` arrays are empty, agent replies: *"Here is the intermediate result: [preview link]"*
5. If any `warnings` array is non-empty, agent reads the Jet error, fixes the template, and POSTs again.
6. The human opens the link, sees the rendered page, and gives feedback — all before any file is uploaded.

The buffer holds the last 10 renders. Links expire when the server restarts or the buffer wraps around.

---

## Live preview with file watcher

Install [watchexec](https://github.com/watchexec/watchexec), then run:

```bash
watchexec -e html -- sh -c '
  curl -s -X POST \
    -H "X-API-Key: $API_KEY" \
    -H "Content-Type: application/json" \
    -d "{
      \"layout\": {
        \"path\": \"_layouts/article.html\",
        \"src\":  $(cat _layouts/article.html | jq -Rs .)
      },
      \"note\": {\"path\": \"posts/hello.md\"}
    }" \
    https://yoursite.com/_system/renderlayout
'
```

Keep `https://yoursite.com/_system/renderlayout?live` open in a browser. It reloads instantly when the POST lands — the long-poll connection is already waiting on the server.

---

## Demo vault

Test files are in `docs/demo/`:

| File | Purpose |
|------|---------|
| `docs/demo/_layouts/article.html` | Basic article layout |
| `docs/demo/_layouts/with_component.html` | Layout using `include` |
| `docs/demo/_layouts/components/header.html` | Included header component |
| `docs/demo/hello.md` | Sample note |

Use these as starting points for testing your own layouts.
