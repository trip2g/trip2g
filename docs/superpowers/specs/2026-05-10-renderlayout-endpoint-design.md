# Spec: `/_system/renderlayout` — Layout Preview Endpoint

**Date:** 2026-05-10  
**Status:** Draft

## Problem

AI agents and developers need to test Jet layout templates without deploying to the server context. Currently there is no way to:
- Render a layout with raw HTML content and see Jet compilation/runtime errors
- Preview how an existing note looks with a modified layout
- Share a preview link without touching production data

The existing `/_system/layouts/render` accepts only `JSONLayout` + a server-side note path, which does not cover local development workflows.

## Solution

A new endpoint `/_system/renderlayout` that accepts raw Jet HTML and/or markdown, renders it in the server's full context (with real note data, real layout loader), stores the result in an in-memory ring buffer, and returns a shareable preview URL.

Primary users:
- **AI agent** — POST after each edit, read `warnings[]` in the JSON response, iterate without human involvement
- **Developer** — open `?live` in a browser, run a local file watcher (e.g. watchexec) that POSTs on save, see changes reload automatically

## Endpoints

### POST `/_system/renderlayout`

**Auth:** `X-API-Key` header (via existing `checkapikey` use case).

**Content-Type:** `application/json`

**Request body:**

```json
{
  "layout": {
    "path": "_layouts/article.html",
    "content": "{% extends '_layout' %}..."
  },
  "note": {
    "path": "posts/hello.md",
    "content": "# Hello\nMarkdown here"
  },
  "extra_layouts": {
    "components/header.html": "...",
    "sidebar.html": "..."
  }
}
```

Rules:
- `layout.path` and `layout.content` are mutually exclusive; exactly one is required.
- `layout.path` — path to an existing server layout file (looked up via `LayoutSourceFiles`).
- `layout.content` — raw Jet HTML string.
- `note` is optional. If omitted, a stub note with empty content is used.
- `note.path` and `note.content` are mutually exclusive.
- `note.path` — path to an existing server note (looked up via `LatestNoteViews`).
- `note.content` — markdown string; rendered via mdloader into `note.Content`.
- `extra_layouts` — map of virtual path → Jet HTML content. These override server layout files of the same path when Jet resolves `include`/`extends`. Optional.

**Response:**

```json
{
  "preview_id":  "a3f9b2c1",
  "preview_url": "/_system/renderlayout?preview_id=a3f9b2c1",
  "warnings":    ["jet: line 12: undefined variable 'note.Tags'"]
}
```

- `warnings` contains both Jet compilation errors and runtime errors. An empty array means a clean render.
- `preview_id` is an 8-character random hex string (not a sequential ID, safe to share).
- On success the entry is added to the ring buffer and the `current` pointer is updated.
- A render with warnings still produces a (partial) HTML entry in the ring buffer.

**HTTP status codes:**

| Code | Condition |
|------|-----------|
| 200  | Rendered (possibly with warnings) |
| 400  | Invalid JSON body or missing required fields |
| 401  | Missing or invalid API key |
| 404  | `layout.path` or `note.path` not found on server |

---

### GET `/_system/renderlayout`

**Auth:** `X-API-Key` header **or** admin session token (IsAdmin). The latter allows opening the URL directly in a browser while logged into the admin.

Returns the HTML of the `current` entry (last POST). Content-Type: `text/html`.

Returns `404` if the ring buffer is empty.

---

### GET `/_system/renderlayout?preview_id=xxx`

**Auth:** None. The `preview_id` is unguessable; possession implies authorization.

Returns the HTML of the specified entry. Content-Type: `text/html`.

Returns `404` if the `preview_id` is not in the buffer (expired or never existed).

---

### GET `/_system/renderlayout?live`

**Auth:** `X-API-Key` header **or** admin session token (IsAdmin).

Same as plain GET, but injects a long-polling script just before `</body>`:

```html
<script>
(function(){
  var v = 7; /* current version at render time */
  function poll(){
    fetch('/_system/renderlayout?longpolling&since='+v)
      .then(function(r){ return r.json(); })
      .then(function(d){
        if(d.action==='reload'){ location.reload(); }
        else { poll(); }
      })
      .catch(function(){ setTimeout(poll, 3000); });
  }
  poll();
})();
</script>
```

If the rendered HTML has no `</body>` tag, the script is appended at the end.

---

### GET `/_system/renderlayout?longpolling&since=N`

**Auth:** None.

Long-polling notification endpoint. Hangs until a new render arrives (version > `since`) or 30 seconds elapse.

**Response:**

```json
{"action": "reload", "version": 8}
```
or
```json
{"action": "wait", "version": 7}
```

The browser JS reconnects immediately on `"wait"`.

---

## Ring Buffer

Location: in-memory struct in the app state, injected via the `Env` interface.

```go
type PreviewBuffer struct {
    mu      sync.RWMutex
    entries []PreviewEntry   // len == capacity from config
    head    int              // next write index (wraps)
    count   int              // filled slots
    version int              // monotonic, incremented on each POST
    notify  chan struct{}     // closed on new render, replaced with new channel
}

type PreviewEntry struct {
    ID      string
    HTML    string
    Warnings []string
    Version int
}
```

- Default capacity: **10**. Configurable in `internal/appconfig` as `PreviewBufferSize int`.
- Oldest entries are overwritten when the buffer is full (FIFO ring).
- `current` is the entry written last (identified by the latest `version`).
- `notify` is replaced atomically: on each POST, the old channel is closed (waking all waiters), a new channel is created and stored.

---

## Jet Loader for Virtual Files

The existing `layoutloader.Load()` uses a `jetLoader` backed by server files. For this endpoint a composite loader is constructed:

1. Check `extra_layouts` map (in-memory, keyed by path).
2. If not found, delegate to the server's layout files.

This composite loader is passed to `jet.NewSet()` when compiling the preview template. It is not reused across requests (created per POST).

Jet errors during `Set.GetTemplate()` (compilation) and `View.Execute()` (runtime) are both captured and returned as `warnings`.

---

## Note Rendering

| Input | Behaviour |
|-------|-----------|
| `note.path` | Looked up via `env.LatestNoteViews().GetByPath()`. Error 404 if not found. |
| `note.content` | Markdown rendered via `mdloader` to produce `Content` HTML. Frontmatter parsed for title, tags, etc. A `templateviews.Note` is constructed from the result. |
| omitted | Stub note: empty content, empty title, no tags. |

The rendered note is passed to the Jet template as `vars["note"]` (same as production rendering). `vars["nvs"]` is populated from `env.LatestNoteViews()` as usual.

---

## Implementation Location

Following the existing use case pattern:

```
internal/case/admin/renderlayoutpreview/   ← existing (JSONLayout, unchanged)
internal/case/admin/renderpreview/         ← new
  resolve.go    — Env interface + Resolve()
  endpoint.go   — HTTP handler, routes POST + GET
  buffer.go     — PreviewBuffer ring buffer
  loader.go     — composite Jet loader (extra_layouts + server fallback)
```

The `PreviewBuffer` is instantiated once at startup (in `internal/app/`) and injected into the `Env` of this use case.

Router registration: after editing `internal/router/`, run `go generate ./internal/router/...`.

---

## Developer Workflow Example

```bash
# Terminal 1 — open live preview in browser
open "https://mysite.com/_system/renderlayout?live" \
  -H "X-API-Key: $TRIP2G_API_KEY"
# (or curl -s ... | open -f for initial load)

# Terminal 2 — watch local layout file, POST on every save
watchexec -e html -- sh -c '
  curl -s -X POST \
    -H "X-API-Key: $TRIP2G_API_KEY" \
    -H "Content-Type: application/json" \
    -d "{
      \"layout\": {\"content\": $(cat ./_layouts/article.html | jq -Rs .)},
      \"note\":   {\"path\": \"posts/hello.md\"}
    }" \
    https://mysite.com/_system/renderlayout
'
```

Browser reloads automatically within ~1.5s of each save. AI agent uses the same POST and reads `warnings[]` directly from the JSON response.

---

## Out of Scope

- Local browser folder picker (Phase 2, separate design)
- Persisting previews across server restarts
- Authentication on `?longpolling` and `?preview_id` GET (security by unguessable ID is sufficient for a dev tool)
- Serving layout assets (CSS, fonts) — assets render empty/broken; acceptable for layout structure testing
