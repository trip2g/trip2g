# WebMCP — in-browser agent tools on public note pages

Status: plan (not implemented). Scope: **public, read-only**. We expose the
site's existing read tools (`search`, `similar`, federation search, dynamic note
methods) plus a couple of page-context helpers to an AI agent running **inside
the visitor's browser**, via the emerging WebMCP API (`navigator.modelContext`).

> Admin/write tools (driving the admin panel, GraphQL mutations) are **explicitly
> out of scope** for now. They could come later as a separate, feature-flagged,
> session-gated deliverable, but this document does not design or analyze them.

## What WebMCP is, and how it differs from our server MCP

We already ship a **server-side MCP server** at `POST /_system/mcp`
(`internal/case/mcp/endpoint.go:117`), JSON-RPC 2.0 over HTTP. An *external*
MCP client (Claude Desktop, Cursor, a CLI) connects to it over the network and
authenticates with a personal token (`t2g_*`), an `X-API-Key`, or a federation
JWT (`internal/case/mcp/endpoint.go:48`, `:71`). The agent runs on the user's
machine or in the cloud; trip2g is a remote server it talks to.

**WebMCP is the inverse.** It is a W3C Web Machine Learning Community Group draft
(accepted Sept 2025; latest CG draft Apr 2026 — *not* a ratified standard) that
adds a browser API, `navigator.modelContext`. A **web page** registers tools in
JavaScript; an AI agent **already running in the user's browser** (Chrome's
built-in agent, an Edge agent, or an extension) discovers and calls them. The
agent reuses the page's own context and the user's already-authenticated browser
session — no token handshake, because the calls execute as page JavaScript in the
user's tab.

| | Server MCP (`/_system/mcp`) | WebMCP (this plan) |
|---|---|---|
| Who runs the agent | External client (desktop/cloud) | Inside the visitor's browser |
| Transport | HTTP JSON-RPC to our server | `navigator.modelContext` JS API |
| Registers tools | Our Go server | Our page JavaScript |
| Auth | `t2g_*` / `X-API-Key` / fed JWT | The visitor's existing browser session/cookies |
| Discovery | `tools/list` over the wire | The agent reads tools off the loaded page |
| Availability | Always (any MCP client) | Chrome 146 behind a flag (Feb 2026); broad H2 2026 — progressive enhancement |

Browser status as of mid-2026: Chrome shipped `navigator.modelContext` in an
early preview behind a flag (Chrome 146, Feb 2026); Edge has begun shipping; broad
support is targeted for H2 2026. The API requires a **secure context (HTTPS)**.
Therefore everything here must be a **progressive enhancement**: feature-detect,
and silently no-op when the API is absent.

## The WebMCP API surface (current draft)

The API lives at `navigator.modelContext`. Two registration styles:

**Imperative** — register/unregister individual tools at runtime:

```js
const reg = navigator.modelContext.registerTool({
  name: "search_notes",
  description: "Search this site's notes by query.",
  inputSchema: {
    type: "object",
    properties: { query: { type: "string", description: "Search query" } },
    required: ["query"],
  },
  async execute({ query }) {
    // ... do the work ...
    return { content: [{ type: "text", text: "..." }] };
  },
});
// later: reg.unregister?.() / navigator.modelContext.unregisterTool(...)
```

There is also `navigator.modelContext.provideContext({ tools: [...] })` to set
the full tool list at once (older shape, still present in some builds), and a
declarative HTML-attribute form. For human-in-the-loop, the draft adds
`agent.requestUserInteraction()` so a tool can ask the browser to get explicit
user confirmation before a sensitive action.

Notes on the shape (verify against the shipping build at implementation time, the
draft is moving):
- The handler key is `execute` in the W3C draft; some libraries (`@mcp-b/*`) use
  `handler`. Target `execute`.
- The return value mirrors MCP `CallToolResult`: `{ content: [{ type: "text",
  text }] }`. This is the **same content shape our server already produces**
  (`internal/case/mcp/types.go:61` `CallToolResult`, `:67` `Content`) — a clean fit.

### Imperative vs declarative — choice

Use the **imperative `registerTool`** API. Rationale:
- Our tool set is dynamic per page (a federated KB pointer page vs a plain note
  exposes different tools; dynamic `mcp_method` notes vary), which the imperative
  API expresses naturally.
- We must feature-detect and no-op cleanly; imperative JS is the only path that
  lets us guard on `navigator.modelContext` before doing anything.
- The declarative HTML form would require teaching the quicktemplate
  (`views.html`) to emit tool markup; keeping tool definitions in one JS file is
  far easier to maintain and test.

Keep `provideContext` as a fallback call only if a target browser lacks
`registerTool` (capability-detect both).

## Tool inventory to expose (public, read-only)

All of these already exist server-side and are already safe for anonymous public
access (the open-KB anonymous path: `/_system/mcp` serves public KBs with no
token, see the GET info page `internal/case/mcp/endpoint.go:136` "Public access:
no token required for open knowledge bases", and the anonymous branch in
`authenticateAnonymousRequest` returning `ctx, nil` at `:93`). Server-side access
control still applies on every call — private/subscriber notes are filtered by
`canReadMCPNote` (`internal/case/mcp/resolve.go:481`) and
`filterSearchResults` (`:409`), so an unauthenticated browser agent sees exactly
what an anonymous visitor may see. **This is the core reason the public surface is
safe: we are not bypassing any check, we are surfacing the anonymous tool set.**

Tools, with their server schemas (source: `handleToolsList`,
`internal/case/mcp/resolve.go:171`):

| WebMCP tool | Server tool | Input schema (from `resolve.go`) |
|---|---|---|
| `search` | `search` (`:174`) | `{ query: string }` (required) |
| `similar` | `similar` (`:185`) | `{ path?, href?, pid?, note_id?, limit? }` |
| `note_html` | `note_html` (`:199`) | `{ path?, href?, pid?, note_id?, match_id?, context_words?, toc_path?[] }` |
| `federated_search` | `federated_search` (`:219`) | `{ query (req), kb_id?, kb_ids?[] }` |
| `federated_similar` | `federated_similar` (`:232`) | `{ kb_id (req), path?, href?, pid?, note_id?, limit? }` |
| `federated_note_html` | `federated_note_html` (`:248`) | `{ kb_id (req), path?, href?, pid?, note_id?, match_id? }` |
| dynamic `mcp_method` notes | per-note (`:266`) | `{}` — content is the response |

The dynamic-method tools come from notes with `mcp_method` frontmatter
(`internal/case/rendernotepage`/model `MCPMethod`); they are discovered
server-side and filtered by readability. To keep the browser tool list correct,
**the page must learn which dynamic tools are available** — fetch the list at
runtime (see "Reuse `/_system/mcp`" below) rather than hardcoding it.

**Out of scope (do not register):** `graphql_introspection` and `graphql_request`
(`resolve.go:285`–`:307`). These are gated behind an API key with
`EnableMcpAdminTools` and must never appear in the public browser surface.

### Page-context tools

"Tell the agent what's on the page = use it." The rendered note page already
exposes its identity in a global config script
(`internal/defaulttemplate/views.html:52`–`:55`):

```js
note_path: "...",
note_path_id: <PathID>,
note_lang: "...",
note_version_id: "...",
```

That gives us, with zero new backend work, the data for page-context tools:

- `get_current_page` — returns `{ note_path, note_path_id, note_lang, title, url }`
  read straight from that global object. Lets the agent know what the user is
  looking at without guessing from the URL.
- `find_related_to_this_page` — convenience wrapper that calls the `similar` tool
  using the current page's `note_path_id` (`pid`), so the user can say "find
  related" with no arguments.
- `read_section` — wrapper over `note_html` with the current `pid` and a
  `toc_path`/`match_id`, for "summarize this section".

These page-context tools are pure client-side conveniences composed from the read
tools above; they add no new server surface.

## Auth / session strategy

The whole public surface is **anonymous**. The browser agent runs in the
visitor's tab; whatever cookies the visitor has are sent automatically by the
browser on same-origin requests. We do **not** add any token. Concretely:

- If the tool handler **proxies to `/_system/mcp`** (recommended, below), the
  fetch is same-origin and the browser attaches the visitor's session cookies
  automatically. A logged-in subscriber's agent therefore transparently sees
  their subscriber content; an anonymous visitor sees only public content —
  exactly mirroring what they'd see by browsing. No special handling needed.
- No `Authorization` header, no `X-API-Key`. We never expose `t2g_*` tokens to
  page JS.

Security posture for this scope is simple because **the server is the trust
boundary** and it already enforces per-note access on every call. The browser
tools are a thin presentation layer over endpoints a visitor can already hit.
The one rule: never register the admin GraphQL tools, never send admin
credentials from page JS. (A future admin surface would need its own session
gating and confirmation design — deliberately not addressed here.)

## Reuse `/_system/mcp` vs call backends directly

**Recommendation: proxy to the existing `/_system/mcp` endpoint.** Each WebMCP
tool's `execute` builds a JSON-RPC `tools/call` request and `fetch`es
`/_system/mcp` same-origin.

Trade-offs:

| | Proxy to `/_system/mcp` (chosen) | Call GraphQL/search directly |
|---|---|---|
| Auth | Free — cookies ride along same-origin | Must wire each backend's auth from JS |
| Code reuse | One code path; server already formats results | Reimplement search/similar/federation glue in JS |
| Access control | Enforced once, server-side (`canReadMCPNote`) | Must re-enforce per call client-side (unsafe) |
| Result shape | Server returns MCP `content` already — pass through | Must map GraphQL/search shapes to MCP `content` |
| Tool discovery | `tools/list` over the same fetch gives dynamic tools | No discovery; hardcode tools |
| Cost | One extra same-origin round trip per call | Marginally fewer hops |

Proxying wins decisively: the server is already the access-control and formatting
authority, and the response is already in WebMCP's `content` shape. The browser
layer becomes a small adapter:

1. On load, `fetch('/_system/mcp', {method:'POST', credentials:'same-origin'})`
   with `{jsonrpc:"2.0", id, method:"tools/list"}` to get the live tool list
   (built-ins + readable dynamic methods, with admin tools absent for anonymous).
2. For each returned tool, `registerTool({ name, description, inputSchema,
   execute })`.
3. `execute(args)` POSTs `{method:"tools/call", params:{name, arguments:args}}`
   and returns the `result` (already `{content:[...]}`), or maps a JSON-RPC
   `error` to `{ content:[{type:"text", text}], isError:true }`.

Page-context tools (`get_current_page` etc.) are registered locally without a
server round trip (or compose the proxy call, e.g. `find_related_to_this_page`
forwards to the `similar` proxy).

This also keeps the WebMCP list automatically in sync with the server: filter a
note to subscribers and it disappears from both surfaces with no client change.

## Where / how to inject the JS

Follow the **mermaid per-note widget pattern** (`docs/dev/mermaid.md`): a small
committed glue bundle under `assets/`, embedded via `assets/embed.go`, emitted as
a `<script defer>` in the rendered page.

- New bundle: `assets/webmcp/src/index.ts` → built artifact `assets/webmcp.js`
  (mirrors `assets/mermaid/` → `assets/mermaid.js`).
- Embed it: add `webmcp.js` to the `//go:embed` list in `assets/embed.go:8`.
- Serve + cache-bust: already handled by the asset FS and `AssetURL()`
  (cache-busting hash), same as mermaid.

**Injection point — unconditional on note pages, not per-note.** Mermaid is
per-note because only some notes have diagrams; WebMCP tools (search, federation,
page-context) make sense on **every** rendered note page. Two clean options:

- **Option A (preferred):** add it to the core bootstrap list `UserJSURLs()`
  (`cmd/server/main.go`, the core scripts returned for note pages) so every
  default-template page loads it. The template already iterates `ctx.JSURLs`
  into `<script defer>` tags (`internal/defaulttemplate/views.html:89`–`:91`).
- **Option B:** append it in `buildDefaultTemplateCtx`
  (`internal/case/rendernotepage/endpoint.go:495`–`:502`), next to the mermaid
  line, but without the `HasCodeLanguage` guard (i.e. always). Use this if we
  want it on note pages only and not other default-template pages.

The script's first action is **feature detection**:

```js
if (!('modelContext' in navigator) || typeof navigator.modelContext?.registerTool !== 'function') {
  return; // WebMCP unsupported — silent no-op, page works as before
}
```

It must also tolerate non-secure contexts (the API simply won't exist over
plain HTTP, so the guard above covers it).

The admin `$mol` app (`/admin`) is **not** a target of this plan; do not inject
the WebMCP bundle there.

## Feature-detection & graceful degradation

- Guard on `navigator.modelContext` and `registerTool` before any call.
- Wrap registration in `try/catch`; a draft-API mismatch must never break the
  page.
- If `tools/list` fetch fails, fall back to registering the **static** built-in
  tools only (search/similar/note_html/federated_*), skipping dynamic discovery.
- Unregister on `pagehide`/`beforeunload` if the API exposes unregister, to avoid
  stale tools in SPA-like navigations (note pages are full reloads today, so this
  is defensive).

## Build commands

Mirror the mermaid widget build (`docs/dev/mermaid.md` "Build"):

| Command | When |
|---|---|
| `npm run webmcp` | After editing `assets/webmcp/src/*` — builds `assets/webmcp.js` (add this script to `package.json`, mirroring `npm run mermaid`) |
| `go generate ./internal/defaulttemplate/...` | Only if `views.html` is touched (Option B without template edits needs none) |
| `npm run build` | TS typecheck + Vite (does **not** build the widget bundle) |
| `npm run test:e2e` | Playwright E2E (see tests below) |

`assets/webmcp.js` is a committed artifact, like `mermaid.js` / `chart.js`.

## Story-by-story breakdown

Written TDD-first (Red → Green → Refactor), matching `docs/dev/mcp.md` style.

### Story 1 — WebMCP glue bundle + feature detection
- **Build:** `assets/webmcp/src/index.ts`, `esbuild`/vite config, `npm run webmcp`
  script in `package.json`. Output `assets/webmcp.js`.
- **Behavior:** on load, feature-detect `navigator.modelContext.registerTool`;
  no-op silently if absent or non-secure context.
- **TDD:** unit-test the detection guard with a stubbed `navigator` (jsdom):
  absent → does nothing, no throw; present → proceeds. Test the JSON-RPC request
  builder and the `result`/`error` → MCP-content mapping in isolation.
- **Done when:** importing the bundle with no `modelContext` is a clean no-op.

### Story 2 — Serve and inject the bundle on note pages
- Add `webmcp.js` to `//go:embed` (`assets/embed.go:8`).
- Inject via Option A (`UserJSURLs()`) or Option B (always-append in
  `buildDefaultTemplateCtx`).
- **TDD:** Go test asserting the asset is embedded and resolvable via the asset
  FS / `AssetURL`. Render-path test (or E2E) asserting a `<script>` for
  `webmcp.js` appears on a rendered note page.
- **Done when:** every default-template note page ships the (cache-busted)
  `webmcp.js` tag.

### Story 3 — Register the built-in read tools (proxy to `/_system/mcp`)
- In the bundle: register `search`, `similar`, `note_html`, `federated_search`,
  `federated_similar`, `federated_note_html` with the schemas from
  `resolve.go:174`–`:262`. Each `execute` POSTs a `tools/call` to `/_system/mcp`
  with `credentials:'same-origin'` and returns the server `result`.
- **TDD:** mock `fetch`; assert each tool sends the right JSON-RPC body and that a
  server `result` is returned verbatim and an `error` becomes
  `{isError:true,...}`. E2E (Playwright, with a stub for `navigator.modelContext`
  capturing registrations): on `/some-note`, `search` is registered and calling
  it returns real results.
- **Done when:** the six read tools are callable through the stub and round-trip
  to the server.

### Story 4 — Dynamic tool discovery
- On load, call `tools/list` against `/_system/mcp`; register any returned
  dynamic `mcp_method` tools not already registered (and trust the server to omit
  admin/private tools for the anonymous/visitor session).
- **TDD:** mock `tools/list` returning a dynamic tool + a built-in; assert the
  dynamic one is registered and `graphql_*` (if ever present) is filtered out
  client-side as a belt-and-suspenders guard. E2E: a note with `mcp_method`
  frontmatter exposes its tool on the page.
- **Done when:** dynamic note tools appear in the browser agent's tool list and
  return the note content.

### Story 5 — Page-context tools
- Register `get_current_page` (reads `window`'s note config object from
  `views.html:52`–`:55`), `find_related_to_this_page` (forwards to `similar` with
  the current `note_path_id`), `read_section` (forwards to `note_html`).
- **TDD:** unit-test that `get_current_page` returns the globals and that
  `find_related_to_this_page` calls the `similar` proxy with the page `pid`. E2E:
  on a real note page the page-context tool returns the correct path/id.
- **Done when:** an in-browser agent can ask "what page am I on / find related"
  with no arguments.

### Story 6 — Docs & rollout
- User docs (bilingual, per project convention): `docs/en/user/webmcp.md` +
  `docs/ru/user/webmcp.md` — how to enable WebMCP in a supported browser and what
  the agent can do on a trip2g site.
- Changelog entry in `docs/en/changelog.md` + `docs/ru/changelog.md`
  (What/Why/How).
- Optional: a `docs/demo/webmcp.md` demo note.
- **Done when:** docs ship and the demo page demonstrates a browser agent calling
  `search`.

## Optional: feature-flag gating

The public read-only surface is low-risk (it only re-exposes anonymous,
access-controlled endpoints), so a flag is optional. If desired, gate injection
behind a config/feature flag (the project has a feature-flags system, see
`docs/dev/features.md`) so an operator can disable the WebMCP script per
instance. A flag becomes **mandatory** the day any admin/write tool is added —
but that is out of scope here.

## Files (anticipated)

| File | Role |
|---|---|
| `assets/webmcp/src/index.ts` | glue: feature-detect, discover, register tools, proxy to `/_system/mcp` |
| `assets/webmcp.js` | built artifact (committed) |
| `assets/embed.go` | embed `webmcp.js` |
| `cmd/server/main.go` *or* `internal/case/rendernotepage/endpoint.go` | inject the script tag (Option A / B) |
| `package.json` | `npm run webmcp` build script |
| `docs/en/user/webmcp.md`, `docs/ru/user/webmcp.md` | user docs |
| `e2e/vault.spec.js` | E2E: tools registered + callable on note pages |

## References (WebMCP spec, mid-2026)

- W3C WebML CG — WebMCP Community Group Draft (`navigator.modelContext`,
  `registerTool`/`unregisterTool`/`provideContext`, `agent.requestUserInteraction`).
- Chrome 146 early preview behind a flag (Feb 2026); Edge shipping; broad support
  targeted H2 2026. Requires a secure context (HTTPS).
- Verify the exact `execute` vs `handler` key and the descriptor shape against the
  shipping browser build at implementation time — the draft is still moving.
