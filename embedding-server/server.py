import os
import time

from fastapi import FastAPI
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer

model_name = os.environ.get("MODEL_NAME", "intfloat/multilingual-e5-base")
print(f"Loading model {model_name}...")
t0 = time.time()
model = SentenceTransformer(model_name)
print(f"Model loaded in {time.time() - t0:.1f}s")

app = FastAPI()


class EmbedRequest(BaseModel):
    input: str | list[str]
    model: str = ""


@app.post("/v1/embeddings")
def embed(req: EmbedRequest):
    texts = [req.input] if isinstance(req.input, str) else req.input
    vecs = model.encode(texts, normalize_embeddings=True).tolist()
    tokens = sum(len(t.split()) for t in texts)
    return {
        "object": "list",
        "data": [
            {"object": "embedding", "index": i, "embedding": v}
            for i, v in enumerate(vecs)
        ],
        "model": model_name,
        "usage": {"prompt_tokens": tokens, "total_tokens": tokens},
    }


@app.get("/health")
def health():
    return {"status": "ok"}
