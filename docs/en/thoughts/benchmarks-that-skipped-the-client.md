---
title: "Three months of benchmarks, and the model never read a note"
description: "We ran a product documentation base through Claude Code the way a user does and found five defects that every earlier benchmark had walked past: a note body that never reached the model, instructions lost on the new protocol, a 25-second connect, full-text search that answered most queries with nothing, and query fields that were measured but never shipped."
free: true
lang: en
lang_redirect: "[[ru/thoughts/benchmarks-that-skipped-the-client]]"
---

*What this is: in October we asked Claude models questions about a documentation base the way its users do, through Claude Code, against the base as it is deployed. Five things were broken. Each had been within reach of a benchmark we had already published. This is what was broken, why none of those benchmarks saw it, and what changed after the fixes.*

A published number stays true for exactly the path it measured. We wrote that in [[en/thoughts/load-recheck-four-months-later|the load re-check]] about speed. It turns out to hold for answers too.

## The run

The base is product documentation in Russian, about 85 notes, served over MCP. The client was the Claude Code CLI with only this server connected, on its `haiku` and `sonnet` models. Eleven questions: four answered plainly in the base, three the base says nothing about (the right answer is "not described", not "no"), one where the base says no outright, one message with three questions, one technical question, and one coding task that has nothing to do with the base. Each question twice per variant, every answer scored blind by a separate Opus judge: grounded, honest about absence, qualifiers kept, short first, plain language, every sub-question answered.

The question we started with was small: how should a base hand a model its answering rules? Answering it meant reading what the models actually did, call by call. That is where the defects were.

## What was broken

**The model never read a note.** `note_html` returned the note in its text block and, since attachments were added the day before, a structured result holding only the path and the attachment list. Claude Code passes a tool's structured result to the model instead of its text block. So in Claude Code a model that opened a note received `{"note_path": "AGENTS.md", "assets": []}`. In 0 of 30 calls did the body arrive. Models told users the note was empty and answered from search snippets. Every tool test passed, because each test checked the text block.

**Instructions vanished on the new protocol.** Claude Code now negotiates MCP 2026-07-28, where `server/discover` replaces `initialize`. trip2g resolved the server name and instructions only for `initialize`, so a current client got an empty name and no instructions at all. [[en/thoughts/do-agents-need-instructions|Our September essay on instructions]] ran through Claude Code too, before the client changed protocol. Its numbers were right for the client it measured.

**Every connection took 25 seconds and then dropped.** The server advertises that its tool list can change, so the client opens `subscriptions/listen` right away. The endpoint buffered the whole response before writing it, and a listen never ends, so the request hung until the client gave up, 25 seconds later. The client then cancelled it with a request that lacked the `Mcp-Method` header, the server refused that, and the client logged a dropped connection. Tools still worked after a reconnect, so nothing looked broken from outside.

**Full-text search answered most queries with nothing.** trip2g's text lane requires every query word to match. [[en/thoughts/search-benchmark|In July we tried relaxing that]] to OR and reverted it: with vector search on, the vector lane carries recall and the AND lane's job is precision, and OR flooded it with loose matches. That is still true. But this base was deployed without an embedder, so the text lane was all it had. Models write long queries, and 58% of the 344 distinct queries they sent came back empty. The right page made the top six for 34% of the questions the base does answer.

**The query fields were never shipped.** [[en/thoughts/query-expansion-by-the-agent|Our September essay]] measured three optional fields an agent can pass with a search, `short_query`, `rephrased_query` and `expected_answer`, and found +0.10 nDCG@5 on the Russian set. The measurement fused the extra lists in a script outside the server. The server's `search` tool still had `query`, `limit` and `detail_limit`. The essay reads as if the fields exist. They did not.

## Before and after

Mean score out of 12, blind judge, 22 answers per cell. "Rules" is the short initialize described below plus answering rules in a note the model opens first.

| | haiku | sonnet |
|---|---|---|
| no rules | 2.9 | 4.4 |
| all rules in initialize, before the fixes | 6.8 | 10.6 |
| short initialize + rules note, before the fixes | 5.2 | 8.2 |
| short initialize + rules note, after the fixes | **10.3** | **11.5** |

| | before | after |
|---|---|---|
| connect | 25 s, then a dropped connection | 39 ms median |
| rules note body reaching the model | 0 of 30 | every load |
| model reads "agent" as itself (sonnet) | 2 of 20 | 0 of 20 |

On the empty searches we tried one experiment: AND, and OR only when AND finds nothing. Empty searches fell from 59% to 9%, the scores did not move, and on the three questions the base cannot answer no model invented anything from the loose matches; they stopped re-querying, that was all. We did not ship it. The fix for a base without vectors is to give it vectors, which is what the July benchmark measured in the first place. The query fields are being built now.

## What the instructions question answered

The question we came for has an answer too, and it is short. A client puts a server's `initialize` instructions into the system prompt of the whole session, so they apply to every turn, including the coding task. Keep them to what must hold every turn: what the base is about, what a word means here when the model could take it for itself ("an agent here is the AI employee the customer configures, not you"), that facts about the subject are not in the model's training data so it searches even when sure, and where the rules are. Put tone, length and checklists in a note the model opens with `note_html`. [[en/user/mcp|The MCP guide]] has the details. A persona line ("answer as an editor would, don't copy him") changed nothing measurable, and no rules leaked into the coding answer in any variant.

## Why five defects lived through four benchmarks

Each benchmark measured one layer and stood in for the rest.

- The July search benchmark measured the engine through a script, with vectors on. The deployment it was meant to describe ran without them.
- The September query-fields benchmark measured the idea through a script that rebuilt the lanes outside the server. The server was never changed.
- The September instructions benchmark went through the real client, before the client moved to a new protocol.
- Tool tests checked what the server sent, not what the client handed the model.

None of them ran the deployed configuration through the current client with a model reading the replies. [[en/thoughts/small-mcp-errors|In September we wrote]] that you should read an assistant's steps, not its answer. The empty searches were in every transcript of every run since. Nobody was reading transcripts, because nothing scheduled the reading.

## What changes

- The benchmark that found these runs through the client a user runs, against the base as deployed, and it reruns when the client updates, not only when we change the server.
- Two numbers go next to every score, because they would have caught four of the five on day one: the share of searches that came back empty, and the share of note reads whose body reached the model.
- An essay that measures a proposal says so, and says when it ships.

Still open: whether the base searches well with vectors on, which we measure after deploy; the query fields; and `subscriptions/listen`, which the server now refuses while still advertising a changing tool list, so the client retries a few times per session.
