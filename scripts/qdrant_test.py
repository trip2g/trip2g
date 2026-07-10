#!/usr/bin/env python3
"""
Test script: index docs/dev/*.md into Qdrant via the local embedding service,
then run search queries and compare results.

Usage: python3 scripts/qdrant_test.py
"""

import os
import re
import sys
import time
import math
import requests
from pathlib import Path

from qdrant_client import QdrantClient
from qdrant_client.models import (
    Distance,
    VectorParams,
    PointStruct,
    SearchParams,
    HnswConfigDiff,
    ScalarQuantization,
    ScalarQuantizationConfig,
    ScalarType,
)

EMBEDDING_URL = os.environ.get("EMBEDDING_URL", "http://localhost:11434/v1/embeddings")
QDRANT_URL = "http://localhost:6333"
COLLECTION = "docs_dev"
DOCS_DIR = Path(__file__).parent.parent / "docs" / "dev"

# Detect model and set parameters
def detect_model() -> tuple[int, str, str]:
    """Returns (dim, passage_prefix, query_prefix) based on running model."""
    resp = requests.post(EMBEDDING_URL, json={"input": "test"})
    resp.raise_for_status()
    dim = len(resp.json()["data"][0]["embedding"])
    model_name = resp.json().get("model", "")
    if "bge-m3" in model_name:
        return dim, "", ""  # bge-m3 uses no prefixes
    elif "e5" in model_name:
        return dim, "passage: ", "query: "
    return dim, "", ""

# --- Chunking (simplified version of internal/mdchunk) ---

def chunk_markdown(text: str, target_size: int = 1500, min_size: int = 200, overlap: int = 150) -> list[str]:
    """Split markdown into paragraph-level chunks."""
    paragraphs = re.split(r'\n{2,}', text.strip())
    chunks = []
    current = ""

    for para in paragraphs:
        para = para.strip()
        if not para:
            continue
        if len(current) + len(para) + 2 > target_size and len(current) >= min_size:
            chunks.append(current.strip())
            # overlap: keep tail of previous chunk
            words = current.split()
            tail = " ".join(words[-overlap // 5:]) if len(words) > overlap // 5 else ""
            current = tail + "\n\n" + para if tail else para
        else:
            current = current + "\n\n" + para if current else para

    if current.strip():
        chunks.append(current.strip())

    return chunks if chunks else [text.strip()]


# --- Embedding ---

def embed_texts(texts: list[str], prefix: str = "") -> list[list[float]]:
    """Call the embedding service. Prefix 'passage: ' for docs, 'query: ' for queries (e5 convention)."""
    prefixed = [prefix + t for t in texts]
    # batch in groups of 32
    all_vecs = []
    for i in range(0, len(prefixed), 32):
        batch = prefixed[i:i+32]
        resp = requests.post(EMBEDDING_URL, json={"input": batch})
        resp.raise_for_status()
        data = resp.json()["data"]
        all_vecs.extend([d["embedding"] for d in data])
    return all_vecs


def cosine_similarity(a: list[float], b: list[float]) -> float:
    """Manual cosine similarity (to compare with Qdrant's)."""
    dot = sum(x * y for x, y in zip(a, b))
    norm_a = math.sqrt(sum(x * x for x in a))
    norm_b = math.sqrt(sum(x * x for x in b))
    if norm_a == 0 or norm_b == 0:
        return 0.0
    return dot / (norm_a * norm_b)


# --- Main ---

def load_docs() -> list[dict]:
    """Load all markdown files, chunk them, return list of {file, chunk_idx, text}."""
    docs = []
    for md_file in sorted(DOCS_DIR.glob("*.md")):
        text = md_file.read_text(encoding="utf-8")
        title = md_file.stem
        chunks = chunk_markdown(text)
        for i, chunk in enumerate(chunks):
            docs.append({
                "file": md_file.name,
                "title": title,
                "chunk_idx": i,
                "text": f"{title}\n\n{chunk}",
            })
    return docs


def create_collection(client: QdrantClient, name: str, quantize: bool = False):
    """Create Qdrant collection with HNSW index."""
    collections = [c.name for c in client.get_collections().collections]
    if name in collections:
        client.delete_collection(name)

    quantization = None
    if quantize:
        quantization = ScalarQuantization(
            scalar=ScalarQuantizationConfig(
                type=ScalarType.INT8,
                quantile=0.99,
                always_ram=True,
            )
        )

    client.create_collection(
        collection_name=name,
        vectors_config=VectorParams(
            size=VECTOR_DIM,
            distance=Distance.COSINE,
        ),
        hnsw_config=HnswConfigDiff(
            m=16,
            ef_construct=100,
        ),
        quantization_config=quantization,
    )


def index_docs(client: QdrantClient, docs: list[dict], embeddings: list[list[float]]):
    """Upload docs with embeddings to Qdrant."""
    points = []
    for i, (doc, vec) in enumerate(zip(docs, embeddings)):
        points.append(PointStruct(
            id=i,
            vector=vec,
            payload={
                "file": doc["file"],
                "title": doc["title"],
                "chunk_idx": doc["chunk_idx"],
                "text": doc["text"][:500],  # truncate payload for display
            },
        ))

    # batch upload
    batch_size = 100
    for i in range(0, len(points), batch_size):
        client.upsert(
            collection_name=COLLECTION,
            points=points[i:i+batch_size],
        )


def search_qdrant(client: QdrantClient, query_vec: list[float], limit: int = 5,
                  ef: int = 128) -> list[dict]:
    """Search Qdrant with HNSW."""
    resp = client.query_points(
        collection_name=COLLECTION,
        query=query_vec,
        limit=limit,
        search_params=SearchParams(hnsw_ef=ef),
    )
    return [{"score": r.score, "file": r.payload["file"],
             "chunk_idx": r.payload["chunk_idx"],
             "text": r.payload["text"][:200]} for r in resp.points]


def search_brute_force(docs: list[dict], embeddings: list[list[float]],
                       query_vec: list[float], limit: int = 5) -> list[dict]:
    """Brute-force cosine similarity search (like current trip2g implementation)."""
    scored = []
    for i, (doc, vec) in enumerate(zip(docs, embeddings)):
        score = cosine_similarity(query_vec, vec)
        scored.append({"score": score, "file": doc["file"],
                       "chunk_idx": doc["chunk_idx"],
                       "text": doc["text"][:200]})
    scored.sort(key=lambda x: x["score"], reverse=True)
    return scored[:limit]


def print_results(label: str, results: list[dict]):
    print(f"\n{'='*60}")
    print(f"  {label}")
    print(f"{'='*60}")
    for i, r in enumerate(results, 1):
        print(f"  {i}. [{r['score']:.4f}] {r['file']}#{r['chunk_idx']}")
        # show first 120 chars of text
        snippet = r["text"].replace("\n", " ")[:120]
        print(f"     {snippet}")
    print()


def main():
    test_queries = [
        "how does vector search work",
        "GraphQL subscriptions SSE",
        "Telegram bot publishing",
        "obsidian sync protocol",
        "SQLite WAL mode configuration",
        "how to add a new GraphQL mutation",
        "frontend mol components",
        "background job queue",
    ]

    client = QdrantClient(url=QDRANT_URL)

    # 1. Load docs
    print("Loading docs/dev/*.md ...")
    docs = load_docs()
    print(f"  {len(docs)} chunks from {len(set(d['file'] for d in docs))} files")

    # 2. Embed all chunks
    print("\nEmbedding chunks via multilingual-e5-base ...")
    t0 = time.time()
    embeddings = embed_texts([d["text"] for d in docs], prefix="passage: ")
    embed_time = time.time() - t0
    print(f"  Embedded {len(docs)} chunks in {embed_time:.1f}s ({len(docs)/embed_time:.0f} chunks/s)")

    # 3. Create collection and index
    print("\nCreating Qdrant collection (HNSW, cosine, no quantization) ...")
    create_collection(client, COLLECTION, quantize=False)
    t0 = time.time()
    index_docs(client, docs, embeddings)
    index_time = time.time() - t0
    print(f"  Indexed {len(docs)} points in {index_time:.2f}s")

    # 4. Collection info
    info = client.get_collection(COLLECTION)
    print(f"\n  Collection: {COLLECTION}")
    print(f"  Points: {info.points_count}")
    print(f"  Segments: {info.segments_count}")
    print(f"  Status: {info.status}")

    # 5. Run searches
    print("\n" + "="*60)
    print("  SEARCH COMPARISON: Qdrant HNSW vs Brute-Force Cosine")
    print("="*60)

    for query in test_queries:
        print(f"\n>>> Query: \"{query}\"")

        # Embed query
        query_vec = embed_texts([query], prefix="query: ")[0]

        # Search both ways
        t0 = time.time()
        qdrant_results = search_qdrant(client, query_vec, limit=5, ef=128)
        qdrant_time = (time.time() - t0) * 1000

        t0 = time.time()
        brute_results = search_brute_force(docs, embeddings, query_vec, limit=5)
        brute_time = (time.time() - t0) * 1000

        print(f"  Qdrant HNSW: {qdrant_time:.1f}ms | Brute-force: {brute_time:.1f}ms")

        # Compare rankings
        qdrant_files = [(r["file"], r["chunk_idx"]) for r in qdrant_results]
        brute_files = [(r["file"], r["chunk_idx"]) for r in brute_results]
        same_top1 = qdrant_files[0] == brute_files[0] if qdrant_files and brute_files else False
        overlap = len(set(qdrant_files) & set(brute_files))
        print(f"  Same top-1: {same_top1} | Top-5 overlap: {overlap}/5")

        print_results("Qdrant HNSW", qdrant_results)
        print_results("Brute-Force", brute_results)

    # 6. Test with quantization
    print("\n" + "="*60)
    print("  QUANTIZATION TEST (INT8)")
    print("="*60)

    quant_collection = COLLECTION + "_quant"
    create_collection(client, quant_collection, quantize=True)
    index_docs(client, docs, embeddings)

    # Re-upload to quantized collection
    points = []
    for i, (doc, vec) in enumerate(zip(docs, embeddings)):
        points.append(PointStruct(
            id=i, vector=vec,
            payload={"file": doc["file"], "title": doc["title"],
                     "chunk_idx": doc["chunk_idx"], "text": doc["text"][:500]},
        ))
    client.upsert(collection_name=quant_collection, points=points)

    query = "how does vector search work"
    query_vec = embed_texts([query], prefix="query: ")[0]

    results_normal = search_qdrant(client, query_vec, limit=5)
    resp_quant = client.query_points(
        collection_name=quant_collection,
        query=query_vec,
        limit=5,
        search_params=SearchParams(hnsw_ef=128),
    )
    results_quant = [{"score": r.score, "file": r.payload["file"],
                      "chunk_idx": r.payload["chunk_idx"],
                      "text": r.payload["text"][:200]} for r in resp_quant.points]

    print_results(f"Normal (query: '{query}')", results_normal)
    print_results(f"INT8 Quantized (query: '{query}')", results_quant)

    normal_files = [(r["file"], r["chunk_idx"]) for r in results_normal]
    quant_files = [(r["file"], r["chunk_idx"]) for r in results_quant]
    overlap = len(set(normal_files) & set(quant_files))
    print(f"  Normal vs Quantized top-5 overlap: {overlap}/5")

    # Cleanup info
    print("\n" + "="*60)
    print("  SUMMARY")
    print("="*60)
    print(f"  Files: {len(set(d['file'] for d in docs))}")
    print(f"  Chunks: {len(docs)}")
    print(f"  Embedding time: {embed_time:.1f}s")
    print(f"  Model: multilingual-e5-base (768d)")
    print(f"  Qdrant collection: {COLLECTION} ({info.points_count} points)")
    print()
    print("  To clean up:")
    print("    docker stop qdrant-test && docker rm qdrant-test")
    print("    rm scripts/qdrant_test.py")


if __name__ == "__main__":
    main()
