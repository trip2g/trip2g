---
title: "The Harness"
free: true
lang: en
lang_redirect: "[[ru/harness]]"
---

# The Harness

**A harness is the automated scaffolding around an AI agent that makes its
behavior verifiable and repeatable.** You build one to move from "the agent
usually does the task" to "the agent provably does the task, and I see the exact
moment it stops." The prompt tells the agent what to do; the harness proves
whether it did.

## The job it's hired for

As a Job-to-be-Done: *when I rely on an agent for a complex, repeated task, I want
to know — without re-reading every output — whether it actually did the job, so I
can trust the result and catch regressions the moment they appear.*

Nobody wants a harness for its own sake; they want the confidence it produces. You
hire one when "looks fine to me" stops being good enough.

## What it actually is

In the practical sense, a script that:

1. runs the agent on a fixed task,
2. inspects the **artifacts** it produced (files, frontmatter, reports — not the chat),
3. runs a batch of formal checks, and
4. emits a score like `N/10 PASS`.

That score is the point. It turns a vague feeling into a number that moves when the
behavior moves — you see a regression the run it happens, and exactly which check
failed.

## When to build one

- A skill or task you want to call **"done"** and rely on. Without a harness, a
  skill is just a hypothesis.
- Anything the agent runs **repeatedly** — every run is a chance to drift silently.
- **Multi-step** work where the agent can quietly skip a step and the output still
  looks plausible.
- Before you **ship** a skill to other agents or teammates.

## Examples

- **`landing-iterate.sh`** — clears the vault, sends the prompt, waits, reads
  artifacts from the container, runs ten checks (file exists? frontmatter valid?
  report created? both sections marked?). Three iterations took a landing-page
  skill from 4 fails to 0.
- **Harness-first engineering** (Datadog, Code with Claude 2026) — invest in
  automated checks instead of reading every line of output; shift from human review
  to machine scaffolding.
- **Offline benchmark as a stop-gate + online evals on production traces** — the
  benchmark is the emergency brake: no green, no ship.
- **"Dreaming"** (Anthropic) — an async process reads agent transcripts, finds
  recurring errors, and updates shared memory. A harness that runs while you sleep.

## Anti-examples

- **Rewriting the prompt louder.** Adding "ALWAYS", "NEVER skip" and eyeballing the
  result is cosmetic. If the agent *can* skip a step, words won't stop it — only
  removing the possibility will (e.g. produce the file and its report in one tool
  call).
- **A one-off manual check.** Looking once and declaring victory; you won't notice
  when it breaks next week.
- **Happy-path-only checks.** A harness that only verifies the case you already knew
  worked passes forever and protects nothing.
- **No harness at all.** Calling a skill finished because the demo looked right. The
  failure is usually a stable structural hole, invisible to the eye.

## The rule it teaches

Once a harness shows you a stable failure, fix the **structure**, not the **text**.
Not "add an exclamation mark" but "move this step first", "merge it with its
neighbor", "make it impossible to skip". Text can't force an agent to comply;
structure can remove the choice.

## Related

- [[en/thoughts/dogfooding|Dogfooding]] — using your own product in real work
