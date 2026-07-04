---
title: "Anatomy of trip2g: 15 Months of Code"
free: true
lang_redirect: "[[ru/thoughts/anatomy-15-months]]"
---

trip2g is a publishing platform: Obsidian vaults → websites with subscriptions and Telegram integration. Go monolith + $mol frontend + SQLite.

After 15 months I counted what's actually inside.

## Own code

| Layer | Lines |
|---|---|
| Go (production) | ~32 000 |
| Go (tests) | ~81 000 |
| TypeScript ($mol UI) | ~41 000 |
| $mol templates (.view.tree) | ~5 000 |
| SQL migrations | ~2 500 (120 files) |
| SQL queries | ~2 600 (read: 1 476, write: 1 159) |
| GraphQL schema | 3 400 |
| obsidian-sync plugin | ~9 700 |
| **Total** | **~177 000** |

Not counted: Go codegen (gqlgen, sqlc, moq) — 139 000 lines in 134 files. Vendor bundles (Tiptap, etc.) — 190 000 lines.

## 1 615 commits in 15 months

## Codegen is 4× bigger than hand-written Go

SQL queries and GraphQL schema are written by hand. Everything else — resolvers, DB access layer, mocks — is generated. This is the point of using gqlgen + sqlc: you describe intent, the tools produce the boilerplate.

## Admin panel: 38 pages

The admin panel has 7 sections with 38 pages in total:

| Section | Pages |
|---|---|
| system | 9 |
| content | 7 |
| users | 5 |
| integrations | 5 |
| monetization | 4 |
| seo | 4 |
| telegram | 3 |

174 $mol components in the admin alone.

## E2E as the main reliability layer

Unit tests cover isolated logic. The main safety net is a Playwright E2E suite: 29 spec files, ~5 400 lines. Each run spins up 4 isolated app instances (main + 3 peers for federation tests), MinIO, and an embedding service against a clean SQLite database.

Test order is not arbitrary: setup → CLI sync → peer sync → main browser tests → CSS hot-reload → federation → webhooks. Webhooks run last intentionally — they wait for an empty job queue.

Telegram tests are a separate opt-in mode (`ENABLE_TG=1`) that uses real channels and checks message snapshots before and after edits.

## Cron, queues, and background jobs

Everything runs in one binary — no separate workers, no Redis, no RabbitMQ.

**Cron** is built on `robfig/cron`. Jobs are defined in code and also stored in the database, so you can enable/disable them from the admin panel without a redeploy. Typical jobs: sync Patreon/Boosty subscribers, send scheduled Telegram posts, clean up expired tokens.

**Background job queue** runs on `goqite` — a job queue backed by SQLite. Short-lived jobs (send one Telegram message, process one webhook) go here. The queue survives restarts because it's just a table. Two queue backends run in parallel: `goqite` for short jobs and `backlite` for longer ones.

**Telegram bot orchestration** handles incoming updates from multiple bots simultaneously. Each bot has its own handler chain: message routing, state machine for multi-step dialogs (attach code flow, invite flow), subgraph access management. Bot state is persisted in SQLite so it survives restarts.

**Telegram account orchestration** uses `gotd/td` (pure-Go TDLib) to publish through real Telegram user accounts — not bots. This bypasses bot limitations on media types and custom emoji. Each account runs its own MTProto session. The orchestrator manages session lifecycle, reconnects, and flood-wait backoff across multiple accounts publishing to multiple channels.

## Notable Go dependencies

**Content pipeline**
- [goldmark](https://github.com/yuin/goldmark) — Markdown parser with a full AST. Used everywhere: note rendering, Telegram post formatting, search indexing, wikilink resolution. The AST lets you walk and transform the document tree rather than regex-patching HTML.
- [CloudyKit/jet](https://github.com/CloudyKit/jet) — a template engine with its own AST and expression evaluator. Powers JSON layouts and template views. More capable than `html/template`: variables, functions, inheritance, direct AST access. See also the [[ru/user/jet|Jet docs (Russian)]].
- `valyala/quicktemplate` — compiled templates for the default site theme. Templates become Go code at build time — no reflection, no parsing at runtime.
- `google/go-jsonnet` — Jsonnet for site config. Supports imports, functions, and computed values, which plain JSON/TOML can't do.

**Search**
- [blevesearch/bleve](https://github.com/blevesearch/bleve) — full-text search embedded in the binary. No Elasticsearch, no external process. Also handles vector search for "similar notes" via OpenAI embeddings.

**Storage**
- `minio/minio-go` — S3-compatible storage for uploaded note assets and automated backups.
- `modernc.org/sqlite` — the pure-Go SQLite driver, used everywhere including production. The CGO driver `mattn/go-sqlite3` survives only as an indirect dependency of other packages.
- `maragu/goqite` — job queue on top of SQLite. One less external dependency.

**Transport**
- `gotd/td` — pure-Go TDLib implementation (no CGO). Powers Telegram userbot publishing.
- `valyala/fasthttp` — used specifically for GraphQL SSE subscriptions where `net/http` streaming has limitations.

**Utilities**
- `oklog/ulid` — sortable IDs (vs UUID). Insertion order is preserved in indexes.
- `bradleyjkemp/cupaloy` — snapshot tests. Useful for testing rendered HTML output.

## 70+ packages, 50+ use cases

The `internal/` directory is organized by domain. Each use case (`internal/case/`) is one operation: one file, one function, one `Env` interface with only the dependencies it needs.

Auth alone has 7 token types: email code, HAT, Telegram, purchase, personal, short API, user token — each isolated in its own package.

Monetization covers Patreon, Boosty, and NOWPayments (crypto). Integrations: GitHub import, Notion import, OpenAI embeddings, federation between trip2g instances, webhooks with agent response parsing.

There are also three transliteration packages for Ru→Latin URL slugs. Some problems have layers.

## Bug found while building this page

Building this very performance report exposed a bug in the layout engine's `asset()` function.

`asset()` in Jet templates returns presigned MinIO URLs for images. These URLs contain `&` separating query parameters. It turns out `jet.NewSet()` in Jet v6 defaults to `template.HTMLEscape` as the output escaper — not obvious from the name, which sounds like the "non-HTML" variant.

Result: `&` in the presigned URL became `&amp;` inside `url()` in CSS inside a `<style>` tag. HTML entity decoding doesn't happen inside `<style>` blocks, so the browser sent the literal string `&amp;X-Amz-Credential=...` to MinIO. This broke the AWS signature verification. MinIO responded: `The authorization mechanism you have provided is not supported. Please use AWS4-HMAC-SHA256.`

The fix: pass `jet.WithSafeWriter(nil)` when creating the Jet set for layouts. Layouts contain trusted, pre-rendered content authored by the site owner — not user input — so HTML escaping is neither necessary nor safe.

## Why this report

Fourteen months of full-time work in ideal conditions. Custom language, custom frontend framework, custom template engine. A federation protocol. RAG. Telegram userbot orchestration. Seven auth token types.

And the inner voice saying "make it even better" still isn't satisfied. That's it — time to change strategy. Can't force this out through brute strength anymore. There's nowhere left to push.

What actually got built: a knowledge base runtime. trip2g turns an Obsidian vault into a website, a landing page, a wiki, a RAG endpoint, an MCP server — accessible on the public internet or inside a private network. The federation protocol lets you build domain-specific knowledge bases and navigate between them: your own nodes, friends, company, public hubs.

The current bet is that the value is in the agent layer. An agent that can maintain an Obsidian vault on one side and execute learned workflows on the other — trip2g is the persistent memory that makes that agent coherent across sessions. The knowledge base is the agent's second brain: it stores what the agent knows about itself, its operator, and its work.

Thanks to [Nikolay Senin](https://journal.nicksenin.com/code-with-claude) for examples of trip2g sites in production. [meditation.2pub.ru](https://meditation.2pub.ru) — an autonomous Hermes instance that meditates, runs a site and Telegram channel, generates music tracks for the main page, and is preparing to publish to YouTube.
