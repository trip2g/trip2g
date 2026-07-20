---
free: true
title: Welcome
---

Press the sync button in the left panel of Obsidian, and the content of this note appears on the front page of {{publicUrl}}.

*Нажмите кнопку синхронизации в левой панели Obsidian, и содержимое этой заметки окажется на главной странице сайта.*

That is the whole loop: **edit a note → sync → it is live.** Everything below is optional.

### Read these next

Five short notes in the `_start` folder. They take a few minutes end to end.

- [[publish-a-note]] — make a note public, set its title and URL
- [[telegram]] — send notes to a Telegram channel, scheduled or instantly
- [[ai-agents]] — let Claude Code, Codex or any MCP client search and edit your notes
- [[on-your-phone]] — open this same vault in Obsidian on iPhone or Android
- [[what-else]] — the rest of the platform, in one list

### Already in this vault

Four working examples you can open, change, and sync:

| Note | What it demonstrates |
|------|----------------------|
| `_header.md` | The site's top navigation — edit the list, sync, the menu changes |
| `robots.md` | A note served as `/robots.txt` via `content_type` |
| `feed.md` | An RSS feed at `/feed.xml`, rendered by `_layouts/rss.html` |
| `simple_layout.md` | A page rendered through your own Jet template |

### Things worth knowing right away

- **Notes are private by default.** A fresh note is visible only to you as admin. `free: true` in the frontmatter opens it to the world.
- **Anything starting with `_` is hidden** from listings, search and RSS — that is why the tutorial notes live in `_start/` and why this file is `_index`. Hidden notes are still reachable by direct link.
- **Delete freely.** `_start/` and the demo notes are examples, not machinery. Remove them once you have read them; nothing breaks.

Full documentation: [trip2g.com](https://trip2g.com/en/user/getting-started). Your admin panel: {{publicUrl}}/admin
