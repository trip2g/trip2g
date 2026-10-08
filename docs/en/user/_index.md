---
title: "Documentation"
free: true
left_sidebar: "user/_sidebar"
right_sidebar: [toc]
lang_redirect: "[[ru/user/_index]]"
lang: en
routes: [/docs]
---

Publish your Obsidian vault as a website in under a minute.

Write in Obsidian, press Sync, your notes are live. The same hub also publishes to Telegram, gates paid content, and answers reader questions over MCP. A Markdown Operating System: one note, served to a human as a page and to an agent as a syscall.

**New here?** Follow [[en/user/getting-started|Getting started]]: sign in to your instance, download the starter vault, open it in Obsidian, press Sync. No instance yet? Try the free [[en/user/cloud|cloud sandbox]].

Below are the key pages, grouped by topic. The sidebar on the left lists everything.

## Getting started

- [[en/user/onboarding-vault|The onboarding vault]] — what's inside and how to keep its API key safe.
- [[en/user/admin_onboarding|Admin panel tour]] — where to manage readers, access and payments.
- [[en/user/protocol|How trip2g works]] — the path a note takes from Obsidian to readers and AI agents.
- [[en/user/publishing|Publishing notes]] — title, URL, access and publish time.
- [[en/user/markdown|Markdown syntax]] — which formatting renders on your site.

## Publishing and sync

- [[en/user/two-way-sync|Two-way sync with Obsidian]] — changes made on the server reach your vault.
- [[en/user/git|Git sync]] — clone your site, edit anywhere, push back.
- [[en/user/editor|In-browser editor]] — edit notes and roll back versions without Obsidian.
- [[en/user/live-editing|Live editing]] — turn on "Live reload" in the ☰ menu and the open page reloads by itself after every edit.
- [[en/user/wikilink-resolution|Wikilink resolution]] — which file `[[Name]]` opens when names collide.

## Telegram

- [[en/user/telegram|Telegram publishing]] — posts from your notes go to your channel on a schedule or right away.
- [[en/user/telegram-instant|Preview in a test channel]] — you see the finished post before your readers do.
- [[en/user/telegram-import|Import from Telegram]] — your channel archive becomes Obsidian notes.
- [[en/user/telegram-emoji|Custom emoji]] — animated emoji from sticker packs in your posts.

## Monetization

Leave `free: true` off and only subscribers can read the note; everyone else gets a preview and a sign-in form.

- [[en/user/monetization|Monetization]] — access through a Patreon or Boosty subscription, or for crypto.
- [[en/user/subgraphs|Subgraphs]] — a public blog, a paid course and an internal wiki in one vault.
- [[en/user/telegram-access|Telegram group access]] — group members read gated notes.

## Templates

- [[en/user/default-template|Default template]] — set up the page layout in frontmatter, no HTML.
- [[en/user/themes|Theming the default template]] — your own colors and fonts in one CSS block.
- [[en/user/templates|Custom templates]] — your own page design: one HTML file in `_layouts/`.
- [[en/user/components|Template components]] — templates built from small reusable blocks.
- [[en/user/spa|An app on top of trip2g]] — a note as an app, such as a kanban board.

## AI agents and automation

- [[en/user/mcp|MCP server]] — an AI client answers questions from your knowledge base.
- [[en/user/instructions_guide|Instructions for a knowledge base]] — teach the agent how to search your base.
- [[en/user/fleet|Fleet]] — agents you describe as plain notes.
- [[en/user/webhooks|Webhooks]] — call external services on note changes or on a schedule.
- [[en/user/forms|Forms in notes]] — collect requests and reader answers right on the page.

## Hosting and setup

- [[en/user/hosting|Hosting]] — the demo, managed hosting or your own server.
- [[en/user/selfhosted|Self-hosted]] — a single binary on Linux, Docker Compose or fly.io.
- [[en/user/fly|Deploy on fly.io]] — one machine and one volume, no separate database.
- [[en/user/backup|Backups]] — database snapshots to S3-compatible storage, or Litestream replication.
- [[en/user/multidomains|Multi-domains]] — your own domain or subdomain for sections of the site.
- [[en/user/multilingual|Multilingual sites]] — one vault in several languages.

## Use cases

For writers with paid channels, teams with internal wikis, course authors and open-source projects.

- [[en/user/many-doors|What can you do with trip2g]] — where to start: a site, a wiki, agents, Telegram, sales.
- [[en/user/telegram-blog-from-obsidian|Telegram blog from Obsidian]] — a channel without copy-paste or manual formatting.
- [[en/user/sell-obsidian-notes|Sell access to your Obsidian notes]] — paid access from your own domain.
- [[en/user/team-knowledge-base-mcp|Team knowledge base for AI agents]] — people read a site, agents query over MCP.
- [[en/user/self-hosted-wiki-sqlite|Self-hosted wiki on SQLite]] — compared with Wiki.js, BookStack and Outline.
- [[en/team-knowledge-base|Team knowledge base on a bare VM]] — deployment on a single virtual machine.
- [[en/user/digital-garden|Digital garden]] — a site readers explore by links, not by date.
- [[en/user/use-cases|All examples]] — YouTuber, AI consultant, Telegram channel, project documentation.

What's new: see the [[en/changelog|changelog]].
