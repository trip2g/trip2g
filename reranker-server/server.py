import os
import time

from fastapi import FastAPI
from pydantic import BaseModel
from sentence_transformers import CrossEncoder

model_name = os.environ.get("MODEL_NAME", "BAAI/bge-reranker-v2-m3")
print(f"Loading reranker {model_name}...")
t0 = time.time()
model = CrossEncoder(model_name)
print(f"Reranker loaded in {time.time() - t0:.1f}s")

app = FastAPI()


# TEI-style contract: {"query","texts"} in, bare [{"index","score"}] out
# (matches internal/reranker/client.go).
class RerankRequest(BaseModel):
    query: str
    texts: list[str]
    model: str = ""


@app.post("/rerank")
def rerank(req: RerankRequest):
    if not req.texts:
        return []
    pairs = [(req.query, text) for text in req.texts]
    scores = model.predict(pairs)
    results = [
        {"index": i, "score": float(s)}
        for i, s in enumerate(scores)
    ]
    results.sort(key=lambda r: r["score"], reverse=True)
    return results


@app.get("/health")
def health():
    return {"status": "ok"}
