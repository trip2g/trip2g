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
docs/**.md → { route: "docs.mysite.com/" }
```

All notes in `docs/` automatically end up on `docs.mysite.com`. Note the trailing slash — without it, notes keep their permalink prefix (see [Pitfalls](#what-can-go-wrong) below).

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

1. Add `route: yourdomain.com/` to the frontmatter of the note you want as the home page
2. At your domain registrar, create a CNAME record: `yourdomain.com → your-server`
3. Once DNS propagates, the note opens on the new domain

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

**DNS hasn't propagated yet.** After adding a CNAME record the domain starts working in a few hours, sometimes up to 48. Check with `dig yourdomain.com` or a service like dnschecker.org.

**Forgot the trailing slash.** `mysite.com` and `mysite.com/` are different routes. Without the slash the note is served at its regular permalink on that domain. With the slash it becomes the home page. For the root you need the slash: `route: mysite.com/`.

**HTTPS certificate hasn't been issued yet.** The certificate is issued automatically the first time the domain is hit; this takes up to several minutes. If the browser shows a warning — wait and reload.

**Auth doesn't work on a custom domain.** Cookies are bound to a domain and don't travel between `yoursite.trip2g.ru` and `mysite.com`. If the note is paid, the reader has to sign in separately on each domain. For open content, set `free: true`.

### Sitemap

Each custom domain gets its own `/sitemap.xml` listing only its own notes.

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

### Demo
