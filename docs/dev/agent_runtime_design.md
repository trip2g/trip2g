# trip2g Agent-Runtime — Unified Design & Implementation Spec (Phases 1–4 + first-class Jsonnet)

## 1. TL;DR / Вывод

We are building a runtime where **notes trigger agents on file edits**: editing a note fires an agent that reads context, decides, and writes notes back. The **spine** is the existing reactive change-webhook path made in-process — `change_webhook` note-match → background delivery job → `agentruntime.Run` (the Phase-1 executor already in this worktree) → write-back through the canonical `updateNotes` path → `noteChanges` SSE. The **kanban board** (shipped `trip2g/kanban_template`) is the live control surface and the **acceptance test**: drag a card → underlying note edited → agent decides → surgical write-back → board re-renders over SSE.

The whole thing is built by **reuse, not rebuild**: one new `executor_mode` discriminator branches the existing delivery jobs; everything else (depth guard, scope globs, secret decryption, retry/logging, jsonnet VM, MCP tools) is already in the codebase. The critic surfaced three lethal contradictions — a three-way KB fork, an overridable budget cap, and a mode-column naming clash — all resolved here into **one KB (`internal/notekb`, write-through-via-`updateNotes` with single deferred apply), a non-overridable server cap (no DB column), and one `executor_mode` column**. Migrations are consolidated into **at most 3 user-confirmed files**. Build order front-loads the kanban vertical slice as the permanent regression oracle.

---

## 2. Goal & the Kanban Demo (end-to-end acceptance)

**The north star, concretely.**

**Note layout (seed under `docs/demo/`):**
- `boards/sprint.md` — the kanban board note. Cards are list items with a status field; the board renders from this note and persists card moves via the canonical `updateNotes` **surgical patch** (preserving unmodeled card metadata).
- `roles/triage.md` — the **role/policy note**: read-scoped instruction the agent follows ("when a card moves to `Review`, set its reviewer field; return only the patched lines"). Editing this note changes agent behavior — the on-theme expression of "notes drive agents."

**The wiring:** one `change_webhook` with
```
include_patterns = ["boards/sprint.md"]
read_patterns    = ["boards/**", "roles/**"]
write_patterns   = ["boards/**"]
executor_mode    = "instruction_llm"
on_update        = true        # only on update
max_depth        = 1           # agent's own write does not re-trigger
model            = "gpt-4o-mini"
```

**The flow (file:line anchors):**
1. User drags a card → React kanban saves via `updateNotes` surgical patch (`internal/case/updatenotes/resolve.go`, patch branch ~66-96), producing a new version of `boards/sprint.md`.
2. `updateNotes` calls `app.HandleLatestNotesAfterSave` (`cmd/server/webhooks.go:16`) → (a) `PublishNoteChanges` to the notebus → SSE (`webhooks.go:73-89`), (b) `handlenotewebhooks.Resolve(ctx, changes, depth=0)` (`webhooks.go:92`).
3. `handlenotewebhooks.Resolve` (`internal/case/handlenotewebhooks/resolve.go:114`): depth `0 < 1` passes (`:132`), `matchChange` matches `boards/sprint.md` (`:85`), `InsertWebhookDelivery` + enqueue.
4. The goqite worker runs `deliverchangewebhook.Resolve` (`internal/case/backjob/deliverchangewebhook/resolve.go:46`). The **new branch** on `wh.ExecutorMode == "instruction_llm"** runs `agentruntime.Run` in-process with the `notekb` KB.
5. The agent reads `boards/sprint.md` + `roles/triage.md`, decides, calls `patch_note` → buffered → applied **once** after the run via `updateNotes` (surgical patch) through the **canonical apply+dispatch helper** with `WebhookDepth=1`.
6. `updateNotes`' own reload + `HandleLatestNotesAfterSave` publishes the second `noteChanges` event → the kanban board re-renders the card live.
7. Re-entry at `depth=1 >= max_depth=1` → guard skips (`:132`) → no self-retrigger. Loop closed.

**Observe via:** the `noteChanges` SSE stream (the same subscription `assets/ui/user/live/live.view.ts` consumes). Acceptance = two SSE `update` events (user's move, then agent's write-back) and the card reflecting the agent's decision.

---

## 3. Architecture Overview

### The executor MODE seam
A single new column `executor_mode` on `change_webhooks` / `cron_webhooks` discriminates **how the delivery job obtains its changes**, branched at the top of `deliverchangewebhook.Resolve` (and the cron twin) after `WebhookByID`:

| `executor_mode` | Behavior | LLM? | Egress |
|---|---|---|---|
| `external_url` (default) | **Today's code, byte-for-byte** — `webhookutil.Deliver` HTTP POST | external | HTTP to webhook URL |
| `instruction_llm` | `agentruntime.Run` in-process; instruction = role note body | yes | LLM BaseURL only |
| `jsonnet` | deterministic transform via `internal/agentjsonnet`; no LLM | no | optional fetch (SSRF-guarded) |

The branch is the **only** structural change to dispatch — secret loading, retry, status, logging, and the apply path are all shared.

### The KB / tool seam (in-process vs HTTP fleet)
`agentruntime.KB` (Phase 1) is the **single transport-agnostic seam**. The same `Run` loop drives:
- **In-process** (`internal/notekb.NoteKB`, Phases 1–2) — reads from `app.LatestNoteViews()`, writes via the in-process `updateNotes` mutation.
- **Remote fleet** (`internal/fleet.RemoteKB`, Phase 3) — same interface over HTTP-MCP (`/_system/mcp`) with a scoped `shortapitoken` Bearer.

`FileKB` (`internal/agentruntime/filekb.go`) stays for the `cmd/agent` one-shot harness.

### Reuse of existing infra
- **Webhooks** are the agent identity and trigger machinery (no new `agents` table).
- **MCP** tools and `graphql_request` admin path are the leaf tool surface; coordinators get KB tools + scoped read-only.
- **Secrets** reuse the `change_webhooks:<id>:` prefix + `dataencryption` AES-256-GCM; Phase 2 generalizes to header-injection.
- **Jsonnet** reuses `internal/frontmatterpatch`'s VM (`NewVM`, `MaxStack=500`) extracted into `internal/jsonneteval`.

---

## 4. Reuse Map vs Net-New

### Lean on (existing — do NOT rebuild)
| Concern | File |
|---|---|
| Executor loop + token hard-cap + Result.Changes | `internal/agentruntime/runtime.go` (Run :79, cap :113, Changes :215) |
| Scope globs (read/write deny) | `internal/agentruntime/scope.go` (`ScopedKB`, `MatchesAny`) |
| LLM seam + model split via BaseURL | `internal/agentruntime/llm.go`, `openai_llm.go:19-25` |
| Change dispatch + depth guard | `internal/case/handlenotewebhooks/resolve.go:114,132,85` |
| Delivery job (secrets/retry/status/apply) | `internal/case/backjob/deliverchangewebhook/resolve.go` (`applyAgentChanges` :265, `loadSecrets` :206) |
| Cron twin | `internal/case/backjob/delivercronwebhook/resolve.go` |
| SSE + cascade fan-out | `cmd/server/webhooks.go:16,73-89,92` |
| Canonical write (upsert/patch, create-only sentinel) | `internal/case/updatenotes/resolve.go:45-96` |
| Depth/actor stamping on bg ctx | `internal/appreq/request.go:269` (`NewContext`) |
| Scoped broker credential | `internal/shortapitoken/token.go` |
| Secret decrypt | `internal/dataencryption`, `internal/case/admin/getsecret` |
| Jsonnet VM | `internal/frontmatterpatch/evaluate.go:11` |
| HTTP egress (SSRF-safe, 1 MB cap) | `internal/webhookutil/httpclient.go:27,73` |
| Change contract | `internal/webhookutil/agentresponse.go` (`AgentChange`) |
| Service-package exemplars | `internal/chartdata`, `internal/gitapi` |

### Net-new
| Component | Phase |
|---|---|
| `internal/notekb` (the unified KB) | 1 |
| Canonical apply+dispatch helper (in delivery job) | 1 |
| `cmd/server/agentruntime.go` (app LLM/cap providers) + `appconfig.AgentRuntime` | 1 |
| `internal/secretref` (header-injection resolver) | 2 |
| `internal/jsonneteval` + `internal/agentjsonnet` | 2 (jsonnet) |
| `internal/fleet` + `cmd/fleet` + `internal/case/admin/reconcilefleet` | 3 |
| `internal/agentgraph` + Agents-overview admin view | 4 |

---

## 5. Per-Phase Design

> **STEP 0 (blocker, resolves the critic's three-way KB fork): the unified KB + the canonical apply helper.** Both land in Phase 1 and are consumed unchanged by all later phases.

### The unified KB (critic resolution — replaces A's `agentkb` and B's immediate-write `notekb`)

**One package `internal/notekb`. `Write`/`Patch` BUFFER in memory** (so within-run reads see the agent's own edits); persistence happens **exactly once after `Run`** via `Result.Changes`, routed through the **`updateNotes` mutation** (not raw `InsertNote`). This simultaneously: preserves unmodeled kanban content (surgical patch), keeps one persistence point (no double-write), and fires SSE+cascade via `updateNotes`' own reload+after-save.

`AgentChange` is extended with a surgical-patch shape so the buffered write can replay as a patch:

```go
// internal/webhookutil/agentresponse.go — extend the existing contract
type AgentChange struct {
    Path         string
    Content      string // full-content upsert (Kind=="" or "upsert")
    Find         string // surgical patch
    Replace      string
    ExpectedHash string // optimistic concurrency
    Kind         string // "" | "upsert" | "patch"
}
```

```go
// internal/notekb/notekb.go — buffering KB over the real note layer
type Env interface {
    SearchLatestNotes(query string) ([]model.SearchResult, error)
    LatestNoteViews() *model.NoteViews
    CanReadNote(ctx context.Context, n *model.NoteView) (bool, error)
}
type NoteKB struct{ env Env; writes map[string]webhookutil.AgentChange }

func (k *NoteKB) Read(ctx context.Context, path string) (string, error) {
    if w, ok := k.writes[path]; ok { return w.Content, nil } // overlay first
    nv := k.env.LatestNoteViews().PathMap[path]
    if nv == nil { return "", agentruntime.ErrNotFound }
    if ok, _ := k.env.CanReadNote(ctx, nv); !ok { return "", agentruntime.ErrReadDenied }
    return string(nv.Content), nil
}
func (k *NoteKB) Patch(ctx context.Context, path, find, replace string) error {
    k.writes[path] = webhookutil.AgentChange{Path: path, Find: find, Replace: replace, Kind: "patch"}
    return nil // buffer only
}
var _ agentruntime.KB = (*NoteKB)(nil)
```

Read isolation is **two gates**: `ScopedKB` globs (inside `Run`) **and** `CanReadNote` (subgraph/private), mirroring `internal/case/mcp/resolve.go:512,584`.

### The canonical apply+dispatch helper (critic resolution — single owner of the appreq/depth/double-write fix)

Owned by the reactive-trigger component; **every later phase consumes it** rather than re-describing it.

```go
// internal/case/backjob/deliverchangewebhook/resolve.go
func applyAgentResult(ctx context.Context, env Env, wh db.ChangeWebhook,
    depth int, changes []webhookutil.AgentChange, writePats []string) error {

    // Seed appreq on the bg-job ctx: depth+1 (bounds cascade) + actor (attribution).
    ctx = appreq.NewContext(ctx, &appreq.Request{
        WebhookDepth:     depth + 1,
        AdminActorUserID: int(wh.CreatedBy),
    })
    for _, c := range changes {
        if len(writePats) > 0 && !webhookutil.MatchesAny(c.Path, writePats) {
            return fmt.Errorf("path %q not in write_patterns", c.Path) // defense-in-depth
        }
        if err := env.UpdateNotesChange(ctx, c); err != nil { return err } // upsert|patch
    }
    return nil // updateNotes' own reload + HandleLatestNotesAfterSave fires SSE + cascade
}
```

This single function closes the critic's unowned defects: background-job ctx has no appreq (→ wrong-depth cascade + lost admin/attribution) and the A/B double-write fork.

---

### Phase 1 — reuse + reactive wiring + trip2g KB

**Changes (files):**
- `internal/case/backjob/deliverchangewebhook/resolve.go` — `switch wh.ExecutorMode` branch; `runInstructionLLM`; the `applyAgentResult` helper above; Env additions.
- `internal/case/backjob/delivercronwebhook/resolve.go` — identical branch; cron stamps `WebhookDepth=1` (matches `resolve.go:105`).
- `internal/notekb/notekb.go` + `notekb_test.go` (+ `mocks_test.go`).
- `cmd/server/agentruntime.go` — `AgentLLM()/AgentDefaultModel()/AgentMaxTokens()/AgentMaxSteps()`; build one shared `OpenAILLM` at boot; compile checks.
- `internal/appconfig/config.go` — nested `AgentRuntime` config + flags.
- `internal/model/secret_prefix.go` (or sibling) — `ExecutorModeExternalURL/InstructionLLM/Jsonnet` constants.
- `queries.write.sql` — add columns to webhook insert/update; `make sqlc` (reads use `select *`).
- **Demo-blocking** (critic gap): `internal/case/admin/{create,update}webhook` + `schema.graphqls` accept `executorMode/model/agentJsonnet` (so the demo webhook can be created via CRUD); `make gqlgen` + `npm run graphqlgen`.
- Seed: `docs/demo/boards/sprint.md` + `docs/demo/roles/triage.md` + one `instruction_llm` webhook (admin call).

**Key interface (mode branch):**
```go
switch wh.ExecutorMode {
case model.ExecutorModeInstructionLLM:
    return runInstructionLLM(ctx, env, wh, params)
case model.ExecutorModeJsonnet:
    return runJsonnet(ctx, env, wh, params)
default: // "" or external_url — existing HTTP path unchanged
}

func runInstructionLLM(ctx context.Context, env Env, wh db.ChangeWebhook, params DeliverChangeWebhookParams) error {
    readPats, _  := webhookutil.ParseJSONStringArray(wh.ReadPatterns)
    writePats, _ := webhookutil.ParseJSONStringArray(wh.WritePatterns)
    modelName := wh.Model; if modelName == "" { modelName = env.AgentDefaultModel() }
    res, err := agentruntime.Run(ctx, agentruntime.Input{
        Instruction:   buildInstruction(wh.Instruction, params.Changes),
        ReadPatterns:  readPats, WritePatterns: writePats,
        Model:    modelName,
        MaxTokens: env.AgentMaxTokens(), // SERVER cap, never a column
        MaxSteps:  env.AgentMaxSteps(),
        LLM: env.AgentLLM(), KB: notekb.New(env),
    })
    // log res|err to webhook_delivery_logs (kind="change", response_body=JSON(res))
    if err != nil { return err }
    return applyAgentResult(ctx, env, wh, params.Depth, res.Changes, writePats)
}
```

**Narrow gate (must pass before Phase 2):** `deliverchangewebhook.Resolve`, `mode=instruction_llm`, stub LLM scripted `[read_note(boards/sprint.md) → patch_note(...) → finish]`: assert exactly one `UpdateNotesChange` (patch shape), `HandleLatestNotesAfterSave` re-entry ctx has `WebhookDepth==params.Depth+1`, delivery `success`, log written. Plus the `external_url` regression case (still `webhookutil.Deliver`).

---

### Phase 2 — secret references (header-injection, zero-migration)

Generalize the prefix+decrypt pattern into `internal/secretref` (Service Package). `secret_var_id` `"group.key"` → secret keyed literally `group.key`; `header_group_id` `"id"` → secret keyed `headergroup:<id>` whose decrypted value is JSON `{header:value}`. **The raw value lives only in the in-memory headers map at HTTP egress** — never in a note, an LLM `Message`, or the delivery log (`InsertWebhookDeliveryLog` persists `request_body`/`response_body`, **not** headers).

**Critic resolution:** keep the **zero-migration prefix convention**; drop G's `secret_vars`/`header_groups` tables. Migrate the existing `payload.Secrets`-in-`request_body` path to header injection **in the same change** so logs stop leaking. For `instruction_llm` (no egress) secrets simply never enter the loop.

```go
// internal/secretref/resolver.go
type Env interface { GetSecretValue(ctx context.Context, key string) (string, error) }
type Resolver struct{ env Env }
func (r *Resolver) ResolveVar(ctx context.Context, ref string) (string, error)
func (r *Resolver) ResolveHeaderGroup(ctx context.Context, id string) (map[string]string, error)
func (r *Resolver) MaterializeHeaders(ctx context.Context, spec RequestSpec) (map[string]string, error) // fail-closed
```

`internal/model/secret_ref.go`: `HeaderGroupSecretKey(id)="headergroup:"+id`; `ValidateSecretVarID` **rejects `:`** (key-confusion guard against namespaced keys).

**Files:** `internal/secretref/{resolver.go,*_test.go}`, `internal/model/secret_ref.go`, `cmd/server/secretref.go` (embed `*Resolver` on app), `internal/case/admin/scansecretreferences` (admin view: `set|missing` diff, never decrypts), `schema.graphqls` (`scanSecretReferences`).

**Narrow gate:** store `headergroup:test={"Authorization":"Bearer SECRET123"}`; `MaterializeHeaders` → POST to httptest asserts the header == `Bearer SECRET123`; assert `SECRET123` appears in **none** of: written note content, the recording-LLM `messages` slice, the delivery log `request_body`.

---

### Jsonnet mode (first-class, lands with Phase 2)

`executor_mode="jsonnet"` runs a deterministic no-LLM transform: build an outbound request, map the response to changes, advance a cursor.

- **`internal/jsonneteval`** — `NewVM()` (MaxStack=500) + `EvalJSON(vm, src, extVars)` + `Validate`. **`internal/frontmatterpatch` is refactored to reuse it** (existing `patch_test.go`/`vault_test.go` are the regression guard).
- **`internal/agentjsonnet`** — pure: `Build(source, Inputs) (Envelope, error)` and `Run(ctx, Driver, RunInput) (RunResult, error)` (bounded fetch/transform loop, `MaxSteps`-guarded, mirrors `agentruntime.Run`). I/O (HTTP via `webhookutil.Send`, secret resolution, note writes) lives in the delivery job via an injected `Driver`.

```go
type Driver interface {
    Send(ctx context.Context, req Request, headerValues map[string]string) (Response, error)
    ResolveSecretRefs(ctx context.Context, refs map[string]string) (map[string]string, error)
}
```

**Critic resolutions applied:** jsonnet source column is **`agent_jsonnet`** (one name); secrets reach jsonnet **only as a sorted list of ref names** (`secret_refs` extVar, values stripped); cursor is stored as a **note** (`cursor_path`, advances only inside one atomic batch with the changes); `notes` input uses the **`attach_notes`** column landed in M1 (no degenerate fallback). `webhookutil.Deliver` is generalized into a method-aware `Send` (existing POST callers unchanged).

**Narrow gate:** `mode=jsonnet`, LLM nil/never invoked; deterministic `Build` picks latest-by-`updated_at` and emits the expected request; `Run` with stub Driver maps response → one change + new cursor; error path → no cursor advance.

---

### Phase 3 — roles-as-notes + fleet + attach_notes + no_overlap + idempotency + setup skill

An agent is a **role-note**: frontmatter declares model/tools/secret-ref/triggers/budget/mode; body is the instruction. A new **`cmd/fleet`** daemon watches a KB folder, parses role notes, validates `tools ⊆ offered toolset` (**fail-fast**), and idempotently registers **ordinary** change/cron webhooks pointing at its own callback URL. trip2g drives the fleet through the **unchanged `external_url` delivery path** — the fleet is a well-behaved external target running `agentruntime.Run` per delivery.

**Server changes (small, additive):**
- `attach_notes` — **single TEXT(JSON-array of globs) column** (critic resolution; drop G's table). `!`-prefix = require-absent. **Presence-gate in the dispatcher** (`handlenotewebhooks.Resolve` before `InsertWebhookDelivery`); **materialization in the delivery job** into `{path,title,content,updated_at,tags,meta}`.
- `concurrency_mode` enum `{allow_overlap(default), skip, queue_one}` — race-free via one **conditional INSERT** on the serialized writer (`InsertWebhookDeliveryIfClear`).
- Delivery `started_at`/`heartbeat_at` + the existing free-text `status` gains value `running` (no DDL for the value). Stale-lock self-heal predicate `coalesce(heartbeat_at,started_at,created_at) >= now - timeout_seconds`. Janitor cron `expirestalewebhookdeliveries`.
- `managed_by` reconcile key `fleet:<id>:<note_path>#<n>` (`''` = human-managed, never touched).

**Exactly-once (two independent layers):**
- **Layer A** — server `no_overlap` conditional-insert + userland atomic claim (`updateNotes` `expectedHash`, create-only sentinel `:45-66`).
- **Layer B** — dedup by item id + cursor advanced **only after** a successful write (cursor as a state note delivered via `attach_notes`).

```go
// internal/agentruntime/runtime.go — additive, backward-compatible
type Input struct {
    /* existing */
    Tools []string // per-note allowlist; empty = current default tool set
}
```

**Critic resolutions:** fleet credential = **one admin/topology-scoped credential type** (extend `checkapikey` elevation); until then `reconcileFleet` **fails closed** (no long-lived broad admin JWT). SSRF = **per-webhook egress allowlist** (allowlist the sidecar host) — **not** a global `ssrfsafe` disable. Setup/turnkey skill (templated role notes from a subject list via `updateNotes` upsert) is a **separate, non-blocking deliverable**.

**5 acceptance tests (deterministic, mocked-LLM, table-driven):**
1. **Overlap** — `concurrency_mode × in-flight state`: fresh `running` ⇒ `skip` inserts no row; `allow_overlap` always inserts; `queue_one` inserts only if no pending.
2. **Race injection** — disable Layer A, two deliveries for the same item; exactly one `ClaimNote` wins (one `HashMismatch`), one output.
3. **Crash/stale lock** — `running` older than `timeout_seconds`, no heartbeat ⇒ treated as not-in-flight; janitor finalizes orphan to `failed`; retry reusing `delivery_id` not counted as overlap (injected clock).
4. **Cursor partial-failure** — fail after item 2/4 ⇒ cursor not advanced; re-run idempotent, 0 duplicates.
5. **Stress / exactly-once property** — N items, M>N racing iterations with A+B on ⇒ output set == input set exactly.

**Setup skill (separate):** `cmd/fleet seed` — from a subject list + template, emit N scoped role notes (`<id>` placeholder) via `updateNotes` upsert.

---

### Phase 4 — observability (Agents view) + cycle graph

Tag agent-authored `note_versions` with the delivery that wrote them via the **existing actor pipe** (`model.NoteActor` + `insertnote.Resolve:91-100` + `cmd/server/notes.go:116`), extended with `created_by_delivery_kind` + `created_by_delivery_id` (the `(kind,id)` idiom from `webhook_delivery_logs`, since change/cron are two tables one FK can't span). Persist `Result.TokensUsed` on the delivery row (`tokens_used`, NULL = unknown/legacy-external).

**`internal/agentgraph`** (pure, offline-testable): `BuildGraph(agents, authored) Graph` + `DetectCycles(Graph) []Cycle` (Tarjan SCC + self-loops). Static edge A→B when `patternsOverlap(A.WritePatterns, B.IncludePatterns)`; dynamic edge when an authored path of A `MatchesAny B.IncludePatterns` (exact, same matcher as the dispatcher).

One admin query `admin.agentsOverview` → a `$mol` view (mirrors `assets/ui/admin/changewebhook/catalog`) listing agents/files/tokens with flagged cycles. `max_depth` bounds spend after the fact; the graph catches cyclic structure ahead of time — **advisory, never the runtime bound.**

**Acceptance:** two webhooks A(include a.md, write b.md, max_depth=2) and B(include b.md, write a.md); edit a.md; mocked LLM always writes the other file. Assert (1) cascade is bounded (deliveries finite, guard at `:132`); (2) `DetectCycles` flags `{A,B}` (static + dynamic).

---

## 6. Consolidated Migration List (confirm before any file is created)

> Per CLAUDE.md **always-ask** rule, nothing below is created without explicit user confirmation. Distinct timestamps, dependency order. The critic's version-collision (three migrations sharing `20260628120000`) and duplicate-column issues are resolved by merging into **3 files**.

### M1 — demo-critical (Phase 1 + jsonnet + attach) — `2026062812**00**00`
| DDL | Reason |
|---|---|
| `ALTER change_webhooks ADD executor_mode TEXT NOT NULL DEFAULT 'external_url'` | the one mode discriminator; default preserves today's HTTP behavior |
| `ALTER change_webhooks ADD model TEXT NOT NULL DEFAULT ''` | per-role model (cheap triage vs strong) |
| `ALTER change_webhooks ADD agent_jsonnet TEXT NOT NULL DEFAULT ''` | deterministic transform source (one name) |
| `ALTER change_webhooks ADD attach_notes TEXT NOT NULL DEFAULT '[]'` | read-context globs + `!`require-absent gate; landed early so jsonnet uses it |
| `ALTER change_webhooks ADD cursor_path TEXT NOT NULL DEFAULT ''` | jsonnet/exactly-once watermark note |
| *(same 5 on `cron_webhooks`)* | cron twin |

### M2 — exactly-once + observability (Phase 3 + 4 delivery columns, de-duplicated) — `2026062812**01**00`
| DDL | Reason |
|---|---|
| `ALTER change_webhooks ADD concurrency_mode TEXT NOT NULL DEFAULT 'allow_overlap' CHECK (...)` | no_overlap policy |
| `ALTER change_webhooks ADD managed_by TEXT NOT NULL DEFAULT ''` | fleet reconcile key; `''`=human |
| `ALTER change_webhook_deliveries ADD started_at datetime` | run start (stale-lock + duration) |
| `ALTER change_webhook_deliveries ADD heartbeat_at datetime` | long-run liveness |
| `ALTER change_webhook_deliveries ADD tokens_used integer` | Phase-4 spend (NULL=unknown) |
| `ALTER change_webhook_deliveries ADD steps integer` | Phase-4 tool-loop steps |
| `CREATE INDEX idx_change_webhook_deliveries_inflight ON (...webhook_id, status)` | fast no_overlap lookup |
| `CREATE INDEX idx_change_webhooks_managed_by ON (managed_by)` | reconcile diff + bulk deregister |
| *(cron twins of all above)* | cron parity |

*(No DDL for `running` — the `status` TEXT column has no CHECK.)*

### M3 — attribution (Phase 4) — `2026062812**02**00`
| DDL | Reason |
|---|---|
| `ALTER note_versions ADD created_by_delivery_kind text` | link version → delivery → agent |
| `ALTER note_versions ADD created_by_delivery_id integer` | the `(kind,id)` pair |
| `CREATE INDEX idx_note_versions_delivery ON (created_by_delivery_kind, created_by_delivery_id)` | per-file aggregates |

**Explicitly dropped (critic resolutions):** `max_tokens`/`max_steps` columns (would defeat the non-overridable cap); `secret_vars`/`header_groups`/`webhook_secret_vars` tables (E's zero-migration prefix wins); G's `attach_notes` table; `parent_delivery_id`/`root_delivery_id`/`frozen_reason` columns are **deferred** — the Phase-4 cycle graph is computed from existing data; revisit only if deep lineage analytics are needed. Down-migrations use SQLite ≥3.35 `DROP COLUMN` (consistent with `20260627000000`).

After each: `make sqlc`; after schema edits `make gqlgen` + `npm run graphqlgen`. **Codegen is serialized per phase** to avoid the concurrent-regen conflicts the critic flagged.

---

## 7. Security & Budget Model

**The boundary is NOT the tool allowlist.** A shell tool subsumes any allowlist. The real containment is four things:

1. **Scoped broker credential** — `shortapitoken` (Depth + Read/WritePatterns HS256 JWT), minted per-delivery, never the master key (`deliverchangewebhook/resolve.go:88`). Consider shrinking the 60-min TTL floor for fast reactive runs.
2. **Egress shape, bounded by construction (Phases 1–4)** — the in-process executor exposes only `search/read_note/write_note/patch_note/finish`, **no shell**. The only outbound reach is the per-role LLM `BaseURL` and (jsonnet/external_url) the SSRF-guarded HTTP path. **A `RiskTier` seam** (`policy.go`) is added now; `TierLeafShell` is reserved for the out-of-scope Phase 5.
3. **Non-overridable cap** — `Input.MaxTokens` comes from **appconfig, never a DB column** (critic resolution: do not add `max_tokens`/`max_steps`). If per-role budgets are ever needed, clamp `effective = min(column_or_default, serverCap)`. Checked before each model call (`runtime.go:113`). **Bounded overshoot:** the check is pre-call, so a single response can exceed the cap by one model response — set `max_completion_tokens` per call to make it a true wall, and have Phase-4 dashboards treat the cap as ceiling+one-response, not equality.
4. **max_depth + cycle graph** — `max_depth` (`handlenotewebhooks:132`, propagated as Depth+1 by the canonical helper) bounds the cascade after the fact; the Phase-4 cycle graph catches A→B→A structure ahead of time.

**Apex shell / credential broker:** secrets resolve server-side at egress into an in-memory headers map (`secretref.MaterializeHeaders`), never into a note, an LLM `Message`, or the delivery log. The existing `payload.Secrets`-in-`request_body` leak is fixed (migrated to header injection) in Phase 2.

**Read isolation:** two gates — `ScopedKB` globs + `CanReadNote` (subgraph/private). A scoped agent never sees a foreign-glob or private note (leak-tested against the recording LLM stub).

**Write elevation caveat:** `updateNotes` elevates to admin inside `GraphQLRequest`; bound it by `write_patterns` (re-checked in `applyAgentResult`) and run the agent with a **role-vetted, non-empty** write scope — empty `write_patterns` + admin = write-anywhere.

**Model split:** cheap/local orchestrator + strong/vendor leaf, interchangeable via per-role `BaseURL`.

**Demo storm guard (critic gap — before enabling on the undersized box):** default auto-triggered roles to `max_depth=1`, `on_update`-only, tight include/exclude, `gpt-4o-mini`, low `max_steps`; run `instruction_llm` deliveries on a bounded queue with a per-webhook delivery dedup/cooldown. Do not let the demo rely on the Phase-3 cycle graph for safety.

---

## 8. Build Order + TDD Plan

Front-load the kanban vertical slice; it becomes the permanent regression oracle. Each step is gated by a deterministic mocked-LLM test (red→green→refactor), table-driven with moq + `testify/require`.

| Step | Scope | Gate test |
|---|---|---|
| **0 (blocker)** | Unified KB (`notekb`, buffer→`updateNotes` apply, surgical `AgentChange`) + canonical `applyAgentResult` helper (seeds appreq depth+1+actor) + coordinated runtime edit (`patch_note` + `Input.Tools`, with `filekb`/`memKB`/`cmd/agent` updated together, then runtime frozen again) | `notekb` buffer/overlay/leak tests green; `agentruntime/...` Phase-1 tests stay green |
| **1** | Confirm Phase-1 reuse (done) | `go test ./internal/agentruntime/...` green |
| **2 (DEMO)** | M1; `instruction_llm` branch (+cron); demo seed + webhook via CRUD/schema (**demo-blocking**) | Narrow gate (§5 P1) **+ headline E2E**: move card → `updateNotes` patch → SSE → agent decides → surgical write-back → 2nd SSE re-renders card. **Verify the shipped `kanban_template` actually subscribes to `noteChanges`; fix the bundle if not** (critic gap). Re-run as regression after every later step |
| **3** | Phase 2 secrets (zero-migration prefix + header injection; migrate `payload.Secrets` off logs) | Secret-leak gate (§5 P2) |
| **4** | Jsonnet mode (same `executor_mode`, same helper, `attach_notes` from M1) | Jsonnet gate, LLM never called |
| **5** | Phase 3 fleet/attach-gate/concurrency/exactly-once on M2; resolve fleet credential + SSRF allowlist **before enabling** | 5 acceptance tests + reconcile-idempotency + tools fail-fast |
| **6** | Phase 4 observability on M3 + `agentgraph` view | Cycle gate + attribution + tokens persisted |

**Never** add `max_tokens`/`max_steps` as overridable columns at any step.

---

## 9. Open Questions for the User (few, high-impact)

1. **Kanban SSE subscription (likely required fix):** project memory says the shipped `trip2g/kanban_template` does `updateNotes` saves but its `noteChanges` **live re-render was never confirmed in a running app**. The headline acceptance test depends on it. Do you want Step 2 to include patching the shipped bundle to subscribe to `noteChanges`, or is there a known-good build to point at?

2. **Cap overshoot policy:** make the token cap a true wall via per-call `max_completion_tokens` (recommended), or accept and document the bounded one-response overshoot?

3. **Fleet credential (Phase 3 blocker):** standardize on extending `checkapikey` to elevate an **admin/topology-scoped API-key type** (recommended for a daemon), or hand the fleet an admin JWT? Until decided, `reconcileFleet` fails closed.

4. **Demo concurrency guard scope:** is a per-webhook delivery dedup/cooldown + a bounded `instruction_llm` queue acceptable to ship in Step 2 (recommended given the undersized prod box), or defer all concurrency control to Phase 5?

All other forks raised by the critic are resolved in this spec and need no human decision.
