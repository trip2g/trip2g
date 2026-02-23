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
     NO  → GetByRoute("", "/hello")  (main domain aliases)
  3. Fallback: GetByPath("/hello")  (permalink map, always)
```

**Key property:** A host is treated as a custom domain *only if* it has explicit routes in `RouteMap`. Unknown hosts (e.g. `localhost` in development) fall through to main domain routing.

Custom domain routing is **isolated**: main domain alias routes (`route: /x`) are NOT served on custom domains, and custom domain routes are NOT served on the main domain.

### Sitemap

When a request for `/sitemap.xml` arrives with a custom domain `Host`, the server serves `DomainSitemaps[host]` instead of the main sitemap (`cmd/server/main.go`, `handleSitemap`).

---

## Frontmatter Patches Integration

Routes added via frontmatter patches work identically to static frontmatter. The patch system runs after note loading and adds `route`/`routes` keys to `RawMeta` before `ExtractRoutes()` is called.

**Important:** When `createFrontmatterPatch` runs inside a GraphQL mutation transaction, `LoadFrontmatterPatches` uses the same transaction connection (read-your-own-writes in SQLite WAL mode), ensuring the new patch is visible immediately when notes reload. See `cmd/server/note_loader_envs.go`, `latestNoteLoaderEnv.LoadFrontmatterPatches`.

---

## Key Files

| File | Role |
|------|------|
| `internal/model/note_routes.go` | `ParsedRoute`, `ParseRoute`, `NormalizeDomain`, `ExtractRoutes` |
| `internal/model/note.go` | `NoteViews.RouteMap`, `RegisterNoteRoutes`, `GetByRoute`, `IsCustomDomain` |
| `internal/case/rendernotepage/resolve.go` | `resolveNote` — domain-aware lookup |
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

---

## Testing

E2E tests: `e2e/multidomain.spec.js`

Unit tests:
- `internal/model/note_test.go` — ParseRoute, RouteMap registration
- `internal/case/rendernotepage/resolve_note_test.go` — resolveNote scenarios

Test vault: `testdata/vault/multidomain/` (four notes: root, about, multi_route, no_route)
