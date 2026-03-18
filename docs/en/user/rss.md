---
title: RSS Feeds
free: true
lang_redirect: "[[ru/user/rss-feeds]]"
---

Any page on your site can become an RSS feed. Links inside a note automatically become feed items.

### How it works

Add `.rss.xml` to any note's URL and get a ready RSS feed.

Example: `/reading` becomes `/reading.rss.xml`

Every link on the page becomes a feed item:
- Link text → item title
- URL → item link
- If the link points to one of your notes — its description and creation date are pulled in

Items appear in the same order as links on the page.

### Setup

#### Basic usage

Nothing to configure. Add `.rss.xml` to any page URL:

```
https://yoursite.com/reading → https://yoursite.com/reading.rss.xml
```

#### Custom title and description

By default the feed title and description come from the note. Override them in frontmatter:

```yaml
---
rss_title: "My Reading List"
rss_description: "Articles and books worth reading"
---
```

| Field | Purpose | Default |
|-------|---------|---------|
| `rss_title` | RSS channel title | Note title |
| `rss_description` | RSS channel description | Note description |

#### Disabling RSS

Add `enable_rss = false` in site settings to disable feed generation entirely.

### Use cases

**Topic curator** — collect links on a topic in one note. Add a new article — subscribers are notified automatically.

**Currently reading** — a growing list of books and articles. Subscribers follow your discoveries in real time.

**Work journal** — daily notes with links to tasks and resources. Colleagues subscribe and see what you're working on.

**Project roadmap** — add links to completed features. Users subscribe to progress updates.

**Changelog** — each release is a link to a note with the change description. History of changes as an RSS feed.

### Troubleshooting

**Feed is empty** — check that the page has links. Images and media don't appear in RSS — only text links.

**No descriptions** — descriptions are pulled only for internal links (your notes). External links have no description.

**No dates** — dates come from the target note's `created_at` field. If missing, the item has no date.
