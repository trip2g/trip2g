# MCP Federation — Stage 1 MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

Spec: `docs/dev/mcp_federation.md` (personal hub + private peers + adapters; minimal HMAC auth).
Existing system reused: `docs/dev/subgraph_payment.md` (`subgraphs`, `offers`, `offer_subgraphs`, `purchases`, `user_subgraph_accesses`, `canreadnote`).

---

## RALPLAN-DR Summary

### Principles (invariants this design must respect)

1. **Reuse existing subgraph ACL — don't fork it.** The `canreadnote.Resolve` decision tree is the only place that decides "may this user see this note". Federation supplies a *precomputed* allowed-subgraph set when JWT is present; it does not invent a parallel access model.
2. **Zero migration on the `subgraphs` table.** All new tables are additive. No `kb_url`, `kid`, or `name`-encoding columns on `subgraphs` (those are marketplace concerns).
3. **HMAC-only auth in MVP.** No mTLS, no asymmetric keys, no OAuth. Shared secret per pair, JWT HS256 with kid-in-header. Possession proves identity.
4. **The hub stores no remote content.** Only KB-notes (in the regular notes vault) for routing and `federation_secrets` for keys. All proxying is live; no replicated index.
5. **One auth surface for everything federated.** trip2g peers and external adapters (GitHub, Telegram) authenticate identically. The hub never special-cases adapter type.

### Decision Drivers (top 3)

1. **Shipping speed for personal use.** This goes live for one operator (me) plus 2–3 partners + 1 adapter. Anything beyond MVP is deferred — no premature multi-tenancy, no marketplace plumbing, no Federation Agreement Metadata, no audit table.
2. **Partner-private content gating.** Bob shares his `team-status` subgraph with me but nothing else. The mechanism must let Bob's hub pin "this kid sees only these subgraphs" without us inventing a new ACL surface.
3. **Zero-config onboarding for new bases.** Add a base = create one note with `mcp_federation_kb_url`. No `.mcp.json` edits, no system-config restart. Operator pastes the secret in admin and it's live.

### Viable Options for Major Choices

#### A. Auth mechanism (hub → peer)

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **HMAC-SHA256 JWT (chosen)** | Symmetric, both sides sign/verify with one key. Standard JWT shape. Constant-time compare easy. No PKI. | Both sides hold the same secret — compromise on either side compromises the channel. Out-of-band key distribution required. | **Chosen.** Pair-private trust model fits personal hubs perfectly. Stage 2/4 can layer asymmetric keys when marketplace appears. |
| RSA/Ed25519 JWT (asymmetric) | Hub holds private key, peers verify with published pubkey. One-to-many key reuse. | Requires JWKS endpoint or out-of-band pubkey transport. Larger payloads. Premature for 2–3 peers. | Rejected as MVP overkill. |
| Plain bearer tokens | Simplest possible. | Token-in-flight = token-on-disk. No expiry without server-side state. No `rid` correlation. | Rejected — JWT cost is trivial and gives `exp`/`rid` for free. |
| mTLS | Strongest. | Cert plumbing per peer; doesn't fit "Telegram chat handoff" onboarding. | Rejected. |

**Invalidation rationale** for asymmetric and mTLS: both add operational steps that defeat the "paste in admin and go" property.

#### B. KB-note discovery vs explicit registration

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **Frontmatter-driven KB-notes (chosen)** | Zero new admin form for adding bases. Notes are already editable in Obsidian. KB-notes auto-vectorize for semantic discovery. Per-user ACL via existing `subgraph` frontmatter for free. | Operator must remember the magic key names. KB-note rename ↔ id-stability is Obsidian's problem (already solved by `pid`). | **Chosen.** Aligns with trip2g's "everything is a note" stance and mirrors the existing `mcp_method` frontmatter precedent. |
| Admin-only registration (separate `federation_kbs` table) | Schema-explicit. Auditable. | Admin form required just to add a base. KBs invisible to local search. Duplicates "note + frontmatter" pattern that already works for `mcp_method`. | Rejected. |
| Hybrid (KB-note primary + override row) | Flexible. | Two sources of truth → bugs. Not needed for MVP scale. | Rejected, possibly Stage 3. |

#### C. Fan-out strategy

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **Goroutines + 2s per-call timeout, no cache (chosen)** | Simple. Bounded latency = max(2s, max(remote_latency)). Failures isolated. RRF merge already exists. | Cold queries always pay full RTT. | **Chosen.** Matches the spec; cache is a Stage-2 hardening once we see real latency. |
| Sequential calls | Trivial code. | Latency adds up — 4 peers × 1s each = 4s. | Rejected — would make federated_search unusable. |
| Goroutines + response cache `(kb_url, query) → results` for 30–60s | Repeat queries instant. Reuses `internal/cache`. | Stale-data window. Adds invalidation surface. Hides quietly-failing peers behind cached good results. | Rejected for MVP, listed as Stage 2. |

#### D. Testing strategy

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **`httptest.NewServer` fixture cluster (chosen)** | Real HTTP path including JSON marshaling, headers, JWT roundtrip. Fast (in-process). Each fixture node controls its own auth and content. | Test setup is verbose. Need to wire `app`-shaped Env for the fixture nodes. | **Chosen.** Required to exercise JWT + fan-out + timeouts realistically. |
| Pure unit tests with mocked HTTP client | Fast, no port allocation. | Doesn't cover header parsing, JSON edge cases, or real fan-out parallelism. | Use *in addition* for helpers (`splitKBID`, `prefixKBID`, `signOutbound`/`verifyInbound` HMAC paths) but not as the primary scenario suite. |
| Integration env (real bob/alice trip2g instances) | Highest fidelity. | Requires multiple SQLite DBs + worker orchestration. Slow. CI-hostile. | Rejected for MVP. Stage 3 if a topology bug demands it. |

**Final approach:** unit tests for helpers in `federation_helpers_test.go` + httptest-fixture suite for the 26 scenarios in `federation_test.go`.

---

## Implementation Steps

Each step has filename(s), a one-paragraph description, and concrete acceptance criteria.

### Step 1 — Schema migration

- **Files:** `db/migrations/20260427100000_create_federation_secrets.sql` (new).
- **Description:** Add `federation_secrets` (HMAC keys, with `kb_url IS NULL` ↔ inbound vs outbound) and `federation_secret_subgraphs` (per-kid scope). Use the SQL from the spec verbatim. Index on `kid` so inbound JWT verify is O(1) and a partial index on `kb_url` for outbound lookup. No changes to `subgraphs`. Match the migration framework used by `db/migrations/20260416132701_create_telegram_chat_usernames.sql` (the most recent neighbor).
- **Acceptance:**
  - Migration runs cleanly forward (and backward if neighbors define `Down`).
  - `pragma foreign_key_check` passes after migration on a populated DB.
  - `make sqlc` regenerates without diff noise outside the new struct/queries.
  - `db.FederationSecret` and `db.FederationSecretSubgraph` structs exist in `internal/db/models.go`.

### Step 2 — sqlc queries for federation

- **Files:** `internal/db/queries.read.sql` (extend), `internal/db/queries.write.sql` (extend), regenerated `internal/db/queries.read.sql.go` + `internal/db/queries.write.sql.go`.
- **Description:** Add named queries:
  - `FederationSecretByKBURL` — newest non-revoked outbound (`kb_url = ? AND revoked_at IS NULL ORDER BY created_at DESC LIMIT 1`).
  - `FederationSecretByKID` — inbound verify (`kid = ? AND kb_url IS NULL AND revoked_at IS NULL`).
  - `ListFederationSecrets` — admin list, joined with scope counts.
  - `InsertFederationSecret`.
  - `RevokeFederationSecret` — `UPDATE … SET revoked_at = current_timestamp WHERE id = ?`.
  - `ListFederationSecretSubgraphsByKID` — returns subgraph names (joined to `subgraphs`).
  - `InsertFederationSecretSubgraph`.
  - `DeleteFederationSecretSubgraph`.
  Run `make sqlc`.
- **Acceptance:**
  - All queries compile.
  - `FederationSecretByKBURL` returns the row with the most recent `created_at` among non-revoked rows for that URL (verified in scenario #24).
  - `FederationSecretByKID` returns `(secret, ok)` filtering on `kb_url IS NULL` and `revoked_at IS NULL`.

### Step 3 — KB-note frontmatter parsing

- **Files:**
  - `internal/model/mcp_federation_note.go` (new).
  - `internal/model/note.go` (add `MCPFederationKBURL`/`MCPFederationKBID`/`MCPFederationKBMaxDepth` fields on `NoteView`; populate inside the existing `RawMeta` parse block alongside `MCPMethod` at lines ~571–575).
  - `internal/model/noteviews.go` (or wherever `NoteViews` aggregations live — add `MCPFederationNotes []*MCPFederationNote` collection populated during finalization, parallel to existing `Subgraphs` extraction).
- **Description:** Define
  ```go
  type MCPFederationNote struct {
      Note     *NoteView
      URL      string
      ID       string
      MaxDepth int
  }
  func newMCPFederationNote(n *NoteView) *MCPFederationNote { ... }
  func hostnameFromURL(raw string) string { ... }
  ```
  ID falls back to `hostnameFromURL(URL)` when `mcp_federation_kb_id` is absent. `MaxDepth` defaults to 0 (leaf). Build the collection during note-finalization where `ExtractSubgraphs` runs.
- **Acceptance:**
  - Note with `mcp_federation_kb_url: https://bob.team.io/_system/mcp` and no `mcp_federation_kb_id` produces `MCPFederationNote{ID: "bob.team.io"}`.
  - `mcp_federation_kb_id: bob` overrides the slug.
  - `mcp_federation_kb_max_depth: 2` parsed as int; absent → 0; non-integer → 0 (with debug log, no panic).
  - Notes without the URL field are not in `MCPFederationNotes`.
  - `internal/model/mcp_federation_note_test.go` covers all four cases plus malformed URLs.

### Step 4 — Federation types + extended search payload

- **Files:** `internal/case/mcp/types.go`.
- **Description:** Add:
  ```go
  type FederatedSearchArguments struct {
      Query  string   `json:"query"`
      KBID   string   `json:"kb_id,omitempty"`
      KBIDs  []string `json:"kb_ids,omitempty"`
  }
  type FederatedSimilarArguments struct {
      KBID   string `json:"kb_id"`
      PID    int64  `json:"pid,omitempty"`
      NoteID int64  `json:"note_id,omitempty"`
      Path   string `json:"path,omitempty"`
      Href   string `json:"href,omitempty"`
      Limit  int    `json:"limit,omitempty"`
  }
  type FederatedNoteHTMLArguments struct {
      KBID    string `json:"kb_id"`
      PID     int64  `json:"pid,omitempty"`
      NoteID  int64  `json:"note_id,omitempty"`
      Path    string `json:"path,omitempty"`
      Href    string `json:"href,omitempty"`
      MatchID string `json:"match_id,omitempty"`
  }
  type FederationRef struct {
      KBID             string `json:"kb_id"`
      KBURL            string `json:"kb_url"`
      AgentInstruction string `json:"agent_instruction"`
  }
  type PayloadContext struct {
      KBInstructions map[string]string `json:"kb_instructions,omitempty"`
  }
  ```
  Extend `SearchResultItem` with `Federation *FederationRef \`json:"federation,omitempty"\``. The existing `URL` field already covers "URL on every result" — verify Step 6 sets it for proxied items too.
- **Acceptance:**
  - All new types compile; JSON-marshal with `omitempty` keeps existing client compatibility.
  - `SearchResultItem` zero-value JSON unchanged.

### Step 5 — Federation transport: signer, verifier, proxy, fan-out

- **Files:** `internal/case/mcp/federation.go` (new), `internal/case/mcp/federation_helpers.go` (new — pure helpers, easy to unit-test).
- **Description:**
  - `signOutbound(secret []byte, kid, iss, rid string) (string, error)` — HS256 JWT with `kid` in header, claims `{iss, iat, exp=+30s, rid}`. **Hand-roll with stdlib only** (`crypto/hmac`, `crypto/sha256`, `encoding/base64` URL-no-padding, `encoding/json`). ~40 LOC; no new dep. The minimal claim set means stdlib is genuinely simpler than pulling `golang-jwt`. Use `subtle.ConstantTimeCompare` for the verify path's signature byte compare.
  - `verifyInbound(ctx, env, jwt string) (kid string, allowedSubgraphs []string, err error)` — parse header, look up `federation_secrets` by kid, `subtle.ConstantTimeCompare` HMAC, validate `iat`/`exp` with 5s skew, then `ListFederationSecretSubgraphsByKID` for scope. Distinct sentinels: `ErrFedAuthUnknownKid`, `ErrFedAuthBadSig`, `ErrFedAuthExpired`, `ErrFedAuthRevoked`.
  - `proxyToKB(ctx, kbURL string, secret *db.FederationSecret, method string, args any, depth int) (json.RawMessage, error)` — `fasthttp.Client.DoTimeout` with 2s. Adds `Authorization: Bearer <jwt>` only if `secret != nil`. Sets `X-MCP-Federation-Depth: <depth+1>` header.
  - `fanout(ctx, kbs []*MCPFederationNote, method string, args any, depth int) []ProxiedResult` — `errgroup` with a 2s child context per call, collects `(kbID, raw|err)`. Logs each call via `mcp:federation` prefix.
  - `splitKBID(id string) (head, rest string)` — split on first `/`.
  - `prefixKBID(localSegment string, items []SearchResultItem)` — prepend `localSegment + "/"` to every item's `Federation.KBID` (where present).
- **Acceptance:**
  - `signOutbound` produces a JWT that round-trips through `verifyInbound`.
  - Bad signature, expired, future-iat-beyond-skew, revoked secret, unknown kid each return a distinct sentinel.
  - `fanout` over 3 fixtures responding after 1s each completes in ≤ 1.5s wall-time (scenario #17).
  - Helpers covered by `federation_helpers_test.go` table-driven tests.

### Step 6 — Federation handlers + KB-note marker in local search

- **Files:** `internal/case/mcp/resolve.go` (modify), `internal/case/mcp/federation_handlers.go` (new).
- **Description:**
  - `handleToolsList` returns 6 tools always (add `federated_search`, `federated_similar`, `federated_note_html` regardless of KB-note count). Each tool's `InputSchema` mirrors the local equivalent + `kb_id` (string, optional for `federated_search`, required for the other two) and `kb_ids` (array, only on `federated_search`).
  - `handleToolsCall` dispatches `federated_*` to new handlers.
  - `handleSearch`: when an item's underlying note has a non-empty `MCPFederationKBURL`, set `Kind = "federation_kb"` and populate `Federation` with `kb_id`, `kb_url`, and the spec's literal `agent_instruction` template. Don't filter these out at vector merge time — agent must see them.
  - `handleFederatedSearch(ctx, env, args)`:
    1. Compute `accessibleKBNotes(ctx, env, user)` (Step 7).
    2. If empty → return structured `{"status":"federation_not_configured"}` payload (not a JSON-RPC error).
    3. Resolve target list: `kb_id` (single, after `splitKBID`), `kb_ids` (intersection with accessible — silently drop inaccessible), or fan-out (all accessible).
    4. Run local `search` in parallel with proxied calls (only when fan-out — `kb_id` mode is purely remote).
    5. For each remote response: rewrite returned `kb_id` via `prefixKBID(localSegment, items)`.
    6. Merge with `mergeResults` (RRF). Drop `kind="federation_kb"` from the merged output.
    7. Per-base `kb_instructions` are **deferred to Stage 2**. MVP returns no `context.kb_instructions`; agents rely on hub-level `initialize` instructions + KB-note bodies surfaced in local `search`. Avoids stub-cache debt.
  - `handleFederatedSimilar` / `handleFederatedNoteHTML`: require `kb_id`; on multi-segment `kb_id`, strip head and forward `rest` as the proxied call's `kb_id` arg. Return result body verbatim (HTML for `note_html`, item list for `similar`).
- **Acceptance:**
  - `tools/list` returns exactly 6 tools regardless of KB-note count (scenario #1).
  - `search` over a vault containing a KB-note returns one result with `kind="federation_kb"` and full `federation` block (scenario #3).
  - `federated_search(kb_id="public-A")` proxies without `Authorization` header (scenario #4).
  - `federated_search` with no KB-notes returns the structured payload, not an error (scenario #2).
  - Multi-segment `kb_id` strips correctly outbound and rewrites correctly on response (scenarios #10, #12).

### Step 7 — Per-user accessible KB-notes filter

- **Files:** `internal/case/mcp/federation_acl.go` (new), `internal/case/mcp/resolve.go` (use it in `handleSearch` and `handleFederated*`).
- **Description:** `accessibleKBNotes(ctx, env, user) ([]*MCPFederationNote, error)` iterates `env.LatestNoteViews().MCPFederationNotes`, runs `canreadnote.Resolve(ctx, env, kb.Note)` per KB-note, returns the subset the operator may read. The function MAY cache per-request via `appreq` if the KB-note count grows; not required for MVP.
- **Acceptance:**
  - Guest user sees only KB-notes whose underlying note is `Free` or in a `require_signin`-free subgraph that the guest qualifies for (scenario #11).
  - Operator-admin sees all KB-notes.
  - Inaccessible `kb_id` argument to `federated_*` returns the structured "not configured for this kb_id" payload (does not leak the URL).

### Step 8 — Inbound JWT verification + scope enforcement

- **Files:** `internal/case/mcp/federation_handlers.go` (modify — keep auth-check inside the federated handlers, NOT in endpoint.go), `internal/case/canreadnote/resolve_with_subgraphs.go` (new sibling of `Resolve`), `internal/case/canreadnote/resolve_with_subgraphs_test.go` (new).
- **Description:**
  - **Don't modify `endpoint.go`** — it stays JSON-RPC-transport-only. Header reading is method-aware concern; do it in the federated handlers.
  - At the top of each `handleFederated*`: read `req.Req.Header.Peek("Authorization")`. If present and starts with `Bearer `, call `verifyInbound(ctx, env, jwt)`. On any `ErrFedAuth*` → return JSON-RPC error with code `-32401` mapped from sentinel; emit warn log per the spec's logging table. On success → keep `(kid, allowedSubgraphs)` in a local variable for downstream filtering. No need for context-stashing because handlers consume them in-place.
  - Add `canreadnote.ResolveWithSubgraphs(ctx, env ResolveWithSubgraphsEnv, note *NoteView, allowed []string) (bool, error)` — same logic tree as `Resolve` but uses `allowed` instead of `ListActiveUserSubgraphs`. The `Env` interface for the sibling drops `ListActiveUserSubgraphs` and `CurrentUserToken`; keeps `Logger` (if used). Federated handlers call this when JWT was verified; they call the existing `Resolve` (anonymous path) when JWT is absent.
- **Acceptance:**
  - Scenarios 5, 6, 18, 19, 20, 21, 22, 23 all pass.
  - Empty `allowed` list ⇒ caller sees only `note.Free` content, identical to anonymous behavior.
  - `Resolve` (existing, non-federated) is unchanged.

### Step 9 — App-layer wiring + outbound key lookup

- **Files:** `cmd/server/main.go` (modify — add Env satisfaction `_ mcp.Env = app`), `internal/case/mcp/resolve.go` (extend `Env` interface), `internal/appconfig/config.go` (modify).
- **Description:** Extend the MCP `Env` interface with **runtime-only** federation methods. Encryption/decryption is admin-side concern, not on `mcp.Env`.

  Added to `mcp.Env`:
  - `FederationSecretByKBURL(ctx, kbURL string) (db.FederationSecret, bool, error)` — `false, nil` when absent (public base).
  - `FederationSecretByKID(ctx, kid string) (db.FederationSecret, bool, error)`.
  - `ListFederationSecretSubgraphNamesByKID(ctx, kid string) ([]string, error)` — names not ids.
  - `DecryptData([]byte) ([]byte, error)` — needed at sign/verify time to unwrap `secret_crypt`. Encryption (`EncryptData`) lives only on the **admin** Env in Step 12 (where secrets are inserted).
  - `FederationMaxDepth() int` — reads `MCP_FEDERATION_MAX_DEPTH` (default 3).
  - `HTTPClient() *fasthttp.Client` (or shared pooled client per existing patterns).
  - `PublicURL() string` — already exists; used for `iss` and self-skip detection.

  Each admin use case in Step 12 declares its OWN narrow Env (Use Case pattern) — `EncryptData` lives on `createfederationsecret.Env`, not `mcp.Env`.

  In `internal/appconfig/config.go` add:
  ```go
  MCPFederationMaxDepth int `env:"MCP_FEDERATION_MAX_DEPTH" envDefault:"3"`
  ```
- **Acceptance:**
  - `var _ mcp.Env = app` compile-time check passes.
  - `MCP_FEDERATION_MAX_DEPTH` read once at startup, stored on `app`.
  - Outbound proxy chooses the newest non-revoked secret on duplicate-kb_url rows (scenario #24).

### Step 10 — Cycle protection (lightweight)

- **Files:** `internal/case/mcp/federation.go` (modify `fanout` + `proxyToKB`).
- **Description:** Two checks:
  - **Self-skip:** before fan-out, drop any KB-note whose URL equals `env.PublicURL() + "/_system/mcp"` (URL-normalize trailing slash).
  - **Depth cap:** propagate `X-MCP-Federation-Depth` header. Every outbound increments. Inbound reads it; if `incoming >= FederationMaxDepth()` the federated handler returns empty results immediately, with a single warn log. This is **not** in JWT claims — header-only for MVP, kept simple.
- **Acceptance:**
  - Hub never proxies to its own URL.
  - Two-hub cycle terminates at depth = 3 with a single warn log per terminator hub.

### Step 11 — Logging at `mcp:federation` prefix

- **Files:** All federation files use `logger.WithPrefix(env.Logger(), "mcp:federation")`.
- **Description:** Implement every event in the spec's logging table (request received, KB-note matched, fan-out start, per-base call start/done/failed, KB unreachable, kb_instructions miss/fetch, inbound auth failures with `unknown kid`/`bad signature`/`revoked secret`/`expired`, federation request done). Use structured fields (`method`, `kb_id`, `kb_url`, `latency_ms`, `results_count`, `rid`, `error`). Levels per the spec table.
- **Acceptance:**
  - Manual smoke test: stop one fixture mid-suite and observe per-base failure log with `latency_ms` ≈ 2000.
  - Log lines greppable via `mcp:federation` prefix.

### Step 12 — GraphQL admin mutations + queries

- **Files:**
  - `internal/graph/schema.graphqls` (extend `AdminMutation` + add admin federation fields).
  - `internal/case/admin/createfederationsecret/{resolve.go,mocks_test.go,resolve_test.go}` (new).
  - `internal/case/admin/revokefederationsecret/...` (new).
  - `internal/case/admin/addfederationsecretsubgraph/...` (new).
  - `internal/case/admin/removefederationsecretsubgraph/...` (new).
  - `internal/case/admin/listfederationsecrets/...` (new — read query for the admin UI).
- **Description:** Mirror the `creategithuboauthcredentials` pattern (ozzo-validation, `EncryptData` for the secret bytes, `db.IsUniqueViolation` mapping). Schema additions:
  - `mutation admin.createInboundFederationSecret(input: { kid: String!, description: String })` — server generates 32-byte random secret, encrypts, returns `{ id, kid, secretHex }`. The plaintext `secretHex` is shown **once** in the response (admin UI surfaces a copy button). After this response the bytes can never be retrieved again.
  - `mutation admin.createOutboundFederationSecret(input: { kid: String!, secretHex: String!, kbURL: String!, description: String })` — accepts pasted bytes from a peer (the inbound side already ran `createInboundFederationSecret` and gave you the kid+bytes). Validates hex length (= 32 bytes), encrypts, stores. Never echoes back the plaintext.
  - Two distinct mutations make intent obvious: "I'm publishing a key for someone to use against me" vs "I received someone's key, here it is".
  - `mutation admin.revokeFederationSecret(id: ID!)`.
  - `mutation admin.addFederationSecretSubgraph(kid: String!, subgraphID: ID!)`.
  - `mutation admin.removeFederationSecretSubgraph(kid: String!, subgraphID: ID!)`.
  - `query Admin { federationSecrets: [FederationSecret!]! }` — joins `federation_secret_subgraphs` for scope display. Never returns `secret_crypt` or any derived plaintext.

  Implementation detail: server-side generation uses `crypto/rand`. Hex-decode for the outbound mutation rejects anything not 32 bytes with `model.ErrorPayload` (validation error, not a 5xx).
- **Acceptance:**
  - `make gqlgen` runs cleanly.
  - Each resolver delegates to `admin/<pkg>/Resolve`.
  - `secretHex` decoded to bytes before encryption; raw secret bytes never returned by any read query.
  - Unit tests per package using `moq`-generated `mocks_test.go` and `testify/require`.

### Step 13 — $mol admin UI

- **Files:**
  - `assets/ui/admin/federation/federation.view.tree` (new).
  - `assets/ui/admin/federation/federation.view.ts` (new).
  - `assets/ui/admin/federation/federation.view.tree.locale=ru.json` (new).
  - `assets/ui/admin/admin.view.tree` (modify — add menu entry).
- **Description:** Three sub-widgets (separate components for clarity):
  - **List peers:** table of KB-notes joined with their `federation_secrets` row by URL. Status column: `public`, `linked`, `revoked`, `no secret`. Reuses `$trip2g_graphql_request` and `$trip2g_graphql_make_map`.
  - **Add peer (modal/form):** kid, secret bytes (hex), kb_url (optional — null = inbound), description. Submit → `createFederationSecret`.
  - **Manage scope (per-kid):** subgraph checkbox list. Toggle calls `addFederationSecretSubgraph` / `removeFederationSecretSubgraph` immediately (matches existing admin pattern).
  - **Revoke:** button per row → `revokeFederationSecret`. Greys out the row.
- **Acceptance:**
  - Manual click-through: paste kid + secret → see row appear → check 1–2 subgraphs → revoke → row goes grey.
  - Localized strings in en (default) and ru locale files.
  - `npm run build` succeeds.

### Step 14 — Tests (unit + scenario suite)

- **Files:**
  - `internal/case/mcp/federation_helpers_test.go` (new).
  - `internal/case/mcp/federation_test.go` (new — 26 scenarios).
  - `internal/case/mcp/mocks_test.go` (regenerate via `go generate`).
  - `internal/model/mcp_federation_note_test.go` (new — frontmatter parsing).
- **Description:** See "Test Strategy" below.
- **Acceptance:**
  - All 26 scenarios pass on `go test ./internal/case/mcp/...`.
  - `go test -race ./...` clean (fan-out is the riskiest area).
  - No flaky timeout-based scenarios — use deterministic `chan struct{}` synchronization where possible.

---

## Schema Migration Plan

`db/migrations/20260427100000_create_federation_secrets.sql`:

```sql
-- +goose Up
-- Trusted HMAC keys for federation. Same row pattern works for both inbound
-- (verify) and outbound (sign): kb_url IS NULL → inbound, NOT NULL → outbound.
CREATE TABLE federation_secrets (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  kid          TEXT    NOT NULL,
  secret_crypt BLOB    NOT NULL,
  kb_url       TEXT,
  description  TEXT,
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by   INTEGER NOT NULL REFERENCES admins(user_id) ON DELETE RESTRICT,
  revoked_at   DATETIME
);
CREATE INDEX idx_federation_secrets_kid    ON federation_secrets(kid);
CREATE INDEX idx_federation_secrets_kb_url ON federation_secrets(kb_url) WHERE kb_url IS NOT NULL;

-- Inbound scope. Each row: "kid X may surface my subgraph Y".
-- No rows → kid is anonymous-equivalent on this base.
CREATE TABLE federation_secret_subgraphs (
  kid          TEXT    NOT NULL,
  subgraph_id  INTEGER NOT NULL REFERENCES subgraphs(id) ON DELETE RESTRICT,
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by   INTEGER NOT NULL REFERENCES admins(user_id) ON DELETE RESTRICT,
  PRIMARY KEY (kid, subgraph_id)
);

-- +goose Down
DROP TABLE federation_secret_subgraphs;
DROP TABLE federation_secrets;
```

Verify migration directives match neighbors before committing.

---

## New Files to Create

| Path | Purpose |
|------|---------|
| `db/migrations/20260427100000_create_federation_secrets.sql` | Schema for `federation_secrets`, `federation_secret_subgraphs`. |
| `internal/model/mcp_federation_note.go` | `MCPFederationNote` wrapper + URL→hostname slug helper. |
| `internal/model/mcp_federation_note_test.go` | Frontmatter parsing unit tests. |
| `internal/case/mcp/federation.go` | Outbound proxy, fan-out orchestrator, key lookup. |
| `internal/case/mcp/federation_handlers.go` | `handleFederatedSearch/Similar/NoteHTML`. |
| `internal/case/mcp/federation_helpers.go` | `signOutbound`, `verifyInbound`, `splitKBID`, `prefixKBID`, `hostnameFromURL`. |
| `internal/case/mcp/federation_helpers_test.go` | Unit tests for helpers. |
| `internal/case/mcp/federation_test.go` | 26-scenario httptest fixture suite. |
| `internal/case/mcp/federation_acl.go` | `accessibleKBNotes` per-user filter wrapping `canreadnote.Resolve`. |
| `internal/case/canreadnote/resolve_with_subgraphs.go` | Sibling of `Resolve` taking precomputed allowed list. |
| `internal/case/canreadnote/resolve_with_subgraphs_test.go` | Unit tests for the sibling. |
| `internal/case/admin/createinboundfederationsecret/{resolve,mocks_test,resolve_test}.go` | Admin: generate inbound secret server-side, return plaintext once. |
| `internal/case/admin/createoutboundfederationsecret/{resolve,mocks_test,resolve_test}.go` | Admin: store outbound secret bytes received from peer. |
| `internal/case/admin/revokefederationsecret/{resolve,mocks_test,resolve_test}.go` | Admin: revoke secret. |
| `internal/case/admin/addfederationsecretsubgraph/{resolve,mocks_test,resolve_test}.go` | Admin: scope add. |
| `internal/case/admin/removefederationsecretsubgraph/{resolve,mocks_test,resolve_test}.go` | Admin: scope remove. |
| `internal/case/admin/listfederationsecrets/{resolve,mocks_test,resolve_test}.go` | Admin: list (read). |
| `assets/ui/admin/federation/federation.view.tree` | UI structure for federation admin page. |
| `assets/ui/admin/federation/federation.view.ts` | UI behavior (GraphQL queries/mutations). |
| `assets/ui/admin/federation/federation.view.tree.locale=ru.json` | Russian locale strings. |

## Files to Modify

| Path | Change |
|------|--------|
| `internal/model/note.go` | Add `MCPFederationKBURL`/`MCPFederationKBID`/`MCPFederationKBMaxDepth` fields on `NoteView`; populate from `RawMeta` in the same block as `MCPMethod` (lines ~571–575). |
| `internal/model/noteviews.go` (or wherever `NoteViews` is finalized) | Build `NoteViews.MCPFederationNotes []*MCPFederationNote` collection during finalize, parallel to `Subgraphs` extraction. |
| `internal/case/mcp/types.go` | Add `FederatedSearchArguments`, `FederatedSimilarArguments`, `FederatedNoteHTMLArguments`, `FederationRef`, `PayloadContext`. |
| `internal/case/mcp/resolve.go` | Add federated tools to `handleToolsList`; dispatch `federated_*` in `handleToolsCall`; mark KB-notes with `kind="federation_kb"` in `handleSearch`; extend `Env` interface with new methods (Step 9). |
| `internal/case/mcp/endpoint.go` | Read `Authorization` header on `federated_*` methods, call `verifyInbound`, stash `(kid, allowedSubgraphs)` on context, return 401 on failure. |
| `internal/db/queries.read.sql` + `queries.write.sql` | Add 8 new named queries (Step 2); regenerate Go via `make sqlc`. |
| `internal/graph/schema.graphqls` | Add 4 admin mutations + 1 admin query for federation secrets; `make gqlgen` regen. |
| `cmd/server/main.go` | Implement new Env methods on `app`; add `var _ mcp.Env = app` compile-time check; load `MCP_FEDERATION_MAX_DEPTH`. |
| `internal/appconfig/config.go` | Add `MCPFederationMaxDepth int` env-var binding (default 3). |
| `assets/ui/admin/admin.view.tree` | Add federation menu entry. |

---

## Test Strategy

**Files:**
- `internal/case/mcp/federation_helpers_test.go` — table-driven helper unit tests.
- `internal/case/mcp/federation_test.go` — 26 scenarios using `httptest.NewServer`.

**Fixture topology:**

```
                      hub (system under test)
        ┌─────────────┼─────────────┬─────────────┐
        │             │             │             │
   public-A       peer-B        peer-C       adapter-X
   (no auth)    (kid=alice)   (kid=alice)  (kid=alice-gh)
                scope=[team]  scope=[]     scope=[repo1]
```

Each fixture node = `httptest.NewServer` with a tiny handler that:
- Verifies JWT (if it requires auth) using a test secret.
- Returns canned `tools/call` results matching the requested method.
- Optionally delays N ms (for parallelism / timeout tests).

The hub's Env is a stubbed `mcp.Env` impl wired with: a fake notes index containing 4 KB-notes (one per fixture node), the test-secret table, and a default `fasthttp.Client`.

**Scenario list (mapped 1:1 from spec):**

| # | Scenario | Step exercised |
|---|----------|----------------|
| 1 | `tools/list` always returns 6 methods, independent of KB-note count | 6 |
| 2 | `federated_search` with no KB-notes → "federation not configured" payload, no error | 6 |
| 3 | `search` finds a KB-note → result has `kind="federation_kb"` + full `federation` block | 6 |
| 4 | `federated_search(kb_id="public-A")` → no Authorization header, results returned | 5,6 |
| 5 | `federated_search(kb_id="peer-B")` with valid scope → JWT signed, base verifies, public+team-scope content | 5,6,8 |
| 6 | `federated_search(kb_id="peer-C")` with empty scope → public layer only | 5,6,8 |
| 7 | `federated_search` fan-out → hits all 4, RRF merge, KB-notes excluded | 6 |
| 8 | `federated_search(kb_ids=["peer-B","adapter-X"])` → exactly those two; results carry their kb_id | 6 |
| 9 | `federated_note_html(pid, kb_id="peer-B")` → HTML returned via proxy | 6 |
| 10 | Targeted call with `kb_id="science/cellbio"` → hub strips first segment, proxies rest | 5,6 |
| 11 | Inaccessible KB-note (operator subgraph ACL) → hidden in `search`, federated returns "not configured" | 7 |
| 12 | Reverse prefix rewriting → sub-hub returns `kb_id="X"`, parent rewrites to `sub/X` | 5,6 |
| 13 | URL on every result → URLs point at real remote hosts, not the hub | 4,6 |
| 14 | One sub-base errors → other results returned; warn logged | 5,11 |
| 15 | One sub-base times out (>2s) → same as #14, latency check | 5,11 |
| 16 | Empty results from a sub-base → merge handles it without panic | 5 |
| 17 | Fan-out parallelism → 3 servers, 1s each, total ≤ 1.5s | 5 |
| 18 | Unknown kid → 401 + warn-level audit log | 5,8 |
| 19 | Wrong signature → 401 | 5,8 |
| 20 | Expired JWT → 401 | 5,8 |
| 21 | Revoked secret → 401 | 5,8 |
| 22 | No JWT on private peer → anonymous, public layer only | 8 |
| 23 | Public-A with JWT → JWT ignored, public layer returned | 8 |
| 24 | Hub picks newest secret on outbound (two non-revoked rows for same kb_url) | 2,5 |
| 25 | Constant-time HMAC compare → **code review only** (grep for `subtle.ConstantTimeCompare` in `verifyInbound`'s signature compare path); runtime timing tests are flaky and removed | 5 |
| 26 | (removed in MVP — `kb_instructions` cache deferred to Stage 2; no test in MVP suite) | — |

**Helper unit tests (`federation_helpers_test.go`):**

- `splitKBID("a") == ("a","")`, `splitKBID("a/b/c") == ("a","b/c")`, `splitKBID("") == ("","")`.
- `prefixKBID` mutates `Federation.KBID` only; leaves other fields intact.
- `signOutbound` + `verifyInbound` round-trip for valid + invalid keys (table-driven).
- `hostnameFromURL("https://bob.team.io/_system/mcp") == "bob.team.io"`; malformed URL → empty string + warn.

---

## Acceptance Criteria (whole MVP)

The MVP is done when **all** of the following hold simultaneously:

1. **Schema:** Both new tables exist, migrations apply cleanly, `pragma foreign_key_check` clean.
2. **Tools/list contract:** `tools/list` returns exactly 6 methods on a hub with 0 KB-notes and on a hub with 5 KB-notes. Identical schema.
3. **Search marker:** A `search` query that surfaces a KB-note returns it with `kind="federation_kb"` + `federation.kb_id|kb_url|agent_instruction`. KB-notes never appear in `federated_search` responses.
4. **Public proxy:** `federated_search(kb_id="X")` against a KB-note with no `federation_secrets` row makes one outbound HTTP call **without** an `Authorization` header and returns the remote's results.
5. **Authenticated proxy:** Same call against a KB-note with a secret sends `Authorization: Bearer <jwt>` with valid HS256 signature, kid in header, 30s exp, 5s skew. Remote's verifier accepts; hub returns the filtered result set.
6. **Inbound auth:** Hub's own `/_system/mcp` endpoint, on a `federated_*` call with valid kid, applies the kid's allowed-subgraph set via `canreadnote.ResolveWithSubgraphs` and returns only matching notes. Without auth, returns only `Free` notes.
7. **Per-user ACL:** Operator with limited subgraph access sees fewer KB-notes in `search` and fewer accessible `kb_id` targets — `canreadnote.Resolve` decides.
8. **Fan-out:** `federated_search` with no `kb_id` queries every accessible KB-note in parallel (max 2s per call), merges via RRF, returns within ~2.1s even when one peer hangs.
9. **Cycle protection:** Hub never proxies to its own URL. `MCP_FEDERATION_MAX_DEPTH` (default 3) caps recursion at the configured depth via the `X-MCP-Federation-Depth` header.
10. **kb_id rewriting:** Targeted calls with multi-segment `kb_id` strip head; responses from sub-hubs have their `kb_id`s prefixed with the local segment.
11. **Logging:** Every event from the spec's logging table emitted at the documented level, fields populated, prefix `mcp:federation`.
12. **Admin UI:** Operator can paste kid+secret in the admin, attach subgraph scope via checkboxes, revoke. All operations succeed end-to-end against live SQLite.
13. **Tests:** All 26 scenarios + helper unit tests pass on `go test -race ./...`.
14. **No-regression:** Existing local `search`/`similar`/`note_html` behaviour unchanged for non-federated traffic. Existing MCP tests still pass.

---

## ADR — Architecture Decision Record

### Decision

Implement Stage 1 of MCP federation as **two new additive tables** (`federation_secrets`, `federation_secret_subgraphs`) + **frontmatter-driven KB-notes** (`mcp_federation_kb_url|id|max_depth`) + **HMAC-SHA256 JWT auth** (HS256, kid in header, claims `iss/iat/exp/rid`) + **goroutine fan-out with 2s per-call timeouts**. Reuse `canreadnote.Resolve` for both KB-note visibility and inbound result filtering (via a precomputed-subgraph sibling, `ResolveWithSubgraphs`). Build admin UI in $mol mirroring `creategithuboauthcredentials`.

### Drivers

1. Personal-hub use case dominates: 1 operator + 2–3 partners + 1 adapter. Anything multi-tenant or marketplace-shaped is overkill.
2. Operator already manages content via Obsidian — frontmatter is the natural extension surface.
3. trip2g's existing subgraph ACL is the single source of truth for "may X read Y"; adding a parallel system would be a maintenance trap.
4. HMAC + shared secret matches the out-of-band onboarding step ("Bob sends Alice a kid+bytes via Telegram") with no PKI ceremony.

### Alternatives Considered (and rejected)

- **Marketplace-mode (`docs/dev/mcp_federation_marketplace.md` design):** Adds `subgraphs.kb_url` column, kid-prefix encoding on subgraph names, `sgr/sub/ver` JWT claims, `/_system/mcp/federation` discovery endpoint, audit table, HMAC-pseudonym `sub`, 2-secret rotation per kid. Defers to Stage 4 / future work. Not blocking personal use.
- **Federation Agreement Metadata:** Structured fields such as `consent`, `use_policy`, attribution requirements, and commercial terms are intentionally deferred for the demo. Keep those rules as prose in the KB-note body until the basic federation loop is proven.
- **Asymmetric (RSA/Ed25519) JWTs:** Stronger compromise model but requires JWKS or pubkey transport. Premature for 2–3 peers.
- **Admin-managed federation_kbs table (no frontmatter):** Schema-cleaner but loses zero-config onboarding. Loses semantic discovery of bases via search.
- **Sequential fan-out:** Adds peer count × RTT to every fan-out query.
- **`__visited` array in proxied requests:** Not needed yet; depth cap + self-skip suffice. Stage 2 hardening.
- **Audit log table:** Application logs cover the operational view. Add only if a real compliance requirement appears.
- **Phase 1B `mcp_tokens` dependency:** Single-user personal hub doesn't need per-user MCP tokens. Couple later if multi-user-via-hub appears.
- **2-secret rotation per kid:** Operator pre-creates a new kid, sends out, revokes old. Simpler than overlapping-secrets rotation.

### Why Chosen

The chosen design is **the smallest set of moving parts that delivers the spec's goals** and integrates with existing trip2g primitives:
- Two tables, both additive, both ON DELETE RESTRICT to `admins`/`subgraphs` so accidental deletes can't dangle keys.
- Reuses `canreadnote.Resolve` (and one new sibling) — no parallel ACL.
- Reuses `EncryptData/DecryptData` already proven on GitHub OAuth and Telegram credentials.
- Reuses `mergeResults` (RRF) for fan-out merge — no new ranking code.
- Reuses the `mcp_method` frontmatter precedent — no new note-class concept.
- Admin UI follows the established `creategithuboauthcredentials` ozzo + EncryptData pattern.

### Consequences

**Good:**
- Onboarding a new base = 1 note + 1 admin paste. Sub-minute operation.
- Public bases need zero auth setup.
- Failures are isolated (per-call timeouts) and observable (`mcp:federation` logs).
- Scope changes are immediate — no token reissue.
- Future Stage 2/3/4 can layer on without breaking MVP contracts.

**Bad / accepted trade-offs:**
- HMAC compromise on either side compromises the channel. Mitigated by per-pair secrets + revocation.
- No replay cache (`jti`) — the 30s `exp` is the only replay protection. Acceptable at this scale.
- No per-peer rate limiting. A misbehaving peer can hammer the hub. Acceptable for personal use; revisit if exposed publicly.
- **Depth header (`X-MCP-Federation-Depth`) is unsigned.** A malicious peer can decrement it to keep the depth-cap from firing, then amplify a request graph. In personal trust mode (everyone in the federation is friendly) this is acceptable. If federation is later opened to less-trusted peers, move depth into the JWT claims (signed) and verify on every hop.
- No automated cycle visited-tracking — depth cap only. Two-base loop terminates at depth 3. Real cycles in production will need Stage 2 hardening.
- TLS not enforced — operator's responsibility. Document but don't block.

### Follow-ups (Stage 2 / 3 / 4)

- **Stage 2:** `__visited` array for cycle detection; per-peer response cache (30–60s); health metrics; out-of-scope warnings round-trip; `kb_instructions` 5-min cache promoted from "stub" to real LRU.
- **Stage 3:** `/_system/mcp/federation` discovery endpoint; discovery refresh cron; mesh visualization; first-party Go adapter skeleton; cron health check; system-reminder enrichment ("you have N federations available").
- **Stage 4:** Marketplace mode if a real reseller use case appears. Adds `subgraphs.kb_url` (or kid-prefix encoded names) — **this is a real schema migration on a hot table, not a pure addition** — expanded JWT claims (`sgr/sub/ver`), audit table, HMAC-pseudonym sub, 2-secret rotation per kid. The other extensions (audit table, JWT claims, rotation) are additive; the subgraphs schema change is not. Plan for downtime/blue-green when Stage 4 lands.

---

## Sequencing & Dependencies

```
1. Schema migration (Step 1)
        │
        ▼
2. sqlc regen (Step 2)
        │
        ├──────────────┐
        ▼              ▼
3. Note model   4. Federation types
   (Step 3)        (Step 4)
        │              │
        └──────┬───────┘
               ▼
5. Transport: signer/verifier/proxy/fan-out (Step 5)
               │
               ▼
6. Federation handlers + KB marker (Step 6)
               │
               ├──────────────┐
               ▼              ▼
7. ACL filter         8. Inbound auth
   (Step 7)              (Step 8)
               │              │
               └──────┬───────┘
                      ▼
9. App-layer wiring (Step 9)
                      │
                      ▼
10. Cycle protection (Step 10)
                      │
                      ▼
11. Logging pass (Step 11)        ← can run lazily but easiest to do here
                      │
                      ▼
12. GraphQL admin (Step 12)        ← needs Steps 1, 2, 9
                      │
                      ▼
13. $mol admin UI (Step 13)        ← needs Step 12 schema regen
                      │
                      ▼
14. Tests (Step 14)                ← incremental as steps land; full suite at the end
```

**Hard ordering:**
- Schema → sqlc regen → Go code (Steps 1 → 2 → everything else).
- Federation types (Step 4) → Transport (Step 5) → Handlers (Step 6).
- Env extension (Step 9) gates Steps 5, 6, 12 from compiling — land it as soon as the method list is stable.
- GraphQL schema (Step 12) regen → Admin UI (Step 13) compile.

**Soft ordering:** Step 14 (tests) runs incrementally with each step's acceptance criteria; the full 26-scenario suite is the final gate.

---

## Estimated Effort

| Step | Effort | Notes |
|------|--------|-------|
| 1. Schema migration | 0.5 h | Spec gives the SQL verbatim. |
| 2. sqlc queries | 1.5 h | 8 queries + regen + verify. |
| 3. KB-note frontmatter | 2 h | Includes hostname slug edge cases + unit tests. |
| 4. Federation types | 1 h | Mostly mechanical. |
| 5. Transport (sign/verify/proxy/fan-out) | 1 d | Hand-rolled JWT + careful error model + concurrency. |
| 6. Federation handlers + KB marker | 1 d | Handler trio + kb_instructions stub + RRF integration. |
| 7. accessibleKBNotes filter | 2 h | Direct reuse of canreadnote.Resolve. |
| 8. Inbound auth + ResolveWithSubgraphs | 4 h | Sibling function + endpoint hook + 401 mapping. |
| 9. App-layer wiring | 3 h | Env method impls, env var, compile-time checks. |
| 10. Cycle protection | 1.5 h | Self-skip + depth header. |
| 11. Logging pass | 1.5 h | Already piecemeal; consolidate and verify table coverage. |
| 12. GraphQL admin | 4 h | 4 mutations + 1 query, ozzo validation, gqlgen. |
| 13. $mol admin UI | 1 d | List + add + scope + revoke widgets, en/ru locales. |
| 14. Tests | 1.5 d | 26 scenarios + helper units; httptest fixture wiring is the bulk. |

**Total:** ~6–7 working days for one engineer landing iteratively, ~3–4 days with parallel work on backend (Steps 5–11) and frontend (Step 13) once Step 12 is in.

---

## Open Questions (carried from spec, not blocking MVP)

- **Multiple users per hub.** Per-user federation ACL via `canreadnote` is in place from day one, but session resolution at the MCP endpoint depends on Phase 1B `mcp_tokens` (`docs/superpowers/specs/2026-03-27-ai-chat-design.md`). Out of scope for this plan.
- **TLS enforcement.** Whether to refuse non-HTTPS peer URLs in production. Document recommendation; don't enforce.
- **Per-channel rate limiting.** Not needed for personal use; revisit if hub exposed publicly.
- **`kb_instructions` cache promotion.** MVP may stub the 5-min cache as "always fetch" if implementation pressure requires; flag clearly so Stage 2 picks it up.
- **JWT library choice.** Hand-roll HS256 with stdlib, or import `github.com/golang-jwt/jwt/v5`? Defer to first commit on Step 5 — check `go.mod` first; prefer no new dep.
- **`__visited` array timing.** Depth cap suffices for MVP per spec, but an early Stage-2 trigger may be warranted if even two real peers cross-reference. Watch the warn logs after rollout.
