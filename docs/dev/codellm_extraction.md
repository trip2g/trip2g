# codellm: extracting code execution into a standalone OpenAI-compatible service

**Status:** Open design (not yet built, 2026-07-14). Sibling to `subprocess_agent.md`, `queues.md`, and the fleet agent-runtime code.
**Builds on:** the fleet agent runtime (`internal/agentruntime`, `internal/fleet`, `cmd/fleet`) and the `process_isolation.md` sandbox research.

## TL;DR

Today fleet runs code in-process: an `executor: code` role extracts fenced blocks from a rendered body and runs them under an in-process Linux sandbox (`internal/agentruntime/runcode.go` + `sandbox_linux.go`). We move that execution into a **standalone `codellm` service** that speaks the OpenAI `/v1/chat/completions` protocol and *pretends to be an LLM*: it receives chat messages containing markdown-with-code, executes the fenced blocks, and returns the writes as OpenAI **`tool_calls`** (`write_note`/`patch_note`/`finish`). Because fleet's `agentruntime.LLM` interface already talks to any OpenAI-compatible endpoint (`internal/agentruntime/llm.go:49`, `openai_llm.go`), a code role becomes an ordinary `executor: llm` run pointed at codellm's `base_url` — and fleet's existing scope enforcement (`ScopedKB` + `write_patterns`) applies the returned tool calls unchanged. codellm holds **no vault, no KB, no auth, no secrets** — that emptiness is the isolation win. The multi-block streaming pipeline and the block-by-block debug seam move inside codellm and stay steppable. The `runcode`/`sandbox`/`interpreters` subsystem leaves fleet.

Chosen return contract: **`tool_calls`, not JSON-in-content** (the alternative reintroduces a code-specific parse branch in fleet, defeating the whole point). **Hard invariant:** code-role runs are pinned to `MaxSteps: 1` and codellm must **always** emit `finish` — otherwise fleet's stateless loop re-calls codellm, which re-extracts the same blocks and re-applies every write up to the step ceiling (see "The re-execution hazard" below). Delivery bag rides the request as a labeled message; env passthrough is **dropped**, which means **secret-dependent code roles are unsupported** under codellm (not "migrate the secret into the bag" — that would push secrets into the HTTP request). Top 3 risks: (1) determinism is only "no external nondeterminism," not bit-reproducibility — the sandbox controls fs/net/resources but not clock/PRNG; (2) codellm is an unauthenticated code-execution oracle that can write arbitrary attacker-controlled bytes to any in-scope path (scope checks the path, never the content), unless the fleet↔codellm channel is locked down (mTLS/shared token/loopback); (3) apply-error semantics diverge between the two paths (a bad patch is a hard 502 today, a soft success under the loop) and must be reconciled.

## Why this came up

`executor: code` and the `exec(program, code)` tool both run untrusted code **inside the fleet process**, confined only by an in-process re-exec sandbox (`MaybeRunSandboxChild` at `internal/agentruntime/sandbox_linux.go:45`, called first in `cmd/fleet/main.go:36`). That couples three concerns that should be separable:

- **Blast radius.** The code runs in the same address space and namespace as fleet, which holds the JWT secret, the LLM API key, and the admin HAT. The sandbox scrubs the *child's* env (`buildChildEnv`, `coderun.go:388`) and re-execs into namespaces, but the confinement lives in the same binary it is protecting against.
- **Reuse.** The clean interface fleet already has for "give me writes" is `agentruntime.LLM.Chat` returning `tool_calls`. Code execution is the odd path that bypasses it (`handler.go:376` branches to `f.codeRunner` instead of `agentruntime.Run`).
- **Debuggability.** The owner wants a future debugger that steps the markdown block-by-block and inspects the pipe *between* blocks. That model already exists as a seam (`pipelineDebugTaps`, `runcode.go:159`; `/debug/run-block`, `internal/fleet/debug.go`) but is buried inside fleet.

Extracting code execution behind an OpenAI-compatible endpoint collapses all three: fleet loses a special path, the sandbox moves to a service whose container *is* the blast-radius boundary, and the block/pipe stepping becomes a first-class API on that service.

## Current-state map: what moves, stays, or is deleted

### Moves (fleet → codellm)

| File / symbol | What it is |
|---|---|
| `internal/agentruntime/coderun.go` — `RunBlock`, `ExtractFencedBlocks`, `parseCodeOutput`, `buildChildEnv`, interpreter registry, `limitedBuffer` | Single-block execution, fence parsing, stdout-JSON contract, secret-scrubbed env, stdout caps |
| `internal/agentruntime/runcode.go` — `RunCode`, `runPipeline`, `buildBlockCmd`, `pipelineDebugTaps` | Multi-block streaming pipeline (`io.Pipe` block→block) + the debug tap seam |
| `internal/agentruntime/sandbox.go`, `sandbox_linux.go`, `sandbox_other.go` (+ tests) | The re-exec sandbox: namespaces, Landlock, rlimits, uid drop, `MaybeRunSandboxChild` |
| `internal/agentruntime/interpreters.json` (+ `LoadInterpretersFile`, `SetInterpretersJSON`) | Program allowlist and fence-label → interpreter map |
| `internal/fleet/debug.go` — `/debug/run-block` (**inline `?body=` path only**), `debug_ui.html` | Loopback step-through UI; becomes codellm's debug surface. The `?path=`/`Role` lookup does **not** move — see the debugger seam section |

### Stays in fleet / agentruntime

| File / symbol | Why it stays |
|---|---|
| `internal/agentruntime/llm.go` (`LLM`, `Message`, `ToolCall`, `ChatResult`, `ToolDef`) | The shared chat surface; codellm implements it on the wire, fleet consumes it |
| `internal/agentruntime/openai_llm.go` (`OpenAILLM`) | fleet's client for **both** the real LLM and codellm — only `base_url` differs |
| `internal/agentruntime/runtime.go` (`Run`, the tool loop, `execTool`) | The agent loop; now also drives code roles by pointing at codellm |
| `internal/agentruntime/scope.go` (`ScopedKB`), `kb.go`, `filekb.go` | **Scope enforcement stays in fleet** — this is the isolation win: codellm cannot write, fleet applies every change through `write_patterns` |
| `internal/fleet/*` (handler, role, fleet, reconcile) | Role discovery, dispatch, delivery, HMAC — unchanged |

### Deleted (after cutover)

- The in-process code path in fleet: the `codeRunner` field (`internal/fleet/fleet.go:24`, default `agentruntime.RunCode` at `:39`) and the `if p.Role.Executor == executorCode` branch in `execRole` (`handler.go:376`). Replaced by an ordinary llm-run against codellm.
- Once the `exec` tool also moves (see Phase 5): `makeExecInvoker`'s in-process `RunBlock` call (`runtime.go:384`), fleet's `--allowed-programs`/`--sandbox`/`--sandbox-network`/`--max-stdout-bytes`/`--interpreters` flags (`cmd/fleet/main.go:483`–`515`), and the `MaybeRunSandboxChild()` call at `main.go:36`.

## The wire protocol

codellm is a real OpenAI chat-completions endpoint. Fleet reaches it through the same `OpenAILLM` it uses for real models (`openai_llm.go:36`), so the request is whatever `agentruntime.Run` already emits, and the response must be a standard `ChatCompletionResponse`.

### Request (fleet → codellm)

`agentruntime.Run` builds the messages internally (`runtime.go:130`): a system prompt wrapping the instruction (the **rendered role body**, which for a code role is markdown containing fenced blocks) plus a `user: "Begin."` turn. The advertised `tools` array carries the write tools. One addition is required (see below): the **delivery bag** as a dedicated message.

```jsonc
POST /v1/chat/completions
{
  "model": "codellm",                       // role model or a fixed id; codellm echoes it
  "messages": [
    { "role": "system",
      "content": "You are a scoped trip2g micro-agent...\n<role body with ```python ... ``` blocks>" },
    { "role": "system", "name": "fleet_input",
      "content": "{\"changed_files\":[...],\"attached_notes\":[...],\"depth\":1,\"now\":\"2026-07-14T..\"}" },
    { "role": "user", "content": "Begin." }
  ],
  "tools": [ /* write_note, patch_note, finish schemas (runtime.go:450) */ ]
}
```

codellm concatenates the message contents, runs `ExtractFencedBlocks` over them (the system-prompt wrapper has no fences, so only the role body's blocks are found), executes the pipeline with the `fleet_input` message as the `$FLEET_INPUT` bag, and parses the last block's stdout as `{"changes":[...],"answer":"..."}` — exactly today's contract (`coderun.go:281`).

**Delivery bag mechanism.** Today the bag is a `[]byte` written to `$FLEET_INPUT` (`CodeInput.Input`, `runcode.go:26`; the change-delivery bag is built by `buildInputBag` at `handler.go:227`, the cron bag by `buildCronInputBag` at `handler.go:351`). Over HTTP it must ride the request body. The bag is delivered as a `system` message named `fleet_input` whose content is the JSON. This needs a small, honest change to the shared runtime: add an optional `InputBag []byte` to `agentruntime.Input` (`runtime.go:40`) and have the message builder append the `fleet_input` message when it is set. A real LLM sees a labeled JSON context block (harmless); codellm treats it as `$FLEET_INPUT`.

**Env passthrough is dropped — but secret-dependent skills are re-supported, better, at the executor.** `EnvPassthrough`/`EnvPrefix` (`runcode.go:27-28`) are live features: parsed from role frontmatter (`env_passthrough:` / `env_prefix:`, `role.go:64-65`), validated (`role.go:244`), forwarded to the code child (`buildChildEnv`, `coderun.go:388`). No parent env crosses the HTTP boundary, so a role that used `env_passthrough` to hand fleet's secret to its code stops working **as written**. The tempting fallback — "move the secret into the bag" — is **wrong**: the bag rides the request, pushing the secret across the very boundary we are hardening.

The correct model: **secrets live at the executor (codellm), not in fleet's env, and a skill declares what it needs.**

- A codellm skill (e.g. the Krisp KB pipeline) ships a manifest declaring `requires_secrets: [OPENROUTER_KEY, ...]` — capability, not value.
- The secret values live in **codellm's own secret config** (deploy-time env / mounted secret for the playground) **or** are fetched at run from trip2g's existing admin secret store (`internal/case/admin/getsecret`) over the locked fleet↔codellm-adjacent channel. Never in the role note, never in fleet's env, never in the per-request bag.
- **Install enforces presence:** registering the skill checks its `requires_secrets` are configured in codellm; if a key is missing, install/run **fails loudly** — "Krisp requires OPENROUTER_KEY; set it in codellm config" — instead of silently mis-running.
- **At run, codellm injects only the declared keys** into that skill's sandbox (a per-skill allowlist), never codellm's whole env.

Why this is a net upgrade, not a loss: today `env_passthrough` puts fleet's secret into the env of *every* code child of that role; the new model scopes secrets per-skill, at the execution boundary where the code actually runs, with an explicit declared-and-enforced requirement. The only thing genuinely gone is "silently inherit whatever fleet happened to have" — which was the leak. Existing roles using `env_passthrough:` still need porting to a manifest (see the pre-Phase-4 audit).

### Response (codellm → fleet) — tool_calls, decided

```jsonc
{
  "id": "codellm-1a2b3c",
  "object": "chat.completion",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "",
      "tool_calls": [
        { "id": "call_0", "type": "function",
          "function": { "name": "write_note",
            "arguments": "{\"path\":\"concepts/x.md\",\"content\":\"...\"}" } },
        { "id": "call_1", "type": "function",
          "function": { "name": "patch_note",
            "arguments": "{\"path\":\"index.md\",\"find\":\"a\",\"replace\":\"b\"}" } },
        { "id": "call_2", "type": "function",
          "function": { "name": "finish", "arguments": "{\"answer\":\"done\"}" } }
      ]
    },
    "finish_reason": "tool_calls"
  }],
  "usage": { "prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0 }
}
```

Fleet's existing loop (`runtime.go:172`) applies each `write_note`/`patch_note` through `ScopedKB` (the `write_patterns` check at `scope.go:141`), records `res.Changes`, then hits `finish` and returns the answer. **Scope enforcement is unchanged and still lives in fleet.** The whole run completes in a single loop turn because codellm returns all changes plus `finish` in one completion.

`{changes}` → tool_calls mapping is mechanical: a `content`-only change → `write_note(path, content)`; a `find`/`replace` change → `patch_note(path, find, replace)`; the `answer` → a trailing `finish(answer)` (kept **last** so writes are applied before the loop returns). Token usage is `0` (matching the old `TokensUsed: 0` for code runs at `runcode.go:109`).

### Why not JSON-in-content

The rejected alternative: codellm returns `{"changes","answer"}` as the assistant message **content** with no tool calls. Fleet's loop would see no tool calls, treat the content as the final answer (`runtime.go:160`), and apply **nothing** — because the llm path has no code-JSON parser. To make it work, fleet would keep a code-specific "parse content, apply changes" branch, i.e. exactly the special path we are deleting. tool_calls wins because it reuses `write_note`/`patch_note`/`finish` + `ScopedKB` verbatim; its only cost is a lossless serialization inside codellm, which is low-risk and fully testable.

### The re-execution hazard (first-class invariant)

Fleet's `Run` loop stops a run only on an explicit `finish` tool call or a completion with **no** tool calls (`runtime.go:160`, `:173`); otherwise it appends the tool results and **calls the LLM again**, up to `MaxSteps` (the fleet step ceiling, default 25, `cmd/fleet/main.go:477`). codellm is **stateless** — every call re-extracts the same fenced blocks from the still-present system prompt and re-runs the whole pipeline. So if codellm ever returns writes **without** a `finish` call (an empty-answer bug, a mapping slip), fleet applies those writes, loops, calls codellm again, and codellm **re-runs every block and re-applies every write** — fully authorized through the scoped write token — up to 25 times. For a non-idempotent block (append, counter increment, external POST via `exec`-style code) this is silent corruption, not a crash.

Two defenses, both required, both first-class:

1. **Pin `MaxSteps: 1` for code-role runs.** A code role is single-shot by definition; one turn is all it ever needs. This caps the blast radius at exactly one re-run even if `finish` is missing. Set explicitly in the Phase-3 `Input` (see migration).
2. **codellm must ALWAYS emit `finish`.** Even on an empty answer, the last tool_call is `finish` (with `answer: ""` if there is nothing to say). This is a hard output invariant of codellm, asserted in its conformance tests: *every* successful completion ends with exactly one `finish`.

Test obligation: a fleet-side test that a code-role `Input` is constructed with `MaxSteps == 1`, and a codellm-side conformance test that no completion omits `finish`.

### Errors

A failing block (non-zero exit / timeout) is a **role-run failure**, not an LLM answer. codellm returns an OpenAI-style error with HTTP **422** (not 5xx) so `OpenAILLM.createWithRetry` does **not** retry it (`openai_llm.go:118` retries only 429/5xx) — a deterministic failure should not be retried. Fleet surfaces the chat error, `execRole` returns it, and the handler responds `502` (`handler.go:334`), preserving today's behavior.

**Apply errors diverge silently — decide the canonical behavior.** "Lossless" holds for the happy path and for scope denials (`ErrWriteDenied` is recorded and the run continues, identically in both paths — `runcode.go:120` vs `runtime.go:269`). It does **not** hold for *apply* errors. Today a failing `patch` (a `find` that is absent or non-unique) is a **hard** `RunCode` failure → `502` → retryable (`runcode.go:124`). Under the tool loop the same failure is a **soft** per-tool error string fed back to the model, and the run still proceeds to `finish` → `200` success with the bad patch simply skipped (`runtime.go:292`). That is a behavior change: a role that used to fail loudly now succeeds quietly. Decision to make explicit (not deferred): either (a) accept the soft semantics (patch failures become denials-style skips, documented), or (b) have codellm treat a `parseCodeOutput`/apply-precondition failure as a **422** so parity with today's hard-fail is preserved. Recommendation: **(b)** — a code role's whole point is deterministic all-or-nothing application; a silently skipped patch is worse than a visible failure. Add a parity test that a bad-`find` patch produces the same outcome (fail) via both paths.

## The codellm service shape

New binary `cmd/codellm` + new package `internal/codellm` (housing the moved `coderun`/`runcode`/`sandbox`/`interpreters`). Deployed as a separate container. Same repo initially (shares Go types and the sandbox re-exec trick); can graduate to its own repo later without protocol change.

**Endpoints**

- `POST /v1/chat/completions` — the OpenAI-compatible surface described above. Non-streaming first; SSE streaming is optional and not needed for the contract (fleet wants the final tool_calls).
- `GET /v1/models` — optional, returns a single synthetic model for client compatibility.
- `GET /healthz` — liveness.
- Loopback-only debug surface (moved from `internal/fleet/debug.go`): `GET /debug/blocks`, `POST /debug/run-block`, `GET /` (the step UI). Bound to loopback via `ValidateLoopbackAddr` (`debug.go:223`), never on the public port.

**Sandbox lives here.** `cmd/codellm/main.go` calls `MaybeRunSandboxChild()` **first** (the re-exec sandbox works by re-execing the *current* binary with a marker env var, `sandbox_linux.go:45`), so it re-execs itself, not fleet. Per-block runs still get the full native posture (namespaces + Landlock + rlimits + uid drop). `--allowed-programs`, `--sandbox`, `--sandbox-network`, `--max-stdout-bytes`, `--interpreters` move to codellm's flags.

**Staying a real OpenAI endpoint.** Because codellm accepts the standard request and returns a standard `tool_calls` response, fleet can point the same code role at a **real LLM** `base_url` instead and it "just works" as an actual AI agent. The code executor becomes swappable with a model — which is the deeper payoff of the whole design, and a good conformance test: run the same role against codellm and against a stub OpenAI server and diff the applied changes.

## The debugger seam (design only — do not build now)

The owner's hard constraint: a future debugger must **step the markdown block-by-block and inspect what is in the pipe between blocks** (e.g. to write/validate a payload mid-pipeline). The design must not preclude it. codellm's internal execution model therefore treats each block **and its inter-block pipe edge** as an addressable, inspectable step — never a fused opaque run. The seam is **half-built today** — be precise about what exists, what is dead, and what moves:

1. **Per-block build (exists, moves).** `runPipeline` builds each block as a separate `builtBlock` with its own `context`/`cancel` and its own stdout/stderr buffers (`runcode.go:161`). Blocks are not merged into one command. This structure is what keeps stepping possible and must be carried over intact — do **not** collapse `runPipeline` into a single subprocess.
2. **Tap-able edges (exists but DEAD CODE, moves + must be tested).** Each inter-block connection is an `io.Pipe`; `wirePipeline` (`runcode.go:220`) threads an optional `io.TeeReader` per edge via `pipelineDebugTaps []*limitedBuffer` (`runcode.go:159`). **No caller passes a non-nil tap** — the parameter is always `nil` (`runcode.go:90`, `:190`), so the TeeReader path is never exercised and is untested. It is a *latent* seam, not a working one; the earlier claim that it is "already exercised for debug capture" was wrong. It must move **and** gain a test that actually drives a non-nil tap, or it will rot before the debugger arrives.
3. **Step API (exists, only the inline path moves).** `/debug/run-block` runs a single block with a **caller-supplied stdin** (the previous block's stdout) and returns stdout/stderr/exit/timing (`debug.go:182`); `/debug/blocks` enumerates (`debug.go:94`). **The actual stepping mechanism today is this single-block `RunBlock`+stdin, not the TeeReader pipeline.** But both endpoints have a `?path=`/`Role` mode that looks the role up in fleet's **live role registry** (`debug.go:102`, `:145`, via `f.roleByKey`) — that binding is fleet-specific and **cannot move** to codellm, which has no registry. Only the inline `?body=`/`Body` mode survives the extraction. codellm's debug surface therefore accepts inline markdown only; resolving a role name stays a fleet concern (fleet can render a role and POST the body to codellm's debug endpoint if needed).

Where a future debugger hooks: a stateful `/debug/step` session that (a) runs block *i*, (b) exposes the captured inter-block buffer for inspection **and edit**, (c) feeds the (possibly edited) buffer as block *i+1*'s stdin. The primitives (per-block build, per-edge tap, single-block-with-stdin runner) exist; the future work is making them stateful/interactive **and first making the tap seam a tested, live path** rather than dead code.

## Isolation model

The point is moving isolation from **in-process** confinement to a **container boundary**, in two layers:

- **Layer 1 — the codellm container.** codellm runs the untrusted code, so its container is the blast-radius boundary: its own PID/mount/network namespace, a cgroup memory/CPU limit, **default-deny egress**, no host mounts, dropped capabilities. codellm holds no long-lived KB handle, no auth, and (given the env-passthrough drop) no fleet secrets — so a compromise has far less to steal than in-process execution. But "nothing to steal" is **overstated**, and two exposures remain real:
  - **Note content transits the boundary.** Every request carries the rendered role body plus the delivery bag, which for change deliveries includes `ChangedFiles` and for cron/attach includes `AttachedNotes` — i.e. **note content**, not just metadata (`buildInputBag`, `handler.go:227`). A compromised or wiretapped codellm sees whatever notes the role's scope feeds it. This is why the channel must be authenticated/encrypted (risk 2), not merely "empty."
  - **Scope checks the path, never the content.** `ScopedKB.Write`/`Patch` enforce only that the *path* matches `write_patterns` (`scope.go:141`, `:152`); they do not inspect the bytes. A buggy or compromised codellm can therefore write **arbitrary attacker-controlled content** to any in-scope path. Scope confinement bounds *where* it can write, not *what*. The container boundary limits exfiltration; it does not make a misbehaving codellm safe to trust for content integrity — that is the fleet operator's residual risk.
- **Layer 2 — the per-block sandbox** (moved `sandbox_linux.go`): empty net + PID + mount namespaces, private `/proc`, Landlock FS confinement to the run workdir, rlimits, `NO_NEW_PRIVS`, uid drop. Still applied per code block, fail-closed in `native` mode.

**Determinism — state it precisely.** "100% deterministic" holds only under: no network egress (`SandboxPolicy.Network` defaults to `false`, `sandbox.go:40`), controlled resources, and a throwaway workdir per run (`coderun.go:160`). That guarantees **no external nondeterminism** (no network, no cross-run FS contamination). It does **not** guarantee bit-reproducibility: the sandbox does **not** freeze the wall clock or seed the PRNG, so `time.now()` / `random()` still vary. If a consumer needs reproducibility, codellm must additionally inject a fixed clock/seed — **not designed here**, flagged as an open question.

**Container/userns interaction (gotcha).** The per-block sandbox needs unprivileged user namespaces, which some container runtimes disable. If userns is unavailable inside the codellm container, `native` mode fails closed (refuses to run). Either the container must permit userns, or codellm relies on the container as the *sole* boundary and runs blocks in `besteffort`/`off` — an explicit, logged operator choice, not a silent downgrade.

## P3 plan (decided 2026-07-14, per-provider-fleet, direct cutover)

Four owner decisions lock the shape:
- **Full per-provider-fleet immediately.** One fleet serves one LLM endpoint; roles tag `fleet_id`; codellm is a fleet with `--llm-base-url=<codellm>` + `--llm-api-key=<key>`. fleet_id identity via the webhook hash-url registry (see fleet_graphql.md).
- **Rely on codellm's always-finish invariant — NO MaxSteps:1 pin.** codellm guarantees exactly one `finish` per completion (invariant + conformance test from #234), so the loop stops after one turn regardless of the role's MaxSteps. No need to detect codellm-bound roles to pin single-shot. (Accepted risk: a codellm always-finish bug would let the normal MaxSteps re-execute; guarded only codellm-side.)
- **Apply-error = hard-fail** (preserve today's all-or-nothing): a bad-`find` patch in the applied tool_calls fails the run loudly, not a soft per-tool skip.
- **Direct cutover** (no A/B flag): `executor: code` runs through the normal llm path against codellm; the in-process `RunCode` branch is deleted after.

### PR breakdown
- **P3a — fleet_id + role partitioning + hash-registry** (foundation, fleet-side, does NOT touch the code-exec path): `--fleet-id` (required) on fleet; `fleet_id` field on `ParseRole` (role.go); fleet reconcile/discovery processes only roles whose `fleet_id` matches; webhook registration URL becomes `/_fleet/<sha256("fleet:"+fleet_id)>/webhook`; reconcile de-dups/takes-over by the `/<h>/` segment (last-writer-wins, crash-recovery free). This is the identity+routing base.
- **P3b — cutover** (touches the live path): delete the `executor: code` → `f.codeRunner` branch (handler.go); code roles run as normal llm runs against the fleet's LLM (codellm); the delivery bag rides as a `fleet_input` system message on those runs (InputBag on agentruntime.Input); enforce apply-error **hard-fail** in the tool-apply loop (runtime.go) so a failed patch fails the run.
- **P3c — delete in-process** (after P3b verified live): remove fleet's in-process RunCode usage; keep `coderun` only for the `exec` tool until P5. Pre-req: env_passthrough audit (secret-dependent roles can't migrate — see the secrets manifest).

Order: P3a (foundation) → P3b (cutover) → P3c (delete). P3a is fleet-side/routing, safe to build first; P3b touches handler.go/runtime.go (the live path).

## Backward-compat and cutover

**Superseded routing model (2026-07-14):** the `--codellm-base-url` + in-fleet `f.codeLLM` second-client approach below is **replaced** by the per-provider-fleet model in `fleet_graphql.md` — one fleet serves one LLM endpoint, a role picks its fleet with `fleet_id`, and **codellm is just a fleet whose `--llm-base-url` points at the codellm service.** So the cutover has no `--codellm-base-url` flag and no second client inside a fleet: you stand up a codellm-configured fleet, retag `executor: code` roles to run against it, and that fleet's single LLM client *is* codellm. The invariants below (`MaxSteps: 1`, the `InputBag` delivery, apply-error 422 enforced at the fleet apply path, the parity tests, the env-passthrough audit) all still hold regardless of which fleet routes the run. The paragraphs below describe the older single-fleet mechanics and are kept for the invariant details; treat the routing specifics as historical.

**Migration of `executor: code` roles — zero role edits.** Keep `executor: code` as author-facing sugar; swap only the backend. Fleet holds two LLM clients: `f.llm` (real model, `cmd/fleet/main.go:80`) and a new `f.codeLLM` (an `OpenAILLM` pointed at `--codellm-base-url`). `execRole`'s code branch (`handler.go:376`) builds an `agentruntime.Input` pointed at codellm instead of calling `f.codeRunner`. The `Input` must be **fully specified** — `Run` hard-requires both `MaxTokens > 0` and `MaxSteps > 0` (`runtime.go:110`, `:113`) or it returns an error immediately, so a naive migration that omits them breaks every code role:

```go
agentruntime.Input{
    Instruction:   p.Instr,                      // rendered role body (markdown+code)
    WritePatterns: p.Role.WritePatterns,         // unchanged; scope stays in fleet
    Tools:         []string{"write_note", "patch_note", "finish"}, // or role's declared tools
    MaxTokens:     codeRoleMaxTokens,             // must be > 0; a small fixed floor (codellm reports 0 usage)
    MaxSteps:      1,                             // HARD: single-shot; caps the re-execution hazard
    InputBag:      p.InputBag,                    // the delivery bag (new field)
    LLM:           f.codeLLM,
    KB:            kb,
}
```

`MaxSteps: 1` is the re-execution defense (above); `MaxTokens` just has to be positive (codellm returns zero usage, so the value is a formality but cannot be `0`). Role notes are untouched; the `executor: code` keyword now means "single-step llm-run against codellm."

Two migration constraints roles must satisfy (checklist for the reconciler's dry-run, `main.go:546`):
1. The role's effective tool set must include `write_note`/`patch_note`/`finish` (empty `tools:` already yields the full default set at `runtime.go:334`, so most roles need nothing).
2. `write_patterns` must be set for any role that writes — otherwise `ScopedKB` denies every change (the existing deny-all trap, `role.go:109`), same as today.
3. **No `env_passthrough`/`env_prefix`.** A role that declares either (`role.go:64-65`) depends on a secret reaching its code child; that capability does not exist under codellm (see the env-passthrough drop). Such roles cannot migrate and must stay on the in-process path or be redesigned. The reconciler dry-run should flag them; see the pre-Phase-4 audit.

**Does the `exec(program, code)` tool move too?** Separate, smaller concern. `exec` (`makeExecInvoker`, `runtime.go:372`) is a tool an LLM role calls mid-reasoning, not a whole-role code run. Initial cutover targets `executor: code` only and **leaves `exec` in-process**. Consequence: to fully delete the sandbox from fleet, `exec` must also move (Phase 5) — until then fleet keeps `sandbox_linux.go` solely for `exec`. This is a genuine scope decision, flagged not papered over: shipping Phases 1–4 removes the code-role path and the debugger need; Phase 5 finishes the job by routing `exec` through codellm and dropping the sandbox from fleet entirely.

**Feature flag / rollout.** `--codellm-base-url` empty → keep the in-process `RunCode` path (old behavior). Non-empty → route `executor: code` through codellm. Per-deploy toggle enables A/B parity checks before deletion. Optionally per-role opt-in during transition.

**Fleet config after cutover.** Adds `--codellm-base-url` (and `--codellm-api-key` or mTLS material if codellm authenticates callers). `--allowed-programs`, `--sandbox`, `--sandbox-network`, `--max-stdout-bytes`, `--interpreters` move to codellm; fleet keeps them transitionally only for `exec` until Phase 5.

## Phased implementation

Each phase is independently shippable and reviewable (a candidate PR).

- **Phase 1 — protocol + service scaffold.** New `cmd/codellm` + `internal/codellm`. Implement `POST /v1/chat/completions` that extracts blocks and runs them **in-process** (reuse a copy of `coderun`/`runPipeline`), returning `tool_calls`. Ship with a golden-file conformance suite: request → response schema, `{changes}`→tool_calls mapping, error→422. **TDD applies strongly here** — the wire contract is the spec; write the golden request/response tests first. No fleet change yet.
- **Phase 2 — move sandbox + interpreters + debug surface.** Move `sandbox*.go`, `interpreters.json`, and the `/debug/*` endpoints into codellm; wire `MaybeRunSandboxChild()` first in `cmd/codellm/main.go`. Port `sandbox_linux_test.go` (TDD: sandbox behavior is security-critical). codellm now sandboxes each block and is steppable via the debug API.
- **Phase 3 — fleet cutover behind a flag.** Add `f.codeLLM` + the `InputBag` field on `agentruntime.Input` and its message builder. Build the code-role `Input` with `MaxSteps: 1` and a positive `MaxTokens` (above). Route `executor: code` through codellm when `--codellm-base-url` is set. Add **parity tests**: the same role note produces the same applied changes via the in-process path and via codellm (run both, diff `res.Changes`), **including the apply-error path** (a bad-`find` patch must fail both ways) and a test asserting the code-role `Input` has `MaxSteps == 1`. Roll out with the flag off, flip per-deploy.
- **Pre-Phase-4 audit.** Before deleting the in-process path, grep prod role notes for `env_passthrough:` / `env_prefix:` usage (the reconciler dry-run, or a one-off scan of the `roles/` folder). Any live role using them cannot run under codellm and would break silently on cutover — resolve each (redesign or keep in-process) before removal.
- **Phase 4 — remove the `executor: code` in-process branch.** Delete the `codeRunner` field (`fleet.go:24`) and the code branch in `execRole` (`handler.go:376`); `executor: code` now routes only through codellm. Fleet **still imports** `RunBlock`/`sandbox*.go` at this point — the `exec` tool depends on them (`runtime.go:384`) — so this phase removes the code-*role* path, not the packages.
- **Phase 5 (deferred) — move `exec` and finish the isolation.** Route the `exec` tool through codellm, then delete `sandbox*.go`/`coderun.go`/`runcode.go` from fleet, drop `MaybeRunSandboxChild()` from `cmd/fleet/main.go:36`, and remove the sandbox/program flags. Only now does fleet stop importing the code-exec/sandbox packages entirely; fleet no longer executes any code.

## Risks and open questions

1. **Determinism ≠ reproducibility.** The sandbox controls fs/net/resources but not wall-clock/PRNG. "Deterministic" means "no external nondeterminism," not bit-reproducible. If a consumer needs reproducibility, codellm must inject a fixed clock/seed — undesigned. *Open.*
2. **codellm is a code-execution oracle, and note content crosses to it.** An unauthenticated `/v1/chat/completions` is RCE-as-a-service; the fleet↔codellm channel must be authenticated (shared token / mTLS) or loopback/private-network only. Note content (`ChangedFiles`/`AttachedNotes`) transits the request, and a compromised codellm can write arbitrary content to any in-scope path (scope checks path, not bytes — `scope.go:141`). "Empty service" reduces the theft surface but does not eliminate content-exposure or content-integrity risk. *Must decide auth before exposing.*
3. **tool_call mapping fidelity + apply-error divergence.** `{changes}`→`write_note`/`patch_note` must be lossless (path/content/find/replace), `finish` must carry the answer and stay last (and always be present — the re-execution invariant), and the migrated role must declare the write tools + `write_patterns` or the loop's allowlist rejects codellm's calls (`runtime.go:180`). Separately, apply errors (bad-`find` patch) are a hard 502 today but a soft skip under the loop — decide and pin the canonical behavior (recommend codellm returns 422 to preserve hard-fail). Both covered by parity tests; the highest-churn correctness surface.
4. **Streaming over HTTP / timeouts.** The block→block pipeline stays inside codellm (good), but the fleet↔codellm call is one request/response held open for the whole run. The trip2g→fleet→codellm timeout chain must align: the role's `EffectiveTimeoutSeconds` (default 300, `role.go:89`) must bound codellm's run *and* fleet's HTTP client to codellm, or a long pipeline is cut mid-run. *Wire all three.*
5. **Small prod VM.** Another always-on service + container on a box already under memory pressure (see `prod_box_constraints`). The `RLIMIT_AS` default of 4 GiB (`sandbox.go:52`) is *virtual* address space, not RSS — a Node process reserves far more virtual than it resident-uses, so that number is not a direct OOM predictor and should not be read as "4 GiB per run." The real control is a **concurrency cap** on codellm (max simultaneous block runs) sized to the box's actual RAM, plus off-peak deploys; rlimits are a backstop, not the budget.
6. **How codellm is itself sandboxed/deployed.** The container is the real boundary; the in-process per-block sandbox needs unprivileged userns, which the container runtime may disable (see Isolation gotcha). Decide: permit userns in the container, or treat the container as the sole boundary with `besteffort`.
7. **System-prompt wrapper.** `agentruntime.Run` wraps the body in the micro-agent system prompt + "Begin." (`runtime.go:81`). codellm ignores the wrapper and scans for fenced blocks — works because the wrapper has no fences, but slightly hacky. Alternative: a dedicated `Input` mode that sends the raw body without the agent wrapper. *Minor; decide during Phase 3.*
