---
title: "Do agents need instructions for a knowledge base"
free: true
lang_redirect: "[[ru/thoughts/do-agents-need-instructions]]"
---

*What we did: took a private knowledge base for a training course, connected to agents over MCP, and asked four Claude models the same questions, with the server instructions and without them. A blind judge scored the answers. The short version: a weak model needs instructions and fails without them; strong models find the instructions on their own. Read it if you are building a knowledge base for agents and wondering whether to write instructions for it.*

A trip2g knowledge base connected over MCP has three ways to tell an agent how to work with it. A note with `mcp_method: initialize` reaches the client as soon as it connects, and the client puts it into the system prompt. A note with `mcp_method: instructions` is served on request through the `instructions()` tool. And the `?method=` parameter selects a different instructions note, for example for a separate role. We had seen plenty of good instructions on real bases, but no measurement of whether they help. So we measured.

## How we tested

The base is private and in Russian: a training course with concepts, step-by-step chains, "situation → what to do" decision maps, and an `_instructions.md` note with routes by task type. The `initialize` instructions are 865 characters: what the base is, where to start, how to search and how to answer, and what to do when there is no material.

The questions were written by a separate agent that was not allowed to read the instructions, so the questions couldn't bend towards the routes. Sixteen questions of four kinds:

- 6 direct, answered in one note;
- 4 situational, "I have X, what do I do next";
- 3 about a specific detail deep in a note: a number, a threshold, a template field;
- 3 the base doesn't answer, where the right answer is "the course doesn't cover this".

Each model answered each question twice: with the server instructions and without. The tools were the same in both cases, `instructions()` included. The models ran through Claude Code, which puts server instructions into the system prompt the way an ordinary client does. For the "without" condition a local proxy stripped them from the server's response on connect. Haiku answered all 16 questions; Sonnet, Opus and Fable answered four, one of each kind.

A separate judge, blind to model and condition, scored the answers on four points: correctness (0–2), whether the answer rests on what was retrieved, whether the source is named, and whether there is a verbatim quote.

## Results

| Model | Instructions | Questions | Right note opened | Correctness (of 2) | Grounded | Source named | Quote | Tool calls |
|---|---|---|---|---|---|---|---|---|
| Claude Haiku 4.5 | no | 16 | 9 of 13 | 1.56 | 0.75 | 0.75 | 0.12 | 45 |
| Claude Haiku 4.5 | yes | 16 | 12 of 13 | 1.81 | 1.00 | 0.81 | 0.81 | 44 |
| Claude Sonnet 5 | no | 4 | 3 of 3 | 2.00 | 1.00 | 1.00 | 0.50 | 16 |
| Claude Sonnet 5 | yes | 4 | 3 of 3 | 2.00 | 1.00 | 0.75 | 0.75 | 19 |
| Claude Opus 5.5 | no | 4 | 3 of 3 | 2.00 | 1.00 | 1.00 | 0.75 | 20 |
| Claude Opus 5.5 | yes | 4 | 3 of 3 | 2.00 | 1.00 | 1.00 | 0.75 | 15 |
| Claude Fable 5.1 | no | 4 | 3 of 3 | 2.00 | 1.00 | 1.00 | 1.00 | 22 |
| Claude Fable 5.1 | yes | 4 | 3 of 3 | 2.00 | 1.00 | 1.00 | 1.00 | 27 |

"Right note opened" counts only the questions the base does answer.

## What it means

**Weak models need instructions.** Every failure in this run came from Haiku without instructions. Once it claimed it had no access to the base and answered from general knowledge. Another time it gave a generic answer not grounded in what it had found. With instructions there was not a single failure: correctness rose from 1.56 to 1.81, grounding from 0.75 to 1.00, and it opened the right note in 12 questions out of 13 instead of 9.

**Strong models find the instructions themselves.** Sonnet, Opus and Fable answered every question correctly in both conditions. Without server instructions, in most runs they called `instructions()` or opened `_instructions.md` on their own: Sonnet in 3 cases of 4, Opus and Fable in 4 of 4. Haiku never did so unprompted, and did in 10 cases of 16 with instructions.

**Format rules work.** The base's instructions say to answer with a quote from the note. The share of Haiku's answers with a verbatim quote rose from 0.12 to 0.81.

**Instructions cut blind searching.** With them Haiku ran 19 searches instead of 29 for the same total number of calls: instead of trying phrasing after phrasing, it followed the routes and read the right notes more often.

## Caveats

- **Small samples.** Sixteen questions for Haiku and four for the others are observations, not proven differences. Especially for the strong models: on four questions they hit the ceiling of the scale.
- **The Claude Code harness.** It has its own system prompt about programming. One of Haiku's failures without instructions, refusing with "I'm a coding assistant", is partly an artefact of that harness.
- **Claude models only.** We dropped the gpt-5.4-mini run from this article: in our own runner, tool errors reached the model as an empty string, which distorted its numbers.
- **One base.** A course with decision maps and a routing note is a well-structured base. On a base without that structure the instructions have less to lean on, and the result may differ.

What this means for knowledge base authors is collected in the guide: [[en/user/instructions_guide|How to write instructions for a knowledge base]].
