# MCP instructions A/B: does a note make Haiku search the docs better? (2026-07-02)

**TL;DR:** Yes, on the metric that matters. With the docs-base `initialize` instructions note in its system prompt, `claude-haiku-4.5` used **26% fewer tool calls**, triggered **41% fewer whole-note dumps**, and read the exact section instead of the whole note **71% of the time vs 29%** (+43 pp). Answer accuracy was identical (87.5% both) — the note does not make a cheap model find *more* answers, it makes it find them *cheaper*. Total spend for all 48 runs: ~$0.92.

Live A/B against `https://trip2g.com/_system/mcp` with real tool execution. Harness: `scripts` reproduced below; raw runs in `2026-07-02_mcp_haiku_ab_results.json`.

## Method

- Model: `anthropic/claude-haiku-4.5` via OpenRouter, `max_tokens=700`, `tool_choice=auto`.
- Tools exposed to the model: the live MCP `search`, `note_html`, `expand`, `similar` (real schemas from `tools/list`, executed against the live endpoint — the model's tool calls hit the real docs base).
- **Variant A (with):** system prompt = the docs-base `initialize` note (`docs/_mcp_initialize.md`, frontmatter stripped).
- **Variant B (without):** system prompt = a bare generic line ("you have MCP tools, use them"), same tools, no note.
- 8 questions across topics, each run 3x per variant = 48 runs total.
- Loop cap: 8 tool-call rounds per run.

Questions: federation with an HMAC secret; what the MCP server exposes; publishing an Obsidian vault; what a subgraph is and how ACL works; how `toc_path` drill-down saves tokens; what a fuzzy pointer is; connecting Telegram; what `expand` does.

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

## Cost

733,569 input + 36,473 output tokens across all 48 runs. At Haiku 4.5 rates (~$1/M in, ~$5/M out) that is ~$0.92. Well under the $2 cap.

## Reproduce

The harness (`ab.py`): fetches live tool schemas, runs each question through the OpenAI-format chat-completions loop with real MCP tool execution, and logs calls/tokens/dumps/section-reads per run. Raw output: `2026-07-02_mcp_haiku_ab_results.json`. The instructions note under test: `docs/_mcp_initialize.md`. How to write such a note: `docs/dev/mcp_instructions_guide.md`.
