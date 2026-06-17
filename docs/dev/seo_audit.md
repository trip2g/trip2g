# SEO Capability Audit

Base-SEO audit of the trip2g rendering engine. Each area lists the **current state**
(present / partial / absent) with `file:line` citations, then **gaps / recommendations**.

Audit performed by reading code (June 2026). The default template is quicktemplate
(`internal/defaulttemplate/views.html`); page metadata is assembled in
`internal/case/rendernotepage/endpoint.go` and flows into `defaulttemplate.Ctx`.

A separate plan for structured data lives in `docs/dev/jsonld.md` (referenced, not duplicated).

---

## 1. `<title>`

**Current state: PRESENT.**

- Rendered at `internal/defaulttemplate/views.html:59` — `<title>{%s ctx.Title %}</title>`.
- `Ctx.Title` is set in `endpoint.go:57` from `resp.Title`.
- The title is composed in `resolve.go` via `formatTitle(note.Title, env.SiteTitleTemplate())`. The template is the config key `site_title_template`, default `"%s"` (`internal/configregistry/registry.go:42,67-69`), so e.g. `"%s | My Blog"` yields `My Post | My Blog`.
- `note.Title` resolves from frontmatter `title`, else a leading H1, else the filename (`internal/model/note.go:941-963`).
- Also duplicated into `og:title` (`views.html:35`) and the JS settings blob (`views.html:49`).

**Gaps:** none critical. Minor: no per-page title-length guard (no truncation/warning for >60 chars).

---

## 2. Meta description

**Current state: PARTIAL.**

- Rendered at `views.html:27-29` — only emitted when `ctx.MetaDescription != nil`.
- Sourced in `endpoint.go:58` — `layoutParams.MetaDescription = resp.Note.Description`.
- `note.Description` is populated **only** from the frontmatter `description` key (`internal/model/note.go:557` -> `extractString("description")`, `note.go:927-939`).
- **There is NO content-based fallback.** `buildOGTags` (`endpoint.go:401`) even carries a `// TODO: use a first paragraph as description`. If `description` is missing, no `<meta name="description">` and no `og:description` are emitted.

**Gaps:**
- Add an auto-fallback to the first paragraph (the note already has a `PartialRenderer().Introduce()` used by magazine cards at `views.html:362`) so pages without an explicit `description` still get a snippet. Truncate to ~155-160 chars.
- Documentation correction: both `docs/en/user/advanced.md` and `docs/ru/user/seo.md` currently claim the description "falls back to the start of content" — this is **not true today**.

---

## 3. Canonical URLs

**Current state: ABSENT.**

- No `<link rel="canonical">` anywhere. Grep of `internal/`, `cmd/`, `assets/` for `canonical` returns only code comments about permalinks (`note.go:432,477`; `endpoint.go:389,418`), never a tag.
- The engine *does* enforce a canonical URL via redirects: alternate transliteration variants 301 to the canonical permalink (`endpoint.go:68-77`), and `slug`/`route` are kept distinct from the permalink (`docs/dev/routes.md`). But the canonical is never declared in the document head.

**Gaps (important):**
- Emit `<link rel="canonical" href="{publicURL}{permalink}">` on every page. This matters because the same note is reachable at multiple URLs: permalink, `/index` aliases (`note.go:1346-1352`), alternate-permalink redirects, custom-domain `route`s, and fall-through-by-permalink on custom domains (`routes.md` "Fallthrough"). Without a canonical, custom-domain + main-domain duplication risks split ranking. The `og:url` logic in `ogURLForNote` (`endpoint.go:419-447`) already computes the right per-host URL and can be reused.

---

## 4. OpenGraph / Twitter Card

**Current state: PARTIAL.**

Present tags (built in `endpoint.go:390-415`, rendered `views.html:35-44`):
- `og:title` — always (`views.html:35`, from `ctx.Title`).
- `og:description` — only if frontmatter `description` is set (`endpoint.go:403-405`).
- `og:url` — always; custom-domain-aware via `ogURLForNote` (`endpoint.go:391-394`).
- `og:type` — hardcoded `"article"` (`endpoint.go:396`).
- `og:image` — only if the note body contains an embedded image (`endpoint.go:407-412`).
- `twitter:card` — hardcoded `"summary_large_image"` (`endpoint.go:398`).

**`og:image` source — IMPORTANT:**
- It comes **only** from `note.FirstImage`, which is the first media-extension link found while walking the note AST (`internal/mdloader/loader.go:302-303`), resolved to a presigned asset URL via `AssetReplaces` (`endpoint.go:408-411`).
- **There is NO `og_image` frontmatter key.** Grep of the whole repo for `og_image` returns ZERO hits. Both `docs/en/user/advanced.md:74` and `docs/ru/user/seo.md:36` wrongly tell users to set `og_image` — that key is silently ignored. The only way to control the OG image today is to place the desired image first in the note body (or use a per-page HTML injection / custom layout).

Missing tags:
- No `og:site_name`, no `og:locale` (despite full multilang support), no `article:published_time` / `article:modified_time` (the data exists as `note.CreatedAt`).
- No `twitter:title` / `twitter:description` / `twitter:image` (Twitter falls back to `og:*`, so this is acceptable but not explicit).

**Gaps:**
- Add a real `og_image` (and/or `cover`) frontmatter key so users can set a social image explicitly — this is the single most impactful fix because the current behaviour is undocumented and surprising.
- Add `og:site_name`, `og:locale` (from `note.Lang`), and `article:published_time`.

---

## 5. Sitemap.xml

**Current state: PRESENT (multi-domain), but NOT multilang-aware.**

- Generator: `internal/sitemap/sitemap.go`. `Generate(nvs, publicURL)` (line 28) and `GenerateForDomain(nvs, domain, baseURL)` (line 85).
- Inclusion rules (correct): only `note.Free` (line 32/94), skips sign-in-required subgraphs (lines 36-46 / 98-108), skips system `/_` paths (line 48 / 110).
- Each `<url>` entry has only `<loc>` and optional `<lastmod>` (`urlEntry`, lines 21-24). `lastmod` uses `note.CreatedAt` in RFC3339 (lines 56-58). No `<priority>`, no `<changefreq>`.
- Served by `cmd/server/main.go:handleSitemap` at `/sitemap.xml` (line 2240); switches on the `Host` header to return `DomainSitemaps[host]` for custom domains (lines 2245-2249+).
- Regenerated on note reload (`docs/dev/routes.md`, `noteloader/loader.go`).

**Gaps:**
- **Not multilang-aware.** Despite full `hreflang` support in the head (see section 6), the sitemap emits no `xhtml:link rel="alternate" hreflang=...` entries. Google recommends declaring language alternates in the sitemap; today each language version is just a separate `<loc>`.
- `lastmod` uses `CreatedAt`, not a real "last modified" timestamp — content edits don't bump it.
- robots.txt does not advertise the sitemap (see section 7).

---

## 6. RSS

**Current state: PRESENT.**

- Generator `internal/rssfeed/rssfeed.go` (`Generate(note, publicURL, notes)`); RSS 2.0.
- Served at `*.rss.xml` by `cmd/server/main.go:handleRSSFeed` (line 2205), gated on the `enable_rss` config bool (default `true`, `registry.go:53`; `endpoint.go:544`).
- Model: any note is a feed. Each link in the note becomes an `<item>`; internal targets are enriched with their `description` + `CreatedAt` (`docs/dev/rss.md`). Frontmatter `rss_title` / `rss_description` override channel title/description (`internal/model/note.go:607-614`).
- Discovery: an auto-discovery `<link rel="alternate" type="application/rss+xml">` is emitted in the head when RSS is enabled (`views.html:17-19`), pointing at `{permalink}.rss.xml`.

**Gaps:**
- Item-level feed is link-derived, not a chronological list of the site's posts — fine for curated index pages, less so as a generic "latest posts" feed. Not strictly an SEO gap.
- No `<atom:link rel="self">` in the channel (minor feed-validator nit).

---

## 7. robots.txt

**Current state: PRESENT (config-driven), but does not reference the sitemap.**

- Handler `cmd/server/main.go:handleRobotsTxt` (line 2183), registered in the middleware chain (line 2279). Driven by config key `robots_txt` (`registry.go:45,84-87`), default `"opened"`.
- Three modes (lines 2190-2197):
  - `"opened"` -> `User-agent: *` + `Disallow:` (everything allowed) — the default.
  - `"closed"` -> `User-agent: *` + `Disallow: /` (block all).
  - any other value -> served verbatim as custom robots.txt.

**Gaps:**
- The generated `opened`/`closed` bodies contain **no `Sitemap:` line**. Add `Sitemap: {publicURL}/sitemap.xml` to the opened default so crawlers discover it automatically.
- The default is site-wide open/closed only; there is no per-path disallow generation (e.g. system or paywalled areas) — those rely on per-page `noindex` instead (see section 10). Acceptable, but a custom robots.txt is the only way to disallow specific paths.

---

## 8. Multilang SEO

**Current state: PRESENT (head), PARTIAL (sitemap).**

- `<html lang="...">` — set from `note.Lang` (`views.html:6`; `endpoint.go:61-63`).
- `hreflang` alternates — built in `endpoint.go:buildHrefLangs` (lines 347-386), rendered at `views.html:14-16`. Rules: hub gets `x-default` (+ its own lang if it has one); each language version gets its own `hreflang` plus siblings (`docs/dev/multilang.md` "Layout: hreflang"). Verified in code, matches the docs.
- Per-language URLs via folder structure + `lang` / `lang_redirect` frontmatter; visitor redirect by cookie/`Accept-Language` (`endpoint.go:238-261`); `?nolang` suppresses redirects for bots/SEO tools (`multilang.md` "Edge cases").
- Content-level language switcher rendered in the article (`views.html:280-292`).

**Gaps:**
- `hreflang` `Href` is built as `publicURL + Permalink` (`endpoint.go:357,381`). For pages also served on a **custom domain**, this points at the main domain — known issue, flagged in `multilang.md`. Combining `lang_redirect` with custom domains produces wrong alternates.
- Sitemap omits language alternates (see section 5).
- `og:locale` not emitted despite `note.Lang` being available (see section 4).

---

## 9. Heading structure, semantic HTML, image alt

**Current state: GOOD (semantic), PARTIAL (alt text).**

- Semantic landmarks: `<header>`, `<nav>`, `<main>`, `<aside>`, `<article>`, `<footer>`, `<time>` are used throughout `views.html` (e.g. `<main class="layout__main">` line 255, `<article class="content">` line 279, `<time>` line 369).
- Single H1: the template renders an `<h1>` only when the note body has no leading H1 (`views.html:304`, `!ctx.Note.HasH1()`), avoiding double-H1. Heading IDs are generated and normalized for anchors/TOC (`note.go:826-901`).
- Headings flow into the TOC widget and `#`-anchors work for `[[note#section]]` links.

**Gaps:**
- **Image alt text** in the body is whatever the author wrote in markdown `![alt](url)` — not enforced. The site logo is hardcoded `alt="Logo"` (`views.html:384`). There is no warning for images missing alt text. Recommend a load-time warning (the `NoteWarning` system at `note.go:362` already exists) for images with empty alt.
- Heading-level normalization (`Normalize()`, `note.go:865`) remaps levels to start at 1 — good for TOC, but means the document outline may not match the author's literal heading levels. Usually fine.

---

## 10. URL / slug structure, routing, trailing slashes, redirects

**Current state: GOOD.**

- Clean, transliterated, lowercase permalinks (`note.go:369-394`, `PreparePermalink` 432-488).
- `slug` overrides the URL (`note.go:405-430`); `route`/`routes` add aliases and custom domains without changing the permalink (`docs/dev/routes.md`).
- Redirects: per-note `redirect` frontmatter -> 302 (`endpoint.go:79-83`); non-canonical transliteration variants -> 301 to canonical (`endpoint.go:68-77`); index notes also registered at `.../index` (`note.go:1346-1352`).
- Per-page indexing control: `noindex` set for onboarding (`endpoint.go:106`), sign-in wall (`endpoint.go:119`), and paywall (`endpoint.go:132`); these also set `Cache-Control: no-store`.

**Gaps:**
- No explicit trailing-slash policy. The permalink has no trailing slash, but `.../index` aliases and custom-domain root `route: foo.com/` exist; without a canonical tag (section 3) these can read as duplicates.
- Custom-domain fall-through means any public note is reachable by permalink on any custom domain (`routes.md` "Fallthrough"), multiplying duplicate URLs — another argument for canonical tags.

---

## 11. Structured data (JSON-LD / schema.org)

**Current state: ABSENT (verified).**

- Grep of the whole repo for `ld+json`, `application/ld`, `schema.org` returns ZERO hits in source.
- No `Article`, `BreadcrumbList`, `Organization`, or `WebSite` structured data is emitted.

A dedicated implementation plan is being authored separately at **`docs/dev/jsonld.md`** — see that document for the design. Not duplicated here.

---

## 12. Performance / SEO-adjacent

**Current state: GOOD.**

- `<meta name="viewport">` present (`views.html:10`); `mobile-web-app-capable` + apple meta (`views.html:11-12`); responsive layout with breakpoints (`default_template.md`).
- Server-side rendered HTML served from in-memory cache — crawlers get fully-rendered content with no client JS framework (rendering pipeline in `note_rendering.md`).
- Critical CSS inlined (`views.html:61-63`); JS loaded `defer` (`views.html:89-91`); per-note widget scripts (charts, mermaid) loaded conditionally only when used (`endpoint.go:495-502`).
- Favicons / web manifest / apple-touch-icon present (`views.html:21-25`).
- Early theme script avoids flash-of-unstyled-content (`views.html:9`).

**Gaps:**
- No `theme-color` meta. No explicit `<meta name="robots">` default on normal pages (absence = indexable, which is fine). No image `loading="lazy"` / width-height hints in the body pipeline (CLS risk).

---

## Prioritized gap list (for solid base SEO)

**P0 — must fix (correctness / impact):**
1. **`<link rel="canonical">` on every page** (section 3). The biggest gap given multi-URL exposure (aliases, transliteration variants, custom-domain fall-through).
2. **Real `og_image` (or `cover`) frontmatter key** (section 4). Today it's undocumented and ignored; user docs actively mislead. Either implement the key or fix the docs immediately.
3. **Meta-description content fallback** (section 2). Pages without `description` emit none; add first-paragraph fallback. Fix the user-doc claim that this already works.

**P1 — high value:**
4. Add `Sitemap:` line to robots.txt `opened` default (section 7).
5. Multilang alternates in sitemap (`xhtml:link hreflang`) (sections 5, 8).
6. JSON-LD `Article` / `WebSite` structured data — per `docs/dev/jsonld.md` (section 11).
7. `og:site_name`, `og:locale`, `article:published_time/modified_time` (section 4).

**P2 — polish:**
8. `lastmod` from real modification time, not `CreatedAt` (section 5).
9. Empty-alt-text load warning (section 9).
10. Fix `hreflang` href for custom-domain language versions (section 8).
11. `theme-color` meta, body image `loading="lazy"` + dimensions (section 12).
