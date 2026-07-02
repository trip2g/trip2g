---
title: "Run a Telegram blog from Obsidian"
description: "Write posts in Obsidian and publish them to a Telegram channel automatically. The publish-out workflow no plugin covers, plus a Markdown-to-Telegram formatting cheatsheet."
free: true
home_position: 44
lang_redirect: "[[ru/user/telegram-blog-from-obsidian]]"
---

Write your posts as normal notes in Obsidian, add two properties, and each note publishes to your Telegram channel on schedule. trip2g handles the formatting conversion, the queue, and later edits. This is the direction almost no tool covers: Obsidian out to a Telegram channel.

### Why this is hard to do any other way

Search for "Obsidian and Telegram" and nearly everything you find goes the wrong way. The popular sync plugins pull Telegram messages *into* your vault as captured notes. That is the inbox direction. Publishing *out* of Obsidian into a channel is a different job, and it is the one people actually mean when they say "I want to run my channel from Obsidian."

The usual fallback is copy-paste: draft in Obsidian, paste into Telegram, fix the formatting by hand, repeat for every post. It works for one post a week. It falls apart the moment you want a queue, a schedule, or edits to a post that is already live.

trip2g treats a note as the source and the channel as one of its outputs. You keep writing in Obsidian. The channel updates itself.

### Telegram is not standard Markdown

This is the part every DIY guide underestimates. Telegram does not render CommonMark. It uses its own MarkdownV2 dialect with different rules, and a raw paste breaks in ways that are annoying to debug:

- Many characters (`_`, `*`, `[`, `]`, `` ` ``, `.`, `-`, `!`) must be escaped or the message is rejected.
- Tables are not supported at all.
- Nested lists flatten.
- Images embedded in text do not appear inline; media is attached separately and compressed.
- A post over 4,096 characters is rejected outright, not split.

trip2g converts your note to Telegram's format for you, so you write ordinary Markdown and get a correctly formatted post. The [[en/user/telegram|Telegram publishing guide]] documents exactly what converts and what does not.

### The trip2g way, step by step

1. Set up a bot once. Create it in [@BotFather](https://telegram.me/BotFather), paste the token into the admin panel under **TG bots**, then add the bot to your channel as an admin. Full steps are in [[en/user/telegram|Telegram publishing]].
2. Write the post as a note in your Obsidian vault. Use normal Markdown: bold, italic, links, code blocks, quotes.
3. Add two properties to the note's frontmatter:

   ```yaml
   telegram_publish_at: 2026-07-10T09:00
   telegram_publish_tags:
     - My channel
   ```

   `telegram_publish_at` must be a **Date & time** property, and `telegram_publish_tags` must be a **List**. The tag matches the channel you configured in the admin panel.
4. Sync. The post enters the queue and goes out at the time you set.

To publish immediately instead of scheduling, use an Instant Tag (see [[en/user/telegram|Telegram publishing]]). A separate test channel with an Instant Tag is the safest way to preview a post before it hits your main audience.

### Editing a post after it is live

Change the note in Obsidian and sync again. trip2g edits the published message in place for text, formatting, links, and lists. Media, post type, and album size are frozen by Telegram after publication, so those need a **Reset** in the admin panel first. The decision map is in [[en/user/telegram|Telegram publishing]].

### Honest alternatives

trip2g is the right answer when you publish regularly and want a queue plus a website copy of each post. For lighter needs, two other paths exist:

| Approach | Good for | The catch |
|---|---|---|
| Manual copy-paste | One-off posts, very low cadence | You reformat every post by hand; no schedule, no edit sync |
| DIY bot (BotFather + a converter script) | Developers who enjoy plumbing | You maintain the escaping, the queue, and the retry logic yourself |
| trip2g (native publish-out) | A real editorial cadence | You run (or rent) a server |

If you post twice a year, copy-paste is fine. Past that, the manual formatting tax is the reason this page exists.

### Formatting cheatsheet

What you write in Obsidian and what Telegram shows:

| You write | Telegram shows |
|---|---|
| `**bold**` | bold |
| `*italic*` | italic |
| `~~strikethrough~~` | strikethrough |
| `` `code` `` | inline code |
| ` ```python ... ``` ` | code block |
| `> quote` | blockquote |
| `> quote\|\|` | collapsible blockquote |
| `==spoiler==` | spoiler |
| `<u>underline</u>` | underline |
| `[text](url)` | link |
| `[[published-note]]` | link to the note's page |
| A table | not supported, drop it |
| An inline image | attach as media instead |

trip2g does the escaping so you never touch backslashes. The full table, including the length limits per post type, is in [[en/user/telegram|Telegram publishing]].

### Common mistakes

- **Pasting from a word processor.** Smart quotes and hidden formatting sneak in. Write in Obsidian, not in a document editor.
- **Expecting inline images.** Embedded images do not show in the body. Attach them as media; Telegram compresses photos.
- **Exceeding the limit.** A text post caps at 4,096 characters, a captioned photo at 1,024. Over the limit, the post does not publish and a warning shows in the admin panel.
- **Renaming a published note.** trip2g identifies a post by its file path, so a rename creates a second post. Reset first, then rename.
- **Leaking the bot token.** Keep it private. Anyone with the token controls the bot.

### Turning the channel into a paid tier

Because the same post also lands on your site, you can gate the extended version behind a subscription: publish the short version to the channel and keep the full write-up on the site for subscribers or paid Telegram-group members. See [[en/user/monetization|Monetization]].

### Related

- [[en/user/telegram|Telegram publishing]] — the complete reference for bots, tags, scheduling, and formatting
- [[en/user/publishing|Publishing notes]] — how to set the `telegram_publish_at` and `telegram_publish_tags` property types correctly
- [[en/user/monetization|Monetization]] — gate the full post behind a paid Telegram group or subscription
