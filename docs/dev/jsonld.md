# JSON-LD structured data (schema.org)

Emit `<script type="application/ld+json">` blocks with schema.org metadata in
every rendered note page so search engines and AI crawlers understand the
content. The decision of **what** to emit is made on the backend per note,
mirroring the mermaid "backend-decided per-note widget" pattern
(`docs/dev/mermaid.md`). No JavaScript, no new endpoint — static JSON injected
into the page `<head>`.

## Why backend-decided

The renderer already parses every note at load time (title, dates, lang,
description, first image, headings, code languages). It knows the note's
metadata before the page is sent. So, exactly like the conditional widget
loader, it can decide the schema.org `@type` and assemble the JSON-LD payload
server-side and put it straight into the initial HTML. No client logic.

The chosen `@type` is a small server-side decision, parallel to
`note.HasCodeLanguage("mermaid")` driving the mermaid script: a single function
inspects the note (frontmatter `type:`, path, magazine flag) and returns the
schema.org type.

## What @type per note

The page kind is decided by `Ctx.JSONLDType()` (new), in priority order:

| Condition | schema.org @type |
|-----------|------------------|
| frontmatter `schema_type:` set (explicit override) | that value verbatim |
| frontmatter `type: profile` or `type: person` | `ProfilePage` + nested `Person` |
| frontmatter `faq: true` or content has a FAQ structure (see Story 5) | `FAQPage` |
| note is a magazine index (`ContentRefs()` contains Magazine) | `CollectionPage` |
| note is the site root / home page (`IsHomePage()` / permalink `/`) | `WebSite` (+ `Organization` publisher) |
| default (a normal note) | `BlogPosting` |

`BlogPosting` is preferred over `Article` because trip2g publishes Obsidian
notes as posts; `BlogPosting` is a strict subclass of `Article` and is what
Google's Article rich-result docs accept. `schema_type:` is the escape hatch
for authors who want `Article`, `NewsArticle`, `TechArticle`, `HowTo`, etc.

In addition to the page-type block, **every** indexable page also emits a
`BreadcrumbList` (derived from the permalink path) and, on every page, a single
site-level `WebSite` + `Organization` graph node. To keep it simple these are
combined into one `@graph` array in a single `<script>` (see Story 4).

## Data mapping: NoteView/Ctx → JSON-LD

The render context already exposes most of what we need. Mapping for the main
`BlogPosting` node:

| JSON-LD property | Source (today) | File |
|------------------|----------------|------|
| `headline` | `ctx.Title` (= `Note.Title()`) | `template.go:47`, `note.go:41` |
| `description` | `*ctx.MetaDescription` (= `Note.Description()`) | `template.go:53`, `note.go:120` |
| `url` / `mainEntityOfPage` | `Notes.ResolveFullURL(note, publicURL)` or `og:url` from `OGTags` | `note.go:1022`, `endpoint.go:390` |
| `inLanguage` | `Note.Lang()` / `ctx.HTMLLang` | `note.go:199`, `template.go:61` |
| `datePublished` | `Note.CreatedAt()` (RFC3339) | `note.go:100`, populated `note_created_at.go` |
| `image` | `Note.FirstImageURL()` (resolved asset URL) | `note.go:251` |
| `dateModified` | **MISSING** — needs new field (Story 1) | — |
| `author` (Person) | **MISSING** — needs new field (Story 1) | — |
| `keywords` (tags) | **MISSING** — needs new field (Story 1) | — |
| `publisher` (Organization) | site config: name + logo (`env.SiteConfig`) | `endpoint.go:544` |
| `wordCount` | derivable from `ReadingTime` or content (optional) | `note.go:210` |

Important: `og:url` already accounts for custom-domain routes
(`ogURLForNote`, `endpoint.go:419`). JSON-LD `url`/`mainEntityOfPage` MUST use
the **same** value as `og:url` so they never disagree. Pass that URL into the
JSON-LD builder rather than recomputing it. On a custom domain the canonical URL
in JSON-LD will then be the domain URL, matching OG.

### Fields missing from NoteView (must be added)

`NoteView` (`internal/model/note.go:156`) today has `Title`, `CreatedAt`,
`Description *string`, `FirstImage *string`, `Permalink`, `Lang`. It has **no**
author, no modified/updated date, no tags. Confirmed: `ExtractMetaData()`
(`note.go:548`) extracts `description`, reading time, headings, charts, code
languages, TOC, layout, MCP, RSS, routes, lang — but not author/tags/updated.
`CreatedAt` comes from the DB and is overridden by frontmatter
`created_at`/`created_on` (`note_created_at.go`).

So we add (Story 1): `Author string`, `UpdatedAt time.Time`, `Tags []string`,
parsed from frontmatter `author`, `updated`/`modified`/`updated_at`, and
`tags`/`keywords`.

## Where and how to inject (quicktemplate)

Inject in `Render()` `<head>` in `internal/defaulttemplate/views.html`, right
after the existing OG tags block (after line 44, before the
`window.__trip2g_settings` script). Gate it behind a guard that suppresses
JSON-LD for non-indexable pages (Story 6).

### Escaping — verified

quicktemplate output tags behave as follows (verified against generated
`views.html.go` and library usage):

- `{%s= x %}` → `qw.N().S(x)` — raw, **no escaping** (would corrupt JSON; not used here).
- `{%q= x %}` → `qw.N().Q(x)` — JSON/JS-quoted string (used for `__trip2g_settings`).
- `{%z= b []byte %}` → `qw.N().Z(b)` — writes raw bytes, **no escaping**. This is
  exactly how the existing form spec is embedded in a `<script>`:
  `views.html:306-308` → `qw.N().Z(specJSON)` (`views.html.go:1372`).
- There is **no** `{%j %}` tag in this quicktemplate version; the codebase always
  uses explicit `json.Marshal` + `{%z= %}` / `{%s= string(b) %}`.

Recommended approach: build the payload in Go (`ctx.JSONLD() []byte` via
`json.Marshal`) and emit with `{%z= %}`, identical to `FormSpecJSON`.

`</script>` safety: Go's `encoding/json.Marshal` escapes `<`, `>`, and `&` to
`<`, `>`, `&` **by default** (HTML-safe), so a `</script>` inside
any string value cannot break out of the `<script>` element. Do NOT disable this
with `SetEscapeHTML(false)`. (Note: `FormSpecJSON` in `templateviews/note.go:351`
uses `enc.SetEscapeHTML(true)` — keep the same for JSON-LD; the `template.go`
`FormSpecJSON` path uses plain `json.Marshal`, which is also HTML-safe.)

Template snippet (after line 44 in `views.html`):

```
{% code jsonld := ctx.JSONLD() %}
{% if len(jsonld) > 0 %}
<script type="application/ld+json">{%z= jsonld %}</script>
{% endif %}
```

After editing `views.html` you MUST run
`go generate ./internal/defaulttemplate/...` and commit the regenerated
`views.html.go` together with the template (`CLAUDE.md`, "Critical: Default
Template Pipeline").

## i18n / locale

JSON-LD is mostly machine data (URLs, dates, names) and needs little
translation. The two locale-sensitive points:

- `inLanguage` — set from `Note.Lang()` (BCP-47). If empty, omit the property
  rather than guessing.
- `Organization`/publisher `name` — comes from site config, already authored by
  the site owner; not template-translated.

If any human-readable label is ever needed (none in the base plan), use
`ctx.T(...)` (`i18n.go:54`) so it follows `UILang`, consistent with the rest of
the template. No new TOML keys are required for the base implementation.

## Multilang & routes

- `inLanguage` = the note's own `Lang`.
- For a note in a `LangGroup`, the JSON-LD MAY add the language alternatives.
  The page already emits `<link rel="alternate" hreflang=...>` from
  `ctx.HrefLangs` (`views.html:14-16`, built by `buildHrefLangs`,
  `endpoint.go:347`). For JSON-LD keep it minimal: set `inLanguage` only;
  hreflang already covers alternates for search engines. (Optional later: add a
  `workTranslation`/`translationOfWork` graph — out of scope for v1.)
- `url`/`mainEntityOfPage` must equal `og:url`, which is already route/custom-
  domain aware (`ogURLForNote`, `endpoint.go:419`; see `docs/dev/routes.md`
  "OG теги"). Reuse `ctx.OGTags["og:url"]` as the canonical URL inside JSON-LD so
  both always agree, including on custom domains.

## Edge cases

- **Missing author**: omit the `author` property entirely (do not emit an empty
  `Person`). Google treats `BlogPosting` without author as valid-but-weaker.
- **Missing dates**: `CreatedAt` is always populated (DB fallback), so
  `datePublished` is always available. `dateModified` is omitted if the new
  `UpdatedAt` is zero.
- **Missing image**: omit `image` if `FirstImageURL()` is "".
- **Non-public notes (paywall / sign-in)**: these pages already set
  `MetaRobots = "noindex, nofollow"` (`endpoint.go:119, 132`) and the body is the
  paywall/sign-in widget, not the article. Do **not** emit article JSON-LD for
  them — there is no content to describe and it would mislead crawlers. Gate on:
  `ctx.PaywallError == nil && ctx.SigninWallError == nil && ctx.OnboardingMode ==
  false && ctx.NotFoundMode == false && ctx.UnsupportedFileExt == "" &&
  ctx.MetaRobots does not contain "noindex"`.
- **Free vs paywalled, but rendered**: a `free: true` note renders fully — emit
  JSON-LD. A note behind a paywall never reaches `SelfContent`, so the gate above
  already excludes it.
- **Draft**: trip2g has no first-class `draft` field today (search found none);
  unpublished content simply isn't synced. If a `draft: true` convention is later
  added, treat it like noindex. Not in scope for v1.
- **System notes** (`_header`, `_footer`, `_sidebar`, layout sections): these are
  embedded, not rendered as their own page through `SelfContent`, so they won't
  carry their own JSON-LD. No special handling needed.
- **404 / unsupported file / onboarding**: excluded by the gate (no article
  content).
- **Federation**: federated/aggregated notes (MCP federation, `MCPFederationKB*`
  fields) are an API concern, not HTML page rendering — they do not go through
  `defaulttemplate.Render`, so JSON-LD does not touch them.

## Implementation plan

### Story 1: NoteView metadata fields

Add to `internal/model/note.go` (`NoteView` struct, near `Description`):

```go
type NoteView struct {
    // ... existing fields ...

    Author    string    // frontmatter "author"
    UpdatedAt time.Time // frontmatter "updated"/"modified"/"updated_at"; zero if unset
    Tags      []string  // frontmatter "tags" or "keywords"
}
```

Extract in `ExtractMetaData()` (call a new `extractJSONLDFields()` alongside the
existing extractors):

```go
func (n *NoteView) extractJSONLDFields(loc *time.Location) {
    if a, ok := n.RawMeta["author"].(string); ok {
        n.Author = strings.TrimSpace(a)
    }
    for _, key := range []string{"updated_at", "updated", "modified"} {
        if v, ok := n.RawMeta[key].(string); ok {
            if t, parsed := parseDate(v, loc); parsed {
                n.UpdatedAt = t
                break
            }
        }
    }
    n.Tags = extractTagList(n.RawMeta["tags"], n.RawMeta["keywords"])
}
```

Reuse the existing `parseDate` helper (`note_created_at.go:43`). `loc` is already
threaded into `ExtractCreatedAt(time.UTC)` from the loader
(`mdloader/loader.go:224`) — pass the same `loc`.

**TDD**: extend the model tests (pattern: `note_codelang_test.go`,
table-driven, `testify/require`). Cases: author present/absent; each date alias;
tags as `[]interface{}`, `[]string`, single string, absent; invalid date adds no
field (and optionally a warning).

### Story 2: templateviews accessors

Add narrow accessors to `internal/templateviews/note.go` (do NOT make templates
`Unwrap()` the model — same rule as mermaid, `mermaid.md`):

```go
func (n *Note) Author() string       { return n.nv.Author }
func (n *Note) UpdatedAt() time.Time  { return n.nv.UpdatedAt }
func (n *Note) Tags() []string        { return n.nv.Tags }
```

`CreatedAt()`, `Title()`, `Description()`, `Lang()`, `FirstImageURL()`,
`Permalink()` already exist.

**TDD**: small accessor tests in `note_test.go`.

### Story 3: JSON-LD builder

Add a `jsonld.go` file in `internal/defaulttemplate/` with the payload structs
and the type-decision function. Build typed Go structs (not `map`) marshaled with
`json.Marshal` so field order and `omitempty` are controlled:

```go
type ldNode struct {
    Context          string      `json:"@context,omitempty"`
    Type             string      `json:"@type"`
    Headline         string      `json:"headline,omitempty"`
    Description      string      `json:"description,omitempty"`
    URL              string      `json:"url,omitempty"`
    MainEntityOfPage string      `json:"mainEntityOfPage,omitempty"`
    InLanguage       string      `json:"inLanguage,omitempty"`
    DatePublished    string      `json:"datePublished,omitempty"`
    DateModified     string      `json:"dateModified,omitempty"`
    Image            string      `json:"image,omitempty"`
    Keywords         []string    `json:"keywords,omitempty"`
    Author           *ldPerson   `json:"author,omitempty"`
    Publisher        *ldOrg      `json:"publisher,omitempty"`
}
```

`ctx.JSONLDType()` implements the decision table above. `ctx.JSONLD() []byte`
assembles a `@graph` with: the page node (BlogPosting/ProfilePage/...), a
`BreadcrumbList`, and the site `WebSite`+`Organization`. Returns `nil` when the
gate (Story 6) says the page is non-indexable.

Use `ctx.OGTags["og:url"]` for `url`/`mainEntityOfPage` (route/domain aware).
Dates via `.Format(time.RFC3339)`. Keep `SetEscapeHTML` enabled (default).

**TDD**: table-driven tests on `JSONLD()` output — unmarshal the bytes back and
assert `@type`, presence/absence of optional fields per input, and that a string
containing `</script>` is escaped (assert raw bytes contain `</script`).
Magazine/home/profile/faq type selection. Tests live next to the builder
(`jsonld_test.go`) and need a `Ctx` with a wrapped `*model.NoteView` — follow
`magazine_test.go` for Ctx construction.

### Story 4: BreadcrumbList + WebSite/Organization graph

Inside `JSONLD()`:

- `BreadcrumbList`: split `note.Permalink` on `/`, build `ListItem`s with `name`
  (title-cased segment, or resolved note title via `ctx.Notes.ByPermalink` when
  available) and `item` (full URL = publicURL + cumulative path). Home is item 1.
- `WebSite`: `name` + `url` from site config; optional `potentialAction`
  SearchAction pointing at the site search if desired (optional).
- `Organization` (publisher): `name` + `logo` from site config. Source the site
  name/logo from `env.SiteConfig(ctx)` (already used in `buildDefaultTemplateCtx`,
  `endpoint.go:544`) — thread the needed values onto `Ctx` in Story 5 rather than
  calling env from the template.

**TDD**: assert breadcrumb item count and URLs for a nested permalink
(`/a/b/c`), and that the graph always includes WebSite+Organization.

### Story 5: wire data into Ctx

`Ctx` (`internal/defaulttemplate/template.go:43`) needs the site-level values not
currently present: site name, site logo URL, and the canonical public base URL.
Add fields:

```go
SiteName    string
SiteLogoURL string
PublicURL   string // base, e.g. https://example.com
```

Populate them in `buildDefaultTemplateCtx`
(`internal/case/rendernotepage/endpoint.go:471`) from
`env.SiteConfig(context.Background())` and `env.PublicURL()` (both already used
in that function). No new env methods required — `PublicURL()` and `SiteConfig()`
are already on the `Env` interface used here.

**TDD**: a focused test that `buildDefaultTemplateCtx` copies these through
(or assert via the existing endpoint/resolve test harness,
`rendernotepage/resolve_test.go`).

### Story 6: template injection + indexability gate

Add `ctx.ShouldEmitJSONLD() bool` to `template.go`:

```go
func (ctx *Ctx) ShouldEmitJSONLD() bool {
    if ctx.Note == nil { return false } // magazine root handled separately if desired
    if ctx.PaywallError != nil || ctx.SigninWallError != nil { return false }
    if ctx.OnboardingMode || ctx.NotFoundMode || ctx.UnsupportedFileExt != "" { return false }
    if strings.Contains(ctx.MetaRobots, "noindex") { return false }
    return true
}
```

`JSONLD()` returns `nil` when `!ShouldEmitJSONLD()`.

Edit `internal/defaulttemplate/views.html` `Render()` `<head>`, after the OG
block (after line 44):

```
{% code jsonld := ctx.JSONLD() %}
{% if len(jsonld) > 0 %}
<script type="application/ld+json">{%z= jsonld %}</script>
{% endif %}
```

Run `go generate ./internal/defaulttemplate/...` and commit `views.html.go`.

**TDD**: render a full page via the existing template test path and assert the
`<script type="application/ld+json">` is present for a normal note and absent for
paywall/404/onboarding. An e2e check can be added to `e2e/vault.spec.js`
(pattern: the mermaid "loads on /mermaid but not elsewhere" test) asserting the
ld+json block exists on a content page and parses as JSON.

### Story 7 (optional): FAQPage detection

Detect a FAQ structure for `FAQPage` `@type`:

- Simplest: frontmatter `faq: true` → treat all H2/H3 + following content as
  Q/A pairs using `Note.Headings` + `PartialRenderer().Sections(level)`
  (`note.go:213`, `note.go:101`). Build `mainEntity: [{ @type: Question, name,
  acceptedAnswer: { @type: Answer, text } }]`.
- This is additive; default behavior (BlogPosting) is unaffected when `faq` is
  unset.

**TDD**: a note with `faq: true` and two H2 sections produces two `Question`
entries with the section text as the answer.

## Build commands (after implementing)

| Command | Why |
|---------|-----|
| `go generate ./internal/defaulttemplate/...` | Regenerate `views.html.go` after editing `views.html` (commit both) |
| `go test ./internal/model/... ./internal/templateviews/... ./internal/defaulttemplate/...` | Run model, accessor, and builder tests |
| `npm run test:e2e` | If an e2e assertion is added |

No CSS, no JS bundle, no `qtc` widget — JSON-LD is pure server-rendered markup,
so none of the asset build steps (`defaulttemplate-css`, `toc`, `mermaid`,
`build`) apply.

## Files to touch

| File | Change |
|------|--------|
| `internal/model/note.go` | `Author`/`UpdatedAt`/`Tags` fields; call `extractJSONLDFields()` |
| `internal/model/note_created_at.go` | reuse `parseDate` (no change, or move helper) |
| `internal/model/jsonld_test.go` (new) | field-extraction tests |
| `internal/templateviews/note.go` | `Author()`/`UpdatedAt()`/`Tags()` accessors |
| `internal/defaulttemplate/jsonld.go` (new) | payload structs, `JSONLDType()`, `JSONLD()`, `ShouldEmitJSONLD()` |
| `internal/defaulttemplate/jsonld_test.go` (new) | builder tests |
| `internal/defaulttemplate/template.go` | `SiteName`/`SiteLogoURL`/`PublicURL` Ctx fields |
| `internal/defaulttemplate/views.html` | inject `<script type="application/ld+json">` in `<head>` |
| `internal/defaulttemplate/views.html.go` | regenerated (commit) |
| `internal/case/rendernotepage/endpoint.go` | populate new Ctx fields in `buildDefaultTemplateCtx` |
| `e2e/vault.spec.js` (optional) | assert ld+json present/parses |

## See also

- `docs/dev/mermaid.md` — the per-note backend-decided emission pattern this mirrors
- `docs/dev/default_template.md` — Ctx fields, `<head>` rendering, i18n
- `docs/dev/multilang.md` — `Lang`, `LangGroup`, hreflang
- `docs/dev/routes.md` — custom-domain `og:url` (reused for JSON-LD `url`)
