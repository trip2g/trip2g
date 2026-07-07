# Vector/hybrid search through the token-economy lens — findings (2026-07-02)

**TL;DR:** The biggest token-economy problem is not retrieval quality — it is the *failure paths*. Whenever a pointer misses, `note_html` silently returns the **entire note**: unusable `p:mN` match_ids (which `search` itself emits), stale or fallback `toc_path`s, and 400-rune-truncated "focused windows" all degrade to the exact full-note anti-pattern the Token Economy article warns against — with no signal to the agent that it just paid 10–37× more than promised. On top of that, the user docs still teach a `toc` field that slim search removed.

Read-only analysis. Lens: `docs/en/user/Token Economy.md`, `Fuzzy Pointer.md`, `expand.md`, `token-economy-bench.md`, `docs/dev/token_economy_bench.md`, `token_economy_docs_grep_2026-06-22.md`. Code: `internal/case/sitesearch/resolve.go`, `internal/case/mcp/{resolve,toc_path,types}.go`, `internal/noteloader/{search,loader}.go`, `internal/mdchunk/*`, `internal/case/backjob/generatenoteversionembedding/resolve.go`, `internal/case/similarnotes/resolve.go`, `embedding-server/server.py`.

## How search maps to the token-economy thesis

The thesis (Token Economy.md, token-economy-bench.md): an agent should pay for the *smallest high-signal slice* — `search` returns a fuzzy pointer (breadcrumb → `toc_path`), the agent reads one section via `note_html(toc_path=…)`, structure unfolds on demand via `expand`. Measured 15× median saving vs whole-note reads, 23× vs naive grep.

The implementation genuinely embodies this where it works:

- **Tight defaults.** `DefaultSearchLimit=6`, `DefaultSearchDetailLimit=3` (`internal/case/mcp/resolve.go:34-35`); results beyond `detail_limit` are lightweight previews (title/path/score, no snippets) — a deliberate, good token design (`buildSearchPayload`, resolve.go:594-640).
- **Slim search.** The flat per-result `toc[]` was removed in favour of `expand` — dead weight eliminated exactly as `token_economy_bench.md:192-195` prescribed.
- **The breadcrumb is worth its tokens.** The `{title} > {h1} > {h2}` prefix costs ~10–25 estimated tokens per chunk, buys +0.023 nDCG on long docs (+0.09 en→en; `search_refactoring.md` F4), *and* doubles as the `toc_path` pointer — one string paying twice. Keep it.
- **Chunk sizing matches the encoder.** `chunkTargetTokens=450` with Cyrillic-aware estimation (`internal/mdchunk/chunk.go:10,24-34`) fixed the F4 silent-truncation bug. Papercuts remain (see findings 12).
- **Snippets are small.** 200 runes for vector snippets, breadcrumb stripped (`snippetFromChunk`).

Where it breaks is below.

## Findings

### 1. `note_html` silently dumps the full note when the pointer misses — HIGH

`internal/case/mcp/resolve.go:911-924`: if `match_id` doesn't resolve (`focusedChunkWindow` → `ok=false`) or `toc_path` doesn't match any `data-header` section (`sectionHTMLByTocPath` → `""`), the handler falls through and returns `string(note.HTML)` — the whole note. No error, no marker.

Why it costs: the agent asked for a ~300-token section and receives 3,000–8,000 tokens (the bench's own templates page is 7,668). This is precisely the anti-pattern from `Fuzzy Pointer.md:74-76`, delivered by the tool that promises to prevent it. It also masks pointer bugs (findings 2, 6) — nothing ever *fails*, so nobody notices the drill-down is broken. `Fuzzy Pointer.md:117` even documents the stale-heading case ("the path may not resolve until the next re-index") without mentioning that "not resolve" means "full-note dump".

Fix direction: never fall back silently. Return an explicit error ("section not found; top-level sections are: …" — an `expand`-shaped nudge costs ~30 tokens) or prefix the response with a loud one-line warning so the agent can abort.

### 2. `search` emits match_ids that `note_html` cannot parse — HIGH

`buildSearchPayload` (`resolve.go:615-624`) defaults `matchID` to `p%d:m%d` and only upgrades to `p%d:c%d` when a chunk index resolves. But `parseChunkMatchID` (`resolve.go:1016-1033`) only accepts the `:c` form. Every `m`-form match_id — i.e. every text-lane hit where `nearestChunkIndexForSnippet` fails its conservative gate (resolve.go:695-699) — is a poisoned pointer: feed it back per the tool description ("note_html(pid=..., match_id=...) for a focused chunk window", resolve.go:187) and finding 1 dumps the full note. The text response even prints it prominently: `match_id: p32:m1` (resolve.go:501).

Fix direction: don't emit unusable match_ids (omit the field when no chunk resolved), or teach `parseChunkMatchID` the `m` form with a graceful "no focused window available" error.

### 3. User docs and tool schema teach the removed `toc` field — HIGH (docs)

Slim search dropped the per-result `toc[]` (`token_economy_bench.md:192-195`; `SearchResultItem` in `internal/case/mcp/types.go:152-162` has no toc field). But:

- `Token Economy.md:29`: "Each result includes `toc` (the full heading hierarchy)…"
- `Token Economy.md:104-106` / `Fuzzy Pointer.md:104-106`: "If you need a sibling section, use toc items from the same search result — no second search call."
- `Fuzzy Pointer.md:40-41`: step 1 returns "the note's full table of contents (`toc` field)".
- Live tool schema, `note_html.toc_path` description (`resolve.go:229`): "Use path from search result toc items."

Why it costs: these docs *are* the agent's operating manual (the MCP `initialize` instructions and the published workflow). An agent following them looks for a field that does not exist, burns a turn, and the documented fallback for confusion is the whole-note read. The docs sell the token economy and then mis-route the agent implementing it. (`expand.md` is correct and already describes the slim behaviour — the older two notes were never updated.)

Fix direction: update `Token Economy.md` and `Fuzzy Pointer.md` (EN+RU) to the `expand`-based sibling workflow; fix the `toc_path` property description in `handleToolsList`.

### 4. The "focused chunk window" truncates each chunk to 400 runes — MEDIUM-HIGH

`focusedChunkWindow` (`resolve.go:994-1013`) returns `snippetFromChunk(chunk.Content, 400)` for chunks `index±1`. A chunk targets ~450 tokens — roughly 1,400–1,800+ chars for mixed text — so 400 runes keeps ~25% of the matched chunk, cut at a word boundary with `...`. The window can drop the answer even when retrieval was perfect.

Why it costs: a cheap arm that loses the answer is scored as a win — the exact trap `token_economy_bench.md:100-101` warns about ("log a correctness guard per arm… so a cheap arm that drops the answer isn't scored as a win"). The agent then re-reads via full note, paying both arms. Note the bench (`te_check.py`) measures `toc_path`, not `match_id`, so this path was never benchmarked for answer retention.

Fix direction: return the full matched chunk (it is already sized to ~450 tokens — that *is* the focused window) and trim only the neighbours; wire up the currently dead `context_words` hint (resolve.go:226 calls it "future").

### 5. `search`/`similar`/`expand` responses double-pay: text block duplicates structuredContent — MEDIUM

`handleSearch` builds a full human-readable listing (rank, title, path, URL, first snippet, match_id per result; resolve.go:492-509) *and* attaches the complete structured payload (`structuredToolResult`, resolve.go:96-101). MCP clients receive both. `Token Economy.md:40,71` explicitly praises `graphql_request` for the opposite: "The `content[0].text` block is a short stub… single-copy, no redundancy. That's by design." Search violates the design its own docs advertise; so do `similar` (resolve.go:824-838) and `expand` (resolve.go:968-992). Per result the item also carries three near-identical locator strings (`note_path`, `href`, `url` — types.go:155-157).

Why it costs: roughly 2× on every search response — the entry-point call of *every* agent session. At ~6 results this is a few hundred wasted tokens per query, paid before any savings from the drill-down start.

Fix direction: shrink the text block to a stub ("6 results — see structuredContent"), matching graphql_request; consider dropping `href` when it equals the `url` suffix.

### 6. `toc_path` fallback is wrong-but-confident — MEDIUM

`tocPathForSnippet` (`internal/case/mcp/toc_path.go:69-100`) never expresses uncertainty: when fuzzy HTML matching fails it returns the chunk breadcrumb, and when that fails, `firstSectionPath` — the note's *first section*, regardless of where the match was. The payload has no confidence flag (`SearchMatch`, types.go:164-171), so a guess is indistinguishable from a verified pointer.

Why it costs: the agent reads a precise-looking pointer to the wrong section (~300 tokens), doesn't find the answer, then reads the whole note (finding 1) — paying *more* than the naive path. The bench's own miss report — "the pointer landed on a neighbouring section" (`token-economy-bench.md:83`) — is this class of failure; the article correctly calls landing "the hard half", but the code hides misses instead of surfacing them. Related wart: `chunkContentByIndex` (resolve.go:645-655) treats `chunkIndex==0` as "no chunk", so first-chunk matches (chunk indices are 0-based, `mdchunk/chunk.go:68-71`) never get the breadcrumb fallback.

Fix direction: return no `toc_path` (or `"toc_path_confidence": "guess"`) instead of the first-section fallback; let the agent choose `expand` when the pointer is soft.

### 7. Freshly edited notes vanish from the vector lane until the next full reload — MEDIUM (staleness)

The chunk query joins on the latest version (`queries.read.sql:1273-1278`, `np.version_count = nv.version`, `embedding is not null`). A push bumps the version, triggers a `Load` that reloads chunks *before* the async embedding job (priority 10, `cmd/server/webhooks.go:27-39`) has embedded the new version — so the note contributes zero chunks. When the job later writes embeddings, nothing refreshes `l.chunks` (the only writer is `loadChunks` inside `Load`, `internal/noteloader/loader.go:336`). The note stays invisible to vector search, `toc_path` generation, and `similar` until the *next* push/save — for a rarely-edited vault, indefinitely.

Why it costs quality: the freshest content — the thing the user just wrote and is most likely to ask about — is exactly what the semantic lane can't see; only BM25 covers it, and BM25 alone degrades `toc_path` to note-title level (the bench's own setup note: "With vector OFF, toc_path degrades to the note title", `token_economy_bench.md:110-111`).

Fix direction: have the embedding job (or a completion hook) trigger a chunk-only reload, or make `NoteChunks()` lazily re-query when the job signals completion.

### 8. Every `Load` re-reads and re-decodes all embeddings — MEDIUM (perf; confirms the pushnotes-perf "embeddings re-decode" note)

`loader.go:336 → loadChunks` (loader.go:537-559) does a full-table read of every chunk row *with* its 4 KB embedding blob and `BytesToFloat32Slice`-decodes each into a fresh `[]float32` — on every push, twice (latest + live loaders), even when zero chunks changed. Whole-note embeddings likewise (`assignEmbeddings`, loader.go:488-498). Contrast: the bleve index in the same function is incrementally maintained via content hashes (`search.go:100-134`). The chunks are not.

Why it costs: this is a top allocation source under push load (the sync-perf report's "embeddings re-decode" candidate; ~45% CPU in GC). Fix direction: cache decoded chunks keyed by `(version_id, chunk_index, content_hash)` and re-query only deltas — the hash column already exists.

### 9. Per-query hot path: brute-force scan plus O(results × chunks) post-processing — MEDIUM (perf)

- Vector lane: brute-force dot/cosine over every chunk per query (`sitesearch/resolve.go:160-166`, `mcp/resolve.go:1147-1163`) — known TODO (resolve.go:1298), fine at docs scale, linear in vault size.
- MCP-only extras per query: for each detailed snippet, `nearestChunkIndexForSnippet` (resolve.go:657-700) scans and re-normalizes *all* chunks (allocating via `snippetFromChunk`+`normalizeSearchSnippet` per chunk); `chunkContentByIndex` is another linear scan; `tocPathForSnippet` parses the full note HTML up to three times (`html.Parse` in `tocPathForSnippet`, `pathExistsInHTML`, `firstSectionPath` — toc_path.go:83,451,465). With detail_limit=3 and multi-fragment bleve hits this multiplies.
- `similar`: `maxChunkSimilarity` is O(srcChunks × all target chunks) cosine pairs per call (`similarnotes/resolve.go:153+`), norms recomputed each time.

Fix direction: pre-index chunks by note path (map, built once per Load, not per query); short-circuit `nearestChunkIndexForSnippet` to the matched note's chunks only (it already filters by path — the scan is over the full slice); parse the note HTML once per result.

### 10. sitesearch and MCP are drifting copies of the same pipeline — LOW (correctness/maintenance)

`vectorSearch`, `mergeResults`, `snippetFromChunk`, `trimWhitespace` exist twice. They have already diverged:

- F5's dot-product shipped only in sitesearch (`sitesearch/resolve.go:368-378`); MCP still runs full cosine (`mcp/resolve.go:1160,1300-1317`) — 3× the flops on the MCP hot path for an algebraically identical result.
- sitesearch sorts with a deterministic URL tiebreaker (`sitesearch/resolve.go:278-283`); MCP sorts by score only (`mcp/resolve.go:1256-1258`). RRF produces exact ties routinely (text rank *i* and vector rank *i* score identically), and Go map iteration is random — identical queries can return differently ordered MCP results. For an agent, unstable ordering means re-reads.

Fix direction: extract a shared hybrid-search package (per the service-package pattern); apply dot-product and the tiebreaker in one place.

### 11. Per-query Warn-level diagnostic log — LOW

`sitesearch/resolve.go:160-166`: `env.Logger().Warn("vector scan complete", "chunks", …, "duration", …)` fires on every site search at Warn. Leftover diagnostic; demote to Debug or remove.

### 12. Chunking papercuts — LOW

- The breadcrumb prefix is *not counted* in chunk sizing (`chunk.go:107-125` sums blocks+overlap only; `flush` prepends `prefix()` after the fact) — a deep heading chain pushes a near-target chunk past 450.
- A single oversized block (large code fence, wall-of-text paragraph) is never split — one block can exceed the encoder window on its own.
- The overlap slice cuts bytes, not runes (`chunk.go:79-81`, `last[len(last)-chunkOverlap:]`) — can split a Cyrillic char, embedding and storing invalid UTF-8 at the chunk head.

Context that softens all three: `chunk.go:10` says "512-token window of bge-m3/E5", but bge-m3's window is actually 8192 (`internal/features/vector_search.go:60-68`; the sidecar uses SentenceTransformer defaults, no explicit cap — `embedding-server/server.py`). The 512 limit is real only for `multilingual-e5-base`. So on the default bge-m3 deployment these are retrieval-quality papercuts, not silent truncation; on an e5 deployment the oversized-block case still truncates silently, F4-style.

### 13. Stored chunk token counts are fake — LOW

`embedding-server/server.py` reports `usage.prompt_tokens = len(t.split())` — whitespace words, not tokens. These land in `note_version_chunks.tokens` (`generatenoteversionembedding/resolve.go:170-180`). Harmless today, but any future token-accounting or window-budget logic built on that column will be ~2–4× off (worse for Cyrillic).

### 14. N+1 auth queries per search for signed-in non-admin users — LOW

Both filters call `CanReadNote` per result (sitesearch/resolve.go:103, mcp/resolve.go:522). For a signed-in non-admin that is `ListActiveUserSubgraphs` — a DB query with a standing "TODO: add caching" (`cmd/server/graphql.go:29-31`) — up to 20× per search. Anonymous and admin/API-key paths skip the DB. Fix: fetch subgraphs once per request.

### Non-findings (checked, fine)

- **`app.Now()` in the search path** — not there, and `SiteConfig` is cached behind an atomic pointer (`internal/configregistry/builder.go:43-52`), so it is not a DB-read-per-call anyway. (It does call `time.LoadLocation` per invocation, but that's outside search.)
- **Reranker removal** — correct call, twice-measured (`reranker.md`); the token lens agrees: a sidecar spending compute to *reduce* pointer precision is negative on both axes.
- **RRF top-N sizing** — the fused pool (50) vs returned (6, detail 3) split is right: wide fusion, narrow return. No evidence of noise-padding; the previews beyond detail_limit cost ~20 tokens each.
- **Deleted notes lingering in bleve** — `buildSearchIndex` can't remove deleted notes from the index (`noteloader/search.go:137-145`, admitted in comment), but stale hits are dropped at `l.nvs.Map[hit.ID]` lookup; the only cost is dead hits occupying a few of bleve's 20 slots. Minor.

## Does the search live up to the token-economy article?

**On the happy path — yes, honestly.** The 15× median is real, reproducible (`scripts/token_economy_check.py`), and the article is unusually candid about its own weak spots (naive baseline, the two pointer misses, grep caveats). Slim search, detail_limit previews, and `expand` are genuine token engineering, not marketing.

**On the miss path — no.** The system's contract with the agent is "pay ~300 tokens for a pointer-guided read", but every miss silently converts into the 3,000–8,000-token full-note dump the article condemns: unusable `m`-form match_ids from search itself (finding 2), stale/renamed headings (finding 1), confident wrong pointers (finding 6), a focused window that amputates the chunk it points at (finding 4). And the two flagship docs still describe a `toc` field that no longer exists (finding 3). The article says "landing on exactly the right section is the hard half" — the code agrees, but responds to a missed landing by quietly charging the agent the maximum price. Fixing the fallbacks to *fail loudly and cheaply* (an error + an `expand` nudge is ~30 tokens) would close the gap between the article and the shipped behaviour at near-zero cost.

Ordered by leverage: findings 1+2 (silent full-note fallback + poisoned match_ids) first, 3 (docs) second — all three are cheap; then 7 (staleness), 4 (window truncation), 5 (double payload), 8 (re-decode).
