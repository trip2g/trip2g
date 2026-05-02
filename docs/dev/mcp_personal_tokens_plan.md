# Personal User Tokens — Implementation Plan

**Goal:** let any user (including admin) issue a long-lived personal token and call API/MCP on their own behalf. MCP gains personal-token support automatically — tokens work everywhere `usertoken` cookie works.

**Driving use case:** test `federated_search` against the telegram demo KB via personal token.

---

## Design

### Generic, not MCP-specific

New table `user_tokens` — universal complement to cookie auth. One token = one user. Any endpoint that reads `request.UserToken()` automatically supports personal tokens.

### Integration point

`internal/appreq/request.go:97 Request.UserToken()` — the single path to "who is this". All use cases go through `CurrentUserToken(ctx)`. We extend this layer; the MCP endpoint gets no special logic (only the Bearer dispatch fix in Task 8).

### Auth resolution order (Request.UserToken())

Resolution order, checked sequentially:
1. **Cookie `trip2g_token`** — current path, JWT.
2. **`Authorization: Bearer <value>`** — if value starts with `t2g_` -> personal token resolve; otherwise ignored (federation JWT handled separately in MCP endpoint).
3. **`?token=<value>`** — same `t2g_` prefix check.

**Cookie wins.** If a user is logged in via cookie, Bearer/query-param are ignored. This prevents an attacker from overriding a browser session.

**Invalid personal token -> hard error (NOT anonymous fallback).** This is an intentional asymmetry with cookie behavior (invalid cookie -> anonymous). Justification: explicit token usage implies intent; silent anonymous fallback would confuse API consumers who expect their token to authenticate them.

### `t2g_` prefix -> disambiguation

- **Personal token:** `t2g_` + 64 random alnum (stored as hash in DB).
- **Federation JWT:** format `<b64>.<b64>.<b64>` (never collides with `t2g_*`).
- **Cookie JWT:** also JWT, but transmitted via cookie — never confused with personal token in Bearer.

### Schema

```sql
create table user_tokens (
  id text primary key,                       -- uuid
  user_id integer not null references users(id) on delete cascade,
  name text not null default '',             -- label, e.g. "Claude Desktop"
  token_hash text not null unique,           -- sha256(plaintext_with_t2g_prefix)
  token_prefix text not null,                -- 't2g_aBcD' (8 chars)
  scope text not null default 'all',         -- reserved for phase 2 (e.g. 'mcp', 'read-only')
  created_at datetime not null default current_timestamp,
  expires_at datetime,                       -- nullable (null = no expiry)
  last_used_at datetime,
  revoked_at datetime
);
create index idx_user_tokens_user_id on user_tokens(user_id);
create index idx_user_tokens_token_hash on user_tokens(token_hash);
```

Analytics and phase 2 fields:
- `last_used_at` — updated on each successful resolve (best-effort, fire-and-forget goroutine, throttled to 1 write per 5 minutes per token).
- `expires_at` — set at creation. **UI default = 90 days.** GraphQL `expiresInDays` is nullable (null = never expires).
- `scope` — reserved for phase 2 (per-token subgraph scope, read-only mode). Always `'all'` in phase 1.

### What we skip in phase 1

- Per-token scope (field exists, unused).
- Rate limit per token.
- Audit log per request (only `last_used_at`). If needed — separate `user_token_logs` table.
- TTL-based cleanup of expired tokens from DB — filter on read by `expires_at < now()` or `revoked_at is not null`.

---

## Tasks

### Task 1 — Migration `user_tokens`

`db/migrations/20260430100000_create_user_tokens_table.sql` (see schema above).

`dbmate up && dbmate dump`. Commit migration + `db/schema.sql`.

**Acceptance:** migration applies cleanly; `db/schema.sql` contains `user_tokens` table and both indexes.

### Task 2 — sqlc queries

`queries.write.sql`:
- `InsertUserToken :one`
- `RevokeUserToken :one` (`where id=? and user_id=? and revoked_at is null returning *`)
- `UpdateUserTokenLastUsedAt :exec`

`queries.read.sql`:
- `UserTokenByHash :one` — returns row where `revoked_at is null and (expires_at is null or expires_at > datetime('now'))`.
- `ListUserTokensByUserID :many` (order by created_at desc).
- `CountActiveUserTokensByUserID :one` (`where revoked_at is null and (expires_at is null or expires_at > datetime('now'))`).

`make sqlc`. `go build ./internal/db/...`.

**Acceptance:** `make sqlc` succeeds; generated Go code compiles; all 6 queries present in generated files.

### Task 3 — `internal/personaltoken/` package

```go
const Prefix = "t2g_"
func Generate() string                       // Prefix + 64 random alnum
func Hash(plaintext string) string           // sha256 hex
func DisplayPrefix(plaintext string) string  // first 8 chars
func IsPersonal(s string) bool               // strings.HasPrefix(s, Prefix)
```

TDD: length, uniqueness, hash is idempotent, `IsPersonal` true/false, different plaintexts produce different hashes.

**Acceptance:** `go test ./internal/personaltoken/...` passes; all functions exported and tested.

### Task 4 — Resolver: plaintext -> `usertoken.Data` (with cache + throttle)

`internal/personaltoken/resolver.go`:

```go
type Env interface {
    UserTokenByHash(ctx context.Context, hash string) (db.UserToken, error)
    AdminByUserID(ctx context.Context, userID int64) (db.Admin, error) // reuse existing query
    UpdateUserTokenLastUsedAt(ctx context.Context, id string) error
}

type Resolver struct {
    env   Env
    cache sync.Map // key: token hash (string), value: cacheEntry
    used  sync.Map // key: token ID (string), value: time.Time (last DB write)
}

type cacheEntry struct {
    data      *usertoken.Data
    fetchedAt time.Time
}

func NewResolver(env Env) *Resolver
func (r *Resolver) Resolve(ctx context.Context, plaintext string) (*usertoken.Data, error)
```

**In-process cache (30s TTL):**
- On resolve: compute hash, check `cache` sync.Map. If entry exists and age < 30s, return cached.
- On miss: DB lookup (`UserTokenByHash` + `AdminByUserID`), populate cache entry.
- On revoke: no explicit invalidation (30s staleness window is acceptable).
- Concurrency-safe via `sync.Map`.

**`last_used_at` throttle (5-minute window):**
- Separate `used` sync.Map keyed by token ID, value = `time.Time` of last DB write.
- On resolve success: if last write was < 5 minutes ago, skip. Otherwise fire-and-forget goroutine: `UpdateUserTokenLastUsedAt` + update map entry.
- Bounded by active token count (max 10/user); no eviction needed.
- Cold start = first use always writes (acceptable).

**Role lookup:** reuse existing `AdminByUserID` query (`internal/db/queries.read.sql.go:90`). Same pattern as `SetupUserToken` in `cmd/server/main.go:1376`. Do NOT create a new `UserRoleByID` query. If `AdminByUserID` returns a row -> role is `"admin"`, otherwise `"user"`.

Returns `*usertoken.Data{ID: int(token.UserID), Role: role}`. Errors (not found, expired, revoked) -> `ErrInvalidToken`.

**Resolver lifecycle:**
- Single long-lived instance (holds caches). Constructed at server init in `cmd/server/main.go`.
- Wired to `appreq.Request` at the same site that already sets `req.TokenManager`. Find that site explicitly and reference it in implementation.
- NOT created in `Acquire()` (which takes no params). After `Acquire()`, in the request entrypoint that already configures `Request` fields.

TDD: valid token resolves, expired returns error, revoked returns error, not-found returns error, admin role detected, non-admin role detected, cache hit on second call within 30s, cache miss after 30s, `last_used_at` throttle skips write within 5 minutes.

**Acceptance:** `go test ./internal/personaltoken/...` passes; resolver correctly caches and throttles; `AdminByUserID` reused (no new role query).

### Task 5 — Extend `appreq.Request.UserToken()`

**File:** `internal/appreq/request.go`

Add optional personal-token resolver to `Request`:

```go
type PersonalTokenResolver interface {
    Resolve(ctx context.Context, plaintext string) (*usertoken.Data, error)
}
type Request struct {
    // ...existing fields...
    PersonalTokenResolver PersonalTokenResolver
}
```

In `UserToken()` after cookie extraction:
1. If cookie yields non-nil -> return (current behavior).
2. Else check `Authorization: Bearer <value>`: if value starts with `t2g_` -> `PersonalTokenResolver.Resolve(ctx, value)`.
3. Else check `?token=<value>`: same `t2g_` prefix check.
4. Else -> `nil` (anonymous).

**`Request.Reset()` must nil the `PersonalTokenResolver` field** (`internal/appreq/request.go:44-55`) to prevent `sync.Pool` contamination. The resolver instance itself is long-lived and shared; the field on `Request` just references it, but must be cleared on pool return.

**Wiring:** in `cmd/server/main.go`, after `Acquire()`, set `req.PersonalTokenResolver = resolverInstance` at the same site that sets `req.TokenManager`.

**Required test cases:**
1. Cookie valid + Bearer `t2g_` valid -> cookie user wins.
2. Cookie absent + Bearer `t2g_` valid -> personal user returned.
3. Cookie absent + Bearer `t2g_` invalid/revoked/expired -> `ErrInvalidToken` (hard error, NOT anonymous).
4. Cookie absent + Bearer non-`t2g_` (federation JWT format) -> nil from `UserToken()` (handled downstream by MCP endpoint).
5. Cookie absent + `?token=t2g_` valid -> personal user returned.
6. Cookie absent + no auth headers -> nil (anonymous).
7. `PersonalTokenResolver` not wired (nil) + Bearer `t2g_` present -> return clear error (not panic). Specify: return `fmt.Errorf("personal token resolver not configured")` or similar.

**Acceptance:** all 7 test cases pass; `Reset()` nils the resolver field; `go test ./internal/appreq/...` passes.

### Task 6 — Use cases: create/revoke

`internal/case/createusertoken/`:
```go
type Env interface {
    CurrentUserToken(ctx) (*usertoken.Data, error)
    CountActiveUserTokensByUserID(ctx, userID int64) (int64, error)
    GenerateUniqID() string
    InsertUserToken(ctx, db.InsertUserTokenParams) (db.UserToken, error)
}
type Input struct {
    Name      string
    ExpiresIn *time.Duration  // nil = no expiry
}
type SuccessPayload struct{ PlaintextToken string; Token db.UserToken }
type ErrorPayload struct{ Message string }
```

Active token limit per user: 10. TDD: success, limit exceeded, `expires_at` computed correctly from duration.

`internal/case/revokeusertoken/` — owner check via `where id=? and user_id=?`.

**Acceptance:** `go test ./internal/case/createusertoken/...` and `revokeusertoken` pass; limit enforced; expiry computed.

### Task 7 — GraphQL

`schema.graphqls`:
```graphql
type UserToken {
  id: ID!
  name: String!
  tokenPrefix: String!
  scope: String!
  createdAt: Time!
  expiresAt: Time
  lastUsedAt: Time
  revokedAt: Time
}
type CreateUserTokenPayload { plaintextToken: String!  token: UserToken! }
type RevokeUserTokenPayload { token: UserToken! }
input CreateUserTokenInput { name: String!  expiresInDays: Int }   # null = no expiry
input RevokeUserTokenInput { id: ID! }

extend type User { tokens: [UserToken!]! }
extend type Mutation {
  createUserToken(input: CreateUserTokenInput!): CreateUserTokenPayload!
  revokeUserToken(input: RevokeUserTokenInput!): RevokeUserTokenPayload!
}
```

`make gqlgen`. Resolvers wrap the use cases.

`User.tokens` resolver: calls `ListUserTokensByUserID(currentUser.ID)`.

`npm run graphqlgen`.

**Acceptance:** `make gqlgen` and `npm run graphqlgen` succeed; resolvers compile; `go build ./...` passes.

### Task 8 — MCP endpoint dispatch + integration test

#### 8a — Bearer dispatch in MCP endpoint

**File:** `internal/case/mcp/endpoint.go:32-45`

BEFORE calling `verifyInbound`, branch on the result of `appreq.Request.UserToken()` (already resolved by middleware from Task 5):

```
if user present in ctx (via appreq):
    // personal token already authenticated the user, skip verifyInbound
    proceed with user context
else if Bearer header present and value is NOT t2g_*:
    verifyInbound(bearer)  // federation JWT path
else:
    anonymous
```

**Key design decision:** branch on `UserToken()` result (user in ctx), NOT on the prefix string in two places. This keeps token-type routing centralized in `appreq`.

**Test:** Bearer with `t2g_` prefix does NOT trigger `verifyInbound`; the user from the personal token reaches `accessibleKBNotes`.

#### 8b — `canReadMCPNote` / `accessibleKBNotes` audit

**No code change required.** Audit confirms the chain works:
`env.CanReadNote(ctx, ...)` -> `canreadnote.Resolve` -> `CurrentUserToken(ctx)` -> `appreq.FromCtx(ctx).UserToken()`.

Once Task 5 is done, the personal-token user is in ctx, and the existing ACL logic applies correctly. Admin user -> sees all. Non-admin -> filtered by subgraph subscription.

**Integration test (required):** MCP endpoint with personal token Bearer -> `accessibleKBNotes` returns notes visible to that user:
- Admin token -> all KB notes visible.
- Non-admin token with subgraph subscription -> only notes in subscribed subgraphs.
- Non-admin token without subscription to a specific subgraph -> those notes filtered out (not error).

Exercises the full ctx chain end-to-end, not mocked.

**Acceptance:** dispatch test passes (personal token skips `verifyInbound`); integration test passes for admin and non-admin paths; `go test ./internal/case/mcp/...` passes.

### Task 9 — UI: Personal Tokens tab

`assets/ui/user/space/space.view.tree`: new tab `tokens <= Tokens $trip2g_user_space_tokens`.

`assets/ui/user/space/tokens/`:
- `tokens.view.tree` — `$mol_page` with token list: name, prefix, scope, created/last_used/expires, Revoke button. Top panel "Add token": input name, select expiry (**default = 90 days**; options: 30d / 90d / 1y / never), Generate button.
- `tokens.view.ts` — three GraphQL operations (list, create, revoke). After create: show modal with plaintext + ready URL `${origin}/_system/mcp?token=${plaintext}` + cURL example, with `$mol_button_copy`. Plaintext kept only in `@$mol_mem` field (not in DB).
- `*.locale=ru.json`.

`npm run build`, smoke in browser.

**Acceptance:** tab renders; token creation shows plaintext modal; revoke works; default expiry selection is 90 days; `npm run build` succeeds.

### Task 10a — Unit & integration tests (Go)

Co-located with implementation tasks (3, 4, 5, 6, 8). Listed here as a checklist so coverage is not left to chance.

**`internal/personaltoken/` (Task 3-4):**
- `Generate` length, prefix `t2g_`, uniqueness across N calls.
- `Hash` is deterministic, idempotent; different plaintexts -> different hashes.
- `IsPersonal` true/false matrix (`t2g_…`, empty, `Bearer t2g_…`, federation JWT, `T2g_` case-sensitivity).
- `Resolver.Resolve`: valid -> returns `*usertoken.Data` with right ID + role; expired -> `ErrInvalidToken`; revoked -> `ErrInvalidToken`; not found -> `ErrInvalidToken`; admin role detected via `AdminByUserID`; non-admin role.
- Cache: second call within 30s does NOT hit `UserTokenByHash`; after 30s does. Verified via mock-call counts.
- Throttle: second call within 5min does NOT call `UpdateUserTokenLastUsedAt`; after 5min does. Mock-based.

**`internal/case/createusertoken/`, `internal/case/revokeusertoken/` (Task 6):**
- `createusertoken`: success returns plaintext + token row; limit exceeded (count >= 10) returns `ErrorPayload`; `ExpiresIn = nil` -> `expires_at` null; `ExpiresIn = 90d` -> `expires_at` set.
- `revokeusertoken`: owner can revoke; not-owner attempt -> `ErrNotFound`; double revoke -> `ErrNotFound` (already revoked).

**`internal/appreq/request_test.go` (Task 5) — table-driven, all 7 cases from Task 5 acceptance:**
- Cookie valid + Bearer `t2g_*` valid -> cookie user (cookie wins; assert `Resolve` not called via mock counter).
- Cookie absent + Bearer `t2g_*` valid -> personal user.
- Cookie absent + Bearer `t2g_*` invalid -> `ErrInvalidToken` (NOT nil, NOT anonymous).
- Cookie absent + Bearer non-`t2g_` -> nil (federation JWT path is not appreq's concern).
- Cookie absent + `?token=t2g_*` valid -> personal user.
- Cookie absent + no auth -> nil.
- `PersonalTokenResolver` field == nil + Bearer `t2g_*` present -> clear error (no panic). Document chosen behavior.
- `Reset()` nils `PersonalTokenResolver` (verified via reflect or assertion after `Release`).

**`internal/case/mcp/endpoint_test.go` + `resolve_test.go` (Task 8):**
- Bearer `t2g_*` does NOT trigger `verifyInbound` (mock the federation env so a `verifyInbound` call would fail; assert no failure surfaces — meaning it wasn't called).
- Bearer `t2g_*` -> `accessibleKBNotes` returns notes filtered for that user (admin sees all KB; non-admin sees only KB-notes in their subgraph).
- Bearer non-`t2g_` JWT -> federation path runs as before (regression).
- No Bearer + cookie set -> existing behavior (regression).

### Task 10b — Playwright e2e (`e2e/personal-tokens.spec.js`)

Use existing two-instance harness from `e2e/federation.spec.js` (hub `:20081`, peer `:20091`). Reuse `e2e/helpers/auth.js` (`graphqlSignIn`) and the `mcpCall` pattern.

**New helper additions:**
- `e2e/helpers/auth.js`: add `createPersonalToken(request, baseURL, cookie, { name, expiresInDays })` that runs `createUserToken` mutation and returns `plaintextToken`. Add `revokePersonalToken(request, baseURL, cookie, id)`.
- `mcpCall(request, url, method, params, { bearer, queryToken })` — extend to accept Bearer header and `?token=` injection.

**Test groups (`test.describe.serial`):**

**Group 1 — Auth resolution (single instance, hub):**
1. `createUserToken` under admin cookie -> plaintext starts with `t2g_`, length 68.
2. MCP `tools/call search` via `Authorization: Bearer t2g_…` -> 200, returns hub-local notes.
3. Same call via `?token=t2g_…` query param -> identical result.
4. Cookie of user A + Bearer of user B's token -> `viewer.user.id` resolves to A (cookie wins). Verified by calling a GraphQL query that exposes current user identity.
5. Bearer `t2g_invalid` -> JSON-RPC error response (not anonymous results).
6. Token created with `expiresInDays: 0` then forced into the past via direct DB UPDATE (or freezing `time.Now` if injectable) -> Bearer call returns error.
7. `revokeUserToken` -> next call with same plaintext returns error within ~30s (cache TTL window — test waits or asserts >=30s gap; if flaky, hit the `clearCache` test-only endpoint or restart resolver).

**Group 2 — ACL for non-admin (single instance):**
8. Setup: admin creates non-admin user, subgraph A (subscribed), subgraph B (not), 1 note in each.
9. Personal token of non-admin -> `search` returns note from A only.
10. Personal token of admin -> `search` returns notes from A and B.

**Group 3 — Federation via personal token (hub + peer):**
11. Setup (extends `federation.spec.js` bootstrap): KB-note on hub with `mcp_federation_kb_url: http://localhost:20091/_system/mcp`, `mcp_federation_kb_id: peer-kb`, in **subgraph A**. Peer has at least one searchable note.
12. Admin's personal token -> `federated_search query="…" kb_id="peer-kb"` -> returns peer notes (proves chain: hub MCP → user-resolved → KB-note visible → outbound federation → peer).
13. Non-admin (subscribed to A)'s personal token -> same `federated_search` -> works (KB-note visible).
14. Non-admin NOT subscribed to A -> KB-note hidden -> `federated_search kb_id="peer-kb"` returns `federation_not_configured` payload, not error.
15. `federated_search` without `kb_id` (fan-out) under non-admin token -> only peers from KB-notes the user can see are queried. Verify by adding a second KB-note in subgraph B and confirming non-admin (subscribed only to A) sees results from peer-A only.

**Group 4 — Smoke / regressions:**
16. Two rapid sequential calls with same token -> both succeed (cache no-regression).
17. Existing federation tests (`federation.spec.js`) still pass — i.e. plain federation Bearer JWT path unaffected. (Run as part of full e2e suite, not duplicated here.)

**Test data setup details:**
- `beforeAll`: graphqlSignIn admin on hub + peer, bootstrap federation secrets (reuse `federation.spec.js` helper), create non-admin user, create subgraph A/B, subscribe non-admin to A, create test notes, create KB-notes pointing at peer.
- `afterAll`: revoke any leftover personal tokens; cleanup (or rely on testdata reset between runs).
- For test-time clock control on expiry: prefer direct DB UPDATE `expires_at = datetime('1970-01-01')` rather than introducing a clock interface in phase 1.

**Acceptance:**
- `npm run test:e2e -- e2e/personal-tokens.spec.js` passes locally and in CI.
- All 17 scenarios produce expected results.
- `federation.spec.js` continues to pass (no regression).
- Helper additions to `auth.js` covered by usage in this spec.

### Task 11 — Documentation

- `docs/en/user/mcp.md` + `docs/ru/user/mcp.md`: section "Personal access token" — how to create, formats (`?token=`, `Bearer`), ACL behavior.
- `docs/dev/mcp_federation.md`: note that federation now works per-user via personal tokens.
- If `docs/dev/auth.md` exists — describe the new auth channel.

**Acceptance:** both language versions exist; content covers creation, usage formats, and ACL semantics.

---

## Task Order

1 -> 2 -> 3 -> 4 (parallel with 6) -> 5 (depends on 4) -> 7 -> 8 -> 9 -> 10 -> 11.

Task 8 can start as soon as Task 5 is done. Without it, personal-token users won't see KB notes in private subgraphs.

---

## ADR: Personal Token Authentication

**Decision:** extend `appreq.Request.UserToken()` with a `PersonalTokenResolver` that handles `t2g_`-prefixed Bearer/query tokens, with an in-process cache and `last_used_at` throttle.

**Drivers:**
1. MCP consumers need stable, long-lived auth (cookies are session-bound and browser-only).
2. Personal tokens must work transparently with all existing ACL checks (no per-endpoint changes).
3. DB load from token resolution must stay minimal (tokens are checked on every request).

**Alternatives considered:**
- **A) Separate MCP-specific auth middleware.** Rejected: duplicates ACL logic, doesn't generalize to other API consumers.
- **B) No cache, direct DB lookup every request.** Rejected: personal tokens hit on every MCP tool call (potentially many per session); even with SQLite read pool, this adds unnecessary load. 30s cache with `sync.Map` is simple and bounded.
- **C) Redis/external cache.** Rejected: trip2g is a single-process SQLite app. `sync.Map` is simpler and has zero operational overhead.
- **D) Eager cache invalidation on revoke.** Rejected: adds complexity for marginal benefit. 30s staleness is acceptable; revocation is rare.

**Why chosen:** Option A (extend `appreq`) with in-process cache gives transparent auth for all endpoints, minimal code surface, and acceptable performance. The cache-vs-live-lookup tradeoff is the principal tension: we accept 30s staleness on token data and 5-minute staleness on `last_used_at` in exchange for dramatically fewer DB round-trips.

**Consequences:**
- Revoked tokens remain valid for up to 30 seconds after revocation.
- `last_used_at` is approximate (5-minute granularity). Acceptable for analytics; not suitable for real-time audit.
- `sync.Map` memory is bounded by active token count (max 10/user, typically single-digit total).

**Follow-ups (phase 2):**
- Per-token scope (`mcp`, `read-only`, per-subgraph).
- Rate limiting per token.
- Request audit log (`user_token_logs` table).
- Configurable cache TTL if 30s proves too aggressive.
