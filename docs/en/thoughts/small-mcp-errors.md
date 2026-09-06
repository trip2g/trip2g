---
title: "Small MCP errors, and who pays for them"
free: true
lang: en
lang_redirect: "[[ru/thoughts/small-mcp-errors]]"
---

*What this is: one afternoon in September. I replayed three questions against the public trip2g hub with a cheap model, read every tool call, and found six defects in my own MCP server that nobody had reported. It follows [[en/thoughts/mcp-instructions-make-cheap-models-faster|the instructions A/B]]: that piece measured how a good manual helps a cheap model. This one is about what a slightly wrong server does to it.*

An MCP server can be wrong in ways a code review does not catch. An error message that repeats the caller's input back to it. A field present in the structured payload but missing from the text. A tie-break between two arguments that goes the wrong way. Each is a one-line fix with an asymmetric cost, and the asymmetry is the subject here.

A strong model digests the defect. It guesses around it, retries in another form, pays two or three extra calls, and answers. The bill goes up by a few cents and nobody sees it, because almost nobody reads an agent log call by call. A small model cannot digest it. It hits the same wall three times, runs out of budget, and answers from memory. Same defect; only the second model makes it visible.

## The setup

The public hub at `https://trip2g.com/_system/mcp` federates several bases. One of them, philosophers.2pub.me, is itself a hub over 21 philosopher corpora. So a question about Marcus Aurelius and Confucius needs two hops in two directions: the Marcus base hangs directly under the top hub as `markavrelii`, and Confucius sits under the philosophers hub as `philosophers/confucius`.

I took the loop from [[search_visualizer|the search visualizer]], the page that animates a model's walk in the browser, and ported it to a headless script, `scripts/mcp_search_logger.py`. It runs `openai/gpt-5.4-mini` in task mode with a budget of 10 to 12 tool calls and writes every thought, call, server text and final answer to JSON. Three questions:

1. What do Marcus Aurelius and Confucius say about getting out of bed in the morning?
2. Where do Epictetus and Wattles disagree about what is in our control?
3. How do I give Claude Code a persistent memory with trip2g?

The first two need federation. The third is local docs and needs reading specific sections.

## Before: the wall that repeats your own words

Here is the first question on 2 September, trimmed from the trace. Arrows out are the model's calls, arrows in are the server's text, and the lines between are the model's own thoughts.

```
→ federated_search(kb_id="philosophers/marcus-aurelius", query="getting out of bed in the morning rise from bed morning")
← Federation is not configured for kb_id "philosophers/marcus-aurelius".
  Bases reached through a hub are addressed  /  — address this base as "philosophers/marcus-aurelius".
model: The Marcus Aurelius kb id needs the hub/base form, so I'll search the Marcus base via the philosophers hub
→ federated_search(kb_id="philosophers/marcus-aurelius", query="morning bed rise laziness duty")
← (the same message)
→ search(query="Marcus Aurelius getting out of bed morning rise from bed")
← 1. Marcus Aurelius — Meditations
     en/hub/markavrelii.md
     https://trip2g.com/en/hub/markavrelii
     match_id: p1063:c0
model: I found the Marcus hub, so I'll descend into the federated Marcus base
→ federated_search(kb_id="trip2g/markavrelii", query="bed morning rise sleep laziness duty")
← Federation is not configured for kb_id "trip2g/markavrelii".
  Bases reached through a hub are addressed  /  — address this base as "trip2g/markavrelii".
```

Read the last error twice. The model sent `trip2g/markavrelii`, and the server told it to address the base as `trip2g/markavrelii`. A template with an unfilled placeholder, echoing the input. The model trusted the server and tried again.

The gap in "addressed  /  " is a second bug, and that one is mine. The server actually wrote `<hub>/<base>`. My logging script stripped HTML tags with a blanket regex and ate the placeholder. For an hour a defect in the observer looked like a defect in the server.

Then the model opened the hub card with `note_html` and got the body without frontmatter. The card said the Marcus base "is federated into this hub" and stopped. The MCP instructions promised the kb_id was in the card's frontmatter, exactly the part `note_html` did not show. `expand` on the card answered `has no subsections (leaf)`. The model's last thought was that "the Marcus hub points to a leaf-only corpus". It ran out of budget and answered with a Confucius note about ritual and a Stoic theme from memory. Ten calls, $0.013, and the real answer sat in Book 5 of the Meditations, one call away the whole time.

The kb_id was in the response. `search` returned `"kind": "federation_kb", "federation": {"kb_id": "markavrelii"}` in its structured content. But the visualizer, like most clients, feeds the model only the text block, and the text did not print it. A field the model cannot see does not exist.

## The rest of the list

The other two questions found their own defects. On Epictetus and Wattles, the philosophers card on trip2g.com did not list its authors, so a local search for "Wattles" never surfaced the pointer. The model tried `en/hub/markavrelii` as a kb_id, got the circular error twice, and wrote the comparison from its own knowledge. Not one philosopher note opened.

On the Claude Code question the model found the right page, expanded its table of contents, and asked for three sections by `toc_path`:

```
→ note_html(pid=1316, match_id="p1316:c12", toc_path=["Step 3. Register the server in Claude Code"])
← You get the raw tool set (`search`, `expand`, `note_html`, `similar`) instead of the composite adapter tool. ... ## Step 4. Verify ...
→ note_html(pid=1316, match_id="p1316:c12", toc_path=["Step 5. Teach Claude to use it"])
← (the same chunk)
```

The model carried the `match_id` from the search result into every read, and the server let `match_id` win over `toc_path` without a word. Three different sections requested, one chunk returned three times, and the final answer had no registration command in it, because the section with the command was never read.

Six defects, then. The kb_id hidden in structured content. The circular error. The pointer card shown without its pointer. `match_id` beating an explicit `toc_path`. `expand` on a leaf section returning only a nudge to call `note_html`, one wasted call per leaf. And the philosophers card without its authors. Not one raised an exception or failed a test. Each is a few lines.

## After

I fixed all six the same day, in [PR #339](https://github.com/trip2g/trip2g/pull/339). Search text now prints `kind: federation_kb · kb_id: markavrelii → federated_search(kb_id="markavrelii")` under a pointer note. `note_html` on a pointer puts a `federation pointer · kb_id: …` line on top of the body. An explicit `toc_path` wins. And the not-configured error names the segment it does not know and lists what it does know:

```
→ federated_search(kb_id="philosophers/marcus-aurelius", ...)
← Federation is not configured for kb_id "philosophers/marcus-aurelius": hub "philosophers" has no base "marcus-aurelius".
  To find its kb_id, search the hub for it: federated_search(kb_id="philosophers", query="marcus-aurelius") — the pointer card prints the kb_id to use.
...
→ federated_expand(kb_id="confucius", ...)
← Federation is not configured for kb_id "confucius": no connected base on this hub is named "confucius".
  Connected bases: markavrelii, philosophers, peer, minionschool, telegram_demo, nicksenin_journal.
model: I found the hub cards: Marcus Aurelius is under the "markavrelii" base, and Confucius is under the philosophers hub.
→ federated_search(kb_id="markavrelii", query="morning bed get up rise from bed")
← 1. Цепочки ... ## Утреннее намерение Как начинать день. ...
  2. Книга 05 ... https://markavrelii.2pub.me/knigi/kniga_05
```

Same model, same budget, eleven calls, $0.016, and the answer quotes Book 5 with a source: "I am rising for a human task", not to stay warm under the blanket.

Then the peers got the fix too, and the first question regressed. The philosophers hub started listing its 21 bases in the same error, and the model stayed inside that list. It went to Epictetus as the nearest Stoic and never looked back at the top hub where `markavrelii` lives. The vaguer morning message, "search the hub for it", had worked better by accident. The same evening the Claude Code question spent three calls on `expand` for Step 1, Step 2 and Step 3, each answered `has no subsections (leaf). Read it with note_html(toc_path)`, and hit its budget with the commands unread.

[PR #341](https://github.com/trip2g/trip2g/pull/341) made the peer error name both levels, "Bases connected directly to this hub: markavrelii, philosophers, …", and made `expand` on a leaf return the section instead of a nudge. On 6 September all three questions passed, with the memcli command, the token step and the stdio adapter config in the Claude Code answer.

| Question | 2 Sept, before | 6 Sept, after #341 |
|---|---|---|
| Marcus and Confucius | 10 calls, $0.013, Marcus unreachable | 3/3, 10–12 calls, $0.013–0.017, Book 5 quoted with source |
| Epictetus and Wattles | 10 calls, $0.012, no philosopher note opened | 1/1, 13 calls, $0.017, answered from the philosophers topic matrix |
| Claude Code memory | 8 calls, $0.010, no command in the answer | 3/3, 10–12 calls, $0.013–0.023, command, token and config present |

One nit is still open. `expand` lists children with a preview after the title, `- Prerequisites — Docker running locally Node.js …`, and in two of three runs the model copied the whole line into `toc_path` and got `section not found`. One recovery call each time. It is on the list.

## Who pays

Here is the part I have to be careful about. I did not run a strong model against the broken build. The claim that a bigger model would have digested these defects quietly is an observation from running agents on bigger models in production, plus an inference from the traces: at the circular error a bigger model would likely have fanned out or tried the bare `markavrelii`, and paid two or three calls for it. A guess with a mechanism behind it, not a measurement. The measurement is cheap: replay the 2 September traces up to the first error against a bigger model and count the recovery steps. I have not done it yet.

What I did measure is the other half. On gpt-5.4-mini the six defects were the difference between answering and not answering, on all three questions. The cost per question barely moved, a cent either way. What moved was whether the budget went into the answer or into the wall.

That is why the defects stayed invisible. A strong model turns a wrong server into a slightly higher bill, and a bill is not a bug report. A cheap model turns it into a wrong answer, and a wrong answer is one, if someone reads the log.

## Do this with your own server

Take the questions your agent answers most, run them through a cheap model with a tool budget, and read the trace line by line. Not the answer, the trace. Every place where the server's text cost the model one more call is a defect, whether or not it was an error. The script is `scripts/mcp_search_logger.py`; the browser version is [[search_visualizer|the visualizer]]. Ten calls and a cent per run was enough to find six.
