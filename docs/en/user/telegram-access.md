---
title: Telegram group access
free: true
home_position: 84
lang_redirect: "[[ru/user/telegram-access]]"
---

Link a Telegram group to a [[en/user/subgraphs|subgraph]] and access flows both ways: group members read the subgraph's notes on your site, and readers with subgraph access get invited into the group. Both directions are live — leave the group and the notes close; let the subscription lapse and the bot removes you from the group.

### The two-way model

| Direction | Who gets what | Revoked when |
|-----------|---------------|--------------|
| Group → content | Every group member can read the linked subgraph's notes | The member leaves or is kicked from the group |
| Content → group | Every reader with active access to the subgraph can join the group | Access expires or is revoked — a background job removes them from the group |

Each direction is a separate switch in the admin. Turn on one, the other, or both. Group membership counts as one more source of "active access" — same standing as a paid offer, a hand grant, Patreon, or Boosty (see [[en/user/subgraphs]] for the access-check order).

### Set it up

**1. Register a bot.** Create one in [@BotFather](https://telegram.me/BotFather), then add the token in Admin → **TG bots** → **+ Add**. The full walkthrough is in [[en/user/telegram]] — it's the same bot you may already use for channel publishing.

**2. Add the bot to your group as administrator.** A plain member bot won't see people joining and leaving, and it can't invite anyone. Admin with the "Add members" right covers both directions. Once added, the group shows up on the bot's page in the admin.

**3. Link the group.** Open Admin → **TG bots** → your bot. Two sections, one per direction:

- **Chats that give access to subgraphs** — check a subgraph next to the group, and its members can read that subgraph's notes.
- **Subgraph access gives invite to this chats** — check a subgraph, and readers who hold access to it can ask the bot to add them to the group.

**4. Tag the notes.** Add `subgraph: club` (or whatever name you linked) to the notes' frontmatter and sync. Tagging a whole folder at once is covered in [[en/user/subgraphs]].

### What a group member sees

1. They join the group. Telegram notifies the bot, the bot records the membership — nothing to do on their side.
2. Anyone in the group sends `/content`. The bot posts an **Open** button that leads to a private chat with the bot.
3. In the private chat the bot verifies membership and replies with the content menu: one button per accessible subgraph. The button opens your site and signs the reader in automatically — no email, no password.
4. They leave the group — the membership record goes away, `/content` comes back empty, the site shows a paywall again.

### What a subscriber sees

The reverse direction, for someone who bought or was granted subgraph access on the site:

1. They connect their Telegram account to their site account (the site shows a link that opens the bot with a one-time attach code).
2. They send `/chats` to the bot. The bot lists the groups linked to their subscriptions, each with a join button.
3. One tap — the bot adds them to the group.
4. When their access expires or you revoke it, a background job kicks them from the group and posts nothing else — the group stays clean without manual moderation.

### Example: a paid club

Notes tagged `subgraph: club`, a private group "My Club", both switches on for `club`.

- You sell an offer that unlocks `club` ([[en/user/monetization]]). A buyer taps `/chats`, joins the group, and reads the notes.
- Or you add a person to the group by hand — they immediately read the `club` notes via `/content`, no purchase involved. Handy for guests and moderators.

For payments you can also keep trip2g out of the money entirely: run any of the popular paid-group bots, let it decide who's in the group, and trip2g grants site access from the membership alone.

**Antipattern:** linking your big public chat to a paid subgraph. Every member of that chat — all of them — gets the paid notes for free. Link only groups whose membership you actually control.

### Related

- [[en/user/subgraphs]] — what a subgraph is, the access-check order, "active access"
- [[en/user/telegram]] — registering the bot, channel publishing
- [[en/user/monetization]] — offers, Patreon, Boosty, crypto
