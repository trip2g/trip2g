# DRAFT — "What we walked away from, and why"

Staging draft for a new Philosophy page (`docs/en/thoughts/paths-not-taken.md` if approved).
Every claim below is verifiable in this repo: commit hashes and doc paths are listed inline as
`[evidence: ...]` markers — **strip the markers before publishing**, they are for review only.
RU adaptation notes at the bottom.

Proposed frontmatter:

```yaml
---
title: "What we walked away from, and why"
free: true
lang_redirect: "[[ru/thoughts/paths-not-taken]]"
---
```

---

Most product pages list what got built. This one lists what got built and then deleted, or evaluated and refused. Not because the ideas were stupid — most of them are textbook recommendations — but because we measured them against this system and the numbers said no. A knowledge platform that asks you to trust it with your notes should show you how it makes decisions, including the reversals.

## The reranker that made search worse

Every RAG tutorial says the same thing: after retrieval, add a cross-encoder reranker. It's the "biggest quality lever." So we built one — `bge-reranker-v2-m3` in a sidecar, re-scoring the fused top results before returning them.

It made search worse. Measured, twice. On 512-character passages, ranking quality (nDCG) dropped from 0.92 to 0.89. On full-note passages it collapsed to 0.39, because the notes exceeded the reranker's token window and all looked identical to it. The failure mode was instructive: the cross-encoder rewards surface word overlap, so a query about slow dough fermentation surfaced the sauerkraut note above the sourdough note — sauerkraut simply has more fermentation vocabulary. The first stage, hybrid keyword-plus-vector fusion, had already ranked those correctly. The reranker undid its work.

We shipped it off by default, then deleted it entirely, along with its Python sidecar — one less process to run, one less model to download. The negative result stayed in the docs. The lesson isn't "rerankers are bad." It's that a second stage is only a lever when the first stage is weak, and ours wasn't. Measure before shipping, even when the textbook says you don't have to.

[evidence: built `00a993f7`, removed `adf84ce7` (PR #78), writeup `docs/dev/reranker.md`, benchmark tables in `docs/en/thoughts/search-benchmark.md`]

## The recall trick that poisoned recall

Same benchmark, same week. Our keyword lane requires every query term to match. The obvious improvement: when that returns few hits, retry with any-term-matches and merge — free recall recovery.

It cut ranking quality by a quarter (nDCG 0.93 → 0.67). The mechanism: fusion ranks by position, not score, so twenty loose one-word matches entered the merge with strong rank signals and pushed out the documents the vector lane had correctly found. Tellingly, the top-1 result survived — only everything below it rotted, which is exactly the kind of damage you never notice without a benchmark. Reverted byte-for-byte the same day.

In a hybrid system the two lanes have different jobs. Vector carries recall; keyword carries precision. A keyword lane that stays quiet on semantic queries — returns nothing rather than noise — turns out to be load-bearing.

[evidence: reverted `c0eaecdb`, recorded `14056c6c`, `docs/dev/search_refactoring.md` F5✗ rows]

## The vector database we didn't need

At some point every project with embeddings gets told to adopt a real vector database. We evaluated Qdrant against our in-memory brute-force scan.

The results were byte-for-byte identical: same top-1, same top-5, on every test query. Qdrant solves a speed problem we don't have at this scale — the embeddings are already in memory, and the scan is not the bottleneck. What Qdrant would add is a container to run, a sync pipeline to maintain, and a new way for search to be down. Each one a failure point in a product whose pitch is "one process on one file."

The real quality lever was the embedding model, not the storage. We upgraded the model and kept the dumb scan. (Fun footnote: a Qdrant node does live in our public hub today — as an external federation peer indexing Telegram channels, proving the protocol isn't trip2g-only. Right tool, right layer.)

[evidence: `docs/dev/why_not_qdrant.md`]

## The kanban board that moved out twice

We wanted a kanban board where a card is a line in a note. First build: a component in our own frontend framework, inside the main repo. Then we rebuilt it as a self-contained React template. Then we deleted that from the repo too and shipped it as a standalone repository you install with one curl command.

Each move was the same realization at a different depth: the board is a *template*, not a platform feature. It renders a note; the platform's job ends at "serve the note and accept the edit." Keeping it in-core meant every board improvement was a platform release. As a standalone template it versions, installs, and forks on its own — and it proved the layout system was strong enough to host a real app without the core knowing about it.

The general rule we took from it: when a feature can live on top of the substrate, it must. The substrate stays small; the doors multiply.

[evidence: $mol attempt `5e4629c4`, React template `48ede2c6`, in-repo copy removed `7f118043` (PR #78), lives at `trip2g/kanban_template`]

## The premium bot that cost $2,000 to say hello

Publishing notes to Telegram with custom emoji looked simple: the Bot API supports it. Then we read the fine print. A bot may only send custom emoji if it's connected to a Fragment username — which costs on the order of 1,000 TON, roughly $2,000, before the first message.

The path we took instead: publish through a real user account (MTProto) with an ordinary Telegram Premium subscription, about $5 a month, paid by whoever owns the channel. Same custom emoji, plus longer media captions the Bot API also caps. The free bot remains the default; the userbot is the opt-in premium path.

This one wasn't a measurement, it was arithmetic. Some walls aren't technical.

[evidence: `docs/dev/telegram_bot_vs_userbot.md` cost table + verdict]

## Smaller goodbyes

Not every reversal deserves a chapter. A one-liner each, because the pattern is the point:

- A load-testing harness (k6), removed once the benchmarks it existed for were done and documented — a tool's job can end. [evidence: `e5c25d1c`]
- A Python layout-preview script, replaced by a dependency-free Node watcher — fewer runtimes to install for one preview loop. [evidence: `d0fe4179`, `docs/dev/preview_tool.md`]
- A convention swap: `sql.Null*` wrappers replaced by plain Go pointers across the whole codebase — deleting an entire conversion layer beat keeping a habit. [evidence: `docs/dev/null_string_refactoring.md`]
- Two embedded rich-text editors, removed as legacy bundles — the platform's contract is markdown files; owning a WYSIWYG editor was never the job. [evidence: `91ab6dae`]

## What the pile of dead code says

Reading back over the list, three rules did all the killing:

**Measure against your own system, not the literature.** The reranker and the recall fallback are both best practices. Both made this system worse, in ways only a benchmark could see.

**Don't add a moving part to solve a problem you don't have.** Qdrant, the Python sidecar, the extra runtimes — every one deleted for the same reason the product runs as one binary on one file.

**Keep the core smaller than you want to.** The kanban board, the editors, the preview tool — everything that could live on top of the substrate was pushed on top of it, even when we'd already built it inside.

None of these were walked away from lightly; most had working code and passing tests. That's the part worth writing down. The expensive discipline isn't building — it's deleting something that works because the numbers, the arithmetic, or the architecture said it doesn't belong.

---

## RU adaptation notes (do not translate word-for-word)

- Title: «От чего мы отказались — и почему» (not a calque of "paths not taken"; «непройденные пути» sounds like Frost translated badly).
- The RU version should be a native retelling in the register of the existing RU thoughts pages: shorter sentences, «мы померили — стало хуже» directness. Ильяхов rules apply: lead each section with the verdict.
- Section titles want RU-native reworking, e.g. «Реранкер, который ухудшил поиск», «Векторная база, которая решала не нашу проблему», «Бот за $2000 до первого сообщения», «Канбан, который дважды переехал».
- Keep the numbers identical to EN (the vibecoding pair already drifted on figures; don't repeat that).
- «Smaller goodbyes» → «Мелкие расставания» or fold into the closing section.
- Cross-links: `[[ru/thoughts/search-benchmark]]`, `[[ru/thoughts/blaming-sqlite-for-my-own-bug]]` (one-process argument), `[[ru/thoughts/contentless-cms]]` (editor point).

## Placement note

If published, add to `_sidebar.md` under **Philosophy** (it pairs naturally with Dogfooding) and to both `_index.md` files. It also gives the landing's philosophy block a fourth, differentiating link: nobody else's landing links to a page of their own deleted features.
