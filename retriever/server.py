import os
import threading
import time

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from sentence_transformers import CrossEncoder, SentenceTransformer


def pick_device():
    """DEVICE env wins; otherwise auto-detect mps -> cuda -> cpu.

    torch is optional here: the wire-contract test tier runs without it
    (see README), so a missing torch just means no MPS/CUDA, i.e. cpu.
    """
    env_device = os.environ.get("DEVICE")
    if env_device:
        return env_device
    try:
        import torch
    except ImportError:
        return "cpu"
    if torch.backends.mps.is_available():
        return "mps"
    if torch.cuda.is_available():
        return "cuda"
    return "cpu"


device = pick_device()

# Single process-wide inference lock: FastAPI runs sync endpoints in a
# threadpool, so concurrent requests can hit the same model object at once.
# On MPS this caused a SIGABRT in the graph compiler; serializing inference
# trades throughput for stability.
inference_lock = threading.Lock()

embed_model_name = os.environ.get("MODEL_NAME", "BAAI/bge-m3")
print(f"Loading embedding model {embed_model_name} on {device}...")
t0 = time.time()
embed_model = SentenceTransformer(embed_model_name, device=device)
embed_model.encode(["warmup"], normalize_embeddings=True)  # pay JIT/compile cost at boot
print(f"Embedding model loaded in {time.time() - t0:.1f}s")

# Micro-batch size for Qwen3Reranker: scoring all pairs in one padded tensor
# makes lm_head materialize a batch x seq x vocab logits tensor, which OOMs
# on MPS for large batches (e.g. 50 passages). Each pair's score is
# independent, so chunking loses nothing.
RERANK_BATCH = int(os.environ.get("RERANK_BATCH", "8"))


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

    def __init__(self, model_name, device="cpu"):
        import torch
        from transformers import AutoModelForCausalLM, AutoTokenizer

        self.torch = torch
        self.device = device
        self.tokenizer = AutoTokenizer.from_pretrained(model_name, padding_side="left")
        # fp16 on non-CPU devices halves memory vs fp32; CPU keeps fp32 as before.
        load_kwargs = {"dtype": torch.float16} if device != "cpu" else {}
        self.model = AutoModelForCausalLM.from_pretrained(model_name, **load_kwargs).eval().to(device)
        self.token_yes = self.tokenizer.convert_tokens_to_ids("yes")
        self.token_no = self.tokenizer.convert_tokens_to_ids("no")

    def predict(self, pairs):
        scores = []
        for i in range(0, len(pairs), RERANK_BATCH):
            scores.extend(self._predict_batch(pairs[i : i + RERANK_BATCH]))
        return scores

    def _predict_batch(self, pairs):
        texts = [
            f"{self.PREFIX}<Instruct>: {self.INSTRUCT}\n<Query>: {query}\n<Document>: {doc}{self.SUFFIX}"
            for query, doc in pairs
        ]
        inputs = self.tokenizer(
            texts, padding=True, truncation="longest_first", max_length=8192, return_tensors="pt"
        )
        inputs = {k: v.to(self.model.device) for k, v in inputs.items()}
        with self.torch.no_grad():
            logits = self.model(**inputs).logits[:, -1, :].float()
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
    print(f"Loading reranker {rerank_model_name} on {device}...")
    t0 = time.time()
    rerank_model = reranker_class(rerank_model_name)(rerank_model_name, device=device)
    rerank_model.predict([("warmup", "warmup")])  # pay JIT/compile cost at boot
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
    with inference_lock:
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
    with inference_lock:
        scores = rerank_model.predict(pairs)
        if device == "mps":
            import torch

            torch.mps.empty_cache()
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
