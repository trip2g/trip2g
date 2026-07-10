"""Contract tests for the embedding-reranker-server.

Two tiers:
- Always-on: wire-shape tests against fake models (no downloads, no torch).
  They pin the OpenAI /v1/embeddings shape, the TEI /rerank shape the Go
  client (internal/reranker/client.go) depends on, /health, and the
  LOAD_RERANKER gating — including that the server requests L2-normalized
  embeddings (normalize_embeddings=True).
- REAL_MODELS=1: loads the actual bge-m3 / bge-reranker-v2-m3 models and
  asserts the numeric guarantees (1024 dims, unit norm, semantic ranking).
  Heavy (~2.4GB RAM); run via docker as documented in README.md.
"""

import importlib.util
import math
import os
import sys
import types
import uuid
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

SERVER_PY = Path(__file__).parent / "server.py"

REAL_MODELS = os.environ.get("REAL_MODELS", "") not in ("", "0", "false")


class _FakeArray:
    def __init__(self, rows):
        self.rows = rows

    def tolist(self):
        return self.rows


class FakeEmbedder:
    """Deterministic unit vectors (dim 1024); records encode() kwargs."""

    last_kwargs = None

    def __init__(self, model_name):
        self.model_name = model_name

    def encode(self, texts, **kwargs):
        FakeEmbedder.last_kwargs = kwargs
        rows = []
        for t in texts:
            v = [0.0] * 1024
            v[sum(ord(c) for c in t) % 1024] = 1.0
            rows.append(v)
        return _FakeArray(rows)


class FakeCrossEncoder:
    """Scores a pair high when a query word appears in the text."""

    def __init__(self, model_name):
        self.model_name = model_name

    def predict(self, pairs):
        scores = []
        for query, text in pairs:
            words = set(query.lower().split())
            scores.append(0.9 if words & set(text.lower().split()) else 0.1)
        return scores


def load_client(monkeypatch, load_reranker):
    """Import a fresh server module against the fake models."""
    fake = types.ModuleType("sentence_transformers")
    fake.SentenceTransformer = FakeEmbedder
    fake.CrossEncoder = FakeCrossEncoder
    monkeypatch.setitem(sys.modules, "sentence_transformers", fake)
    if load_reranker:
        monkeypatch.setenv("LOAD_RERANKER", "1")
    else:
        monkeypatch.delenv("LOAD_RERANKER", raising=False)

    spec = importlib.util.spec_from_file_location(f"server_{uuid.uuid4().hex}", SERVER_PY)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return TestClient(mod.app)


def test_embeddings_openai_shape(monkeypatch):
    client = load_client(monkeypatch, load_reranker=False)
    resp = client.post("/v1/embeddings", json={"model": "bge-m3", "input": "hello"})
    assert resp.status_code == 200
    body = resp.json()
    assert body["object"] == "list"
    assert len(body["data"]) == 1
    item = body["data"][0]
    assert item["object"] == "embedding"
    assert item["index"] == 0
    assert len(item["embedding"]) == 1024
    assert body["usage"]["prompt_tokens"] >= 1
    assert body["usage"]["total_tokens"] >= 1


def test_embeddings_batch_input(monkeypatch):
    client = load_client(monkeypatch, load_reranker=False)
    resp = client.post("/v1/embeddings", json={"input": ["one", "two", "three"]})
    assert resp.status_code == 200
    data = resp.json()["data"]
    assert [d["index"] for d in data] == [0, 1, 2]


def test_embeddings_request_normalization(monkeypatch):
    # The app's dot-product similarity assumes unit vectors: the server MUST
    # ask sentence-transformers for L2-normalized output.
    client = load_client(monkeypatch, load_reranker=False)
    client.post("/v1/embeddings", json={"input": "check"})
    assert FakeEmbedder.last_kwargs.get("normalize_embeddings") is True


def test_rerank_tei_wire_shape(monkeypatch):
    # internal/reranker/client.go decodes a BARE array [{"index","score"}].
    client = load_client(monkeypatch, load_reranker=True)
    resp = client.post("/rerank", json={"query": "cats", "texts": ["dogs bark", "cats purr", "stocks"]})
    assert resp.status_code == 200
    out = resp.json()
    assert isinstance(out, list)
    assert {r["index"] for r in out} == {0, 1, 2}
    assert set(out[0].keys()) == {"index", "score"}
    scores = [r["score"] for r in out]
    assert scores == sorted(scores, reverse=True), "must be sorted by score desc"
    assert out[0]["index"] == 1, "text containing the query word must rank first"


def test_rerank_empty_texts(monkeypatch):
    client = load_client(monkeypatch, load_reranker=True)
    resp = client.post("/rerank", json={"query": "q", "texts": []})
    assert resp.status_code == 200
    assert resp.json() == []


def test_rerank_unavailable_without_load_reranker(monkeypatch):
    client = load_client(monkeypatch, load_reranker=False)
    resp = client.post("/rerank", json={"query": "q", "texts": ["a", "b"]})
    assert resp.status_code == 503
    assert "LOAD_RERANKER" in resp.json()["detail"]


def test_health_reflects_reranker_flag(monkeypatch):
    on = load_client(monkeypatch, load_reranker=True).get("/health").json()
    assert on == {"status": "ok", "reranker": True}
    off = load_client(monkeypatch, load_reranker=False).get("/health").json()
    assert off == {"status": "ok", "reranker": False}


# ---- REAL_MODELS=1: numeric guarantees with the actual models ----


@pytest.fixture(scope="module")
def real_client():
    if not REAL_MODELS:
        pytest.skip("set REAL_MODELS=1 to run model-backed tests (~2.4GB RAM)")
    os.environ["LOAD_RERANKER"] = "1"
    spec = importlib.util.spec_from_file_location("server_real", SERVER_PY)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return TestClient(mod.app)


def test_real_embeddings_dims_and_unit_norm(real_client):
    resp = real_client.post("/v1/embeddings", json={"model": "bge-m3", "input": "hello world"})
    assert resp.status_code == 200
    vec = resp.json()["data"][0]["embedding"]
    assert len(vec) == 1024
    norm = math.sqrt(sum(x * x for x in vec))
    assert abs(norm - 1.0) < 1e-3, f"embedding must be L2-normalized, got norm {norm}"


def test_real_rerank_semantic_ranking(real_client):
    resp = real_client.post(
        "/rerank",
        json={"query": "cats", "texts": ["stock market news", "felines purr softly"]},
    )
    assert resp.status_code == 200
    out = resp.json()
    assert out[0]["index"] == 1, "semantically relevant text must outrank the irrelevant one"
    assert out[0]["score"] > out[1]["score"]
