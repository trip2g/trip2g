# DSPy for fleet prompt optimization: research + design (2026-07-02)

**Verdict: yes, but narrowly and not yet.** DSPy is a good fit for *benchmarking* fleet roles as black boxes once the OpenAI-compatible endpoint (task #41) exists, and a real fit for *optimizing* exactly two krisp roles (`transcript-segment`, `transcript-wiki`) where a metric is definable. It is not a general "make all role-notes better" button: every use needs a labeled trainset (~20-200 examples) and a scoring metric, and that is the actual cost, not the integration code. On portability the honest answer is: DSPy does **not** transfer prompts between models; it **re-optimizes** per model. What transfers is the program (signature + module structure) and the trainset; the demos and instructions get re-derived for each target LM. That re-optimization loop is precisely what would formalize our current hand-run haiku/nano A/B.

Recommended integration mode: **mode 2 first** (fleet endpoint as the LM under test, DSPy as eval harness), **mode 1 second** (DSPy compiles the prompt, result written back into the role-note) for the segment/extract pair only. Effort: ~2-4 days for a working eval harness after #41 lands; the trainset labeling is the long pole and is ongoing work, not a one-time task.

## 1. What DSPy actually is (verified against docs)

DSPy (Stanford NLP, [dspy.ai](https://dspy.ai/)) treats prompts as *compiled artifacts*, not source code. You write a **program**; DSPy derives the prompt.

- **Signature**: a typed I/O declaration. The docstring is the seed instruction:
  ```python
  class BasicQA(dspy.Signature):
      """Answer questions with short factoid answers."""
      question: str = dspy.InputField()
      answer: str = dspy.OutputField(desc="often between 1 and 5 words")
  ```
- **Modules**: strategies that turn a signature into LM calls: `dspy.Predict` (direct), `dspy.ChainOfThought` (adds a reasoning field), `dspy.ReAct` (tool loop). Modules compose into multi-step programs.
- **LM abstraction**: `dspy.LM` wraps any backend via LiteLLM. An OpenAI-compatible endpoint is first-class: prefix the model with `openai/` and set `api_base`:
  ```python
  lm = dspy.LM("openai/qwen/qwen3-14b",
               api_base="http://localhost:11434/v1",
               api_key="local", model_type="chat")
  dspy.configure(lm=lm)
  ```
- **Evaluation**: `dspy.Example(...).with_inputs(...)` builds a devset; a metric is a plain function `def metric(gold, pred, trace=None) -> float`; `dspy.evaluate.Evaluate(devset=..., metric=..., num_threads=...)` runs the program over the set and reports the aggregate score. The metric can itself be a small DSPy program (LLM-as-judge), which the docs explicitly endorse for free-form outputs.
- **Optimizers** ("teleprompters"): take (program, metric, trainset), return a better program:

  | Optimizer | Optimizes | Data guidance (from docs) |
  |---|---|---|
  | `LabeledFewShot` / `BootstrapFewShot` | few-shot demos | ~10 examples |
  | `BootstrapFewShotWithRandomSearch` | demo sets, searched | 50+ |
  | `MIPROv2` | instructions + demos, Bayesian search | 200+ (or `auto="light"` for less) |
  | `COPRO`, `SIMBA`, `GEPA` | instruction text (GEPA: reflective, reads metric feedback) | varies |
  | `BootstrapFinetune` | LM weights | after prompt-level optimization |

  All share one interface: `optimized = optimizer.compile(program, trainset=trainset)`. Optimized programs serialize with `program.save('./v1.json')` / `.load()`, and the JSON contains the derived instructions + picked demos, which is what we would write back into a role-note.

Sources: [LM configuration](https://dspy.ai/api/models/LM/) ([GitHub source](https://github.com/stanfordnlp/dspy/blob/main/docs/docs/learn/programming/language_models.md)), [optimizers overview](https://github.com/stanfordnlp/dspy/blob/main/docs/docs/learn/optimization/optimizers.md), [evaluation overview](https://github.com/stanfordnlp/dspy/blob/main/docs/docs/learn/evaluation/overview.md), [cheatsheet](https://dspy.ai/cheatsheet/), [FAQ](https://github.com/stanfordnlp/dspy/blob/main/docs/docs/faqs.md), [issue #852 on OpenAI-compatible endpoints](https://github.com/stanfordnlp/dspy/issues/852).

## 2. The two user claims, assessed

### 2.1 "Benchmark prompts": true, with a dataset tax

Yes, and this is the low-risk part. `Evaluate` gives parallel scored runs over a devset with a progress table. It is exactly the harness our hand-run haiku-vs-nano comparison lacked. The catch is undramatic: you must supply the devset and the metric. For krisp we already have the raw material (8 calls in `~/krisp-run/vault-out/`, cached extractions in `work-vault/`, per-model output namespacing like `qwen3-14b/wiki/**` in the wiki role) but nothing is labeled as gold yet.

### 2.2 "Port prompts between models": mostly false as stated; true as re-compile

This is the claim to be precise about. DSPy does **not** lift-and-shift a tuned prompt onto a new model. The docs' own framing (FAQ): "the same program expressed in 10 or 20 lines of DSPy can easily be compiled into multi-stage instructions for GPT-4, detailed prompts for Llama2-13b, or finetunes for T5-base." Compiled *for* each target, per model.

What transfers vs what gets re-derived:

| Transfers (your asset) | Re-derived per model (DSPy's job) |
|---|---|
| program structure (modules, control flow) | few-shot demos (bootstrapped against the new LM) |
| signature (fields, types, seed docstring) | instruction wording (proposed + scored on the new LM) |
| trainset + metric | the final prompt string |

So "portability" means: switching the krisp pipeline from qwen3-14b to gpt-5.4-nano is one `compile()` run against the new `dspy.LM`, not a manual prompt rewrite. That is genuinely valuable, and genuinely not free: each compile costs optimizer LM calls (MIPROv2 `auto="light"` is tens of runs over the trainset).

## 3. Fleet fit

Grounding: a fleet role-note is flat frontmatter config + a Jet-templated body (`internal/fleet/role.go`; e.g. `docs/fleet/krisp/roles/transcript-segment.md` with `{{ change_file.Path }}` / `{{ change_file.Content }}` substitutions). The body becomes the `Instruction` in `agentruntime.Input` and lands inside a fixed system-prompt template with a tool loop (search/read_note/write_note/patch_note/finish) in `internal/agentruntime/runtime.go`. The fleet is already an OpenAI-compatible *client* (`docs/dev/openai_compatible_endpoint.md`, `LLMBaseURL`). Task #41, the OpenAI-compatible *server* (`POST /fleet/<note-path>/v1/chat/completions` exposing a role as a model), is designed but **not built**.

### 3.1 Can DSPy treat a fleet agent as an LM? Yes, once #41 exists

This is the clean seam. With #41, a fleet role is just another chat-completions backend:

```python
segment_role = dspy.LM(
    "openai/fleet/krisp/roles/transcript-segment",   # model name = role path (per #41's contract)
    api_base="https://box.example.com/fleet/krisp/roles/transcript-segment/v1",
    api_key=os.environ["TRIP2G_HAT"],                # HAT auth, as fleet admin lane already does
    model_type="chat", cache=False)
```

DSPy then evaluates the *whole agent* (Jet rendering + system prompt + tool loop + scope enforcement) as a black box. Nothing inside fleet changes; DSPy never sees Jet. Requirements this puts on #41's design: the endpoint must accept a plain user message as the task input and return the run's final answer as `choices[0].message.content`, plus `usage` token counts (DSPy/LiteLLM reads them). Non-streaming is fine.

### 3.1a "Can I benchmark the *skills inside* the agent?" — no, not through the endpoint

This is the precise question, and the answer is a clean split, verified against current DSPy docs (v3.3.0b1, May 2025):

- **End-to-end benchmarking of the agent-as-a-model: yes.** A `dspy.LM` on the #41 endpoint is a single opaque predictor. `dspy.Evaluate(devset, metric)` compares gold answers to the run's final output over a dataset — exactly what the DSPy specialist described ("valish through dataset, compares эталонные ответы vs current prompt output"). Metric is a plain `def metric(example, prediction) -> float` (0.0–1.0); it can be rule-based or LLM-as-judge. This is our formalized haiku/nano A/B. ([metrics docs](https://dspy.ai/getting-started/metrics/))
- **Optimizing/scoring the *internal steps* (skills/sub-prompts) through the endpoint: no.** DSPy's optimizers reach internal steps only via the **trace**: the metric's 3rd arg `trace` (and GEPA's `pred_name`/`pred_trace`) is populated during compilation, and MIPROv2/GEPA walk the *module graph* to inject per-predictor instructions + demos. A black-box HTTP endpoint exposes none of that — DSPy sees one module, so it can only optimize the single prompt hitting the endpoint, never the server-side sub-steps. ([MIPROv2](https://dspy.ai/api/optimizers/MIPROv2/), [GEPA](https://dspy.ai/getting-started/gepa-optimization/))

So the specialist is exactly right: **to benchmark/optimize skills *inside* the agent, the agent loop itself must be written as a DSPy program** (modules: `Predict`/`ChainOfThought`/`ReAct`), not consumed as an OpenAI endpoint. His "агент как обычный цикл, может добавили минимальную абстракцию — смотри docs" maps to `dspy.ReAct`, and the newer **ReActV2** in 3.3.0b1 (native tool-calls, structured `dspy.History`, provider tool-call IDs) is that minimal-abstraction refresh. GEPA then optimizes each sub-module of a ReAct program individually (docs explicitly show it tuning both the `Predict` and `ChainOfThought` inside one ReAct agent).

**The catch for fleet:** our agent loop lives in Go (`internal/agentruntime/runtime.go`), not in a DSPy/Python program. There is no free path to "DSPy optimizes the skills inside a fleet run" — it would require re-expressing the fleet tool loop as a DSPy `ReAct` program in Python (a parallel reimplementation, only worth it for one role where per-step tuning pays off), or exposing the individual role sub-steps as separately addressable endpoints and treating the *composition* as the DSPy program. Endpoint-level (#41) buys mode-2 end-to-end eval only. That is the honest ceiling.

### 3.2 Two integration modes

**Mode 2 (recommended first): fleet endpoint as LM under test, DSPy as benchmark harness.**
DSPy program is trivial (`dspy.Predict` of a one-field signature, or even a raw `lm(messages=...)` call); all the intelligence stays in the role-note. DSPy contributes `Example`/`Evaluate`/metric/parallelism/score tables. Honest framing: this uses maybe 20% of DSPy, and a bespoke Python eval script could do the same. DSPy earns its keep by making the *same* devset+metric reusable for mode 1 later, and by the LLM-as-judge metric plumbing.

**Mode 1: DSPy owns the prompt, result written back into the role-note.**
Here you restate the role's task as a signature + module, compile with MIPROv2/GEPA against the *raw backing model* (qwen3-14b via OpenRouter/Ollama, same `--llm-base-url` targets fleet already supports), then extract the optimized instruction + demos from the saved program JSON and paste them into the role-note body above the Jet-templated data section.

The honest seam, stated plainly: **a Jet-templated free-form prompt is not a DSPy signature.** Our role bodies mix three things DSPy separates: (a) the instruction ("разметь смысловые границы..."), (b) the output format contract (the strict frontmatter + `## [MM:SS–MM:SS]` skeleton), (c) the data injection (`{{ change_file.Content }}`). To use mode 1 you must factor the note the same way:

```
role-note body = [DSPy-optimized instruction block]   <- overwritten by the optimizer
               + [output format contract]              <- ours, frozen (write_note path, frontmatter keys)
               + [Jet data injection]                  <- ours, frozen ({{ change_file.* }})
```

Only the first block is DSPy's. Also, DSPy modules produce field-structured outputs while our roles end with a `write_note` tool call; the compile loop therefore runs against a *proxy task* (text in, segmented markdown out) rather than the full tool loop. That approximation is acceptable for segment/extract (single write, deterministic path) and wrong for multi-step tool-heavy roles. Roles with `executor: code` are out of scope entirely (no prompt to optimize).

Which fits fleet? **Mode 2 fits everything; mode 1 fits only roles that are "one LLM transformation wearing a tool-call coat"**, which segment and wiki-extract are.

### 3.3 Concrete example: auto-tuning the krisp `segment` role

1. **Trainset.** ~20-50 `dspy.Example(transcript=..., segments=gold).with_inputs("transcript")` from `~/krisp-run` calls. Gold = hand-corrected boundary maps (we already produce these when reviewing runs; see `2026-07-02_boundary_and_extraction.md` evidence file).
2. **Metric: boundary stability + format.** Programmatic part: parse `[MM:SS–MM:SS]` headers, score boundary alignment against gold within a ±60s tolerance window (precision/recall on boundaries), plus hard checks from the role's own rules (segment count within `duration/5` ±, frontmatter keys present). Judge part (small DSPy program): "is each header a Minto-style conclusion, not a topic label?" Weighted sum, `def segment_metric(gold, pred, trace=None) -> float`.
3. **For `extract` (transcript-wiki)** the metric shifts to extraction faithfulness: every `[[WikiLink]]` entity must appear in the source transcript (string/fuzzy check kills hallucinated entities, aligned with our provenance invariant in `2026-07-02_call_kb_design.md`), plus judge-scored coverage of gold decisions/actions.
4. **Compile.** `MIPROv2(metric=segment_metric, auto="light").compile(program, trainset=...)` against qwen3-14b. Then run `Evaluate` on a held-out devset for qwen3-14b, gpt-5.4-nano, haiku: this is our A/B, formalized into one table with a number per model instead of eyeballing `qwen3-14b/wiki/**` vs another namespace.
5. **Write-back.** Read the saved program JSON, take the optimized instruction (and demos, rendered as "Пример входа/выхода" sections), splice into the instruction block of `docs/fleet/krisp/roles/transcript-segment.md`, keep the format contract + Jet lines untouched, push via `updateNotes`/obsidian-sync. Frontmatter gets `optimized_by: dspy-miprov2`, `optimized_at`, `eval_score` for audit. Per model, the note either carries the winning model's prompt, or we keep per-model role variants (the wiki role already namespaces output per model, so the pattern exists).

### 3.4 A fleet maintenance role that re-optimizes other roles?

Feasible, with feet on the ground. DSPy is a Python library, so this is an `executor: code` python role (the same shape as krisp ingest), cron mode, with `read_patterns: [fleet/**/roles/**, fleet/eval/**]` and `write_patterns` covering the roles plus a report note. The run: load devset notes, hit the raw model (or, post-#41, the fleet endpoint for pure benchmarking), `Evaluate`, and only if score improves by a threshold, `patch_note` the instruction block, gated by `needs_review: true` so a human approves the prompt change. Practical frictions: DSPy + LiteLLM must be installed in the code-executor environment (heavier than stdlib; needs the interpreter image to carry it or a venv), compile runs cost real tokens so cron should be weekly not per-change, and an agent rewriting agents' prompts wants the same guardrail we use elsewhere: never silently merge, always leave a diff for review. Start with benchmark-only cron (score report note, no write-back); promote to auto-patch after the metric has earned trust.

## 4. What #41 unlocks, and effort

Without #41: mode 1 is already possible today (DSPy talks to the raw model directly; fleet is only the deployment target of the tuned text). With #41: mode 2 becomes possible, meaning end-to-end regression benchmarks of *agent behavior* (prompt + template + tool loop + scoping together), cross-model A/B of whole roles, and third-party harnesses beyond DSPy (promptfoo, OpenAI evals) for free, since the interface is the standard one.

Effort estimate:
- Mode 2 harness (post-#41): **2-3 days**: `dspy.LM` against the endpoint, devset loader from vault notes, `Evaluate` wrapper, score report. Plus #41 itself, which is its own task.
- Mode 1 for segment+extract: **3-4 days**: signature/proxy-task factoring, the two metrics (boundary scorer is real parsing work), one MIPROv2 compile, write-back script.
- Trainset labeling: **ongoing**, the real cost. ~1-2 hours per batch of 10 gold calls; 20 minimum to start (BootstrapFewShot tier), 50+ before MIPROv2 medium is honest.
- Maintenance role: **+2 days** on top, only after the above proves out.

## 5. Honest caveats

- **Portability is re-optimization, not transfer.** Each new target model = a compile run with optimizer-LM cost. Budget it; don't promise "tune once, run anywhere".
- **Jet-vs-signature mismatch is structural.** Mode 1 requires factoring role bodies into optimizable-instruction / frozen-format / frozen-data blocks and optimizing a proxy task without the tool loop. Multi-step tool-heavy roles and `executor: code` roles don't fit mode 1 at all.
- **Dataset and metric burden dominates.** DSPy without gold labels and a trustworthy metric is a random-prompt generator. The boundary metric needs a tolerance-window design decision; the faithfulness metric needs the provenance check implemented. Garbage metric in, confidently-worse prompt out. We already learned this lesson with the reranker (measured worse, removed): measure first, adopt second.
- **Optimized prompts drift from readable role-notes.** MIPROv2 output can be verbose and demo-stuffed. Role-notes are also human documentation; keep the optimized block clearly delimited and reviewable, or the vault's "prompt as a note you can read" property erodes.
- **DSPy for mode 2 alone is heavy.** If we never do mode 1, a 200-line pytest-style eval script gives the same benchmark. Adopt DSPy when the optimization story (section 3.3) is actually wanted, which for the krisp cascade it is.

## Status

Research only. Nothing built. Depends on: task #41 (OpenAI-compatible endpoint, not built), gold-label devset for krisp calls (not started). First concrete step when picked up: label 20 gold segment maps and build the boundary metric; that is useful even without DSPy.
