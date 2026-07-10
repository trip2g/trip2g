# Vector Search

trip2g ships a hybrid retrieval pipeline: BM25 text search (bleve) fused with dense vector search via Reciprocal Rank Fusion (RRF). The two lanes complement each other — BM25 handles exact-term and lexical matches; the vector lane handles synonyms, paraphrase, and cross-lingual queries.

For benchmark methodology and per-change measurement results, see [search_refactoring.md](./search_refactoring.md) and [retrieval_eval.md](./retrieval_eval.md).

## How retrieval works

A search request goes through three stages:

1. **BM25** — bleve full-text index returns up to 20 ranked results.
2. **Vector** — the query is embedded by the embedding server; brute-force dot-product over all in-memory chunk embeddings returns up to 50 unique-note candidates (the `vectorTopK = 50` constant in `internal/case/sitesearch/resolve.go`).
3. **RRF fusion** — the two ranked lists are merged by Reciprocal Rank Fusion with `k = 60`, producing a single list capped at 20.

An optional cross-encoder reranker exists as a fourth stage but ships **disabled by default** (see [Cross-encoder reranker](#cross-encoder-reranker-off-by-default) below).

## Hybrid pipeline in detail

### BM25 lane

`internal/noteloader/search.go` indexes every note in a bleve in-memory index. Each document has four analyzed fields so both Russian and English queries stem correctly:

| Field | Analyzer |
|-------|----------|
| `Title` | Russian |
| `Title_en` | English |
| `Body` | Russian |
| `Body_en` | English |

The query is a disjunction of per-field match queries, each using `MatchQueryOperatorAnd` (all terms must appear within that field's language). A document matches via whichever field suits its language. The index uses the Russian-analyzed fields as the **default mapping** (so bleve applies them even for untyped structs); the English fields are additional named mappings over the same source field.

The AND operator is intentional and load-bearing: in a hybrid system the BM25 lane's job is **precision**, not recall. The vector lane covers recall. Switching to OR floods the BM25 lane with low-precision matches that RRF then promotes above genuinely relevant results — a regression measured at nDCG@10 0.9263 → 0.6744 (see [search_refactoring.md F5](./search_refactoring.md)).

### Vector lane

**Embedding model.** The embedding server (`embedding-server/server.py`) defaults to `intfloat/multilingual-e5-base` (the `MODEL_NAME` env var in both `server.py` and `Dockerfile`). The trip2g deployment and benchmark override this to `BAAI/bge-m3` (1024-dim, multilingual) via the `MODEL_NAME` env var — so both names are correct in their context. The server returns L2-normalized unit vectors (`normalize_embeddings=True`).

**Per-chunk embeddings.** Notes are split into paragraph-level chunks by `internal/mdchunk`. Each chunk gets its own embedding. This lets a dense query match the most relevant *section* of a long note rather than the averaged whole-note signal.

**Similarity.** Because the embedding server returns unit-norm vectors, cosine similarity equals the dot product. `internal/case/sitesearch/resolve.go` uses `dotSimilarity` — a plain dot product loop — instead of computing the redundant square roots that cosine would require.

```go
// dotSimilarity — equivalent to cosine when vectors are unit-norm.
func dotSimilarity(a, b []float32) float64 {
    var dot float64
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
    }
    return dot
}
```

**Candidate pool.** All chunks are scored in a brute-force scan. The top-50 results (deduped by note path) are passed to RRF fusion. Truncating before fusion at a lower number discards recall at zero compute saving — the cosine scan already ran. The final result list is capped to 20 after fusion.

### RRF fusion

Reciprocal Rank Fusion merges the two ranked lists without requiring score normalization. Each document's contribution from a list is `1 / (k + rank)` where `k = 60`. A document that appears in both lists accumulates contributions from both; a document that appears in only one still enters the fused list. The merged list is sorted by accumulated RRF score, then capped at 20.

## Per-chunk embeddings and the F4 breadcrumb format

### Chunk format

`internal/mdchunk/chunk.go` (`Split` function) produces chunks in this format:

```
{title} > {h1} > {h2}\n\n{body}
```

The heading breadcrumb (`{title} > {h1} > {h2}`) is prepended to every chunk. Benefits:

- A deep chunk carries its document context into the embedding (cheap contextual retrieval). A chunk about "starter hydration" under "Sourdough > Feeding schedule" embeds with that context rather than as isolated paragraphs.
- The breadcrumb text is the same format used by `toc_path` arrays in the search result. This enables the fuzzy→precise drill-down described below.

The `NoteChunk.Content` field (`internal/model/chunk.go`) stores the full `{breadcrumb}\n\n{body}` string. The `snippetFromChunk` function in `resolve.go` strips everything before the first `\n\n` to extract a clean display snippet.

### Token-aware sizing

Chunk boundaries are based on estimated token counts, not character counts:

```
chunkTargetTokens = 450  // keep under bge-m3's 512-token encoder window
chunkMinTokens    = 60
chunkOverlap      = 200  // chars — tail of the previous chunk prepended to the next
```

Cyrillic tokenizes to ~1 token per 2 characters; Latin to ~1 per 4. The old 2000-character target overflowed bge-m3's 512-token window for Russian text, silently truncating the tail of each chunk before it was embedded.

## Per-language BM25 analyzer

The bleve mapping (`internal/noteloader/search.go`, `createSearchIndex`) registers `Title` / `Body` under two analyzed fields each. A query is run as a disjunction across all four fields so an English query stems against the English analyzer, and a Russian query against the Russian analyzer, regardless of whether the note's `lang` frontmatter is set.

This is a correctness fix with no measurable nDCG delta on the bilingual benchmark (the vector lane already dominates cross-lingual queries), but it matters for exact-term English queries — rare words, code identifiers, names — that the vector lane misses.

## Cross-encoder reranker (off by default)

A cross-encoder reranker (`bge-reranker-v2-m3`, self-hosted sidecar) was implemented as an optional second stage after RRF fusion. It measured strictly worse on the benchmark — nDCG@10 dropped 0.9221 → 0.8881, MRR 0.9417 → 0.8708 — because the first stage was already strong and the corpus is dense with topically-adjacent documents. The cross-encoder over-weighted term overlap and promoted near-neighbour distractors that RRF had correctly ranked lower.

The reranker ships **off by default** (`vector_search.reranker.enabled = false`). The client, config, and sidecar are present so it can be A/B-tested per deployment.

Measured results and per-query analysis in [search_refactoring.md F3](./search_refactoring.md).

## Fuzzy→precise drill-down: breadcrumb → TOC → toc_path

The chunk breadcrumb enables a two-step drill-down in the MCP tool (`internal/case/mcp`):

1. A vector hit returns a chunk snippet plus its breadcrumb (`{title} > {h1} > {h2}`).
2. `tocPathForSnippet` (`internal/case/mcp/toc_path.go`) maps the snippet to a `toc_path` array — first by fuzzy HTML section matching, then by parsing the chunk breadcrumb as a fallback, then by falling back to the first section.
3. The caller passes `toc_path` to `note_html(toc_path=[...])` to retrieve only that section of the rendered note.

This means an agent can go from a fuzzy semantic match to the exact paragraph that answered the question, without loading the whole note.

## Embedding generation pipeline

Embeddings are generated asynchronously:

1. A note is created or updated → `HandleLatestNotesAfterSave` enqueues a `GenerateNoteVersionEmbedding` job via goqite.
2. The background worker embeds documents using the **passage** prefix (not the query prefix). Whole-note text is `passagePrefix + title + "\n\n" + strippedContent` (`resolve.go` line 79); each chunk is embedded as `passagePrefix + chunk.Content` (`resolve.go` lines 150–153), where `chunk.Content` already contains the full `{breadcrumb}\n\n{body}` string — the title is not re-concatenated. The query side (search path, `sitesearch/resolve.go` line 128) uses the **query** prefix: `queryPrefix + query`. This query/passage split is the standard asymmetric embedding convention used by e5 and bge model families.
3. A `regenerate_note_embeddings` cronjob runs at startup and weekly on Sunday at 03:00 (`"0 0 3 * * 0"`), comparing content hashes and re-queuing stale chunks.

**Known limitation:** the in-memory chunk cache loads at boot. A note synced after the last boot does not have chunk embeddings in memory until the app restarts. The vecbench stack handles this by waiting for the job queue to drain before restarting (`/debug/wait_all_jobs`).

## Configuration

```bash
FEATURES='{"vector_search": {"enabled": true, "model": "bge-m3"}}'
```

The embedding server URL is configured separately (environment variable `OPENAI_BASE_URL` or equivalent in the app config). The `MODEL_NAME` env var on the embedding server selects the model (default `intfloat/multilingual-e5-base` as shipped in `server.py` and `Dockerfile`; trip2g deployments and benchmarks override it to `BAAI/bge-m3`).

### Reranker config

```bash
FEATURES='{
  "vector_search": {
    "enabled": true,
    "model": "bge-m3",
    "reranker": {
      "enabled": false,
      "base_url": "http://localhost:8082",
      "model": "bge-reranker-v2-m3",
      "top_n": 20,
      "output_k": 10
    }
  }
}'
```

## Package map

```
internal/
├── mdchunk/
│   ├── chunk.go           Split() — breadcrumb + token-aware chunking
│   └── frontmatter.go     StripFrontmatter()
├── model/
│   └── chunk.go           NoteChunk struct, Float32SliceToBytes / BytesToFloat32Slice
├── noteloader/
│   ├── search.go          bleve index creation, per-language mapping, Search()
│   └── loader.go          loads NoteChunks into memory on reload
├── case/
│   ├── sitesearch/
│   │   └── resolve.go     vectorSearch, mergeResults (RRF), dotSimilarity, rerankResults
│   ├── mcp/
│   │   ├── resolve.go     hybrid search in MCP tool (DefaultVectorSearchLimit=50, rrfK=60)
│   │   └── toc_path.go    tocPathForSnippet, tocPathFromChunkContent — breadcrumb→toc_path
│   └── backjob/
│       └── generatenoteversionembedding/
│           └── resolve.go  per-chunk embedding job
├── reranker/
│   └── client.go          cross-encoder reranker client
└── openai/
    └── client.go          embedding API client (OpenAI-compatible)

embedding-server/
└── server.py              FastAPI server, normalize_embeddings=True
```

The reranker has no bespoke server: the vecbench stack runs the official
HuggingFace `text-embeddings-inference` image (`/rerank` endpoint).

## similarNotes (legacy whole-note similarity)

The `similarNotes` GraphQL query predates per-chunk hybrid search. It compares whole-note embeddings stored in `note_version_embeddings` (SQLite) via cosine similarity in-memory. It is not part of the hybrid search pipeline and is unaffected by the F1–F5 changes.

```graphql
similarNotes(input: { noteId: "/my-article", limit: 5 }) {
  score
  note { id title path }
}
```

## Measured results summary

All numbers from the bilingual benchmark (48-note corpus, 60 queries, bge-m3). Full table and analysis: [search_refactoring.md](./search_refactoring.md).

| State | Recall@10 | nDCG@10 | MRR |
|-------|-----------|---------|-----|
| Baseline | 0.9833 | 0.9157 | 0.9417 |
| F1: widen pool (vectorTopK 5→50) | 1.0000 | 0.9221 | 0.9417 |
| F2: per-language analyzer | 1.0000 | 0.9221 | 0.9417 |
| F3: reranker (reverted/off) | — | 0.8881 | 0.8708 |
| F4: breadcrumb + token sizing | 0.9917 | 0.9263 | 0.9500 |
| F5: dot product (shipped) | 0.9917 | 0.9263 | 0.9500 |
| F5✗: AND→OR BM25 (reverted) | 0.6750 | 0.6744 | 0.9556 |

Long-doc track (16 queries, multi-chunk notes): F4 raised nDCG@10 from 0.9308 → 0.9539, en→en from 0.8155 → 0.9077.
