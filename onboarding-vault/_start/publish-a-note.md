---
free: true
title: Publish a note
---

Every note in this vault becomes a page. The URL comes from the file path: `guides/setup.md` → `/guides/setup`.

### Private by default

A freshly synced note is **not** public. Signed in as admin you see it; a visitor gets a sign-in wall. To open a page to the world, add one property:

```yaml
---
free: true
---
```

Sync again and reload in an incognito window to confirm.

In Obsidian, press **Cmd/Ctrl + ;** to add a property. Give `free` the type **Checkbox** and tick it — Obsidian will then autocomplete it on every new note.

### The properties you will actually use

| Property | Type | What it does |
|----------|------|--------------|
| `free` | Checkbox | Makes the page public |
| `title` | Text | The page heading. Defaults to the filename |
| `description` | Text | SEO description and the social preview text |
| `slug` | Text | A custom URL, replacing the path-derived one |
| `subgraphs` | Text or List | Puts the note in a section with its own access rules and sidebar |

```yaml
---
free: true
title: How I set up my home server
description: A short writeup of the hardware and the software.
slug: home-server
---
```

That note lands at `yoursite.com/home-server`.

### Hiding notes

Any file or folder whose name starts with `_` is hidden: it stays out of listings, search and the RSS feed, but is still reachable by direct link and can be embedded elsewhere. That is how `_index`, `_header` and this `_start/` folder behave.

Drafts you do not want online at all are simpler still — keep them outside the synced folder.

### Writing

Standard Markdown, plus the Obsidian things you expect: `[[wikilinks]]` (they resolve to real URLs on the site), `![[embeds]]`, callouts, task lists, footnotes, tables. Mermaid diagrams and CSV-driven charts render automatically. Images, video, audio and documents upload with the note when you sync.

Full syntax reference: [Markdown](https://trip2g.com/en/user/markdown). All properties: [Publishing notes](https://trip2g.com/en/user/publishing).

### Next

[[telegram]] — the same notes, published to a Telegram channel.
