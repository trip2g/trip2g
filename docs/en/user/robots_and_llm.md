---
title: "robots.txt and llms.txt"
description: "Publish robots.txt and llms.txt as ordinary notes using the content_type frontmatter. No special configuration needed."
free: true
lang_redirect: "[[ru/user/robots_and_llm]]"
---

Both `robots.txt` and `llms.txt` are plain-text files that crawlers and AI agents look for at predictable paths. In trip2g you publish them as ordinary notes using the [[en/user/content_type|content_type]] frontmatter.

### robots.txt

Create a note at any path in your vault with this frontmatter:

```yaml
---
slug: /robots.txt
content_type: text/plain; charset=utf-8
free: true
search: false
---
User-agent: *
Disallow:

Sitemap: https://yourdomain.com/sitemap.xml
```

Replace `https://yourdomain.com` with your site's public URL. The `Sitemap:` line tells crawlers where to find your sitemap; use the absolute URL, not a relative path. trip2g generates the sitemap automatically at `/sitemap.xml`.

The `slug: /robots.txt` overrides the note's URL so it is served at the root path that crawlers expect.

Sites without a `robots.txt` note return 404 for that path. Crawlers treat a missing `robots.txt` the same as an allow-all file, so no note means everything is indexed.

### llms.txt

`llms.txt` is a plain-text summary of your site for AI agents and language models. Create a note:

```yaml
---
slug: /llms.txt
content_type: text/plain; charset=utf-8
free: true
search: false
---
# My Site

Short description of what this site is about.

## Key pages
- Docs: https://yourdomain.com/docs
- About: https://yourdomain.com/about

## Facts for AI assistants
- What this site covers and who it is for.
```

Write whatever you want AI agents to know about your site. The note body is served as-is.

### How it works

See [[en/user/content_type|Custom Content-Type notes]] for the full explanation. The short version: `content_type` in frontmatter sets the HTTP `Content-Type` header and causes the note body to be served verbatim, without an HTML wrapper.
