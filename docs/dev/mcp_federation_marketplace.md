# MCP Federation — Marketplace / Reseller Mode (FUTURE DESIGN)

> **Status: design only — not implemented in MVP.**
>
> This document captures the federated-marketplace extension where one operator
> resells access to another operator's base, charges users on top, and reconciles
> revenue out-of-band. It introduces user-capability JWTs (`sub`/`sgr`), kid-prefix
> encoding in `subgraphs.name`, a `/_system/mcp/federation` discovery endpoint,
> secret rotation, scope warnings, and protocol versioning.
>
> The active MVP spec is `docs/dev/mcp_federation.md` — a much simpler personal-hub
> + private-peers + adapters model. Build that first; revisit this document only
> when a real commercial reselling use case appears.
>
> Implementation order: ship `mcp_federation.md` end-to-end, run it for several
> months, then re-evaluate this doc against accumulated reality before committing
> to any of its additional surface area.

---

# MCP Federation — Routing Queries Across Knowledge Bases

**Problem:** With 10-20+ knowledge bases (RAG) on trip2g, registering each one in the agent's `.mcp.json` doesn't scale. It requires manual config edits for every new base, and Claude sees dozens of identical method sets.

**Solution:** A single MCP hub in the agent's `.mcp.json`. The hub knows about other bases through **KB-notes** — ordinary markdown notes with `mcp_federation_kb_*` frontmatter fields. When the agent finds such a note in search results, it gets a direct instruction to call the federated method with that `kb_id`. Adding a new base to the federation = creating a note. No config changes.

---

## Goals

1. **Zero-config federation growth.** Create a note with `mcp_federation_kb_url` — the base is connected.
2. **Discovery through the same search.** The agent finds KB-notes the same way it finds regular results, by semantic/lexical relevance. No separate "list of bases" mechanism needed.
3. **Transparent proxying.** `federated_note_html` returns a remote note's content as if it were local.
4. **Hierarchy (optional).** A hub can reference other hubs. Depth is controlled via `mcp_federation_kb_max_depth`.
5. **Don't break standalone installs.** A hub without KB-notes still exposes federated methods (so the agent's tool cache stays stable), but those methods return a clear "no federation configured" payload.

---

## Non-goals

- We do not replicate the index/chunks across bases. The hub stores no sub-base data, only proxies queries.
- No shared accounts / SSO out of the box. Auth is its own multi-stage track (see "Authorization").
- No centralized registry. The source of truth is notes inside the bases themselves.

---

## Architecture

### Topology

```
                    ┌─────────────────────┐
                    │  Claude (.mcp.json) │
                    │  hub.2pub.me only   │
                    └──────────┬──────────┘
                               │ federated_search("burnout")
                               │
                  ┌────────────▼────────────┐
                  │  hub.2pub.me            │
                  │  (KB-notes:             │
                  │   • sinitsin            │
                  │   • marcus              │
                  │   • science-hub)        │
                  └─┬──────┬────────────┬───┘
                    │      │            │
            ┌───────▼─┐ ┌──▼────┐ ┌─────▼────────┐
            │sinitsin │ │marcus │ │ science-hub  │
            │ (leaf)  │ │(leaf) │ │ (KB-notes:   │
            └─────────┘ └───────┘ │  • cellbio   │
                                  │  • neuro)    │
                                  └─┬────────┬───┘
                                    │        │
                              ┌─────▼──┐ ┌───▼────┐
                              │cellbio │ │ neuro  │
                              └────────┘ └────────┘
```

The agent's `.mcp.json` only contains the hub. Everything else is a regular trip2g base, connected via KB-notes.

### Request flow (typical scenario)

```
1. Agent calls search("engineer burnout") on hub.2pub.me.

2. Hub runs local search (FTS + vector). Among the results
   there's a KB-note "Sinitsin's KB" (its frontmatter
   contains mcp_federation_kb_url).

3. The KB-note is returned to the agent with a federation block
   containing kb_id, kb_url, and agent_instruction telling the
   agent how to query that base.

4. The agent calls federated_search(query="...", kb_id="sinitsin").

5. Hub resolves kb_id → URL → proxies the request to
   sinitsin.2pub.me. It receives results, prefixes each kb_id
   with "sinitsin", and returns them to the agent. The response
   also carries the base's own instructions inline.

6. Agent calls federated_note_html(pid=42, kb_id="sinitsin").
   Hub proxies and returns HTML.
```

---

## KB-note frontmatter

```yaml
---
mcp_federation_kb_url: https://sinitsin.2pub.me/_system/mcp
mcp_federation_kb_id: sinitsin           # optional, default = hostname(URL)
mcp_federation_kb_max_depth: 0           # optional, default 0 (leaf)
---
Use when: management, engineering career, startups, product, teams.
Don't use when: philosophy, biology.
```

| Field | Required | Purpose |
|-------|----------|---------|
| `mcp_federation_kb_url` | yes | Full endpoint URL of the sub-base (`https://host/_system/mcp`). The marker that "this note is a KB-note". |
| `mcp_federation_kb_id` | no | Slug used inside `kb_id`. Defaults to the URL hostname. Useful to shorten "sinitsin.2pub.me" to "sinitsin". |
| `mcp_federation_kb_max_depth` | no | How many federation levels are allowed **inside this base** when queried. `0` (default) = leaf. `1+` = intermediate hub. |

**Note body** (after frontmatter) is the "when to use this base" instruction. Used as vectorization context: semantically close queries surface this KB-note and prompt the agent to proxy.

**Note title** (= filename in Obsidian) serves as a human-readable name in result listings.

### Why the `mcp_federation_*` prefix

- Consistent with existing `mcp_method` / `mcp_description` (see `internal/model/note.go:571-575`).
- A clear namespace — won't collide with other `kb_*` fields that may appear in the system later.

---

## KB-note visibility (per-user routing ACL)

A KB-note is a regular note. It can carry `subgraphs:` in frontmatter exactly like any other note, and **its visibility to a user is the existing `canreadnote.Resolve` decision** — federation reuses the site's note ACL as the routing ACL.

```yaml
---
mcp_federation_kb_url: https://premium.example.com/_system/mcp
subgraphs: [premium]                       # only premium subscribers see this route
---
Use when: deep technical content...
```

### Consequences

- A user can route through a KB-note **only if** `canreadnote(user, kb_note) == true`.
- Different users see different sets of KB-notes → different available federations → different fan-out targets.
- Anonymous (no `mcp_token`) sees only KB-notes that pass `canreadnote` with a guest token (i.e. `note.Free` and/or `require_signin=false` subgraphs).
- Authenticated user without paid grants: sees free KB-notes plus any `require_signin` ones.
- Paid user: sees the union of free + their `ListActiveUserSubgraphs` → all KB-notes in those subgraphs.

### Where the filter lives

Every federation entry point reads from a per-request **accessible KB-notes** view, not from the global `MCPFederationNotes` index:

```go
accessibleKBNotes(ctx, env, user) []*MCPFederationNote
  -> filter env.LatestNoteViews().MCPFederationNotes
     where canreadnote.Resolve(ctx, env, kb.Note) == true
```

Used by:
- `search` — KB-notes excluded from results if not accessible.
- `federated_search` (fan-out) — only iterates the accessible set.
- `federated_search(kb_id)` / `federated_note_html(kb_id)` / `federated_similar(kb_id)` — resolves `kb_id` against the accessible set; if the requested kb_id maps to a non-accessible KB-note, the response is **indistinguishable from "kb not found"** (same "federation not configured for this kb_id" structured payload — never reveal that the kb exists).

### Authoring guidance

Two natural patterns emerge:

1. **Free route + paid destination.** KB-note in a free subgraph (or no subgraph at all). The route is world-discoverable; gating happens at the destination, where `federation_secret_subgraphs` and the user's `sgr` claim filter content. Recommended default — predictable behavior, fewer surprises during marketplace onboarding.

2. **Paid route + paid destination.** KB-note carries `subgraphs: [premium]`. Only paying users even *see* the route. Used when the operator wants to keep the existence of a federated source itself private (e.g. confidential B2B feeds whose presence shouldn't leak to free tier users).

Mixed cases (free route, free destination) just work like a public discovery layer.

### Marketplace alignment

The KB-note's frontmatter `subgraphs:` only accepts the local naming alphabet (`^[a-zA-Z0-9_]+$` — no slashes), so it can only reference **local** subgraph rows for routing visibility. Federated subgraph rows live as `<kid>/<remote_name>` and are not addressable from frontmatter.

In practice this means an offer for a federated base bundles **two** subgraph rows:

- A local route subgraph (e.g. `premium`) — the KB-note frontmatter has `subgraphs: [premium]`, controlling route visibility.
- One or more registered remote subgraphs (e.g. `kid_friend/course1`) — these end up in `sgr` on the outbound JWT.

A single `offer_subgraphs` block attaches both to the same offer. One purchase grants both, and the user gets both route visibility and content access in one step. The split is a feature, not friction: it keeps frontmatter validation strict and lets operators route a single KB-note to several remote subgraphs without touching the note.

### Schema implication

None. KB-notes already have `subgraphs:` parsing in `model/note.go:NoteView.ExtractSubgraphs`. The `MCPFederationNote` struct (proposed in Stage 1) just wraps a `*NoteView` — the underlying note's subgraph membership is preserved automatically. Federation needs only to **call** `canreadnote.Resolve` per request before exposing the route.

---

## kb_id and hierarchy

### Format

`kb_id` is a path of segments separated by `/`:

```
sinitsin                          # standalone KB
science/cellbio                   # cellbio inside science-hub
science/neuro                     # neuro inside science-hub
```

**Separator is `/`**, not a dot — a dot collides with hostnames.

### Default = hostname

If `mcp_federation_kb_id` is not set, the hostname from URL is used. For most bases the default is good; a slug is only needed when you want to shorten the id.

### Reverse prefixing — REQUIRED on every hop

When a sub-hub returns results to its parent, it returns `kb_id`s **without its own segment** (relative). The parent **rewrites every kb_id** in the response by prepending its own segment.

Example: `science-hub` returns results with `kb_id="cellbio"`. The root `hub` rewrites them to `kb_id="science-hub/cellbio"` before serving the agent. The agent later uses this exact `kb_id` to call `federated_note_html`. The hub splits the first segment off, proxies the remainder, and the chain unwinds in reverse.

This rewriting applies to:
- `kb_id` on every result item.
- `kb_id` inside the `federation` block of any KB-note result that surfaced from a sub-hub's `search`.
- Any `kb_id` returned in the response payload.

---

## MCP methods

### Always 6 methods (cache-stable contract)

`tools/list` always returns the same six tool definitions. This keeps the agent's tool cache stable: the user can add or remove KB-notes at runtime without invalidating the conversation's tool list.

```
search(query)                       — local search (FTS + vector)
similar(note_id|pid)                — similar to a local note
note_html(note_id|pid)              — HTML of a local note

federated_search(query, kb_id?, kb_ids?)
federated_similar(note_id, kb_id)
federated_note_html(note_id, kb_id)
```

The federated methods are **mirror methods** of the local ones — same semantics, with optional/required `kb_id`(s) for routing.

### What happens when no KB-notes exist

The federated methods are still listed in `tools/list`, but invoking them returns a structured response:

```json
{
  "result": {
    "content": [{"type": "text", "text": "Federation is not configured for this hub. No KB-notes were found. To enable federation, create a note with mcp_federation_kb_url frontmatter pointing to another MCP endpoint."}],
    "structuredContent": {
      "federation": {"configured": false, "kb_count": 0}
    }
  }
}
```

It's not an error — just an empty federation. The agent handles it gracefully.

### `kb_id` vs `kb_ids` in `federated_search`

`federated_search` accepts EITHER `kb_id` (single base) OR `kb_ids` (multiple bases) OR neither (full fan-out across all known KB-notes).

| Args | Behavior |
|------|----------|
| no `kb_id`, no `kb_ids` | Full fan-out: query every KB-note's base in parallel. |
| `kb_id: "sinitsin"` | Targeted: query only that base. |
| `kb_id: "science/cellbio"` | Targeted, two-level proxying. |
| `kb_ids: ["sinitsin", "marcus"]` | Multi-target: query exactly these two in parallel. |
| `kb_ids: ["science/cellbio", "sinitsin"]` | Mixed depth: each id resolved independently, in parallel. |
| Both `kb_id` and `kb_ids` provided | `InvalidParams` error. |

The agent decides at call time which form to use, guided by:
- The hub's `instructions` (returned from `initialize`).
- KB-note bodies surfaced in earlier `search` results.
- Per-base instructions returned in earlier `federated_search` responses (see "Instructions propagation").

### `federated_similar` and `federated_note_html`

Both **require `kb_id`** (single string). Multi-base variants don't make sense — a `note_id` is unique only within a base.

```
federated_similar(note_id, kb_id, limit?)
federated_note_html(note_id, kb_id, match_id?)
```

Without `kb_id`, returns `InvalidParams`.

### Why mirror methods instead of one set with optional kb_id

- **Explicit intent in name.** `search` = local only, `federated_search` = may go beyond. No semantic ambiguity from a parameter.
- **Stable response shape.** Local methods never carry federation metadata; federated ones always do.
- **Static `tools/list`.** Always six methods, regardless of base count or runtime state.

---

## Instructions propagation

Each base may have its own usage rules in its `mcp_method: initialize` note ("write citations with date", "Sinitsin's voice is Russian", etc.). These need to reach the agent without dumping every base's instructions at hub init time.

### Three-layer instruction model

**Layer 1 — Hub-level (returned by `initialize`):**

The hub's own `mcp_method: initialize` note describes federation usage at this hub:

```
This is a federated hub. Methods:
- search/similar/note_html — local notes only.
- federated_search(query) — fan-out across all bases.
- federated_search(query, kb_id="X") — query base X.
- federated_search(query, kb_ids=["X","Y"]) — query a subset.
- federated_note_html(pid, kb_id="X") — open a remote note.

When a search result has kind="federation_kb", that's a pointer
to a sub-base. Read its federation.agent_instruction for next-step
guidance. The base's own usage rules will arrive inline with the
first federated_search response targeting that base.
```

This stays small regardless of base count — it doesn't list bases.

**Layer 2 — KB-note body (surfaced in `search` results):**

When local `search` returns a KB-note, its body ("Use when: management…") tells the agent _when_ to route to that base. This is the discovery layer: the agent learns what each base is for, on demand, by topic relevance.

**Layer 3 — Per-base instructions (returned with `federated_search` results):**

When a `federated_search` response comes back from a specific base, the response payload includes that base's own `instructions` inline:

```json
{
  "results": [...],
  "context": {
    "kb_id": "sinitsin",
    "kb_instructions": "Sinitsin writes in Russian about engineering management. Cite original posts as sinitsin.ru/<slug>. Material spans 2015-2024; prefer recent posts when topics overlap."
  }
}
```

The hub fetches and caches each base's `initialize.instructions` lazily (first call to that base) and embeds it in subsequent `federated_search` responses. Cache TTL: 5 minutes.

The agent locally assembles context per query:
```
[hub instructions]
+ [KB-note body for the chosen base, if surfaced earlier]
+ [kb_instructions from the federated_search response]
+ [the actual search results]
```

This way, no instruction set is preloaded; each one arrives only when relevant.

---

## KB-note in search results

A KB-note must surface in local `search` results so the agent learns the base exists. But its body is metadata, not content, so it's marked specially:

```json
{
  "title": "Sinitsin's KB",
  "note_id": 17,
  "note_path": "kbs/sinitsin.md",
  "kind": "federation_kb",
  "url": "https://hub.2pub.me/kbs/sinitsin",
  "score": 0.71,
  "snippet": "Use when: management, engineering career...",
  "federation": {
    "kb_id": "sinitsin",
    "kb_url": "https://sinitsin.2pub.me/_system/mcp",
    "agent_instruction": "This is a knowledge base pointer. To search inside it, call federated_search with kb_id=\"sinitsin\". To open notes from it, call federated_note_html(note_id=..., kb_id=\"sinitsin\")."
  }
}
```

Key points:
- `kind: "federation_kb"` — distinct result type alongside `note`, `index`, `source`.
- `federation` block is a single nested object containing `kb_id`, `kb_url`, and `agent_instruction`. All federation-specific data lives together.
- `agent_instruction` is human-readable. Claude follows direct instructions in search results more reliably than inferences from metadata.

In `federated_search` responses (with or without `kb_id`/`kb_ids`), KB-notes are **excluded** — fan-out is already happening, the agent doesn't need to recurse into the same base again.

---

## URL on every result

Every `SearchResultItem` (in all six methods) includes a full URL:

```json
{ "url": "https://sinitsin.2pub.me/notes/burnout-engineers" }
```

The agent uses these URLs as proof links to the user. For proxied results, the URL is on the **real sub-base domain**, not the hub's. Otherwise the link 404s.

---

## Fan-out search

`federated_search` without `kb_id`/`kb_ids`:

1. Compute `accessibleKBNotes(ctx, env, user)` — the user-filtered subset (see "KB-note visibility").
2. Local `search` runs in parallel with sub-base calls.
3. Fan-out via goroutines to every base in the accessible set, query passed as is.
4. Each sub-base responds with its results (and its `kb_instructions` for the per-base context).
5. Results merged via RRF (`mergeResults` reuse); KB-notes excluded.
6. Each result has correct `url` and `kb_id` (with prefix rewriting applied at this hop).
7. `context` field aggregates `kb_instructions` from all responding bases.

**Parameters:**
- Per-call timeout: 2 seconds per sub-base.
- If a base errors / times out → log warn, continue. Try every time on subsequent calls; no "disable failing base" optimization yet (do it later when needed).
- Cache base responses for 30-60s by `(kb_url, query)` — reuse `internal/cache`.

`federated_search` with `kb_ids: [a, b, c]` is the same path but limited to the listed bases — and each requested id must resolve against `accessibleKBNotes`. Inaccessible ids are dropped from the request silently (same kb-not-found semantics as in targeted mode).

---

## Cycle handling

Two routing modes have different cycle properties.

### Targeted mode (`kb_id` / `kb_ids`) — no protection needed

Cycles are **structurally impossible**. Each hop strips one segment off the kb_id:

```
hub receives kb_id="A/B/C"  → proxies to A with kb_id="B/C"
A receives kb_id="B/C"      → proxies to B with kb_id="C"
B receives kb_id="C"        → proxies to C with kb_id=""
```

The chain can only get shorter. Termination is guaranteed by construction.

### Fan-out mode (no `kb_id`) — `__visited` + self-skip + depth ceiling

Cycles are possible: A's KB-note points to B, B's KB-note points to A. URL-based equality is the only reliable check (hubs don't know their own external slug as seen by a parent).

**Mechanism:**

Every proxied request carries a service field `__visited` — array of MCP endpoint URLs traveled so far on this request path:

```json
{
  "method": "federated_search",
  "params": {
    "query": "...",
    "__visited": ["https://hub.2pub.me/_system/mcp", "https://science-hub.2pub.me/_system/mcp"]
  }
}
```

On every hop:
1. Hub checks if its own URL is in `__visited`. If yes — log `WARN cycle detected: <visited_path> -> self`, return empty result for this branch (rest of fan-out continues). Warning is logged on every cycle hit, not deduplicated — operators want to see every occurrence to fix the topology.
2. Otherwise — append own URL to `__visited` before proxying further.

Two extra guards on top:

- **Self-skip at fan-out:** hub also skips any KB-note whose `kb_url == self_url` before even calling. Catches the simplest self-cycle without round-tripping.
- **Global `MCP_FEDERATION_MAX_DEPTH`** (env, default `3`): hard ceiling in case `__visited` propagation breaks (mismatched protocol versions, custom adapters that drop the field). Belt-and-braces.

`__visited` is cheap (one extra string array per request) and gives precise debug messages, so it's worth keeping in Stage 2 even though depth ceiling alone would technically suffice.

---

## Logging

Federation is the easiest place to lose visibility — debug logs are mandatory.

Required log points (all at `debug` level except where noted):

| Event | Level | Fields |
|-------|-------|--------|
| Federation request received | debug | method, kb_id, kb_ids, query, visited_count |
| KB-note matched in local search | debug | kb_id, kb_url, score |
| Fan-out start | debug | targets (kb_ids), query, parent_visited |
| Per-base proxy call start | debug | kb_id, kb_url, method |
| Per-base proxy call done | debug | kb_id, latency_ms, results_count |
| Per-base proxy call failed | warn | kb_id, kb_url, error, latency_ms |
| Cycle detected | warn | visited_path, attempted_url |
| KB-note URL unreachable | warn | kb_id, kb_url, error |
| Reverse prefix rewrite | debug | original_kb_id, rewritten_kb_id, count |
| `kb_instructions` cache miss / fetch | debug | kb_id, latency_ms |
| Federation request done | debug | method, total_latency_ms, total_results |

Use the `mcp:federation` logger prefix (consistent with existing `logger.WithPrefix(env.Logger(), "mcp:handleSearch")`).

---

## Implementation plan

### Stage 1: MVP (1 day)

**Goal:** simple case "hub → one level of leaves" works. No hierarchy, no `__visited`, no auth.

1. **`internal/model/mcp_federation_note.go` (new):**
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
2. **`NoteViews` (in `model/note.go`):** add field `MCPFederationNotes []*MCPFederationNote`. Populated during finalization.
3. **`internal/case/mcp/types.go`:** new argument structs (`FederatedSearchArguments` with `KBID` and `KBIDs`, `FederatedSimilarArguments`, `FederatedNoteHTMLArguments`). Extend `SearchResultItem`: `URL`, `Federation *FederationRef` (with `KBID`, `KBURL`, `AgentInstruction`). Add `Context` field to federated search response payloads.
4. **`internal/case/mcp/federation.go` (new):**
   - `proxyToKB(ctx, kb, method, args, visited) (*Response, error)` via `fasthttp.Client`.
   - `fanout(ctx, kbs, method, args, visited) []SearchResultItem` — parallel, with timeout.
   - `splitKBID(id) (head, rest string)`.
   - `prefixKBID(localID, results)` and `prefixKBIDInFederation(...)`.
   - `kbInstructionsCache` — TTL 5min, reuse `internal/cache`.
5. **`internal/case/mcp/resolve.go`:**
   - `handleToolsList`: always returns 6 methods.
   - `handleToolsCall`: new cases for `federated_search`, `federated_similar`, `federated_note_html`. If no KB-notes — return "federation not configured" payload.
   - `handleSearch`: mark KB-notes with `kind="federation_kb"` and the `federation` object including `agent_instruction`. Add full `URL` on every result.
   - `handleFederatedSearch`: validate (`kb_id` xor `kb_ids`), dispatch to single proxy or fan-out, apply reverse prefixing.
   - `handleFederatedNoteHTML`, `handleFederatedSimilar`: split `kb_id`, proxy to first segment with the rest.
6. **`Env`:** add `FederationConfig() FederationConfig` (timeout, http client, instruction cache TTL).
7. **`mcp_method: initialize` note on the hub:** describes federation usage. Returned in `initialize.instructions`. (Drafted by user, lives in Obsidian — not code.)
8. **Tests:** see "Test plan" below.

### Stage 2: hierarchy and cycles

1. `__visited` plumbing through every proxied request.
2. Reverse prefixing for every level (covered conceptually in MVP, but tested only at depth 1).
3. Global `MCP_FEDERATION_MAX_DEPTH` from env.
4. Cache for sub-base responses.
5. Tests for cycles and multi-level topology.

### Stage 3: production readiness

1. Authorization — see dedicated section below.
2. Per-base health metrics, latency, cache hit rate.
3. **`/_system/federation` endpoint** — returns the federation topology reachable from this hub: list of direct KB-notes plus a recursive call to their `/_system/federation`. Used for mesh visualization, debugging, cycle discovery.

### Stage 4: adapters

1. Telegram channel history adapter — exposes a TG channel as an MCP base (search, note_html on messages).
2. Git repository docs adapter — exposes a git-tracked docs folder as an MCP base.
3. Google Docs adapter — vector-indexed Google Drive folders as MCP bases.

Each adapter is a separate service that speaks the trip2g MCP dialect. Hubs federate to them via KB-notes the same way as to other trip2g bases.

---

## Authorization

This whole section answers one operational question: how do I, as a hub operator, agree with another base owner to proxy queries to their content and bill my users for it?

The design reuses trip2g's existing access primitives (`subgraphs`, `offers`, `offer_subgraphs`, `purchases`, `user_subgraph_accesses` — see `docs/dev/subgraph_payment.md`). Federation adds two new tables, a federation-specific `InsertFederationSubgraph` writer (encodes the remote name as `<kid>/<remote_name>` in the existing `subgraphs.name`), and one new HTTP endpoint. Zero schema changes on `subgraphs`.

### Dependencies

- **Phase 1B (MCP tokens)** from `docs/superpowers/specs/2026-03-27-ai-chat-design.md` must be complete before federation auth can ship. Phase 1B introduces `mcp_tokens` (URL `?token=` resolves to a user via SHA256 hash) and the auth resolution path inside the MCP endpoint. Federation reuses that resolved user identity to decide what `sgr` to put in outbound JWTs. Currently only the registration of `mcp_tokens` is implemented; the resolution side is still TODO. Federation work is blocked until that lands.
- `docs/dev/subgraph_payment.md` — required reading; this section assumes its terminology.
- `docs/dev/hat.md` — federation JWTs follow the HAT pattern (HMAC, short TTL, signed claims).

### Mental model

```
                      ┌────────────────────────────────────┐
                      │ secret = symmetric HMAC key (HS256)│
                      │ kid    = short id chosen by the    │
                      │   BASE; the hub stores it as-is.   │
                      │ The hub puts kid in every outbound │
                      │ JWT header so the base can find    │
                      │ the right row in O(1).             │
                      └────────────────────────────────────┘

  HUB (reseller)                                 BASE (publisher)
  ──────────────                                 ────────────────
  KB-note in vault:                              federation_secrets row:
    mcp_federation_kb_url: https://base/...        kid="reseller-acme"
                                                   secret_crypt=...
  federation_secrets row (outbound):             federation_secret_subgraphs:
    kid="reseller-acme"                            (kid="reseller-acme",
    secret_crypt=encrypt(<bytes>)                   subgraph_id=42  // course1)
    kb_url=https://base/...                        (kid="reseller-acme",
                                                    subgraph_id=58  // course2)
  subgraphs (registered remote, encoded         No mention of the hub anywhere.
    as "<kid>/<remote_name>"):
    (name="reseller-acme/course1")
    (name="reseller-acme/course2")
  offers + offer_subgraphs:
    "acme-bundle" → [reseller-acme/course1,
                     reseller-acme/course2]
```

Same `kid` + same secret bytes on both sides. The hub uses them to **sign** outbound; the base uses them to **verify** inbound. Asymmetry comes from extra rows on either side: `federation_secret_subgraphs` on the base (what this kid is allowed to surface) and `subgraphs` rows on the hub whose `name` carries the kid prefix (which local rows correspond to remote subgraphs and how to route them).

### Setup walkthrough (one-time per pairing)

What the hub operator and the base operator do, in order. No code on either side after this.

1. **Hub operator: create the KB-note.** A regular markdown note in the hub's vault with `mcp_federation_kb_url: https://base.com/_system/mcp`. After save, hub's KB-notes index picks it up.

2. **Base operator: register the secret on their side.** Admin panel form: pick a local kid (free-form text used in the JWT header — e.g. `acme-2026`), let the panel generate fresh secret bytes (or paste your own). Stored encrypted at rest with the existing trip2g master key, same primitive used for Telegram client tokens.

3. **Base operator: attach scope.** Pick which of the base's `subgraphs` rows this kid may surface (`federation_secret_subgraphs`). Scope can be edited any time; every request reads it fresh.

4. **Base operator: hand `(kid, secret_bytes)` to the hub operator out-of-band.** Telegram DM, signed email, etc. The bytes are sent once and never re-displayed in the base's admin (the panel hashes/encrypts them after copy).

5. **Hub operator: register the secret in admin.** Admin panel form: paste the kid and bytes from the base, pick the KB-note URL the secret pairs with. Bytes are encrypted at rest. The frontmatter never sees the secret — only `mcp_federation_kb_url` lives in markdown.

6. **Hub operator: discover available subgraphs.** Click "fetch subgraphs" against the KB-note. The hub mints a JWT (signed with the bytes, header `kid` set to whatever the base sent), hits `GET https://base.com/_system/mcp/federation`, gets back a structured payload with `subgraphs[]`. The hub admin sees them on the page.

7. **Hub operator: register remote subgraphs locally.** Click "register" on each one (or "register all"). A federation-specific writer (`InsertFederationSubgraph(kid, remote_name)`) inserts a row with `name="<kid>/<remote_name>"`. The existing `case/updatesubgraphs/Resolve` keeps inserting local-only names without slashes — no overlap, no validator changes for note frontmatter — see "Encoding remote subgraphs in `subgraphs.name`" below.

8. **Hub operator: attach to offers / Patreon / Boosty / TG chats.** Same admin UI as for local subgraphs. Once attached, users who buy the offer (or join the chat, or subscribe on Patreon) get `user_subgraph_accesses` on those rows automatically through the existing payment pipelines. No federation code is touched at payment time.

After setup, traffic flows automatically. The base owner can edit scope (revoke a course, add a new one) without telling the hub; the hub will simply observe results changing on next refresh of `/_system/mcp/federation`.

### New schema

```sql
-- Generic key store. The same kid+secret_crypt is mirrored on both ends
-- out-of-band; this table holds whichever side's view of the pairing
-- lives on this server. Multiple rows MAY share the same kid to support
-- secret rotation (see "Secret rotation"); the runtime always picks the
-- newest row by created_at.
create table federation_secrets (
  id           integer primary key autoincrement,
  kid          text not null,                       -- short id placed in JWT header (NOT unique)
  secret_crypt blob not null,                       -- HMAC bytes, encrypted at rest
  kb_url       text,                                -- non-null = OUTBOUND (we sign for this URL)
                                                    --     null = INBOUND  (verify only)
  description  text,
  created_at   datetime not null default current_timestamp,
  created_by   integer not null references admins(user_id) on delete restrict,
  revoked_at   datetime
);
-- Compound index for "find active rows for kid, newest first":
create index idx_federation_secrets_kid_recent on federation_secrets(kid, created_at desc);

-- Inbound scope. Each row says "kid X may surface my subgraph Y".
-- Lives on the BASE side. NO ROWS for a kid means the secret has no special
-- access; calls authenticated by it see only the public layer (anything that
-- canreadnote.Resolve approves for a guest).
create table federation_secret_subgraphs (
  kid          text not null,                       -- not secret_id: scope follows kid across rotations
  subgraph_id  integer not null references subgraphs(id) on delete restrict,
  created_at   datetime not null default current_timestamp,
  created_by   integer not null references admins(user_id) on delete restrict,
  primary key (kid, subgraph_id)
);
```

**No `federation_audit_log` in MVP.** Audit / accounting is deferred to the billing phase, partly because dense per-request writes contend with SQLite's WAL and partly because reconciliation is out of scope until billing exists. For now, federation events go to the standard application logger (see "Logging" earlier). Add the table back when the billing module is designed.

### Encoding remote subgraphs in `subgraphs.name`

To avoid a schema migration on the existing `subgraphs` table, federation does **not** add a `kb_url` column. Instead, the local row for a remote subgraph is named `<kid>/<remote_name>`:

```
local subgraphs row             | meaning
--------------------------------|--------------------------------
name="A"                        | local subgraph (no slash, regular)
name="B"                        | local subgraph
name="kid_friend/course1"       | remote subgraph "course1" reachable
                                | through federation_secrets[kid="kid_friend"]
```

**Rules:**

- A name without `/` is local. Existing validator (`internal/validator/subgraph.go`, `^[a-zA-Z0-9_]+$`) keeps gating local names — note frontmatter still cannot reference federated subgraphs by mistake.
- A name with `/` is federated. The validator skips this regex for rows inserted by the federation discovery flow (a separate code path, e.g. `InsertFederationSubgraph(kid, remoteName)` that does not go through `case/updatesubgraphs/Resolve`).
- Existing `subgraphs.name unique` constraint is preserved as-is — a slash makes the full string unique even when the suffix repeats across kids.

**Why this shape, repeating it for clarity:** zero migration on `subgraphs`, zero new columns. The kid stays in the name forever; rotation rotates the secret bytes only, kid does not change. Renaming a kid is rare and would require a manual data migration (rare path, document it when needed).

**Parsing in code:** one small helper `splitFederatedName(name) (kid, remote string, isFederated bool)` lives in the federation package. Used by:
- The hub when grouping `ListActiveUserSubgraphs` output by destination base.
- Display surfaces (admin UI) to render remote subgraphs distinctly.

### Use case: selective subgraph exposure

A common scenario: the base owner gave the hub access to several subgraphs (`C` and `D`), but the hub operator wants to expose only one of them (`C`). Configuration is fully on the hub side via selective registration plus offer composition.

**Setup state.** Hub has local subgraphs `A` (route subgraph used by the KB-note's frontmatter) and `B`. The base offers `C` and `D`. Base side has `federation_secret_subgraphs(kid, [C.id, D.id])`.

**Configuration steps on the hub.**

1. **Discovery.** `GET /_system/mcp/federation` returns:
   ```json
   { "kb_url": "https://base.com/_system/mcp",
     "ver": 1,
     "subgraphs": [{"name":"C", ...}, {"name":"D", ...}] }
   ```
2. **Selective registration.** Admin clicks "register" only on `C`. `InsertFederationSubgraph(kid="kid_friend", remote="C")` runs and inserts a row with `name="kid_friend/C"`. `D` is **not** registered. After this:
   ```
   subgraphs (id=1, name="A")                          -- route
   subgraphs (id=2, name="B")
   subgraphs (id=10, name="kid_friend/C")              -- registered remote
   -- D does not exist locally
   ```
3. **Offer composition.** The hub bundles the route grant and the content grant into a single offer:
   ```sql
   offer_subgraphs(offer_id="premium", subgraph_id=1)    -- route A
   offer_subgraphs(offer_id="premium", subgraph_id=10)   -- registered remote kid_friend/C
   ```

**Runtime behavior.**

- User buys "premium" → `user_subgraph_accesses` rows on `(user, A)` and `(user, kid_friend/C)`.
- `ListActiveUserSubgraphs(user)` → `["A", "kid_friend/C"]`.
- User sees the KB-note (passes `canreadnote` via `A`).
- The hub's `federated_search` groups active names via `splitFederatedName`:
  - `A` has no slash → local subgraph, drives local search.
  - `kid_friend/C` → bucket `kid_friend` with remote name `C`.
- JWT for base: header `kid="kid_friend"`, claim `sgr=["C"]`.
- Base computes `effective = sgr ∩ allowed_for_kid = {"C"} ∩ {"C","D"} = {"C"}`. `D` content is never returned.

`D` is unreachable through this hub by construction: there is no local `subgraphs` row for it → no offer can grant it → no user has it active → no JWT carries it.

**Adding `D` later** is forward-only: re-run discovery, click "register" on `D`, attach `(id=11, name="kid_friend/D")` to an existing or new offer.

**If the base later narrows scope (drops `D` from `federation_secret_subgraphs`):** nothing breaks on the hub if `D` was never registered. If `D` had been registered and was attached to an offer, queries from users with `kid_friend/D` in their active set get empty results from that base — and the response carries a warning (see "Out-of-scope warnings"). The hub admin reacts by detaching `D` from the offer.

### Discovery endpoint

```
GET https://base.com/_system/mcp/federation
Authorization: Bearer <federation JWT>
```

The JWT is signed with the same kid+secret used for `federated_*` calls (header carries `kid`, claims may omit `sub`/`sgr` since this is operator-side, not user-side). The endpoint:

1. Parses the JWT, looks up `federation_secrets` by `kid` (newest active row wins — see "Secret rotation"), verifies HMAC.
2. Reads `federation_secret_subgraphs` for that kid.
3. Returns a JSON envelope:

```json
{
  "ver": 1,
  "kb_url": "https://base.com/_system/mcp",
  "last_modified": "2026-04-25T10:15:00Z",
  "subgraphs": [
    {"name": "course1", "description": "Engineering management course"},
    {"name": "course2", "description": "Hiring playbook"}
  ]
}
```

**Both the envelope and each `subgraphs[]` item are objects on purpose**, so the response can grow new fields without breaking older hubs:

- Envelope: `ver` (current = `1`, see "Protocol versioning"), `last_modified`, future `server_info`, `pagination`.
- Per-item: future `price_hint`, `note_count`, `last_modified`, `tags`, `sample_titles`, `requires_signin`. Hubs ignore unknown fields.

Clients MUST treat `subgraphs[]` as an array of objects keyed by `name`; never assume the array is bare strings.

If the JWT is missing or invalid → `401`. If the secret is revoked → `403`. If the secret has no scope rows → `200` with `subgraphs: []`.

The endpoint is intentionally **separate** from `/_system/mcp` (which speaks JSON-RPC) so its purpose is unambiguous and it can evolve independently.

### Discovery refresh (cron)

Discovery isn't only manual. A background job runs **every hour** on the hub and re-fetches `/_system/mcp/federation` for every KB-note paired with a secret. Comparison with the last snapshot:

- New name appeared → admin gets a panel notification ("base added subgraph X — register?").
- Name disappeared → admin gets a red warning ("base removed subgraph X; you have N users with active grants on this name; remove from offers or keep until expiry").
- Description changed → silent update of the local cached description.
- `last_modified` field can be used to short-circuit (`If-Modified-Since` header) when bases support it.

The hub's last snapshot lives in a small kv-style table or a JSON column on the KB-note's secret row; details left to implementation.

### Runtime: what happens on a user's federated_search

When the hub receives a `federated_search` from an authenticated user (Phase 1B mcp_token resolved them):

```
1. active = ListActiveUserSubgraphs(user.id)             // existing function, []string
2. For each name in active:
     kid, remote, ok := splitFederatedName(name)
     if ok:
       buckets[kid] = append(buckets[kid], remote)
     else:
       localNames = append(localNames, name)
3. localNames feed the hub's existing local search.
4. For each (kid, remotes) bucket:
     secret = federation_secrets where kid = bucket_kid AND revoked_at IS NULL
              ORDER BY created_at DESC LIMIT 1
     kb_url = secret.kb_url
     jwt = sign({
       ver: 1,
       iss, iat, exp=now+30s, rid,
       sub: hmac_sha256_b64(secret.bytes, user.email)[:24],
       sgr: remotes,
     }, secret.bytes, kid)
     POST kb_url with the federated_search payload + jwt
5. Merge per-bucket responses with the local search results (RRF).
   Append response.warnings into the hub's structured "context" payload
   so the agent can see them; also log warn-level for the operator.
```

For an unauthenticated request (no mcp_token) the hub still proxies, but `sgr` is empty → the base answers with the public-layer set.

`hmac_sha256_b64(secret_bytes, email)[:24]` produces a stable 24-char base64 pseudonym for the user that both sides can compute without exposing the email. Truncated HMAC: collision-resistant enough for billing/audit reconciliation, irreversible at the base.

### What the base does on inbound

```
1. Parse JWT. If header.alg != "HS256" → 401.
2. Lookup federation_secrets by header.kid:
     all rows where kid = ? AND revoked_at IS NULL
       ORDER BY created_at DESC LIMIT 2
   (Up to two active rows allowed during rotation.)
   Try HMAC verify against each, constant-time compare. None matches → 401.
3. Check exp/iat with 5s skew → expired → 401.
4. allowed = SELECT subgraph_id FROM federation_secret_subgraphs WHERE kid = ?
5. asserted = JWT.sgr (may be empty)
6. effective = asserted ∩ allowed
7. out_of_scope = asserted \ allowed                     // names hub asked for but base won't surface
8. Run the search/note_html/similar pipeline with a precomputed
   "active subgraphs" set = effective. Reuse a sibling of canreadnote.Resolve
   that takes the precomputed set instead of computing it from user_id.
9. Filter results to: notes that pass canreadnote-with-guest-token
                      ∪ notes whose subgraph names overlap effective.
10. Build response with:
      results: [...],
      context: { kb_instructions, kb_id },
      warnings: [
        {code: "subgraph_out_of_scope", subgraph: "<name>"} for each in out_of_scope
      ]
   Log warn-level for each out_of_scope name with kid/iss/rid for the operator.
```

### Out-of-scope warnings

When the hub asks for `sgr=["X"]` and `X` is not in this kid's scope on the base:

- Base **silently drops X from `effective`** (existing intersection rule — never widen).
- Base **adds a warning to the response**: `{code: "subgraph_out_of_scope", subgraph: "X"}`.
- Base logs a warn-level entry with `kid`, `iss`, `rid`.
- Hub receives the response, logs the warning at warn-level too, and surfaces it in admin (e.g. "yesterday users hit base.com 12 times asking for X — base no longer authorizes that name").

This catches the otherwise-silent failure mode where the hub had registered a name that the base later revoked. Hub operator notices it within a day instead of waiting for users to complain.

### Secret rotation

`federation_secrets.kid` is **not** unique — multiple rows may share the same kid as long as they don't overlap with revocation. Runtime semantics:

- **Signing (outbound):** pick the newest non-revoked row for the kid (`ORDER BY created_at DESC LIMIT 1`).
- **Verifying (inbound):** accept up to **2** newest non-revoked rows for the kid. Try them in order, return success on first match.

Rotation procedure (week-long, no downtime):

1. Base operator generates a new row for the same kid: `INSERT INTO federation_secrets (kid, secret_crypt, ...) VALUES (...)`. Now there are two rows: old + new.
2. Base operator hands the new bytes to the hub operator out-of-band.
3. Hub operator inserts the new row on their side. Hub starts signing with the new bytes immediately.
4. Base accepts both old and new for up to a week (anyone still using old bytes during the swap window doesn't break).
5. After the window: `UPDATE federation_secrets SET revoked_at = current_timestamp WHERE kid = ? AND id = old_id` on both sides.

Cron job: prune `revoked_at < now - 30 days` rows to keep the table tidy.

### Revocation

Pure base-side operation:

```sql
-- Drop one subgraph from the kid's scope.
delete from federation_secret_subgraphs where kid = ? and subgraph_id = ?;

-- Revoke ALL active rows for a kid (kills the pairing entirely).
update federation_secrets set revoked_at = current_timestamp where kid = ? and revoked_at is null;
```

Subsequent hub calls hit verification with no matching active row → 401, hub logs error, alerts operator. No coordination protocol; just out-of-band heads-up if the deletion is courteous.

### Reconciliation

Logging only in MVP — `federation_audit_log` is deferred. Operators can extract counts from the application logs (everything is structured) and correlate by `rid` between sides. Once billing arrives, the audit table comes back with proper indexing and reconciliation queries. Until then, the protocol is sufficient for permission enforcement; commercial bookkeeping is a separate concern.

### JWT format

Header: `{"alg": "HS256", "kid": "<federation_secrets.kid>", "typ": "JWT"}`

Claims:
```json
{
  "ver": 1,
  "iss": "https://hub.example.com/_system/mcp",
  "iat": 1234567890,
  "exp": 1234567920,                            // 30s TTL, 5s skew
  "rid": "req-abc-xyz",
  "sub": "OPAQUE-HMAC-PSEUDONYM",               // base64(hmac_sha256(secret_bytes, email))[:24], or omitted
  "sgr": ["course1", "course2"]                 // omitted = empty assertion
}
```

The `sub` claim is a per-pair pseudonym, not the user's email. The hub can resolve it back to an email locally; the base treats it as an opaque string useful only for grouping repeated calls from the same user (audit, debug, future billing). This avoids leaking PII into the base's logs.

### Protocol versioning

Both the JWT (`ver` claim) and the discovery envelope (`ver` field) carry an integer protocol version, currently `1`. Rules:

- Unknown future fields are ignored by both sides.
- `ver` is used to gate breaking changes only. Adding new optional fields does not bump it.
- A receiver seeing a version it does not support → 400 with `{"error": "unsupported version"}`.

This lets the federation evolve over years without simultaneous redeploys.

### Implementation notes

- `federation_secrets.secret_crypt` reuses the trip2g master key encryption used by the Telegram client package. Plaintext bytes only at sign/verify time.
- A receiving base adds **one** new resolve helper: same logic as `canreadnote.Resolve` but takes a precomputed `[]string` of active subgraph names instead of computing them from `user_id`. The federation handler uses this helper; the existing user path is untouched.
- JWT verification: unknown `kid` → 401. Wrong signature → 401. Expired → 401. Revoked secret → 403. All HMAC compares are constant-time (`subtle.ConstantTimeCompare`).
- No `jti` replay cache in MVP. 30s TTL + 5s clock skew is the only protection. Add `jti` if a real abuse case appears.
- Inbound federation auth is independent of local `search` / `similar` / `note_html`, which keep their existing public/authenticated behavior.

### Considered alternatives (rejected)

- **OAuth / RFC 8693 token exchange.** Standardized, granular, but heavyweight; defer until an enterprise customer explicitly demands it.
- **Per-(hub, base) shared secret with no scope table.** Same secret authorizes every subgraph on the base. Rejected: forces operators to mint a new secret for every team boundary; scope table is cheap and avoids the proliferation.
- **Service bearer token in KB-note frontmatter.** Tempting for a 1-day MVP, but mixes config with credential and complicates Obsidian sync. The DB-backed secret store with admin-panel input is barely more code.
- **`subgraphs.kb_url` column for federated rows.** Earlier draft. Rejected because it requires a sqlite migration on a hot table; encoding the kid as a `kid/remote` prefix in `subgraphs.name` reaches the same outcome with zero migration. The kid never appears in note frontmatter (validator still bans `/`), so users can't accidentally reference federated subgraphs as if they were local.
- **Pure user-delegation (`sgr` removed, base resolves grants by `sub`).** Works only when the buyer has an account on the publisher. In cross-owner reselling the buyer paid the reseller, not the publisher; the publisher has no row for them. The hub-asserted `sgr` claim is what makes reselling possible at all.
- **Per-request audit table in MVP.** Deferred to billing phase to avoid SQLite write contention and because reconciliation is a billing concern. Application logger covers the operational observability need.

---

## Future ideas (capture)

- **"AI-router for your systems in a week"** — federation as a pitch to mid-size companies that have knowledge spread across Notion, Confluence, GitHub Wiki, etc. Position trip2g hub + adapters as the routing layer, no ETL needed.
- **Mesh of additional sources via adapters** — Telegram channels, git docs, Google Docs, Slack archives, Substack RSS, Linear, Confluence. Each adapter is a small MCP-speaking service that fronts an external system. Hub federates to all of them uniformly.
- **Marketplace of expert KBs** — experts host their own bases; a marketplace hub bills per-query and routes to relevant expert bases. Federation makes the routing trivial.
- **Compliance-friendly RAG** — bases stay in their own jurisdiction; hub never holds data, only proxies. KB-notes can carry policy hints (`mcp_federation_kb_jurisdiction`) for filtering at routing time.
- **Living textbooks** — publishers expose books as MCP bases; courses are hubs that compose book sets via KB-notes. Authors update notes in Obsidian; students get the new content the same day.
- **`/_system/federation` mesh probe** (see Stage 3) — endpoint that returns the topology, recursively. Useful both for ops and as a basis for visualization tools.

---

## Test plan

### Reference fixture

Balanced tree with one skew:

```
                          hub
            ┌──────────────┼──────────────┐
            │              │              │
        sub-A          sub-B          sub-C
        ┌─┼─┐          ┌─┼─┐          ┌─┼─┐
       A1 A2 A3       B1 B2 B3       C1 C2 C3
                                              │
                                          C3-leaf
                                       (extra level)
```

- 1 hub (root).
- 3 sub-hubs (`sub-A`, `sub-B`, `sub-C`), each with `mcp_federation_kb_max_depth: 1`.
- Each sub-hub has 3 KB-notes pointing to leaves (`A1..A3`, `B1..B3`, `C1..C3`).
- One leaf, `C3`, is itself a sub-hub pointing to `C3-leaf`. Uneven tree.
- Every leaf hosts uniquely identifiable notes with known `note_id`s and titles.

`httptest.NewServer` per node. URLs passed dynamically (resolved at startup).

### Scenarios

| # | Scenario | What we check |
|---|----------|---------------|
| 1 | `tools/list` without KB-notes | Returns all 6 methods. |
| 2 | `tools/list` with KB-notes | Same 6 methods (cache-stable contract). |
| 3 | `federated_search` with no KB-notes | Returns "federation not configured" structured response. No error. |
| 4 | `search` finds a KB-note | Result has `kind="federation_kb"`, `federation.kb_id`, `federation.kb_url`, `federation.agent_instruction`. |
| 5 | `search` does NOT return KB-note as plain content | KB-note's `kind` is always `federation_kb`. |
| 6 | `federated_search(kb_id="sub-A")` | Direct proxy. Results come from `A1..A3`. Each result's `kb_id` starts with `"sub-A"`. |
| 7 | `federated_search(kb_ids=["sub-A","sub-B"])` | Parallel to two sub-hubs. Results from A and B only, never C. |
| 8 | `federated_search` (no kb_id, no kb_ids) | Full fan-out. Results from all 3 sub-hubs. KB-notes absent from response. |
| 9 | `federated_search` with both `kb_id` and `kb_ids` | `InvalidParams` error. |
| 10 | `federated_search(kb_id="sub-C/C3")` | Two-level proxy: hub → sub-C → C3-leaf. C3-leaf content returned. |
| 11 | `federated_note_html(pid=N, kb_id="sub-A")` | HTML matches what sub-A returns directly. |
| 12 | `federated_note_html(pid=N, kb_id="sub-C/C3")` | Two-level HTML proxy. |
| 13 | `federated_similar(pid=N, kb_id="sub-B")` | Similar via two-level chain. |
| 14 | `federated_similar` / `federated_note_html` without `kb_id` | `InvalidParams`. |
| 15 | Reverse prefix rewriting | After fan-out, results from sub-C show `kb_id="sub-C/Ci"`, never `kb_id="Ci"`. |
| 16 | URL is real host | Result URLs point at sub-base hosts, not at the hub. |
| 17 | Per-base `kb_instructions` propagated | After `federated_search(kb_id="sub-A")`, response payload contains `context.kb_instructions` matching sub-A's `initialize` note. |
| 18 | `kb_instructions` cached | Second call to same `kb_id` doesn't re-fetch base's `initialize`. |
| 19 | One sub-base errors | Other sub-bases still return; warn logged for the failing one. |
| 20 | One sub-base times out | Other sub-bases still return; warn logged. |
| 21 | Empty result from a sub-base | No panic; merge correct. |
| 22 | Cycle (Stage 2) | sub-A's KB-note points back at hub. Each cycle hit logs `WARN cycle detected`. Result is empty for that branch, rest continues. |
| 23 | `max_depth` (Stage 2) | KB-note with `mcp_federation_kb_max_depth: 0` is queried directly; its own KB-notes not queried. |
| 24 | Global `MCP_FEDERATION_MAX_DEPTH` (Stage 2) | Env ceiling clips recursion regardless of per-KB depth. |
| 25 | Fan-out parallelism | All 3 sub-hubs respond after 1s each; total time ≈ 1s, not 3s. |
| 26 | Uneven tree fan-out | Full fan-out from hub returns results from all 9 leaves + C3-leaf if depth allows; ≤ 9 if not. Both paths tested. |

### KB-note visibility ACL (per-user)

| # | Scenario | What we check |
|---|----------|---------------|
| 27 | Anonymous can't see KB-note in restricted subgraph | KB-note has `subgraphs: [premium]`; guest `search` does not return it; `federated_search(kb_id=…)` to it returns "federation not configured for this kb_id" structured payload. |
| 28 | Authenticated user without grant: same as anonymous | User logged in via `mcp_token` but no access to `premium`; KB-note still hidden; targeted `kb_id` indistinguishable from non-existent. |
| 29 | User with grant sees KB-note and can route | User has `user_subgraph_accesses` on `premium`; `search` returns the KB-note with `kind="federation_kb"`; `federated_search(kb_id=…)` succeeds. |
| 30 | Fan-out only iterates accessible KB-notes | Hub has 3 KB-notes in different subgraphs (`free`, `premium`, `enterprise`). User has only `premium`. Fan-out hits 2 bases (free + premium), never the enterprise one. |
| 31 | KB-note ACL change propagates per request | Toggle subgraph membership on a KB-note between calls; visibility flips on the next call without restart. |

### Selective remote subgraph registration (use case)

Fixture: hub plus a single base. Base has subgraphs `C` and `D`, both in the secret's scope. Hub registers only `C`.

| # | Scenario | What we check |
|---|----------|---------------|
| 32 | Discovery returns C and D | `GET /_system/mcp/federation` with valid JWT lists both, with descriptions. |
| 33 | Selective registration creates one row | After clicking "register" on C only, hub's `subgraphs` table has exactly one federated row with `name="kid_friend/C"`. No row for D. |
| 34 | User offer composition grants route + content | Offer `premium` attaches local route subgraph `A` and registered remote `C`. Buying it (via `processnowpaymentsipn`) creates two `user_subgraph_accesses` rows. |
| 35 | sgr never carries D | User with `premium` calls `federated_search`. Outbound JWT to base has `sgr=["C"]` only. Verified by intercepting the JWT in the test base. |
| 36 | Base intersection drops out-of-scope D | Even when a malicious or stale hub injects `sgr=["D"]`, the base filters via `effective = sgr ∩ allowed_for_kid`. With D registered base-side, sgr=[D] succeeds; with D revoked base-side, sgr=[D] returns empty. Both transitions tested. |
| 37 | Adding D later is forward-only | After running registration for D and attaching it to a new offer, fresh users buying that offer get D in their active set; pre-existing offer behavior unchanged. |
| 38 | Base scope shrinks with no hub-side change | Base deletes `federation_secret_subgraphs` row for D. Hub's queries for users who had `kid_friend/D` in their active set return empty results plus a `subgraph_out_of_scope` warning in the response. Hub logs the warning at warn-level. No code change on hub side. |

### JWT verification edge cases

| # | Scenario | What we check |
|---|----------|---------------|
| 39 | Unknown kid → 401 | JWT signed with a kid that doesn't exist in `federation_secrets`. App log at warn level with `kid`, `iss`, `rid`. |
| 40 | Wrong signature → 401 | JWT body untampered but signed by a different secret. |
| 41 | Expired JWT → 401 | exp in the past beyond skew. |
| 42 | Revoked secret → 403 | `federation_secrets.revoked_at` set; verification still happens, then refused. |
| 43 | Discovery endpoint without JWT → 401 | `GET /_system/mcp/federation` with no Authorization header. |
| 44 | Discovery for empty scope → 200 with `subgraphs: []` | Valid kid with no `federation_secret_subgraphs` rows; not a 4xx. |
| 45 | Secret rotation: two active rows for one kid | Insert new `federation_secrets` row with same kid, both `revoked_at IS NULL`. Hub signs with newer; base accepts both during the swap window. Old-secret JWT still verifies until that row is revoked. |
| 46 | Rotation: only newest signs | After step 45, `federation_secrets` has two rows. Outbound signing always picks the newest by `created_at`. Older row is for verification only. |
| 47 | Rotation: more than two active rows | If a third row exists, the verifier still tries only the two newest. Oldest is treated as inactive even without `revoked_at`. |
| 48 | `sub` is per-pair HMAC pseudonym | Hub mints two JWTs for the same user against two different kids; `sub` differs because secret bytes differ. The same (user, kid) pair always produces the same `sub`. Email is never present in the JWT. |
| 49 | `ver: 1` accepted; unknown `ver` rejected | Version 1 JWT works. JWT with `ver: 99` returns 400 `unsupported version`. Discovery envelope behavior is symmetric. |
| 50 | Out-of-scope warning round-trip | Hub sends `sgr=["kid_friend/X"]` where X is not in scope. Base returns 200 with `warnings: [{code:"subgraph_out_of_scope", subgraph:"X"}]`. Hub records a warn-level log entry with `kid`, `rid`, `subgraph`. |
| 51 | Discovery cron diff | Snapshot before: `[C, D]`. Cron re-fetches; base now exposes `[C, E]`. Hub admin gets two notifications: "D removed" (with grant-impact count), "E added (register?)". |
| 52 | `splitFederatedName` parsing | Unit: `"A"` → local. `"kid/x"` → federated kid="kid", remote="x". `"kid/x/y"` → federated kid="kid", remote="x/y" (only first slash splits). `""` → invalid. |

### Layout

`internal/case/mcp/federation_test.go` — main file.

```go
type federationTestbed struct {
    hub    *httptest.Server
    subs   [3]*httptest.Server  // sub-A, sub-B, sub-C
    leafs  [9]*httptest.Server  // A1..A3, B1..B3, C1..C3
    c3Leaf *httptest.Server
}

func newFederationTestbed(t *testing.T) *federationTestbed { ... }
```

Each fixture node is a full mock MCP server: accepts JSON-RPC, answers per protocol. KB-notes for each node are generated programmatically so neighbor URLs can be substituted after servers come up.

Existing test infrastructure (`mocks_test.go` via `moq` for `Env`) is reused for unit tests of standalone helpers (`splitKBID`, `prefixKBID`, `newMCPFederationNote`, `hostnameFromURL`).

---

## Prior art

- **Hierarchical RAG / Federated Search** — the general pattern.
- **GraphRAG** (Microsoft) — hierarchy inside a single store, not across servers.
- **RAPTOR** (Stanford) — hierarchical document clustering inside one index.
- **Glean, Hebbia** — enterprise search; centralize the index, harder to satisfy data residency.

What's distinctive here: hierarchy across **independent MCP servers**, no shared index. Each base is self-contained; federation is a thin layer on top. This mirrors trip2g's actual structure.
