---
subgraph: private
---

# Agent context

## Where you are

This folder is a **local Obsidian vault** — plain Markdown files on disk — paired with a live trip2g site at {{publicUrl}}. The files here are what you edit; the site is what readers see. Three things follow from that:

- **Nothing is live until you sync.** Editing a file changes nothing on the site; publishing is an explicit step — see [Sync](#sync). The same files are also open in Obsidian on the user's machine, so treat them as shared state.
- **Synced notes become searchable.** Published notes are indexed for vector search (RAG) through the `my-trip2g-instance` MCP server — `search` → `note_html`. A note you never synced does not exist for it.
- **Managing the site needs the user's consent.** Admin GraphQL is off by default. If a task requires it (subgraphs, API keys, settings), ask the user to enable it — see [Enable admin GraphQL](#enable-admin-graphql). Check whether you already have it by looking for the introspection tool on `my-trip2g-instance`; don't assume.

## Quick start (for an agent)

- **Publish / first sync:** run `node .obsidian/plugins/trip2g/trip2g-sync.mjs --folder .` from the vault root — no flags or credentials needed, they are read from `.obsidian/plugins/trip2g/data.json`. Every note becomes a page; `_index.md` is the homepage.
- **Edit content:** notes are plain Markdown files in this folder; edit them, then sync.
- **Manage the site:** call admin GraphQL / search via the `my-trip2g-instance` MCP server (`.mcp.json`).

## How publishing works

- **Every note becomes a page** on the site, at a URL derived from its path.
- **Hidden notes:** any path component starting with `_` (e.g. `_index.md`, `_layouts/`, `drafts/_wip.md`) is a system/hidden note — excluded from listings and search, but still reachable directly when you have access.
- **Access is closed by default:** a new note is gated behind the paywall (subscriber-only) — on a fresh site with no subscribers, only you (owner/admin) see it. To open it up, either add `free: true` to its frontmatter, or put it in a subgraph and set that subgraph's access to **public** (vs. subscription-only or admin-only) via `subgraphs:` in frontmatter.
- **Templating engine:** pages are fully customizable via Jet-based layouts under `_layouts/` — see [Custom layouts](#custom-layouts).
- **Rich content:** **Mermaid** diagrams, **datachart** (charts from CSV), and an **RSS** feed are rendered automatically.
- **Admin via MCP:** manage the whole site (notes, subgraphs, keys, settings) by calling admin GraphQL through `my-trip2g-instance` — see [Enable admin GraphQL](#enable-admin-graphql).

Look up any of these on `trip2g-docs-public-hub` (`search` for "subgraphs", "mermaid", "datachart", "rss", "routes") — see [Platform docs](#platform-docs), already wired in and usable right now, no setup needed.

## Custom layouts

Pages render through Jet templates under `_layouts/`; a note opts in via `layout: <path>` frontmatter (path relative to `_layouts/`, no extension). Layouts build on reusable **Jet blocks** — define them once with `{{ block name(args) }}...{{ end }}` (conventionally in a `_blocks.html`), then `{{ import "_blocks" }}` and `{{ yield name(args) }}` them from any layout.

Working example: `_layouts/demo/simple.html` imports a block from `_layouts/demo/_blocks.html` and yields it; `simple_layout.md` uses it via `layout: demo/simple` — copy this pattern to build your own. Full API and JSON-layout format: search `trip2g-docs-public-hub` for **Layouts API** and **JSON Layouts**.

**Ready-made: Kanban board.** [github.com/trip2g/kanban_template](https://github.com/trip2g/kanban_template) renders any obsidian-kanban note as a live draggable board that saves edits back into the note. Install into the vault, then add `layout: kanban` to a kanban note's frontmatter:

```bash
mkdir -p _layouts
curl -L -o _layouts/kanban.html \
  https://github.com/trip2g/kanban_template/releases/latest/download/kanban.html
```

Details: search `trip2g-docs-public-hub` for **Kanban board template**.

## MCP servers

`.mcp.json` / `codex.json` are pre-configured with your credentials.

| Name | Endpoint | Use for |
|------|----------|---------|
| `my-trip2g-instance` | `{{publicUrl}}/_system/mcp` | Vector search over **your** notes |
| `trip2g-docs-public-hub` | `https://trip2g.com/_system/mcp` | Platform docs & GraphQL query examples |

**Antigravity:** copy `antigravity-mcp-config.json` → `~/.gemini/antigravity/mcp_config.json`

### Search your notes (vector search)

`my-trip2g-instance` indexes every published note for semantic search. Core tools:

| Tool | Purpose |
|------|---------|
| `search(query)` | Vector search — returns matches with a `toc` outline and `matches[].toc_path` |
| `note_html(pid, toc_path?)` | Full note, or just one section when `toc_path` is given |
| `similar(pid)` | Notes similar to a given one |

**Fuzzy Pointer — don't load whole notes.** `search` does not return the exact line; it returns a *fuzzy pointer*: the `toc_path` breadcrumb of the section that matched (e.g. `["Goroutines", "Worker pool"]`). Resolve it to just that section instead of the whole note:

```
1. search("limit concurrency Go")   → match with toc_path: ["Goroutines","Worker pool"]
2. note_html(pid=42, toc_path=["Goroutines","Worker pool"])   → ~300 tokens, not ~3000
```

Loading `note_html(pid)` without `toc_path` costs ~10× the tokens and buries the answer deep in context. Always pass the `toc_path` from the search result. Need a sibling section? Reuse `toc` items from the same result — no second `search`. Full guide: search `trip2g-docs-public-hub` for **Fuzzy Pointer** and **MCP Server**.

### Federation — search across other bases

Your hub can fan out to other trip2g/MCP bases through one endpoint. Extra tools on `my-trip2g-instance`:

| Tool | Purpose |
|------|---------|
| `federated_search(query, kb_id?)` | Search peers — fan-out to all, or target one `kb_id` |
| `federated_similar(pid, kb_id)` | Similar notes on a specific peer |
| `federated_note_html(pid, kb_id)` | Note content from a specific peer |

Register a peer by adding a note with `mcp_federation_kb_url` in its frontmatter (body = when to use that base). When local `search` surfaces such a KB-note, it returns a `kind: "federation_kb"` marker telling you which `kb_id` to query. Full guide: search `trip2g-docs-public-hub` for **MCP Federation**.

### One graph per owner — federate the rest

trip2g's editing model assumes **one editor per vault**. Two or more writers on the same graph step on each other quickly — sync conflicts, clashing note structures, unclear ownership. Don't scale a vault by adding editors; scale by adding vaults: each knowledge base has a single owner responsible for their slice of the graph, and the bases link up through federation (`mcp_federation_kb_url` notes + `federated_search`).

This also works across a local and a server instance at once. A practical split for an agent:

- **Local trip2g** = your private RAG store. Fast `search` over your own notes, cheap section reads via `note_html` + `toc_path`.
- **Server trip2g** = the shared graph. Reach it from the local instance through `federated_search` / `federated_note_html` — no direct write access needed to read someone else's slice.

Write only to the graph you own; read everything else through federation.

### Platform docs

`trip2g-docs-public-hub` is already wired into `.mcp.json` and usable right now, no setup — use it to look up how anything works (publishing, templates, subgraphs, GraphQL, monetization, self-hosting). Run `search(query)` against it the same way, then fetch sections with the fuzzy-pointer workflow above.

## Sync

Three ways to sync, same notes:

- **Obsidian plugin** — built-in, configured automatically.
- **CLI** — bundled at `.obsidian/plugins/trip2g/trip2g-sync.mjs` (for agents). Run from the vault root:

  ```bash
  node .obsidian/plugins/trip2g/trip2g-sync.mjs --folder .
  ```

  The CLI auto-reads `apiUrl` and `apiKey` from `.obsidian/plugins/trip2g/data.json`, so no `--api-key`/`--api-url` flags are needed. Add `--two-way` to also pull server changes, `--dry-run` to preview. Override with `TRIP2G_API_KEY` / `TRIP2G_ENDPOINT` env vars if needed.

- **Git** — push/pull notes over Git smart-HTTP at `{{publicUrl}}/_system/git`. First issue a **Git token** in `{{publicUrl}}/admin` → Integrations → Git Tokens, then clone (auth is HTTP Basic — username is always `user`, password is the Git token):

  ```bash
  git clone {{publicUrl}}/_system/git my-notes
  # Username: user
  # Password: <git-token>
  ```

  See [Git round-trip](#git-round-trip-for-agents) below.

### Git round-trip (for agents)

The whole site is a git repo — clone, edit markdown, commit, push; the server applies your commits to the live site:

```bash
git clone {{publicUrl}}/_system/git site   # user / <git-token>
# non-interactive: insert user:<git-token>@ after https:// in the URL
cd site
# ... edit .md files ...
git add -A && git commit -m "agent: update notes"
git push   # live once the push succeeds
```

Rules of the road:

- **The database is canonical; git is a mirror** rebuilt from it on every clone/pull/push. Plugin and editor changes show up as `server sync` snapshot commits — don't treat git history as the site's edit history.
- **Single branch `master`, fast-forward only.** A rejected push means the site changed under you: `git pull --rebase && git push`.
- **Only `.md` / `.html` changes apply.** Pushed binary assets are ignored — upload images via the sync CLI instead. Deleting a note file in git hides that note on the site.
- If applying a push fails server-side, the push is rolled back — the mirror never diverges from the site.

Full guide: search `trip2g-docs-public-hub` for **Git Access**.

## Enable admin GraphQL

Managing the site (subgraphs, keys, settings) requires **"Execute admin GraphQL"** on the API key. It is off unless the user turned it on at download time — an agent cannot grant it to itself, so ask the user to open the link below:

[Open API key settings]({{publicUrl}}/admin#!nav=integrations/integrations_nav=apikeys/id=key1) → Enable Admin GraphQL

Or manually: `{{publicUrl}}/admin` → Integrations → API Keys → your key → Enable Admin GraphQL

After enabling, an introspection tool becomes available in `my-trip2g-instance`.

## Calling GraphQL from the shell

The bundled CLI sends queries for you — same credentials, no `curl`, no flags:

```bash
node .obsidian/plugins/trip2g/trip2g-sync.mjs graphql '{publicUrl}'
node .obsidian/plugins/trip2g/trip2g-sync.mjs graphql '<query>' '<variables json>'
```

**Start by asking the schema what exists** instead of guessing field names:

```bash
node .obsidian/plugins/trip2g/trip2g-sync.mjs graphql --introspect                 # Query, Mutation, AdminQuery, AdminMutation
node .obsidian/plugins/trip2g/trip2g-sync.mjs graphql --introspect AdminMutation    # one type, as SDL
```

Admin operations live under the `admin` field, never on `Mutation` directly. Add a user, then make that user an admin:

```bash
CLI=.obsidian/plugins/trip2g/trip2g-sync.mjs

node $CLI graphql 'mutation($i:CreateUserInput!){admin{createUser(input:$i){
  __typename ... on CreateUserPayload{user{id email}} ... on ErrorPayload{message}}}}' \
  '{"i":{"email":"bob@example.com"}}'

node $CLI graphql 'mutation($i:CreateAdminInput!){admin{createAdmin(input:$i){
  __typename ... on ErrorPayload{message}}}}' \
  '{"i":{"userId":2}}'
```

Ask for `__typename` and the `ErrorPayload` branch on every mutation — that is how a validation error reaches you instead of looking like an empty success. A non-zero exit means the call failed: the messages go to stderr, and any partial data still goes to stdout.

Without **Execute admin GraphQL** on your key these return an authorization error — that is the signal to ask the user to enable it, not to retry.
