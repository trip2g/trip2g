# Agent context

Publishing platform: Obsidian vault → website. Sync via Obsidian plugin, CLI, or Git.

## How publishing works

- **Every note becomes a page** on the site, at a URL derived from its path.
- **Hidden notes:** any path component starting with `_` (e.g. `_index.md`, `_layouts/`, `drafts/_wip.md`) is a system/hidden note — excluded from listings and search, but still reachable directly when you have access.
- **Subgraphs + access control:** group notes into subgraphs and gate each one — **public**, **subscription-only**, or admin-only. Set `subgraphs:` in frontmatter; `free: true` makes a note public.
- **Rich content:** **Mermaid** diagrams, **datachart** (charts from CSV), and an **RSS** feed are rendered automatically.
- **Admin via MCP:** manage the whole site (notes, subgraphs, keys, settings) by calling admin GraphQL through `my-trip2g-instance` — see [Enable admin GraphQL](#enable-admin-graphql).

Look up any of these on `trip2g-docs-public-hub` (`search` for "subgraphs", "mermaid", "datachart", "rss", "routes").

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

### Platform docs

Use `trip2g-docs-public-hub` to look up how the platform works (publishing, templates, GraphQL, monetization, self-hosting). Run `search(query)` against it the same way, then fetch sections with the fuzzy-pointer workflow above.

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

  See `trip2g-docs-public-hub` (`search` for "git sync") for the full workflow.

## Enable admin GraphQL

To manage the site via agent, enable **"Execute admin GraphQL"** for your API key:

[Open API key settings]({{publicUrl}}/admin#!nav=integrations/integrations_nav=apikeys/id=key1) → Enable Admin GraphQL

Or manually: `{{publicUrl}}/admin` → Integrations → API Keys → your key → Enable Admin GraphQL

After enabling, an introspection tool becomes available in `my-trip2g-instance`.
Use `trip2g-docs-public-hub` for query examples or run introspection directly.
