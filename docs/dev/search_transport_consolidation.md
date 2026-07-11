# Search Transports: GraphQL vs MCP — Consolidation Analysis

**TL;DR.** Do NOT move the site frontend to MCP search. The premise "both are thin wrappers over shared internals" is half-wrong: GraphQL `search` is a 1-line wrapper over `sitesearch.Resolve`, but MCP `search` is a **separate ~200-line duplicate retrieval pipeline** that only shares low-level primitives. The right move is the inverse consolidation (option c): extract one shared retrieval engine, keep both transports as thin adapters. Effort ≈ 0.5–1 day. Bonus: it fixes a real behavioral divergence (MCP always searches *latest* notes, even for anonymous clients, where the site correctly serves *live*).

Status: analysis only, nothing changed. Written 2026-07-11.

## 1. The actual call graph

There are **three** consumers of search, not two:

| Surface | Entry point | Engine used |
|---------|-------------|-------------|
| GraphQL `search(input:)` | `internal/graph/schema.resolvers.go:3205-3207` | `sitesearch.Resolve` — literally one line |
| HTML `/search?q=` page | `internal/case/rendersearchpage/endpoint.go:35` | `sitesearch.Resolve` (64-line endpoint, `Env = sitesearch.Env`) |
| MCP `search` tool | `internal/case/mcp/resolve.go:489` (`handleSearch`) | **its own pipeline**, not `sitesearch` |

Also: GraphQL `notePaths(filter:{search})` reuses the Search resolver (`schema.resolvers.go:3230`), so GraphQL search has a second internal consumer.

### What MCP duplicates instead of reusing

`internal/case/mcp/resolve.go` contains a parallel copy of the whole first-stage retrieval:

- `handleSearch` (`resolve.go:489-557`): text lane → vector lane → RRF merge → rerank → filter, same shape as `sitesearch.Resolve` (`internal/case/sitesearch/resolve.go:50-158`).
- `vectorSearch` / `vectorResultsFromChunks` (`resolve.go:1216-1290`) — duplicate of `sitesearch` `vectorSearch` (`resolve.go:166-257`).
- `mergeResults` RRF fusion (`resolve.go:1318-1369`) — duplicate of `sitesearch.mergeResults` (`resolve.go:290-344`).
- Verbatim helper copies: `chunkPassage`, `snippetFromChunk`, `trimWhitespace`, `lastIndexByte` (`resolve.go:1292-1399` vs `sitesearch/resolve.go:262-429`).

The only genuinely shared pieces are the primitives both call: `env.SearchLatestNotes/SearchLiveNotes` (bleve), `env.OpenAI().CreateEmbedding`, note chunks/views, and the second-stage `reranker.BlendRRF` (`sitesearch/resolve.go:357`, `mcp/resolve.go:524`).

### Divergences the duplication has already produced

| Aspect | sitesearch | MCP |
|--------|-----------|-----|
| Latest vs live corpus | chosen by `ShowDraftVersions \|\| admin` (`resolve.go:58`) | **always latest** (`resolve.go:502,513,1225`) — anonymous MCP clients search draft content of public notes |
| Vector similarity | `dotSimilarity` (vectors are L2-normalised) (`resolve.go:436`) | `cosineSimilarity` with redundant norms (`resolve.go:1404`) |
| Stale-embedding guard | skips dim-mismatched chunks + warns (`resolve.go:198-207`) | none — mismatched dims silently score 0-ish garbage |
| Sort determinism | tie-break by URL/path (`resolve.go:211-216,336-341`) | no tie-break (`resolve.go:1254,1360`) |
| Unreadable notes | "Закрытый материал." placeholder pushed to end (`resolve.go:129-140`) | silently dropped (`filterSearchResults`, `resolve.go:559`) |
| Scoped-token `read_patterns` | enforced fail-closed (`resolve.go:108-113`) | different model: API key = see-all, federation JWT = subgraph ACL, anon = `CanReadNote` (`canReadMCPNote`, `resolve.go:631`) |
| Output shape | `SearchConnection` with bleve HTML-highlighted title/content | `SearchResultPayload` with plain snippets, `match_id`, `toc_path`, `limit`/`detailLimit` (`resolve.go:641-688`) |

The dot-vs-cosine and missing dim-guard rows are the smoking gun: fixes landed on one side and never reached the other. That is the real cost of the status quo.

## 2. What the frontend actually uses

Public site search is the $mol island `$trip2g_user_search`, mounted by the default template (`internal/defaulttemplate/views.html:421-423`). It fires the GraphQL query `SiteSearch` per keystroke (1s debounce, min 3 chars) via `$trip2g_graphql_request` and renders `highlightedTitle` / `highlightedContent` — **HTML `<mark>` fragments from bleve** (`assets/ui/user/search/search.view.ts:2-14,56-76`). Anonymous visitors work because GraphQL search needs no auth (`CurrentUserToken` handles anonymous; permission filtering is per-note).

Switching this widget to the MCP endpoint would require:

- a JSON-RPC 2.0 client in the template bundle (MCP is same-origin, so CORS is a non-issue, and anonymous MCP access exists — `internal/case/mcp/endpoint.go:150-154`);
- losing HTML match highlighting (MCP returns plain snippets — by design, agents must not get injectable remote HTML);
- losing the "closed material" placeholder UX and the live/latest draft logic;
- keeping GraphQL search anyway for `notePaths(filter:{search})` and the `/search` HTML page — so nothing gets deleted.

## 3. Options

**(a) Frontend → MCP, remove GraphQL `search`.** Worst of all worlds. The MCP pipeline is the *less correct* of the two (always-latest corpus, no dim guard); the frontend loses highlighting and hidden-result UX; a JSON-RPC client must be added to the visitor bundle; `notePaths` and `/search` still need the engine so no code is deleted; and it contradicts the federated-search design (`docs/dev/federated_search_ui.md`), which — written today — proposes a **new** `federatedSearch` GraphQL query as the human-visitor transport. Effort: days. Value: negative.

**(b) Status quo.** Zero immediate effort, but the "two thin transports" framing is false — it is two *engines*, and they have already drifted (table above). Every retrieval improvement (reranker tuning, RRF changes, embedding-model migration) must be applied twice or silently applies once.

**(c) Inverse consolidation — one engine, two thin adapters.** Extract the candidate-generation pipeline (text lane + vector lane + RRF + `BlendRRF` rerank) into a single shared function; `sitesearch.Resolve` keeps its permission/placeholder/output logic on top, and MCP `handleSearch` calls the same engine then applies `canReadMCPNote` + `buildSearchPayload`. The transport-specific parts (GraphQL types with HTML highlighting vs MCP `match_id`/`toc_path` payload; session auth vs API-key/federation auth) are legitimately different and stay. Fold in the latest/live fix: anonymous MCP should search *live* notes like the site does; API-key/admin keeps *latest*.

## 4. Recommendation

**Option (c).** The owner's self-correction was directionally right but factually optimistic — the internals are *not* shared today, and that is precisely the problem worth fixing. Removing GraphQL search solves nothing (the engine must survive for `/search` and `notePaths`, and the fed-search UI plan builds *on* GraphQL), while unifying the retrieval core removes ~200 duplicated lines from `internal/case/mcp`, closes the dot/cosine and dim-guard drift, and gives the federated-search work a single engine to call. This also matches the in-flight `internal/fedsearch` extraction proposed in `docs/dev/federated_search_ui.md` §3 — the same service-package move, one layer down.

**Migration sketch (0.5–1 day):**

1. Extract `retrieve(ctx, env, query, useLatest) ([]model.SearchResult, passageByURL, error)` — the text+vector+RRF+rerank core — from `sitesearch` (either exported from `sitesearch` or a new `internal/search` service package per the gitapi pattern). `sitesearch.Env` already declares everything it needs.
2. `sitesearch.Resolve` becomes: corpus choice → `retrieve` → scope/permission filter → placeholders → caps. (Mostly unchanged.)
3. MCP `handleSearch` becomes: corpus choice (`mcpAPIKeyAuthed` → latest, else live) → `retrieve` → `filterSearchResults` → `buildSearchPayload`. Delete `mcp` copies of `vectorSearch`, `vectorResultsFromChunks`, `mergeResults`, `snippetFromChunk`, `chunkPassage`, `trimWhitespace`, `lastIndexByte`, `cosineSimilarity` (`resolve.go:1216-1421`).
4. Keep both transports' contract tests; add one test asserting both surfaces rank identically for the same query/corpus.

**Trade-offs of (c):** the shared engine gains a `useLatest` parameter and both callers' expectations become coupled — a retrieval change now affects agents and visitors simultaneously (that is the point, but it means rerank experiments hit both surfaces; if a surface ever needs different retrieval behavior, add an options struct, don't re-fork). The latest→live change for anonymous MCP is a user-visible behavior change for agent clients on draft-showing sites; it should be called out in the changelog.

## References

- `internal/case/sitesearch/resolve.go:50` — the site engine (text+vector+RRF+rerank+ACL)
- `internal/graph/schema.resolvers.go:3205` — GraphQL wrapper (1 line)
- `internal/graph/schema.resolvers.go:3230` — `notePaths(search:)` reuses it
- `internal/case/rendersearchpage/endpoint.go:35` — `/search` HTML page, third consumer
- `internal/case/mcp/resolve.go:489` — MCP `handleSearch`, parallel pipeline
- `internal/case/mcp/resolve.go:1216-1421` — duplicated vector/RRF/snippet code
- `internal/case/mcp/resolve.go:502,513` — MCP always-latest corpus
- `assets/ui/user/search/search.view.ts:2` — frontend uses GraphQL `search`
- `internal/defaulttemplate/views.html:421` — template mounts the widget
- `docs/dev/federated_search_ui.md` — in-flight design that extends GraphQL search
