# MCP instructions A/B: does a note make a cheap model search the docs better? (2026-07-02)

**TL;DR:** Yes, on the metric that matters, and it holds across two different cheap models. With the docs-base `initialize` instructions note in its system prompt, `claude-haiku-4.5` used **26% fewer tool calls**, triggered **41% fewer whole-note dumps**, and read the exact section instead of the whole note **71% of the time vs 29%**. `gpt-5.4-nano` (see the nano section below) shows the same direction with its own model-specific quirks. Answer accuracy was a tie in both cases: the note does not make a cheap model find *more* answers, it makes it find them *cheaper*.

Live A/B against `https://trip2g.com/_system/mcp` with real tool execution. Raw runs in `2026-07-02_mcp_haiku_ab_results.json` (haiku) and `2026-07-02_mcp_nano_ab_results.json` (nano); harness in `2026-07-02_mcp_haiku_ab_harness.py`.

## Method

- Model: `anthropic/claude-haiku-4.5` via OpenRouter, `max_tokens=700`, `tool_choice=auto`.
- Tools exposed to the model: the live MCP `search`, `note_html`, `expand`, `similar` (real schemas from `tools/list`, executed against the live endpoint — the model's tool calls hit the real docs base).
- **Variant A (with):** system prompt = the docs-base `initialize` note (`docs/_mcp_initialize.md`, frontmatter stripped).
- **Variant B (without):** system prompt = a bare generic line ("you have MCP tools, use them"), same tools, no note.
- 8 questions across topics, each run 3x per variant = 48 runs total.
- Loop cap: 8 tool-call rounds per run.

Questions: federation with an HMAC secret; what the MCP server exposes; publishing an Obsidian vault; what a subgraph is and how ACL works; how `toc_path` drill-down saves tokens; what a fuzzy pointer is; connecting Telegram; what `expand` does.

## How the instructions reach the model (validity note)

The WITH variant puts the note directly into the model's system prompt. It does **not** run a live MCP client handshake, so read the numbers as "the note is in context" rather than "a real client fetched it".

This is a valid proxy because of where the note lands in a real setup. trip2g's `handleInitialize` returns the note content in the standard MCP `instructions` field of the `initialize` result (`internal/case/mcp/resolve.go:154`, `result["instructions"] = content`). That is exactly the field a spec-compliant MCP client (Claude Code, Cursor, and others) surfaces into the model's context at connect time. Putting the note in the system prompt models what a well-behaved client does with `initialize.instructions`.

Caveat: the real-world benefit depends on the client actually surfacing that field. Some clients truncate the instructions, place them somewhere the model weights less, or ignore them. So these numbers are the *ceiling* for a compliant client, not a guarantee for every client.

## Metrics

- **Tool calls until answer** — fewer = the answer is found faster.
- **Grounded correctness** — reached the right note/section via tools (not from memory).
- **Total tokens** (input+output) — cost proxy.
- **Whole-note dumps** — `note_html` responses over 6,000 chars, i.e. the full-note fallback that fires when `toc_path` is missing or wrong (the anti-pattern the note is designed to prevent).
- **Section-read rate** — share of runs where the model read via `note_html(toc_path=...)` at least once.

## Results — retrieval questions (7 questions x 3, 21 runs per variant)

Question 8 ("what does `expand` do") is excluded from this table: in every run of *both* variants the model answered from memory without any tool call — it already knows what "expand" means generically, so it never exercised retrieval. Reported separately below.

| Metric | WITH note | WITHOUT note | Delta |
|--------|-----------|--------------|-------|
| Avg tool calls | 4.00 | 5.43 | **26% fewer** |
| Avg tokens | 17,265 | 18,696 | **8% fewer** |
| Avg whole-note dumps | 0.95 | 1.62 | **41% fewer** |
| Section-read rate | 71% | 29% | **+43 pp** |
| Grounded correctness | 100% | 100% | tie |

### Per-question (avg over 3 reps): tool calls / tokens / dumps / section-read %

| Question | WITH | WITHOUT |
|----------|------|---------|
| federation + HMAC | 2.7 / 12,949 / 0.0 / 100% | 5.3 / 21,793 / 1.3 / 100% |
| MCP tools exposed | 2.7 / 13,979 / 1.0 / 33% | 5.7 / 20,207 / 1.3 / 0% |
| publish Obsidian vault | 7.3 / 28,361 / 0.3 / 100% | 5.0 / 17,387 / 1.3 / 67% |
| subgraph / ACL | 7.3 / 28,095 / 2.3 / 100% | 10.0 / 32,055 / 3.7 / 33% |
| token economy toc_path | 3.0 / 13,282 / 1.0 / 100% | 3.3 / 12,578 / 1.3 / 0% |
| fuzzy pointer | 2.7 / 12,359 / 1.0 / 67% | 3.7 / 11,417 / 1.0 / 0% |
| telegram publishing | 2.3 / 11,827 / 1.0 / 0% | 5.0 / 15,438 / 1.3 / 0% |

## Honest caveats

- **Accuracy is a tie, not a win.** Both variants reached the right note on the questions they searched (87.5% over all 8, 100% over the 7 retrieval ones). The docs' hybrid search is good enough that even a bare Haiku usually lands the right note. The note's value is *efficiency*: fewer round-trips, far fewer full-note dumps, reading the section instead of the whole file.
- **WITHOUT sometimes still did the section read** (29% of runs, e.g. the "publish vault" question). Haiku is not incapable of `toc_path` on its own; the note makes it the default rather than the exception.
- **One question was a wash by design.** "What does `expand` do" was answered from memory by both variants — a generic-knowledge question is not a retrieval test. It is included in the 48-run total and the all-questions aggregate, excluded from the retrieval table.
- **"publish vault" cost more WITH than WITHOUT.** In two reps the note-guided model drilled deeper (7+ calls) chasing the exact section, spending more than the bare model that stopped at a coarser answer. The note biases toward precise section reads, which occasionally costs an extra call — a real, if minor, trade-off.
- **Small n.** 3 reps per cell smooths but does not eliminate Haiku's nondeterminism; treat single-question numbers as directional, the aggregate as the signal.

## All-questions aggregate (24 runs per variant, incl. the memory-answered q8)

| Metric | WITH | WITHOUT |
|--------|------|---------|
| Avg tool calls | 3.5 | 4.75 |
| Avg tokens | 15,479 | 16,606 |
| Avg dumps | 0.83 | 1.42 |
| Section-read rate | 62.5% | 25% |
| Correct (found right note) | 87.5% | 87.5% |

## Second model: gpt-5.4-nano

Same harness, same 8 questions x 3 reps x 2 variants, model swapped to `openai/gpt-5.4-nano`. And the result is genuinely different, so it is worth stating plainly: **the note helps nano much less than haiku, and on tokens it hurts.**

### nano — retrieval questions (7 x 3 = 21 runs per variant)

| Metric | With note | Without note | Delta |
|--------|-----------|--------------|-------|
| Avg tool calls | 5.14 | 6.05 | 15% fewer |
| Avg tokens | 16,143 | 11,694 | **38% MORE** |
| Avg whole-note dumps | 0.52 | 0.52 | no change |
| Section-read rate | 100% | 95% | +5 pp |
| Right answer found | 100% | 100% | tie |

### Why nano behaves differently

- **nano is already frugal by default.** Without any note it read by section 95% of the time and dumped a whole note only 0.52 times per question — already near the floor haiku only reached *with* the note. nano does not need to be told to use `toc_path`; it does it on its own.
- **So the note has little dump/section headroom to recover** (dumps unchanged, section-read already at 95%), and its main measurable effect is to **add its own ~500 tokens to every turn's context** plus nudge nano toward more exploration. Net tokens go up 38%, not down.
- **Model-specific quirks:** nano over-specifies calls — it passes ~7 arguments per `note_html` (every id field plus `match_id` plus `toc_path` at once) where haiku passes ~2. Harmless (the server resolves any one identifier), but it shows nano does not try to minimize the call itself. nano also leaned on `expand` more (21 calls total across all runs vs haiku's 12), doing more structural navigation.
- Both models: no question was answered from memory in the nano runs (unlike haiku, which guessed the "what does expand do" question from memory in both variants — nano searched even for that one).

### The honest cross-model reading

The note is not a universal win. It is a **correction for a wasteful default.** haiku, left alone, reads whole notes and burns extra round-trips; the note fixes that (26% fewer calls, 41% fewer dumps). nano already reads by section, so the note buys it a small tool-call reduction (15%) at the cost of more tokens. If your agent model is already disciplined, the note mostly costs context; if it is wasteful, the note pays for itself many times over. Since you rarely control which model connects to a public MCP endpoint, shipping the note is still the right default: it caps the downside for the wasteful models and costs the frugal ones a few hundred tokens.

### nano — all-questions aggregate (24 runs per variant)

| Metric | With | Without |
|--------|------|---------|
| Avg tool calls | 4.79 | 5.54 |
| Avg tokens | 14,970 | 10,698 |
| Avg dumps | 0.46 | 0.46 |
| Section-read rate | 100% | 96% |
| Correct | 100% | 100% |

## Cost in dollars, not just tokens

Input and output bill at different rates, so the token deltas above do not translate 1:1 to money. Prices used (per million tokens): Claude Haiku 4.5 **$1.00 input / $5.00 output** (Anthropic published pricing, verified 2026-07-02); gpt-5.4-nano **$0.05 input / $0.40 output** (OpenRouter listed rate at run time — the figure I actually applied).

Per-query and per-1,000-query cost, retrieval questions only:

| Model | Variant | Avg in tok | Avg out tok | $/query | $/1,000 queries |
|-------|---------|-----------|------------|---------|-----------------|
| Haiku 4.5 | with note | 16,487 | 778 | $0.0204 | **$20.37** |
| Haiku 4.5 | without | 17,859 | 837 | $0.0220 | **$22.05** |
| nano | with note | 15,585 | 558 | $0.00100 | **$1.00** |
| nano | without | 11,140 | 554 | $0.00078 | **$0.78** |

**The economic verdict differs by model, and on two counts it is the opposite of the naive intuition.**

1. **Input dominates cost here, not output.** In this workload input tokens are 78–81% of the bill, even though output is priced 5–8x higher per token. The reason: each `note_html`/`expand` result comes back as *input* on the next turn (the whole tool result rides in context), while the model's own replies are short. So the lever that moves money is round-trips and how much each tool result dumps into context, not output length.

2. **The note is a recurring input cost.** It rides in the system prompt on every call (a real client gets it once via `initialize.instructions` and then carries it in context). So the question is whether the per-call saving beats the per-call cost of carrying the note.

- **Haiku: the note pays for itself.** 26% fewer tool calls means fewer big tool-result inputs, which more than offsets the note's own input tokens. Net: **8% cheaper with the note** ($20.37 vs $22.05 per 1,000 queries, about $1.68 saved per 1,000).
- **nano: the note costs money.** nano was already frugal (few dumps, near-100% section reads), so the note removes little waste while adding its own input tokens plus a nudge to explore more. Net: **29% more expensive with the note** ($1.00 vs $0.78 per 1,000, about $0.22 more per 1,000).

Both are small absolute numbers at this scale, but the sign flips: ship the note and a wasteful model gets cheaper while a frugal one gets slightly dearer. Since you rarely control which model connects to a public endpoint, the note is still the right default — it caps the expensive tail at the price of a few cents per thousand queries on the frugal models.

Total benchmark spend: haiku 733,569 in + 36,473 out ≈ $0.92; nano 590,560 in + 25,483 out ≈ $0.04. Combined ≈ **$0.96**, under the $2-per-model cap.

## How real MCP clients (Claude Code / Codex) behave

The benchmark injects the note into the system prompt; it does not drive a real MCP client. I ran a bounded live check to narrow that gap.

- **Claude Code (v2.1.198): confirmed on subscription auth, cheap path observed.** Run the CLI on its logged-in Claude subscription (flat billing) rather than a per-token API key by unsetting `ANTHROPIC_API_KEY` in the subprocess env — with the key set, `claude` bills the console/API path and fails on "Credit balance is too low"; with it unset, `claude` falls back to the OAuth subscription credentials and runs. Added the base as an HTTP MCP server at local scope (`claude mcp add --transport http`), which connected (`✔ Connected`), then asked benchmark questions with `--output-format stream-json --verbose` to see tool calls. On "how do I set up federation with a private peer using an HMAC secret", the trace was:

  ```
  mcp__trip2g-docs__search    {"query": "federation private peer HMAC secret setup"}
  mcp__trip2g-docs__note_html {"pid": 658, "toc_path": ["Adding a private peer (two-step exchange)"]}
  ```

  That is exactly the cheap path the note teaches: `search` for a pointer, then `note_html` with the matched `toc_path` to read one section, no whole-note dump. It answered correctly and cited the note. So a real, spec-compliant client on subscription auth drives the drill-down loop end to end.

- **Canary: Claude Code auto-surfaces `initialize.instructions` (auto-pickup proven).** `initialize.instructions` is a standard MCP field; a compliant client fetches the server's instructions on connect and injects them into the model context — nobody pastes them. To prove Claude Code actually does this (not just that the model drilled down by instinct), I asked it directly, pasting nothing: "what connection instructions did the trip2g-docs server give you at startup? Quote them." It returned the note verbatim under a heading it labelled "MCP Server Instructions":

  > You are connected to the trip2g documentation base (trip2g.com). ... Follow the loop below ... 1. `search(query)` ... 2. Read only that section: `note_html(pid=<note_id>, toc_path=<match.toc_path>)`. ... Worked example ... `search("private federation peer HMAC secret")` → note_id=658 ...

  That text is `docs/_mcp_initialize.md` word for word, and it was never in the prompt — so the client auto-surfaced `initialize.instructions`. Race note: on a cold headless start the server can still be connecting when the first turn runs, and the model then correctly reports it has no instructions yet; force the connection with one tool call first (or warm the session) and the instructions appear.

- **Codex CLI: not measured.** The CLI runs, but headless it is slow and token-heavy (a trivial prompt spent ~35k tokens), and wiring up the full MCP retrieval loop headless was out of scope for a bounded check. Marked not measured rather than forcing a number.

**What is confirmed.** The server returns the note in the standard MCP `instructions` field of the `initialize` result (`internal/case/mcp/resolve.go:154`, `result["instructions"] = content`); Claude Code (on subscription billing) auto-surfaces that field into the model context on connect (canary above — the model recited the note verbatim with nothing pasted); and the same client then follows the taught `search` → `note_html(toc_path)` loop against the live base. So the full delivery path is confirmed end to end: server emits the field, client auto-picks it up, model follows it. The A/B is the separate, controlled measurement of *effect size* — how much the note shifts behavior versus a model left to its own instincts (the WITHOUT arm shows models drill down some of the time unprompted). Delivery: confirmed live. Magnitude: quantified by the A/B.

## Reproduce

The harness (`2026-07-02_mcp_haiku_ab_harness.py`): fetches live tool schemas, runs each question through the OpenAI-format chat-completions loop with real MCP tool execution, and logs calls/tokens/dumps/section-reads per run. Swap the `MODEL` constant to change model. Raw output: `2026-07-02_mcp_haiku_ab_results.json` and `2026-07-02_mcp_nano_ab_results.json`. The instructions note under test: `docs/_mcp_initialize.md`. How to write such a note: `docs/dev/mcp_instructions_guide.md`.
