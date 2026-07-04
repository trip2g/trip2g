---
title: "What can you do with trip2g"
free: true
lang_redirect: "[[ru/user/many-doors]]"
---

trip2g is one server that does several jobs with the same markdown files: it publishes your Obsidian vault as a website, serves your notes to AI agents over MCP, posts to Telegram, and gates paid content behind subscriptions. You don't have to want all of that. Pick the door that fits what you already do — the rest is there when you need it.

Underneath every door is the same thing: your notes stay plain markdown files on your disk, and one server renders them into whatever interface the moment calls for.

### Door 1: publish your Obsidian vault as a website

The most concrete one. You write in Obsidian, press Sync, your notes are live. Wikilinks become working links, the graph becomes site navigation, images and attachments come along. No export step, no static-site build pipeline.

Start here: [[en/user/getting-started|Getting started]] — under a minute with the starter vault, or try the free [[en/user/cloud|cloud playground]] with zero setup.

### Door 2: a digital garden

Same mechanics, different intent: not a blog with a timeline, but a growing network of ideas readers explore by connection. Publish the parts of your vault you want public and keep tending them.

See [[en/user/digital-garden|Digital garden]].

### Door 3: a self-hosted wiki or CMS

A team wiki, project documentation, a company knowledge base — with access control per section, full-text search, and every page just a markdown file anyone can edit in Obsidian or push from CI. One Go binary on SQLite; runs on a small VM.

See [[en/user/self-hosted-wiki-sqlite|Self-hosted wiki on SQLite]] and the [[en/user/selfhosted|self-host guide]].

### Door 4: connect an AI agent to your notes

The same server exposes an MCP endpoint. Any MCP client (Claude, Cursor, your own agent) can `search` your notes and read them — scoped by the same access rules readers get. Your agent's memory is files you can open, diff, and `git clone`, not a vector store you can't inspect.

See [[en/user/mcp|MCP]] and [[en/user/mcp-memory-server|MCP memory server]]. Multiple hubs can answer one query together: [[en/user/federation|Federation]].

### Door 5: publish to Telegram

Write a post in Obsidian, it goes to your channel — formatting, links, and images preserved, on a schedule if you want. The site version can carry the extended detail the channel post links to.

See [[en/user/telegram|Telegram publishing]] and [[en/user/telegram-blog-from-obsidian|Telegram blog from Obsidian]].

### Door 6: sell access to your knowledge

Split your notes into free, public, and subscriber-only zones. Readers subscribe; paid Telegram group members get site access automatically. No third-party course platform in the middle.

See [[en/user/monetization|Monetization]] and [[en/user/sell-obsidian-notes|Sell your Obsidian notes]].

### Doors combine

That's the point of one server doing all of it. A YouTuber uses doors 1, 5 and 6 together: articles on the site, announcements in Telegram, depth behind a subscription. A consultant uses 1 and 4: a public knowledge base plus an AI that answers routine questions from it. A team uses 3 and 4: a wiki that both people and agents read. Concrete walkthroughs of these setups: [[en/user/use-cases|Use cases]].

Whichever door you enter, you're not committing to it. The files are the same markdown either way, so adding another door later is a config change, not a migration.
