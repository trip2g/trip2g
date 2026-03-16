---
title: "The Admin Panel as a Universal Config"
free: true
lang_redirect: "[[ru/thoughts/admin-panel]]"
---

Every feature needs settings. A Telegram bot needs a token and group IDs. OAuth needs keys for Google and GitHub. Patreon needs an API token and subscription tiers.

You could build a separate editor for each entity. A form for the bot, a form for OAuth, a form for Patreon. But that's just duplication.

## One editor for everything

The trip2g admin panel is a universal configuration editor. Any entity is just a record in a table with fields. A bot, an account, a provider, a domain — everything is edited the same way.

Want two Telegram bots? Two records. Three Patreon accounts? Three records. The system makes no distinction between "one" and "many" — it's just a list.

## Two editors

**The admin panel** — a configuration editor. Set it up once: tokens, keys, domains, publishing rules.

**Your markdown editor** (Obsidian, Cursor, VS Code) — a content editor. Your daily driver: writing notes, adding images, editing text.

Config and content are separate. The admin panel looks complex — lots of sections. But most pages are opened once. After that, you work in your own editor and forget the admin panel exists.

This approach is what we call [[en/thoughts/contentless-cms|Contentless CMS]] — the system doesn't store content, it only publishes it.
