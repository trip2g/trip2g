# MCP Federation — Personal Router Across Knowledge Bases

**Use case.** I run my own trip2g instance, pointed at a single MCP endpoint from my agent's `.mcp.json`. From that one endpoint I want to reach:

1. **Public bases** — philosophers, courses, OSS docs that I curate as references.
2. **Private peer bases** — partners on my team running their own trip2g, sharing a private "work-status" subgraph with me but keeping the rest closed.
3. **External-system adapters** — small MCP-speaking services that front GitHub repos, Telegram channels, etc. From my hub's POV they look like another base.
4. **My own local content** — notes I write directly in trip2g, mixed in with everything else.

I write skills on top of the agent that compose all of this into team workflows: "what's everyone working on this week", "what did we decide in the design call", "find me references from my philosophy bases that apply to this engineering decision". One router, one auth surface, one MCP endpoint.

**Out of scope for this document.** Reselling someone else's content with a markup, charging end users, multi-tenant hubs that serve many users with different access. That track is preserved as `docs/dev/mcp_federation_marketplace.md` (design only) and revisited later if real demand appears. This doc covers what I need to ship now to wire myself + my partners + adapters together.

---

## Goals

1. **Zero-config growth.** Add a knowledge base by creating one note with `mcp_federation_kb_url:` in its frontmatter. No edits to `.mcp.json` or system config.
2. **Discovery via search.** The agent finds out a base exists by hitting it as a normal `search` result — KB-notes surface alongside content with a special `kind="federation_kb"` marker.
3. **Transparent proxying.** `federated_note_html` returns a remote note's content as if it were local. The agent sees a uniform interface across all bases.
4. **Minimal auth.** Public bases need no auth at all. Private peer bases need only a shared HMAC secret per pair, plus a per-secret list of allowed subgraphs.
5. **Adapter parity.** A GitHub or Telegram adapter is just another MCP server speaking the same protocol; nothing about federation distinguishes them from a trip2g base.

## Non-goals

- No commercial reselling, end-user billing, or revenue reconciliation across operators.
- No multi-tenant hub serving many users — each operator runs their own hub.
- No automated topology discovery — peers share URL/secret out-of-band (Telegram chat is fine).
- No replicated index across bases — the hub stores no data, only proxies.

---

## Architecture

```
                                    ┌──────────────────────────┐
                                    │ My agent (Claude/Cursor) │
                                    │  .mcp.json: my hub only  │
                                    └────────────┬─────────────┘
                                                 │
                                  ┌──────────────▼──────────────┐
                                  │  My trip2g hub              │
                                  │  /_system/mcp               │
                                  │   - my own notes            │
                                  │   - KB-notes pointing at:   │
                                  │       • public bases        │
                                  │       • private peers       │
                                  │       • adapters            │
                                  └─┬────────┬─────────┬────────┘
                                    │        │         │
              ┌─────────────────────┘        │         └────────────────────┐
              │                              │                              │
       ┌──────▼───────┐              ┌───────▼────────┐            ┌────────▼────────┐
       │ Public base  │              │ Partner peer   │            │ Adapter         │
       │ (no auth):   │              │ (HMAC secret): │            │ (HMAC secret):  │
       │ philosophers │              │ Bob's trip2g   │            │ GitHub repo     │
       │ courses      │              │ Charlie's …    │            │ Telegram channel│
       └──────────────┘              └────────────────┘            └─────────────────┘
```

The hub holds nothing remote. It holds my own content plus a small pile of metadata: KB-notes for routing, federation secrets for peer auth.

---

## KB-note frontmatter

```yaml
---
mcp_federation_kb_url: https://bob.team.io/_system/mcp
mcp_federation_kb_id: bob                  # optional, defaults to URL hostname
mcp_federation_kb_max_depth: 0             # optional, 0 (default) = leaf, 1+ = sub-hub
---
Use when: Bob's work-status updates, our shared design notes.
Don't use when: personal stuff, anything you wouldn't ask Bob in person.
```

| Field | Required | Purpose |
|-------|----------|---------|
| `mcp_federation_kb_url` | yes | Full endpoint URL of the remote MCP server. The marker that "this note is a KB-note". |
| `mcp_federation_kb_id` | no | Slug used inside `kb_id`. Defaults to URL hostname. |
| `mcp_federation_kb_max_depth` | no | How many federation levels are allowed inside this base if queried. `0` = leaf. |

**Note body** = "when to use this base", semantic guidance for the agent (vectorized like other content).

**No secret in frontmatter.** Federation secrets live in the `federation_secrets` table and are linked to a KB-note by URL.

---

## kb_id and hierarchy

`kb_id` is a slash-separated path:

```
bob                    # one of my peers
science/cellbio        # cellbio reachable through a science-hub
github/trip2g          # the github adapter, repo "trip2g"
```

When a sub-hub returns results to its parent, the parent **rewrites every kb_id** by prepending its own segment. When the parent receives a targeted call back, it strips the first segment and proxies the rest. This works recursively for arbitrary depth.

For a personal hub with mostly leaf bases, you'll rarely see hierarchies deeper than 1 — but the protocol supports them.

---

## MCP methods

`tools/list` always returns six methods. Stable contract regardless of what's connected.

```
search(query)                       — local search (FTS + vector) on hub's own content
similar(note_id|pid)                — similar to a local note
note_html(note_id|pid)              — HTML of a local note

federated_search(query, kb_id?, kb_ids?)
federated_similar(note_id, kb_id)
federated_note_html(note_id, kb_id, match_id?)
```

The federated trio mirrors the local trio. `kb_id` argument selects a single base; `kb_ids` selects multiple; absence in `federated_search` means full fan-out. `federated_similar` and `federated_note_html` require `kb_id`.

If no KB-notes are configured, federated methods return a structured "federation not configured" payload — they don't error.

---

## Instructions propagation

Three layers, each populated only when relevant:

1. **Hub-level (returned by `initialize`):** the hub's own `mcp_method: initialize` note describes federation usage in general terms — when to fan out, how to use kb_id. Doesn't enumerate bases.
2. **KB-note body (surfaced in `search` results):** the agent learns what each base is for from its body when a topic-relevant search surfaces it.
3. **Per-base instructions (returned with federated_search responses):** each remote base may have its own `initialize.instructions` that the hub fetches lazily on first call (cached 5 min) and includes inline in the response payload as `context.kb_instructions`.

The agent assembles context as it queries, no upfront dump.

---

## KB-note in search results

```json
{
  "title": "Bob's KB",
  "note_id": 17,
  "note_path": "team/bob.md",
  "kind": "federation_kb",
  "url": "https://hub.local/team/bob",
  "score": 0.71,
  "snippet": "Use when: Bob's work-status updates, our shared design notes.",
  "federation": {
    "kb_id": "bob",
    "kb_url": "https://bob.team.io/_system/mcp",
    "agent_instruction": "This is a knowledge base pointer. To search inside it, call federated_search with kb_id=\"bob\". To open notes from it, call federated_note_html(note_id=..., kb_id=\"bob\")."
  }
}
```

- `kind: "federation_kb"` — distinct from `note`, `index`, `source`.
- `federation.agent_instruction` — direct human-readable cue for the agent (more reliable than metadata inference).
- KB-notes are **excluded** from `federated_search` responses — fan-out already covers the destination.

---

## URL on every result

Every search result, federated or local, carries a full `url` field — agents use this for citations. For proxied results the URL is on the **real remote host**, not the hub. Otherwise links 404 when the user clicks.

---

## KB-note visibility (per-user routing ACL)

A KB-note is a regular note — it can carry `subgraphs:` in frontmatter. Federation reuses the existing `canreadnote.Resolve` decision: if the operator can read the KB-note, they can route through it.

In the personal-hub case, the operator is the only user, so this rarely matters. It becomes useful if you later let a colleague hit your hub directly — they only see KB-notes whose subgraphs they have local access to.

A per-request filter `accessibleKBNotes(ctx, env, user)` runs `canreadnote.Resolve` over the global KB-note index and feeds:
- KB-note exclusion in `search` for inaccessible ones.
- Fan-out target list.
- `kb_id` resolution in targeted calls (inaccessible kb_id → "federation not configured for this kb_id").

---

## Fan-out search

`federated_search` without `kb_id`/`kb_ids`:

1. Compute `accessibleKBNotes(ctx, env, user)`.
2. Local `search` runs in parallel with sub-base calls.
3. Fan out via goroutines to each KB-note's URL, query unchanged.
4. Each remote responds with its own results plus optional `kb_instructions`.
5. Merge via RRF (reuse `mergeResults`); KB-notes excluded.
6. Each result has correct `url` and `kb_id` (with prefix rewriting).

**Parameters:**
- Per-call timeout: 2 seconds per remote.
- Failed/timed-out remote: log warn, continue. Try every time on next call (no failure-suppression cache yet).
- Optional response cache `(kb_url, query)` for 30-60s, reusing `internal/cache`.

`federated_search` with `kb_ids: [a, b, c]` is the same path limited to those ids; inaccessible ids are silently dropped.

---

## Cycle handling

**Targeted mode (`kb_id` / `kb_ids`)** — cycles structurally impossible. Each hop strips one segment off `kb_id`; chain only shortens.

**Fan-out mode** — global env `MCP_FEDERATION_MAX_DEPTH` (default `3`) caps recursion depth. Self-skip: hub ignores its own URL among KB-notes. That's enough for this scale.

A blind fan-out (no `kb_id`/`kb_ids`) is bounded by three knobs:

- `FEDERATED_FANOUT_LIMIT` (default `7`) — max peers a blind fan-out touches, in registration order. Peers beyond the limit are returned in the response's `skipped` list (reason `fanout_limit`) so the caller can query them directly with `kb_id`; they are never dropped silently. Explicit `kb_id`/`kb_ids` calls are not capped.
- `FEDERATED_FANOUT_CONCURRENCY` (default `5`) — max peers queried in parallel.
- `FEDERATED_FANOUT_TIMEOUT` (default `5s`) — per-peer timeout for the whole call (client build + request); a hung peer is reported in `errors[]` and never blocks the others.

If real cycles ever bite (10+ peers actively cross-referencing), add `__visited` array to proxied requests as a Stage 2 hardening. Not in MVP.

---

## Logging

Federation events use the `mcp:federation` logger prefix.

| Event | Level | Fields |
|-------|-------|--------|
| Federation request received | debug | method, kb_id, kb_ids, query, visited_count |
| KB-note matched in local search | debug | kb_id, kb_url, score |
| Fan-out start | debug | targets, query |
| Per-base proxy call start | debug | kb_id, kb_url, method |
| Per-base proxy call done | debug | kb_id, latency_ms, results_count |
| Per-base proxy call failed | warn | kb_id, kb_url, error, latency_ms |
| KB-note URL unreachable | warn | kb_id, kb_url, error |
| `kb_instructions` cache miss / fetch | debug | kb_id, latency_ms |
| Inbound auth: unknown kid | warn | kid, iss, rid |
| Inbound auth: bad signature | warn | kid, iss, rid |
| Inbound auth: revoked secret | warn | kid, iss, rid |
| Federation request done | debug | method, total_latency_ms, total_results |

No audit table in MVP — application logs cover the operational view; structured fields make grepping easy.

---

## Authorization

Federation works transparently with personal user tokens. Any MCP call authenticated via a personal token inherits the user's access control — the personal token resolver in `appreq.Request.UserToken()` establishes user identity, and then `canreadnote.Resolve` and `accessibleKBNotes` apply the same ACL logic. For details on personal tokens, see [[en/user/mcp#personal-access-tokens]].

Federation also needs three distinct auth profiles for peer-to-peer federation secrets:

1. **Public base** — no auth. Hub calls anonymously, base returns its public layer.
2. **Private peer** — shared HMAC secret per pair, optional subgraph scope.
3. **Adapter** — same as private peer (HMAC + scope), implementation lives outside trip2g but speaks the same protocol.

### Trust model

A federation **secret** is an HMAC-SHA256 key shared between two peers (or between a hub and an adapter). The base side stores it in `federation_secrets` and links it to a list of allowed subgraphs in `federation_secret_subgraphs`. The hub side stores the same secret to sign outbound JWTs to that base.

If `federation_secret_subgraphs` has no rows for a given kid, that secret authorizes only the **public layer** of the base (everything `canreadnote.Resolve` approves for a guest). Adding a row opens up one more subgraph. No row = no privilege beyond anonymous; the secret never grants less than public access.

### Schema

```sql
-- Trusted HMAC keys for federation. Same row pattern works for both inbound
-- (verify) and outbound (sign): kb_url IS NULL → inbound, NOT NULL → outbound.
create table federation_secrets (
  id           integer primary key autoincrement,
  kid          text not null,                       -- short id placed in JWT header
  secret_crypt blob not null,                       -- HMAC bytes, encrypted at rest
  kb_url       text,                                -- non-null = OUTBOUND (sign for this URL)
                                                    --     null = INBOUND  (verify only)
  description  text,
  created_at   datetime not null default current_timestamp,
  created_by   integer not null references admins(user_id) on delete restrict,
  revoked_at   datetime
);
create index idx_federation_secrets_kid on federation_secrets(kid);

-- Inbound scope. Each row: "kid X may surface my subgraph Y".
-- No rows → kid is anonymous-equivalent on this base.
create table federation_secret_subgraphs (
  kid          text not null,
  subgraph_id  integer not null references subgraphs(id) on delete restrict,
  created_at   datetime not null default current_timestamp,
  created_by   integer not null references admins(user_id) on delete restrict,
  primary key (kid, subgraph_id)
);
```

`subgraphs` table is unchanged. No `kb_url` column, no kid-prefix encoding — that's marketplace-mode complexity, not needed here. The hub just stores secrets and forwards calls; nothing on the hub side maps remote subgraphs to local rows because we're not selling them.

### JWT

Header: `{"alg": "HS256", "kid": "<federation_secrets.kid>", "typ": "JWT"}`

Claims:
```json
{
  "iss": "https://my.hub.local/_system/mcp",
  "iat": 1234567890,
  "exp": 1234567920,           // 30s TTL, 5s skew
  "rid": "req-abc-xyz"          // request id for log correlation
}
```

Minimal. The kid + signature ARE the identity assertion; the secret was given to a specific peer out-of-band, so possession proves trust. No `sub`, `sgr`, or `ver` in MVP.

### What the hub does on outbound

```
For a federated_* call on KB-note with mcp_federation_kb_url = U:
  secret = SELECT * FROM federation_secrets
           WHERE kb_url = U AND revoked_at IS NULL
           ORDER BY created_at DESC LIMIT 1
  if secret is null:
    proxy U without Authorization header (public base)
  else:
    jwt = sign({iss, iat, exp=+30s, rid}, decrypt(secret.secret_crypt), kid=secret.kid)
    proxy U with Authorization: Bearer <jwt>
```

### What the base does on inbound

```
1. Read Authorization header.
2. If missing: treat as anonymous → only public layer (canreadnote with guest token).
3. Else parse JWT, look up federation_secrets by kid (kb_url IS NULL, revoked_at IS NULL).
4. Verify HMAC with constant-time compare. Bad signature → 401.
5. Check exp/iat with 5s skew. Expired → 401.
6. allowed = SELECT subgraph_id FROM federation_secret_subgraphs WHERE kid = ?
7. Filter result set: notes that pass canreadnote-with-guest-token
                       ∪ notes whose subgraphs ∩ allowed ≠ ∅.
8. Return.
```

The same `canreadnote` logic (intersection on subgraph names) is reused — federation just substitutes a precomputed `allowed` set for what would otherwise come from `ListActiveUserSubgraphs(user_id)`. Add one sibling helper next to `canreadnote.Resolve` that takes a precomputed set and skips the user lookup.

### Setup walkthrough (per peer)

1. **Bob: create the secret on Bob's base.** Admin form: pick a kid (`alice-2026`), generate fresh secret bytes, set scope (which Bob's subgraphs Alice may see — say `team-status`).
2. **Bob: send `(kid, bytes)` to Alice out-of-band** (Telegram).
3. **Alice: paste in admin** as outbound (`kb_url=https://bob.team.io/_system/mcp`, kid, bytes). Encrypted at rest.
4. **Alice: create KB-note in her vault** with `mcp_federation_kb_url: https://bob.team.io/_system/mcp`. Optionally place it in a subgraph that gates KB-note visibility if she has multi-user concerns (she usually doesn't).

Alice is now wired to Bob. To wire bidirectionally, Alice creates her own secret for Bob and they repeat. Each pair manages its own keys — there's no central registry.

### Public-base case

KB-note with `mcp_federation_kb_url: https://philosophers.example.com/_system/mcp` and **no** `federation_secrets` row. Hub proxies anonymously. Public base returns its guest layer. Done.

### Revocation

```sql
delete from federation_secret_subgraphs where kid = ? and subgraph_id = ?;     -- shrink scope
update federation_secrets set revoked_at = current_timestamp where kid = ?;    -- kill the secret
```

The hub keeps trying; subsequent calls hit 401. Hub admin notices via warn-level logs. No coordination needed.

### Implementation notes

- `secret_crypt` reuses trip2g's master-key encryption (same primitive as Telegram client tokens).
- All HMAC compares use `subtle.ConstantTimeCompare`.
- No `jti` replay cache — 30s TTL is enough for now.
- Inbound auth applies to federated methods only; local `search`/`similar`/`note_html` keep their existing public/authenticated behavior.

---

## Adapter integration

Adapters bring non-trip2g sources into the federation. From the hub's POV they look exactly like another peer: an MCP endpoint at some URL, optionally protected by a federation secret.

### Protocol contract

An adapter implements the same six MCP methods. A bare minimum is:

- `search(query)` — returns results in trip2g's `SearchResultPayload` shape, with `note_id`, `title`, `url`, `score`, `snippet`. Items don't need to be markdown notes — they can be GitHub issues, Telegram messages, Notion blocks, anything that can be addressed by a stable id.
- `note_html(pid)` — returns HTML or plain text content for one item. For a GitHub issue, that's the issue body + comments. For a Telegram message, the rendered text + media descriptors.
- `similar(pid)` — optional; if not implemented, return an empty result set.

Adapters that surface authenticated content reuse `federation_secrets` / `federation_secret_subgraphs` exactly the same way trip2g bases do. The "subgraph" on an adapter is whatever logical slice it exposes — a GitHub repo, a Telegram channel, a Notion workspace.

### Two concrete adapters in scope

**GitHub adapter** (separate repo / binary):
- Subgraphs map to `(org, repo)` pairs.
- `search` queries the GitHub API across configured repos.
- `note_html` fetches an issue body, PR description, or wiki page by id.
- Auth: a federation secret per hub. Each kid's scope lists which repos it can see.
- The adapter holds GitHub PATs / app tokens internally; trip2g hubs never see them.

**Telegram adapter** (separate repo / binary):
- Subgraphs map to channels or saved-message buckets.
- `search` over indexed message history.
- `note_html` fetches a message + thread by id, rendered as HTML.
- Auth: a federation secret per hub. Each kid's scope lists which channels it can read.
- The adapter holds Telegram session credentials internally.

These adapters live outside the trip2g codebase. This document defines only the protocol surface they must implement to be hubbable. Each adapter gets its own design doc when actually built.

### Why adapters use the same auth as bases

So that the hub has one and only one way to authenticate to anything on the federation. No special-casing GitHub vs Telegram vs another trip2g — they're all just "an MCP endpoint with maybe a kid+secret".

---

## Implementation plan

### Stage 1: MVP

**Goal:** my hub talks to my own content + 2-3 public bases + 2-3 private peers + 1 adapter (GitHub or Telegram).

1. **Schema migration** (new `db/migrations/...sql`):
   - `federation_secrets`, `federation_secret_subgraphs`.
2. **`internal/model/mcp_federation_note.go` (new):**
   ```go
   type MCPFederationNote struct {
       Note     *NoteView
       URL      string
       ID       string  // kb_id slug
       MaxDepth int
   }
   func newMCPFederationNote(n *NoteView) *MCPFederationNote { ... }
   func hostnameFromURL(raw string) string { ... }
   ```
3. **`NoteViews`:** add `MCPFederationNotes []*MCPFederationNote`, populated during finalization.
4. **`internal/case/mcp/types.go`:** new arguments (`FederatedSearchArguments` with `KBID`/`KBIDs`; `FederatedSimilarArguments`; `FederatedNoteHTMLArguments`). Extend `SearchResultItem` with `URL` and `Federation *FederationRef`. Add `Context` field to federated payloads.
5. **`internal/case/mcp/federation.go` (new):**
   - `signOutbound(secret, kid, rid) (jwt string, err error)`.
   - `verifyInbound(jwtStr) (kid string, err error)` — looks up `federation_secrets`, constant-time HMAC verify.
   - `proxyToKB(ctx, kb, secret, method, args) (*Response, error)` via `fasthttp.Client`.
   - `fanout(ctx, kbs, method, args) []SearchResultItem` — parallel with timeout.
   - `splitKBID(id) (head, rest string)`, `prefixKBID(localID, results)`.
6. **`internal/case/mcp/resolve.go`:**
   - `handleToolsList`: always 6 methods.
   - `handleToolsCall`: dispatch federated_*.
   - `handleSearch`: mark KB-notes with `kind="federation_kb"` + `federation` block. Add `URL`.
   - `handleFederatedSearch` / `Similar` / `NoteHTML`.
   - `accessibleKBNotes(ctx, env, user)` filter via `canreadnote.Resolve`.
7. **Inbound auth:** middleware on `/_system/mcp` (or inline in `Resolve`) that, when an Authorization header is present on a federated_* call, runs `verifyInbound` and stashes the kid for downstream filtering.
8. **Result filtering on inbound:** add a sibling of `canreadnote.Resolve` that takes a precomputed `[]string` of allowed subgraph names. Used by federated handlers when JWT is present.
9. **Outbound key lookup:** `federation_secrets WHERE kb_url = ? AND revoked_at IS NULL ORDER BY created_at DESC LIMIT 1`.
10. **Admin UI** (the bulk of v1 effort, mol widgets):
    - Federation peers list: KB-notes + linked secret status.
    - Add peer: paste kid + secret bytes → encrypt → `federation_secrets`.
    - Manage scope: list subgraphs, checkbox per kid → `federation_secret_subgraphs`.
    - Revoke: soft-delete via `revoked_at`.
11. **Tests** — see "Test plan" below.

### Stage 2: hardening (when needed, not preemptively)

- `__visited` array for cycle detection (real production data shows it's needed).
- Sub-base response cache.
- Health metrics per peer (latency, error rate).
- Out-of-scope warning round-trip (response carries a `warnings` array; hub logs).

### Stage 3: tooling around federation (post-MVP)

- `/_system/mcp/federation` discovery endpoint — in marketplace doc; consider if useful even outside marketplace.
- Discovery refresh cron.
- Federation mesh visualization (`/_system/federation` topology probe).

### Stage 4: marketplace

If a real reseller use case appears, see `docs/dev/mcp_federation_marketplace.md` for the full design. It's additive on top of this MVP.

---

## Test plan

### Reference fixture

```
                          hub (mine)
            ┌──────────────┼──────────────┬──────────────┐
            │              │              │              │
        public-A       peer-B         peer-C         adapter-X
        (no auth)    (kid=alice)    (kid=alice)    (kid=alice-gh)
                     scope=[team]   scope=[]       scope=[repo1]
```

- 1 hub.
- `public-A`: anonymous, returns content for any caller.
- `peer-B`: requires JWT; kid `alice` has scope `["team"]`.
- `peer-C`: requires JWT; kid `alice` has empty scope (anonymous-equivalent for this kid).
- `adapter-X`: GitHub-shaped adapter; requires JWT; kid `alice-gh` has scope `["repo1"]`.

`httptest.NewServer` per node. KB-notes generated programmatically.

### Scenarios

| # | Scenario | What we check |
|---|----------|---------------|
| 1 | `tools/list` always returns 6 methods | Independent of KB-note count. |
| 2 | `federated_search` with no KB-notes | "federation not configured" structured payload, not an error. |
| 3 | `search` finds a KB-note | Result has `kind="federation_kb"` and full `federation` block. |
| 4 | `federated_search(kb_id="public-A")` | Direct proxy with no Authorization header; results returned. |
| 5 | `federated_search(kb_id="peer-B")` with valid scope | JWT signed, base verifies, returns public + team-scope content. |
| 6 | `federated_search(kb_id="peer-C")` with empty scope | JWT signed, base verifies, returns public layer only. |
| 7 | `federated_search` fan-out | Hits all 4, merges via RRF; KB-notes excluded from response. |
| 8 | `federated_search(kb_ids=["peer-B","adapter-X"])` | Hits exactly those two; results from each include their kb_id. |
| 9 | `federated_note_html(pid, kb_id="peer-B")` | HTML returned via proxy. |
| 10 | Targeted call with two-level kb_id | Hub strips first segment, proxies the rest. |
| 11 | Inaccessible KB-note (operator subgraph ACL) | KB-note hidden in `search`, `federated_*` returns "not configured". |
| 12 | Reverse prefix rewriting | Sub-hub returns kb_id="X"; parent rewrites to "sub/X". |
| 13 | URL on every result | Result URLs point at real remote hosts, not the hub. |
| 14 | One sub-base errors | Other results returned; warn logged. |
| 15 | One sub-base times out | Same as #14, latency check. |
| 16 | Empty results from a sub-base | Merge handles it without panic. |
| 17 | Fan-out parallelism | 3 servers responding after 1s each; total ≈ 1s, not 3s. |
| 18 | Unknown kid → 401 | JWT signed with random kid not in `federation_secrets`. Audit at warn-level log. |
| 19 | Wrong signature → 401 | Tampered body, valid kid. |
| 20 | Expired JWT → 401 | exp in past beyond skew. |
| 21 | Revoked secret → 401 | `revoked_at` set; JWT is otherwise valid. |
| 22 | No JWT on private peer | Treated as anonymous, sees only public layer. |
| 23 | Public-A with JWT | JWT ignored (no `federation_secrets` entry); public layer returned. |
| 24 | Hub picks newest secret on outbound | Two non-revoked rows for same kb_url; outbound picks the most recent `created_at`. |
| 25 | Constant-time HMAC compare | Differential timing test (best-effort) — varying signature prefixes, equal latency within tolerance. |
| 26 | `kb_instructions` cached | Second call to same kb_id doesn't re-fetch base's `initialize`. |

### Layout

`internal/case/mcp/federation_test.go` — main file. `httptest.NewServer` for each fixture node. Existing `mocks_test.go` reused for unit tests of helpers (`splitKBID`, `prefixKBID`, `signOutbound`, `verifyInbound`).

---

## Open questions

- **Multiple users per hub.** If I let a colleague point their agent at my hub, federation per-user ACL via `canreadnote` already works (their session, their subgraphs). But session resolution at MCP endpoint depends on Phase 1B from `docs/superpowers/specs/2026-03-27-ai-chat-design.md`; that's a separate dependency for that day. Not blocking the personal-only case.
- **Per-channel rate limiting.** Not needed for personal use, would matter on public hubs.
- **TLS enforcement.** Should hub refuse to talk to non-https peer URLs? Recommend yes in production, but document only — operator's call.

---

## Future ideas

- **Marketplace mode** (`docs/dev/mcp_federation_marketplace.md`) — reselling, billing, user-capability JWTs, discovery endpoint, hash pseudonyms. Design only, deferred.
- **Mesh visualization** — `/_system/federation` topology endpoint, recursively crawled.
- **Adapter library** — first-party Go skeleton for building new MCP adapters quickly.
- **Cron health check** — periodic ping of every connected peer, surfacing breakage in admin.
- **System-reminder enrichment** — automatic injection of "you have N federations available" into agent's startup context.

## Federation hop depth (`mcp-federation-max-depth`)

Every federated request carries a cumulative hop counter in the
`X-MCP-Federation-Depth` header, incremented by 1 at each outbound hop
(`internal/federation/client.go`). On receipt each instance compares the
**incoming** depth against its **own** `mcp-federation-max-depth` (default `3`,
`appconfig/config.go`) and rejects with `federation max depth exceeded` when
`incomingDepth >= localMax` (`internal/case/mcp/endpoint.go`). Direct clients
send no header (depth 0) and are never rejected by this check — the limit only
governs requests arriving *through* federation.

A nested `kb_id` path consumes one depth level per segment: from the root,
`philosophers/nietzsche` reaches depth 2 (works at default 3), while
`philosophers/nietzsche/schopenhauer` reaches depth 3 and is rejected. So the
default `3` allows **two levels of nesting** below a direct client.

### Heterogeneous limits across instances

The counter is global to a chain; each instance judges it by its own local max.
Consequences when instances disagree:

- The chain is cut at the **first** instance whose `max <= the depth it
  receives at`. The **stingiest instance on the path caps everything beyond
  it** — a larger limit downstream is useless if an upstream peer already
  rejected.
- Reachability is **route-dependent and asymmetric**: the same leaf may be
  reachable through one hub and not through another, purely by which instances
  sit at which positions. This is **expected**, not a bug.
- An entry instance can reject early by checking `incomingDepth +
  segments(kb_id)` against its **own** max, but it cannot know downstream peers'
  limits, so deeper cuts still happen live.

**Operational guidance:** keep `mcp-federation-max-depth` consistent across a
federation you operate, or monotonically non-decreasing along expected paths
(hub ≥ leaves). Otherwise callers see `max depth exceeded` that depends on the
route taken.

### Prior art

This is the same shape as two well-worn standards. The `/`-delimited `kb_id`
resolved hop-by-hop through delegating peers is **DNS-style hierarchical
delegation** (each instance authoritative for its own zone, aware only of its
direct children); the caller-frame `kb_id` rewrite on the way back is the
equivalent of qualifying a name toward an FQDN. The depth counter is an
application-layer **TTL / `Max-Forwards`** (IP TTL RFC 791; HTTP `Max-Forwards`
RFC 7231; SIP `Max-Forwards` RFC 3261) — a per-hop counter that drops a request
when exhausted, with the same inherent, accepted route-dependence.
