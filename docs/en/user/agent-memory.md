---
title: "Long-term memory for AI agents with trip2g"
free: true
lang_redirect: "[[ru/user/agent-memory]]"
---

Self-host trip2g in Docker, point your agent's MCP client at it, sync your notes in, and your agent reads and searches them as persistent long-term memory. Focused section reads keep token cost low — typically ~15× cheaper than dumping the whole note.

## 1. Run the daemon (Docker)

trip2g needs an S3-compatible store for assets. For a local dev setup, start MinIO first, then the app.

```bash
# MinIO (skip if you already have one)
docker run -d --name trip2g-minio -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=trip2g -e MINIO_ROOT_PASSWORD=trip2g-secret \
  minio/minio:latest server /data --console-address ":9001"

# trip2g app on port 24081 (health on 24082), fresh local DB
# This pulls the current published image. To use a locally built image instead,
# run `docker build -t trip2g:local .` from a trip2g source checkout and replace
# the image tag below with `trip2g:local`.
mkdir -p /tmp/trip2g-local
docker run -d --name trip2g-local --network host \
  -e LISTEN_ADDR=0.0.0.0:24081 -e INTERNAL_LISTEN_ADDR=:24082 \
  -e DB_FILE=/data/local.sqlite3 \
  -e DEV=true \
  -e OWNER_EMAIL=hello@example.com \
  -e MINIO_ENDPOINT=localhost:9000 \
  -e MINIO_ACCESS_KEY_ID=trip2g -e MINIO_SECRET_KEY=trip2g-secret \
  -e MINIO_BUCKET=trip2g-local -e MINIO_USE_SSL=false \
  -e PUBLIC_URL=http://localhost:24081 \
  -e JWT_SECRET=dev-secret-not-for-prod \
  -e USER_TOKEN_INSECURE=true \
  -e GIT_API_REPO_PATH=/data/git -e GIT_API_BASE_PATH=/git \
  -e RESEND_API_KEY=dev -e MAIL_FROM=dev@example.com \
  -v /tmp/trip2g-local:/data \
  ghcr.io/trip2g/trip2g:latest

# wait until healthy — also verify the container is actually running
docker ps | grep trip2g-local
until curl -sf http://localhost:24082/healthz >/dev/null; do sleep 1; done; echo "up"
```

The daemon listens on **port 24081**. The MCP endpoint is at `http://localhost:24081/_system/mcp`.

Notes:
- `DEV=true` enables a fixed sign-in code (`111111`) so no real email is needed — **never use in production**.
- `--network host` lets the app reach MinIO on `localhost:9000` and exposes the app port directly.
- For a production self-hosted setup with Caddy, TLS, and external object storage, see [[en/user/selfhosted]].

### Mint an API key

To push memory notes and authenticate the MCP endpoint, you need an API key. With `DEV=true`:

```bash
GQL=http://localhost:24081/graphql

curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:RequestEmailSignInCodeInput!){requestEmailSignInCode(input:$i){__typename}}","variables":{"i":{"email":"hello@example.com"}}}' >/dev/null

TOKEN=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:SignInByEmailInput!){signInByEmail(input:$i){__typename ... on SignInPayload{token} ... on ErrorPayload{message}}}","variables":{"i":{"email":"hello@example.com","code":"111111"}}}' \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

API_KEY=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' -H "Cookie: trip2g_token=$TOKEN" \
  -d '{"query":"mutation($i:CreateApiKeyInput!){admin{createApiKey(input:$i){__typename ... on CreateApiKeyPayload{value} ... on ErrorPayload{message}}}}","variables":{"i":{"description":"local"}}}' \
  | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

echo "API key: $API_KEY"
```

## 2. Connect your agent (MCP)

trip2g exposes its MCP endpoint at `/_system/mcp`. The stdio adapter wraps the three retrieval steps — search, navigate TOC, read section — into **one tool** your agent calls.

The adapter script is `docs/en/user/trip2g_mcp_stdio_adapter.py` in the trip2g source repository (also served from the deployed docs site). Download it from there and place it at an absolute path on your machine.

Register it in your MCP client (Claude Desktop, Cursor, Claude Code, or any agent that speaks MCP over stdio):

```json
{
  "mcpServers": {
    "trip2g-memory": {
      "command": "python3",
      "args": ["/absolute/path/to/trip2g_mcp_stdio_adapter.py"],
      "env": {
        "TRIP2G_MCP_URL": "http://localhost:24081/_system/mcp",
        "TRIP2G_TOKEN": "t2g_your-personal-access-token"
      }
    }
  }
}
```

`TRIP2G_TOKEN` must be a **personal access token** (format `t2g_…`), not the admin API key from the mint step above. Create one under User → Tokens in the trip2g UI. See [[en/user/mcp]] ("Personal access tokens") for details.

The admin API key (minted above) is used for direct `X-API-Key` curl calls and for the sync CLI — not for the stdio adapter.

Note: the `expand` tool requires a recent image. Older local builds may return a flat `toc` structure instead of the navigable tree — use `ghcr.io/trip2g/trip2g:latest` to avoid this.

No `pip install` needed — the adapter uses Python 3 stdlib only.

### Available tools

The full MCP endpoint (used directly or through the adapter) exposes:

| Tool | What it does |
|------|--------------|
| `search(query)` | Vector or full-text search across all memory notes. Returns slim snippets — heading breadcrumb and `toc_path` per match, not the full note |
| `expand(pid, toc_path?)` | Walk a note's table of contents one level at a time. Returns direct children of a TOC node for drill-down |
| `note_html(path, toc_path?)` | Read a full note or a specific section identified by `toc_path` |
| `similar(path)` | Find notes similar to a given note |

The adapter wraps search → expand → note_html into a single composite tool that returns only the most relevant section. See [[en/user/ai-agent-mcp-adapter]] for the full adapter description.

### Authenticate with an API key

API keys are accepted directly on the MCP endpoint:

```bash
curl http://localhost:24081/_system/mcp \
  -H "X-API-Key: <your-api-key>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

API key auth gives admin-level access to all notes. For user-scoped access, use a personal token (`t2g_…`) from User → Tokens.

## 3. Sync your memory in

The sync CLI publishes a folder of Markdown notes into the running trip2g instance over the API. This is the memory-ingestion step: your notes become searchable knowledge the agent can recall.

The sync CLI (`obsidian-sync/dist/trip2g-sync.mjs`) is part of the trip2g source repository — it is not included in the Docker image. Run the following from a trip2g source checkout:

```bash
# from the root of a trip2g source checkout
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /path/to/your/vault \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/graphql \
  --verbose
```

Run this command again whenever you add or update notes. Only changed notes are re-uploaded.

A note is publicly accessible (reachable by the agent without authentication) only when it has `free: true` in frontmatter. For private memory that only your agent reads, omit `free: true` and always authenticate with the API key or token.

For the full sync CLI reference and options, see [[en/user/local-quickstart]].

## 3a. Continuous sync: `--watch` as a sidecar

The one-shot sync from section 3 uploads notes once. If your agent writes notes — and you want those writes visible to readers or to other agents immediately — run the sync CLI in watch mode as a long-running sidecar alongside the trip2g daemon.

```bash
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder /path/to/your/vault \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/_system/graphql
```

On startup it does a full two-way reconcile. After that:

- Any note your agent writes to disk is pushed to the server within ~500 ms (filesystem watcher + debounce).
- Any note changed on the server side (e.g. by another process or a human editor) is written back to the vault folder immediately via the `noteChanges` SSE subscription.

The process stays in the foreground and exits non-zero on a fatal error. In Docker Compose, add it as a second service that mounts the same vault volume as the agent container. Ctrl-C shuts it down cleanly.

To limit which paths the sync daemon follows from the server, pass `--include` and `--exclude` globs. See [[en/user/local-quickstart]] (section "Filtering with --include and --exclude") for the full reference.

### Watch the agent work in the browser

Once `--watch` is running, open any page on your site and append `?#!live_follow=1` to the URL:

```
http://localhost:24081/some-note?#!live_follow=1
```

The browser enters live-follow (cinema) mode and automatically navigates to whichever note changes next. The setting persists across the auto-navigations, so the browser keeps following until you disable it. This gives you a real-time view of what the agent is editing without any manual refreshing.

For full details on live-follow and the reload toggle, see [[en/user/live-editing]].

## 4. Recall: search → expand → note_html

Once notes are synced, the agent retrieves memory through the MCP tools in three steps:

```
1. search(query)
   → slim results: heading breadcrumb + toc_path per match

2. expand(pid=N)                     # survey top-level structure
   expand(pid=N, toc_path=[...])    # drill into the right branch
   → repeat until leaf (has_children: false)

3. note_html(pid=N, toc_path=[...])
   → read only the needed section
```

If `search` already returns an exact `toc_path` for the match, skip `expand` and call `note_html` directly with that path. The adapter handles this automatically.

### Why this keeps token cost low

Reading one section instead of a whole note is typically **~15× cheaper** on this site's own notes — and the answer lands at the top of the context window where model recall is strongest, not buried in the tail where it degrades. Numbers and a reproducible benchmark: [[en/user/token-economy-bench]].

The gain scales with note size. Long notes (architecture docs, changelogs, how-to guides) save the most. Short notes (a quick fact, a config snippet) save little — they're already cheap.

For a deeper explanation of the mechanism: [[Token Economy]].

## Related

- [[en/user/local-quickstart]] — full local setup reference, including `--watch` flag reference
- [[en/user/selfhosted]] — production Docker Compose setup with Caddy and TLS
- [[en/user/ai-agent-mcp-adapter]] — the stdio adapter: one tool, just the right section
- [[en/user/mcp]] — all MCP methods, access control, and named entry points
- [[en/user/expand]] — level-by-level TOC navigation explained
- [[en/user/token-economy-bench]] — measured token savings, reproducible benchmark
- [[en/user/live-editing]] — live-follow (cinema mode) and reload toggle for watching edits in real time
