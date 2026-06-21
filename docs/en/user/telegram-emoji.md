---
title: Custom emoji in Telegram posts
free: true
lang: en
lang_redirect: "[[ru/user/Кастомные эмодзи]]"
---

Telegram supports custom emoji — animated and static images from sticker packs. They are more expressive than standard Unicode emoji and display inline with text.

**Requirement:** custom emoji only work when publishing through a [[en/user/telegram|Telegram account with Premium status]]. Bots cannot send custom emoji.

### Get the markdown code

Send the custom emoji to [@trip2g_emoji_bot](https://t.me/trip2g_emoji_bot). The bot replies with ready-to-paste markdown:

```
![emoji](https://ce.trip2g.com/5373112999076699207.webp)
```

Paste that line into your note wherever you want the emoji to appear.

### How it works

**In Obsidian:** the emoji renders as an image in the editor. Animated emoji play in preview mode. The size adjusts automatically to match surrounding text.

**When publishing:** trip2g converts the `ce.trip2g.com` URL back into Telegram's native custom emoji format. The post in your channel shows the real animated or static custom emoji.

### Limitation

Custom emoji require a Telegram Premium account connected in your admin panel. Posts published through a regular account or a bot display the emoji as a plain image at best, or not at all.

### Related

- [[en/user/telegram|Telegram publishing]] — scheduling posts, formatting, media limits
