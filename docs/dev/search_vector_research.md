# Vector Search Research

## Summary of Findings

Research on best practices for semantic search with `text-embedding-3-small` over an Obsidian Markdown vault.

---

## 1. Chunking

**Should you chunk?** Yes, always. Embedding an entire note (2000–10000 tokens) degrades retrieval quality because the embedding averages over too many topics. Even short notes benefit from consistent chunking.

**Recommended chunk size for Markdown:** 400–512 tokens with 50–100 tokens overlap (10–20%). Use a **semantic/recursive splitter** on `["\n\n", "\n", ". ", " "]` — respects heading and paragraph boundaries. Fixed-size splitting is worse.

**Parent Document Retriever pattern (dominant in production RAG):**
- Embed and index only **small child chunks** (200–512 tokens) for vector search
- Store full note keyed by `doc_id` metadata on every child chunk
- On query: find top-K matching child chunks → fetch full parent note per `doc_id`
- Gives precise retrieval (small chunks match better) + full context for display
- Natural parent boundary for Obsidian: the full note file

---

## 2. Short Query vs Long Document Problem

Single-word query ("space") vs long document gives low cosine similarity (~0.2–0.3) even for relevant pairs. This is expected — the query is a sparse point, the document is a centroid over many concepts.

**Fixes by implementation cost:**

| Technique | How | Cost | Gain |
|---|---|---|---|
| **Prepend title to chunk** | Add note title + H1 to each chunk before embedding | Zero | Low-medium |
| **Query expansion (LLM)** | Rewrite short query into 2–3 sentences before embedding | +1 LLM call | High |
| **HyDE** | Generate hypothetical answer doc, embed that | +1 LLM call | High |
| **Multi-query** | 3–5 query variants, union results | +1 LLM call | Medium |
| **BM25 hybrid (already done)** | FTS + vector + RRF | Already implemented | High for keyword queries |

**Easiest quick win:** prepend note title and H1 heading to every chunk text at index time. Zero cost.

**HyDE for short queries:** when query length < 5 words, use GPT-4o-mini to generate a hypothetical passage that would answer the question, then embed that passage. Document-to-document similarity is much higher than query-to-document.

---

## 3. HyDE (Hypothetical Document Embeddings)

1. Take the user query
2. Prompt LLM: "Write a short passage that would answer: [query]"
3. Embed the generated hypothetical document
4. Search corpus with that embedding
5. Discard hypothetical document — never shown to user

**Why it helps:** the hypothetical doc lives in the same dense region as real documents. Particularly strong for multilingual retrieval. Outperforms BM25 and unsupervised dense retrieval on benchmark tasks.

**When it hurts:** LLM hallucination in domain-specific/private vaults can drift the embedding away from the real corpus.

**Verdict:** implement as optional path for short queries (< 5 words).

---

## 4. Similarity Threshold Calibration

**text-embedding-3-small returns lower scores than ada-002** (~5-10% lower). Community rule of thumb: start at **0.25–0.30**, not 0.70+.

**Fixed threshold is a bad idea as sole gate.** Two failure modes:
- Too high → zero results for valid queries
- Too low → noise floods results

**Practical approach:**
1. Start with **top-K (K=5–10) without threshold** during development
2. Sample 50–100 relevant + 50–100 irrelevant query/note pairs from your vault
3. Plot score distribution, set threshold at the natural gap
4. **Hybrid gate in production:** `score >= threshold OR rank <= 3` — always return at least 1 result

**Expected similarity values for text-embedding-3-small (Russian):**

| Pair type | Expected cosine similarity |
|---|---|
| Highly relevant (same topic, similar phrasing) | 0.55–0.75 |
| Relevant (same topic, different phrasing) | 0.35–0.55 |
| Tangentially related | 0.20–0.35 |
| Unrelated | 0.10–0.25 |
| Short 1–3 word query vs full document | 0.15–0.30 even for relevant pairs |

Russian scores ~5–10% lower than English equivalents (training data is English-dominant).

---

## 5. Asymmetric Search — Better Models for Russian

`text-embedding-3-small` is symmetric (single encoder for query and document). Dedicated asymmetric models are better for retrieval.

**Best open-source alternatives for Russian (ruMTEB benchmark):**

| Model | Notes |
|---|---|
| **multilingual-e5-large-instruct** | Best on ruMTEB; use `query:` prefix for queries, `passage:` for docs |
| **BGE-M3** | Best on Russian retrieval specifically; supports dense + sparse; 100+ languages |
| **ru-en-RoSBERTa** | Bilingual RU+EN; competitive at lower cost |

Instruction-tuned models (`Instruct: Retrieve passages that answer the question`) directly address the asymmetric query/document problem without needing HyDE.

**Verdict:** `text-embedding-3-small` is pragmatic if already using OpenAI API. Upgrade to BGE-M3 or mE5large-instruct only if measured recall is unsatisfactory.

---

## 6. Implementation Roadmap (Ordered by ROI)

1. **Top-K without threshold** — remove fixed threshold during development, inspect real distributions
2. **Prepend note title to every chunk** — zero cost, improves short query recall
3. **Chunking (400–512 tokens, Markdown boundaries)** + Parent Document Retriever pattern
4. **HyDE for short queries** (< 5 words) — one GPT-4o-mini call
5. **Threshold from data** — calibrate from vault-specific query/note pairs
6. **BGE-M3 or mE5large-instruct** if Russian quality insufficient

---

---

## 7. Reference Implementations

### tobi/qmd (TypeScript, local-only)

Offline semantic search over Markdown via MCP. Dual-model hybrid:

- **SQLite FTS5** (BM25) + **sqlite-vec** cosine similarity
- Local models: EmbeddingGemma-300M or Qwen3-Embedding-0.6B (GGUF via node-llama-cpp)
- Query prompt template: `"task: search result | query: {text}"`

**Chunking (sophisticated, markdown-aware):**
- Target ~900 tokens, 15% overlap
- Boundary scoring: headings (50–100 pts), code fences (80 pts), paragraph breaks (20 pts)
- Squared-distance decay within ±200-token window
- Code fences never split mid-block

**Full pipeline:**
```
Query → LLM query expansion (3 variants)
      → BM25 + vector search per variant
      → RRF (k=60)
      → Top 30 candidates
      → Cross-encoder reranker (Qwen3-Reranker-0.6B)
      → Position-aware blend
      → Final results
```

Applicable to trip2g: sqlite-vec, markdown-aware chunking, RRF fusion. Query expansion and cross-encoder only if latency allows.

---

### tomohiro-owada/devrag (Go + CGO)

Token-efficient RAG for Claude Code. Pure vector search, no BM25. Written in Go — directly portable.

- **multilingual-e5-small** (384-dim) via ONNX Runtime in-process
- **sqlite-vec** for storage and cosine search
- Clean `Embedder` interface (Embed / EmbedBatch / Close) — swappable: mock uses deterministic SHA256-based vectors for unit tests
- Simple fixed-size 500-char chunking (no markdown awareness — upgrade this)
- Incremental indexing with glob config

**Key Go libraries:**

| Library | Purpose |
|---------|---------|
| `mattn/go-sqlite3` / `modernc.org/sqlite` | SQLite driver (already in use) |
| `asg017/sqlite-vec` | Vector similarity extension |
| `yalue/onnxruntime_go` | In-process ONNX inference (CGO) |
| FTS5 | Built into SQLite |

> **Note (trip2g):** trip2g uses `modernc.org/sqlite` (pure Go, no CGO). sqlite-vec requires CGO and loadable extensions — incompatible with the current driver. Switching to `mattn/go-sqlite3` breaks pure-Go builds and complicates cross-compilation. Not worth it at current scale (~300 notes).

---

## 8. Recommended Architecture for trip2g

Combining both repos + research findings:

### Storage
Use **sqlite-vec** — same SQLite file, no new infrastructure. FTS5 already in SQLite for BM25.

### Chunking
Split at **heading boundaries**, protect code fences, target 600–900 tokens, 10–15% overlap. Prepend note title + H1 to each chunk before embedding.

### Retrieval
```
query
  → SQLite FTS5 BM25                ← already implemented (Bleve → migrate to FTS5)
  → sqlite-vec cosine similarity    ← new
  → RRF(k=60) fusion                ← already implemented
  → top-N results (no fixed threshold, or very low ~0.25)
```

### Embedding model options
- **Hosted**: OpenAI `text-embedding-3-small` (1536-dim) — current approach, simpler
- **Local/free** *(future)*: `multilingual-e5-small` ONNX (384-dim, 50MB) via `onnxruntime-go` — no API cost, better Russian quality. Requires CGO (`onnxruntime-go`).

Use the swappable `Embedder` interface so both are interchangeable.

### Short query fix
For queries < 5 words: **HyDE** (one GPT-4o-mini call to generate hypothetical passage, embed that instead of the raw query).

---

## Sources

- [OpenAI Embeddings Guide](https://platform.openai.com/docs/guides/embeddings)
- [Modified RAG: Parent Document Retriever — LanceDB](https://lancedb.com/blog/modified-rag-parent-document-bigger-chunk-retriever-62b3d1e79bc6/)
- [Best Chunking Strategies for RAG 2025 — Firecrawl](https://www.firecrawl.dev/blog/best-chunking-strategies-rag)
- [HyDE — Zilliz](https://zilliz.com/learn/improve-rag-and-information-retrieval-with-hyde-hypothetical-document-embeddings)
- [Reduced cosine scores thread — OpenAI Community](https://community.openai.com/t/reduced-cosine-of-similarity-relevance-scores-with-text-embedding-3-small-vs-text-embedding-ada-002/873048)
- [Rule-of-thumb thresholds — OpenAI Community](https://community.openai.com/t/rule-of-thumb-cosine-similarity-thresholds/693670)
- [ruMTEB benchmark — arXiv 2408.12503](https://arxiv.org/html/2408.12503v1)
- [Advanced RAG Techniques — Neo4j](https://neo4j.com/blog/genai/advanced-rag-techniques/)
