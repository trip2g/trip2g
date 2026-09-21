---
free: true
title: Preview a post in a test channel
lang_redirect: "[[ru/user/telegram-instant]]"
---

Instant Tags exist for one job: see the finished post in a channel of your own before your readers see it. Give a private test channel an Instant Tag, and a note carrying that tag arrives there the moment you sync it, without waiting for the schedule. You check the layout, the media and the length on a real post in Telegram rather than on a preview in Obsidian.

### How to set it up

1. Open the admin panel → your bot
2. Find the section **Publish to this groups**
3. In the **Instant Tags** field, select the tags

Notes with those tags publish to that channel as soon as they are synced.

### What to expect

The note still needs both `telegram_publish_at` and `telegram_publish_tags`. Instant Tags change when a note is delivered, not whether it needs a date. A note missing either property is skipped.

Every sync of a changed note sends a new message to the test channel, and the earlier ones stay. Instant posts are never edited, so you get a fresh render each time — including media, which Telegram does not let an edit replace.

The preview does not use up the schedule. The note stays unpublished, and the real post goes out at its own time.

### Without Instant Tags

Copy the note, give the copy a tag that only the test channel uses, and set its `telegram_publish_at` to a time in the past — yesterday's date is enough. The copy goes out on the next scheduler run. Nothing to configure, but the copy is a real note — it gets its own page on the site, and you keep the two versions in sync by hand.

### Related

- [[en/user/telegram]] — set up the bot, the channel and the publishing properties
- [[en/user/publishing]] — property types and how to use `telegram_publish_at` and `telegram_publish_tags` correctly
