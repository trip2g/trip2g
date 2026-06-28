# Agent runtime — spec (phased)

Micro-agents over the trip2g knowledge base: a role is a note (its body is an instruction), there is one
thin runtime, and triggers are webhooks. This document specifies a phased implementation. Validate each
phase with a narrow test before the next.

## What already exists in trip2g (reuse / generalize, do not rebuild)
- **`cron_webhooks`** (`url, cron_schedule, instruction, secret, pass_api_key, read_patterns,
  write_patterns, max_depth, max_retries, next_run_at`) + `cron_webhook_deliveries`.
- **`change_webhooks`** (`include_patterns, exclude_patterns, instruction, secret, pass_api_key,
  on_create/on_update/on_remove, read_patterns, write_patterns, max_depth, max_retries`) +
  `change_webhook_deliveries` + `webhook_delivery_logs`.
  → trigger (scheduled / reactive by patterns) + prompt (`instruction`) + secret + scope
  (`read_patterns`/`write_patterns`) + an anti-cycle bound (`max_depth`) all exist.
- **`secrets`** (`key` unique, `value_crypt` blob, encrypted) — a generic secret store; the target for
  `secret_var_id` references.
- `federation_secrets`, `internal/dataencryption` (master-key AES-256-GCM).
- MCP: `search` / `note_html` / `graphql_request` (mutations) / `instructions` / `federated_*`.

Conclusion: most of the foundation exists. Missing pieces: an internal **executor of `instruction`**
(today a webhook POSTs to an external `url`; we want an internal LLM executor), **generalized secret
references**, and **observability**.

## Phase 1 — instruction executor (MVP)
**Goal:** a webhook's `instruction` is executed by an in-process LLM agent, scoped by patterns.
- New package `internal/agentruntime` (the per-note loop) + a thin `cmd/agent` that runs ONE note's loop
  once (the per-note executor — for manual runs / a demo). The long-running orchestrator daemon that
  manages MANY notes is `cmd/fleet` (Phase 3); both share the same `internal/agentruntime` loop.
- Reuses `internal/openai` (LLM client, configurable `BaseURL` = local or vendor model — the "give it an
  OpenAI-compatible endpoint" knob), `internal/case/mcp` (tools), and the note/model layer.
- Takes one delivery (`{instruction, read_patterns, write_patterns, secret, model}`) from a cron/change
  webhook.
- LLM loop: `instruction` + context (reads limited to `read_patterns`) → tool calls (`search`/`note_html`
  read; `graphql_request` write, limited to `write_patterns`) → until done or budget exhausted.
- **Non-overridable token hard cap** per run (the agent cannot raise it) — a safety floor.
- **Test (narrow):** one role on a cheap/local model. Verify: reads stay within `read_patterns`, writes
  stay within `write_patterns`, the cap stops the loop, `max_depth` bounds cascades. Run manually.

## Phase 2 — secret references (refactor + extract a common primitive)
**Goal:** keys are never stored in notes; a reference `secret_var_id` / `header_group_id`, values in the
secret store.
- **Generalize the existing** webhook `secret` + the `secrets` table into a shared resolver:
  `secret_var_id: "group.key"` (single) / `header_group_id: "..."` (a header group) in config/frontmatter
  → at tool-call / chart-query / external-request time the runtime resolves from `secrets`, decrypts
  server-side, and injects as auth headers/token. **The raw value never renders into a note and never
  enters the LLM context.**
- Admin: CRUD `secrets` + scan references → highlight "expected here / set vs missing".
- **Test:** a query with `header_group_id` → headers from the store → authorized request; the key is not
  visible in the note or the context. Independent of Phase 1.

### Optional: a deterministic transform mode (Jsonnet)
Besides the LLM `instruction`, a webhook can carry a deterministic **Jsonnet transform** (e.g.
`agent.jsonnet`) that builds a custom request to the destination — no LLM. trip2g/the box already uses
Jsonnet for patch rendering, so the mechanics are familiar. This is the cheap, deterministic path for
structured fetch/transform (pull a source, paginate, segment, map fields) where understanding is not
needed; the LLM path is reserved for steps that do need it.

**State between runs (handled by the existing mechanism).** A deterministic fetch needs a cursor /
high-watermark (e.g. `max_id`). The webhook already supports returning a new state and deleting marker
notes on completion, so the agent maintains the cursor itself: it reads the current high-watermark
(attached via `attach_notes`), fetches only items past it, processes them, then writes back the advanced
`max_id` (and removes processed markers) at the end — arming the next run. The cursor lives as a small
state note keyed by the webhook. The only care needed: advance the watermark **only after a successful
write** (at-least-once + dedup by item id keeps it idempotent on retries).

## Phase 3 — roles as notes
**Goal:** creating an agent = writing a note.
- A role note declares in frontmatter: `model`, `tools` (rights), `secret_var_id`/`header_group_id`,
  `cron_webhooks`/`change_webhooks` (creates the corresponding webhook from the note),
  `budget_tokens_per_run`, `mode` (sync/async). The note body is the prompt (`instruction`), or it points
  at a Jsonnet transform for the deterministic mode.
- The org config IS the set of role notes (no separate spec file). A privileged **topology-admin** role
  (admin scope) reconciles notes into webhook records + rights.
- **The fleet daemon (`cmd/fleet`) is the runtime form of this.** It watches an agents folder (the
  presence of a note = an agent exists — simplest), reconciles each note's frontmatter meta into
  cron/change webhook records, **idempotently registering itself and its hooks into the trip2g instance
  over the instance auth (a JWT), and deregistering them on shutdown.** When trip2g triggers a webhook,
  the fleet runs the per-note agent loop (`internal/agentruntime`) for that note. Add an agent = drop a
  note in the folder; remove the note = the fleet removes its hooks. This is what scales to dozens of
  micro-agents from one daemon.
- **A fleet exposes its own MCP toolset; each note trims it in frontmatter.** The fleet offers the full
  set of tools its machine can serve (optionally trimmed at fleet startup — the machine's capability, e.g.
  no shell or no external MCP on this fleet). Each agent note's frontmatter `tools:` then narrows that to
  the subset THIS agent may call. The note's `tools:` is a REQUIREMENT against the fleet's toolset: at
  startup the fleet validates every note's declared tools are within what it offers, and **fails fast /
  complains** if a note needs a tool the fleet cannot serve — no silently broken agents.
- **Multiple fleet sidecars over one shared KB.** Different fleets run on different machines, each
  watching a different agents folder of the same shared trip2g knowledge base. Each sidecar carries its
  own capabilities (which tools/resources it offers — a local model/GPU, egress, data access), so an
  agent is placed by folder onto the fleet that can serve it. Capability-based placement: the KB stays
  central, execution is sharded across machines.
- **Setup skill (non-expert UX) — a separate piece of work.** A non-expert does not write hundreds of
  notes by hand. The topology-admin / setup skill generates the topology from simple input: a list of
  subjects → a templated role note with scoped patterns + a schedule per subject. Template + a loop over
  the list. This is "turnkey" vs raw notes, and a product surface in itself.
- **Test:** 2-3 role notes; one role's `change_webhook` triggers another via a task note (delegation =
  creating a note). Verify per-agent scope/rights.

### `attach_notes`: payload attachment (+ gating as a side effect)
`attach_notes: ["glob"]` attaches the matched notes' **content + metadata into the webhook delivery
payload**, so both the Jsonnet transform and the LLM instruction receive them as context — no separate
fetch step. In the Jsonnet you can read the attached metadata and select deterministically (e.g. pick the
latest item by date) and build the outbound request from it; for the instruction path, the attached notes
are the context the agent operates on.

As a side effect it also acts as a **presence gate**: if nothing matches, there is nothing to attach, so
the run is skipped. That gives lightweight coordination:
- **Work-available:** `attach_notes: ["inbox/*.md"]` → fires (and attaches) only when there is something
  to process (no tokens burned idle).
- **Pipeline gate:** `attach_notes: ["stageA/*.done"]` → stage B runs only after A produced its output.
- **Semaphore / claim (mutual exclusion):** require-present + require-absent (do not fire if a `*.lock`
  note exists); acquire the lock by atomically moving/renaming the item note (a `graphql_request`
  mutation) so exactly one worker grabs it.
Composes with the existing `include_patterns`/`exclude_patterns` and feeds the influence/cycle graph
(Phase 4).

### Concurrency: no-overlap (singleton cron)
A cron webhook must not pile up on itself when a run outlasts its interval. Add a **`no_overlap` flag**
(a checkbox): when set, the dispatcher checks for an in-flight delivery of that webhook (the
`*_webhook_deliveries` tables already record deliveries — add a `running` status) and **skips** the new
fire while one is still running, logging "skipped: previous run in flight". Per-webhook choice: skip (the
common default), queue-one (run after the current finishes, at most one pending), or allow-overlap.
- **Stale-lock guard:** expire the `running` status by the webhook's existing `timeout_seconds` (or a
  heartbeat), so a crashed run does not block forever; a retry (`max_retries`) does not count as an
  overlap.
This is the server-side, race-free form. The userland equivalent is a note lock: a required-absent
`*.lock` via `attach_notes`, acquired at start and released on completion (the webhook can return a new
state / delete the marker on finish).

### Semaphore guarantees: defense-in-depth (and acceptance tests)
A lock alone is not enough. Use **two independent layers** so a single-layer failure cannot corrupt data:
- **Layer A — concurrency prevention:** server-side `no_overlap` (the dispatcher won't start a second
  delivery while one runs) + a userland atomic claim (acquire a lock note by moving/renaming the item,
  guarded by a required-absent `*.lock`).
- **Layer B — idempotency (the safety net):** dedup by item id, and advance the cursor / high-watermark
  ONLY after a successful write. So even if Layer A races or has a bug, Layer B still guarantees each item
  is written at most once and none is lost (effectively once = at-least-once + dedup).

Acceptance tests deliberately INJECT failures to prove each layer independently:
1. **Overlap:** fire the same cron webhook concurrently / on an interval shorter than a run → assert the
   second is skipped (`no_overlap`) and the lock prevents a double-claim.
2. **Race injection:** disable Layer A and race two workers on the same item → assert Layer B (dedup) still
   produces exactly one write, no duplicate.
3. **Crash / stale lock:** kill a run mid-processing while it holds the lock → assert the stale-lock guard
   (`timeout_seconds` / heartbeat) releases it, the next run proceeds, no deadlock, no corruption.
4. **Cursor:** partial failure → cursor not advanced → re-run reprocesses idempotently (0 duplicates);
   success → cursor advanced → only new items next run.
5. **Stress / property:** N overlapping, racing iterations → exactly-once effect (each item processed once,
   no duplicates, no skips).

## Phase 4 — observability + cycles
**Goal:** see who changes what, and prevent loops from burning tokens.
- Admin **"Agents"** view from `note_versions` (authorship + time) + `webhook_delivery_logs` (what / how
  much): "agent X changed Y, N tokens", grouped by agent and by file.
- **Influence graph:** static (from `change_webhooks.include_patterns` vs who writes there → edge A→B) +
  dynamic (from delivery logs). **Cycle detection** (DFS / SCC) → highlight A→B→A loops. `max_depth` is
  already a partial anti-cycle guard; the graph complements it (catches structure ahead of time; the cap
  catches spend after the fact).
- **Test:** deliberately create an A→B→A loop, verify `max_depth` + the graph catch it.

## Phase 5 — gRPC + Consul Connect mTLS (later, optional enterprise tier, feature-gated)
**Goal:** an mTLS mesh as an enterprise upgrade, not in the base install.
- gRPC as a second channel behind a clean seam (HTTP/MCP stays for dev). Consul Connect mTLS on top.
- Internal agent-to-agent triggers → gRPC (mTLS); external webhooks (third party) stay HTTP at the ingress.
- Enable only on the enterprise tier (mesh maintenance cost; the base relies on egress controls + a
  credential broker).

## Worked example: per-subject scoped profiles (validates the phases)
A neutral example that exercises the phases: a per-subject profile `profiles/<subject>/` (a vault) + AI
questions over it + access rights.
- **Phase 1** → a role answers questions over a subject's vault: `read_patterns: ["profiles/<id>/**"]`
  (reads only that profile), `write_patterns: []` (answer only; or a narrow write for an answer log). The
  Phase 1 test is exactly this Q&A — ask about a subject, get a grounded answer from its profile without
  escaping `read_patterns`.
- **One agent per subject (scale pattern):** one role note per subject,
  `read_patterns`/`write_patterns: ["profiles/<id>/**"]` → rights are scoped automatically (an agent
  cannot see others). A scheduled `cron_webhooks` cycle: ingest a source (e.g. a transcript) → update the
  profile → analyze → compare to prior periods. N subjects = N scheduled agents, light load. The agent
  ingests a source; it does not hold a live conversation.
- **Cross-subject aggregation/ranking** — a separate privileged aggregator role with broad read scope,
  not a per-subject agent. Access is critical here → top tier only.
- **Ingestion (e.g. segment a transcript by topic and file each segment):** a role with `write_patterns`
  scoped to the target profile — or, for purely structured pulling, the deterministic Jsonnet transform
  mode (no LLM). Just a role + an endpoint.

## Order
Phases 1 and 2 are independent (executor and secret refs), tested separately. Phase 3 builds on 1.
Phase 4 on 1-3. Phase 5 is a separate enterprise track. Validate each phase with a narrow test before the
next.

## Security (across all phases)
- **A shell / terminal tool is the apex privilege:** it subsumes the `tools` allowlist (with `curl` you
  can do anything). The boundary for a shell-capable agent is therefore **egress controls + a scoped,
  short-lived credential (broker pattern, not the master key) + the risk tier + freeze-and-report**, not
  the tool list. A coordinator does not need a shell (it moves notes); shells belong to leaf executors
  behind the egress boundary.
- **Budget:** smart allocation is itself a role (a budget-coordinator that spawns siblings by writing task
  notes); the code holds only a non-overridable hard cap (a kill switch the agent cannot rewrite).
- **Anti-runaway:** `max_depth` (exists) + the cycle graph (Phase 4) + the budget cap.
- **Model split:** orchestration is structured, so a cheap/local model can run the cluster (near-zero
  inference cost); strong vendor models are interchangeable executors on the leaves, on sanitized
  subtasks.
