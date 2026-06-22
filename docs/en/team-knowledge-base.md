---
title: "Team knowledge base on a bare VM"
free: true
lang_redirect: "[[ru/team-knowledge-base]]"
---

Deploy trip2g on a single bare VM — no MinIO, no cloud storage — and your team gets a shared, searchable knowledge base that any AI agent can query through a single MCP endpoint. Notes stay private by default; federation search lets agents on other instances query your KB without exposing the whole vault.

## 1. Deploy on a bare VM (local storage, no MinIO)

trip2g's local filesystem storage backend stores assets on disk. No S3 or MinIO container needed — one Docker container and one data volume.

**Generate secrets before starting:**

```bash
# 32-byte DATA_ENCRYPTION_KEY (required in production — server panics on default)
openssl rand -hex 16   # produces a 32-character hex string

# JWT_SECRET — any long random string
openssl rand -base64 32
```

**Run the server:**

```bash
docker run -d --name trip2g-kb \
  --restart unless-stopped \
  -p 8081:8081 \
  -e LISTEN_ADDR=0.0.0.0:8081 \
  -e INTERNAL_LISTEN_ADDR=:8082 \
  -e DB_FILE=/data/kb.sqlite3 \
  -e STORAGE_BACKEND=local \
  -e STORAGE_LOCAL_DIR=/data/storage \
  -e OWNER_EMAIL=owner@yourteam.example \
  -e PUBLIC_URL=https://kb.yourteam.example \
  -e JWT_SECRET=<your-long-random-string> \
  -e DATA_ENCRYPTION_KEY=<your-32-char-hex> \
  -v /opt/trip2g-kb:/data \
  ghcr.io/trip2g/trip2g:latest
```

Check it started:

```bash
docker ps | grep trip2g-kb
until curl -sf http://localhost:8082/healthz >/dev/null; do sleep 1; done; echo "up"
```

**Required environment variables:**

| Variable | Purpose |
|----------|---------|
| `STORAGE_BACKEND=local` | Use local disk instead of S3/MinIO |
| `STORAGE_LOCAL_DIR=/data/storage` | Asset directory inside the mounted volume |
| `JWT_SECRET` | Signs session tokens. Required; no default works in production |
| `DATA_ENCRYPTION_KEY` | 32-byte hex key for encrypted fields. Must differ from the default — the server panics with "in production, data encryption key must be changed from default" |
| `OWNER_EMAIL` | The admin account email address |
| `PUBLIC_URL` | External URL used in links and auth flows |
| `DB_FILE` | SQLite database path inside the container |
| `LISTEN_ADDR` | Main HTTP bind address |
| `INTERNAL_LISTEN_ADDR` | Internal address for health checks |

**Omit:**
- `DEV=true` — dev mode only; disables production security
- `RESEND_API_KEY` / `SMTP_PASSWORD` — only needed for email sign-in codes
- `GIT_API_REPO_PATH` — only needed if you use the built-in git mirror

**Reverse proxy.** Put Caddy or Nginx in front to handle TLS:

```caddyfile
kb.yourteam.example {
    encode zstd gzip
    reverse_proxy localhost:8081
}
```

**Systemd alternative.** If you prefer running the binary directly (build with `make build`):

```ini
[Unit]
Description=trip2g team KB
After=network.target

[Service]
EnvironmentFile=/etc/trip2g-kb.env
ExecStart=/usr/local/bin/trip2g
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Important: the image must include the local storage backend. This feature ships as of `feat/filestorage` — use `ghcr.io/trip2g/trip2g:latest` or any release after that.

## 2. Get an admin account and API key

Sign in as the owner, then mint one API key for the sync CLI. The team uses this single key to push content.

**Option A — HAT (Hot Auth Token) sign-in, no email required.**

HAT is the zero-email admin login. The server signs a JWT with the owner email and the `JWT_SECRET`. This is the approach used by memcli and other headless tools. The exact mutation is `/_system/hat`. See [[en/user/agent-memory]] (section "Mint an API key") for the curl flow.

**Option B — email sign-in (production).**

Set `RESEND_API_KEY`, `MAIL_FROM`, and a verified sender domain in Resend. The server emails a one-time code to `OWNER_EMAIL`. Once signed in, create the key in Admin → API Keys → Create.

**Mint the key via curl** (using a session token from either sign-in flow above):

```bash
API_KEY=$(curl -s -X POST https://kb.yourteam.example/graphql \
  -H 'Content-Type: application/json' \
  -H "Cookie: trip2g_token=$TOKEN" \
  -d '{"query":"mutation($i:CreateApiKeyInput!){admin{createApiKey(input:$i){__typename ... on CreateApiKeyPayload{value} ... on ErrorPayload{message}}}}","variables":{"i":{"description":"team-sync"}}}' \
  | grep -o '"value":"[^"]*"' | cut -d'"' -f4)
echo "API key: $API_KEY"
```

For the full mint flow, see [[en/user/local-quickstart]] (section "Mint an API key").

## 3. Push content with the sync CLI

Publish a vault folder to the server. Run from a trip2g source checkout:

```bash
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /path/to/team-vault \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/graphql \
  --verbose
```

For continuous sync (notes update live as you edit them in Obsidian):

```bash
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder /path/to/team-vault \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/_system/graphql
```

For the full sync CLI reference, see [[en/user/local-quickstart]].

## 4. Wire federation search

Federation lets any AI agent call a single MCP endpoint on your KB and fan out across other connected knowledge bases, or let a peer hub query your KB's content. It requires a **KB-note** in the vault.

### Critical: `free: true` is required

A KB-note without `free: true` is invisible to unauthenticated MCP callers. Without it, the federation scan (`accessibleKBNotes`) ignores the note and federated tools return "Federation is not configured."

Create a file in your vault (e.g. `hub/peer-name.md`):

```yaml
---
free: true
mcp_federation_kb_url: https://hub.example.com/_system/mcp
mcp_federation_kb_id: hub-name
---
Use when: searching shared team knowledge and public references.
```

Sync the vault. The local `/_system/mcp` now exposes `federated_search`, `federated_similar`, `federated_note_html`, and `federated_expand`, which query the peer hub.

### Example MCP call

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "federated_search",
    "arguments": {
      "query": "deployment checklist",
      "kb_id": "hub-name"
    }
  }
}
```

Returns matching notes from the peer. Without `kb_id`, the call fans out across all registered KB-notes in parallel.

### SSRF and depth

Public hubs (external URLs) are allowed by default. For private/internal network addresses, the server requires `MCP_FEDERATION_ALLOW_PRIVATE=true`. Fan-out stops at depth 3 by default (`MCP_FEDERATION_MAX_DEPTH`). Per-peer timeout is 2 seconds (`MCP_FEDERATION_FANOUT_TIMEOUT`).

For the full federation setup — including private peers, HMAC key exchange, and subgraph scope — see [[en/user/federation]].

## 5. Access control: who sees what

`/_system/mcp` is publicly accessible for notes marked `free: true`. Everything else requires authentication.

**For private or subscriber-only content**, callers authenticate with a Bearer token:

```
Authorization: Bearer t2g_<token>
```

or as a URL query parameter:

```
https://kb.yourteam.example/_system/mcp?token=t2g_...
```

The token format is `t2g_<...>`. Create personal access tokens under User → Tokens in the trip2g admin. Members who need to query private notes get their own token.

**Connecting Claude Code or another MCP client:**

```json
{
  "mcpServers": {
    "team-kb": {
      "command": "python3",
      "args": ["/path/to/trip2g_mcp_stdio_adapter.py"],
      "env": {
        "TRIP2G_MCP_URL": "https://kb.yourteam.example/_system/mcp",
        "TRIP2G_TOKEN": "t2g_member-token-here"
      }
    }
  }
}
```

**Federating a private KB into another hub.**

Another trip2g instance can federate your KB into its own MCP search. This uses HMAC key exchange (federation secrets), which scopes exactly which subgraphs the peer can see. See [[en/user/federation]] (section "Adding a private peer") for the two-step key exchange flow. The technical recipe is in `docs/dev/federation_agent_setup.md`.

**Access summary:**

| Caller | Token | Sees |
|--------|-------|------|
| Anonymous agent / public hub | None | `free: true` notes only |
| Authenticated team member | `t2g_<token>` | Notes in the subscriber's scope |
| Admin API key | `X-API-Key: <key>` | All notes |
| Federated peer with HMAC secret | Signed JWT | Notes in the secret's subgraph scope |

## 6. Guest homepage: full-screen warning

You can show anonymous visitors a full-screen warning while authenticated members see the real KB. The mechanism uses trip2g's access control and the `route` frontmatter field.

Create an index note (e.g. `_home.md`) and route it to the root of your domain:

```yaml
---
route: kb.yourteam.example/
free: true
---
```

The note body contains a full-viewport HTML block centered with flexbox:

```html
<div style="
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  font-size: 3rem;
  font-weight: bold;
  text-align: center;
  font-family: system-ui, sans-serif;
">
  Keep out.<br>Authorised members only.
</div>
```

This note has `free: true`, so the warning is visible to anyone who hits the root URL unauthenticated. When a member signs in, they navigate to the real KB content through the sidebar or direct links.

For the Russian equivalent: substitute the warning text (e.g. "Не лезь — убьёт") and the same markup applies.

**Uncertainty note.** The exact behavior when a note has `route: domain/` and `free: true` is documented in `docs/dev/multidomain.md` and `docs/dev/default_template.md`. If the homepage note should only show the warning to anonymous users while automatically redirecting authenticated users to a different page, that redirect logic is not built into the default template and would require a custom Jet layout. Verify the desired UX before building it — the approach above (static warning page, members navigate manually) is the simplest and confirmed to work.

## Related

- [[en/user/local-quickstart]] — full local setup reference, sync CLI flags
- [[en/user/agent-memory]] — single-agent memory setup; describes HAT mint flow
- [[en/memcli]] — automated server + API key + sync watcher in one command
- [[en/user/federation]] — full federation setup: public peers, private HMAC exchange, subgraph scope, federation graph
- [[en/user/selfhosted]] — Caddy + MinIO + TLS production setup (if you need email sign-in or external object storage)
- [[en/user/mcp]] — all MCP tools, authentication modes, personal access tokens
