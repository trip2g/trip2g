# retriever

**TL;DR:** a small local server that replaces the official TEI image on ARM hosts.
TEI (`ghcr.io/huggingface/text-embeddings-inference`) publishes no arm64 manifest
for any CPU tag, so on Apple-silicon / ARM boxes it simply cannot run. The vector
search stack is defined by its wire protocol, not its engine — this server speaks
the same two contracts, so trip2g's clients work unchanged:

- `POST /v1/embeddings` — OpenAI shape, `BAAI/bge-m3` (1024 dims), **L2-normalized**
  output (the app's dot-product similarity assumes unit vectors, same guarantee
  TEI hardcodes).
- `POST /rerank` — TEI wire: request `{"query","texts"}`, response a bare array
  `[{"index","score"}]` sorted by score desc (`BAAI/bge-reranker-v2-m3` cross-encoder).
  Loaded only when `LOAD_RERANKER=1` (saves ~2GB RAM otherwise).
- `GET /health` — reports whether a reranker is loaded and which model.

Reranker backend is selected by `RERANKER_MODEL` (one per boot):
- default `BAAI/bge-reranker-v2-m3` — sentence-transformers CrossEncoder;
- `RERANKER_MODEL=Qwen/Qwen3-Reranker-0.6B` — LLM-logit reranker (temporary,
  for engine comparisons): the model is a causal LM, each (query, doc) is
  scored as softmax([no, yes])[yes] of the last-position logits under the
  official yes/no judging template. Same `/rerank` wire either way.

One python/sentence-transformers process, both models, runs natively on arm64 and amd64.
`memcli up --embedded` builds and runs it automatically (image `trip2g-embedding-server`).

## Device selection (Metal/MPS)

`DEVICE` picks the torch device explicitly (`cpu`, `mps`, `cuda`); unset, the
server auto-detects `mps` -> `cuda` -> `cpu`. This mainly matters for running
the server natively on Apple-silicon dev boxes (Docker on macOS has no Metal
access, so Metal only helps outside the container).

On non-CPU devices the `Qwen3Reranker` backend loads in fp16 (softmax math
still runs in fp32) and the `/rerank` endpoint calls `torch.mps.empty_cache()`
after each request — torch's MPS allocator otherwise grows unbounded across
requests of different sizes. `RERANK_BATCH` (default 8) chunks the pairs
passed to the reranker so a single large batch doesn't make lm_head
materialize a batch x seq x vocab logits tensor and OOM.

All inference (`/v1/embeddings` and `/rerank`) is serialized behind a single
process-wide lock — FastAPI runs sync endpoints in a threadpool, and
concurrent calls into the same model crashed the MPS graph compiler. This is
a deliberate throughput-for-stability tradeoff; it also applies on CPU/CUDA,
where it costs nothing today since nothing else contends for it. CPU
behavior (default, no `DEVICE` set) is unchanged: fp32, no device moves, no
`empty_cache` calls.

## Model cache

Models (~2GB each) are cached under `HF_HOME` (`/data` inside the container).
memcli mounts the host dir from the `MODELS_DIR` env var, default `${HOME}/models`.
Point it elsewhere for a shared cache:

```bash
MODELS_DIR=/mnt/extssd/models memcli up --embedded --reranker ...
```

Notes:
- An older TEI run may have created `~/models` as root — not removable/writable
  without sudo. On this box the alexes-owned copy lives at `/mnt/extssd/models`;
  use that via `MODELS_DIR`.
- The cache layout is the standard HF hub tree: `<MODELS_DIR>/hub/models--BAAI--bge-m3`
  (and `models--BAAI--bge-reranker-v2-m3`). Pre-populated caches in per-model
  subdirs (`embedding/hub`, `reranker/hub`) can be consolidated with symlinks.

## Tests

`test_server.py` has two tiers:

- **Always-on** (no models, no torch): wire-shape contracts — OpenAI `/v1/embeddings`
  shape, TEI `/rerank` bare-array shape, `/health`, `LOAD_RERANKER` gating, and
  that the server requests L2-normalized output.

  ```bash
  pip install fastapi httpx pytest && pytest test_server.py
  ```

- **`REAL_MODELS=1`** (loads both models, ~2.4GB RAM): 1024 dims, unit norm,
  semantic ranking. Easiest inside the docker image with the shared cache:

  ```bash
  docker run --rm -e REAL_MODELS=1 -e HF_HOME=/data \
    -v "${MODELS_DIR:-$HOME/models}":/data -v "$PWD":/app-src \
    trip2g-embedding-server sh -c "pip -q install pytest httpx && cd /app-src && python -m pytest test_server.py"
  ```

## Run manually

```bash
docker build -t trip2g-embedding-server .
docker run -p 8000:8000 -v "${MODELS_DIR:-$HOME/models}":/data \
  -e LOAD_RERANKER=1 trip2g-embedding-server
```
