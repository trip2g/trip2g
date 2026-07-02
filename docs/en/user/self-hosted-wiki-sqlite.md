---
title: "A self-hosted wiki on SQLite (one binary)"
description: "Run a self-hosted markdown wiki with no database server. trip2g is a single Go binary on SQLite, compared honestly with Wiki.js, BookStack, DokuWiki, and Outline."
free: true
home_position: 38
lang_redirect: "[[ru/user/self-hosted-wiki-sqlite]]"
---

If you want a self-hosted wiki without standing up Postgres and Redis first, trip2g runs as one Go binary on a single SQLite file. You author in Obsidian, sync, and the notes are served as a wiki with full-text search, access control, and a built-in MCP endpoint. This page surveys the common self-hosted wikis honestly, then walks the one-binary setup.

### Why the stack weight matters

The friction with most self-hosted wikis is not the wiki, it is everything it drags in. Wiki.js wants Postgres. Outline wants Postgres plus Redis plus S3. Even a small team wiki can turn into a multi-container deployment with its own backup and upgrade story per service. On a small VPS that is real overhead, and every added service is another thing that can break at 2am.

SQLite changes that calculus. The whole database is one file. Backing up the wiki is copying that file (or streaming it with Litestream). There is no separate database daemon to run, patch, or tune. For a personal or small-team wiki, that is usually the right amount of machinery.

### The options, honestly

| Wiki | Stack | Database | Editor | Best for |
|---|---|---|---|---|
| DokuWiki | PHP | flat files, no DB | wiki markup | tiny setups, minimal deps |
| BookStack | PHP | MySQL/MariaDB | WYSIWYG | teams wanting a book/chapter structure |
| Wiki.js | Node | Postgres (or others) | markdown/WYSIWYG | feature-rich team wikis |
| Outline | Node | Postgres + Redis + S3 | block editor | Notion-style team docs |
| trip2g | single Go binary | SQLite | Obsidian | Obsidian authors who want a live, queryable wiki |

Each of the others earns its place. DokuWiki is the lightest if you accept its markup and dated feel. BookStack's shelf/book/page hierarchy is excellent for structured manuals. Wiki.js is the most feature-complete team wiki. Outline is the closest to a Notion feel if you can run its stack. Pick trip2g when your source of truth is an Obsidian vault and you want that same vault to be the wiki, searchable by people and by agents, on the least infrastructure.

### What trip2g gives you

- **One binary, one file.** The `trip2g` binary embeds all frontend assets; the only host dependencies are `git` and CA certificates. State lives in one SQLite database.
- **Obsidian as the editor.** Write in the editor you already like; wikilinks and frontmatter carry over. No web form.
- **Search built in.** Full-text (BM25 with morphology) works with zero setup; semantic search is optional if you add an embeddings key.
- **Access control.** Notes are private by default; mark `free: true` to publish, and scope the rest to subscribers or a team. See [[en/user/monetization|access control]].
- **Queryable by agents.** The same wiki is an [[en/user/mcp|MCP endpoint]], so your team's AI tools can search it.
- **HTTPS out of the box.** The binary handles Let's Encrypt itself, no reverse proxy required for a single site.

### Set it up (single binary)

The full guide is in [[en/user/selfhosted|Self-hosted deployment]]; the short version:

1. Get the binary. Download from GitHub Releases, or extract it from the Docker image:

   ```bash
   docker create --name tmp ghcr.io/trip2g/trip2g && \
     docker cp tmp:/trip2g /usr/local/bin/trip2g && \
     docker rm tmp
   sudo chmod +x /usr/local/bin/trip2g
   ```

2. Install runtime deps (just two):

   ```bash
   apt-get update && apt-get install -y git ca-certificates
   ```

3. Configure `/etc/trip2g.env` with your domain, a SQLite path, and generated secrets. The key lines:

   ```dotenv
   ACME_DOMAIN=wiki.example.com
   PUBLIC_URL=https://wiki.example.com
   DB_FILE=/var/lib/trip2g/data.sqlite3
   STORAGE_BACKEND=local
   JWT_SECRET=<openssl rand -hex 32>
   DATA_ENCRYPTION_KEY=<openssl rand -hex 16>
   OWNER_EMAIL=you@example.com
   ```

4. Run it under systemd and start it. The binary listens on 443 with TLS and redirects 80. Full unit file in [[en/user/selfhosted|Self-hosted]].
5. Verify:

   ```bash
   curl -I https://wiki.example.com/
   ```

   Expect `HTTP/2 200` with a valid Let's Encrypt certificate. Then sign in with `OWNER_EMAIL` and connect Obsidian per [[en/user/getting-started|Getting started]].

### The backup story

Because the database is one file, backup is simple and that simplicity is the point:

- **Snapshots.** `SIMPLE_BACKUP=true` writes periodic SQLite backups to object storage.
- **Continuous replication.** [Litestream](https://litestream.io) streams the database to any S3-compatible target with a one-second interval; the `infra/` directory ships a ready config. See [[en/user/selfhosted|Self-hosted]].

No `pg_dump`, no coordinating a database dump with a file-storage dump. One file, and optionally one stream.

### Honest tradeoffs

- **SQLite is single-writer.** For a personal or small-team wiki this is a non-issue; for very high concurrent write load a client-server database scales further.
- **Authoring is markdown in Obsidian.** If your contributors need a browser WYSIWYG editor, BookStack or Wiki.js fit them better.
- **It is a live server.** DokuWiki on flat files or a static generator needs even less to run if you never want dynamic features. trip2g earns the server by giving you auth, paywalls, and an MCP endpoint the static options cannot.

Pick trip2g when your wiki starts in Obsidian and you want it live, searchable, access-controlled, and agent-queryable on a single binary and a single file.

### Related

- [[en/user/selfhosted|Self-hosted deployment]] — the complete single-binary and Docker Compose guides
- [[en/user/getting-started|Getting started]] — connect Obsidian and publish your first notes
- [[en/user/mcp|MCP server]] — let agents query the wiki
- [[en/user/search|How search works]] — full-text and semantic search
- [[en/user/backups|Backups]] — snapshots and Litestream replication
