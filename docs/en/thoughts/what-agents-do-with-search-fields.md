---
title: "What agents actually do with search fields"
description: "We shipped three optional query fields for MCP search and measured them in the hands of real Claude agents: which fields they fill, whether OR helps the short one, whether telling them to fill more helps, and whether showing the size of a note changes how they read it."
free: true
lang: en
lang_redirect: "[[ru/thoughts/what-agents-do-with-search-fields]]"
---

*What this is: in September [[en/thoughts/query-expansion-by-the-agent|we measured]] three optional fields an agent can pass with a search, `short_query`, `rephrased_query` and `expected_answer`, but the fields were written by a strong model in a script. In October we shipped them and watched what Claude Haiku and Claude Sonnet do with them through Claude Code. Four experiments, about 1,500 agent runs. The short version: the fields work when someone fills them well, agents mostly don't, pushing them to fill more makes things worse, and one plain number in the search result changed behaviour more than any wording.*

## The setup

Three bases:

- a Russian product documentation base, 100 notes, 42 questions with the right page labelled;
- the Russian base set of the [[en/thoughts/search-benchmark-obsidian-help|Obsidian Help benchmark]], 58 questions;
- a synthetic English manual for a made-up product: 12 long notes (4,000–10,000 words, 15–35 sections each) and 40 short ones, 56 questions with one checkable answer each, most of them deep inside a long note.

The agents ran through the Claude Code CLI with only the base's MCP server connected, vector search on (bge-m3). For retrieval we recorded the agent's first `search` call and replayed it against the server with fields switched on and off, scoring MRR@10 and nDCG@5 against the labels. Each comparison is paired, with a bootstrap 95% interval; on 40–60 questions a difference needs to be about ±0.1 MRR before it means anything.

## How agents fill the fields

First `search` call on the documentation base:

| Writer | `short_query` | `rephrased_query` | `expected_answer` | rewrote `query` |
|---|---|---|---|---|
| Haiku | 89% | 86% | 14% | 72% |
| Sonnet | 91% | 53% | 0% | 15% |
| strong model, question and descriptions only | 100% | 100% | 100% | — |

Sonnet never wrote `expected_answer`. Haiku wrote it one time in seven and rewrote `query` almost three times in four, although the description asks it not to.

Effect on MRR@10 of adding each field to the query (95% interval):

| Writer | + `short_query` | + `rephrased_query` | + `expected_answer` | all three |
|---|---|---|---|---|
| Haiku | +0.01 | +0.05 | +0.01 | +0.05 |
| Sonnet | +0.09 | 0.00 | — | +0.04 |
| strong model | +0.10 | +0.04 | **+0.12** [+0.03, +0.21] | **+0.14** [+0.01, +0.27] |

Only the strong writer's `expected_answer` and the three fields together clear noise. The mechanism works on this base about as well as it did on Obsidian Help in September (+0.10 there). What limits it is the agent: the field that helps most is the one real agents skip.

`rephrased_query` helped nobody beyond noise and, for Sonnet on top of `short_query`, made 4 questions worse for 1 better. We kept it: qmd, whose design the fields follow, has the same field, and 42 questions prove "useless" no better than they prove "useful".

## OR for `short_query`

Full-text search in trip2g requires every query word. `short_query` is two or three words, so OR there looked safe: the long `query` keeps AND for precision, the short field gets OR for recall. MRR@10 with `query` + `short_query`:

| Writer | AND | OR | AND, OR if empty |
|---|---|---|---|
| Haiku | 0.521 | 0.595 (+0.07 [+0.01, +0.16]) | 0.537 |
| Sonnet | 0.654 | 0.559 (−0.09) | 0.630 |
| strong model | 0.663 | 0.593 (−0.07) | 0.656 |

OR rescues Haiku's weaker word choices and drowns everyone else's precise ones. On this base the word for the product's main entity is on almost every page, so `short_query: "copy agent"` under OR returns a near-random list, and rank fusion gives it the same weight as the precise lists. This is [[en/thoughts/search-benchmark|July's result]] again, one field down. The fallback barely fires: with two or three words, AND is rarely empty. We kept AND.

## Telling agents to fill the fields

If the useful field goes unused, ask for it. Five wordings, on the documentation base and Obsidian Help, Haiku and Sonnet, 1,000 runs:

- V0: the shipped text ("Optional. …");
- V1: `expected_answer` marked "Recommended", with "you do not need to know the answer; it is never shown to anyone";
- V2: "You SHOULD fill this whenever you understand the question" on all three, plus "fill all three on every search" in the tool description;
- V3: V2 plus "if you are not sure whether the answer is yes or no, do not assert either";
- V4: only the sentence in the tool description.

Filling all three fields rose by 44–51 percentage points for V2, V3 and V4; Sonnet's `expected_answer` went from 0% to 100% on the documentation base. One sentence in the tool description (V4) did as much as rewriting every field.

Retrieval did not move. The pooled MRR change against V0 was between +0.001 and +0.027, with a 95% interval of about ±0.045, in every variant. And `expected_answer` started to lie: on questions whose true answer is "no", the filled field asserted "yes" in 30–40% of calls ("open the agent's card and choose Clone" for a feature that does not exist). V3's guard sentence did not stop it. Where a wrong guess met the right page, it pushed that page out of the top ten.

We left the descriptions as shipped. An agent pushed to guess guesses, and a guess stated as fact is worse for search than no guess.

## Showing how long a note is

The search result used to say nothing about the note's size, so an agent could not tell a 9,000-word manual from a paragraph before opening it. We added two numbers to every result: `size_tokens`, the same estimate the chunker uses, and `sections`, the headings `expand` can walk. Then the synthetic manual, three variants: S0 without the numbers, S1 with them, S2 with them plus a sentence in the tool description ("for a note over about 2,000 tokens with several sections, read the matched section, or call `expand` first").

Long-note questions, two runs each:

| | correct | read a long note whole | read by section | tool output received |
|---|---|---|---|---|
| Haiku S0 | 90% | 0.76 per question | 0.24 | 30k chars |
| Haiku S1 | 83% | 0.36 | 0.68 | 16k |
| Haiku S2 | 79% | 0.12 | 1.02 | 12k |
| Sonnet S0 | 88% | 0.49 | 0.43 | 20k |
| Sonnet S1 | 90% | 0.04 | 0.92 | 6k |
| Sonnet S2 | 88% | 0.02 | 1.20 | 7k |

The two numbers alone cut whole-note reads and the volume of tool output by half to two thirds for both models, beyond noise. Sonnet's runs got about half as expensive at the same accuracy. The extra sentence added nothing for Sonnet and cost Haiku about ten points of accuracy: told to read by section, it read the wrong section. Short-note questions did not change.

We shipped the numbers and not the sentence.

## What this says

- **The agent is the bottleneck, not the server.** Each field works when it is filled well. Real agents fill the cheap fields, skip the useful one, and rewrite the query they were asked to keep.
- **Wording moves behaviour, not results.** One sentence changed fill rates by fifty points and retrieval by nothing measurable, and it made the agent assert things it did not know.
- **A fact beats an instruction.** `size_tokens` is a number, not advice, and both models used it correctly without being told what to do with it. The instruction on top made the weaker model worse.
- **Measure with the agent that will use it.** The September numbers were real for a strong model filling fields in a script. In the hands of the agents a user runs, most of that gain is not there.

The earlier part of this work, the defects these runs were built on top of, is in [[en/thoughts/benchmarks-that-skipped-the-client|Three months of benchmarks, and the model never read a note]].
