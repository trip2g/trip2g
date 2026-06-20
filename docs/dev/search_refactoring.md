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

_(F4…F5 documented below as they land)_
