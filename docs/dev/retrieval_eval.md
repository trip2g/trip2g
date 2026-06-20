# Retrieval evaluation (vector-search benchmark)

A reproducible benchmark for trip2g's hybrid search (BM25 + vector, fused with RRF). It measures **Recall@k**, **nDCG@k**, and **MRR** over a versioned golden set of bilingual RU/EN queries, so every change to the search pipeline can be A/B-tested with numbers instead of guesses.

## What's where

| Path | Role |
|------|------|
| `docker-compose.vecbench.yml` | Isolated stack: one app + MinIO + a bge-m3 embedding server, on its own ports (21081/21082, MinIO 22000/22001, embedding 11436). Independent of the e2e stack and dev. |
| `scripts/vecbench.sh` | Driver: `up` / `sync` / `eval` / `down` / `logs`. |
| `testdata/vecbench/vault/` | The corpus: 6 topics × {EN, RU} = 12 notes, so all four retrieval directions are testable. See its `README.md`. |
| `testdata/eval/golden_set.json` | Versioned query → relevant-URL pairs (qrels). |
| `cmd/evalretrieval/` | CLI: runs the golden set against a live GraphQL endpoint, prints metrics, writes a JSON artifact, gates on a min nDCG. |
| `internal/retrievaleval/` | Pure-Go metrics + golden-set loader + GraphQL search client + report aggregation. |
| `docs/superpowers/eval-runs/*.json` | Committed run artifacts — the data spine for the before/after comparison. |

## Prerequisites

- Docker (the stack builds the app image and runs a local bge-m3 server).
- The bge-m3 model (~2.3 GB) downloads on first run into the shared `~/models/embedding` cache, reused across the e2e and vecbench stacks.
- Node (for `obsidian-sync`, used to push the vault).

## Run it

```bash
# 1. Bring up the isolated stack, push the vault, generate embeddings, reload.
./scripts/vecbench.sh up        # or: make vecbench-up

# 2. Run the benchmark; writes docs/superpowers/eval-runs/<label>.json
./scripts/vecbench.sh eval baseline-bgem3

# 3. Tear down when done (keeps the model cache).
./scripts/vecbench.sh down
```

After editing the vault, re-ingest with `./scripts/vecbench.sh sync` (re-pushes, drains embedding jobs, reloads the app).

## How it works (and two gotchas)

1. **The embeddings endpoint is required.** The vector lane embeds the query live on every request, so the benchmark needs a reachable bge-m3 server. The compose file provides one.
2. **Embeddings are generated *after* the vault sync, by async jobs.** The in-memory chunk cache (and the bleve index) load at boot, so right after a fresh sync the vector lane is empty until the app reloads. `vecbench.sh up`/`sync` handle this by waiting for the job queue to drain (`/debug/wait_all_jobs`) and then restarting the app. (This stale-until-reload behaviour is a known limitation of the current loader — see `docs/dev/vector_search.md`.)
3. **The benchmark searches as admin.** Anonymous search returns only *live* notes; the synced vault is draft/latest. `vecbench.sh eval` signs in with the dev code and passes the session JWT as a bearer token so the CLI sees the full (latest) index.

## Building the golden set

Public benchmarks (MIRACL, ruMTEB, BEIR) measure embedding-model quality, but no public set matches this corpus. The golden set is small and project-specific:

- Generate candidate queries with an LLM (3 per note: synonyms, no copied phrases, plus one "need-based" phrasing), in both languages, including cross-lingual (RU query → EN note and vice versa — the topics are parallel).
- **Hand-verify every pair**: run it through the live endpoint, confirm the intended note is genuinely correct, and set `expected_urls` to the actual permalink(s) the endpoint returns. Drop ambiguous pairs (expect ~20% loss). Only `verified: true` pairs count.
- Keep the four `direction` buckets (`ru->ru`, `en->en`, `ru->en`, `en->ru`) roughly balanced so cross-lingual regressions surface.

For picking/justifying the embedding model itself, the relevant public benchmarks are MIRACL-ru, ruMTEB retrieval, RusBEIR, NeuCLIRBench (EN→RU cross-lingual), and small BEIR sets (SciFact/NFCorpus) as an English CI guard.

## Metrics

| Metric | Meaning |
|--------|---------|
| Recall@k | fraction of relevant notes found in the top k |
| nDCG@k | rank-discounted relevance (rewards putting the right note higher) |
| MRR | mean of 1/rank of the first relevant result |

The CLI reports each overall and per `direction`. A run with `-fail-under-ndcg X` (or `EVAL_MIN_NDCG=X`) exits nonzero when overall nDCG@k drops below `X` — use this as a CI regression gate once a baseline is established.

## Run history

| Run | Recall@10 | nDCG@10 | MRR | Notes |
|-----|-----------|---------|-----|-------|
| _(baseline pending — Phase 1)_ | | | | bge-m3, full golden set |
