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

_(rows added as each fix lands)_

## Changes

### F0 — Baseline

The pipeline as found: bleve BM25 + brute-force cosine over per-chunk embeddings, fused with Reciprocal Rank Fusion (k=60), capped at 20. Weakest direction is en→ru (English query → Russian note): the English note ranks well but its Russian counterpart is buried under distractors.

### F1 — Widen the fusion candidate pool

`internal/case/sitesearch/resolve.go` scored cosine similarity for **every** chunk, then truncated to `vectorTopK = 5` unique notes *before* Reciprocal Rank Fusion. A note ranked #6–#50 by the vector lane — but high by BM25 — contributed zero vector signal to fusion, even though its cosine score was already computed. Pure lost recall at no compute saving. The MCP path had the same bug with `DefaultVectorSearchLimit = 10`.

**Change:** raise both to 50 (the final result list is still capped after merge: 20 for sitesearch, `MaxMergedResults`/`DefaultDisplayLimit` for MCP). No re-embedding needed — code only.

**Result:** Recall@10 0.9833 → **1.0000** — the cross-lingual counterparts previously buried below top-10 (e.g. `/ru/zakvaska` for an English sourdough query) now surface. nDCG +0.006, en→ru +0.015. MRR unchanged, as expected: F1 adds candidates lower in the list, so it lifts recall/nDCG without moving the #1 hit.

_(F2…F5 documented below as they land)_
