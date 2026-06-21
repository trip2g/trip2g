---
title: How vector search and embeddings work
lang: en
free: true
---

Vector search finds documents by meaning rather than by exact keywords. It works by turning text into embeddings — lists of numbers that place similar meanings close together in a high-dimensional space.

## Embeddings

An embedding model reads a piece of text and outputs a fixed-length vector. Two passages about the same idea land near each other even if they share no words, which is what lets a query in one phrasing match a document written in another.

## Similarity

To rank results, the system compares the query vector to every document vector using cosine similarity — the cosine of the angle between them. A score near one means nearly the same direction, and therefore nearly the same meaning.

## Hybrid retrieval

Pure vector search can miss exact terms like names or codes, so production systems combine it with keyword search such as BM25. The two ranked lists are merged, often with reciprocal rank fusion, so a result that both methods like rises to the top.
