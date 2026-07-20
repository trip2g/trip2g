---
free: true
title: Publish to Telegram
---

A note can be a web page and a Telegram post at the same time. You write it once in Obsidian; the scheduler sends it.

### One-time setup

1. **Create a bot** — message [@BotFather](https://t.me/BotFather), send `/newbot`, copy the token.
2. **Add the token** in your admin panel under **TG bots**.
3. **Create a channel** (public or private, both work) and add the bot as an admin with *Post messages*.
4. **Link the channel** to a tag in the bot's settings.

### Schedule a post

Two properties on the note, then sync:

```yaml
---
telegram_publish_at: 2026-08-01T09:00
telegram_publish_tags:
  - My channel
---
```

`telegram_publish_at` must have the Obsidian type **Date & time** — plain **Date** is silently ignored by the scheduler. `telegram_publish_tags` must be a **List**. The note goes to every channel whose tags match.

Times are in your site's timezone. Set it in the admin panel before scheduling anything, or 9:00 will mean 9:00 UTC.

### Publish instantly

In the bot's settings there is an **Instant Tags** field. A note carrying one of those tags is sent the moment you sync it, no `telegram_publish_at` needed. Point a spare test channel at an instant tag and you have a preview button.

### Two things that catch people out

- **Editing works, within limits.** Change the text and re-sync and the post is edited in place. Switching between a text post and a media post is not something Telegram allows — reset the post in the admin panel and let it send again.
- **Renaming a note creates a second post.** Posts are tracked by file path. Reset first, then rename, then sync.

Check what is queued and what was sent in the admin panel under **Telegram posts**.

Full guide, including formatting limits and custom emoji: [Telegram publishing](https://trip2g.com/en/user/telegram).

### Next

[[ai-agents]] — point an AI agent at these notes.
