---
free: true
title: Multi-domains
home_position: 30
lang_redirect: "[[ru/user/multidomains]]"
---

You can attach domains and subdomains to your site and control which notes appear where. The mechanism is a `route` property in the note's frontmatter.

Say you have a portfolio, a blog, and a landing page for a client. Without custom domains everything lives on one address and looks like folders of a single site. With routes, each project gets its own domain — and you manage all of them from one place.

### Trick: a subdomain for an entire folder

If you want to assign a route to a whole folder at once, use [[en/user/frontmatter-patches|frontmatter patches]]. One rule instead of editing every note by hand:

```
docs/** → { route: "docs.mysite.com" }
```

All notes in `docs/` now open on `docs.mysite.com` at their regular permalink: `docs/intro.md` becomes `docs.mysite.com/docs/intro`.

Give the domain's home page to exactly one note. A patch overwrites the note's own `route`, so inside a patched folder use `routes`:

```yaml
routes:
  - docs.mysite.com/
```

Don't put a trailing slash in the patch. `docs.mysite.com/` sends every note in the folder to the same address `/`, the last note loaded wins, and the home page becomes an arbitrary note. To choose paths note by note, see [Folder → custom domain](#folder_custom_domain).

### Who it's for

- Freelancers with a portfolio and a separate blog
- Agencies running multiple client sites
- Authors who want a branded landing page on their own domain

### How it works

Add `route` or `routes` to a note's frontmatter:

```yaml
---
route: mysite.com/
---
```

The note opens at the given URL. The note's main permalink stays untouched.

### Variants

**Root of a custom domain** — the note becomes the domain's home page:

```yaml
route: mysite.com/
```

**A page on a custom domain**:

```yaml
route: mysite.com/about
```

**Several URLs at once**:

```yaml
routes:
  - mysite.com/
  - mysite.com/home
  - /alias-on-the-main-domain
```

**Alias on the main domain** — an extra URL without changing the permalink:

```yaml
route: /blog
```

**Domain without a slash** — the note becomes available on the custom domain at its regular permalink (not at the root):

```yaml
route: mysite.com
```

> `www.mysite.com` and `mysite.com` are treated as the same domain. Letter case doesn't matter. Port is preserved: `localhost:8081`.

### DNS setup

1. Add `route: yourdomain.com/` to the frontmatter of the note you want as the home page, and routes for every other page of the domain
2. At your DNS provider, point the domain at your server: an `A`/`AAAA` record with the server's IP address, or a `CNAME` to the server's hostname for a subdomain like `docs.yourdomain.com`. A bare domain (`yourdomain.com`) cannot have a `CNAME`: use `A`/`AAAA`, or your provider's `ALIAS`/`ANAME` record or CNAME flattening
3. Once DNS propagates, the note opens on the new domain

Add the routes before you switch DNS. A domain with no routes shows your whole main site (see below).

### Difference from slug

| | slug | route / routes |
|---|------|----------------|
| Changes the permalink | Yes | No |
| Custom domain | No | Yes |
| Multiple URLs | No | Yes |
| Aliases | No | Yes |

`slug` and `route` can be used together — they work independently.

### Domain isolation

Domains are isolated:
- Notes on `mysite.com` don't appear on the main domain at the same path
- Main-domain aliases (`route: /blog`) don't work on custom domains

The note itself still opens at its own permalink on the main domain. See [Duplicate pages](#duplicate_pages).

This lets you use a single trip2g site as several independent sites with different content.

### Links between domains

Wikilinks automatically adapt to the reader's domain.

Say note A on `mysite.com` links to note B. Where the link goes depends on where B lives:

| Note B is available on... | The link becomes |
|---|---|
| The same domain `mysite.com` | Relative path: `/about` |
| A different custom domain `other.com` | Full URL: `https://other.com/path` |
| Only on the main domain | Canonical permalink |

On the main domain it works the same way in reverse. If the target note lives only on a custom domain, the link points to the full URL of that domain. If the note has a main-domain alias (`route: /about`), the link uses the alias.

Embeds `![[note]]` always use the canonical permalink — domains don't affect them.

The RSS feed, the GraphQL API, MCP, and Telegram posts also use canonical links. Custom domains don't apply there.

### What can go wrong

**DNS hasn't propagated yet.** After you change a DNS record the domain starts working in a few hours, sometimes up to 48. Check with `dig yourdomain.com` or a service like dnschecker.org.

**Forgot the trailing slash.** `mysite.com` and `mysite.com/` are different routes. Without the slash the note is served at its regular permalink on that domain. With the slash it becomes the home page. For the root you need the slash: `route: mysite.com/`.

**No HTTPS certificate.** trip2g's built-in Let's Encrypt client issues certificates only for the hosts in its ACME domain list (the `-acme-domain` flag, once per host; `www.mysite.com` is a separate host). A route does not add a domain to that list. On your own server, either add the domain to the list and restart trip2g, or terminate TLS in a reverse proxy in front of it. See [[en/user/selfhosted|Self-hosting]].

**The domain shows your whole main site.** A host that reaches trip2g but has no route pointing at it is treated as the main domain, so every page of the main site opens under that name. Add the routes first, then switch DNS. The same happens when you remove the last route of a domain.

**A page returns 404 although it has a route.** If the note's frontmatter is not valid YAML, trip2g ignores the whole block: the route is never registered, and `free` and every other property are lost with it. A common cause is an unquoted colon inside a value, like `title: Pricing: plans`. Quote it: `title: "Pricing: plans"`.

**Auth doesn't work on a custom domain.** Cookies are bound to a domain and don't travel between `yoursite.trip2g.ru` and `mysite.com`. If the note is paid, the reader has to sign in separately on each domain. For open content, set `free: true`.

### SEO on a custom domain

**Sitemap.** Each custom domain answers `/sitemap.xml` with its own sitemap. It lists only the notes routed to that domain that are free and not `noindex`. Submit it in each domain's search console separately.

**robots.txt and llms.txt.** A note with `slug: /robots.txt` answers only on the main domain. A custom domain needs its own note:

```yaml
---
route: mysite.com/robots.txt
content_type: text/plain; charset=utf-8
free: true
noindex: true
search: false
---
User-agent: *
Disallow:

Sitemap: https://mysite.com/sitemap.xml
```

`noindex: true` keeps the file itself out of the domain's sitemap. Write the `Sitemap:` line as an absolute URL on the same domain. `llms.txt` works the same way with `route: mysite.com/llms.txt`. See [[en/user/robots_and_llm|robots.txt and llms.txt]].

#### Duplicate pages

A note routed to a custom domain still opens at its permalink on the main domain and is listed in the main sitemap. With the default template each copy names itself as canonical, so search engines see two pages with the same text. There are two ways out:

- **Keep the domain's notes in a folder whose name starts with `_`**, for example `_mysite/`, and give each note a route with an explicit path: `mysite.com/about`. trip2g never serves a note at a path containing `/_` and leaves such notes out of the main sitemap, so only the custom-domain copy exists. A route without a path (`mysite.com`) won't work here, because it reuses the permalink. Notes in such folders count as system notes: they also drop out of site search, related notes and note lists.
- **Give those notes a custom layout whose canonical points at the custom domain** (see below). Both copies then name the same page as canonical. The main-domain copy stays in the main sitemap.

#### Custom layouts

The default template writes `<link rel="canonical">` and `og:url` with the address the page was opened at: the route on a custom domain, the permalink on the main domain. A custom [[en/user/templates|Jet layout]] gets only `publicURL`, the main domain's address, so build the canonical from the note's route yourself.

A custom layout also writes no `<meta name="robots">` on its own. trip2g sends the `X-Robots-Tag: noindex` header only for notes with `noindex: true`. Render the tag from the note's property with an indexable default: `note.M().GetBool("noindex", false)`. A default of `true` marks every page without the property as `noindex`: the whole site drops out of search while the HTTP headers look clean.

#### Languages and analytics

**hreflang** links always point at the main domain, both in the page `<head>` and in the main sitemap. Domain sitemaps carry none. See [[en/user/multilingual|Multilingual sites]].

**HTML injections** from site settings (analytics counters, verification tags) run on every domain. For a separate counter per domain, put it in the custom layout used by that domain's notes instead.

### Auth

Cookies are browser-scoped and don't move between domains. For notes on custom domains use `free: true` if the content is public.

### Edge cases

**Canonical link doesn't work on a custom domain.** If a note is available at `mysite.com/about`, you can't open it via its regular permalink on `mysite.com` — that's a 404. A custom domain only serves notes with an explicit `route`.

**`route: /` overrides the home page.** Setting `route: /` on the main domain makes that note replace `_index.md` at the root path.

**Two notes with the same route.** The last loaded wins. Avoid duplicate routes.

**A note on multiple domains.** List all the routes explicitly:

```yaml
routes:
  - mysite.com/
  - other.com/
```

### Routes via the API

Routes can also be added not in the note file itself but through the API — via [[en/user/frontmatter-patches|frontmatter patches]]. The patch adds `route` or `routes` to the note's metadata. The result is identical to writing them in the frontmatter by hand.

#### Folder → custom domain

A common case: everything in a `landing/` folder should appear on `landing.example`. The pattern below — applied as a patch to `landing/**` — lets every note in the folder land on the new domain while still giving you control over the path on a per-note basis:

```jsonnet
if std.objectHas(meta, "route") then
  if std.startsWith(meta.route, "/") then
    { route: "landing.example" + meta.route }
  else
    {}
else
  { route: "landing.example" }
```

How it behaves:

| Note's own `route` | Result |
|---|---|
| not set | `route: landing.example` — served on the new domain at its regular permalink |
| starts with `/` (e.g. `/about`) | rewritten to `landing.example/about` — the relative alias is anchored to the custom domain |
| absolute (e.g. `other.com/x`) | left untouched |

For the home page of `landing.example`, write the route explicitly on `landing/_index.md`:

```yaml
route: landing.example/
```

The patch leaves absolute routes untouched, so this is safer than `route: /` — a bare `/` would hijack the root of your main domain if the patch ever gets disabled. For other pages, leave `route` out (the note keeps its permalink on the new domain) or set a relative alias like `route: /about` to land at `landing.example/about`.
