---
free: true
title: SEO
---

A common question: "How is your SEO?" Short answer: the same as anywhere else. HTML is rendered on the server, pages load instantly, and search engines see finished content.

### How it works

trip2g renders HTML the moment your markdown loads. Visitors get a ready page from cache. It behaves like a static site generator — without the manual build step. No browser-side JavaScript frameworks. Google's bot sees the same page a real visitor does.

In terms of speed it is close to [[Quartz]] and other static generators. The difference: you can hide part of the content behind a paywall. A static generator serves everyone identical pages. trip2g checks access and shows the full text only to those who paid.

### Compared to WordPress

| | WordPress | trip2g |
|---|---|---|
| Rendering | Server-side (PHP) | Server-side |
| Speed | Depends on plugins and cache | Fast out of the box |
| Sitemap | Plugin | Automatic |
| Meta tags | Plugin (Yoast, RankMath) | Via note properties |
| Caching | Needs setup | Built in |

There is no fundamental SEO difference [[Wordpress|compared to WordPress]]. The difference is convenience: fewer settings, fewer plugins, less hassle.

### What trip2g does automatically

**Sitemap.xml** — includes every public page, refreshed when content changes. Available at `/sitemap.xml`.

**Robots.txt** — controlled from site settings. Open (everything indexable) by default; you can close the whole site or supply your own robots.txt.

**Clean URLs** — `/docs/seo` instead of `/page.php?id=123`.

**Title tag** — taken from the note title. You can add a site-wide suffix like `%s | My Blog` in site settings.

**Open Graph and Twitter previews** — work out of the box, so links look right when shared on social media. Set the preview image with the `og_image` property (a `[[link]]` or path); if you don't, the first image in your note is used.

**Multilingual tags** — when you publish in several languages, trip2g adds the right `<html lang>` and `hreflang` tags so search engines pair the versions. See [[en/user/multilingual|Multilingual sites]].

**Fast page loads** — no heavy frameworks, minimal JavaScript.

### What you control with note properties

Set these in a note's frontmatter:

| Property | What it does |
|---|---|
| `title` | Page title (the `<title>` tag and the link text). Falls back to the first heading, then the filename. |
| `description` | The text shown under your link in search results, and the social-preview description. Write it yourself — there is no auto-generated fallback, so a note without `description` ships without one. |
| `slug` | Custom URL for the note. |
| `route` / `routes` | Serve the note at extra paths or on a custom domain. See [[en/user/advanced|Custom domains]]. |
| `lang` | The note's language, for multilingual sites. |

The single most useful field is `description`. It is the snippet people read before they click.

### Structured data for search engines

trip2g already gives search engines clean server-rendered HTML, semantic tags (`<article>`, `<nav>`, `<main>`, a single `<h1>`), and Open Graph metadata. Richer structured data (the markup that drives rich results) is on the roadmap — until then your pages are indexed and ranked on content and the metadata above.

### Analytics

Connect Google Analytics, Plausible, or any other analytics service. Paste the counter script into site settings and it runs on every page.

### Paywalled content and SEO

If some pages are paid, what does Google see?

The crawler sees the same thing an unauthenticated visitor sees: title, description, a content preview. The full text stays behind the paywall. This is how The New York Times, Medium, and Substack work.

For SEO this is fine: the page is indexed, appears in search, and brings traffic. The visitor reads the opening and decides whether to pay.

### Wikilinks and internal links

Obsidian links like `[[article]]` turn into ordinary HTML links. Google sees `<a href="/docs/article">`. Internal linking works as expected. Anchors work too: `[[article#section]]` becomes a link to a specific heading.

### What trip2g does NOT do

**It doesn't pick keywords** — that's the author's or an SEO specialist's job.

**It doesn't write content** — though with the [[en/user/mcp|MCP server]] you can connect an AI to help.

**It doesn't buy links** — and we don't recommend you do either.

**It doesn't guarantee the top of Google** — nobody honest does.

SEO is 80% good content: expert articles that answer real reader questions. The technical side matters but comes second. trip2g handles the technical side so you can focus on content.

### How search engines find your site

Over time search engines discover your site on their own — through links from other sites, through your sitemap, through crawling. This happens automatically as long as indexing isn't closed in site settings.

Want to speed it up? Add the site manually:

- [Google Search Console](https://search.google.com/search-console) — for Google
- [Yandex Webmaster](https://webmaster.yandex.ru) — for Yandex

After adding the site, submit your sitemap for indexing. First pages usually appear within 2-7 days.

### How to check it works

1. Open a page in incognito mode — you see what the crawler sees
2. Connect the site in Google Search Console or Yandex Webmaster
3. Use the "URL inspection" tool — the engine shows how it sees your page
4. The sitemap is at `/sitemap.xml`

### Migrating without losing rankings

Moving from another platform? The key is keeping URLs or setting up redirects. If old addresses match new ones, you lose nothing. If addresses change, set up 301 redirects from old URLs to new ones (the `redirect` note property handles per-note redirects).

We help with migration — message us on [Telegram](https://t.me/jrpcd).

### Didn't find what you need?

If you have SEO requirements we didn't cover, message us on [Telegram](https://t.me/jrpcd). We'll add it.
