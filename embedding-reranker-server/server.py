import os
import time

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from sentence_transformers import CrossEncoder, SentenceTransformer

embed_model_name = os.environ.get("MODEL_NAME", "BAAI/bge-m3")
print(f"Loading embedding model {embed_model_name}...")
t0 = time.time()
embed_model = SentenceTransformer(embed_model_name)
print(f"Embedding model loaded in {time.time() - t0:.1f}s")

# The reranker is optional: loading it costs ~2GB of RAM, so it is only
# loaded when LOAD_RERANKER is set (memcli --reranker).
rerank_model = None
rerank_model_name = os.environ.get("RERANKER_MODEL", "BAAI/bge-reranker-v2-m3")
if os.environ.get("LOAD_RERANKER", "") not in ("", "0", "false"):
    print(f"Loading reranker {rerank_model_name}...")
    t0 = time.time()
    rerank_model = CrossEncoder(rerank_model_name)
    print(f"Reranker loaded in {time.time() - t0:.1f}s")

app = FastAPI()


class EmbedRequest(BaseModel):
    input: str | list[str]
    model: str = ""


@app.post("/v1/embeddings")
def embed(req: EmbedRequest):
    texts = [req.input] if isinstance(req.input, str) else req.input
    # normalize_embeddings=True: the app's dot-product similarity assumes unit
    # vectors (cosine ≡ dot), same guarantee TEI hardcodes for /v1/embeddings.
    vecs = embed_model.encode(texts, normalize_embeddings=True).tolist()
    tokens = sum(len(t.split()) for t in texts)
    return {
        "object": "list",
        "data": [
            {"object": "embedding", "index": i, "embedding": v}
            for i, v in enumerate(vecs)
        ],
        "model": embed_model_name,
        "usage": {"prompt_tokens": tokens, "total_tokens": tokens},
    }


# TEI-style contract: {"query","texts"} in, bare [{"index","score"}] out
# (matches internal/reranker/client.go).
class RerankRequest(BaseModel):
    query: str
    texts: list[str]
    model: str = ""


@app.post("/rerank")
def rerank(req: RerankRequest):
    if rerank_model is None:
        raise HTTPException(status_code=503, detail="reranker not loaded (set LOAD_RERANKER=1)")
    if not req.texts:
        return []
    pairs = [(req.query, text) for text in req.texts]
    scores = rerank_model.predict(pairs)
    results = [
        {"index": i, "score": float(s)}
        for i, s in enumerate(scores)
    ]
    results.sort(key=lambda r: r["score"], reverse=True)
    return results


@app.get("/health")
def health():
    return {"status": "ok", "reranker": rerank_model is not None}
