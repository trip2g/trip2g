---
title: RSS Feeds
free: true
lang_redirect: "[[ru/user/rss-feeds]]"
---

RSS is a note. Add a note with `layout: rss` frontmatter and it renders as an RSS 2.0 feed of your other notes.

### Setup

Create a note (e.g. `feed.md`) with:

```yaml
---
slug: /feed.xml
content_type: application/rss+xml; charset=utf-8
layout: rss
free: true
rss_glob: "**"
rss_limit: 20
---
```

That's it — visit `/feed.xml` and you have a feed. Only publicly readable notes (free, not sign-in-gated, not system) ever appear in it.

| Field | Purpose | Default |
|-------|---------|---------|
| `rss_glob` | Which notes to include (glob, e.g. `"blog/**"`) | `**` (everything) |
| `rss_limit` | Max items | `20` |
| `rss_title` | Feed title | Note title |
| `rss_description` | Feed description | Note description |

### Customizing the feed

The feed is rendered by `_layouts/rss.html` — a normal Jet layout. Edit it (or point `layout:` at your own file) to change item shape, add fields, or build several differently-scoped feeds (e.g. `/blog-feed.xml` with `rss_glob: "blog/**"`).

### Migrating from the old per-note `.rss.xml`

The old automatic `<permalink>.rss.xml` feed (one feed per note, items = the note's links) is gone. Existing subscriber URLs of that form now 404 — there's no redirect. Set up a `/feed.xml` note as above.
