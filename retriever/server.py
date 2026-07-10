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

class Qwen3Reranker:
    """LLM-logit reranker for Qwen/Qwen3-Reranker-* (not a CrossEncoder).

    Per the model card: each (query, doc) is formatted with the official
    yes/no judging template, one forward pass is run, and the score is
    softmax([no_logit, yes_logit])[yes] at the last position.
    """

    INSTRUCT = "Given a web search query, retrieve relevant passages that answer the query"
    PREFIX = (
        "<|im_start|>system\nJudge whether the Document meets the requirements "
        "based on the Query and the Instruct provided. Note that the answer can "
        'only be "yes" or "no".<|im_end|>\n<|im_start|>user\n'
    )
    SUFFIX = "<|im_end|>\n<|im_start|>assistant\n<think>\n\n</think>\n\n"

    def __init__(self, model_name):
        import torch
        from transformers import AutoModelForCausalLM, AutoTokenizer

        self.torch = torch
        self.tokenizer = AutoTokenizer.from_pretrained(model_name, padding_side="left")
        self.model = AutoModelForCausalLM.from_pretrained(model_name).eval()
        self.token_yes = self.tokenizer.convert_tokens_to_ids("yes")
        self.token_no = self.tokenizer.convert_tokens_to_ids("no")

    def predict(self, pairs):
        texts = [
            f"{self.PREFIX}<Instruct>: {self.INSTRUCT}\n<Query>: {query}\n<Document>: {doc}{self.SUFFIX}"
            for query, doc in pairs
        ]
        inputs = self.tokenizer(
            texts, padding=True, truncation="longest_first", max_length=8192, return_tensors="pt"
        )
        with self.torch.no_grad():
            logits = self.model(**inputs).logits[:, -1, :]
        stacked = self.torch.stack([logits[:, self.token_no], logits[:, self.token_yes]], dim=1)
        return self.torch.nn.functional.softmax(stacked, dim=1)[:, 1].tolist()


def reranker_class(model_name):
    if model_name.startswith("Qwen/Qwen3-Reranker"):
        return Qwen3Reranker
    return CrossEncoder


# The reranker is optional: loading it costs ~2GB of RAM, so it is only
# loaded when LOAD_RERANKER is set (memcli --reranker). One backend per boot,
# selected by RERANKER_MODEL.
rerank_model = None
rerank_model_name = os.environ.get("RERANKER_MODEL", "BAAI/bge-reranker-v2-m3")
if os.environ.get("LOAD_RERANKER", "") not in ("", "0", "false"):
    print(f"Loading reranker {rerank_model_name}...")
    t0 = time.time()
    rerank_model = reranker_class(rerank_model_name)(rerank_model_name)
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
    return {
        "status": "ok",
        "reranker": rerank_model is not None,
        "reranker_model": rerank_model_name if rerank_model is not None else None,
    }
