# Federated Search in the Site Frontend — Design

**TL;DR.** Yes, we can give human visitors federated search, reusing the MCP fan-out that agents already get. Recommended UX: keep local search exactly as it is, and add an explicit **"Also search N connected bases"** expander under the local results (progressive disclosure). Federated results render as **per-base sections**, each with a base badge (favicon + host), plain-text snippets, and absolute deep links to the peer's own site. A status line shows "N of M bases answered" and names the bases that timed out (fail-loud). Backend: extract the existing MCP fan-out into a shared `internal/fedsearch` service package and expose it as a `federatedSearch` GraphQL query — no cross-base rerank in v1, per-peer order is trusted.

Status: design only, not built. Written 2026-07-11.

## 1. Current state

### Local site search (what visitors have today)

- The search UI is a $mol island injected by the default template: `internal/defaulttemplate/views.html:421-423` mounts `$trip2g_user_search`.
- The widget (`assets/ui/user/search/search.view.ts:2-14`) fires a `search(input: {query})` GraphQL query, debounced 1s, min 3 chars (`search.view.ts:56-76`). Results render as an overlay list with highlighted title/content, score, and a "read more" link (`search.view.tree:38-59`).
- Backend: `internal/case/sitesearch/resolve.go:50` — bleve text lane + optional vector lane, RRF-fused (`resolve.go:290`), optional cross-encoder rerank (`resolve.go:350`), then per-result permission filtering. Local only — no federation anywhere in this path.
- GraphQL surface: `SearchInput { query }` → `SearchConnection { nodes: [SearchResult] }` with `highlightedTitle`, `highlightedContent`, `url`, `score`, `matchOrigin` (`internal/graph/schema.graphqls:1422-1447`).

### Federation (what agents have, MCP only)

- Peers are declared as **KB-notes**: any note with `mcp_federation_kb_url` (+ optional `mcp_federation_kb_id`) frontmatter (`internal/model/mcp_federation_note.go:16-24`). The hub's router notes live in `docs/en/hub/*.md`. The registry is just `LatestNoteViews().MCPFederationNotes`, ACL-filtered per requester via `accessibleKBNotes` (`internal/case/mcp/federation_acl.go:15`).
- `federated_search` (MCP tool, `internal/case/mcp/federation_handlers.go:11`) fans out `search` calls to all accessible KBs. On `main` (PR #169) the fan-out is bounded: `errgroup.SetLimit(concurrency)` + per-peer timeout with a slot-releasing goroutine (`internal/case/mcp/federation.go`, main: `fanout`/`callPeer`), and blind fan-outs are capped (`capBlindFanout(selected, env.FederatedFanoutLimit())`, main `federation_handlers.go:36`). Defaults: concurrency 5, limit 7 peers, 5s per-peer timeout (`internal/appconfig/config.go:527-531` on main).
- Results are aggregated with per-KB attribution and per-KB errors — `FederatedCallPayload { results: [{kb_id, result, latency}], errors: [{kb_id, error}] }` (`internal/case/mcp/types.go:130-145`, `federation.go:103`). Partial results are already first-class: a response is only an error if *every* peer failed.
- Each peer answers its own `search` tool (`internal/case/mcp/resolve.go:460`) — the same hybrid lanes + permission filter as site search — and returns `SearchResultPayload` with `title`, `note_path`, **absolute `url`**, `score`, `matches[].snippet` (`types.go:147-163`). So deep links to the peer's own site come back ready to use.
- Transport: `internal/federation/client.go` — JSON-RPC over HTTP, SSRF-safe dialing, optional HMAC-signed JWT, `X-MCP-Federation-Depth` header. The fan-out calls the peer's plain `Search` (not `FederatedSearch`), so there is **no recursive cascade** — one hop only. Anonymous peers see only their public notes (peer-side `filterSearchResults`, `resolve.go:531`).

The gap: all of this is reachable only through the MCP endpoint. GraphQL has no federated query, and the template widget knows nothing about peers.

## 2. UX design

### Recommended pattern: progressive disclosure + grouped-by-base sections

Weighing the four options from the brief:

| Option | Verdict |
|--------|---------|
| (a) merged list, per-result KB badge | Attribution is right, but merging requires comparable ranking across bases (scores aren't comparable — different corpora, different lanes) and fights progressive arrival: late peers reshuffle the list under the user's cursor. |
| (b) grouped-by-base sections | Maps 1:1 onto how results actually arrive (per-peer, partial, some timing out). Each section is independently appendable. Attribution is structural, not a decoration. |
| (c) local-first + explicit "search the federation" expander | Solves the latency and privacy problems at the root: the slow, query-leaking fan-out never runs unless the visitor asks. Local search stays instant-as-you-type. |
| (d) kb picker / filter chips | Useful, but a power-user refinement — not the entry point. |

**Recommendation: (c) as the trigger, (b) as the presentation, with (a)'s badge anatomy inside each section.** Concretely:

1. Local search behaves exactly as today (debounced live overlay).
2. When the site has ≥1 accessible KB-note, a row appears **under** the local results:
   > 🌐 **Also search 7 connected bases** — your query will be sent to those sites.
   One explicit click (or Enter on it) fires the fan-out. Never fired per-keystroke.
3. While running: a status line "Searching connected bases… 2 of 7 answered" + one skeleton block.
4. Done: per-base sections appended below local results, plus a fail-loud footer when partial:
   > ⚠ 5 of 7 bases answered. No answer from **nietzsche.2pub.me** (timeout), **laozi.2pub.me** (error).

Why this wins: it matches the fan-out's real behavior (bounded, partial, 0–5s per peer) instead of hiding it; it matches the attribution-first research finding (source identity *is* the value — a Nietzsche hit and a Rockefeller hit for the same query mean different things); and it keeps the default path fast and private.

### Result-item anatomy

```
┌───────────────────────────────────────────────────────────┐
│ ▸ [favicon] Ницше · nietzsche.2pub.me            (kb_id)  │  ← section header, once per base
│───────────────────────────────────────────────────────────│
│   Title of the note                                        │  ← link = peer's absolute url
│   …plain-text snippet from matches[0].snippet…             │
│   nietzsche.2pub.me/notes/will-to-power ↗                  │  ← visible host, external-link affordance
└───────────────────────────────────────────────────────────┘
```

- **Section badge** = KB-note title (the hub already curates human names in `docs/en/hub/*.md`) + host from the result URL + favicon (`https://<host>/favicon.ico`, lazy `<img>` with fallback glyph). `kb_id` shown small/secondary — it's an agent-facing identifier.
- **Title**: plain text from `SearchResultItem.Title`. No `<mark>` highlighting — peers return plain snippets, and we must not innerHTML remote content anyway (see Risks).
- **Snippet**: `matches[0].snippet`, rendered as **text**, never HTML.
- **Link**: `SearchResultItem.URL` — already absolute to the peer's site. Open in the same tab (it's still "the federation", not an ad); `↗` marks it as leaving the current site.
- **No score display.** Scores aren't comparable across bases; showing "3/10" from another corpus misleads. Per-base order is the peer's own ranking — keep it.
- Cap each section at ~5 results with a "more on this site ↗" link to the peer's own search (later; v1 just caps).

### Latency and partial results

- Fan-out worst case = per-peer timeout (5s default). The UI must not look frozen: show the "N of M answered" line immediately (M is known up front from the KB registry) and a skeleton.
- **v1: one request, one response.** The bounded fan-out already returns everything (successes + per-KB errors) in a single payload after at most ~5s. The status line during the wait is a simple spinner + "searching M bases…"; true incremental arrival is deferred.
- **Later: streaming.** The stack already does GraphQL subscriptions over SSE (`docs/dev/gqlgen_fasthttp.md`); a `federatedSearchStream` subscription can emit one event per peer as `fanout` completes each slot, and sections appear one by one. That's an additive change — same service, different transport.
- **Degradation is explicit, never silent:** timed-out/errored bases are named in the footer with a retry affordance ("try again" re-runs the fan-out for failed KBs only — later; v1 re-runs all). If *all* peers fail: "No connected base answered. Try again." If federation isn't configured, the expander simply doesn't render.

## 3. Backend shape

### Extract a shared service: `internal/fedsearch`

Today the fan-out logic (registry read + ACL + cap + bounded fan-out + per-KB aggregation) lives inside `internal/case/mcp`. Duplicating it in a GraphQL resolver would fork the bounding policy. Instead, per the service-package pattern (`docs/dev/app_patterns.md:26-72`):

```go
// internal/fedsearch/fedsearch.go
type Env interface {
    LatestNoteViews() *model.NoteViews
    CanReadNote(ctx context.Context, note *model.NoteView) (bool, error)
    FederationClient(ctx context.Context, kbID string) (model.Federation, error)
    FederatedFanoutConcurrency() int
    FederatedFanoutLimit() int
    FederatedFanoutTimeout() time.Duration
}

type FedSearch struct{ env Env }

// KBs returns the ACL-filtered registry (for the UI's "N connected bases" count).
func (f *FedSearch) KBs(ctx context.Context) ([]*model.MCPFederationNote, error)

// Search fans out query to the given (or all) KBs; always returns partial results.
func (f *FedSearch) Search(ctx context.Context, query string, kbIDs []string) ([]KBResult, error)

type KBResult struct {
    KB      *model.MCPFederationNote
    Status  Status        // OK | ERROR | TIMEOUT | SKIPPED (capped)
    Latency time.Duration
    Results []model.FederatedSearchItem // parsed from peer StructuredContent
    Err     string
}
```

- Move `fanout`/`callPeer` (main `internal/case/mcp/federation.go`), `selectFederationKBs`, `capBlindFanout`, and `accessibleKBNotes` (`federation_acl.go:15`) into the package. `internal/case/mcp` keeps only MCP payload formatting (`aggregateFederationResults`, `federation.go:103`) on top of `[]KBResult`.
- The service **parses the peer's `StructuredContent`** into typed `SearchResultItem`s (`internal/case/mcp/types.go:152-163`) once, in one place. The MCP handler re-serializes; the GraphQL resolver maps to models. Unparseable structured content → `Status: ERROR` for that KB (fail-loud, not silent empty).
- Embed anonymously in `app` (`*fedsearch.FedSearch`), compile-time `var _ fedsearch.Env = (*app)(nil)`. No proxy methods.

### GraphQL surface

```graphql
type FederationKB { id: String!, title: String!, host: String! }

input FederatedSearchInput { query: String!, kbIds: [String!] }

enum FederatedKBStatus { OK, ERROR, TIMEOUT, SKIPPED }

type FederatedKBResult {
  kb: FederationKB!
  status: FederatedKBStatus!
  latencyMs: Int!
  results: [FederatedSearchResult!]!   # empty unless OK
  error: String                        # human-readable, fail-loud
}

type FederatedSearchResult {
  title: String!
  url: String!        # absolute, peer's own site
  snippet: String     # plain text
}

extend type Query {
  federationKbs: [FederationKB!]!                      # drives the expander + "M bases"
  federatedSearch(input: FederatedSearchInput!): [FederatedKBResult!]!
}
```

Resolvers stay thin (`app_patterns.md:173-189`): delegate to `fedsearch.Search`. Per-KB failures are data (`status`/`error` fields), not GraphQL errors — the partial-results contract carries through to the client. A whole-request `error` is reserved for infrastructure failure.

### Frontend

Extend `$trip2g_user_search_panel`: after local results, query `federationKbs` (cheap, local, cached per session); if non-empty, render the expander row. Click → `federatedSearch` request via `$trip2g_graphql_request` → per-KB `Section*` sub-views. Standard view.tree + view.ts split, localized strings via `@` markers (`assets/ui/user/search/search.view.tree` conventions).

### The rerank question

Should we rerank across merged federated results locally? **v1: no. Trust per-peer order, don't merge.**

- Cross-base raw scores are incomparable (different corpora, different lanes, different reranker configs) — that's why the design groups instead of merging.
- RRF over per-peer *ranks* would be a legitimate merge (rank-only, no score normalization — same trick as `sitesearch/resolve.go:287-290`), but it still buries source identity, which the grouped UX treats as the point.
- A genuinely comparable merged list needs a **local cross-encoder pass** over `(query, snippet)` for all federated results — `reranker.BlendRRF` machinery exists (`internal/case/sitesearch/resolve.go:350-358`), snippets are window-sized-ish, so it's feasible. But it adds a serial rerank hop after the slowest peer, and the reranker is optional infrastructure (and was removed from the default deployment as measured-worse for local search). Park it as a possible "merged view" toggle later, gated on `Features().VectorSearch.Reranker.Enabled`.

## 4. Scope: v1 vs later

**v1 (minimal):**
1. `internal/fedsearch` service package extracted from `internal/case/mcp` (pure refactor + one new parse step); MCP handlers rewired on top — behavior-identical, existing MCP tests keep passing.
2. `federationKbs` + `federatedSearch` GraphQL queries (public, same visibility rules as MCP anonymous access).
3. Search panel: expander row with privacy hint → single fan-out request → grouped per-base sections, plain-text snippets, absolute links, "N of M answered" + named failures footer.
4. Per-IP rate limit on `federatedSearch` (see Risks).

**Later:**
- Per-KB filter chips / picker (option d) and per-section "retry this base".
- SSE streaming (`federatedSearchStream`) for true progressive arrival.
- Optional cross-encoder merged view; "federated similar" on note pages.
- Short-TTL response cache keyed `(query, kbID)`.
- Peer favicon/name caching in the KB registry instead of live `<img>` fetches.

## 5. Risks

1. **Privacy — query leakage (the main one).** Every federated search sends the visitor's query string to up to 7 third-party servers, with the hub's federation identity attached. Agents opted into this; a human typing into "the site's search box" has not. Mitigations: fan-out only on explicit click; the expander row itself states "your query will be sent to N connected sites"; never auto-federate as-you-type; document it in the hub notes.
2. **Abuse / amplification.** One anonymous HTTP request triggers up to 7 outbound requests — a cheap traffic amplifier against peers and against our own egress. Mitigations: existing bounds (concurrency 5 / cap 7 / 5s timeout) + per-IP rate limit on the GraphQL query (e.g. a few fan-outs per minute; the pattern of `signinCounter`, `app_patterns.md:74-90`) + min query length 3 (same as local).
3. **Untrusted remote content.** Peer snippets/titles must be rendered as text, never innerHTML (the local `$trip2g_user_search_dimmer` path innerHTMLs highlighted content — federated results must not reuse it). URLs come from the peer; render as-is in `href` but they're already the peer's own domain by construction (KB-note declares the endpoint; SSRF-safe dialing in `internal/federation/client.go:39`).
4. **Latency perception.** 5s worst case behind one click is acceptable *only* with the progress line and named-failure footer; without them it reads as broken.
5. **Registry drift.** Dead peers make the feature look bad ("2 of 7 answered" forever). The fail-loud footer doubles as the operator's signal to prune KB-notes; later, surface per-KB error rates in the admin metrics that `metrics.go` already records.

## References

- `internal/case/sitesearch/resolve.go:50,287,350` — local hybrid search, RRF, rerank blend
- `assets/ui/user/search/search.view.ts:2,56` — search widget, GraphQL query, debounce
- `internal/defaulttemplate/views.html:421` — widget mount point in the template
- `internal/graph/schema.graphqls:1422-1447` — current search GraphQL types
- `internal/case/mcp/federation_handlers.go:11` — `federated_search` MCP handler (main: cap + bounded fanout at :36-38)
- `internal/case/mcp/federation.go` (main) — `fanout`/`callPeer`, `aggregateFederationResults`
- `internal/case/mcp/federation_acl.go:15` — ACL-filtered KB registry
- `internal/case/mcp/types.go:130-163` — per-KB payloads, `SearchResultItem` (absolute `url`)
- `internal/model/mcp_federation_note.go:16` — KB-note frontmatter extraction
- `internal/federation/client.go:33,49` — peer client, timeouts, SSRF-safe, HMAC auth
- `internal/appconfig/config.go:527-531` (main) — fan-out defaults: concurrency 5, limit 7, 5s
- `docs/dev/app_patterns.md:26,173` — service-package + thin-resolver rules
