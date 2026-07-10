# Cross-encoder reranker (blend mode)

**TL;DR:** search has an optional second-stage cross-encoder reranker. It does
**not** replace the first-stage (RRF) order — it **blends** its score with the
RRF score, so a strong first stage keeps control and the cross-encoder only
nudges. It reranks **windowed passages** (the best-matching chunk per note,
~450 tokens), never truncated whole notes. It is **off by default** and gated
behind `vector_search.reranker.enabled`. Whether to turn it on is a benchmark
decision (see [Benchmark](#benchmark)).

This replaces an earlier *replace-mode* reranker that measured strictly worse
and was removed. The failure and the fix are documented below so we don't
repeat the mistake.

## Why a reranker exists

Search is hybrid: a bi-encoder vector lane (ANN over `bge-m3` embeddings) plus a
BM25 lane, fused with Reciprocal Rank Fusion (RRF). Both stages are *cheap
scorers*: the bi-encoder embeds query and document **independently**, so it can
miss fine query↔document interactions.

A cross-encoder scores a `(query, document)` pair **jointly** in one forward
pass. That is strictly more expressive than independent embeddings, which is why
retrieve-then-rerank is a standard IR pattern — a cheap recall stage proposes
candidates, an expensive precision stage reorders the top-N. It needs a
rerank-capable server (the official HuggingFace `text-embeddings-inference`
image, or Infinity), not the plain `/embeddings` endpoint: different
architecture, different API (pair scoring vs vectors).

## When it helps vs when it hurts

A cross-encoder is not free precision. It helps in some regimes and hurts in
others:

**Helps when:**
- the **first stage is weak** (vector-only, or BM25-only, no fusion) — there is
  order left to fix;
- the corpus is **large and diverse**, so true answers and distractors are far
  apart and the CE's extra expressiveness separates them;
- inputs are **short, windowed passages** that fit the CE's ~512-token window,
  so the model sees the whole candidate, not a truncated head.

**Hurts when:**
- the **first stage is already strong** (our hybrid RRF is ~0.92 nDCG) — there
  is little to gain and plenty to break;
- the corpus is **small and topically adjacent** — near-neighbour distractors
  share query terms, and a CE that over-weights surface term overlap promotes
  them over the true answer;
- inputs are **whole notes longer than the window** — after truncation the
  candidates look alike and the ordering degrades to noise.

Our site-search corpus sits squarely in the *hurts* column on the first two
axes. That is exactly why replace mode failed and why blend mode is conservative
by construction.

## The old failure: replace mode

The first attempt re-scored the fused top-N and **replaced** the order with the
cross-encoder's ranking. It measured strictly worse, twice
(`docs/dev/search_refactoring.md`, variants F3 / F3b):

| variant | nDCG@10 | MRR | Δ nDCG |
|---|---|---|---|
| no reranker (hybrid RRF) | 0.9157 | 0.9417 | baseline |
| F3: replace, 512-char passages | 0.8881 | 0.8708 | **−0.034** |
| F3b: replace, full-text passages | 0.3880 | 0.4500 | **−0.534** |

Per-query, the cross-encoder over-weighted surface query↔passage term overlap
and promoted near-neighbour distractors the strong first stage had correctly
ranked *below* the answer:

- "медленное брожение теста" (slow dough fermentation) → promoted *sauerkraut*
  over sourdough;
- "каналы вместо общей памяти" (channels vs shared memory) → promoted *mutexes*
  over goroutines;
- "пул воркеров" (worker pool) → promoted *errgroup* over goroutines.

On full-note passages (F3b) it collapsed to ~0.39: passages far exceeded the
~512-token window, so after truncation the notes looked identical and the order
became noise, pushing relevant notes out of the top-10.

**Two root causes:** (1) *overriding* a good prior instead of combining with it,
and (2) feeding *truncated whole notes* past the model window. Blend mode fixes
both.

## The fix: blend mode + window discipline

### Blend, don't replace

The cross-encoder score is combined with the first-stage RRF score rather than
substituted for it:

```
final = (1 - w) * rrf_norm + w * ce_norm
```

- `rrf_norm` — the candidate's RRF score, min-max normalised to [0,1] over the
  reranked head.
- `ce_norm` — the cross-encoder score, min-max normalised to [0,1] over the
  same head.
- `w` = `vector_search.reranker.blend_weight`, default **0.5**.

At `w=0` the reranker is a no-op (pure RRF order). At `w=1` it is the old
replace mode. In between, the RRF prior stays in control and the cross-encoder
nudges — a confident CE can flip a *near-tie*, but it cannot overturn a clearly
better first-stage candidate. This is the whole point: the CE corrects, it does
not dominate.

Implementation: `rerankResults` in `internal/case/sitesearch/resolve.go`;
scores come from `internal/reranker`. On any reranker error the fused order is
returned unchanged (graceful degradation, same as a failing vector lane).

### Window discipline

We rerank the **best-matching chunk passage** per candidate note, not the whole
note. Chunks are capped at ~450 tokens at ingest (`internal/mdchunk`), safely
under the CE's ~512-token window, so the model always sees a complete passage.

The winning passage per note comes from the vector lane's own scan
(`passageByURL`). A candidate that has **no** window-sized passage (a text-only
BM25 hit with no chunk) is **not** sent to the cross-encoder — it keeps its
stage-1 RRF prior and is never truncated into the window. This directly prevents
the F3b collapse.

## Configuration

```json
{
  "vector_search": {
    "enabled": true,
    "model": "bge-m3",
    "base_url": "http://embedding:8000/v1",
    "reranker": {
      "enabled": true,
      "base_url": "http://reranker:8000/rerank",
      "model": "BAAI/bge-reranker-v2-m3",
      "top_n": 50,
      "output_k": 20,
      "blend_weight": 0.5
    }
  }
}
```

| field | default | meaning |
|---|---|---|
| `enabled` | `false` | turn the second stage on |
| `base_url` | — | rerank endpoint (required when enabled) |
| `model` | — | cross-encoder model name |
| `top_n` | 50 | fused candidates rescored |
| `output_k` | 20 | results kept after blend |
| `blend_weight` | 0.5 | CE weight `w` in the blend |
| `timeout_seconds` | 10 | per-request rerank timeout; raise for CPU inference |

The vecbench stack serves `bge-reranker-v2-m3` on CPU via the official
HuggingFace `text-embeddings-inference` image. First boot downloads the model
(~2.2 GB). It is intentionally **not** in the vecbench `app.depends_on`, so
baseline runs never pull the model. CPU inference is slow:
a warm rerank of 20 passages takes ~6 s, of 50 passages ~25 s on a loaded box —
so raise `timeout_seconds` well above the 10 s default when serving on CPU, or
the lane silently degrades to the RRF order on timeout.

## Benchmark

Eval harness: `cmd/evalretrieval` against a running instance, golden set
`testdata/eval/golden_set.json` (60 hand-verified bilingual queries, four
directions). Driver: `scripts/vecbench.sh` on the isolated
`docker-compose.vecbench.yml` stack. Metrics: nDCG@10 / MRR / Recall@10, on the
**passage granularity** (chunks, not whole notes). Both configs run against the
same `bge-m3` hybrid first stage; the only change is whether the blended
reranker second stage is applied.

Two configs, `blend_weight=0.5`, `top_n=20` (blend run had **zero** rerank
timeouts):

| config | Recall@10 | nDCG@10 | MRR | Δ nDCG |
|---|---|---|---|---|
| stage-1 only (reranker off) | 0.9917 | 0.9263 | 0.9500 | baseline |
| blend reranker (w=0.5) | 0.9917 | **0.9491** | **0.9694** | **+0.0228** |

Per-direction nDCG@10 — the blend improves **every** direction, most where the
first stage was weakest (cross-language `en→ru`):

| direction | stage-1 | blend | Δ |
|---|---|---|---|
| ru→ru | 0.9579 | 0.9716 | +0.0136 |
| en→en | 0.9410 | 0.9478 | +0.0068 |
| ru→en | 0.9160 | 0.9428 | +0.0269 |
| en→ru | 0.8669 | 0.9237 | **+0.0568** |

Artifacts: `docs/superpowers/eval-runs/blend-00-stage1-off.json`,
`docs/superpowers/eval-runs/blend-01-w050.json`.

This is the mirror image of the old replace-mode result (−0.034): keeping the
RRF prior and only *nudging* with the cross-encoder turns a regression into a
net win, and the biggest gains land on the weak cross-language direction — the
regime where the first stage had the most room to be corrected.

**Decision: blend beats stage-1 on the benchmark → recommend enabling it where
the reranker sidecar is available.** The `enabled` flag stays **default-off**
because the second stage needs an extra service (the CPU cross-encoder sidecar)
and adds ~6–25 s of latency per query on CPU — it is opt-in infrastructure, not
a free default. Turn it on (with a generous `timeout_seconds`) when the sidecar
is deployed; the measured quality gain justifies it.
