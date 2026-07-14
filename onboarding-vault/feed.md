---
slug: /feed.xml
content_type: application/rss+xml; charset=utf-8
layout: rss
free: true
search: false
rss_title: My site feed
rss_description: Latest notes
rss_glob: "**"
rss_limit: 20
---

This note renders an RSS 2.0 feed via `_layouts/rss.html`. Edit the `rss_glob`
(which notes to include) and `rss_limit` frontmatter to configure it. Only
publicly readable notes are ever included.
