---
title: RSS Feeds
free: true
lang_redirect: "[[ru/user/rss-feeds]]"
---

RSS in trip2g is a note rendered by a Jet template — not a built-in module. The entire implementation is about 20 lines of markup you can read, copy, and modify to any wire format.

This is "everything is a note" in practice. A feed note carries frontmatter that controls which notes appear and how many. The engine passes those fields to `_layouts/rss.html`, a Jet layout that lives inside your Obsidian vault.

Edit `_layouts/rss.html` like any other file and sync. The template ships in the [[en/user/onboarding-vault|onboarding vault]]; the full source is at [github.com/trip2g/trip2g — rss.html](https://github.com/trip2g/trip2g/blob/main/onboarding-vault/_layouts/rss.html).

[[en/user/kanban|Kanban]], [[en/user/theme-editor|the theme editor]], and RSS all work the same way: layouts that ship in the vault, editable in Obsidian.

### Setting up a feed

Create a note — the onboarding vault already has one at `feed.md` — with:

```yaml
---
slug: /feed.xml
content_type: application/rss+xml; charset=utf-8
layout: rss
free: true
rss_title: My site feed
rss_description: Latest notes
rss_glob: "**"
rss_limit: 20
---
```

Visit `/feed.xml` and the feed is live. Only publicly readable notes ever appear in it — free, not sign-in-gated, not system.

| Field | Purpose | Default |
|-------|---------|---------|
| `rss_glob` | Which notes to include (glob, e.g. `"blog/**"`) | `**` (everything) |
| `rss_limit` | Max items | `20` |
| `rss_title` | Feed title | Note title |
| `rss_description` | Feed description | Note description |

### Customizing the template

Open `_layouts/rss.html` in Obsidian and edit it. The current template produces RSS 2.0 with full `<content:encoded>` bodies. You can add `<author>`, `<category>`, or a `<enclosure>` for podcast audio. To produce a different format — Atom, JSON Feed, a sitemap — write a new layout file and point `layout:` at it in the feed note's frontmatter.

To run multiple feeds with different scopes, create multiple feed notes. For example, one at `/blog-feed.xml` with `rss_glob: "blog/**"` and another at `/podcast.rss` with `rss_glob: "episodes/**"`.
