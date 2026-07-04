---
title: "We Built a Benchmark, Then Fixed Five Search Bugs"
free: true
lang_redirect: "[[ru/thoughts/search-benchmark]]"
---

Here is what we found when we actually measured trip2g's vector search.

| Change | Recall@10 | nDCG@10 | MRR | en→ru nDCG |
|--------|-----------|---------|-----|------------|
| Baseline | 0.983 | 0.916 | 0.942 | 0.845 |
| F1: widen fusion pool | **1.000** | 0.922 | 0.942 | 0.860 |
| F2: per-language analyzer | 1.000 | 0.922 | 0.942 | 0.860 |
| F3: cross-encoder reranker (replace mode) | 1.000 | **0.888** | 0.871 | 0.879 |
| → shipped reranker OFF | 1.000 | 0.922 | 0.942 | 0.860 |
| → re-added as blend (opt-in, w=0.5), re-measured | 0.992 | **0.949** | 0.969 | **0.924** |
| F4: heading breadcrumb + token-aware chunks | 0.992 | **0.926** | 0.950 | 0.867 |
| F5: dot-product similarity | 0.992 | 0.926 | 0.950 | 0.867 |
| F5✗: AND→OR BM25 fallback (reverted) | 0.675 | **0.674** | 0.956 | 0.606 |

Long-document track (notes split across 6–10 chunks):

| Change | nDCG@10 | en→en nDCG | Δ nDCG |
|--------|---------|------------|--------|
| Baseline (post F1–F3) | 0.931 | 0.816 | — |
| F4: heading breadcrumb | **0.954** | **0.908** | +0.023 |
| F5✗: AND→OR fallback (reverted) | 0.766 | 0.908 | −0.188 |

Numbers come from committed eval artifacts in `docs/superpowers/eval-runs/`. The story behind them is below.

---

## The setup

trip2g's search is hybrid: BM25 (bleve) fused with per-chunk vector search (bge-m3, 1024 dimensions) via Reciprocal Rank Fusion at k=60. We suspected the implementation had bugs but did not know which ones mattered. So we built a benchmark.

The corpus is 48 notes: six themes (sourdough bread, fermented tea, goroutines, channels, sauerkraut, green tea) with a correct answer note and three near-neighbour distractors per theme, in both English and Russian. A query about sourdough has to surface the sourdough note above the sauerkraut and yeast-bread distractors. The golden set is 60 hand-labeled queries across four language directions.

We kept the same bge-m3 model throughout. Every change was applied in isolation and the numbers were committed before the next change started.

---

## F1: a truncation bug that silently discarded vector recall

The first fix was the most impactful and also the most embarrassing.

The code computed cosine similarity for every chunk in the index — that part worked fine. Then it kept only the top `vectorTopK = 5` unique notes before merging with BM25 via RRF. So if a note ranked #6 by the vector lane but high by BM25, it contributed zero vector signal to fusion, even though the similarity had already been computed and paid for.

The cross-lingual note (English query → Russian note) was the systematic victim. It ranked in the 6–50 range by vector, never made it into the RRF merge, and the BM25 lane cannot match across languages. So it simply vanished from results.

Fix: raise `vectorTopK` to 50. The final result cap (20 hits) stays the same; only the candidate pool before fusion widens.

**Result: Recall@10 0.983 → 1.000.** nDCG +0.006, en→ru +0.015. MRR unchanged — F1 adds candidates below the top hit, which is why MRR does not move.

---

## F2: English text analyzed with Russian stemming rules

The BM25 index analyzed all content with the Russian analyzer, so the query "embedding" would be stemmed and matched against text that said "embeddings" — using Russian morphological rules applied to an English word.

A second bug compounded this: the document mapping was registered under a named type, but notes were indexed as plain structs with no type field, so bleve silently fell back to the dynamic default and the named mapping never applied.

Fix: index Title and Body under two fields each — one with a Russian analyzer, one with English (`Title_en`, `Body_en`). Query via a per-field disjunction so each field's query string is analyzed by its own stemmer.

**Result: nDCG Δ 0.000 on this benchmark.** Zero, and honestly we expected that. The bilingual golden set relies on the vector lane for cross-language matching (BM25 cannot do that by definition), and bge-m3 already ranks monolingual queries well. What F2 fixes is the class of exact-term English queries — rare words, code identifiers, names — where BM25 should be stronger than the embedder. That class of query is not in the current golden set. The fix has zero regression and is correct, so it shipped.

---

## F3: the reranker — worse as a replacement, better as a blend

The plan was to add a cross-encoder reranker as a second stage, the textbook "biggest quality lever." We added `bge-reranker-v2-m3` behind a feature flag, re-scoring the fused top-N before returning results.

In its first form it was the only change that made things actively worse.

With 512-character passages: nDCG 0.922 → 0.888, MRR 0.942 → 0.871. With full-note passages it collapsed to nDCG 0.388: passages exceeded the cross-encoder's ~512-token window, content was truncated, the notes looked identical to the reranker, and it reordered them by noise.

The per-query diffs showed the failure mode. The cross-encoder promotes surface term overlap, so "медленное брожение теста" (dough fermentation) surfaced the sauerkraut note above sourdough, because sauerkraut has more fermentation vocabulary. "Каналы вместо общей памяти" (channels vs shared memory) surfaced mutexes above goroutines. The bi-encoder + RRF first stage had ranked these below the answer. The reranker, told to reorder everything, undid that work.

That is the diagnosis, and it points at the fix. The problem was not the cross-encoder's signal; it was letting that signal overwrite a first stage that was already strong. So we changed two things and measured again. Blend instead of replace: the final score is `(1-w)·rrf + w·ce`, both normalized, so the cross-encoder nudges the ranking instead of dictating it (`w=0.5`). And window discipline: rerank only each note's best-matching passage, always under the 512-token window, never a truncated whole note, the exact input that produced the 0.388 collapse.

In that form it wins. nDCG 0.926 → 0.949, MRR 0.950 → 0.969, with the largest gain exactly where the first stage is weakest: cross-language en→ru retrieval, 0.867 → 0.924 (+0.057). The mirror image of the replace-mode result.

So the lesson is narrower and more useful than "rerankers don't help." A reranker that reorders everything degrades a strong first stage on a small, topically-adjacent corpus. The same model, blended with the first-stage score and fed only inputs that fit its window, recovers the upside the first stage misses, the hard cross-language cases, without trampling what it already had right. It stays behind a flag, default off: it needs a CPU cross-encoder sidecar and adds latency, so it is opt-in infrastructure, not a free default. No longer a negative result, but a shipped option that earns its keep where the sidecar runs.

---

## F4: a chunk format that carries its section context

Two chunking fixes in `internal/mdchunk`.

**Token-aware sizing.** The chunk size target was 2000 characters. For Cyrillic text, 2000 characters is roughly 1000 tokens — nearly double bge-m3's 512-token window. The tail of every long Russian chunk was silently truncated server-side and never embedded. We switched to token estimation (~450 target), which fits inside the window with a safety margin.

**Heading breadcrumb.** Each chunk now starts with its section path: `{title} > {h1} > {h2}\n\n{body}`. Before, a deep chunk from a long document embedded with no context about where in the document it came from. Now the embedding carries that context, and the breadcrumb has a second use: it aligns with the search-result TOC, so a fuzzy vector hit can be drilled into the exact section via `note_html(toc_path=[...])`.

**Result:** on the short-note corpus, the effect is close to zero — those notes are single-chunk, so the breadcrumb adds information but does not change chunk boundaries. On the long-document track (6 Marcus Aurelius chapters, 6–10 chunks each): nDCG@10 0.931 → 0.954 (+0.023), en→en retrieval 0.816 → 0.908 (+0.092). F4 requires a full re-embed.

---

## F5: two ideas, one shipped, one reverted

**Dot-product similarity — shipped.** The embedding server returns L2-normalized vectors (`normalize_embeddings=True`). For unit vectors, cosine similarity is algebraically identical to the dot product. The original code recomputed both magnitudes and performed a division per chunk pair. We replaced it with a plain dot product: one multiply-add loop per chunk instead of three passes.

Result: nDCG/Recall/MRR identical to F4 on both corpora — exactly expected for an algebraically equivalent change. Pure latency reduction, no quality cost.

**AND→OR BM25 fallback — tried, measured worse, reverted.** The BM25 query requires all terms to appear (AND). The idea: when AND returns few hits, retry with OR and merge — "free recall recovery."

The actual numbers: short corpus nDCG@10 0.926 → 0.674 (−0.252), Recall@10 0.992 → 0.675. Long-doc track: nDCG 0.954 → 0.766 (−0.188). Tellingly, MRR barely moved (0.950 → 0.956) — the top-1 result stayed right, but everything below it rotted.

The reason is mechanical. RRF fuses by rank, not by score. The golden set is natural-language and semantic, so the AND query almost never matches all terms — the OR fallback fired on nearly every query and flooded the BM25 lane with up to 20 loose matches (documents sharing one common word). RRF assigned them ranks 1/(60+1), 1/(60+2), and so on — strong signals. Those low-precision matches outranked the relevant documents that the vector lane had correctly found. MRR survived because the vector lane still dominates the very top hit; recall collapsed because everything below it was replaced by noise.

In a hybrid system where the vector lane already carries recall, the BM25 lane's job is precision. Keeping it quiet on semantic queries — returning nothing rather than 20 loose matches — turns out to be load-bearing. A change that looks like free recall recovery is actually corrosive. We only know because we measured before merging; the AND→OR commit would have silently halved retrieval quality otherwise.

---

## What we would do differently

The benchmark was built after the pipeline, not before. We wrote the eval harness, committed a corpus and 60 hand-labeled queries, and only then ran the baseline. The reranker and the AND→OR fallback would have failed to meet a quality gate and never landed; instead they had to be reverted after the fact.

The other lesson is specific to hybrid search: the two lanes have different jobs. Vector carries recall; BM25 carries precision for exact terms and rare vocabulary. Tuning them independently — and measuring both together — catches the interactions that single-lane intuition misses.
