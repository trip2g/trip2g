---
title: Rerankers for two-stage retrieval
lang: en
free: true
---
Two-stage retrieval separates the search problem into a cheap retrieval pass followed by an expensive but accurate reranking pass. The first stage uses fast indexes — BM25, HNSW, or both — to retrieve a candidate set of, say, 100 documents. The second stage applies a reranker that scores each candidate in context of the full query, then returns the top-k results.

## Cross-Encoders

A cross-encoder accepts a (query, document) pair as a single concatenated input and produces a relevance score. Because it attends jointly over both texts, it captures fine-grained interactions — negations, conditional statements, entity overlap — that a bi-encoder misses when it encodes query and document independently. The tradeoff is cost: running a cross-encoder over thousands of documents at query time is too slow, so it is always applied only to the shortlist from stage one.

## Why Reranking Improves Precision

Recall-oriented retrieval intentionally returns a broad set to avoid missing relevant documents. Rerankers then sharpen precision by promoting documents where the query intent genuinely matches the document content, not just the surface tokens or embedding neighborhood. This pipeline pattern — retrieve broadly, rerank precisely — has become the default architecture in production RAG systems, enterprise search, and open-domain question answering.
