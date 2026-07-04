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


class RerankRequest(BaseModel):
    query: str
    documents: list[str]
    model: str = ""


@app.post("/rerank")
def rerank(req: RerankRequest):
    if not req.documents:
        return {"results": []}
    pairs = [(req.query, doc) for doc in req.documents]
    scores = model.predict(pairs)
    results = [
        {"index": i, "relevance_score": float(s)}
        for i, s in enumerate(scores)
    ]
    results.sort(key=lambda r: r["relevance_score"], reverse=True)
    return {"results": results}


@app.get("/health")
def health():
    return {"status": "ok"}
