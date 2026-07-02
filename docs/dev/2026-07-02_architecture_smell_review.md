# Architecture Smell Review (2026-07-02)

Skeptical pass over `main` against the project's own stated principles (`app_patterns.md`, `principles.md`) and Go idiom. Read-only analysis, every claim grounded in file:line.

## TL;DR

The weakest part of the architecture is the `app` god object in `cmd/server` combined with the shallow-copy transaction mechanism: a ~70-field, 25-embed struct with 272 methods whose correctness depends on Go struct-copy semantics (`newEnv := *a`). Every synchronization primitive on it must be a pointer or the system deadlocks or races, and the struct carries multi-paragraph warning comments to keep contributors from stepping on that mine. The top Go-idiom problem is the same pattern seen from the interface side: `graph.Env` is a roughly 340-method mega-interface that only `*app` can satisfy, which inverts the codebase's otherwise excellent small-Env-per-consumer discipline at exactly the point where it matters most.

Verdict up front: the leaf-level architecture (use-case packages, per-case Env, moq tests, SQLite R/W split, service packages) is genuinely good. The rot is concentrated in one place, the composition root, and it is fixable incrementally.

## The weakest link: `app` plus shallow-copy tx envs

### What it is

`cmd/server/main.go:91-247` defines `type app struct` with:

- 2 embedded query facades (`*db.Queries`, `*db.WriteQueries`, main.go:92-93)
- ~20 embedded job types (`*sendtelegrampost.SendTelegramPostJob` and friends, main.go:97-119)
- ~45 named fields covering queues, caches, loaders, clients, token managers, mutexes, atomics, contexts
- 272 methods spread over 20 files in `cmd/server` (`grep -c 'func (a \*app)'`)

`app` is the single concrete type behind every Env interface in the system (196 `type Env interface` declarations across `internal/`). That part is by design and mostly fine. The problem is how transactions are scoped:

```go
// cmd/server/graphql.go:55-59 (same shape in config.go:65-69)
newEnv := *a
newEnv.queries = queries.Queries
newEnv.Queries = queries.Queries
newEnv.WriteQueries = queries
newEnv.currentTx = tx
```

A per-request or per-transaction env is a shallow copy of the entire 70-field struct with three fields swapped. This has three consequences:

1. **Copy semantics become a systemwide invariant.** Any field that must be shared across copies has to be a pointer. The struct itself documents the failure modes: `noteWriteMu` "A value mutex would be copied, defeating cross-request serialization and, worse, deadlocking if copied while locked" (main.go:129-134); `stopped`, `ready`, `appHandler` all repeat the same warning (main.go:141-155); `assetsMu` warns about "a fatal concurrent map write" (main.go:206-209). Four separate defensive comments exist because the pattern was bitten four separate times. The next contributor who adds `mu sync.Mutex` instead of `mu *sync.Mutex` reintroduces the bug class silently; no linter catches it.

2. **A side registry duplicates state already in the copy.** `graphTransactions` at `cmd/server/graphql.go:31-34` is `map[*app]*sql.Tx` keyed by the address of each copied env, populated at graphql.go:67 and drained at graphql.go:91. But the tx is already stored in the copy itself (`newEnv.currentTx`, graphql.go:59, readable via `CurrentTx()` at config.go:42-44). The map exists only so `ReleaseTxEnvInRequest` can find the tx again from the request env. If any code path skips release (a panic that gqlgen recovers past the `handler.go:174-184` commit/rollback hooks, a future second call site), the map entry and the open write transaction leak forever, and on SQLite an orphaned write tx blocks the single writer. The map is also unbounded growth waiting for a bug.

3. **Nested transactions are silently wrong.** `WithTransaction` (config.go:48-88) begins a fresh tx on `a.writeConn` unconditionally. Because the copied env still carries `writeConn`, calling `env.WithTransaction` from inside an already-transactional env opens a second, independent SQLite write transaction rather than a savepoint or an error. Nothing prevents a use case from doing this; the invariant lives in developers' heads.

### Why this is #1

Blast radius: every request, every mutation, every background job flows through this object; `cmd/server` is where all 63 use-case packages, all queues, and all service packages meet. Churn: `case_methods.go`, `note_loader_envs.go`, `config.go` etc. grow with every feature. Testability: `cmd/server/main_test.go:38` constructs a hand-picked partial `app` literal, which works only because tests know which of the 70 fields a code path touches; there is no compiler help.

### Refactor direction

Keep the Env idea, kill the copy trick:

- Replace the shallow copy with a small explicit scope type, e.g. `type txScope struct { *app; q *db.Queries; wq *db.WriteQueries; tx *sql.Tx }` whose method set overrides only the query accessors. The compiler then guarantees the rest of `app` is shared by reference, all mutex/atomic footguns disappear, and the warning comments can be deleted.
- Delete `graphTxs.EnvMap`; `ReleaseTxEnvInRequest` can read the tx from the scope it already holds via `req.Env`.
- Make `WithTransaction` detect an existing tx (`currentTx != nil`) and either reuse it or fail loudly.
- Longer term, group `app` fields into 5-6 domain structs (telegram, payments, notes/loaders, auth/tokens, infra) embedded once, shrinking the flat field list and giving the boot sequence in `main()` natural phases.

## Top smells (ranked)

| # | Smell | Location | Severity | Why it hurts |
|---|-------|----------|----------|--------------|
| 1 | God object + shallow-copy tx env | cmd/server/main.go:91-247, graphql.go:55, config.go:65 | High | Copy-semantics correctness invariant, tx leak vector, nested-tx hazard |
| 2 | `graph.Env` mega-interface | internal/graph/resolver.go:156-497 | High | ~340 methods, only `*app` satisfies it; small-Env discipline inverted |
| 3 | sqlc row types leak through all layers | 137 of the case `resolve.go` files import `internal/db`; graph.Env returns `db.*` rows | Medium-high | Schema change ripples to resolvers and frontend contract |
| 4 | No context in the markdown pipeline | internal/mdloader/loader.go:111 `func Load(options Options)` | Medium-high | Hottest CPU path (full reload per push) is non-cancellable |
| 5 | Hidden IO in innocent getters | cmd/server/config.go:31-35 (`SiteTitleTemplate`), config.go:89-101 (`Now()`) | Medium | `Now()` performs a DB config read with a fresh `context.Background()` per call |
| 6 | Fire-and-forget goroutine per page view | internal/case/rendernotepage/resolve.go:449-454 | Medium | Unbounded, unsupervised, error swallowed, no shutdown coordination |
| 7 | Copy-paste duplication | cmd/server/graphql.go:181-271; internal/case/backjob (12 telegram job clones); case_methods.go `*WithTx` pairs | Medium | Fix-in-one-forget-the-other class of bugs |
| 8 | `appreq.Request` god context + panics on env access | internal/appreq/request.go:28-84; internal/graph/resolver.go:142-154 | Medium | Service-locator with `Env interface{}`; resolver `panic(err)` on missing/mistyped env |
| 9 | Complexity exclusions are un-refactored, not irreducible | internal/case/rendernotepage/resolve.go:192 (Resolve), internal/noteloader/loader.go:139 (228-line Load) | Low-medium | Orchestration functions doing sequential phases inline |
| 10 | `internal/model` as secondary dumping ground | internal/model (5.6k lines), note.go (1472 lines), `RawMeta map[string]interface{}` at note.go:198 | Low | NoteView mixes parsing, rendering, routing, JSON-LD, telegram concerns |

### 2. The `graph.Env` mega-interface

`internal/graph/resolver.go:156-497` embeds roughly 150 use-case `Env` interfaces plus ~190 direct method declarations into one interface. The per-case Env pattern is supposed to produce small consumer-defined interfaces, and at the leaf it does. But the GraphQL layer aggregates them all back into one interface that only `*app` can ever satisfy, so the abstraction buys nothing at that boundary: no alternative implementation is possible, no test can stub it (the wiring test at handler_wiring_test.go stubs a tiny 2-method subset instead, which is telling: the real seam is small). Meanwhile `Resolver.env()` at resolver.go:142-154 panics twice on the happy-path lookup (`panic(err)`, `panic("request env is not of type Env")`), turning a wiring mistake into a process-level recover instead of a 500 with context. Direction: gqlgen's resolver root can hold several domain-scoped interfaces (`AdminEnv`, `TelegramEnv`, `NotesEnv`, ...) as fields; the aggregation then lives in the struct literal at wiring time, not in one unreviewable interface.

### 3. sqlc rows as the de facto domain model

Env methods across the codebase traffic in `db.User`, `db.Offer`, `db.CreateNoteAssetParams`, `db.GetFormStringValuesBySubmitIDRow` (resolver.go:161-210 is a wall of them). This is a deliberate pragmatic choice (sqlc rows are cheap, and `principles.md` never promised hexagonal purity), but the cost is real: renaming a column or reshaping a query type touches use cases, graph resolvers, and via gqlgen bindings potentially the schema. The `Row`-suffixed types leaking into `graph.Env` (e.g. `AllWaitListEmailRequestsRow`, resolver.go:185) are query-shape artifacts appearing in an API-layer contract. Not worth a big-bang fix, but new subsystems should prefer `model` types at Env boundaries, which `noteloader.Env` (loader.go:51-76) already does well.

### 4. Context stops at the markdown pipeline

`mdloader.Load(options Options)` (loader.go:111) and everything below it take no `context.Context`. This is the exact path prior perf work identified as the hot one (full `noteloader.Load` per push, ~45% CPU in GC under load, `docs/dev/sync_perf_memory_2026-06-29.md`). A slow reload cannot be cancelled on shutdown or superseded when a newer push arrives; `noteloader.Load` (loader.go:139, 228 lines, cyclop/gocyclo excluded) holds the loader mutex throughout. Threading ctx through mdloader is mechanical and unlocks cancellation, deadline, and per-phase tracing.

### 5. `Now()` reads the database

`func (a *app) Now() time.Time` (config.go:89-101) calls `a.SiteConfig(ctx)` with a fresh `context.Background()` and 5-second timeout to fetch the configured timezone, then `time.LoadLocation` per call. A function named `Now()` sitting in dozens of Env contracts looks free and pure; it is neither. `SiteTitleTemplate()` (config.go:31-35) has the same shape. Under the covers SiteConfig is presumably cached, but the pattern violates the project's own "IO on the app layer, explicit ctx" discipline and least surprise. Direction: cache the `*time.Location` on config reload and make `Now()` a pure read; getters that can do IO should take ctx.

### 6. Unsupervised goroutine per authenticated page view

rendernotepage/resolve.go:449-454 spawns a goroutine per logged-in view to `RecordUserNoteView` with `context.Background()`, ignoring the returned state entirely. Under a crawl or hot page this is unbounded goroutine creation all funneling writes at SQLite's single writer, with no error visibility and no shutdown drain (a SIGTERM can lose views mid-flight, acceptable, or worse interleave with the writer-slot handoff). Direction: a small buffered channel plus one consumer goroutine owned by `app`, dropping on overflow, closed on shutdown.

### 7. Duplication

`GraphQLRequest` and `GraphQLRequestScoped` (graphql.go:181-223 vs 229-271) are ~45 identical lines differing in one token-stamping line; extract the shared body. The `internal/case/backjob` telegram family (send/update x post/message x bot/account = 12 packages, 43-1200 lines each) plus the paired `X` / `XWithTx` proxies in case_methods.go:22-70 are structural clones; a parameterized job runner or shared core would collapse most of it. gochecknoglobals exclusions, by contrast, are clean: nearly all are immutable lookup tables with honest justifications.

### 8. `appreq.Request` as service locator

`internal/appreq/request.go:28-84` is a pooled, mutable struct carrying `Env interface{}`, the fasthttp ctx, token state, webhook scoping, federation identity, and delivery attribution in one bag, reset field-by-field at request.go:86-106 (forget a field on Reset when adding one and you get cross-request state bleed from the pool; the `mu sync.Mutex` on a pooled struct at least survives reuse). Fetching the env from it with a type assertion is the classic service-locator anti-pattern Go's Env idiom exists to avoid; here both live side by side, and the assertion failure path panics (resolver.go:150). This is load-bearing plumbing for the tx-env swap, so fixing smell #1 shrinks this too.

### 9 and 10, briefly

The golangci complexity exclusions map to long sequential-phase orchestrators, not irreducible algorithms: `rendernotepage.Resolve` (resolve.go:192, ~140 lines) is note lookup, paywall, banner, token, telegram links, sidebars in one function and would split cleanly along those phases. `russkayalatinica` is the exception: a genuinely complex transliteration algorithm where the exclusion is honest. `model/note.go` (1472 lines) accretes routing, JSON-LD, og-image, telegram, and chart concerns onto `NoteView` with `map[string]interface{}` frontmatter (note.go:198); the sibling `note_*.go` split shows the right instinct, it just needs to keep going.

## What's actually good

Calibration matters, and most of this codebase holds up:

- **The per-case Env + moq pattern works.** 63 use-case packages with minimal package-local Env interfaces, 115 `moq` generate directives, table-driven tests. `noteloader.Env` (loader.go:51-76) is a textbook consumer-defined interface with documented methods.
- **Error handling is consistent.** `%w` wrapping with descriptive prefixes everywhere sampled (pushnotes/resolve.go:77-99), sentinel helpers like `db.IsNoFound`, and the ErrorPayload vs system-error split is followed.
- **SQLite discipline is real engineering.** Read/write/queue connection split (main.go:283-284), prepared-statement cache on the read pool with a correct note about why tx paths must not use it (main.go:290-295), and the Block A / writer-slot / Block B boot sequence (main.go:375-520) is well designed and unusually well commented.
- **Service packages follow their own doc.** `gitapi`, `chartdata`, `notebus`, `pagecache` each declare a minimal Env, are named after the domain, embed into `app` without proxy methods, and keep state off package level. `var _ mcp.Env = (*app)(nil)` compile checks are present (main.go:83-84).
- **Globals are almost all justified.** The gochecknoglobals nolints are immutable lookup tables and context keys, each with a reason.
- **The lint config is strict** (golden config, cyclop 30, funlen 150) and the exclusion list is narrow and explained rather than a blanket mute.

## Prioritized remediation

1. **Replace `newEnv := *a` with an explicit tx scope type and delete `graphTxs.EnvMap`** (graphql.go:31-109, config.go:48-88). Highest leverage: removes the copy-semantics invariant, the tx-leak vector, and the nested-tx hazard in one move. Effort: 1-2 days plus careful review of the ~30 `WithTransaction` call sites.
2. **Thread `context.Context` through mdloader and bound the reload path** (mdloader/loader.go:111 downward). Mechanical, unlocks cancellation on the known hot path. Effort: half a day.
3. **Make `Now()` and `SiteTitleTemplate()` pure** by caching the timezone/location on config reload (config.go:31-35, 89-101). Effort: hours.
4. **Supervise `RecordUserNoteView`** with a bounded channel and single writer owned by `app` (rendernotepage/resolve.go:449-454). Effort: hours.
5. **Deduplicate `GraphQLRequest`/`GraphQLRequestScoped`** and then the telegram backjob clones behind a parameterized core. Effort: hours for the first, 2-3 days for the jobs.
6. **Split `graph.Env` into domain-scoped interfaces on the resolver root** and turn the `env()` panics into errors. Effort: 2-4 days, purely mechanical after deciding the grouping; do it after item 1 so the scope type is what gets asserted.
7. **Background, opportunistic:** prefer `model` types over `db.*Row` types in new Env contracts; keep splitting `model/note.go` by concern; phase-split `rendernotepage.Resolve` next time it changes and drop its lint exclusion.

Items 1-4 remove the actual hazards. Items 5-7 are hygiene that pays down churn cost. Nothing here requires a rewrite; the composition root is the only place where the architecture fights its own principles.
