# codeexec: executor:code as an OpenAI-compatible endpoint

> **TL;DR / вывод.** Feasible and cheap. The sandbox runner (`internal/agentruntime.RunBlock` + the pipeline) is already a self-contained library with zero fleet dependencies, and the fleet debug endpoint (`internal/fleet/debug.go`) already proves the "run a code block over HTTP" shape. Build a new ~400-LOC `cmd/codeexec` Go binary that speaks `POST /v1/chat/completions`: the last message carries fenced code block(s), the response's `choices[0].message.content` is the sandboxed program's stdout. **Reading (a) — pure executor, no internal LLM** — is the right design; reading (b) (task → LLM writes code → run) already exists as a fleet LLM role and should not be duplicated as a fake "model". MVP ≈ 1–2 days. The hardest problems are container privileges for the namespace sandbox, honest error/timeout mapping onto completion semantics, and concurrency limits on the resource-tight prod box.

## 1. What executor:code is today

`executor: code` is a **role type**, not a tool: a role note whose frontmatter says `executor: code` skips the LLM loop entirely. The dispatch is in `internal/fleet/handler.go:374-402` (`execRole`): fleet Jet-renders the role body with the trigger context, then calls `agentruntime.RunCode` instead of `agentruntime.Run`. (Separately, LLM roles get an `exec` *tool* backed by the same runner — `AllowedPrograms`/`Sandbox` are also passed into `agentruntime.Run` at `handler.go:397-398` — but the "code role" is the deterministic, LLM-free path.)

The exact current contract (`internal/agentruntime/runcode.go:16-130`):

- **Input:** the Jet-rendered markdown body. `ExtractFencedBlocks` pulls every ` ```lang ` block (`coderun.go:345`); each block's program resolves from the fence label via the embedded `interpreters.json` registry (python, node, bash, ruby, php, perl — `coderun.go:20`, overridable with `--interpreters`), then is checked against the operator allowlist `--allowed-programs` (empty = code exec disabled, `runcode.go:134-153`).
- **Trigger data:** a JSON "delivery bag" written to a temp file and exposed as `$FLEET_INPUT` (`handler.go:227-236`, `coderun.go:171-178`). The child env is a hard-scrubbed allowlist: `PATH` + `FLEET_INPUT` + explicit `env_passthrough`/`env_prefix` only (`coderun.go:388-417`) — parent secrets (JWT, LLM keys) never leak.
- **Execution:** one block → `RunBlock` (`coderun.go:153`): per-run throwaway workdir, timeout, stdout/stderr capped (default 1 MiB). Multiple blocks → a true streaming pipeline, block *i*'s stdout piped into block *i+1*'s stdin via `io.Pipe` (`runcode.go:181-198`).
- **Sandbox (Tier-0, commits 43887814 + 54875402):** the daemon re-execs *itself* with the `__fleet-coderun-child` argv sentinel and hands a spec over an inherited pipe on fd 3 (`sandbox_linux.go:197-267`); the child sets up PID + mount (+ empty network unless `--sandbox-network`) namespaces, user namespace when unprivileged, a private `/proc`, rlimits (CPU/AS/FSIZE/NPROC/NOFILE), `NO_NEW_PRIVS` on all threads, Landlock FS confinement (workdir RW, minimal system dirs RO — `sandboxROPaths`, `sandbox_linux.go:273`), and a uid drop to 65534 when root. `native` mode **fails closed** (`sandbox.go:15-34`): if any of this can't be applied, the run is refused, never degraded.
- **Output:** the *last* block's stdout must be `{"changes":[...],"answer":"..."}` JSON; changes are applied through `ScopedKB(write_patterns)` (`runcode.go:97-129`) — code exec is not a scope bypass.

Two facts make the extraction easy:

1. `internal/agentruntime` imports nothing from `internal/fleet` — it is already a library. The only fleet-side glue is config plumbing and the KB.
2. **The HTTP shape already exists.** The loopback-only debug endpoint `POST /debug/run-block` (`internal/fleet/debug.go:129-208`) accepts `{body, block, stdin, input}` and returns `{stdout, stderr, ok, error, duration_ms}` — a raw code-exec-over-HTTP service, minus the OpenAI framing and minus the KB/changes contract (raw stdout, exactly what we want here).

## 2. Which reading: (a) pure executor

Two interpretations of "code execution as an OpenAI-compatible model":

- **(a) The endpoint receives CODE and returns its stdout.** No LLM inside. A deterministic "model" whose completion is the program output.
- **(b) The endpoint receives a TASK, an internal LLM writes code, the sandbox runs it, stdout comes back.** A hosted code-interpreter.

**Recommendation: (a).** Reasons:

- The owner's phrasing — "returns the sandboxed program's STDOUT as the completion output" — is literally (a).
- (b) already exists: a fleet LLM role with the `exec` tool *is* task→code→run→answer, with budgets, scope enforcement, and write-back. Rebuilding it inside a fake model discards all of that and hides an LLM spend behind an endpoint that looks free and deterministic.
- (a) composes into (b) for free: any agent (fleet, Claude Code, a retriever-style sidecar) that can write code can point its OpenAI client at codeexec and get a sandboxed interpreter. The LLM stays where the caller controls it.
- (a) is testable like the retriever: pure wire-contract tests, no model fixtures.

## 3. Wire design: `POST /v1/chat/completions`

### Request mapping

```json
{
  "model": "python",
  "messages": [
    {"role": "system", "content": "<optional stdin, see below>"},
    {"role": "user", "content": "```python\nimport sys, json\nprint(json.dumps({\"ok\": True}))\n```"}
  ]
}
```

- **`model` = program name** from the interpreter registry (`python`, `bash`, `node`, …), checked against `--allowed-programs` exactly like fleet. This is the natural home for the field: clients already select "models", and `GET /v1/models` lists the allowlisted programs — free discoverability with any OpenAI SDK.
- **Code = fenced blocks in the LAST message.** Multiple blocks → the existing streaming pipeline (stdout→stdin), same semantics as a multi-block role. If the last message has *no* fence, treat its entire content as source for `model`'s interpreter (ergonomic for programmatic callers that don't want to wrap in markdown). A fence lang that contradicts `model` resolves by fence (explicit wins), mirroring `resolvePrograms` (`runcode.go:134`).
- **stdin:** prior messages are NOT a conversation — this is not a chat model. MVP: ignore all but the last message. V2 option: a `system` message's content becomes the first block's stdin (covers the "pipe data in" case without inventing fields). The delivery-bag equivalent (`$FLEET_INPUT`) is an optional non-standard top-level field `input` (JSON object) — OpenAI SDKs pass unknown fields through `extra_body`.
- **Ignored params:** `temperature`, `top_p`, `n`, `presence_penalty`… silently ignored (deterministic executor). **Repurposed:** `max_tokens` → stdout cap override, clamped by the server cap; a non-standard `timeout_seconds` (via `extra_body`), clamped by the server ceiling.

### Response mapping

```json
{
  "id": "codeexec-8f2…", "object": "chat.completion", "model": "python",
  "choices": [{
    "index": 0,
    "message": {"role": "assistant", "content": "{\"ok\": true}\n"},
    "finish_reason": "stop"
  }],
  "usage": {"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0},
  "codeexec": {"exit_code": 0, "stderr": "", "timed_out": false, "duration_ms": 41, "blocks": 1, "sandbox": "native"}
}
```

- **`content` = raw stdout of the last block.** No `{"changes":…}` contract — that JSON envelope is fleet's KB write-back protocol and stays in fleet; codeexec is stdout-in-stdout-out.
- **`finish_reason`:** `"stop"` on exit 0; `"length"` when stdout hit the cap (semantically honest: output truncated). Non-zero exit and timeout are still HTTP 200 with `finish_reason: "stop"` and empty-or-partial stdout in `content`, with the truth in the `codeexec` extension block (`exit_code`, `stderr`, `timed_out`). Rationale: a failing *program* is a successful *execution* — same stance as `/debug/run-block`'s `OK=false` with HTTP 200 (`debug.go:41-42`). Callers that only read `content` get "what the program printed"; callers that care inspect `codeexec.*`. OpenAI clients ignore unknown top-level fields.
- **HTTP errors (OpenAI error object)** only for endpoint-level failures: unknown/disallowed program (400/404 model_not_found), sandbox refused in enforcing mode (500 — the code never ran), auth (401), over-capacity (429).
- **`usage`:** zeros in MVP (nothing is tokenized). Byte counts would be misleading in dashboards that price tokens.
- **Streaming:** `stream: true` is where this design shines later — stdout can be chunked to SSE as the process produces it (swap the `limitedBuffer` for a flushing writer). MVP: accept `stream: true` but reply with one content chunk + `[DONE]`, which every SDK tolerates.

## 4. Reuse: a `codeexec/`-style unit next to `retriever/`

The retriever precedent (`retriever/server.py` + `retriever/Dockerfile` + a compose service) is the right *packaging* shape but the wrong *language*: retriever is Python because the models are; the sandbox is Go and depends on re-exec'ing the *same binary* (`sandboxCommand` execs `os.Executable()`, `sandbox_linux.go:198`). So:

- **`cmd/codeexec/main.go`** — new binary, ~300–400 LOC: flag parsing (reuse `appconfig.EnvFlag` with a `TRIP2G_CODEEXEC_` prefix), `agentruntime.MaybeRunSandboxChild()` first line of `main` (mandatory — the sandbox re-exec lands back in this binary), the `/v1/chat/completions` + `/v1/models` + `/health` mux, bearer-token check, a concurrency semaphore.
- **`internal/agentruntime` reuse — one small export needed.** `RunBlock` is exported and sufficient for single blocks. The multi-block pipeline (`runPipeline`, `runcode.go:181`) is unexported and only reachable through `RunCode`, which drags in the KB + changes-JSON contract. Add one exported entry, e.g. `RunBlocks(ctx, blocks, programs, opts) (stdout, stderr, error)` — a thin wrapper over `runPipeline` with `KB`-free options — and make `RunCode` use it. Nothing moves, nothing is duplicated; fleet behavior is untouched.
- **`Dockerfile.codeexec`** — clone of `Dockerfile.fleet` (static Go build, alpine + the interpreters you allowlist). A compose service unit like `retriever`'s / `fleet`'s.
- **Fleet does NOT need to change** in MVP. A later optional step: teach fleet to delegate `executor:code` runs to a codeexec URL instead of running in-process (decouples interpreter deps from the fleet image), but that adds a network hop and a second failure domain — not worth it until something needs it.

### Deployment / privilege constraints (the honest part)

The Tier-0 sandbox needs kernel features that plain containers may deny:

- **Namespaces (user+PID+mount+net) and a private `/proc` mount.** Docker's default seccomp/caps profile can return EPERM on namespace creation — that is exactly why the `sandbox-tests` compose service runs `privileged: true` (`docker-compose.test.yml:515-531`) and CI runs the enforcement lane in a privileged container (`.github/workflows/ci.yml:55-72`). Note the tension: the e2e `fleet` service runs *un*privileged with `sandbox=native` and python allowed (`docker-compose.test.yml:468-508`) and the suite passes, so on current Docker/kernel the unprivileged-userns path evidently works there. **Verify per target host** — this is the #1 deployment risk. Options in order of preference: (1) run codeexec directly on the host / in a VM (fly.io Firecracker = root-in-VM, no outer seccomp — see `docs/dev/2026-07-02_fleet_sandbox_research.md` §"Fly"); (2) container with `security_opt: seccomp=unconfined` + `cap_add: SYS_ADMIN` (narrower than `privileged`); (3) full `privileged` as the sledgehammer.
- **Landlock** needs kernel ≥ 5.13 with the LSM enabled; the child probes the ABI and **refuses to run** when absent (`sandbox_linux.go:165-172`). On a kernel without Landlock, `native` mode means the endpoint returns 500 for every request — surface this in `/health` (probe once at boot, report `sandbox: native|unavailable`).
- **Fail-closed stance carries over:** codeexec should default to `--sandbox native` and refuse to boot into `off` without an explicit flag. An unsandboxed remote code-exec endpoint must be impossible to configure by accident.

## 5. Value and risk

**Unlocks:**

- Code execution becomes a network service any OpenAI SDK can call — fleet roles, Claude/Codex agents, the retriever pattern, curl. No per-client integration: `model: "python"`, message in, stdout out.
- The fleet role stops being the only consumer of the sandbox work; the Tier-0 hardening investment gets a second (and third) customer.
- LLM-judge/eval loops and DSPy-style harnesses (see the graph-search prompt-optimization plan) get a deterministic "tool model" they can mix into pipelines that only speak the OpenAI protocol.

**Risks (all manageable, none optional):**

- **It is an arbitrary-code-execution service by definition.** The sandbox confines the *child*; the *boundary* still needs: mandatory bearer auth (reject empty `--api-key` at boot, same spirit as fleet's required-config validation, `cmd/fleet/main.go:335`), and a default bind of loopback/internal docker network — public exposure must be an explicit, documented act.
- **Resource pressure on the prod box.** One small VM hosts every instance; rlimits are *per run*, not global. A concurrency semaphore (`--max-concurrent`, default 2–4) with 429 on saturation is part of MVP, not a follow-up. Default `MemoryBytes` is 4 GiB *virtual* per child (`sandbox.go:52`) — fine for AS, but N concurrent real 1-GiB residents would kill the box; cap concurrency, not just memory.
- **No multi-tenancy:** one allowlist, one sandbox policy, one API key. Fine for the box; per-key policies are out of scope.
- **Egress:** network stays off inside the sandbox by default; `--sandbox-network` is a global boot flag, not per-request (a per-request escape hatch would let any caller opt out of the strongest control).

## 6. Recommendation

**Worth doing, reading (a), MVP scope:**

1. `agentruntime.RunBlocks` export (pipeline without KB) + tests — ~0.5 day.
2. `cmd/codeexec`: `/v1/chat/completions` (non-streaming), `/v1/models`, `/health`, bearer auth, semaphore, env/flag config — ~1 day incl. contract tests (httptest, mock-free; sandbox integration reuses the existing `TestSandbox` lane).
3. `Dockerfile.codeexec` + compose service + a paragraph in `fleet_packaging.md` — ~0.5 day.

Defer: SSE streaming of stdout, stdin-via-system-message, `$FLEET_INPUT`-style input bags, fleet delegating its code roles to codeexec.

**Hardest problems, in order:**

1. **Container privileges vs fail-closed sandbox** — whether the target host's Docker/kernel lets an unprivileged container create user+mount namespaces and use Landlock. Must be probed per host; the fallback matrix (host-run / cap_add / privileged) is a deploy decision, not a code one.
2. **Error semantics that don't lie to OpenAI clients** — a completion API has no native slot for exit codes and stderr; the `codeexec` extension block + `finish_reason` mapping above is a judgment call that needs to be documented and kept stable.
3. **Boundary hardening on a shared, resource-tight box** — auth, bind address, and global concurrency limits are the actual security perimeter; the kernel sandbox is the *second* line.
