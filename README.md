# trip2g

**Your Obsidian vault → website + Telegram channel + AI assistant. You own everything.**

trip2g is a self-hosted publishing platform for knowledge creators. Write in Obsidian, publish to a website with subscription paywalls, sync to Telegram channels, and let AI answer questions from your knowledge base.

## Get started

- **Cloud** — [simplecloud.2pub.me](https://simplecloud.2pub.me) (hosted, no setup)
- **Self-hosted** — run on your own server (docs coming soon)

## What it does

- **Publish from Obsidian** — push markdown notes to your site. Internal wikilinks work. One note = one page.
- **Paywall by subgraph** — mark groups of notes as paid. Free notes are public; paid notes require a subscription.
- **Telegram channel sync** — notes become Telegram posts on a schedule. Formatting preserved.
- **AI over your knowledge base** — vector search + MCP server. Readers connect their AI client and ask questions answered by your notes.
- **You own everything** — notes stay as plain markdown files on your computer. Switch platforms anytime.

## Features

### Publishing

- Markdown rendering with wikilinks, backlinks (`inlinks`), forward links (`outlinks`)
- Properties-based page layout: header, footer, sidebars, magazine grid — configured per-note via frontmatter
- Asset upload to S3/MinIO (images, files)
- Full-text search (bleve)
- Multi-domain routing — serve notes on custom domains via frontmatter ([docs](docs/multidomain.md))
- Multilingual content — automatic redirects, `hreflang` SEO tags, language switcher ([docs](docs/multilang.md))
- RSS feeds — any note is an RSS feed ([docs](docs/rss.md))
- Sitemap.xml — auto-generated from all published notes ([docs](docs/sitemap.md))

### Monetization

- Subgraph-based paywalls (group notes into paid products)
- Crypto payments via [NowPayments](https://nowpayments.io)
- Patreon and Boosty integration — grant access to existing subscribers

### Telegram

- Publish notes as channel posts
- Scheduled publishing with calendar view
- Edit/delete posts when notes are updated
- Publish via Telegram user accounts (MTProto) — long posts, custom emoji ([docs](docs/telegram_publish_through_accounts.md))
- Export channels to Markdown

### AI

- Vector search (semantic similarity)
- MCP (Model Context Protocol) server — connect Claude, Cursor, or any MCP-compatible AI client to your knowledge base
- Knowledge bot: Telegram bot that answers questions from your notes, tracks unanswered questions

### Obsidian Plugin

- One-click sync from Obsidian to your trip2g instance
- Push only changed files (hash-based diff)

### Webhooks & Automation

- **Change webhooks** — notify external agents when notes are created, updated, or removed. Agent receives a POST with changed content and can write notes back via API. Supports glob patterns, HMAC signing, recursion protection. See [docs/change_webhooks.md](docs/change_webhooks.md).
- **Cron webhooks** — call external agents on a schedule (e.g. `0 9 * * *`). Agent can generate digests, reports, or any content and push it back as notes. See [docs/cron_webhooks.md](docs/cron_webhooks.md).

### Admin

- User management, ban/unban
- Subgraph access control
- Asset browser

## Tech stack

| Layer | Tech |
|---|---|
| Backend | Go 1.26, FastHTTP, gqlgen (GraphQL) |
| Database | SQLite (default), PostgreSQL (optional) |
| Migrations | dbmate |
| Frontend | [mol.hyoo.ru](https://mol.hyoo.ru), TypeScript, Tiptap editor |
| Search | bleve (FTS), pgvector / SQLite-vec (semantic) |
| Assets | S3-compatible (MinIO for dev) |

## Obsidian plugin

Install the trip2g Obsidian plugin:

1. In Obsidian: Settings → Community plugins → Browse → search "trip2g"
2. Or manually: download from the [plugin repository](https://github.com/trip2g/obsidian-sync) and place in `.obsidian/plugins/trip2g/`
3. Configure: Settings → trip2g → enter your instance URL and API key

### Publishing a note

Add frontmatter to any note:

```yaml
---
subgraph: my-course      # which paid product this belongs to (omit for free)
free: true               # explicitly mark as free (public)
title: My Custom Title   # overrides filename in navigation
slug: my-url             # custom URL slug
---
```

Push from Obsidian: click the trip2g sync button. Only changed files are uploaded.

## Page layout

Configure page layout per-note using frontmatter properties:

```yaml
---
header: "[[Navigation]]"
left_sidebar:
  - TOC
  - inlinks
content:
  - magazine
  - selfcontent
footer: "[[Footer]]"
magazine_property: published_date
magazine_include_files: "blog/**/*.md"
---
```

See [docs/template-system.md](docs/template-system.md) for the full reference.

## MCP server

Connect your knowledge base to any MCP-compatible AI client:

```json
{
  "mcpServers": {
    "my-knowledge-base": {
      "url": "https://yourdomain.com/_system/mcp"
    }
  }
}
```

The MCP server exposes: `search_notes`, `get_note`, `list_notes`. Access is scoped to the user's subscription level.

## License

MIT
