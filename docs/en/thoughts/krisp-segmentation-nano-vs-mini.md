---
title: "Ask for the turn, not the interval"
free: true
lang_redirect: "[[ru/thoughts/krisp-segmentation-nano-vs-mini]]"
---

A language model is good at seeing where a conversation turns. It is bad at arithmetic on timestamps. My first attempt at splitting call transcripts into topics ignored that. I asked for `[start–end]` blocks, and the model couldn't keep the intervals straight. What fixed it wasn't a sharper prompt, and it wasn't a bigger model. It was a different question: ask where each topic starts (one real timestamp from the transcript) and let code work out the spans. The cheapest model then placed transitions 100% in order and 97% on real timestamps, and it beat the bigger model.

## The step

Turning calls into a knowledge base starts with segmentation: cut a transcript into topic blocks so you can navigate by time and feed the later extraction pass. Wrong boundaries, and everything downstream inherits the error. I tested two cheap models, call them nano (cheapest) and mini (mid), on 28 of my own calls.

## First pancake: asking for intervals

I told the model to output `[MM:SS–MM:SS]` blocks with a one-line label. It got the topics right and the structure wrong. Across the 28 calls, the cheapest model's ranges were fully contiguous on only 29% of them. Here is the failure, masked (real topics swapped for neutral stand-ins, structure untouched):

```
[42:52–49:53]  …
[44:00–47:50]  …   begins and ends inside the previous block
…
[09:25–05:17]  …   ends four minutes before it starts
```

So I did the tempting thing and fought the prompt. A strict "contiguous, no gaps, cover the whole call" fixed the ordering. Then it over-split, chopping the opening into per-utterance crumbs, about 52 blocks a call:

```
[00:02–00:07]  greeting
[00:07–00:14]  names exchanged
[00:14–00:18]  aside about the weather
…thirteen blocks in the first two minutes
```

A granularity cap ("a block is a topic, 8 to 18 of them") fixed that too, back to roughly 11 clean blocks at 100% contiguous, still for the cheapest price. And the bigger model handled it fine straight away, at 3.7 times the cost. So I had two tidy conclusions on the table. Perfect the prompt, or pay for the model. Both of them dodged the real problem: I was patching a task that shouldn't exist. The model can't subtract `05:17` from `09:25`, so why am I making it try?

## The turn: mark transitions, derive spans

Stop asking for intervals. Ask the one thing the model does well. Where does a new topic start? One line per transition: a timestamp copied from the transcript, plus the topic that begins there. The spans are arithmetic, so they belong to code, not the model. Block *i* runs from marker *i* to marker *i+1*.

Same 28 calls. "Anchored" means the timestamp is a real transcript line; "monotonic" means the markers come in order.

| | avg markers | monotonic | anchored |
|---|---|---|---|
| **nano** | 24 | **100%** | **97%** |
| mini | 21 | 96% | 93% |

The cheapest model is nearly perfect, and it sits ahead of the bigger one. That isn't luck. The task is now "spot a shift, copy a timestamp," which is pattern-matching, the thing it's good at, instead of interval bookkeeping, the thing it's bad at. The bigger model's extra reasoning buys nothing here, and it sometimes rounds or invents a time, which costs it on anchoring. The one thing it still does a little better is phrase the label as a crisp takeaway instead of a topic name. That gap is small, and it's the only reason left to reach for it.

## The lessons

When a model keeps failing a sub-task, suspect the question before you suspect the model. The interval iterations were a long detour around the fact that I was asking for arithmetic. Reshaping the task to the model's strength did in one move what three prompt versions couldn't.

Prompt and task shape come before a model upgrade. Here the reshaped task makes the cheapest model the best choice, so the upgrade would have been wasted money.

And keep the deterministic part deterministic. The model marks the turns. Code computes the spans, guarantees contiguity, and never drifts.

## What this is and isn't

This ran on my own recent calls, not a curated benchmark. The prompts are tuned to my domain and my language, and one person's call set is not a general claim about the models. Treat the numbers as directional. What generalizes is the shape: spotting beats arithmetic, the cheap model catches up (here it wins), and task design outweighs model choice.

This is stage one of [[en/thoughts/knowledge-graph|the knowledge graph]]. The markers give clean blocks. Pulling out the meaning, and the links between blocks, into `[[wikilinks]]`, comes next.
