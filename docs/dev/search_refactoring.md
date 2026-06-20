# Search refactoring: measured improvements to vector search

This is the running report for a data-driven refactor of trip2g's hybrid search. Every change is applied in isolation and benchmarked before/after with the harness in [`retrieval_eval.md`](./retrieval_eval.md), so each claim is backed by a committed artifact in `docs/superpowers/eval-runs/`.

## Setup

- **Corpus:** `testdata/vecbench/vault/` — 48 notes (6 themes × {1 core + 3 intra-theme distractors} × {EN, RU}). Distractors force the retriever to discriminate between near-neighbours (green tea vs black tea / coffee / herbal; sourdough vs yeast bread / yogurt / sauerkraut; goroutines vs mutexes / context / errgroup; …).
- **Golden set:** `testdata/eval/golden_set.json` — 60 hand-verified queries, balanced across four directions. Relevance = both language versions of the topic, so nDCG/MRR reward ranking *both* the same- and cross-language note at the top.
- **Model:** bge-m3 (1024-dim), self-hosted, OpenAI-compatible. Same model throughout, so deltas isolate pipeline changes, not the embedder.
- **Metric note:** the baseline is already strong (~0.92 nDCG), so absolute gains are small by construction; the interesting signal is the weak directions (especially en→ru) and MRR.

## Results

| # | Change | Recall@10 | nDCG@10 | MRR | en→ru nDCG | Δ nDCG |
|---|--------|-----------|---------|-----|------------|--------|
| 0 | Baseline (bge-m3, current pipeline) | 0.9833 | 0.9157 | 0.9417 | 0.8451 | — |
| F1 | Widen fusion pool (`vectorTopK` 5→50) | **1.0000** | 0.9221 | 0.9417 | 0.8599 | +0.0064 |
| F2 | Per-language bleve analyzer (en + ru) | 1.0000 | 0.9221 | 0.9417 | 0.8599 | +0.0000 |
| F3 | Cross-encoder reranker, 512-char passages | 1.0000 | 0.8881 | 0.8708 | 0.8794 | **−0.0340** |
| F3b | Cross-encoder reranker, full-text passages | 0.4083 | 0.3880 | 0.4500 | 0.2863 | **−0.5341** |
| → | **Shipped: reranker OFF by default** | 1.0000 | 0.9221 | 0.9417 | 0.8599 | (= F2) |
| F4 | Heading breadcrumb + token-aware chunking | 0.9917 | 0.9263 | 0.9500 | 0.8669 | +0.0042 |
| F5 | Dot-product similarity (vectors are unit-norm) | 0.9917 | 0.9263 | 0.9500 | 0.8669 | +0.0000 |
| F5✗ | AND→OR BM25 fallback — **reverted** | 0.6750 | 0.6744 | 0.9556 | 0.6055 | **−0.2519** |
| → | **Shipped: dot-product only (AND→OR off)** | 0.9917 | 0.9263 | 0.9500 | 0.8669 | (= F4) |

(F4's effect on the short corpus is small — its notes are single-chunk. Its real impact shows on the long-doc track below.)

### Long-doc track (multi-chunk notes)

A separate golden set (`testdata/eval/golden_set_longdocs.json`, 16 queries) over 6 long Marcus-Aurelius chapters (RU/EN/mixed) that split into 6–10 chunks each — so chunking-dependent fixes are measurable.

| # | Change | Recall@10 | nDCG@10 | MRR | en→en nDCG | Δ nDCG |
|---|--------|-----------|---------|-----|------------|--------|
| 04 | Baseline (post F1–F3, reranker off) | 1.0000 | 0.9308 | 0.9062 | 0.8155 | — |
| F4 | Heading breadcrumb + token-aware chunking | 1.0000 | **0.9539** | 0.9375 | **0.9077** | **+0.0231** |
| F5 | Dot-product similarity (vectors are unit-norm) | 1.0000 | 0.9539 | 0.9375 | 0.9077 | +0.0000 |
| F5✗ | AND→OR BM25 fallback — **reverted** | 0.8750 | 0.7664 | 0.7377 | 0.9077 | **−0.1875** |

_(rows added as each fix lands)_

## Changes

### F0 — Baseline

The pipeline as found: bleve BM25 + brute-force cosine over per-chunk embeddings, fused with Reciprocal Rank Fusion (k=60), capped at 20. Weakest direction is en→ru (English query → Russian note): the English note ranks well but its Russian counterpart is buried under distractors.

### F1 — Widen the fusion candidate pool

`internal/case/sitesearch/resolve.go` scored cosine similarity for **every** chunk, then truncated to `vectorTopK = 5` unique notes *before* Reciprocal Rank Fusion. A note ranked #6–#50 by the vector lane — but high by BM25 — contributed zero vector signal to fusion, even though its cosine score was already computed. Pure lost recall at no compute saving. The MCP path had the same bug with `DefaultVectorSearchLimit = 10`.

**Change:** raise both to 50 (the final result list is still capped after merge: 20 for sitesearch, `MaxMergedResults`/`DefaultDisplayLimit` for MCP). No re-embedding needed — code only.

**Result:** Recall@10 0.9833 → **1.0000** — the cross-lingual counterparts previously buried below top-10 (e.g. `/ru/zakvaska` for an English sourdough query) now surface. nDCG +0.006, en→ru +0.015. MRR unchanged, as expected: F1 adds candidates lower in the list, so it lifts recall/nDCG without moving the #1 hit.

### F2 — Per-language bleve analyzer

`internal/noteloader/search.go` analyzed **all** content with the Russian analyzer, so English notes and queries were stemmed with Russian rules. A subtle second bug compounded it: the document mapping was registered under a named type (`AddDocumentMapping("note", …)`), but notes are indexed as plain structs with no type field — so bleve silently fell back to the dynamic default mapping and the named mapping never applied.

**Change:** index `Title`/`Body` under both a Russian-analyzed field and an English-analyzed field (`Title_en`/`Body_en`), make it the **default** mapping, and query with a per-field disjunction so the query is analyzed with each field's own analyzer. Proven by unit test: "run race" now matches "running races" (en stemming), and Russian still matches. Confirmed live: the singular query "embedding" now lexically matches the English notes that say "embeddings".

**Result: no measurable change on this benchmark** (Δ 0.0000). Two honest reasons: cross-lingual directions ride the *vector* lane (BM25 can't match across languages), and bge-m3's vector lane already ranks the monolingual cases well and dominates RRF, so the improved English BM25 doesn't change the top-10. F2 is kept anyway — it's a genuine correctness fix with zero regression, and it helps exact-term/lexical English queries (rare words, code identifiers, names) that the vector lane misses. Capturing that gain would require a lexical-query golden set; the current set is deliberately natural-language/semantic.

### F3 — Cross-encoder reranker (negative result; shipped off)

The textbook "biggest quality lever": a second stage that re-scores the fused candidates with a cross-encoder (`bge-reranker-v2-m3`, self-hosted sidecar). We added it behind a feature flag, reranking the fused top-N and keeping `OutputK`.

**It measured strictly worse — twice.** With 512-char passages nDCG dropped 0.9221 → 0.8881 and MRR 0.9417 → 0.8708; with full-note passages it collapsed to 0.39 (passages far exceed the cross-encoder's ~512-token window, so after truncation the notes look alike and ordering goes to noise, pushing relevant notes out of the top-10).

**Why it hurt** (from per-query diff): the cross-encoder over-weights surface query↔passage term overlap and promotes **near-neighbour distractors** that the strong bi-encoder + RRF first stage had correctly ranked below the answer. Examples: "медленное брожение теста" (dough fermentation) promoted *sauerkraut* over sourdough; "каналы вместо общей памяти" (channels vs shared memory) promoted *mutexes* over goroutines; "пул воркеров" promoted *errgroup* over goroutines.

**Lesson:** a reranker is not free. When the first stage is already strong (~0.92) and the corpus is full of topically-adjacent documents, a naive "replace the order with the cross-encoder's" hurts. This is exactly why we measure instead of assuming.

**Decision:** ship it **off by default** (`vector_search.reranker.enabled=false`); keep the client, config, and sidecar so it can be A/B-tested per deployment. A promising future variant — *blend* the rerank score with the RRF rank instead of overriding it (keep RRF as a strong prior) — is left as follow-up.

### F4 — Heading breadcrumb in chunks + token-aware sizing

Two chunking fixes in `internal/mdchunk`:
1. **Heading breadcrumb.** Each chunk's content is now prefixed with its section path — `{title} > {h1} > {h2}\n\n{body}` — instead of just `{title}`. A deep chunk carries its document context into the embedding (a cheap form of contextual retrieval), and the breadcrumb doubles as a *navigable* pointer: it aligns with the search-result TOC, so an agent can drill from a fuzzy vector hit to the exact section via `note_html(toc_path=…)`.
2. **Token-aware sizing.** Sizing switched from characters to estimated tokens (~450 target). The old 2000-char target was ~1000 tokens for Cyrillic — over bge-m3's 512-token window — so the tail of Russian chunks was silently truncated server-side and never embedded.

**Result:** on the long-doc track nDCG@10 0.9308 → **0.9539** (+0.023), with English multi-chunk retrieval jumping en→en 0.8155 → 0.9077 (+0.09). On the short corpus the effect is small (those notes are single-chunk) and roughly neutral (one query slipped from recall 1.0 → 0.99 due to changed chunk boundaries). Requires a full re-embed (chunk content changed).

### F5 — Dot-product similarity (shipped) + AND→OR fallback (negative result, reverted)

Two ideas, one shipped, one rejected by measurement.

**1. Dot-product similarity — shipped, quality-neutral, a small perf win.** The embedding server returns L2-normalized vectors (`embedding-server/server.py`, `normalize_embeddings=True`; confirmed empirically — every stored embedding has `‖v‖ = 1.000000`). For unit vectors cosine similarity *is* the dot product, so `cosineSimilarity` (which recomputed both magnitudes and a division per chunk) was replaced with a plain dot product in `internal/case/sitesearch/resolve.go`. Brute-force scan over every chunk now does one multiply-add loop instead of three. **Result:** nDCG/Recall/MRR identical to F4 to the last digit on both corpora (short 0.9263, long 0.9539) — exactly as expected for an algebraically-equivalent change. Pure latency win, zero quality cost.

**2. AND→OR BM25 fallback — tried, measured strictly worse, reverted.** The idea: the BM25 lane (`internal/noteloader/search.go`) requires every query term to appear (`MatchQueryOperatorAnd`); when AND returns few hits, re-run with OR and merge to "recover recall." It looked free — recall was already 1.0, so what's the harm?

It collapsed the short corpus from **nDCG@10 0.9263 → 0.6744 and Recall@10 0.992 → 0.675** (long-doc 0.9539 → 0.7664). Tellingly, **MRR barely moved** (0.95 → 0.9556): the #1 hit stayed right, but everything below it rotted.

**Why it hurt** (the instructive part): this is a *hybrid* search — BM25 and the vector lane are fused by Reciprocal Rank Fusion, which scores by **rank**, not score. The golden set is natural-language/semantic, so the AND query almost never matches all terms → the fallback fired on nearly every query → it flooded the BM25 lane with up to 20 loose OR matches (documents sharing a single common term). RRF then handed those low-precision matches strong ranks (1/(k+1), 1/(k+2), …), and they outranked the genuinely relevant documents that the vector lane had correctly surfaced. Relevant notes got pushed below the top-10 → recall collapse. The #1 spot survived because the vector lane still dominates the very top, which is why MRR masked the damage.

**The lesson** (a sibling of the F3 reranker result): in a hybrid where the vector lane already carries recall, the BM25 lane's job is **precision**, and keeping it *quiet* on semantic queries (AND-only, returning nothing rather than noise) is load-bearing. A "free recall win" that injects low-precision candidates into a rank-fusion is not free — it's actively corrosive. We only know because we measured before shipping; the change had been committed and would have silently halved retrieval quality otherwise.

**Decision:** ship the dot-product, revert the AND→OR fallback (`search.go` restored byte-for-byte to the pre-F5 baseline). Artifacts: regression `06-f5-*.json`, restored `07-f5-fixed-*.json`.
