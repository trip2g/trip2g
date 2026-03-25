# Why Not Qdrant (Vector Database)

Evaluated March 2026. Test: 77 markdown files from docs/dev/, 762 chunks, multilingual-e5-base (768d), 8 search queries.

## Test Results

| Metric | Qdrant HNSW | Brute-Force (current) |
|--------|------------|----------------------|
| Search latency | 2-3ms | 30-35ms |
| Top-1 accuracy vs brute-force | 100% (identical) | baseline |
| Top-5 overlap | 8/8 queries: 5/5 | baseline |
| INT8 quantization quality loss | none (5/5 overlap) | n/a |
| Garbage query score ("xyzzynonexistent") | 0.827 | 0.827 |
| Best real query score | 0.890 | 0.890 |

Results are **byte-for-byte identical** at this scale. HNSW approximation doesn't kick in meaningfully below ~10K vectors.

## Why We Stay With In-Memory Brute-Force

1. **Scale doesn't justify it.** A single site has hundreds to low thousands of notes. At 5K chunks, brute-force takes ~50-100ms — imperceptible in search UX.

2. **Zero quality difference.** The test showed 100% identical rankings across all query types (technical, bilingual, vague). Qdrant solves a speed problem we don't have.

3. **The real problem is the model, not storage.** E5 cosine scores are compressed into 0.75-1.0 range. Garbage queries score 0.82, good matches score 0.87. Qdrant doesn't fix this — it's a model/metric issue. Improvements that actually help:
   - Cross-encoder reranking (second pass)
   - Better score normalization before RRF fusion
   - Hybrid search (BM25 naturally rejects garbage — already implemented)

4. **Architectural simplicity.** Go + SQLite monolith, single binary. Adding Qdrant means: Docker container, data sync pipeline (SQLite → Qdrant), health monitoring, version upgrades, backup coordination. Each moving part is a failure point.

5. **Vectors are already in memory.** Embeddings are loaded at startup for similar-notes features. Brute-force search reuses the same data — no extra memory or sync cost.

6. **No filtering needs (yet).** Qdrant's payload filtering (search within a tag/category) is powerful, but we filter by access permissions in Go after scoring. This works fine at current scale.

## When To Reconsider

| Trigger | Why |
|---------|-----|
| >10K chunks in a single searchable index | HNSW latency advantage becomes real (sub-ms vs 500ms+) |
| Multi-tenant shared index | Combined vectors across sites could reach 50K+ |
| Need for filtered vector search | Qdrant's native payload filters are more efficient than post-filtering in Go at scale |
| Memory pressure | Offloading vectors to Qdrant frees Go process RAM (~6MB per 1K notes, becomes significant at 50K+) |
| Need for vector-level deduplication or clustering | Qdrant has built-in grouping and recommendation APIs |

## What To Improve Instead

1. **Score normalization.** Normalize E5 cosine scores to a meaningful 0-1 range before RRF fusion. The raw 0.75-1.0 compression makes threshold-based filtering unreliable.
2. **Post-RRF minimum score.** Apply a relevance threshold after merging BM25 + vector results, not before. BM25 returning zero results for garbage queries naturally pushes junk down.
3. **Model evaluation.** Test `multilingual-e5-large` (1024d) or `bge-m3` for better score separation between relevant and irrelevant results.
4. **Reranker.** Cross-encoder reranking of top-20 candidates is expensive but dramatically improves precision. Could run as a second pass in the embedding-server.

## Test Setup

```
Qdrant: qdrant/qdrant:latest (Docker)
Embedding: intfloat/multilingual-e5-base via embedding-server (sentence-transformers)
Chunking: paragraph-level, ~1500 char target, 200 char overlap
Collection: HNSW m=16, ef_construct=100, cosine distance
Script: scripts/qdrant_test.py
```
