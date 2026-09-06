---
title: "Small MCP errors, and who pays for them"
free: true
lang: en
lang_redirect: "[[ru/thoughts/small-mcp-errors]]"
---

*One afternoon in September I replayed three questions against the public trip2g hub with a cheap model, read every step, and found six defects in my own MCP server that nobody had reported. A follow-up to [[en/thoughts/mcp-instructions-make-cheap-models-faster|the instructions A/B]].*

An MCP server can be wrong in ways a code review does not catch. An error that repeats the caller's mistake back to it. A hint where the answer should be. A fact printed where the model never looks. Each is a one-line fix with an asymmetric cost.

A strong model digests the defect. It guesses around it, pays two or three extra calls, and answers. The bill goes up by a few cents and nobody notices, because almost nobody reads an agent log step by step. A small model hits the same wall three times, runs out of budget, and answers from memory. Only the second model makes the defect visible.

## The setup

The public trip2g hub connects several knowledge bases; one of them, the philosophers hub, is itself a hub over 21 corpora. Marcus Aurelius hangs directly under the top hub, Confucius one level deeper. I took the loop from [[search_visualizer|the search visualizer]], ported it to a headless script, `scripts/mcp_search_logger.py`, and ran gpt-5.4-mini with about ten tool calls per question. Two questions about the philosophers, one about setting up Claude Code memory from the trip2g docs.

## The wall that repeats your own words

On 2 September none of the three was answered from the sources. Here is where the first one died, right after the model found the Marcus Aurelius card:

```
model:  I found the Marcus hub, so I'll descend into the federated Marcus base
→ federated_search(kb_id="trip2g/markavrelii", …)
← Federation is not configured for kb_id "trip2g/markavrelii".
  … address this base as "trip2g/markavrelii".
```

The model sent an address, and the server told it to use that same address. A template with an unfilled placeholder, echoing the input. The model trusted the server, tried again, tried a third guess, and ran out of budget. The right address was in the reply all along, but only in the structured part, and most clients, mine included, show the model only the text. A fact the model cannot see does not exist.

The other questions hit the same family. The model asked the Claude Code tutorial for three different sections and got the same chunk three times, so the answer had no setup command. Expanding a section with no subsections returned a hint to call another tool instead of the section. None of this raised an error or failed a test.

## A better error message made it worse

I fixed all of it the same day, in [PR #339](https://github.com/trip2g/trip2g/pull/339). Search results say in plain text where a card points. A named section wins. The error names the part of the address it does not know and lists the bases it does. Within the hour the first question was answered, Book 5 of the Meditations quoted with a source.

Then the peers got the fix, and the first question broke again. The philosophers hub now listed its 21 bases in the error, and the model stayed inside that list: it went to Epictetus as the nearest Stoic and never looked back at the top hub where Marcus lives. The morning's vaguer message, "search the hub for it", had worked better by accident. A more helpful error had shrunk the model's world to one hub. [PR #341](https://github.com/trip2g/trip2g/pull/341) made the error name both levels. On 6 September all three questions passed on every run: same model, same budget, about a cent per run before and after. The cost did not move. What moved was whether the budget went into the answer or into the wall.

## Who pays

Here I have to be careful. I did not run a strong model against the broken build. That a bigger model would have digested these defects quietly is an observation from running agents on bigger models in production, plus an inference from the traces: at the echoing error a bigger model would likely have tried the short address and paid two or three calls. A guess with a mechanism, not a measurement. The measurement is cheap: replay the September traces up to the first error on a bigger model and count the recovery steps. I have not done it.

The other half I did measure: on the cheap model these defects were the difference between answering and not answering. A strong model turns a wrong server into a slightly higher bill, and a bill is not a bug report. A cheap model turns it into a wrong answer, and a wrong answer is one, if someone reads the log.

## Do this with your own server

Take the questions your agent answers most, run them through a cheap model with a tool budget, and read the trace step by step. Not the answer, the trace. Every place where the server's reply cost the model one more call is a defect, error or not. The script is `scripts/mcp_search_logger.py`; the browser version is [[search_visualizer|the visualizer]]. Ten calls and a cent per run found six.
