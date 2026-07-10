# embedding-reranker-server

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
- `GET /health`

One python/sentence-transformers process, both models, runs natively on arm64 and amd64.
`memcli up --embedded` builds and runs it automatically (image `trip2g-embedding-server`).

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

## Run manually

```bash
docker build -t trip2g-embedding-server .
docker run -p 8000:8000 -v "${MODELS_DIR:-$HOME/models}":/data \
  -e LOAD_RERANKER=1 trip2g-embedding-server
```
