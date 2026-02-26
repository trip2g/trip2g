# Multi-Domain Routing

## Overview

Notes can be accessible on custom domains via `route`/`routes` frontmatter fields. A note can serve as the root of a custom domain, appear at a custom path on it, or have alias URLs on the main domain — without changing its system-generated permalink.

### Key distinction from `slug`

| Field | Changes Permalink | In `RouteMap` | Purpose |
|-------|:-----------------:|:-------------:|---------|
| `slug` | Yes | No | Legacy: changes the canonical URL |
| `route`/`routes` | No | Yes | Aliases, custom domain routes |

---

## Frontmatter Syntax

```yaml
# Single route
route: customdomain.example/

# Multiple routes
routes:
  - customdomain.example/
  - customdomain.example/about
  - /main-domain-alias   # route on the main domain
```

### Parsing rules

| Value | Parsed as | Example result |
|-------|-----------|----------------|
| `/about` | Main domain alias | `{Host: "", Path: "/about"}` |
| `foo.com` | Custom domain, path = note's permalink | `{Host: "foo.com", Path: ""}` |
| `foo.com/` | Custom domain root | `{Host: "foo.com", Path: "/"}` |
| `foo.com/hello` | Custom domain, explicit path | `{Host: "foo.com", Path: "/hello"}` |

Hosts are normalized: lowercased, `www.` stripped. Port is preserved (`localhost:8081`).

---

## Data Model

### `ParsedRoute` (`internal/model/note_routes.go`)

```go
type ParsedRoute struct {
    Host string // "" = main domain alias; "foo.com" = custom domain
    Path string // "" = use note's Permalink; "/" = root; "/x" = explicit path
}
```

### `NoteViews.RouteMap` (`internal/model/note.go`)

```go
// host → path → *NoteView
RouteMap map[string]map[string]*NoteView
```

`RouteMap[""]` — main domain alias routes (from `route: /about`)
`RouteMap["foo.com"]` — custom domain routes

Built once during `Load()`, never mutated. Thread-safe for concurrent reads.

### `NoteViews.DomainSitemaps`

Pre-generated sitemaps keyed by normalized domain, populated in `internal/noteloader/loader.go` after each load.

---

## Request Routing

**File:** `internal/case/rendernotepage/resolve.go`, `resolveNote()`

```
Request arrives with Host: foo.com, path: /hello
  1. NormalizeDomain(Host) = "foo.com"
  2. IsCustomDomain("foo.com")?  → checks RouteMap["foo.com"] exists
     YES → GetByRoute("foo.com", "/hello")
           ↳ not found → 404  (no fallback to nv.Map)
     NO  → GetByRoute("", "/hello")  (main domain aliases)
           ↳ not found → GetByPath("/hello")  (permalink map)
```

**Key property:** A host is treated as a custom domain *only if* it has explicit routes in `RouteMap`. Unknown hosts (e.g. `localhost` in development) fall through to main domain routing.

Custom domain routing is **strictly isolated**: notes are accessible on a custom domain *only* if they have an explicit `route: customdomain.com/...` entry. There is no fallback to the global permalink map on custom domains. To serve a note on multiple domains, list all routes explicitly:

```yaml
routes:
  - main.com/path      # accessible on main.com
  - other.com/path     # accessible on other.com
```

### Sitemap

When a request for `/sitemap.xml` arrives with a custom domain `Host`, the server serves `DomainSitemaps[host]` instead of the main sitemap (`cmd/server/main.go`, `handleSitemap`).

---

## Frontmatter Patches Integration

Routes added via frontmatter patches work identically to static frontmatter. The patch system runs after note loading and adds `route`/`routes` keys to `RawMeta` before `ExtractRoutes()` is called.

**Important:** When `createFrontmatterPatch` runs inside a GraphQL mutation transaction, `LoadFrontmatterPatches` uses the same transaction connection (read-your-own-writes in SQLite WAL mode), ensuring the new patch is visible immediately when notes reload. See `cmd/server/note_loader_envs.go`, `latestNoteLoaderEnv.LoadFrontmatterPatches`.

---

## Domain-Aware Wikilink Resolution

When a note on a custom domain links to another note via `[[wikilink]]`, the generated `<a href>` uses the domain-correct path, not the canonical permalink.

**How it works:**

At load time, after normal HTML rendering, notes with custom domain routes are re-rendered with domain-specific link resolution (`generateDomainHTMLs` in `internal/mdloader/domain_render.go`). The result is stored in `NoteView.DomainHTML[host]`. At serve time, `Response.NoteHTML()` returns `DomainHTML[host]` when the request is on a known custom domain.

**Resolution rules per link target:**

| Context | Target has… | Generated href |
|---------|-------------|----------------|
| Custom domain `foo.com` | Route on `foo.com` | `/custom-path` (relative) |
| Custom domain `foo.com` | Route on another domain only | `https://bar.com/path` (full URL) |
| Custom domain `foo.com` | No custom domain routes | `https://main.com/permalink` (full URL) |
| Custom domain `foo.com` | Only main-domain alias (`route: /about`) | `https://main.com/permalink` (full URL) |
| **Main domain** | **Route on a custom domain only** | **`https://bar.com/path` (full URL)** |
| Main domain | Main-domain alias (`route: /about`) | `/about` (alias path) |
| Main domain | No routes | Canonical permalink (unchanged) |

**Main domain wikilinks to "domain-only" notes** use the full URL of the first custom domain route. For example, if `extra.md` has `route: extra.trip2g.com/`, then `[[extra]]` on the main domain generates `href="https://extra.trip2g.com/"`, not `href="/extra"`. This applies whenever the target note has custom domain routes but no main-domain alias routes.

**Custom domain wikilinks to main-domain-only notes** use the full main-domain URL. For example, if `extra.md` on `extra.trip2g.com` links to `docs.md` (which has no `extra.trip2g.com` route), the link generates `href="https://trip2g.com/docs"`, not `href="/docs"` (which would 404 on the custom domain). Requires `PublicURL` to be configured on the server.

**Known behavior — cross-domain full URLs:** When note A on `foo.com` links to note B that only has a route on `bar.com`, the generated href is `https://bar.com/path`. This is an absolute URL pointing to the other domain. If `bar.com` is an internal or private domain, this URL will be visible in the HTML of `foo.com`. Configure routes accordingly.

**Not domain-aware (main-domain HTML used):** RSS feed, GraphQL API, MCP, Telegram posts. These always use canonical permalinks. `FreeHTML` (paywall preview) also uses the main-domain version.

**Embed links** (`![[note]]`) are never domain-rewritten — they always use the canonical permalink so the embedded content renders correctly.

---

## Key Files

| File | Role |
|------|------|
| `internal/model/note_routes.go` | `ParsedRoute`, `ParseRoute`, `NormalizeDomain`, `ExtractRoutes` |
| `internal/model/note.go` | `NoteViews.RouteMap`, `RegisterNoteRoutes`, `GetByRoute`, `IsCustomDomain` |
| `internal/mdloader/domain_render.go` | `generateDomainHTMLs` — per-domain HTML re-render at load time |
| `internal/case/rendernotepage/resolve.go` | `resolveNote`, `Response.NoteHTML()`, `Response.SidebarHTML()` |
| `internal/case/rendernotepage/endpoint.go` | Extracts `Host` header into `Request`; OG URL for custom domains |
| `internal/sitemap/sitemap.go` | `GenerateForDomain` |
| `internal/noteloader/loader.go` | Generates `DomainSitemaps` after load |
| `cmd/server/main.go` | `handleSitemap` — serves domain-specific sitemap |

---

## Edge Cases

| Case | Behavior |
|------|----------|
| `route: /` — collision with `_index.md` | RouteMap wins: the note with `route: /` serves the main domain root |
| `slug` + `route` on same note | Independent: `slug` changes Permalink, `route` adds alias |
| `www.foo.com` vs `foo.com` | Normalized identically |
| Two notes with same `route: foo.com/` | Last registered wins |
| Main domain alias on custom domain | NOT served on custom domain (isolation) |
| Unknown host (localhost in dev) | Treated as main domain |
| Custom domain visitor and auth | Cookies are browser-scoped — use `free: true` for domain notes |
| Note on custom domain accessed via canonical permalink on that domain | 404 — custom domains only serve explicitly routed notes |
| Note needs to appear on two domains | Must declare both routes explicitly: `routes: [a.com/, b.com/]` |
| Wikilink to note with only custom domain routes | Main domain: full URL `https://custom.com/path`; same domain: relative path |
| Wikilink to note with main-domain alias (`route: /about`) | Main domain: `/about` (alias path); custom domain: full main-domain URL |
| Wikilink from custom domain to main-domain-only note | Full main-domain URL `https://main.com/permalink` (avoids 404 on isolated domain) |
| Embed `![[note]]` to domain-routed note | Always uses canonical permalink (embed rendering requires nv.Map lookup) |

---

## Testing

E2E tests: `e2e/multidomain.spec.js`

Key E2E scenarios:
- Custom domain root/subpath/multi-route — serve correct notes
- Main domain alias not served on custom domain
- **Custom domain does not serve notes without explicit routes** (404, no fallthrough)
- **Main domain: link to custom-domain-only note uses full URL**
- Route via frontmatter patch is accessible
- Domain-aware wikilinks use domain path on custom domain

Unit tests:
- `internal/model/note_test.go` — ParseRoute, RouteMap registration
- `internal/case/rendernotepage/resolve_note_test.go` — resolveNote scenarios including `TestResolveNote_KnownCustomDomain_PermalinkNotAccessible`
- `internal/mdloader/domain_render_test.go` — domain HTML re-render, including main domain pass (`DomainHTML[""]`)

Test vault: `testdata/vault/multidomain/` (root, about, multi_route, no_route, domain-link-a, domain-link-b)
