---
title: "Good MCP instructions make a cheap model find answers faster"
free: true
lang_redirect: "[[ru/thoughts/mcp-instructions-make-cheap-models-faster]]"
---

*What this is: a measured A/B. I gave Claude Haiku an MCP connection to the trip2g docs and asked it eight questions, once with a short instructions note in its system prompt and once without. Same model, same tools, same questions. The note made Haiku search 26% faster and dump whole notes 41% less often, at the same accuracy. Read it if you run a knowledge base over MCP and wonder whether the instructions note earns its keep.*

The short answer: a good instructions note does not make a cheap model smarter. It makes it stop being wasteful. On my docs base, adding one 40-line note to Haiku's system prompt cut its tool calls by a quarter and its whole-note dumps by nearly half, while it found exactly as many right answers. The savings are real and they compound on every query.

## The payoff, measured

I ran `claude-haiku-4.5` against the live trip2g MCP endpoint. Seven retrieval questions, three times each, in two variants: with the docs-base instructions note in the system prompt, and with a bare "you have tools, use them" line instead. Real tool execution against the real docs.

| Metric | With note | Without note | Delta |
|--------|-----------|--------------|-------|
| Avg tool calls | 4.00 | 5.43 | 26% fewer |
| Avg tokens | 17,265 | 18,696 | 8% fewer |
| Avg whole-note dumps | 0.95 | 1.62 | 41% fewer |
| Section-read rate | 71% | 29% | +43 pp |
| Right answer found | 100% | 100% | tie |

The headline is the last row read together with the rest. Accuracy is a tie. The note buys efficiency, not correctness: fewer round-trips to the same answer, and far fewer of those round-trips ending in a full-note dump.

That last metric is the one I care about most. A "dump" is when the model reads a whole note instead of the one section it needed. On this base a section is a few hundred tokens and a whole note can be fifteen thousand characters. Without the note, Haiku dumped a full note 1.6 times per question on average. With it, less than once. The note teaches the model to read the section, and it listens.

## How the note earns it

The note teaches one loop, and the loop is the whole trick:

1. `search(query)` returns pointers, not documents. Each hit carries a `toc_path`: a breadcrumb to the exact section that matched.
2. `note_html(pid=N, toc_path=[...])` reads only that section. A few hundred tokens instead of the whole file.
3. `expand(pid=N)` walks the table of contents one level at a time when the pointer is soft, so the model can navigate to the right section instead of guessing.
4. Read the whole note only when `expand` shows it has no sections worth drilling into.

A bare model, given the same tools, tends to `search` and then read the whole top result to be safe. That works, and it is expensive. The note flips the default: pull the section first, fall back to the whole note last. On the federation question, the note-guided model answered in 2.7 calls and never dumped a note; the bare model took 5.3 calls and dumped one almost every time.

The other half of the note is guardrails written from how the tools actually fail. On this base a wrong `toc_path` does not raise an error. It silently returns the whole note. So the note tells the model: if a section read comes back much longer than a section, your pointer missed, call `expand` and read the real heading, do not retry blind. That one instruction turns a silent 15,000-character mistake into a cheap recovery.

## The honest parts

The note is not magic, and pretending otherwise would waste your time.

The bare model was not helpless. On its own it did the section read about a third of the time. Haiku knows `toc_path` exists once it sees the schema; the note just makes precise reading the habit instead of the exception.

Accuracy did not move. Both variants found the right note on every retrieval question. If your search is weak, an instructions note will not save you. Mine is a hybrid keyword-plus-vector search that already lands the right note reliably, so the note had nothing to fix there and everything to fix in how the note got read.

One question cost more with the note than without. On "how do I publish a vault", the guided model kept drilling for the exact section (7+ calls) while the bare model stopped at a coarser answer sooner. Biasing toward precise section reads occasionally costs an extra call. It is a real trade, small, and worth it for the dump reduction everywhere else.

And one question was a wash by design: "what does `expand` do" got answered from memory by both variants, because a cheap model already knows what "expand" means in the abstract. A generic-knowledge question is not a retrieval test, so I scored it separately.

## Method, so you can redo it

Model: `claude-haiku-4.5` over OpenRouter, tools set to the live MCP `search`, `note_html`, `expand`, `similar`. Variant A's system prompt was the note at `docs/_mcp_initialize.md` with its frontmatter stripped, exactly what the MCP `initialize` response serves. Variant B's was a single generic line. Eight questions spread across federation, publishing, access control, the token-economy tools, and Telegram; three runs each; forty-eight runs total. A "dump" is a `note_html` result over 6,000 characters. Total spend was about $0.92, comfortably under my $2 cap.

The full table, per-question breakdown, raw runs, and the harness are in the dev notes: `2026-07-02_mcp_haiku_ab.md` and `2026-07-02_mcp_haiku_ab_results.json`.

## Write your own

If you run a trip2g base, give it its own `initialize` note. Lead with what the base is, teach the search then `toc_path` then `expand` loop, show one worked example with a real note id, and write the guardrails from your tools' real failure paths. Keep it tight; a cheap model degrades under a wall of text. The full recipe, with the shadowing gotchas and a live validation checklist, is in the dev note `docs/dev/mcp_instructions_guide.md` (ships with the instructions note itself).

The upside scales the wrong way for skipping it: the note costs you an afternoon once, and every agent that ever connects pays 26% less to find its answers.
