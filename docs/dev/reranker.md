# Cross-encoder reranker (removed; negative result)

**TL;DR:** we built a second-stage cross-encoder reranker for site search, measured it **strictly worse** than the existing hybrid ranking (twice), shipped it off by default, and have now removed the code. This note keeps the finding; the implementation lives in git history (`internal/reranker`, `reranker-server/`, `VectorSearchConfig.Reranker`, removed on branch `chore/release-cleanup`).

## What it was

Search is a hybrid: a bi-encoder vector lane (ANN over `bge-m3` embeddings) plus a BM25 lane, fused with Reciprocal Rank Fusion (RRF). The reranker added a *second stage*: re-score the fused top-N candidates with a cross-encoder (`BAAI/bge-reranker-v2-m3`) served from a small Python sidecar (`reranker-server/`, FastAPI + `sentence_transformers.CrossEncoder`), then keep `OutputK`. It was gated behind `vector_search.reranker.enabled` (default false).

A cross-encoder scores `(query, doc)` pairs jointly, unlike the bi-encoder which embeds each text independently. That is also why it can't be served from the plain embeddings endpoint: different architecture, different API (pair scoring vs `/embeddings` vectors). It needs a rerank-capable server (TEI / Infinity / the bespoke sidecar).

## Does it improve quality? No — measured worse, twice

From the search benchmark (`docs/dev/search_refactoring.md`, variant F3):

| variant | nDCG | MRR | Δ nDCG |
|---|---|---|---|
| no reranker (shipped) | 0.9221 | 0.9417 | baseline |
| F3: reranker, 512-char passages | 0.8881 | 0.8708 | **−0.034** |
| F3b: reranker, full-text passages | ~0.39 | — | **−0.53** |

**Why it hurt** (from the per-query diff): the cross-encoder over-weights surface query↔passage term overlap and promotes **near-neighbour distractors** that the strong bi-encoder + RRF first stage had correctly ranked *below* the answer:

- "медленное брожение теста" (slow dough fermentation) promoted *sauerkraut* over sourdough,
- "каналы вместо общей памяти" (channels vs shared memory) promoted *mutexes* over goroutines,
- "пул воркеров" (worker pool) promoted *errgroup* over goroutines.

On full-note passages it collapsed to ~0.39: passages far exceed the cross-encoder's ~512-token window, so after truncation the notes look alike and ordering degrades to noise, pushing relevant notes out of the top-10.

**Lesson:** a reranker is not free. When the first stage is already strong (~0.92) and the corpus is full of topically-adjacent documents, naively replacing the order with the cross-encoder's hurts. Measure before shipping.

## The idea worth keeping

The failure mode was *overriding* a good prior. A promising untried variant: **blend** the rerank score with the RRF rank instead of replacing it, keeping RRF as a strong prior (e.g. a weighted sum of normalized rerank score and RRF score). That is a fresh feature, not a revival of the removed override code — it would be built anew. Left as a follow-up.
