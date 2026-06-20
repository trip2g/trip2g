---
title: Approximate nearest neighbour indexes
lang: en
free: true
---
Finding the exact nearest neighbour in a high-dimensional vector space requires comparing a query vector against every stored vector — an O(n) operation that becomes prohibitively slow at millions of documents. Approximate Nearest Neighbour (ANN) indexes trade a small amount of recall for dramatic speed gains, making sub-millisecond search over billions of vectors practical.

## HNSW

Hierarchical Navigable Small World (HNSW) builds a layered graph where each node connects to a small set of neighbours. At query time, search starts at the top layer (few, long-range connections) and greedily descends to lower layers with denser, shorter connections, converging on the approximate nearest neighbours. HNSW offers excellent recall-speed tradeoffs and supports incremental inserts, making it the default choice in libraries such as FAISS and Qdrant.

## IVF and the Recall-Speed Tradeoff

Inverted File Index (IVF) partitions the vector space into clusters using k-means. A query is compared only against vectors in the nearest clusters, skipping the rest. The nprobe parameter controls how many clusters are searched: higher values raise recall but increase latency. Unlike HNSW, IVF requires a training phase on representative data before indexing, but it scales to very large collections with lower memory overhead. Both approaches expose tunable parameters so engineers can dial in the recall-latency operating point their application requires.
