---
title: "Long-term memory for AI agents with trip2g"
free: true
lang_redirect: "[[ru/user/agent-memory]]"
---

Self-host trip2g in Docker, point your agent's MCP client at it, sync your notes in, and your agent reads and searches them as persistent long-term memory. Focused section reads keep token cost low, typically ~15× cheaper than dumping the whole note.

Paste this into your coding agent (Claude Code, Cursor, or similar) to set up trip2g as its long-term memory:

```text
Follow https://trip2g.com/en/user/agent-memory to set up trip2g as your persistent memory.

Step 1: clone the repo (memcli ships prebuilt, no separate install needed):
  git clone https://github.com/trip2g/trip2g

Step 2: boot the local memory server:
  node trip2g/cli/memcli/dist/memcli.js up --folder ./memory-vault

  This starts a trip2g instance on http://localhost:24081, mints an admin key
  via HAT, launches the trip2g-sync --watch sidecar (any .md file you write to
  ./memory-vault is pushed to the server within ~500 ms), and writes a hub.md
  federation note so you can also search trip2g.com's knowledge base.

Step 3: register the MCP server in your agent's MCP client config:
  {
    "mcpServers": {
      "trip2g-memory": {
        "command": "python3",
        "args": ["/absolute/path/to/trip2g/docs/en/user/trip2g_mcp_stdio_adapter.py"],
        "env": {
          "TRIP2G_MCP_URL": "http://localhost:24081/_system/mcp",
          "TRIP2G_TOKEN": "t2g_your-personal-access-token"
        }
      }
    }
  }
  TRIP2G_TOKEN is a personal access token (format t2g_…). Create one under
  User → Tokens in the trip2g UI at http://localhost:24081. The adapter uses
  Python 3 stdlib only; no pip install needed.

Step 4: verify recall is working:
  Call search("trip2g memory") via the MCP tool, then call note_html() on one
  of the returned paths and show a result.

Once set up, follow these rules every session:
- REMEMBER: write durable facts as Markdown files into ./memory-vault. The
  watcher syncs them in ~500 ms, making them searchable immediately.
- RECALL: always use search → expand → note_html. Read only the section you
  need. It is ~15× cheaper than loading the whole note.
```

**Two credential types (quick orientation):**
- **Admin API key**: used by the sync CLI and direct `X-API-Key` curl calls. Minted automatically by memcli, or manually via the GraphQL mutations in the manual-setup section below.
- **Personal token (`t2g_…`)**: used by the stdio MCP adapter. Create one under User → Tokens in the trip2g UI after the server is up.

These are different things. The confusion usually shows up when configuring the MCP adapter. See [[en/user/mcp]] if you hit auth errors there.

## 1. Start in 30 seconds with memcli

**memcli** is the fastest path. One command boots the server with local storage, mints an admin key via HAT (no email, no `DEV=true`), starts the `trip2g-sync --watch` sidecar, and writes a `hub.md` federation note into your vault.

The compiled bundle ships in the repository, so no build step is needed from a fresh checkout:

```bash
node cli/memcli/dist/memcli.js up --folder ./memory-vault
```

That's it. When it finishes you'll see:

```
memory live — web: http://localhost:24081  read/write .md in ./memory-vault
```

The server is on port 24081. The MCP endpoint is at `http://localhost:24081/_system/mcp`. The sync watcher runs in the background; any `.md` file you drop into `./memory-vault` is pushed to the server within ~500 ms.

memcli generates random secrets (`JWT_SECRET`, `DATA_ENCRYPTION_KEY`) on first run and stores them in `./memory-vault/.trip2g-memory/env`. Subsequent `up` calls reuse the same secrets and are idempotent.

**Other memcli commands:**

```bash
node cli/memcli/dist/memcli.js status          # container + watcher status
node cli/memcli/dist/memcli.js logs            # server logs
node cli/memcli/dist/memcli.js down            # stop everything
node cli/memcli/dist/memcli.js key             # rotate the API key
```

**Optional flags for `up`:**

| Flag | Default | Purpose |
|------|---------|---------|
| `--folder <path>` | `./memory-vault` | Vault directory |
| `--port <n>` | `24081` | Public port |
| `--no-hub` | (none) | Skip writing `hub.md` |
| `--hub-url <url>` | `https://trip2g.com/_system/mcp` | Override federation hub |
| `--image <ref>` | `ghcr.io/trip2g/trip2g:latest` | Docker image |

**Rebuild from source** (only if you modify the CLI source):

```bash
cd cli/memcli && npm install && npm run codegen && npm run build
```

`codegen` reads the in-repo schema; no running server is needed.

### Federation hub note

`memcli up` writes `hub.md` into your vault by default. Its frontmatter:

```yaml
mcp_federation_kb_url: https://trip2g.com/_system/mcp
```

This federates your local instance to trip2g.com: anything connecting to your local `/_system/mcp` can also query trip2g.com's published knowledge base through your own instance. It is a live federation, not a copy.

Pass `--no-hub` to skip the note. Pass `--hub-url <url>` to point at a different hub:

```bash
node cli/memcli/dist/memcli.js up --folder ./memory-vault --hub-url https://example.com/_system/mcp
```

`hub.md` is a plain Markdown note. Deleting it removes the federation link on the next sync.

## 2. Manual setup (without memcli, or for an existing compose stack)

Use this path when you need fine-grained control over the container, or when integrating with an existing Docker Compose stack.

### Run the daemon

```bash
# trip2g on port 24081, healthcheck on 24082, local asset storage on disk
# Replace ghcr.io/trip2g/trip2g:latest with trip2g:local if you built from source.
mkdir -p /tmp/trip2g-local
docker run -d --name trip2g-local \
  -p 24081:24081 -p 24082:24082 \
  -e LISTEN_ADDR=0.0.0.0:24081 \
  -e INTERNAL_LISTEN_ADDR=0.0.0.0:24082 \
  -e DB_FILE=/data/local.sqlite3 \
  -e STORAGE_BACKEND=local \
  -e STORAGE_LOCAL_DIR=/data/storage \
  -e DEV=true \
  -e OWNER_EMAIL=hello@example.com \
  -e PUBLIC_URL=http://localhost:24081 \
  -e JWT_SECRET=dev-secret-not-for-prod \
  -e USER_TOKEN_INSECURE=true \
  -e GIT_API_REPO_PATH=/data/git \
  -e GIT_API_BASE_PATH=/git \
  -e RESEND_API_KEY=dev \
  -e MAIL_FROM=dev@example.com \
  -v /tmp/trip2g-local:/data \
  ghcr.io/trip2g/trip2g:latest

# verify the container started, then wait until healthy
docker ps | grep trip2g-local
until curl -sf http://localhost:24082/healthz >/dev/null; do sleep 1; done && echo "up"
```

Env vars at a glance:

```
LISTEN_ADDR / INTERNAL_LISTEN_ADDR   # app port + healthcheck port (inside container)
DB_FILE / STORAGE_BACKEND            # SQLite path + asset backend (local = disk, no S3)
STORAGE_LOCAL_DIR                    # where assets land inside the data volume
DEV=true                             # fixed sign-in code 111111 (do not use in production)
OWNER_EMAIL                          # email for the first admin account (needed with DEV=true)
PUBLIC_URL                           # the URL the app thinks it's served from
JWT_SECRET                           # HMAC secret; use a random value in production
USER_TOKEN_INSECURE                  # allows plain-HTTP tokens (fine for localhost)
GIT_API_REPO_PATH / GIT_API_BASE_PATH  # git mirror for note history
RESEND_API_KEY / MAIL_FROM           # email transport (stub values fine for DEV mode)
```

Note on `-p` vs `--network host`: `-p 24081:24081 -p 24082:24082` publishes ports on all host interfaces and works on Linux, Mac, and Windows (Docker Desktop). `--network host` is Linux-only. Docker Desktop runs the engine in a Linux VM, so host networking does not publish ports to your Mac or Windows machine and `localhost:24081` won't reach the container.

For a production setup with Caddy, TLS, and S3-compatible storage, see [[en/user/selfhosted]].

### Mint an API key

With `DEV=true`, the fixed sign-in code is `111111`. Use these three curl calls to get an API key:

```bash
GQL=http://localhost:24081/graphql

# 1. request a sign-in code (sent to the dev log, not real email)
curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:RequestEmailSignInCodeInput!){requestEmailSignInCode(input:$i){__typename}}","variables":{"i":{"email":"hello@example.com"}}}' >/dev/null

# 2. sign in with code 111111, capture the session token
TOKEN=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:SignInByEmailInput!){signInByEmail(input:$i){__typename ... on SignInPayload{token} ... on ErrorPayload{message}}}","variables":{"i":{"email":"hello@example.com","code":"111111"}}}' \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

# 3. create an API key
API_KEY=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' -H "Cookie: trip2g_token=$TOKEN" \
  -d '{"query":"mutation($i:CreateApiKeyInput!){admin{createApiKey(input:$i){__typename ... on CreateApiKeyPayload{value} ... on ErrorPayload{message}}}}","variables":{"i":{"description":"local"}}}' \
  | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

echo "API key: $API_KEY"
```

## 3. Connect your agent (MCP)

trip2g exposes its MCP endpoint at `/_system/mcp`. The stdio adapter wraps the three retrieval steps (search, navigate TOC, read section) into **one tool** your agent calls.

The adapter script is `docs/en/user/trip2g_mcp_stdio_adapter.py` in the trip2g source repository (also served from the deployed docs site). Download it and place it at an absolute path on your machine.

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

`TRIP2G_TOKEN` must be a **personal access token** (format `t2g_…`), not the admin API key. Create one under User → Tokens in the trip2g UI. See [[en/user/mcp]] ("Personal access tokens") for details.

The admin API key (from memcli or the manual mint above) is used for direct `X-API-Key` curl calls and for the sync CLI, not for the stdio adapter.

Note: the `expand` tool requires a recent image. Older local builds may return a flat `toc` structure instead of the navigable tree; use `ghcr.io/trip2g/trip2g:latest`.

The adapter uses Python 3 stdlib only; no `pip install` is needed.

### Available tools

The full MCP endpoint (used directly or through the adapter) exposes:

| Tool | What it does |
|------|--------------|
| `search(query)` | Vector or full-text search across all memory notes. Returns slim snippets: heading breadcrumb and `toc_path` per match, not the full note |
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

## 4. Sync your memory in

The sync CLI publishes a folder of Markdown notes into the running trip2g instance over the API. This is the memory-ingestion step: your notes become searchable knowledge the agent can recall.

The sync CLI (`obsidian-sync/dist/trip2g-sync.mjs`) is part of the trip2g source repository; it is not included in the Docker image. Run the following from a trip2g source checkout:

```bash
# from the root of a trip2g source checkout
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /path/to/your/vault \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/graphql \
  --verbose
```

Run this again whenever you add or update notes. Only changed notes are re-uploaded.

A note is publicly accessible (reachable by the agent without authentication) only when it has `free: true` in frontmatter. For private memory that only your agent reads, omit `free: true` and always authenticate with the API key or token.

For the full sync CLI reference, see [[en/user/local-quickstart]].

If you used memcli, the `--watch` sidecar is already running, so you can skip the one-shot sync and write `.md` files directly to the vault folder.

## 4a. Continuous sync: `--watch` as a sidecar

The one-shot sync from section 4 uploads notes once. If your agent writes notes and you want those writes visible immediately, run the sync CLI in watch mode as a long-running sidecar alongside the trip2g daemon.

```bash
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder /path/to/your/vault \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/_system/graphql
```

On startup it does a full two-way reconcile. After that:

- Any note your agent writes to disk is pushed to the server within ~500 ms (filesystem watcher + debounce).
- Any note changed on the server side is written back to the vault folder immediately via the `noteChanges` SSE subscription.

The process stays in the foreground and exits non-zero on a fatal error. In Docker Compose, add it as a second service that mounts the same vault volume as the agent container. Ctrl-C shuts it down cleanly.

To limit which paths the sync daemon follows from the server, pass `--include` and `--exclude` globs. See [[en/user/local-quickstart]] (section "Filtering with --include and --exclude") for the full reference.

### Watch the agent work in the browser

Once `--watch` is running, open any page on your site and append `?#!live_follow=1` to the URL:

```
http://localhost:24081/some-note?#!live_follow=1
```

The browser enters live-follow (cinema) mode and automatically navigates to whichever note changes next. The setting persists across the auto-navigations, so the browser keeps following until you disable it.

For full details on live-follow and the reload toggle, see [[en/user/live-editing]].

## 5. Recall: search → expand → note_html

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

Reading one section instead of a whole note is typically **~15× cheaper** on this site's own notes. The answer also lands at the top of the context window where model recall is strongest, not buried in the tail where it degrades. Numbers and a reproducible benchmark: [[en/user/token-economy-bench]].

The gain scales with note size. Long notes (architecture docs, changelogs, how-to guides) save the most. Short notes (a quick fact, a config snippet) save little, because they are already cheap.

For a deeper explanation of the mechanism: [[Token Economy]].

## Related

- [[en/user/local-quickstart]]: full local setup reference, including `--watch` flag reference
- [[en/user/selfhosted]]: production Docker Compose setup with Caddy and TLS
- [[en/user/ai-agent-mcp-adapter]]: the stdio adapter: one tool, just the right section
- [[en/user/mcp]]: all MCP methods, access control, and named entry points
- [[en/user/expand]]: level-by-level TOC navigation explained
- [[en/user/token-economy-bench]]: measured token savings, reproducible benchmark
- [[en/user/live-editing]]: live-follow (cinema mode) and reload toggle for watching edits in real time
- [[en/user/agent-status]]: auto-broadcast your status to teammates on session end
