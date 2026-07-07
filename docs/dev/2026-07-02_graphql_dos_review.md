# GraphQL DoS review — nested-selector bomb / query complexity

**Date:** 2026-07-02 · **Scope:** defensive, read-only. PoC ran against the local `memcli` instance (`http://localhost:24091`) only. Prod was never touched.

## Verdict

A nested-selector-bomb / query-complexity amplification DoS is **not feasible** — neither as an unauthenticated guest nor as a regular user. The reason is a single load-bearing control that is actually wired and working: `extension.FixedComplexityLimit(30)` in `internal/graph/handler.go:80`. Every field in an operation costs at least 1, so both deep nesting and mass-aliasing hit the ceiling of 30 fast and get rejected before any resolver runs.

The owner's belief about the persisted-query allowlist is **correct**: `$trip2g_graphql_persist_queries` is not enforced, and arbitrary ad-hoc queries execute freely. But that gap does not translate into an amplification DoS, because the complexity limit sits in front of everything and does not care whether a query is on the allowlist.

Severity of the amplification-bomb class: **LOW** (mitigated). The real residual exposure is plain volumetric flooding (many cheap-but-real queries in a loop) — see MEDIUM/HIGH findings below.

## Evidence

### The complexity limit is real and effective

`internal/graph/handler.go` registers, on the HTTP server:

```go
srv.Use(extension.Introspection{})
srv.Use(extension.FixedComplexityLimit(30))
srv.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](100)})
```

`gqlgen.yml` sets `omit_complexity: true`, so the generated `executableSchema.Complexity(...)` returns `(0, false)` unconditionally (`internal/graph/generated.go:895`). That looked like it might defeat the limit — it does not. When the custom estimator returns `false`, gqlgen falls back to its **default** per-field cost of `1 + childComplexity`. So `FixedComplexityLimit(30)` behaves as a hard cap of ~30 total fields across the whole selection tree (nesting levels and aliases each count).

Local PoC as guest (no auth), bounded on purpose:

| Query | Result |
|---|---|
| `{ __typename }` | executes — confirms arbitrary ad-hoc queries run (no allowlist) |
| `{ publicUrl }` | executes (complexity 1) |
| `publicUrl` aliased 25× | executes (complexity 25 ≤ 30) |
| `publicUrl` aliased 50× | **rejected**: `operation has complexity 50, which exceeds the limit of 30` (`COMPLEXITY_LIMIT_EXCEEDED`) |
| deep `inLinks { inLinks { … } }` nesting | rejected at validation on the local schema; on any schema where it validates, each level adds ≥1 so it dies at 30 anyway |

The aliasing bomb — the classic amplifier (same expensive field 100× under different aliases) — is exactly what the 30-cap kills.

### The allowlist is not wired

`assets/ui/graphql/queries.ts` holds the generated `$trip2g_graphql_persist_queries` map, but nothing on the server consults it. `internal/graph/handler.go` adds `transport.POST{}`, `transport.SSE{}`, `transport.MultipartForm{}`, `transport.Options{}` and the `AutomaticPersistedQuery` extension. APQ is a **hash-caching optimization, not an allowlist**: any client can register any query by hash and replay it. Confirmed empirically — `{ __typename }` and other ad-hoc queries execute. So the allowlist is dead as designed; the complexity limit is what actually protects the endpoint.

### Introspection is off for guests

`internal/graph/handler.go:101 disableIntrospection` sets `opCtx.DisableIntrospection = true` whenever there is no valid user token and the server is not in dev mode (`makeAroundOperations`, line 151). Guests in prod cannot introspect the schema. Good.

### The self-referential amplifier exists but is mostly out of guest reach

`NoteView.inLinks: [NoteView!]!` (`internal/graph/schema.graphqls:162`) is self-referential — the note → inLinks → note → inLinks graph traversal. It is reachable from `Viewer.latestNoteView`, `AdminQuery.noteView(id)`, `recentlyModifiedNoteViews`, etc. The **guest** `note(input)` query returns `PublicNote`, which deliberately does **not** expose `inLinks` (only `pathId/path/url/title/html/toc`). Either way the 30-cap bounds any traversal.

### Guest-reachable surface and its caps

`Query` root guest-reachable fields: `viewer`, `publicUrl`, `googleAuthUrl`, `oidcAuthUrl`, `githubAuthUrl`, `note` (→ `PublicNote`), `search`, `similarNotes`. Notably:

- `search(input: SearchInput)` — `SearchInput` only carries `query: String!` (no client-controlled limit). The resolver `internal/case/sitesearch/resolve.go` hard-caps results (`vectorTopK = 50`, final slice `[:20]`).
- `similarNotes(input)` — `limit` is clamped server-side to max 20 (`internal/case/similarnotes/resolve.go:68 clampLimit`, `maxLimit = 20`).

So arg-driven list amplification is contained by hardcoded resolver caps rather than by the complexity system.

### No request-rate limiting in front of /graphql

There is no general HTTP rate limiter on the GraphQL endpoint. The rate-limiting that exists is domain-specific: `internal/notfoundtracker` (404 IP throttling), `internal/personaltoken` (5-min throttle), Telegram API backoff. A guest can issue complexity-≤30 queries in a tight loop.

## Findings, rated

| # | Finding | Severity |
|---|---|---|
| 1 | Persisted-query allowlist generated but never enforced; arbitrary ad-hoc queries accepted. Compensated by the complexity limit, so not an amplification DoS on its own. | LOW–MEDIUM |
| 2 | No arg-based complexity weighting (`omit_complexity: true` → all fields cost 1). Today's list resolvers self-cap, but any **new** list/connection field added without an internal cap would be unprotected, since pagination args do not raise complexity. Fragile-by-default pattern. | MEDIUM |
| 3 | No HTTP-level rate limiting in front of `/graphql`. Volumetric flooding of real complexity-30 queries (e.g. `search`, which can trigger vector/embedding work) remains possible. Matters more than usual because the prod box is small and memory-fragile. | MEDIUM–HIGH |
| 4 | No explicit max-depth guard; depth is only bounded indirectly by the 30-cap. Acceptable today, but coupling depth protection to the complexity number means a future limit bump silently widens depth too. | LOW |
| 5 | No alias cap independent of complexity; again only bounded by the 30-cap. | LOW |
| 6 | Self-referential `NoteView.inLinks` amplifier reachable by authenticated users (admin/viewer paths). Bounded by the 30-cap, so not exploitable, but it is the field to watch if the limit is ever raised. | LOW |

## Mitigations, prioritized

1. **Add a request-rate limiter in front of `/graphql`** (per-IP token bucket, e.g. `golang.org/x/time/rate`, tighter for unauthenticated callers). Closes the only realistic DoS path (volumetric) — finding #3. Highest value for a small box.
2. **Keep, and consider lowering, `FixedComplexityLimit`.** 30 already blocks the bomb class. If the real frontend operations fit comfortably under it, leave it; do not raise it casually, since depth/alias protection ride on this number.
3. **Turn on real per-field complexity** (drop `omit_complexity: true`, add weighted `@goField`/complexity funcs for `forceResolver` and list fields — `inLinks`, `search`, `similarNotes`, `warnings`, `url`, `html`). This makes cost reflect actual work and removes the "new uncapped list field is silently unprotected" trap — finding #2. Medium effort.
4. **Wire the persisted-query allowlist** you already generate (`$trip2g_graphql_persist_queries`) as defense-in-depth: reject non-allowlisted operations from first-party clients, or at least gate the guest surface to known queries. Do not treat APQ as this control — it is not. Finding #1.
5. **Add an explicit max-depth extension** (e.g. depth ≤ 10) so depth protection is independent of the complexity number — finding #4.
6. **Audit every list/connection resolver for a hard server-side cap** and make "caps its own result set" a checklist item for new resolvers, until #3 lands — finding #2.

## Bottom line

The endpoint accepts arbitrary queries and the allowlist is indeed inert, but a nested-selector-bomb DoS does not work because `FixedComplexityLimit(30)` is enabled and empirically rejects both aliasing and nesting amplification. The honest gap is not amplification — it is plain flooding: there is no rate limit in front of `/graphql`, which on a small box is the thing worth fixing first.
