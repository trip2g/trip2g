---
title: "Admin panel tour"
free: true
lang_redirect: "[[ru/user/admin_onboarding]]"
---

The admin panel manages everything that is not a note: readers and their access, Telegram bots, payments, API keys, redirects, and system health. It lives at `/_system/admin` — or click the gear icon in the top bar once you are signed in as admin. On a fresh instance you touch three places first: the **Dashboard** (confirm the first sync landed), **Integrations → API Keys** (keys for the sync plugin and AI agents), and **Telegram → TG Bots** (if you publish to a channel). Everything else waits until you need it.

This page walks through every menu section in the order the sidebar shows them.

### Dashboard

The landing page of the panel. One glance answers "is my site healthy and current?":

- **Build info** — the software version your instance runs, plus a link to this documentation.
- **Storage usage** — how much space the database and note assets take.
- **Note Warnings** — notes that rendered with problems (broken links, bad frontmatter). Check this after big syncs.
- **Recently Modified Notes** — the latest changes with permalinks, useful to verify a sync landed.

The toolbar also holds two shortcuts: the [[en/user/editor|in-browser file editor]] and [[en/user/Browser Sync]].

### Users

Your readers: who they are and what they may read. See [[en/user/user_management]] for scripted management via GraphQL.

- **Users** — every reader account. Add users by hand, inspect one, ban one.
- **User Bans** — the list of banned accounts.
- **Subgraph Accesses** — hand-granted access to gated [[en/user/subgraphs|subgraphs]]. This is where you comp a reader without a purchase.
- **Email Waitlist Requests** — emails readers left at a waitlist paywall, with timestamps and IPs.
- **Telegram Waitlist Requests** — the same requests arriving through a Telegram bot.

### Notes & Content

The server-side view of your published vault. You edit notes in Obsidian; this section shows what the server made of them.

- **Subgraphs** — the access groups your notes belong to. See [[en/user/subgraphs]].
- **Note Views** — every published note: permalink, path, title, free flag. The fastest way to answer "is this note live and public?"
- **Note Assets** — images and files attached to notes.
- **Note Graph** — a visual graph of links between published notes.
- **Layout Editor** — edit page layouts in the browser with a block list and live preview. See [[en/user/templates|Custom templates]].
- **Frontmatter Patches** — server-side edits to note properties that survive re-sync. See [[en/user/frontmatter-patches]].
- **Form Notes** — notes that contain forms, and the submissions they collected. See [[en/user/forms]].

### Telegram

Publishing to channels and access through groups. The deep dive is [[en/user/telegram]].

- **TG Bots** — connect a bot: it publishes notes to your channel, signs readers in, and guards [[en/user/telegram-access|group access]].
- **Telegram Accounts** — publish through a regular user account instead of a bot, which lifts Bot API limits on media and custom emoji.
- **Telegram Publish Notes** — the publishing queue: which notes are scheduled, which were sent. **Send Now** pushes a post immediately; **Reset** re-queues it.

### Monetization

Selling access to gated notes. The deep dive is [[en/user/monetization]].

- **Offers** — the products you sell: an offer unlocks a subgraph for a price.
- **Purchases** — payment records: buyer email, payment provider, status.
- **Patreon Credentials** — connect Patreon; tiers and members sync in, patrons get access automatically.
- **Boosty Credentials** — the same for Boosty.

### Integrations

Keys and credentials that connect the outside world to your instance.

- **API Keys** — keys for the Obsidian sync plugin, the [[en/user/mcp|MCP server]], and the GraphQL API. The first thing you create on a new instance — unless the [[en/user/onboarding-vault|onboarding vault]] already did it for you.
- **Git Tokens** — tokens for [[en/user/git|Git access]] to your content.
- **Google OAuth** / **GitHub OAuth** — let readers sign in with Google or GitHub. See [[en/user/oauth]].
- **Federation Secrets** — inbound and outbound secrets for [[en/user/federation|MCP Federation]] between instances.

### SEO & URLs

Keeping URLs healthy after moves and renames. See [[en/user/seo]].

- **Redirects** — 301 redirects from old URLs to new ones. Essential when migrating from another platform.
- **Not Found Paths** — a log of URLs visitors requested and got a 404 for. Feed the real ones into Redirects.
- **Not Found Ignored Patterns** — patterns (bot noise, probes) to keep out of the 404 log.
- **HTML Injections** — HTML snippets injected into `<head>` or the end of `<body>` on every page: analytics counters, verification tags. Injections can be scheduled with active-from/to dates.

### System

The engine room. Most of it you visit only when something looks off or when you automate.

- **Cron Jobs** — built-in scheduled jobs; each can be run by hand.
- **Background Queues** — the job queues behind sync and publishing; start, stop, or clear a queue.
- **Site Config** — instance settings as config entries, with who changed what and when.
- **Health Checks** — status of the instance's subsystems.
- **Browser Sync** — sync a vault folder straight from the browser, no Obsidian needed. See [[en/user/Browser Sync]].
- **Releases** — publish frozen snapshots of your site. See [[en/user/releases]].
- **Admins** — grant admin rights to other users.
- **Audit Logs** — a time-stamped log of notable events on the instance.
- **Change Webhooks** — call external URLs when notes change. See [[en/user/webhooks]].
- **Cron Webhooks** — call external URLs on a schedule.

### Related

- [[en/user/getting-started|Getting started]] — the first-sync walkthrough
- [[en/user/onboarding-vault|The onboarding vault]] — the pre-configured vault a fresh instance offers
- [[en/user/user_management]] — managing users via GraphQL
- [[en/user/monetization]] — offers, Patreon, Boosty
