---
title: "Good MCP instructions make a cheap model find answers faster"
free: true
lang_redirect: "[[ru/thoughts/mcp-instructions-make-cheap-models-faster]]"
---

*What this is: a measured A/B on two cheap models. I gave Claude Haiku and GPT-5.4-nano an MCP connection to the trip2g docs and asked each eight questions, once with a short instructions note in its context and once without. Same tools, same questions. The note made Haiku search 26% faster with 41% fewer whole-note dumps. nano, already frugal on its own, barely needed it. Read it if you run a knowledge base over MCP and wonder whether the instructions note earns its keep.*

The short answer: a good instructions note does not make a cheap model smarter. It corrects a wasteful default. The catch is that not every cheap model is wasteful. On Haiku the note was a clear win. On nano, which already reads carefully without being told, the note bought almost nothing and even cost tokens. Both stories are below, because the honest lesson is in the contrast.

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

## The second model told a different story

Then I ran the exact same A/B on `gpt-5.4-nano`. The direction flipped on the metric that matters most for cost.

| Metric | With note | Without note | Delta |
|--------|-----------|--------------|-------|
| Avg tool calls | 5.14 | 6.05 | 15% fewer |
| Avg tokens | 16,143 | 11,694 | 38% MORE |
| Avg whole-note dumps | 0.52 | 0.52 | no change |
| Section-read rate | 100% | 95% | +5 pp |
| Right answer found | 100% | 100% | tie |

nano barely benefits, because nano was already doing the right thing. Without any note it read by section 95% of the time and dumped a whole note only half as often as Haiku managed even with the note. There was almost no waste for the note to remove. So the note's main effect was to add its own few hundred tokens to every turn and nudge nano to explore a little more, which pushed tokens up, not down.

The lesson is not "the note is bad." It is that the note is a correction for a wasteful default, and nano did not have one. Haiku, left alone, reads whole notes to be safe; the note fixes that. nano reads by section on instinct; the note has nothing to fix and still costs context. If you could pick the model, you would skip the note for a disciplined one. But you rarely pick the model that connects to a public endpoint, so shipping the note is still the right call: it caps the damage from the wasteful models and costs the careful ones a few hundred tokens.

A couple of nano-specific quirks worth noting: it over-specifies its calls, passing about seven arguments to `note_html` (every id field plus `match_id` plus `toc_path` at once) where Haiku passes two. Harmless, since the server resolves any single identifier, but it shows nano does not try to trim the call itself. And nano leaned on `expand` more than Haiku did, doing more structural navigation before committing to a read.

## What it costs in money

Tokens are the mechanism, but the bill is what you pay, and input and output are priced differently. I used Claude Haiku 4.5 at $1.00 per million input tokens and $5.00 per million output (Anthropic's published pricing), and gpt-5.4-nano at $0.05 input and $0.40 output (the OpenRouter rate at run time).

| Model | Variant | $/query | $/1,000 queries |
|-------|---------|---------|-----------------|
| Haiku 4.5 | with note | $0.0204 | $20.37 |
| Haiku 4.5 | without | $0.0220 | $22.05 |
| nano | with note | $0.00100 | $1.00 |
| nano | without | $0.00078 | $0.78 |

Two things surprised me here, both worth stating plainly.

First, input dominates the bill, not output. In this workload input tokens are around 80% of the cost, even though output is priced five to eight times higher per token. The reason is structural: every section or note the model reads comes back as input on the next turn, so a big read is a big input charge, while the model's own answers are short. What moves money is the number of round-trips and how much each read dumps into context, which is exactly what the note controls.

Second, the note itself is a recurring input cost. It rides in the context on every call, so the real question is whether the per-call saving beats the per-call cost of carrying it. For Haiku it does: cutting a quarter of the tool calls removes more input than the note adds, so the note-guided run is about 8% cheaper, roughly $1.68 less per thousand queries. For nano it does not: nano was already frugal, so the note removes little while adding its own weight, and the guided run costs about 29% more, roughly 22 cents more per thousand queries.

So the note is not free, and on an already-careful model it is a small net cost. It earns its keep on the wasteful models, where it cuts both the round-trips and the whole-note dumps that actually drive the bill. Since you do not choose which model connects to a public endpoint, paying a few cents per thousand queries on the frugal ones to cap the expensive tail is a trade worth making.

## How the note reaches the model

One honest limit on all of the above: this benchmark puts the note in the model's system prompt directly. It does not run a real MCP client handshake. So strictly, I measured "the note is in context", not "a real client fetched it and showed it to the model".

That is a fair stand-in, because of where the note lands in a real setup. `initialize.instructions` is a standard field in the MCP protocol: a spec-compliant client fetches the connected server's instructions during the connection handshake and surfaces them into the model's context automatically. Nobody pastes the note into the prompt; the client does it on connect. trip2g returns the docs-base note in exactly that field (`internal/case/mcp/resolve.go:154`). Putting the note in the system prompt for the A/B models what a well-behaved client does with it.

I designed that mechanism a while ago and, embarrassingly, had never checked that a real client actually picks the field up. So I ran a canary. Using Claude Code on its flat subscription (unset `ANTHROPIC_API_KEY` before launching, or it bills the metered console path and stops on an out-of-credit error), I registered the base as an HTTP MCP server and asked it, without pasting anything: "what connection instructions did the trip2g-docs server give you at startup?" It quoted the note back verbatim, under a heading it labelled "MCP Server Instructions": the identity line, the four-step retrieval loop, and the worked federation example, word for word from `docs/_mcp_initialize.md`. Since I never put that text in the prompt, the only way it reached the model was the client auto-surfacing `initialize.instructions`. The mechanism works. (One race worth noting: on a cold headless start the server can still be connecting when the first turn runs, and then the model correctly reports no instructions yet. Force the connection with one tool call first, or let it warm up.)

And it drives the loop end to end, not just recites it. On "how do I set up federation with a private peer using an HMAC secret", the tool trace was `search("federation private peer HMAC secret setup")` followed by `note_html(pid=658, toc_path=["Adding a private peer (two-step exchange)"])`. That is the taught loop exactly: search for a pointer, then read one section by its `toc_path`, no whole-note dump. It answered correctly and cited the note.

So the delivery path is real, not assumed: the canary proves the client auto-surfaces the note, and the federation trace proves the model then follows it against the live base. The A/B is still the part that measures the effect size: how much the note shifts behavior versus a model left to its own instincts, which the without-note arm shows drills down some of the time anyway. Delivery confirmed live; magnitude quantified by the A/B. (Codex I left alone: headless it is slow and token-heavy, and a bounded check was not the place to wire it up.)

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

And one question was a wash by design, at least for Haiku: "what does `expand` do" got answered from memory by both Haiku variants, because the model already knows what "expand" means in the abstract. A generic-knowledge question is not a retrieval test, so I scored it separately. nano, for what it is worth, searched even for that one rather than trusting its memory.

## Method, so you can redo it

Models: `claude-haiku-4.5` and `gpt-5.4-nano` over OpenRouter, tools set to the live MCP `search`, `note_html`, `expand`, `similar`. Variant A's system prompt was the note at `docs/_mcp_initialize.md` with its frontmatter stripped, exactly what the MCP `initialize` response serves. Variant B's was a single generic line. Eight questions spread across federation, publishing, access control, the token-economy tools, and Telegram; three runs each; forty-eight runs per model. A "dump" is a `note_html` result over 6,000 characters. Total spend was about $0.92 for Haiku and $0.04 for nano.

The full tables, per-question breakdowns, raw runs, and the harness are in the dev notes: `2026-07-02_mcp_haiku_ab.md`, `2026-07-02_mcp_haiku_ab_results.json`, and `2026-07-02_mcp_nano_ab_results.json`.

## Write your own

If you run a trip2g base, give it its own `initialize` note. Lead with what the base is, teach the search then `toc_path` then `expand` loop, show one worked example with a real note id, and write the guardrails from your tools' real failure paths. Keep it tight; a cheap model degrades under a wall of text. The full recipe, with the shadowing gotchas and a live validation checklist, is in the dev note `docs/dev/mcp_instructions_guide.md` (ships with the instructions note itself).

The upside is asymmetric. The note costs you an afternoon once and a few hundred tokens per connection. In return, a wasteful model like Haiku pays a quarter less to find its answers, and a careful one like nano is no worse off on the metric that decides success. When you cannot choose which model connects, that is the trade you want.
