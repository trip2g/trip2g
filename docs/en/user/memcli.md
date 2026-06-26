---
title: "memcli — one command to give an agent a local memory"
free: true
routes: [en/memcli]
lang_redirect: "[[ru/user/memcli]]"
---

One command boots a local trip2g server, mints an admin API key via HAT, and starts a two-way `trip2g-sync --watch` sidecar — your agent reads and writes plain Markdown files, and the server indexes them instantly. No S3, no email, no dev mode required.

## Run it

The compiled bundle ships in the repository. No build step needed:

```bash
node cli/memcli/dist/memcli.js up --folder ./memory-vault
```

**What `up` does:**

1. Creates `./memory-vault/.trip2g-memory/` (mode 700, state dir) and generates a random `JWT_SECRET` there — reused on subsequent runs, so `up` is idempotent.
2. Starts a Docker container named `trip2g-memory` with `STORAGE_BACKEND=local` — assets on disk, no S3 or MinIO.
3. Mints an admin API key via HAT (Hot Auth Token) — no email address, no `DEV=true` flag needed.
4. Writes the key to `.obsidian/plugins/trip2g/data.json` inside the vault.
5. Launches `trip2g-sync --watch` as a detached background process, syncing the vault and the server in both directions.
6. Prints: `memory live — web: http://localhost:24081  read/write .md in ./memory-vault`

Once up, your agent uses its normal Write/Read/Grep tools on the vault folder. No special memory API needed.

**Preview without side effects:**

```bash
node cli/memcli/dist/memcli.js up --folder ./memory-vault --dry-run
```

**Optional: rebuild from source** (only if you modify the CLI source):

```bash
cd cli/memcli && npm install && npm run codegen && npm run build
```

`codegen` reads the in-repo GraphQL schema — no running server needed. This regenerates `dist/memcli.js`.

## Subcommands

| Command | Effect |
|---------|--------|
| `up` | Boot server + sync watcher, mint API key (default subcommand) |
| `down` | Stop container and watcher |
| `status` | Show container and watcher status |
| `logs` | Snapshot of container logs (for diagnostics) |
| `key` | Rotate the admin API key — mints a new one and disables the previous |
| `daily "<text>"` | Append a thought to today's daily note |
| `log <file> "<text>"` | Append a thought under today's section in a named note |
| `hub <url>` | Bind an additional remote federation hub to the vault |
| `mcp` | Run as MCP stdio server (also auto-detected when stdin is piped) |

**Flags for `up`:**

| Flag | Default | Description |
|------|---------|-------------|
| `--folder <path>` | `./memory-vault` | Vault directory |
| `--port <n>` | `24081` | Public port |
| `--email <addr>` | `memory@local` | Owner email embedded in the HAT JWT |
| `--image <ref>` | `ghcr.io/trip2g/trip2g:latest` | Docker image |
| `--public-url <url>` | `http://localhost:<port>` | Override `PUBLIC_URL` |
| `--no-hub` | — | Skip writing the federation hub note |
| `--hub-url <url>` | `https://trip2g.com/_system/mcp` | Point the hub at a different MCP endpoint |
| `--name <id>` | — | Run a second isolated instance `trip2g-memory-<id>`; pass the same `--name` to `down`/`status`/`logs`, and give it a distinct `--port` |
| `--dry-run` | — | Print commands without executing |

To run two instances side-by-side, give the second one a `--name` and a distinct `--port`. State stays per-`--folder`, so each instance keeps its own vault:

```bash
node cli/memcli/dist/memcli.js up --folder ./vault-blog --name blog --port 24181
node cli/memcli/dist/memcli.js up --folder ./vault-work --name work --port 24281
# stop by name:
node cli/memcli/dist/memcli.js down --name blog
```

**Flags for `daily` / `log`:**

| Flag | Default | Description |
|------|---------|-------------|
| `--folder <path>` | `./memory-vault` | Vault directory |
| `--context <n>` | `15` | Lines of the note to echo back after the write |

## Capturing thoughts: daily vs log

These two commands are the primary interface for an agent to record what it is thinking or discovering mid-session.

### `memcli daily "<text>"`

Appends the text to the day's **daily note** at `daily/<YYYY-MM-DD>.md` — the general working space for that day's thoughts.

- The first entry of the day is written plain (no timestamp).
- Later entries on the same day get an `HH:MM` prefix on the first line.

Use `daily` for anything you want to capture now without a specific named topic: a passing observation, a decision, a reminder.

```bash
node cli/memcli/dist/memcli.js daily "deployment should always be off-peak — prod box is single VM"
```

### `memcli log <file> "<text>"`

Appends the text under a `### [[<YYYY-MM-DD>]]` section header in `<vault>/<file>.md` — an **append-only journal** of how one idea or topic evolves over days.

- The first entry under a new day header is plain.
- Later same-day entries get an `HH:MM` prefix.
- Never rewrite, only append. The history of when each thought appeared is preserved.

```bash
node cli/memcli/dist/memcli.js log work/deploy-rules "confirmed: never deploy during peak hours"
```

On the next day, a follow-up entry lands under a new `### [[2026-06-23]]` section in the same file, keeping the chronological record intact.

**Antipattern:** using `log` to rewrite or summarize the note loses the when-did-this-change information. Keep entries atomic. Edit the note directly with Write/Edit if you need to update structured content.

Both `daily` and `log` write directly to the vault folder — the running `trip2g-sync --watch` daemon picks up the change within ~500 ms and pushes it to the server.

## The trip2g.com hub

On first `up`, memcli writes a file `hub.md` into your vault with this frontmatter:

```yaml
mcp_federation_kb_url: https://trip2g.com/_system/mcp
```

This connects the local server's MCP endpoint to trip2g.com's published knowledge base via **MCP federation**. Any agent (or MCP client) that connects to `http://localhost:24081/_system/mcp` can also query trip2g.com's docs — including the memory documentation itself — without leaving the local memory scope. Federation, not a copy.

`hub.md` is a plain Markdown note. Deleting it removes the federation link on the next sync cycle.

- Pass `--no-hub` to skip hub creation entirely.
- Pass `--hub-url <url>` to federate to a different knowledge base instead.

```bash
node cli/memcli/dist/memcli.js up --folder ./memory-vault --hub-url https://example.com/_system/mcp
```

## Bind your own remote hub

You can federate your memory to **your own trip2g instance** (or any other) alongside the default trip2g.com hub. Multiple hubs coexist — each gets its own note file.

```bash
node cli/memcli/dist/memcli.js hub https://demo.lahab.cc/_system/mcp
```

This writes `hub-demo.lahab.cc.md` into the vault with `free: true` (required for the federation scan to recognize the note) and `mcp_federation_kb_url: https://demo.lahab.cc/_system/mcp`. After the running `--watch` daemon pushes the note to the server, `federated_search` also queries your hub.

Pass `--id` to use a shorter identifier:

```bash
node cli/memcli/dist/memcli.js hub https://demo.lahab.cc/_system/mcp --id my-team
```

Pass `--folder` to target a specific vault and `--dry-run` to preview without writing:

```bash
node cli/memcli/dist/memcli.js hub https://demo.lahab.cc/_system/mcp --folder ./memory-vault --dry-run
```

The `memory_bind_hub` MCP tool does the same thing from within an MCP session.

Notes:
- The `free: true` frontmatter field is handled automatically — do not remove it or the federation scan will ignore the note.
- Re-running `hub` with the same URL overwrites the note (updating the id or content).
- The default `hub.md` (trip2g.com) and additional `hub-<host>.md` files coexist independently.

## MCP mode

Run memcli with piped stdin (or pass `mcp` as the subcommand) and it becomes an MCP stdio server — JSON-RPC 2.0, hand-rolled, zero external dependencies:

```bash
# Explicit
node cli/memcli/dist/memcli.js mcp

# Auto-detected when stdin is piped (e.g. from an MCP client)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | node cli/memcli/dist/memcli.js
```

The MCP server exposes all commands as tools:

| Tool | What it does |
|------|--------------|
| `memory_up` | Boot the server (idempotent) |
| `memory_down` | Stop server and watcher |
| `memory_status` | Check running status |
| `memory_logs` | Snapshot of container logs |
| `memory_key` | Rotate the API key |
| `memory_daily` | Append to today's daily note |
| `memory_log` | Append to a named note's log section |

This lets an agent manage its own memory infrastructure as MCP tool calls — useful when the agent is connected to the local server's MCP endpoint and wants to also control it without a separate CLI session.

## Browser view

The URL printed by `up` (e.g. `http://localhost:24081`) opens the memory vault as a navigable website. Useful for a human observer to review what the agent has written.

Add `?#!live_follow=1` to any page URL and the browser enters cinema mode — it scrolls to whichever note the agent is currently editing in real time:

```
http://localhost:24081/?#!live_follow=1
```

## Related

- [[en/user/agent-memory]] — full agent memory setup: MCP connection, search → expand → note_html recall flow, token cost explained
- [[en/user/local-quickstart]] — sync CLI reference, `--watch` flag, include/exclude filters
