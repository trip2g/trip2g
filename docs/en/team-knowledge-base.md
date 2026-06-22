---
title: "Team knowledge base on a bare VM"
free: true
lang_redirect: "[[ru/team-knowledge-base]]"
---

Deploy trip2g on a single bare VM — no MinIO, no cloud storage — and your team gets a shared, searchable knowledge base that any AI agent can query through a single MCP endpoint. Notes stay private by default; federation search lets agents on other instances query your KB without exposing the whole vault.

## Deploy on a bare VM (runbook)

These steps are verified on a real Yandex Cloud deployment. Each step names the failure mode you avoid by following it.

### Step 1. Create the VM — size and network

Create a VM with **at least 8 GB RAM** (or configure swap before building the image). The Docker image build runs a Go compilation stage that silently uses 4–6 GB. On a 4 GB VM the OOM killer fires during the build — the SSH session becomes unresponsive and the VM looks dead.

- OS: Ubuntu 22.04
- Assign a **static (reserved) IP**. If you recreate or resize the VM its ephemeral IP changes; DNS and Cloudflare records break.
- Pass your SSH public key at VM create time via the metadata flag (`--metadata ssh-keys=ubuntu:<key>` on Yandex Cloud). On Yandex Cloud the default image enables OS Login, which ignores keys injected post-creation.
- Use a **passphrase-less SSH key**. Non-interactive steps (git archive pipe, scp inside scripts) cannot supply a passphrase.

### Step 2. Install Docker and build the image on the VM

Install Docker on the VM, then build the image **from source on the VM itself**. Do not push a locally built image from your laptop: if your machine is arm64 (Apple Silicon) and the VM is x86_64, the image will refuse to start with an `exec format error`.

Transfer source to the VM and build there:

```bash
# on your machine
git archive HEAD | ssh ubuntu@<vm-ip> 'mkdir -p ~/trip2g && tar x -C ~/trip2g'

# on the VM
cd ~/trip2g
docker build -t trip2g:local .
```

The build takes 5–10 minutes. The Go compilation stage produces no log output — this is normal. Do not kill it.

### Step 3. Run the server

Mount a data volume and start the container. Use `STORAGE_BACKEND=local` — no MinIO or S3 container needed.

**Generate secrets first:**

```bash
# DATA_ENCRYPTION_KEY — must be exactly 32 hex chars (16 bytes)
openssl rand -hex 16

# JWT_SECRET — any long random string
openssl rand -base64 32
```

**Run:**

```bash
docker run -d --name trip2g-kb \
  --restart unless-stopped \
  -p 80:80 \
  -e LISTEN_ADDR=0.0.0.0:80 \
  -e INTERNAL_LISTEN_ADDR=:8081 \
  -e DB_FILE=/data/kb.sqlite3 \
  -e STORAGE_BACKEND=local \
  -e STORAGE_LOCAL_DIR=/data/storage \
  -e OWNER_EMAIL=owner@yourteam.example \
  -e PUBLIC_URL=https://kb.yourteam.example \
  -e JWT_SECRET=<your-jwt-secret> \
  -e DATA_ENCRYPTION_KEY=<your-32-char-hex> \
  -v /opt/trip2g-kb:/data \
  trip2g:local
```

**Gotchas:**

- `DATA_ENCRYPTION_KEY` must be set to a non-default value. The server **panics** at startup with `"in production, data encryption key must be changed from default"` if you omit it or copy-paste a placeholder.
- `PUBLIC_URL` must be the final `https://` domain, not `http://localhost`. Auth flows and redirect URLs are derived from it.
- `LISTEN_ADDR=0.0.0.0:80` binds the public port directly. If you use a reverse proxy (Caddy/Nginx), bind to an internal port instead and proxy from there.
- Omit `DEV=true` in production. It disables security checks and enables fixed sign-in codes.
- Omit `RESEND_API_KEY` / `SMTP_PASSWORD` unless you need email sign-in.
- Omit `GIT_API_REPO_PATH` unless you use the built-in git mirror.

**Wait for readiness:**

```bash
until curl -sf http://localhost:8081/readyz >/dev/null; do sleep 2; done && echo "ready"
```

Use `/readyz` (not `/healthz`) — it waits for the database warmup to finish before returning 200.

**Required environment variables:**

| Variable | Purpose |
|----------|---------|
| `STORAGE_BACKEND=local` | Use local disk instead of S3/MinIO |
| `STORAGE_LOCAL_DIR=/data/storage` | Asset directory inside the mounted volume |
| `JWT_SECRET` | Signs session tokens. No default works in production |
| `DATA_ENCRYPTION_KEY` | 32-char hex key for encrypted fields. Omitting or using the default causes a panic on startup |
| `OWNER_EMAIL` | Admin account email |
| `PUBLIC_URL` | External URL for links and auth flows — set the final https:// domain at first run |
| `DB_FILE` | SQLite database path inside the container |
| `LISTEN_ADDR` | Main HTTP bind address |
| `INTERNAL_LISTEN_ADDR` | Internal address for health/readiness checks |

### Step 4. Point your domain and configure TLS

Point your domain at the VM's static IP via Cloudflare (proxied). The server listens on port 80; Cloudflare terminates TLS.

Set Cloudflare SSL mode to **Flexible** (Cloudflare → origin uses plain HTTP) or install a certificate on the VM and use **Full**.

`PUBLIC_URL` must equal the final `https://` domain. If you change the domain later you need to recreate the database or patch the stored URLs.

### Step 5. Get an admin API key (HAT flow, no DEV mode)

Without `DEV=true` there is no fixed sign-in code. Use HAT (Hot Auth Token) — a short-lived JWT you sign with `JWT_SECRET` and exchange for a session cookie.

```bash
# 1. Sign a JWT: payload {"e":"<owner-email>","ae":true,"exp":<now+300>}
#    Algorithm: HS256, secret: JWT_SECRET
#    Using PyJWT:
JWT=$(python3 -c "
import jwt, time, os
print(jwt.encode({'e':'owner@yourteam.example','ae':True,'exp':int(time.time())+300},
  os.environ['JWT_SECRET'], algorithm='HS256'))
")

# 2. Exchange for a session cookie
TOKEN=$(curl -s -c - -X POST https://kb.yourteam.example/_system/hat \
  -H "Authorization: Bearer $JWT" \
  | grep trip2g_token | awk '{print $NF}')

# 3. Create an API key
API_KEY=$(curl -s -X POST https://kb.yourteam.example/_system/graphql \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"mutation($i:CreateApiKeyInput!){admin{createApiKey(input:$i){__typename ... on CreateApiKeyPayload{value} ... on ErrorPayload{message}}}}","variables":{"i":{"description":"team-sync"}}}' \
  | python3 -c "import sys,json; d=json.load(sys.stdin); r=d['data']['admin']['createApiKey']; print(r['value'] if r['__typename']=='CreateApiKeyPayload' else r['message'])")

echo "API key: $API_KEY"
```

The `createApiKey` mutation returns a union — `CreateApiKeyPayload{value}` on success, `ErrorPayload{message}` on failure. Parse `__typename` before reading `value`.

See [[en/user/local-quickstart]] for the full HAT + key mint reference.

### Step 6. Push content with the sync CLI

```bash
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /path/to/team-vault \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/_system/graphql \
  --verbose
```

For live sync during editing:

```bash
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder /path/to/team-vault \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/_system/graphql
```

See [[en/user/local-quickstart]] for all sync CLI flags.

### Step 7. Set the root homepage

The site homepage at `/` must be a note named **`index.md`** or **`_index.md`** in the vault root. The `index`/`_index` filename segment is dropped, so its permalink becomes `/`.

A `route: kb.yourteam.example/` frontmatter field does **not** make a note the main-domain homepage. It routes the note to that path on a custom domain, but the root `/` slot is owned by the `index`/`_index` note.

### Step 8. Custom HTML pages — use a Jet layout

Raw HTML in a note body is sanitized by the Markdown pipeline. A block like:

```html
<div style="display:flex; height:100vh; ...">Keep out.</div>
```

becomes `<!-- raw HTML omitted -->` in the rendered page.

For a full custom page — such as a guest "keep out" landing — use a **Jet layout**: a server-side `.html` template that is not sanitized.

1. Create `_layouts/guard.html` in your vault — a complete `<!DOCTYPE html>` page:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Authorised access only</title>
  <style>
    body { margin: 0; display: flex; align-items: center; justify-content: center;
           height: 100vh; font-family: system-ui, sans-serif; }
    .msg { font-size: 3rem; font-weight: bold; text-align: center; }
    a { display: block; margin-top: 1rem; font-size: 1rem; }
  </style>
</head>
<body>
  <div class="msg">
    Keep out.<br>Authorised members only.
    <a href="/kb">Member sign-in</a>
  </div>
</body>
</html>
```

2. In your root `index.md`, reference the layout:

```yaml
---
free: true
layout: guard
---
```

The layout file renders on the server — the HTML is not sanitized. The `/kb` link points to any gated note: because it has no `free: true`, trip2g shows the built-in sign-in/paywall page when an anonymous visitor hits it. There is no standalone `/login` URL — the paywall page is the sign-in entry point.

### Step 9. Wire federation search

To let agents query another hub from your instance's `/_system/mcp`, add a KB-note to the vault:

```yaml
---
free: true
mcp_federation_kb_url: https://hub.example.com/_system/mcp
mcp_federation_kb_id: hub-name
---
Use when: searching shared team knowledge and public references.
```

**`free: true` is required.** Without it, the federation scan (`accessibleKBNotes`) ignores the note and `federated_search` returns "Federation is not configured." This is the most common misconfiguration.

After syncing, `/_system/mcp` exposes `federated_search`, `federated_similar`, `federated_note_html`, and `federated_expand`.

`memcli hub <url>` automates the KB-note creation and sync. See [[en/memcli]].

---

## Detailed reference

### Get an admin account and API key

Sign in as the owner, then mint one API key for the sync CLI. The team uses this single key to push content.

**Option A — HAT (Hot Auth Token) sign-in, no email required.**

HAT is the zero-email admin login. The server validates a short-lived JWT signed with `JWT_SECRET`. This is the approach used by memcli and other headless tools. The endpoint is `/_system/hat`. The full mint flow is in [[en/user/local-quickstart]] (section "Mint an API key") and [[en/user/agent-memory]].

**Option B — email sign-in (production).**

Set `RESEND_API_KEY`, `MAIL_FROM`, and a verified sender domain in Resend. The server emails a one-time code to `OWNER_EMAIL`. Once signed in, create the key in Admin → API Keys → Create.

### Push content with the sync CLI

Publish a vault folder to the server. Run from a trip2g source checkout:

```bash
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /path/to/team-vault \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/_system/graphql \
  --verbose
```

For the full sync CLI reference, see [[en/user/local-quickstart]].

### Wire federation search

Federation lets any AI agent call a single MCP endpoint on your KB and fan out across other connected knowledge bases, or let a peer hub query your KB's content. It requires a **KB-note** in the vault.

**Critical: `free: true` is required.**

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

**Example MCP call:**

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

**SSRF and depth.** Public hubs (external URLs) are allowed by default. For private/internal network addresses, the server requires `MCP_FEDERATION_ALLOW_PRIVATE=true`. Fan-out stops at depth 3 by default (`MCP_FEDERATION_MAX_DEPTH`). Per-peer timeout is 2 seconds (`MCP_FEDERATION_FANOUT_TIMEOUT`).

For the full federation setup — including private peers, HMAC key exchange, and subgraph scope — see [[en/user/federation]].

### Access control: who sees what

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

Another trip2g instance can federate your KB into its own MCP search. This uses HMAC key exchange (federation secrets), which scopes exactly which subgraphs the peer can see. See [[en/user/federation]] (section "Adding a private peer") for the two-step key exchange flow.

**Access summary:**

| Caller | Token | Sees |
|--------|-------|------|
| Anonymous agent / public hub | None | `free: true` notes only |
| Authenticated team member | `t2g_<token>` | Notes in the subscriber's scope |
| Admin API key | `X-API-Key: <key>` | All notes |
| Federated peer with HMAC secret | Signed JWT | Notes in the secret's subgraph scope |

## Related

- [[en/user/local-quickstart]] — full local setup reference, sync CLI flags, HAT mint flow
- [[en/user/agent-memory]] — single-agent memory setup; HAT mint flow detail
- [[en/memcli]] — automated server + API key + sync watcher in one command; `hub` subcommand for federation
- [[en/user/federation]] — full federation setup: public peers, private HMAC exchange, subgraph scope, federation graph
- [[en/user/selfhosted]] — Caddy + MinIO + TLS production setup (if you need email sign-in or external object storage)
- [[en/user/mcp]] — all MCP tools, authentication modes, personal access tokens
