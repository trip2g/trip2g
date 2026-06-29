# trip2g Agent Runtime — Unified Design (Fleet-as-Executor)

> **This is the canonical design.** It supersedes the earlier in-process draft (preserved in git history) where the executor ran inside trip2g — that approach is dropped (see §4 "DROPPED vs the prior doc"). The fleet-as-executor model below is the one we build.

---

## 1. TL;DR / Вывод

**We move the agent executor OUT of trip2g and INTO a separate fleet process. trip2g stays a dumb event source.**

The spine, end to end:

```
note save (updateNotes)
  → trip2g fires an external_url webhook  (EXISTING path, unchanged)
  → the FLEET (cmd/fleet) receives the POST, runs the agent loop (agentruntime.Run)
  → the fleet writes back via the trip2g API using a per-delivery SCOPED token (updateNotes)
  → noteChanges SSE re-renders the board
```

- **trip2g = event source only.** `deliverchangewebhook` / `delivercronwebhook` keep doing `webhookutil.Deliver` (HTTP POST). trip2g never imports `agentruntime`, never runs an LLM.
- **The fleet = the agent host AND the `external_url` target.** It owns `internal/agentruntime/*` (the agent loop) and `cmd/fleet` (the daemon). The note-driven offline harness is `cmd/fleet --once <role-note>`; the former `cmd/agent` raw-instruction CLI is removed — agent specs come only from notes.
- **Agent = note.** A role note's **frontmatter** carries all config (model, tools, read/write patterns, budgets, triggers, mode); the **body** is the instruction. No trip2g DB columns for model/budget/tools/patterns.
- **trip2g changes are minimal and additive:** `transform_jsonnet` (outbound payload transform), `attach_notes` (push context), `concurrency_mode` (no-overlap guard), and delivery attribution/spend — **2 migration files**.
- **Acceptance = the kanban demo**: drag a card → trip2g webhook → fleet agent triages → writes back → board re-renders, with `max_depth=1` preventing self-retrigger.

**Second demo — Krisp transcript → knowledge-base pipeline.** Raw call transcripts become topic-split segments, then wiki notes with `[[WikiLinks]]`. The raw-transcript extraction + topic-splitting step is a **deterministic `executor: code` agent — no LLM** (following the original Krisp topic-split demos); the LLM is reserved for the semantic wiki-extraction step. Deterministic ingest keeps the source auditable and re-processable. Delivered by the code-executor role kind (separate follow-up).

**Critic-driven corrections baked in:** the "scoped token" is made *real* (server-side pattern enforcement — today it is decorative); the token TTL floor is dropped and the token/secrets are kept out of the jsonnet ExtVar + the logged body; `patch_note` is added to the runtime for surgical write-back; the kanban bundle's `noteChanges` subscription is a Step-0 blocker; the reconcile admin key is named honestly as **full-admin**.

---

## 2. Goal & kanban demo (the regression oracle)

**Goal:** a working vertical slice where a note edit drives an external LLM agent that writes back, observable over SSE, safe against loops and overlap.

**Seed (in `docs/demo/`):**

| File | Role |
|------|------|
| `boards/sprint.md` | Kanban board; cards are list items with a status field (surgical-patch friendly) |
| `roles/triage.md` | The agent. Frontmatter = model + read/write patterns + budget + triggers + mode; **body = triage instruction** |

**`roles/triage.md` frontmatter (flat schema, maps 1:1 onto `NoteView.meta`):**

```yaml
model: gpt-4o-mini
tools: [search, read_note, patch_note]
read_patterns:  ["boards/**", "roles/**"]
write_patterns: ["boards/**"]
max_tokens: 4000
max_steps: 6
mode: change
trigger_include: ["boards/sprint.md"]
trigger_on: [update]
attach_notes: ["boards/**", "roles/**"]
max_depth: 1
concurrency: skip
```

**The reconciled webhook (fleet registers it via admin CRUD):**

```
url             = <CallbackURL>/deliver/<urlKey("roles/triage.md")>
include_patterns= ["boards/sprint.md"]
attach_notes    = ["boards/**", "roles/**"]
read_patterns   = ["boards/**", "roles/**"]
write_patterns  = ["boards/**"]
pass_api_key    = true
on_update       = true
max_depth       = 1
concurrency_mode= skip
transform_jsonnet = ""           # identity; fleet expects the native payload
description     = "fleet:<FleetID>:roles/triage.md#<ver>"   # reconcile marker, no managed_by column
```

**End-to-end flow:**

1. Drag card → `updateNotes` surgical patch saves `boards/sprint.md`.
2. `HandleLatestNotesAfterSave` (cmd/server/webhooks.go:16) publishes SSE + runs `handlenotewebhooks.Resolve` (depth=0).
3. depth 0 < 1 (resolve.go:132); `matchChange` hits (:85); **attach gate** passes; **concurrency=skip** inserts only if clear; enqueue.
4. `deliverchangewebhook` marks delivery `running`, materializes `boards/sprint.md`+`roles/triage.md`, mints a scoped token (Depth=1), POSTs payload to the fleet.
5. Fleet verifies HMAC → builds `agentruntime.Input` (budget clamped) → `Run` over `remoteKB` with the scoped token.
6. Agent decides → `patch_note` → fleet calls `updateNotes` (scoped token, Depth=1) → note version stamped with delivery attribution.
7. 2nd `noteChanges` SSE → board re-renders. Re-entry at depth=1 ≥ max_depth=1 → **skipped** (no third delivery).

**Concurrency guard (demo):** server-side `concurrency_mode=skip` (skip in-flight) **+** a per-webhook cooldown (appconfig, no column) collapses a card-drag storm into one delivery.

**Acceptance test:** exactly **two** `noteChanges` events; card reflects the agent's decision; no third delivery.

> **Step-0 blocker:** the shipped `trip2g/kanban_template` bundle has never been confirmed to subscribe to `noteChanges` live. Verify `assets/ui/user/live` consumes it and patch the bundle if not — otherwise the headline test cannot pass regardless of backend correctness.

### 2b. Second worked example — forms→inbox triage (cron lane)

Validates the **cron** path + the work-available/cursor patterns. A public form (existing `internal/case/submitform`) collects "request to be added to the knowledge index." Submissions land in `form_submits` + EAV — **not notes** — so this is a **cron role**, not a reactive one:

- `roles/inbox-triage.md` frontmatter: `mode: cron`, `cron_schedule: "*/5 * * * *"`, `read_patterns: ["index/**"]`, `write_patterns: ["index/**"]`, `tools: [search, read_note, patch_note]`, small budget. Body = the triage instruction (decide KB-inclusion).
- The fleet reconciles a **cron_webhook** (url=fleet callback, `pass_api_key=true`). On each tick trip2g POSTs; the fleet lists unprocessed submissions via the admin lane (`admin.formSubmits`), decides per item, writes accepted ones as `index/<slug>.md` (scoped token, `updateNotes`), and calls `markFormSubmitProcessed` — which **is** the cursor/high-watermark: advance only after a successful write = at-least-once + dedup by submit id = effectively once.
- Alternative (reactive): a tiny submit→note bridge writing `inbox/<id>.md`, then the `attach_notes: ["inbox/*.md"]` work-available gate + atomic-claim semaphore apply (same as the kanban lane). Prefer the cron lane first — zero new trip2g code.

---

## 3. Architecture overview

**One idea per layer:**

| Layer | What it is | Where |
|-------|-----------|-------|
| **Event source** | trip2g, unchanged. Webhooks fire on note-change/cron → POST to `external_url`. | `cmd/server/webhooks.go`, `internal/case/handlenotewebhooks`, `internal/case/backjob/deliver*webhook` |
| **Agent host** | `cmd/fleet` daemon. Discovers role notes, reconciles webhooks, receives deliveries, runs `agentruntime.Run`, writes back. The `external_url` target. | `cmd/fleet`, `internal/fleet` |
| **RemoteKB** | The fleet's `agentruntime.KB` impl over the trip2g API, using the **per-delivery scoped token**; seeded by `attach_notes`. | `internal/fleet/remotekb.go` |
| **Outbound transform** | `transform_jsonnet` — a trip2g-side layer that reshapes the POST body before delivery. Orthogonal to the target being a fleet. | `internal/jsonneteval`, `deliver*webhook` |
| **Context push** | `attach_notes` — trip2g materializes matched notes into the payload (fewer round-trips); fleet can still pull more with its scoped token. | `handlenotewebhooks`, `deliver*webhook` |
| **Config** | Role-note frontmatter. **Cap = min(frontmatter, fleet ceiling).** The fleet ceiling is the non-overridable machine floor. | `roles/*.md`, `internal/fleet/handler.go` |

**Two auth lanes (a key correction over the brief):**

- **Reconcile + discovery → admin lane.** `POST /_system/mcp`, `X-API-Key` with `EnableMcpAdminTools=true` → `graphql_request` → `app.GraphQLRequest` does `appreq.WithAdminToken` elevation (cmd/server/graphql.go:182). This is a **different** elevation from `checkapikey`, so the prior critic's "`checkapikey` elevates JWTs not API keys" blocker dissolves — **zero new server auth code**. ⚠️ This key is **full admin**, not webhook-scoped (see §7).
- **Per-delivery note IO → scoped lane.** `POST /_system/graphql`, `Authorization: Bearer <shortapitoken>`. `checkapikey.Resolve` accepts the shortapitoken and stamps Depth + Read/WritePatterns. The MCP endpoint **rejects** shortapitoken (it only accepts `t2g_*` / `X-API-Key` / federation JWT), so the brief's "read via MCP" is wrong — scoped reads/writes MUST use `/_system/graphql`.

**Self-retrigger containment:** scoped token Depth = original+1 = 1; role `max_depth=1`; guard at `handlenotewebhooks/resolve.go:132`. **Structural** loop closure — but only holds if the fleet writes with the *scoped* token, never the admin key (enforced by design + test).

---

## 3b. Topology, coordinator/worker & deployment isolation

**Why a separate process at all:** you cannot put a network policy on a goroutine. Out-of-process execution makes least-privilege **enforceable at the infra layer**, below anything the LLM can talk its way around. An injected or compromised in-process agent would run with trip2g's own network identity and reach.

**Two roles (straight from `agent.md`'s security section):**

| Role | Scope | Egress / shell | Placement |
|------|-------|----------------|-----------|
| **Coordinator** | broad read; *moves notes* (writes task-notes that trigger worker webhooks = delegation) | no shell, no external egress | trusted zone — a central coordinator fleet co-located with trip2g |
| **Worker** | narrow read/write (its `*_patterns`) | shell + only the external egress its capability needs | isolated zone — its own container/server with an egress allowlist |

**Defense-in-depth (layers fail independently):**

| Layer | Mechanism | Bounds |
|-------|-----------|--------|
| Infra | per-worker network policy / container / server | egress allowlist: a worker reaches *only* trip2g's API + its LLM endpoint + its capability's destinations |
| Credential | per-delivery scoped shortapitoken | read/write globs (real only after the §5c.5 fix) |
| App | `ScopedKB` + server-side pattern enforcement | can't see/write foreign or private notes |
| Budget | fleet token/step ceiling | runaway cost |

**Reachability nuance (critic):** trip2g→fleet delivery must reach the worker (allowlist the fleet host on the trip2g side, or DevMode for private addressing) — this is **separate** from the worker's own locked-down egress. Don't conflate them.

**Embeddable-later seam (fleetbox direction):** the coordinator is a fleet role-note today (trip2g stays a dumb hub). Keep `cmd/fleet` / `internal/fleet` free of trip2g internals so a coordinator *could* be embedded in the box binary later without rework — but ship it out-of-process for v1 (isolation > convenience).

---

## 4. Reuse map vs net-new — and what is DROPPED

### Reused (lean on, do not rebuild)

| Capability | File:line |
|-----------|-----------|
| Agent loop (token hard-cap, max-steps, Result.TokensUsed/Steps) | `internal/agentruntime/runtime.go:79,111,113,53-54` |
| KB interface + ScopedKB glob gating | `agentruntime/kb.go:21`, `scope.go` |
| OpenAI-compatible LLM (configurable BaseURL) | `agentruntime/openai_llm.go:19` |
| HMAC sign/verify, Deliver, retry/backoff | `internal/webhookutil/hmac.go:18`, `httpclient.go:27`, `deliver*webhook` `handleDeliveryError` |
| Glob matching + JSON array parse | `internal/webhookutil/patterns.go:11,23` |
| Admin webhook CRUD mutations | `internal/case/admin/{create,update,delete}webhook` + cron twins; `schema.graphqls:3240-3251` |
| Per-delivery token minting (Depth+1, Read/WritePatterns) | `deliverchangewebhook/resolve.go:88-92` |
| shortapitoken acceptance → appreq scope | `internal/case/checkapikey/resolve.go:54-62,123-143` |
| Note actor pipe (user/api_key/client attribution) | `cmd/server/notes.go:116` → `insertnote/resolve.go:91-100` → `queries.write.sql:27` |
| Jsonnet VM (MaxStack=500, ExtVar, EvaluateAnonymousSnippet) | `internal/frontmatterpatch/evaluate.go:11,53-57` |
| Single serialized write connection (makes conditional INSERT race-free) | `cmd/server/main.go:153`, `graphql.go:47` |
| Standalone-cmd-in-module pattern; ArrayFlags | `cmd/agent`, `cmd/businessdemo`, `appconfig/config.go:36` |
| Discovery surface: notePaths(like), note(path:), noteChanges SSE | `schema.graphqls:1528,152,168,2048` |

### Net-new

- **`cmd/fleet/main.go`** + **`internal/fleet/`** (config, fleet, role, discovery, reconcile, client, remotekb, handler, secret).
- **`internal/jsonneteval/`** (NewVM/EvalJSON/Validate) — extracted shared seam.
- `patch_note` tool + `AgentChange.Find/Replace/Kind` in `agentruntime` (coordinated runtime edit, then re-freeze).
- Server enforcement of `WebhookWritePatterns`/`WebhookReadPatterns` (the security fix).
- 4 additive trip2g columns sets across 2 migrations + CRUD plumbing.
- Stale-lock janitor cron.

### DROPPED vs the prior doc (do **not** build)

| Dropped | Why |
|---------|-----|
| `executor_mode` column | trip2g no longer chooses an executor; it always POSTs |
| In-process LLM executor in trip2g | executor lives in the fleet |
| `instruction_llm` mode | not a mode; the fleet runs the loop |
| `internal/notekb` in-process apply branch | no in-process apply; fleet writes via the API |
| In-process canonical-apply branch in `deliverchangewebhook` | `webhookutil.Deliver` only; `applyAgentChanges` no-ops on empty changes |
| `model` / `max_tokens` / `max_steps` / `tools` / `read_patterns` / `write_patterns` / `agent_jsonnet` / `cursor_path` columns | all live in **frontmatter** |
| `managed_by` column | reconcile dedups via the existing `description` field |
| `secret_vars` / `header_groups` / `webhook_secret_vars` tables | reuse `secrets` table + `ChangeWebhookSecretPrefix` convention |
| Prior-doc §6 migration timestamps' DDL | reassigned here; strike §6 |

---

## 5. Components

### 5a. `cmd/fleet` daemon — discovery, frontmatter, reconcile, auth, delivery, budget

**Responsibility:** standalone trip2g-module binary that runs `agentruntime.Run` as a well-behaved external webhook target. Its own process → the "embed in app" Service-Package rule does not apply; it declares its own small types with injected deps (Env-style for testability).

**Files:** `cmd/fleet/main.go`, `internal/fleet/{config,fleet,role,discovery,reconcile,client,remotekb,handler,secret}.go` (+ `*_test.go`).

**Config:**

```go
// internal/fleet/config.go
type Config struct {
    FleetID       string        // reconcile marker prefix "fleet:<FleetID>:"
    ListenAddr    string        // ":9090"
    CallbackURL   string        // trip2g-reachable base; webhook url = CallbackURL+"/deliver/"+urlKey(path)
    Trip2gBaseURL string
    AdminAPIKey   string        // X-API-Key, EnableMcpAdminTools=true — FULL ADMIN (see security)
    FleetSecret   string        // per-role secret = HMAC(FleetSecret, FleetID+":"+path+":"+ver)
    LLMBaseURL    string        // OpenAI-compatible; fleet-local, NOT a trip2g secret
    LLMAPIKey     string
    DefaultModel  string
    TokenCeiling  int           // non-overridable: effective = min(frontmatter, ceiling)
    StepCeiling   int
    AgentsFolder  string        // "agents/" -> notePaths like "agents/%"
    OfferedTools  []string      // role.Tools must be a subset (fail-fast)
    PollInterval  time.Duration // PRIMARY change-watch (SSE is an optimization, see open Qs)
}
```

**Role (parsed note):**

```go
// internal/fleet/role.go — flat schema maps 1:1 onto NoteView.meta + webhook columns
type Role struct {
    NotePath      string   // identity / reconcile key
    Body          string   // the instruction
    Model         string
    Tools         []string
    ReadPatterns, WritePatterns []string
    MaxTokens, MaxSteps int   // pre-clamp
    Mode          string     // "change" | "cron" | "both"
    Change        *ChangeTrigger
    Cron          *CronTrigger
    TransformJsonnet string
}
func ParseRole(notePath, body string, meta map[string]string) (Role, error)
func (r Role) Validate(offered []string) error // tools ⊄ offered -> fail fast
```

**Auth lanes + reconcile + correlation:**

```go
// internal/fleet/client.go
func (c *Client) AdminGraphQL(ctx, query string, vars map[string]any) (json.RawMessage, error) // POST /_system/mcp, X-API-Key
func (c *Client) UpdateNotesScoped(ctx, token string, changes []ScopedChange) error            // POST /_system/graphql, Bearer
```

- **Discovery:** `notePaths(like:"<AgentsFolder>%")` then `note(path:)` per hit, body via `NoteView.content`, frontmatter via `NoteView.meta` (flat key/raw). Admin lane.
- **Reconcile** (`reconcile.go`): desired set = one change/cron webhook per role; idempotent Create/Update(specHash changed)/Delete keyed by the `description` marker `fleet:<FleetID>:<path>#<ver>`. Secret is **derived** and passed as `input.Secret` on create (`createwebhook/resolve.go:64`); `UpdateInput` has no secret field, so rotation = bump `<ver>` → Deregister+recreate.
- **Correlation:** URL path `/deliver/<urlKey(notePath)>` — the delivery header `X-Webhook-ID` is the *delivery* id, not the webhook id, so the role key is encoded in the URL the fleet itself registered.

**Delivery handler + budget clamp:**

```go
// internal/fleet/handler.go
func (f *Fleet) ServeDelivery(w http.ResponseWriter, r *http.Request) {
    role := f.registry[roleKey(r)]                 // 404 if unknown
    if !webhookutil.VerifyHMAC(body, f.secret(role), sig) { http.Error(w, "", 401); return }
    in := agentruntime.Input{
        Instruction:   role.Body,
        ReadPatterns:  role.ReadPatterns,
        WritePatterns: role.WritePatterns,
        Model:         orDefault(role.Model, f.cfg.DefaultModel),
        MaxTokens:     min(role.MaxTokens, f.cfg.TokenCeiling), // non-overridable cap
        MaxSteps:      min(role.MaxSteps,  f.cfg.StepCeiling),
        LLM:           f.llm,
        KB:            newRemoteKB(f.client, payload.APIToken, payload.AttachedNotes),
    }
    res, _ := agentruntime.Run(ctx, in)
    // changes:[] — writes ALREADY happened via remoteKB during the run (avoids server double-apply)
    writeJSON(w, 200, resp{Status: res.Status, Changes: nil, TokensUsed: res.TokensUsed, Steps: res.Steps})
}
```

> Cooldown/skip path returns **202 Accepted**, which the server explicitly excludes from change-parsing (`deliverchangewebhook/resolve.go:152 != http.StatusAccepted`).

**Narrow validation tests:** `ParseRole`/`Validate` (tools-subset fail-fast); budget clamp `min()`; Reconcile diff (create/update/delete idempotency, foreign webhooks untouched); handler HMAC 200/401/404; happy path asserts **exactly one** `UpdateNotesScoped` with the scoped token and `changes:[]`, **admin key never used for writes**; RemoteKB overlay (attached read = no API call); cooldown → 202, no LLM run. All offline with moq `Client` + stub `LLM`.

---

### 5b. `internal/fleet` RemoteKB — attach_notes seeding, scoped token, spend reporting

```go
// internal/fleet/remotekb.go
type remoteKB struct {
    client  *Client
    token   string            // per-delivery scoped token from payload.api_token
    overlay map[string]string // materialized attach_notes (path -> content)
}
func (k *remoteKB) Search(ctx, q string) ([]agentruntime.Doc, error)         // SearchScoped
func (k *remoteKB) Read(ctx, path string) (string, error)                    // overlay first, else NoteScoped
func (k *remoteKB) Write(ctx, path, content string) error                    // updateNotes upsert
func (k *remoteKB) Patch(ctx, path, find, replace string) error              // updateNotes SURGICAL patch
var _ agentruntime.KB = (*remoteKB)(nil)
```

- **attach_notes seeding:** the overlay is populated from the payload's materialized notes — zero round-trips for in-band context. Reads of non-attached in-scope paths fall back to `NoteScoped`.
- **Scoped token only:** every IO uses `k.token`; the admin key is structurally unreachable from RemoteKB.
- **Surgical write-back (`Patch`):** routes to the `updateNotes` patch branch (find/replace) so unmodeled kanban card metadata is preserved. Requires the `patch_note` tool + `AgentChange.Find/Replace/Kind="patch"` addition in `agentruntime` (one coordinated runtime edit: update `runtime.go`, `kb.go`, `scope.go`, `filekb.go`, `cmd/agent` together, then re-freeze). **Without this the demo only "passes" via lossy full rewrite — it misrepresents production.**

**Validation tests:** overlay hit returns content with no client call; out-of-scope read denied by `ScopedKB` before any client call; `Patch` issues the find/replace `updateNotes` variant; spend (`res.TokensUsed/Steps`) flows into the HTTP response.

---

### 5c. trip2g minimal changes — transform_jsonnet, attach_notes, concurrency, attribution

**Invariant:** the egress path is unchanged. All five changes hang off the existing seams; trip2g never imports `agentruntime`.

**(1) `transform_jsonnet` — outbound transform.** New shared package, NOT a call into `frontmatterpatch.Evaluate` (which hardcodes `meta`/`path` ExtVars + shallow-merge — wrong for an arbitrary outbound body):

```go
// internal/jsonneteval/jsonneteval.go — single MaxStack source of truth
func NewVM() *jsonnet.VM { vm := jsonnet.MakeVM(); vm.MaxStack = 500; return vm }
func EvalJSON(src string, extVars map[string]string) (json.RawMessage, error)
func Validate(src string, sampleExtVars map[string]string) error
```

Applied **strictly between `json.Marshal` (resolve.go:101) and `SignHMAC` (:107)** so the signature and the logged `request_body` both cover the transformed bytes:

```go
if wh.TransformJsonnet != "" {
    out, terr := jsonneteval.EvalJSON(wh.TransformJsonnet, transformExtVars(payloadBytes))
    if terr != nil { // never send a half-built request
        handleDeliveryError(ctx, env, params,
            webhookutil.DeliveryResult{Err: fmt.Errorf("transform_jsonnet: %w", terr)}, wh)
        return nil
    }
    payloadBytes = out
}
signature := webhookutil.SignHMAC(payloadBytes, wh.Secret)
```

> **Security:** `transformExtVars` exposes the change/attached-notes/meta — but **NOT** `api_token` and **NOT** `secrets` (the powerful credential and secret values are delivered via separate fields/headers the jsonnet cannot see, so they never reach the logged `request_body`). `frontmatterpatch.NewVM` is refactored to delegate to `jsonneteval.NewVM`; `Evaluate` untouched (its tests remain the guard). CRUD validates via `jsonneteval.Validate` → `ErrorPayload` on failure. Cron twin identical.

**(2) `attach_notes` — gate + materialize.**

```go
// handlenotewebhooks.Resolve — presence GATE after matched non-empty (:159), before Insert (:169)
attach, _ := webhookutil.ParseJSONStringArray(wh.AttachNotes)
if !attachGateSatisfied(attach, nvs) { continue } // plain glob: require ≥1; "!glob": require 0
```

Delivery job materializes matched (non-`!`) notes into the payload as `{path,title,content,updated_at,tags,meta}` from `model.NoteView` (`note.go:156`). **`meta` should be tags + a small allowlist, not full RawMeta** (avoid leaking frontmatter the role didn't ask for).

**(3) `concurrency_mode` — race-free no-overlap.** All mutations run on the single serialized write connection → `WHERE NOT EXISTS` is atomic; no app lock:

```sql
-- name: InsertWebhookDeliveryIfClear :many
insert into change_webhook_deliveries (webhook_id, attempt, status)
select sqlc.arg(webhook_id), 1, 'pending'
where not exists (
  select 1 from change_webhook_deliveries
  where webhook_id = sqlc.arg(webhook_id)
    and status in ('pending','running')
    and coalesce(heartbeat_at, started_at, created_at) >= datetime('now', sqlc.arg(stale_window)))
returning *;          -- 0 rows => skipped
```

```go
switch wh.ConcurrencyMode {
case "skip":      d, ok, _ := env.InsertWebhookDeliveryIfClear(ctx, wh.ID, staleWindow(wh), cooldown); if !ok { continue }
case "queue_one": d, ok, _ := env.InsertWebhookDeliveryIfNoPending(ctx, wh.ID); if !ok { continue }
default:          d, _ := env.InsertWebhookDelivery(ctx, ...) // allow_overlap = today
}
```

- `'running'` status needs **no DDL** (status column has no CHECK, `20260209100000:32`).
- `MarkWebhookDeliveryRunning` (`status='running', started_at=now where id=? and status='pending'`) at job pickup.
- Stale predicate `coalesce(heartbeat_at,started_at,created_at)` self-heals a crashed-fleet `running` row.
- **Janitor cron** `expirestalewebhookdeliveries` finalizes orphans to `failed` (registered `cmd/server/cronjobs.go:45`).
- **Demo cooldown** = appconfig `AgentDeliveryCooldownSeconds` (no column).
- ⚠️ CRUD must validate the `concurrency_mode` enum **before** insert, else a raw SQLite CHECK error surfaces as `nil,error` ("Internal Error") instead of a clean `ErrorPayload`.

**(4) Attribution + spend.** The write-back is a separate `updateNotes` call authenticated by the scoped shortapitoken — which today produces a **virtual api key ID 0** (`checkapikey/resolve.go:135`), so attribution is blind. Carry the delivery identity in the token and stamp it through the existing actor pipe:

```go
// internal/shortapitoken/token.go
type Data struct {
    Depth         int      `json:"d"`
    ReadPatterns  []string `json:"rp"`
    WritePatterns []string `json:"wp"`
    DeliveryKind  string   `json:"dk,omitempty"` // "change" | "cron"
    DeliveryID    int64    `json:"di,omitempty"`
}
// internal/model/note_actor.go
type NoteActor struct { UserID *int64; APIKeyID *int64; Client *string; DeliveryKind *string; DeliveryID *int64 }
// internal/webhookutil/agentresponse.go
type AgentResponse struct { Status, Message string; Changes []AgentChange; TokensUsed, Steps int }
```

Chain (≈6 files, must land atomically or attribution is silently NULL): `shortapitoken.Data` → `checkapikey.resolveShortAPIToken` sets `req.WebhookDeliveryKind/ID` → `appreq.Request`+`NewContext` → `NoteVersionActor` (`cmd/server/notes.go:116`) → `insertnote.Resolve` → `InsertNoteVersion` (+`make sqlc`). `(kind,id)` because change/cron are two tables (mirrors `webhook_delivery_logs`). Spend: `deliver*webhook` parses `AgentResponse.{TokensUsed,Steps}` → `UpdateWebhookDeliveryResult`.

**(5) THE SECURITY FIX — make the scope real.** Critic-verified: `WebhookReadPatterns`/`WritePatterns` are *set* but have **zero consumers** — the scope is decorative server-side. Add enforcement:

```go
// internal/case/updatenotes/resolve.go — for each change:
if wp := appreq.WebhookWritePatterns(ctx); len(wp) > 0 && !webhookutil.MatchesAny(path, wp) {
    return model.ErrorPayload("write denied for path: "+path), nil // validation -> ErrorPayload
}
// mirror in note(path:) / search resolvers for WebhookReadPatterns
```

This converts least-privilege from fiction to fact and closes the central security hole.

**Narrow tests:** jsonnet identity vs remap vs runtime-error→`handleDeliveryError` (no POST); HMAC over **transformed** bytes; empty column = byte-for-byte regression; attach gate truth table; concurrency table (allow/skip/queue_one × in-flight/stale/cooldown) + a two-concurrent-skip → exactly-one property; janitor (stale→failed, fresh→untouched, injected clock); attribution end-to-end (token → note_version row links to delivery); **write-pattern enforcement: out-of-pattern write → ErrorPayload**.

---

### 5d. Phase 2 — secrets under the new model

- trip2g-side secret resolution serves the `transform_jsonnet` outbound path + existing webhook secret (generalize the prefix+decrypt). **Zero new tables** — reuse `secrets` + `ChangeWebhookSecretPrefix`.
- The fleet's **own LLM API key is fleet-local config**, never a trip2g secret, never in a note/message/log.
- If an agent tool needs an authed external call, the fleet uses fleet-local config OR fetches **materialized headers** from a scoped trip2g endpoint — **never the master key**.
- Secret VALUES resolve server-side in trip2g and are kept out of the jsonnet ExtVar and the logged body (see §7).

```go
// sketch — Phase 2 scoped header endpoint (no master key crosses the boundary)
func (c *Client) MaterializeHeaders(ctx, token, group string) (map[string]string, error) // POST /_system/graphql, scoped
```

**Test:** secret values never appear in `request_body` logs; fleet never receives the master key; header materialization is gated by the scoped token.

---

## 6. Consolidated MIGRATION LIST (2 files) — confirm before creating

> **CLAUDE.md always-ask rule:** present this enumerated list for **ONE user confirmation** before any file is created. **No model/budget/tools/patterns columns. No secret tables.** Reconcile dedups via the existing `description` field (no `managed_by`). Latest on-disk is `20260627000000`.

**M1 — `db/migrations/20260628120000_webhook_transform_context_concurrency.sql`**

| Column (on `change_webhooks` AND `cron_webhooks`) | Reason |
|---|---|
| `transform_jsonnet TEXT NOT NULL DEFAULT ''` | outbound payload transform |
| `attach_notes TEXT NOT NULL DEFAULT '[]'` | context globs + `!`-require-absent gate |
| `concurrency_mode TEXT NOT NULL DEFAULT 'allow_overlap' CHECK (concurrency_mode IN ('allow_overlap','skip','queue_one'))` | no-overlap policy |

| Column (on `change_webhook_deliveries` AND `cron_webhook_deliveries`) | Reason |
|---|---|
| `started_at DATETIME` | run start: stale-lock + duration |
| `heartbeat_at DATETIME` | long-run liveness |
| `tokens_used INTEGER` | fleet-reported spend (NULL=unknown) |
| `steps INTEGER` | fleet-reported tool-loop steps |
| `idx_*_deliveries_inflight (webhook_id, status)` | fast no-overlap conditional insert |

No DDL for `'running'` status (column has no CHECK).

**M2 — `db/migrations/20260628120100_note_version_delivery_attribution.sql`**

| Column on `note_versions` | Reason |
|---|---|
| `created_by_delivery_kind TEXT` | link version → delivery (`'change'`/`'cron'`) |
| `created_by_delivery_id INTEGER` | the `(kind,id)` pair; two tables → no single FK |
| `idx_note_versions_delivery (created_by_delivery_kind, created_by_delivery_id)` | per-webhook authored-path aggregates |

**After confirmation:** `dbmate up` → `make sqlc` → (after schema.graphqls edits) `make gqlgen` → `npm run graphqlgen`. Serialize codegen.

**Explicitly excluded:** `executor_mode`, `model`, `agent_jsonnet`, `cursor_path`, `max_tokens`, `max_steps`, `managed_by`, `secret_vars`/`header_groups` tables.

---

## 7. Security & budget model

| Control | Mechanism | Status |
|---------|-----------|--------|
| **Per-delivery least privilege** | Fleet uses only the scoped shortapitoken (Depth+1, Read/WritePatterns) for note IO; master/admin key never in a delivery. | **Only real after the §5c fix.** |
| **Scope enforcement (THE fix)** | `updateNotes` / `note` / `search` resolvers reject paths outside `WebhookWritePatterns`/`ReadPatterns`. Today these are unenforced (decorative). | **Must land before any non-local run.** |
| **Token TTL** | Drop the ≥60-min floor (`deliverchangewebhook:84-85`) for reactive deliveries → `ttl = TimeoutSeconds + small margin`. Require TLS for `CallbackURL`. | Neutralizes the long-lived bearer. |
| **Token/secret redaction** | `api_token` and `secrets` are NOT exposed as jsonnet ExtVars and NOT in the logged `request_body`; delivered via separate fields/headers. | Closes the log-leak. |
| **HMAC authenticity** | Every delivery verified with the per-role derived secret before any LLM call. | Rejects forged/replayed envelopes. |
| **Rotatable HMAC** | `deriveSecret(FleetSecret, FleetID+path+ver)`; `ver` in the `description` marker → Deregister+recreate rotates cleanly. FleetSecret is high-value. | |
| **Reconcile admin key blast radius** | `EnableMcpAdminTools=true` → `WithAdminToken` elevates **EVERY** mutation — this is **full admin**, not webhook-scoped. Keep fleet-local, TLS, rotatable; documented honestly. Build an admin-scoped elevation only if the exposure is unacceptable. | **Named, accepted for v1.** |
| **Write-only-with-scoped-token** | Fleet write-back uses exclusively the scoped token on `/_system/graphql`, never the admin key — preserves Depth=1 loop containment. | Enforced + tested. |
| **transform_jsonnet / SSRF** | go-jsonnet has no IO; MaxStack=500; `Validate` at CRUD. Egress still `webhookutil.Deliver` → `ssrfsafe.DialTimeout` (private IPs blocked). Transform reshapes the body only, never the destination. | SSRF posture preserved. |
| **Fleet ceiling = non-overridable cap** | `effective = min(frontmatter, fleetCeiling)`, clamped before `Run`; `agentruntime` enforces (`runtime.go:111,113`). A note author cannot exceed the machine limit. | |
| **Tools fail-fast** | `Role.Validate` rejects tools outside `OfferedTools` at discovery, before any webhook is registered. | |
| **Depth + cycle graph** | `max_depth` + scoped-token Depth = structural loop bound. Advisory influence graph: edge A→B when an authored path of A's deliveries `MatchesAny` B.include_patterns. | |
| **Model split** | Fleet's LLM key fleet-local; trip2g secrets resolve server-side, never decrypted in the fleet. | |

---

## 8. Build order + TDD plan

**Front-load the kanban fleet vertical slice as the regression oracle.** Every step gates on a deterministic check (mocked LLM + mock trip2g API).

| # | Step | Gate |
|---|------|------|
| 0 | **Verify/patch `kanban_template` `noteChanges` subscription** (acceptance oracle). | `assets/ui/user/live` consumes `noteChanges`; bundle subscribes; a manual SSE save re-renders. **Blocker — do first.** |
| 1 | **Get the single migration confirmation** (M1+M2). | User approves the §6 list. Nothing created before. |
| 2 | **Land Component B first** (security backbone has hard build dependents): migrations → `schema.graphqls` Create/Update inputs (`attachNotes`/`transformJsonnet`/`concurrencyMode`) + cron twins → `createwebhook`/`updatewebhook` + cron twins → `make sqlc` → `make gqlgen` → `npm run graphqlgen`. | Inputs round-trip; CRUD enum-validates → `ErrorPayload`; empty `transform_jsonnet` = byte-for-byte regression test passes. |
| 3 | **Security backbone (before any non-local run):** server-side `WebhookWritePatterns`/`ReadPatterns` enforcement; TTL-floor removal; api_token/secrets out of the jsonnet ExtVar + redacted from `request_body`. | Out-of-pattern write → `ErrorPayload` test; token TTL test; log-redaction test. |
| 4 | **`internal/jsonneteval` + transform wiring.** | identity/remap/error tests; HMAC over transformed bytes; cron twin mirror. |
| 5 | **attach_notes gate + materialize.** | gate truth table; payload carries `{path,title,content,updated_at,tags,meta}` (allowlisted meta). |
| 6 | **Concurrency + janitor + attribution + spend.** | concurrency table + exactly-one property; janitor clock test; attribution end-to-end NULL→linked; spend persisted. |
| 7 | **`patch_note` in `agentruntime`** (coordinated runtime edit, re-freeze). | `filekb` + `cmd/agent` round-trip a find/replace; `runtime_test.go` green. |
| 8 | **Component A: `cmd/fleet`** (discovery, reconcile via admin MCP lane, scoped writes via `/_system/graphql`, RemoteKB, handler). | ParseRole/Validate; budget clamp; Reconcile diff idempotency; handler 200/401/404/202; happy path = one scoped `UpdateNotesScoped`, `changes:[]`, admin key unused for writes. |
| 9 | **Headline E2E (local DevMode):** seed `boards/sprint.md`+`roles/triage.md`, run `cmd/fleet`, drag a card. | **Exactly two `noteChanges` events; card reflects the decision; no third delivery (depth=1 ≥ max_depth=1).** |
| 10 | Defer prod deployment. | Until CallbackURL SSRF/egress + SSE-watch auth resolved (open Qs). |
| 11 | **Docs**: this dev reference + the bilingual user pair (`docs/en/user/*`, `docs/ru/user/*` via the trip2g-docs agent); update `docs/dev/agent.md` status. | Docs reviewed; answer-first per project style. |

**Determinism:** all fleet tests use moq `Client` + stub `agentruntime.LLM` (the runtime is already stub-testable); no live trip2g, no live LLM. The E2E runs against a local trip2g in DevMode (SSRF guard bypassed).

---

## 9. Open questions for the user

1. **Migration confirmation (required):** approve the two files — `20260628120000` (webhook+delivery columns) and `20260628120100` (note_versions attribution) — as enumerated in §6, before any file is created?
2. **Reconcile admin key:** accept the **full-admin** `EnableMcpAdminTools` key (fleet-local, TLS, rotatable, documented) for v1, or invest in an admin-scoped API-key elevation path now?
3. **Change-watch transport:** ship with `PollInterval` as the reliable role-edit watch (SSE admin `X-API-Key` auth on `noteChanges` is **unverified** — it cannot share the MCP POST transport), and add SSE as an optimization once that auth is confirmed?
4. **Prod CallbackURL reachability:** will the fleet be co-located (needs a per-webhook egress allowlist / DevMode for private addressing) or public? This blocks prod deployment, not the local demo.

> Everything else from the prior open-question set is now resolved: scoped reads use `/_system/graphql` (not MCP); scope is enforced server-side (the fix); `patch_note` ships in Step 7; spend recording is in scope; flat frontmatter schema is adopted; reconcile dedups via `description`.
