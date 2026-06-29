# trip2g Agent Runtime (Fleet-as-Executor) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the fleet-as-executor agent runtime — trip2g fires `external_url` webhooks, `cmd/fleet` runs the agent loop and writes back via the scoped API, so notes-on-edit trigger agents (kanban demo as the acceptance oracle).

**Architecture:** trip2g stays a dumb event source (delivery path unchanged). A new `cmd/fleet` daemon discovers role-notes, reconciles webhooks, receives deliveries, runs `agentruntime.Run` over a scoped-token `RemoteKB`, and writes back via `updateNotes`. Minimal additive trip2g changes: `transform_jsonnet` (outbound transform), `attach_notes` (context push + gate), `concurrency_mode` (no-overlap), delivery attribution/spend — plus the **server-side scope enforcement** that today is missing (patterns are decorative).

**Tech Stack:** Go monolith; SQLite (dbmate + sqlc); gqlgen GraphQL; go-jsonnet; go-openai; $mol frontend. TDD throughout (table-driven, `testify/require`, `moq`).

**Canonical spec:** `docs/dev/agent_runtime_design.md`. **Migrations M1+M2 approved.** Reconcile uses a full-admin `EnableMcpAdminTools` key (v1). Role-note watch = poll-first.

---

## Shared contracts (read first — every task uses these names verbatim)

These are the cross-cutting types/shapes the tasks depend on. The canonical, fuller version lives in the spec.

- **Migrations (approved):** M1 `db/migrations/20260628120000_webhook_transform_context_concurrency.sql` (on both webhook tables: `transform_jsonnet TEXT '' `, `attach_notes TEXT '[]'`, `concurrency_mode TEXT 'allow_overlap' CHECK(...)`; on both delivery tables: `started_at`, `heartbeat_at`, `tokens_used`, `steps` + inflight index). M2 `db/migrations/20260628120100_note_version_delivery_attribution.sql` (`note_versions.created_by_delivery_kind/_id` + index). `'running'` status needs no DDL.
- **`webhookutil.AgentChange`** gains: `Find string`, `Replace string`, `Kind string` (`"" | "upsert" | "patch"`).
- **`agentruntime.KB`** gains `Patch(ctx, path, find, replace string) error` (implemented in `ScopedKB` with write-scope enforcement, `FileKB`, the test `memKB`); runtime gains a `patch_note` tool.
- **`webhookutil.AgentResponse`** (fleet→trip2g HTTP response) gains `TokensUsed int`, `Steps int` → trip2g persists to `deliveries.tokens_used/steps`.
- **`shortapitoken.Data`** gains `DeliveryKind string ("dk")`, `DeliveryID int64 ("di")` → stamped through `checkapikey` → `appreq` → `NoteActor` → `note_versions`.
- **Scope enforcement (THE fix):** `updatenotes` resolver rejects out-of-`WebhookWritePatterns` paths (`ErrorPayload`); `note(path:)` + `search` resolvers honor `WebhookReadPatterns`.
- **`internal/jsonneteval`:** `NewVM()`, `EvalJSON(src, extVars) (json.RawMessage, error)`, `Validate(...)`; `frontmatterpatch.NewVM` delegates. `transform_jsonnet` is applied between `json.Marshal(payload)` and `SignHMAC`, with `api_token`/`secrets` excluded from ExtVars + the logged body.
- **Fleet:** `cmd/fleet/main.go` + `internal/fleet/{config,fleet,role,discovery,reconcile,client,remotekb,handler}.go`. Admin lane = `POST /_system/mcp` `X-API-Key` (`graphql_request`); scoped lane = `POST /_system/graphql` `Bearer <shortapitoken>` (MCP rejects shortapitoken). Cap = `min(frontmatter, fleetCeiling)`.

---

## Section A — Steps 0-3: kanban-verify, migrations, CRUD/schema plumbing, security-backbone enforcement

> Worktree root for every path below: `/home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime/`. All paths shown relative to it. Go 1.26.1 (has the `min` builtin). Commit convention (project CLAUDE.md): short one-liner, Conventional Commits, **no body, no Co-Authored-By**.

> Hard ordering inside this section: **A2 → A3 → A4 → A5 → A6** are sequential (sqlc/gqlgen codegen dependencies — serialize, never parallelize). A1 is an independent blocker. A7/A8/A9/A10 depend only on A2 being merged (the migration must exist so the build compiles) and are otherwise independent of each other.

---

### A1 — [BLOCKER, Step 0] Verify (and patch if missing) the kanban board's `noteChanges` live re-render path

The headline E2E (Section C, Step 9) asserts "exactly two `noteChanges` events; the card reflects the agent's decision". That is only observable if the rendered board re-renders when a *remote* write lands on `boards/sprint.md`. Two re-render paths exist; **at least one must work end-to-end** or the acceptance test cannot pass regardless of backend correctness. This task confirms a working path and patches the shipped `trip2g/kanban_template` bundle only if neither path re-renders.

**Files:**
- Inspect (no edit): `assets/ui/user/live/live.view.ts`
- Inspect (external repo): the shipped `trip2g/kanban_template` bundle
- Patch only if absent: the `trip2g/kanban_template` bundle entry (external repo)

**Steps:**

1. [ ] **Confirm the page-level consumer subscribes to `noteChanges`.** Run:
   ```
   grep -n "noteChanges\|reload_enabled\|location.reload" assets/ui/user/live/live.view.ts
   ```
   Expected (already true at HEAD): the `QUERY` const declares `subscription NoteChanges($filter: NoteChangesFilter!)`, `subscription()` calls `$trip2g_graphql_raw_subscription(QUERY, { filter: { includePatterns: [ '**/*.md' ] } })`, and `watcher_result()` calls `location.reload()` when `reload_enabled()` and the changed `pathId` equals the viewed note. This is the **page-reload re-render path**: when the viewed note (`boards/sprint.md`) changes, the page reloads and the kanban bundle re-fetches+re-renders the board.

2. [ ] **Inspect the shipped kanban bundle for its own subscription.** Fetch the standalone repo and grep its source + built bundle:
   ```
   gh repo clone trip2g/kanban_template /tmp/claude-1000/-home-alexes-projects2-trip2g/987d48f3-890b-4186-b701-f5457a7f8e33/scratchpad/kanban_template
   grep -rni "notechanges\|EventSource\|/_system/graphql\|subscription" /tmp/claude-1000/-home-alexes-projects2-trip2g/987d48f3-890b-4186-b701-f5457a7f8e33/scratchpad/kanban_template --include=*.ts --include=*.tsx --include=*.js | grep -v node_modules
   ```
   - **If a `noteChanges` subscription is present** in the bundle → the **in-place re-render path** exists. Record the consumer file:line. Skip step 4.
   - **If absent** → the bundle relies entirely on the page-reload path (step 1). That path IS sufficient for the demo (board re-fetched on reload), provided the demo board page renders the live-reload toggler.

3. [ ] **Decide and document the working path.** Record explicitly in the PR description which path the acceptance test will use:
   - Path A (preferred, zero bundle change): page-reload via `assets/ui/user/live` with `reload_enabled()` ON on the `boards/sprint.md` page.
   - Path B: bundle self-subscription (only if step 2 found it).
   The gate (manual proof): with a local trip2g serving `docs/demo`, open the rendered `boards/sprint.md`, enable the live-reload toggler, then in a second client run an `updateNotes` patch on `boards/sprint.md`; the rendered board must reflect the new card state (after at most one reload). Capture the network tab showing the `noteChanges` SSE event.

4. [ ] **(Conditional) Patch the bundle only if Path A does NOT re-render the board and step 2 found no subscription.** In the kanban_template repo, in the module that loads the board note (the GraphQL `note(input:{path})` fetch), add an `EventSource` subscription to `noteChanges` filtered to the board note's path, and on a matching `pathId` re-run the board-note fetch and re-render (do not full-reload). Rebuild with the repo's build script and cut a new release / reinstall via its `curl`-install. Re-run the step-3 manual gate until the board reflects a remote write.

5. [ ] **Commit (only if step 4 patched the bundle, in the kanban_template repo):**
   ```
   feat(kanban): live-rerender board on noteChanges remote write
   ```
   (No commit in this repo if Path A was confirmed — record the confirmation evidence in the plan PR instead.)

---

### A2 — [Step 1] Create migrations M1 + M2, apply, regenerate db models

Migrations are APPROVED exactly as in the contract. sqlc reads its schema from `db/schema.sql` (see `sqlc.yaml`), which `dbmate up` regenerates — so the order is **write `.sql` → `make db-up` → `make sqlc`**.

**Files:**
- Create: `db/migrations/20260628120000_webhook_transform_context_concurrency.sql`
- Create: `db/migrations/20260628120100_note_version_delivery_attribution.sql`
- Regenerated (do not hand-edit): `db/schema.sql`, `internal/db/models.go`

**Steps:**

1. [ ] **Write the failing check (assert the new columns do not yet exist).** Run:
   ```
   grep -c "TransformJsonnet\|ConcurrencyMode\|CreatedByDeliveryKind" internal/db/models.go
   ```
   Expected: `0` (no such fields yet) — this is the red state.

2. [ ] **Create M1** `db/migrations/20260628120000_webhook_transform_context_concurrency.sql`:
   ```sql
   -- migrate:up

   alter table change_webhooks add column transform_jsonnet text not null default '';
   alter table change_webhooks add column attach_notes text not null default '[]';
   alter table change_webhooks add column concurrency_mode text not null default 'allow_overlap' check (concurrency_mode in ('allow_overlap','skip','queue_one'));

   alter table cron_webhooks add column transform_jsonnet text not null default '';
   alter table cron_webhooks add column attach_notes text not null default '[]';
   alter table cron_webhooks add column concurrency_mode text not null default 'allow_overlap' check (concurrency_mode in ('allow_overlap','skip','queue_one'));

   alter table change_webhook_deliveries add column started_at datetime;
   alter table change_webhook_deliveries add column heartbeat_at datetime;
   alter table change_webhook_deliveries add column tokens_used integer;
   alter table change_webhook_deliveries add column steps integer;

   alter table cron_webhook_deliveries add column started_at datetime;
   alter table cron_webhook_deliveries add column heartbeat_at datetime;
   alter table cron_webhook_deliveries add column tokens_used integer;
   alter table cron_webhook_deliveries add column steps integer;

   create index idx_change_webhook_deliveries_inflight on change_webhook_deliveries(webhook_id, status);
   create index idx_cron_webhook_deliveries_inflight on cron_webhook_deliveries(cron_webhook_id, status);

   -- migrate:down

   drop index if exists idx_cron_webhook_deliveries_inflight;
   drop index if exists idx_change_webhook_deliveries_inflight;

   alter table cron_webhook_deliveries drop column steps;
   alter table cron_webhook_deliveries drop column tokens_used;
   alter table cron_webhook_deliveries drop column heartbeat_at;
   alter table cron_webhook_deliveries drop column started_at;

   alter table change_webhook_deliveries drop column steps;
   alter table change_webhook_deliveries drop column tokens_used;
   alter table change_webhook_deliveries drop column heartbeat_at;
   alter table change_webhook_deliveries drop column started_at;

   alter table cron_webhooks drop column concurrency_mode;
   alter table cron_webhooks drop column attach_notes;
   alter table cron_webhooks drop column transform_jsonnet;

   alter table change_webhooks drop column concurrency_mode;
   alter table change_webhooks drop column attach_notes;
   alter table change_webhooks drop column transform_jsonnet;
   ```
   Note the cron delivery FK column is `cron_webhook_id` (not `webhook_id`) — confirmed in `db/migrations/20260209100000_create_webhook_tables.sql`. `drop column` is valid (SQLite ≥ 3.35), matching `db/migrations/20260518072631_add_processed_fields_to_form_submits.sql`.

3. [ ] **Create M2** `db/migrations/20260628120100_note_version_delivery_attribution.sql`:
   ```sql
   -- migrate:up

   alter table note_versions add column created_by_delivery_kind text;
   alter table note_versions add column created_by_delivery_id integer;

   create index idx_note_versions_delivery on note_versions(created_by_delivery_kind, created_by_delivery_id);

   -- migrate:down

   drop index if exists idx_note_versions_delivery;

   alter table note_versions drop column created_by_delivery_id;
   alter table note_versions drop column created_by_delivery_kind;
   ```

4. [ ] **Apply and regenerate.** Run (serialize, do not background):
   ```
   make db-up && make sqlc
   ```
   Expected: `dbmate up` applies both migrations and rewrites `db/schema.sql`; `make sqlc` regenerates `internal/db/` (also runs `./internal/db/fix_write_queries.sh`). No errors.

5. [ ] **Run the check to verify it passes.** Run:
   ```
   grep -n "TransformJsonnet\|AttachNotes\|ConcurrencyMode" internal/db/models.go
   grep -n "StartedAt\|HeartbeatAt\|TokensUsed\|Steps" internal/db/models.go
   grep -n "CreatedByDeliveryKind\|CreatedByDeliveryID" internal/db/models.go
   go build ./internal/db/...
   ```
   Expected: `ChangeWebhook`/`CronWebhook` structs carry `TransformJsonnet string`, `AttachNotes string`, `ConcurrencyMode string`; `ChangeWebhookDelivery`/`CronWebhookDelivery` carry `StartedAt *time.Time`, `HeartbeatAt *time.Time`, `TokensUsed *int64`, `Steps *int64`; `NoteVersion` carries `CreatedByDeliveryKind *string`, `CreatedByDeliveryID *int64`; build clean.

6. [ ] **Commit:**
   ```
   feat(webhooks): migrations for transform/attach/concurrency + delivery attribution
   ```

---

### A3 — [Step 2] Add the three webhook columns to the Insert/Update SQL queries, regenerate sqlc

`make sqlc` in A2 added the columns to the *row* structs (`db.ChangeWebhook`), but the `InsertWebhookParams`/`UpdateWebhookParams` structs only gain the new columns when the queries reference them.

**Files:**
- Modify: `queries.write.sql`
- Regenerated (do not hand-edit): `internal/db/queries.write.sql.go`

**Steps:**

1. [ ] **Write the failing check.** Run:
   ```
   grep -c "TransformJsonnet" internal/db/queries.write.sql.go
   ```
   Expected: `0` (params structs lack the field) — red.

2. [ ] **Edit `InsertWebhook`** (`queries.write.sql`, currently lines 936-939) to add the three columns before `created_by`:
   ```sql
   -- name: InsertWebhook :one
   insert into change_webhooks (url, include_patterns, exclude_patterns, instruction, secret, max_depth, pass_api_key, include_content, timeout_seconds, max_retries, description, on_create, on_update, on_remove, read_patterns, write_patterns, transform_jsonnet, attach_notes, concurrency_mode, created_by)
   values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
   returning *;
   ```

3. [ ] **Edit `UpdateWebhook`** (lines 941-961) — add three `coalesce` lines immediately before `updated_at = datetime('now')`:
   ```sql
       write_patterns = coalesce(sqlc.narg(write_patterns), write_patterns),
       transform_jsonnet = coalesce(sqlc.narg(transform_jsonnet), transform_jsonnet),
       attach_notes = coalesce(sqlc.narg(attach_notes), attach_notes),
       concurrency_mode = coalesce(sqlc.narg(concurrency_mode), concurrency_mode),
       updated_at = datetime('now')
   ```

4. [ ] **Edit `InsertCronWebhook`** (lines 989-992) to add the three columns before `created_by`:
   ```sql
   -- name: InsertCronWebhook :one
   insert into cron_webhooks (url, cron_schedule, instruction, secret, pass_api_key, timeout_seconds, max_depth, max_retries, next_run_at, read_patterns, write_patterns, transform_jsonnet, attach_notes, concurrency_mode, description, created_by)
   values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
   returning *;
   ```

5. [ ] **Edit `UpdateCronWebhook`** (lines 994-1009) — add three `coalesce` lines immediately before `updated_at = datetime('now')`:
   ```sql
       write_patterns = coalesce(sqlc.narg(write_patterns), write_patterns),
       transform_jsonnet = coalesce(sqlc.narg(transform_jsonnet), transform_jsonnet),
       attach_notes = coalesce(sqlc.narg(attach_notes), attach_notes),
       concurrency_mode = coalesce(sqlc.narg(concurrency_mode), concurrency_mode),
       enabled = coalesce(sqlc.narg(enabled), enabled),
   ```
   (Preserve the existing remaining lines — `enabled`, `description`, `updated_at` — in their current order; insert the three new lines after `write_patterns`.)

6. [ ] **Regenerate.** Run:
   ```
   make sqlc && go build ./internal/db/...
   ```
   Expected: clean.

7. [ ] **Run the check to verify it passes.** Run:
   ```
   grep -n "TransformJsonnet\|AttachNotes\|ConcurrencyMode" internal/db/queries.write.sql.go
   ```
   Expected: `InsertWebhookParams`, `UpdateWebhookParams`, `InsertCronWebhookParams`, `UpdateCronWebhookParams` all carry `TransformJsonnet`, `AttachNotes`, `ConcurrencyMode` (the Update params as `*string` via `narg`).

8. [ ] **Commit:**
   ```
   feat(webhooks): add transform/attach/concurrency columns to insert/update queries
   ```

---

### A4 — [Step 2] Add GraphQL inputs + output fields for the three webhook columns, regenerate gqlgen

**Files:**
- Modify: `internal/graph/schema.graphqls`
- Modify: `internal/graph/schema.resolvers.go` (implement the two new `AttachNotes` forceResolver stubs)
- Regenerated: `internal/graph/generated.go`, `internal/graph/model/models_gen.go`, `assets/ui/graphql/queries.ts`

**Steps:**

1. [ ] **Write the failing check.** Run:
   ```
   grep -c "TransformJsonnet" internal/graph/model/models_gen.go
   ```
   Expected: `0` — red.

2. [ ] **Add input fields.** In `internal/graph/schema.graphqls`, append these three lines inside each of the four inputs (before the closing `}`):
   - `ChangeWebhookCreateInput` (after `writePatterns: [String!]` at line 3294):
     ```graphql
     transformJsonnet: String
     attachNotes: [String!]
     concurrencyMode: String
     ```
   - `ChangeWebhookUpdateInput` (after `writePatterns: [String!]` at line 3325): same three lines.
   - `CreateCronWebhookInput` (after `writePatterns: [String!]` at line 3396): same three lines.
   - `UpdateCronWebhookInput` (after `writePatterns: [String!]` at line 3422): same three lines.

3. [ ] **Add output fields** so the inputs round-trip and reconcile (Section C) can read them back. In `AdminChangeWebhook` (line 724, `@goModel(model: "trip2g/internal/db.ChangeWebhook")`), after `writePatterns` (line 747) add:
   ```graphql
   transformJsonnet: String!
   attachNotes: [String!]! @goField(forceResolver: true)
   concurrencyMode: String!
   ```
   In `AdminCronWebhook` (line 776, `@goModel(model: "trip2g/internal/db.CronWebhook")`), after `writePatterns` (line 790) add the same three lines. `transformJsonnet`/`concurrencyMode` auto-bind to the non-null db string columns; `attachNotes` is a JSON-encoded TEXT column so it needs `forceResolver` (mirrors `readPatterns`).

4. [ ] **Regenerate gqlgen.** Run:
   ```
   make gqlgen
   ```
   Expected: `internal/graph/model/models_gen.go` inputs gain `TransformJsonnet *string`, `AttachNotes []string`, `ConcurrencyMode *string`; `internal/graph/schema.resolvers.go` gains two new `panic("not implemented")` stubs `(r *adminChangeWebhookResolver) AttachNotes(...)` and `(r *adminCronWebhookResolver) AttachNotes(...)`.

5. [ ] **Implement the two `AttachNotes` resolvers** in `internal/graph/schema.resolvers.go`, mirroring `ReadPatterns` (line 409 for change, line 598 for cron). Replace the change-webhook stub body with:
   ```go
   func (r *adminChangeWebhookResolver) AttachNotes(ctx context.Context, obj *db.ChangeWebhook) ([]string, error) {
   	var patterns []string
   	err := json.Unmarshal([]byte(obj.AttachNotes), &patterns)
   	return patterns, err
   }
   ```
   And the cron-webhook stub body with:
   ```go
   func (r *adminCronWebhookResolver) AttachNotes(ctx context.Context, obj *db.CronWebhook) ([]string, error) {
   	var patterns []string
   	err := json.Unmarshal([]byte(obj.AttachNotes), &patterns)
   	return patterns, err
   }
   ```

6. [ ] **Run to verify it passes.** Run:
   ```
   go build ./... && grep -n "TransformJsonnet\|ConcurrencyMode\|AttachNotes" internal/graph/model/models_gen.go
   ```
   Expected: build clean; the four input structs carry the three new fields.

7. [ ] **Regenerate frontend types.** With the dev server running (`make air` in a separate terminal), run:
   ```
   make graphqlgen
   ```
   Expected: `npm run graphqlgen` regenerates `assets/ui/graphql/queries.ts` with the new webhook fields. (Backend Go tests do not depend on this; run it per the build-order contract so admin-UI types stay in sync.)

8. [ ] **Commit:**
   ```
   feat(webhooks): expose transform/attach/concurrency in graphql inputs and admin types
   ```

---

### A5 — [Step 2] Wire the three fields + concurrency-mode enum validation into the create CRUD

**Files:**
- Modify: `internal/case/admin/createwebhook/resolve.go`
- Modify: `internal/case/admin/createcronwebhook/resolve.go`
- Create: `internal/case/admin/createwebhook/resolve_test.go`

**Steps:**

1. [ ] **Write the failing test** `internal/case/admin/createwebhook/resolve_test.go`. The existing `Env` has two methods; use a hand mock (matching the package's style). Assert (a) a valid `concurrencyMode` and `attachNotes` round-trip into `InsertWebhookParams`, and (b) an invalid `concurrencyMode` returns an `*model.ErrorPayload` **before** any insert:
   ```go
   package createwebhook_test

   import (
   	"context"
   	"testing"

   	"github.com/stretchr/testify/require"

   	"trip2g/internal/case/admin/createwebhook"
   	"trip2g/internal/db"
   	"trip2g/internal/graph/model"
   	"trip2g/internal/ptr"
   	"trip2g/internal/usertoken"
   )

   type mockEnv struct {
   	inserted *db.InsertWebhookParams
   }

   func (m *mockEnv) CurrentAdminUserToken(_ context.Context) (*usertoken.Data, error) {
   	return &usertoken.Data{ID: 1, Role: "admin"}, nil
   }

   func (m *mockEnv) InsertWebhook(_ context.Context, params db.InsertWebhookParams) (db.ChangeWebhook, error) {
   	m.inserted = &params
   	return db.ChangeWebhook{ID: 7}, nil
   }

   func TestResolve_PersistsConcurrencyAndAttach(t *testing.T) {
   	env := &mockEnv{}
   	out, err := createwebhook.Resolve(context.Background(), env, model.ChangeWebhookCreateInput{
   		URL:             "https://example.com/h",
   		IncludePatterns: []string{"boards/sprint.md"},
   		AttachNotes:     []string{"boards/**", "roles/**"},
   		TransformJsonnet: ptr.To("{ x: 1 }"),
   		ConcurrencyMode: ptr.To("skip"),
   	})
   	require.NoError(t, err)
   	_, isErr := out.(*model.ErrorPayload)
   	require.False(t, isErr)
   	require.NotNil(t, env.inserted)
   	require.Equal(t, "skip", env.inserted.ConcurrencyMode)
   	require.Equal(t, `["boards/**","roles/**"]`, env.inserted.AttachNotes)
   	require.Equal(t, "{ x: 1 }", env.inserted.TransformJsonnet)
   }

   func TestResolve_RejectsBadConcurrencyMode(t *testing.T) {
   	env := &mockEnv{}
   	out, err := createwebhook.Resolve(context.Background(), env, model.ChangeWebhookCreateInput{
   		URL:             "https://example.com/h",
   		IncludePatterns: []string{"boards/sprint.md"},
   		ConcurrencyMode: ptr.To("bogus"),
   	})
   	require.NoError(t, err)
   	ep, ok := out.(*model.ErrorPayload)
   	require.True(t, ok)
   	require.Equal(t, "concurrencyMode", ep.ByFields[0].Name)
   	require.Nil(t, env.inserted, "must not insert on invalid concurrency_mode")
   }
   ```

2. [ ] **Run it to verify it fails.** Run:
   ```
   go test ./internal/case/admin/createwebhook/...
   ```
   Expected: compile error / failure — `ConcurrencyMode`/`AttachNotes`/`TransformJsonnet` not yet read into params, no enum validation.

3. [ ] **Implement in `createwebhook/resolve.go`.** Add the enum validator above `Resolve`:
   ```go
   func validateConcurrencyMode(mode string) *model.ErrorPayload {
   	switch mode {
   	case "allow_overlap", "skip", "queue_one":
   		return nil
   	default:
   		return &model.ErrorPayload{ByFields: []model.FieldMessage{
   			{Name: "concurrencyMode", Value: "must be one of allow_overlap, skip, queue_one"},
   		}}
   	}
   }
   ```
   In `Resolve`, after the existing `writeJSON` marshal block (around current line 102), add:
   ```go
   	attachNotes := input.AttachNotes
   	if attachNotes == nil {
   		attachNotes = []string{}
   	}
   	attachJSON, err := json.Marshal(attachNotes)
   	if err != nil {
   		return nil, fmt.Errorf("failed to marshal attach_notes: %w", err)
   	}

   	transformJsonnet := ""
   	if input.TransformJsonnet != nil {
   		transformJsonnet = *input.TransformJsonnet
   	}

   	concurrencyMode := "allow_overlap"
   	if input.ConcurrencyMode != nil {
   		concurrencyMode = *input.ConcurrencyMode
   	}
   	if cErr := validateConcurrencyMode(concurrencyMode); cErr != nil {
   		return cErr, nil
   	}
   ```
   Then add to the `db.InsertWebhookParams{...}` literal:
   ```go
   		TransformJsonnet: transformJsonnet,
   		AttachNotes:      string(attachJSON),
   		ConcurrencyMode:  concurrencyMode,
   ```

4. [ ] **Mirror in `createcronwebhook/resolve.go`** (same `validateConcurrencyMode` helper, same default/marshal block after its `writeJSON` block around current line 116, same three fields appended to `db.InsertCronWebhookParams`).

5. [ ] **Run to verify pass.** Run:
   ```
   go test ./internal/case/admin/createwebhook/... ./internal/case/admin/createcronwebhook/... && go build ./...
   ```
   Expected: PASS, build clean.

6. [ ] **Commit:**
   ```
   feat(webhooks): persist transform/attach/concurrency in create CRUD with enum validation
   ```

---

### A6 — [Step 2] Wire the three fields + enum validation into the update CRUD

**Files:**
- Modify: `internal/case/admin/updatewebhook/resolve.go`
- Modify: `internal/case/admin/updatecronwebhook/resolve.go`
- Create: `internal/case/admin/updatewebhook/resolve_test.go`

**Steps:**

1. [ ] **Write the failing test** `internal/case/admin/updatewebhook/resolve_test.go`:
   ```go
   package updatewebhook_test

   import (
   	"context"
   	"testing"

   	"github.com/stretchr/testify/require"

   	"trip2g/internal/case/admin/updatewebhook"
   	"trip2g/internal/db"
   	"trip2g/internal/graph/model"
   	"trip2g/internal/ptr"
   	"trip2g/internal/usertoken"
   )

   type mockEnv struct{ updated *db.UpdateWebhookParams }

   func (m *mockEnv) CurrentAdminUserToken(_ context.Context) (*usertoken.Data, error) {
   	return &usertoken.Data{ID: 1, Role: "admin"}, nil
   }

   func (m *mockEnv) UpdateWebhook(_ context.Context, p db.UpdateWebhookParams) (db.ChangeWebhook, error) {
   	m.updated = &p
   	return db.ChangeWebhook{ID: p.ID}, nil
   }

   func TestResolve_UpdatesConcurrencyAndAttach(t *testing.T) {
   	env := &mockEnv{}
   	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
   		ID:              7,
   		AttachNotes:     []string{"boards/**"},
   		TransformJsonnet: ptr.To("{ y: 2 }"),
   		ConcurrencyMode: ptr.To("queue_one"),
   	})
   	require.NoError(t, err)
   	_, isErr := out.(*model.ErrorPayload)
   	require.False(t, isErr)
   	require.NotNil(t, env.updated.ConcurrencyMode)
   	require.Equal(t, "queue_one", *env.updated.ConcurrencyMode)
   	require.NotNil(t, env.updated.AttachNotes)
   	require.Equal(t, `["boards/**"]`, *env.updated.AttachNotes)
   	require.NotNil(t, env.updated.TransformJsonnet)
   	require.Equal(t, "{ y: 2 }", *env.updated.TransformJsonnet)
   }

   func TestResolve_RejectsBadConcurrencyMode(t *testing.T) {
   	env := &mockEnv{}
   	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
   		ID:              7,
   		ConcurrencyMode: ptr.To("nope"),
   	})
   	require.NoError(t, err)
   	ep, ok := out.(*model.ErrorPayload)
   	require.True(t, ok)
   	require.Equal(t, "concurrencyMode", ep.ByFields[0].Name)
   	require.Nil(t, env.updated, "must not update on invalid concurrency_mode")
   }
   ```

2. [ ] **Run it to verify it fails.** Run:
   ```
   go test ./internal/case/admin/updatewebhook/...
   ```
   Expected: failure (fields not wired, no enum validation).

3. [ ] **Implement in `updatewebhook/resolve.go`.** Add the same `validateConcurrencyMode` helper as A5. In `Resolve`, after `boundsErr` is checked, add the conditional enum validation:
   ```go
   	if input.ConcurrencyMode != nil {
   		if cErr := validateConcurrencyMode(*input.ConcurrencyMode); cErr != nil {
   			return cErr, nil
   		}
   	}
   ```
   Add to the `db.UpdateWebhookParams{...}` literal:
   ```go
   		TransformJsonnet: input.TransformJsonnet,
   		ConcurrencyMode:  input.ConcurrencyMode,
   ```
   And after the existing `marshalOptionalJSON` block for `write_patterns`, add:
   ```go
   	params.AttachNotes, err = marshalOptionalJSON(input.AttachNotes)
   	if err != nil {
   		return nil, fmt.Errorf("failed to marshal attach_notes: %w", err)
   	}
   ```
   (`TransformJsonnet`/`ConcurrencyMode` are `*string` on both the input and the `narg` params, so they pass through directly; `AttachNotes` uses the existing `marshalOptionalJSON` helper.)

4. [ ] **Mirror in `updatecronwebhook/resolve.go`** (same helper, same conditional enum check after `boundsErr`, same `TransformJsonnet`/`ConcurrencyMode` direct assignment in `db.UpdateCronWebhookParams`, same `params.AttachNotes` marshal block).

5. [ ] **Run to verify pass.** Run:
   ```
   go test ./internal/case/admin/updatewebhook/... ./internal/case/admin/updatecronwebhook/... && go build ./...
   ```
   Expected: PASS, build clean.

6. [ ] **Commit:**
   ```
   feat(webhooks): persist transform/attach/concurrency in update CRUD with enum validation
   ```

---

### A7 — [Step 3, THE security fix] Enforce `WebhookWritePatterns` server-side in `updateNotes`

Today `WebhookReadPatterns`/`WebhookWritePatterns` are *set* by the shortapitoken (`internal/case/checkapikey/resolve.go:131-133`) but have **zero consumers** — the scope is decorative. This task makes the write scope real and adds the two free-function accessors the security backbone uses verbatim.

**Files:**
- Modify: `internal/appreq/request.go` (add `WebhookReadPatterns(ctx)` + `WebhookWritePatterns(ctx)` accessors)
- Modify: `internal/case/updatenotes/resolve.go`
- Modify: `internal/case/updatenotes/resolve_test.go`

**Steps:**

1. [ ] **Write the failing test.** Append to `internal/case/updatenotes/resolve_test.go` (it already uses `package updatenotes_test` with the hand `mockEnv`; add the imports `"trip2g/internal/appreq"` and `"trip2g/internal/db"` if not present):
   ```go
   func TestResolve_WriteDeniedOutOfPattern(t *testing.T) {
   	req := &appreq.Request{WebhookWritePatterns: []string{"boards/**"}}
   	ctx := appreq.NewContext(context.Background(), req)

   	env := &mockEnv{
   		latestNoteViews: func() *appmodel.NoteViews { return appmodel.NewNoteViews() },
   		// insertNote intentionally nil — must NOT be reached.
   	}

   	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
   		ApiKey: db.ApiKey{},
   		Changes: []model.NoteChangeInput{
   			{Upsert: &model.NoteChangeUpsertInput{Path: "secrets/private.md", Content: "x"}},
   		},
   	})
   	require.NoError(t, err)
   	ep, ok := out.(*model.ErrorPayload)
   	require.True(t, ok)
   	require.Contains(t, ep.Message, "write denied for path: secrets/private.md")
   }

   func TestResolve_WriteAllowedInPattern(t *testing.T) {
   	req := &appreq.Request{WebhookWritePatterns: []string{"boards/**"}}
   	ctx := appreq.NewContext(context.Background(), req)

   	called := false
   	env := &mockEnv{
   		latestNoteViews:    func() *appmodel.NoteViews { return appmodel.NewNoteViews() },
   		insertNote:         func(_ context.Context, _ appmodel.RawNote) (int64, error) { called = true; return 1, nil },
   		prepareLatestNotes: noopPrepare,
   		handleLatestNotesAfterSave: noopHandle,
   	}

   	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
   		ApiKey: db.ApiKey{},
   		Changes: []model.NoteChangeInput{
   			{Upsert: &model.NoteChangeUpsertInput{Path: "boards/sprint.md", Content: "x"}},
   		},
   	})
   	require.NoError(t, err)
   	require.IsType(t, model.UpdateNotesSuccessPayload{}, out)
   	require.True(t, called)
   }
   ```
   (The existing tests with no appreq context exercise the `len(wp)==0` backward-compat path — unscoped callers are unaffected.)

2. [ ] **Run it to verify it fails.** Run:
   ```
   go test ./internal/case/updatenotes/...
   ```
   Expected: `TestResolve_WriteDeniedOutOfPattern` fails (write proceeds / mock panics on nil `insertNote`).

3. [ ] **Add the accessors** to `internal/appreq/request.go` (after `FromCtx`, ~line 105). `appreq` does not import `webhookutil`, and `webhookutil` does not import `appreq`, so the call-site `MatchesAny` stays in the resolver:
   ```go
   // WebhookWritePatterns returns the write-scope globs stamped on the request
   // by a shortapitoken delivery. Empty/nil means unscoped (no enforcement).
   func WebhookWritePatterns(ctx context.Context) []string {
   	req, err := FromCtx(ctx)
   	if err != nil {
   		return nil
   	}
   	return req.WebhookWritePatterns
   }

   // WebhookReadPatterns returns the read-scope globs stamped on the request
   // by a shortapitoken delivery. Empty/nil means unscoped (no enforcement).
   func WebhookReadPatterns(ctx context.Context) []string {
   	req, err := FromCtx(ctx)
   	if err != nil {
   		return nil
   	}
   	return req.WebhookReadPatterns
   }
   ```

4. [ ] **Implement enforcement** in `internal/case/updatenotes/resolve.go`. Add imports `"trip2g/internal/appreq"` and `"trip2g/internal/webhookutil"`. Add a helper above `Resolve`:
   ```go
   func webhookWriteDenied(ctx context.Context, path string) *model.ErrorPayload {
   	if wp := appreq.WebhookWritePatterns(ctx); len(wp) > 0 && !webhookutil.MatchesAny(path, wp) {
   		return &model.ErrorPayload{Message: "write denied for path: " + path}
   	}
   	return nil
   }
   ```
   Inside the `for _, change := range input.Changes` loop, add the guard at the top of each write branch — for `change.Upsert != nil` (after `upsert := change.Upsert`):
   ```go
   			if denied := webhookWriteDenied(ctx, upsert.Path); denied != nil {
   				return denied, nil
   			}
   ```
   for `change.Patch != nil` (after `patch := change.Patch`):
   ```go
   			if denied := webhookWriteDenied(ctx, patch.Path); denied != nil {
   				return denied, nil
   			}
   ```
   and for `change.Hide != nil` (after `hide := change.Hide`):
   ```go
   			if denied := webhookWriteDenied(ctx, hide.Path); denied != nil {
   				return denied, nil
   			}
   ```

5. [ ] **Run to verify pass.** Run:
   ```
   go test ./internal/case/updatenotes/... ./internal/appreq/... && go build ./...
   ```
   Expected: all PASS (including the pre-existing unscoped tests), build clean.

6. [ ] **Commit:**
   ```
   feat(security): enforce webhook write_patterns in updateNotes
   ```

---

### A8 — [Step 3] Mirror read-scope enforcement in `note(path:)` and `search`

Make `WebhookReadPatterns` real on the read side so a scoped delivery cannot read notes outside its read globs.

**Files:**
- Modify: `internal/graph/schema.resolvers.go` (`queryResolver.Note`, line 3094)
- Modify: `internal/case/sitesearch/resolve.go`
- Create: `internal/case/sitesearch/scope_test.go`

**Steps:**

1. [ ] **Write the failing test** `internal/case/sitesearch/scope_test.go` (separate `_test` package + moq, since the existing in-package test only covers pure helpers). Generate the mock:
   ```
   cd internal/case/sitesearch && go tool github.com/matryer/moq -out scope_mocks_test.go -pkg sitesearch_test . Env
   ```
   Then:
   ```go
   package sitesearch_test

   //go:generate go tool github.com/matryer/moq -out scope_mocks_test.go -pkg sitesearch_test . Env

   import (
   	"context"
   	"testing"

   	"github.com/stretchr/testify/require"

   	"trip2g/internal/appreq"
   	"trip2g/internal/case/sitesearch"
   	"trip2g/internal/features"
   	"trip2g/internal/graph/model"
   	"trip2g/internal/logger"
   	appmodel "trip2g/internal/model"
   	"trip2g/internal/usertoken"
   )

   func TestResolve_ReadPatternsOmitOutOfScope(t *testing.T) {
   	inScope := appmodel.SearchResult{URL: "/boards/sprint", NoteView: &appmodel.NoteView{Path: "boards/sprint.md", Permalink: "/boards/sprint"}}
   	outScope := appmodel.SearchResult{URL: "/secrets/p", NoteView: &appmodel.NoteView{Path: "secrets/p.md", Permalink: "/secrets/p"}}

   	env := &EnvMock{
   		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) { return &usertoken.Data{}, nil },
   		SiteConfigFunc:       func(_ context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
   		SearchLiveNotesFunc:  func(_ string) ([]appmodel.SearchResult, error) { return []appmodel.SearchResult{inScope, outScope}, nil },
   		FeaturesFunc:         func() features.Features { return features.Features{} },
   		OpenAIFunc:           func() *openaiClientPlaceholder { return nil }, // see note below
   		CanReadNoteFunc:      func(_ context.Context, _ *appmodel.NoteView) (bool, error) { return true, nil },
   		LoggerFunc:           func() logger.Logger { return &logger.DummyLogger{} },
   	}

   	req := &appreq.Request{WebhookReadPatterns: []string{"boards/**"}}
   	ctx := appreq.NewContext(context.Background(), req)

   	conn, err := sitesearch.Resolve(ctx, env, model.SearchInput{Query: "x"})
   	require.NoError(t, err)
   	require.Len(t, conn.Nodes, 1)
   	require.Equal(t, "boards/sprint.md", conn.Nodes[0].NoteView.Path)
   }
   ```
   Note: the `OpenAI()` mock returns the real `*openai.Client` type — use the moq-generated signature (import `trip2g/internal/openai` and return `nil`). With `features.Features{}` zero value, `VectorSearch.Enabled` is false, so the vector branch is skipped and `OpenAI()` is never dereferenced. Adjust the `OpenAIFunc` return type to the moq-generated `func() *openai.Client { return nil }`.

2. [ ] **Run it to verify it fails.** Run:
   ```
   go test ./internal/case/sitesearch/...
   ```
   Expected: failure — both `inScope` and `outScope` returned (`len == 2`).

3. [ ] **Implement read filtering** in `internal/case/sitesearch/resolve.go`. Add imports `"trip2g/internal/appreq"` and `"trip2g/internal/webhookutil"`. Inside the `for _, res := range results` permission loop (starts line 89), as the first statement after the existing `if res.NoteView != nil {` system/exclude check, add:
   ```go
   			if rp := appreq.WebhookReadPatterns(ctx); len(rp) > 0 && !webhookutil.MatchesAny(res.NoteView.Path, rp) {
   				continue
   			}
   ```
   (Place it before the `CanReadNote` call so out-of-scope notes are fully omitted, not cropped.)

4. [ ] **Implement read filtering** in `queryResolver.Note` (`internal/graph/schema.resolvers.go`, line 3094). After the block that resolves `path` (the `if input.PathID != nil { ... }` block, before constructing `rendernotepage.Request`), add:
   ```go
   	if rp := appreq.WebhookReadPatterns(ctx); len(rp) > 0 && !webhookutil.MatchesAny(path, rp) {
   		return nil, nil
   	}
   ```
   `appreq` is already imported in this file; add `"trip2g/internal/webhookutil"` to the import block if not present.

5. [ ] **Run to verify pass.** Run:
   ```
   go test ./internal/case/sitesearch/... && go build ./...
   ```
   Expected: PASS (`len == 1`), build clean.

6. [ ] **Commit:**
   ```
   feat(security): enforce webhook read_patterns in note and search resolvers
   ```

---

### A9 — [Step 3] Remove the 60-minute scoped-token TTL floor

The scoped token is a long-lived bearer today: `deliverchangewebhook/resolve.go:83-85` floors the TTL to 60 minutes. Replace the floor with `TimeoutSeconds + small margin` so the token expires shortly after the delivery window. (TLS-for-CallbackURL is an ops requirement, documented in Section C docs.)

**Files:**
- Modify: `internal/case/backjob/deliverchangewebhook/resolve.go`
- Modify: `internal/case/backjob/delivercronwebhook/resolve.go`
- Modify: `internal/case/backjob/deliverchangewebhook/resolve_test.go`

**Steps:**

1. [ ] **Write the failing test.** Append to `internal/case/backjob/deliverchangewebhook/resolve_test.go` a test that captures the delivered `api_token` and asserts its lifetime is short (decode the JWT payload segment; no verification needed). Add a base64/json/strings helper inline:
   ```go
   func tokenLifetimeSeconds(t *testing.T, jwtStr string) int64 {
   	t.Helper()
   	parts := strings.Split(jwtStr, ".")
   	require.Len(t, parts, 3)
   	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
   	require.NoError(t, err)
   	var claims struct {
   		Exp int64 `json:"exp"`
   		Iat int64 `json:"iat"`
   	}
   	require.NoError(t, json.Unmarshal(raw, &claims))
   	return claims.Exp - claims.Iat
   }

   func TestResolve_TokenTTLHasNoSixtyMinuteFloor(t *testing.T) {
   	var body []byte
   	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
   		body, _ = io.ReadAll(r.Body)
   		w.WriteHeader(http.StatusAccepted)
   	}))
   	defer srv.Close()

   	env := baseEnv(t, srv.URL, nil)
   	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
   		return db.ChangeWebhook{ID: id, Url: srv.URL, TimeoutSeconds: 10, PassApiKey: true, ReadPatterns: "[]", WritePatterns: "[]"}, nil
   	}

   	err := deliverchangewebhook.Resolve(context.Background(), env, handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 100, Attempt: 1})
   	require.NoError(t, err)

   	var payload map[string]any
   	require.NoError(t, json.Unmarshal(body, &payload))
   	tok, _ := payload["api_token"].(string)
   	require.NotEmpty(t, tok)
   	require.Less(t, tokenLifetimeSeconds(t, tok), int64(300), "60-min floor must be gone; TTL ~= timeout + margin")
   }
   ```
   Add imports `"encoding/base64"`, `"strings"` to the test file.

2. [ ] **Run it to verify it fails.** Run:
   ```
   go test ./internal/case/backjob/deliverchangewebhook/...
   ```
   Expected: failure — lifetime is 3600s (the 60-min floor).

3. [ ] **Implement.** In `deliverchangewebhook/resolve.go`, add a package-level const near the top:
   ```go
   // tokenTTLMargin is the small grace window added to the delivery timeout for
   // the scoped write-back token. Replaces the former 60-minute floor.
   const tokenTTLMargin = 30 * time.Second
   ```
   Replace lines 83-85:
   ```go
   		ttl := time.Duration(wh.TimeoutSeconds) * time.Second
   		if ttl < 60*time.Minute {
   			ttl = 60 * time.Minute
   		}
   ```
   with:
   ```go
   		ttl := time.Duration(wh.TimeoutSeconds)*time.Second + tokenTTLMargin
   ```

4. [ ] **Mirror in `delivercronwebhook/resolve.go`** — add the same `tokenTTLMargin` const and replace its lines 99-102 with `ttl := time.Duration(wh.TimeoutSeconds)*time.Second + tokenTTLMargin`.

5. [ ] **Run to verify pass.** Run:
   ```
   go test ./internal/case/backjob/... && go build ./...
   ```
   Expected: PASS, build clean.

6. [ ] **Commit:**
   ```
   fix(security): drop 60-minute floor on scoped webhook token TTL
   ```

---

### A10 — [Step 3] Redact `api_token` and `secrets` from the logged `request_body`

The full payload — including the scoped `api_token` and decrypted `secrets` — is written verbatim into `webhook_delivery_logs.request_body` (`deliverchangewebhook/resolve.go:124`, cron `:139`). The HTTP body to the fleet must still carry them (the fleet needs the scoped token to write back), but the persisted log must not.

**Files:**
- Modify: `internal/case/backjob/deliverchangewebhook/resolve.go`
- Modify: `internal/case/backjob/delivercronwebhook/resolve.go`
- Modify: `internal/case/backjob/deliverchangewebhook/resolve_test.go`

**Steps:**

1. [ ] **Write the failing test.** Append to `internal/case/backjob/deliverchangewebhook/resolve_test.go` a test that captures BOTH the HTTP body (must contain the token + secret) and the logged `RequestBody` (must NOT):
   ```go
   func TestResolve_LoggedRequestBodyRedactsTokenAndSecrets(t *testing.T) {
   	var httpBody []byte
   	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
   		httpBody, _ = io.ReadAll(r.Body)
   		w.WriteHeader(http.StatusAccepted)
   	}))
   	defer srv.Close()

   	env := baseEnv(t, srv.URL, map[string]string{"auth_token": "tok-abc"})
   	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
   		return db.ChangeWebhook{ID: id, Url: srv.URL, TimeoutSeconds: 10, PassApiKey: true, ReadPatterns: "[]", WritePatterns: "[]"}, nil
   	}
   	var loggedBody string
   	env.InsertWebhookDeliveryLogFunc = func(_ context.Context, p db.InsertWebhookDeliveryLogParams) error {
   		if p.RequestBody != nil {
   			loggedBody = *p.RequestBody
   		}
   		return nil
   	}

   	err := deliverchangewebhook.Resolve(context.Background(), env, handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 100, Attempt: 1})
   	require.NoError(t, err)

   	// HTTP body to the fleet keeps the credential + secrets.
   	require.Contains(t, string(httpBody), "api_token")
   	require.Contains(t, string(httpBody), "tok-abc")
   	// Logged body must not leak them.
   	require.NotContains(t, loggedBody, "tok-abc")
   	require.NotContains(t, loggedBody, "api_token")
   }
   ```

2. [ ] **Run it to verify it fails.** Run:
   ```
   go test ./internal/case/backjob/deliverchangewebhook/... -run LoggedRequestBodyRedacts
   ```
   Expected: failure — `loggedBody` contains `tok-abc` and `api_token`.

3. [ ] **Implement** in `deliverchangewebhook/resolve.go`. Replace the single line `requestBodyStr := string(payloadBytes)` (line 124) with a redacted re-marshal of the payload struct (the `payload` value is in scope; `changeWebhookPayload` is a value type so the copy is cheap):
   ```go
   	// Persist a redacted copy: never log the scoped token or secret values.
   	redacted := payload
   	redacted.APIToken = ""
   	redacted.Secrets = nil
   	redactedBytes, redErr := json.Marshal(redacted)
   	if redErr != nil {
   		redactedBytes = []byte("{}")
   	}
   	requestBodyStr := string(redactedBytes)
   ```
   The HTTP send above already uses `payloadBytes` (the full body) — leave it untouched.

4. [ ] **Mirror in `delivercronwebhook/resolve.go`** — replace its `requestBodyStr := string(payloadBytes)` (line 138) with the same redacted re-marshal of its `payload` (`cronWebhookPayload`, which also has `APIToken` and `Secrets` fields).

5. [ ] **Run to verify pass.** Run:
   ```
   go test ./internal/case/backjob/... && go build ./...
   ```
   Expected: PASS, build clean.

6. [ ] **Commit:**
   ```
   fix(security): redact api_token and secrets from logged webhook request_body
   ```

> Note for Section B: the jsonnet ExtVar half of "api_token/secrets out of the jsonnet ExtVar" is enforced structurally there by `transformExtVars` (which never includes `api_token`/`secrets`). This task closes the log-leak half, which is the only concrete surface present before the transform seam lands.

---

**Relevant files for this section (all under `/home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime/`):** `db/migrations/20260628120000_*.sql`, `db/migrations/20260628120100_*.sql`, `queries.write.sql`, `internal/graph/schema.graphqls`, `internal/graph/schema.resolvers.go`, `internal/case/admin/{createwebhook,updatewebhook,createcronwebhook,updatecronwebhook}/resolve.go`, `internal/appreq/request.go`, `internal/case/updatenotes/resolve.go`, `internal/case/sitesearch/resolve.go`, `internal/case/backjob/{deliverchangewebhook,delivercronwebhook}/resolve.go`, and `assets/ui/user/live/live.view.ts` (Step-0 blocker, inspect-only).


---

## Section B — Steps 4-6: jsonneteval+transform, attach_notes gate+materialize, concurrency+janitor+attribution+spend

> **Prerequisites (delivered by Section A, assume present):** migrations M1 (`20260628120000_*`) + M2 (`20260628120100_*`) are applied and `make sqlc` has run, so `db.ChangeWebhook`/`db.CronWebhook` already carry `TransformJsonnet`, `AttachNotes`, `ConcurrencyMode`; `db.ChangeWebhookDelivery`/`db.CronWebhookDelivery` already carry `StartedAt`, `HeartbeatAt`, `TokensUsed`, `Steps`; `note_versions` already has `created_by_delivery_kind` / `created_by_delivery_id`; and the `schema.graphqls` Create/Update inputs already expose `transformJsonnet`/`attachNotes`/`concurrencyMode` with `concurrency_mode` enum-validated in CRUD. Section B builds the runtime behavior on top of those columns. All Go work follows TDD (red → green → refactor), table-driven where natural, `testify/require`, moq where the package already uses it. Commits use the project convention `feat(scope): desc` — one line, no body, no co-author.

---

### Task B1 — `internal/jsonneteval` shared seam (NewVM / EvalJSON / Validate)

**Files**
- Create: `internal/jsonneteval/jsonneteval.go`
- Test: `internal/jsonneteval/jsonneteval_test.go`

**Steps**

1. [ ] **Write the failing test.** Create `internal/jsonneteval/jsonneteval_test.go`:

```go
package jsonneteval_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/jsonneteval"
)

func TestNewVM_MaxStack(t *testing.T) {
	vm := jsonneteval.NewVM()
	require.NotNil(t, vm)
	require.Equal(t, 500, vm.MaxStack)
}

func TestEvalJSON_Identity(t *testing.T) {
	src := `std.parseJson(std.extVar("payload"))`
	in := `{"a":1,"b":["x","y"]}`
	out, err := jsonneteval.EvalJSON(src, map[string]string{"payload": in})
	require.NoError(t, err)

	var got, want map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	require.NoError(t, json.Unmarshal([]byte(in), &want))
	require.Equal(t, want, got)
}

func TestEvalJSON_Remap(t *testing.T) {
	src := `local p = std.parseJson(std.extVar("change")); { renamed: p }`
	out, err := jsonneteval.EvalJSON(src, map[string]string{"change": `[{"path":"a.md"}]`})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	require.Contains(t, got, "renamed")
}

func TestEvalJSON_RuntimeError(t *testing.T) {
	_, err := jsonneteval.EvalJSON(`error "boom"`, nil)
	require.Error(t, err)
}

func TestValidate(t *testing.T) {
	require.NoError(t, jsonneteval.Validate(`{ ok: std.extVar("payload") }`,
		map[string]string{"payload": "{}"}))
	require.Error(t, jsonneteval.Validate(`}{ not jsonnet`, nil))
}
```

2. [ ] **Run it to verify it fails.** `go test ./internal/jsonneteval/...` → expected: `build failed` / `package jsonneteval is not in std` (the package does not exist yet).

3. [ ] **Implement.** Create `internal/jsonneteval/jsonneteval.go`:

```go
// Package jsonneteval is the single source of truth for evaluating jsonnet
// snippets against JSON ext-vars (outbound webhook transforms, frontmatter
// patches). It owns the safe VM stack limit so every caller is consistent.
package jsonneteval

import (
	"encoding/json"
	"fmt"

	jsonnet "github.com/google/go-jsonnet"
)

// NewVM returns a jsonnet VM with a safe stack limit (no IO; go-jsonnet is pure).
func NewVM() *jsonnet.VM {
	vm := jsonnet.MakeVM()
	vm.MaxStack = 500 // Prevent stack overflow from recursive jsonnet.
	return vm
}

// EvalJSON evaluates src with the given ext-vars bound via std.extVar and
// returns the result as raw JSON. Each ext-var value is bound verbatim (the
// caller decides whether a value is a JSON string to std.parseJson).
func EvalJSON(src string, extVars map[string]string) (json.RawMessage, error) {
	vm := NewVM()
	for k, v := range extVars {
		vm.ExtVar(k, v)
	}

	out, err := vm.EvaluateAnonymousSnippet("transform", src)
	if err != nil {
		return nil, fmt.Errorf("evaluate jsonnet: %w", err)
	}

	if !json.Valid([]byte(out)) {
		return nil, fmt.Errorf("jsonnet output is not valid JSON")
	}

	return json.RawMessage(out), nil
}

// Validate compiles and runs src against sampleExtVars, discarding the result.
// Used at CRUD time to reject a transform that cannot even evaluate.
func Validate(src string, sampleExtVars map[string]string) error {
	_, err := EvalJSON(src, sampleExtVars)
	return err
}
```

4. [ ] **Run to verify pass.** `go test ./internal/jsonneteval/... -count=1` → expected: `ok  trip2g/internal/jsonneteval`.

5. [ ] **Commit.** `feat(jsonneteval): shared jsonnet VM with EvalJSON and Validate`

---

### Task B2 — `frontmatterpatch.NewVM` delegates to `jsonneteval.NewVM`

DRY: collapse the duplicate `MaxStack=500` VM constructor. `frontmatterpatch.Evaluate` is untouched; its existing tests remain the guard.

**Files**
- Modify: `internal/frontmatterpatch/evaluate.go`

**Steps**

1. [ ] **Run the guard test first (must already pass).** `go test ./internal/frontmatterpatch/... -count=1` → expected: `ok`. This is the regression oracle for the refactor.

2. [ ] **Implement.** In `internal/frontmatterpatch/evaluate.go` replace the local `NewVM` body and its now-unused `jsonnet` import. Change the import block:

```go
import (
	"encoding/json"
	"fmt"

	jsonnet "github.com/google/go-jsonnet"

	"trip2g/internal/jsonneteval"
)
```

and replace the function:

```go
// NewVM creates a new jsonnet VM with safe stack limits.
// Delegates to jsonneteval so the MaxStack limit has one source of truth.
func NewVM() *jsonnet.VM {
	return jsonneteval.NewVM()
}
```

(The `jsonnet` import stays — it is still referenced by `Evaluate`'s `vm *jsonnet.VM` parameter.)

3. [ ] **Run to verify pass.** `go test ./internal/frontmatterpatch/... ./internal/jsonneteval/... ./internal/mdloader/... -count=1` → expected: all `ok` (mdloader uses `frontmatterpatch.NewVM` at `loader.go:203`).

4. [ ] **Commit.** `refactor(frontmatterpatch): delegate NewVM to jsonneteval`

---

### Task B3 — Apply `transform_jsonnet` in `deliverchangewebhook` (between Marshal and SignHMAC)

The transform reshapes the outbound body strictly between `json.Marshal(payload)` (resolve.go:101) and `webhookutil.SignHMAC` (:107) so both the signature and the logged `request_body` cover the transformed bytes. `transformExtVars` exposes `change`/`attached_notes`/`meta` but never `api_token` or `secrets`. A runtime error aborts delivery via `handleDeliveryError` (never sends a half-built request).

**Files**
- Modify: `internal/case/backjob/deliverchangewebhook/resolve.go`
- Test: `internal/case/backjob/deliverchangewebhook/resolve_test.go`

**Steps**

1. [ ] **Write the failing test.** Append to `internal/case/backjob/deliverchangewebhook/resolve_test.go`:

```go
func TestTransformExtVars_RedactsSecrets(t *testing.T) {
	payload := []byte(`{"changes":[{"path":"a.md"}],"attached_notes":[{"path":"b.md"}],` +
		`"api_token":"SECRET-TOKEN","secrets":{"k":"v"}}`)
	ev := deliverchangewebhook.TransformExtVarsForTest(payload)

	require.Contains(t, ev, "change")
	require.Contains(t, ev, "attached_notes")
	require.NotContains(t, ev, "api_token")
	require.NotContains(t, ev, "secrets")
	for _, v := range ev {
		require.NotContains(t, v, "SECRET-TOKEN")
	}
}

func TestResolve_TransformJsonnet_AppliedAndSigned(t *testing.T) {
	var body []byte
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		gotSig = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
			ID:               id,
			Url:              srv.URL,
			Secret:           "hook-secret",
			TimeoutSeconds:   10,
			WritePatterns:    "[]",
			ReadPatterns:     "[]",
			TransformJsonnet: `{ marker: "transformed", n: std.length(std.parseJson(std.extVar("change"))) }`,
		}, nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{
			WebhookID:  1,
			DeliveryID: 100,
			Attempt:    1,
			Changes:    []handlenotewebhooks.ChangeInfo{{Path: "a.md", Event: "update"}},
		})
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "transformed", out["marker"])
	require.EqualValues(t, 1, out["n"])
	// HMAC must cover the TRANSFORMED bytes the server actually received.
	require.Equal(t, webhookutil.SignHMAC(body, "hook-secret"), gotSig)
}
```

Add the `webhookutil` import to the test file's import block.

2. [ ] **Run it to verify it fails.** `go test ./internal/case/backjob/deliverchangewebhook/... -run 'Transform' -count=1` → expected: `undefined: deliverchangewebhook.TransformExtVarsForTest` and `unknown field TransformJsonnet`/build failure.

3. [ ] **Implement.** In `internal/case/backjob/deliverchangewebhook/resolve.go` add the `jsonneteval` import (`"trip2g/internal/jsonneteval"`), then insert the transform block immediately before line 107 (`signature := webhookutil.SignHMAC(...)`):

```go
	// Apply outbound transform (if configured) strictly between marshal and
	// sign, so both the HMAC and the logged request_body cover the result.
	if wh.TransformJsonnet != "" {
		out, terr := jsonneteval.EvalJSON(wh.TransformJsonnet, transformExtVars(payloadBytes))
		if terr != nil {
			// Never send a half-built request.
			handleDeliveryError(ctx, env, params,
				webhookutil.DeliveryResult{Err: fmt.Errorf("transform_jsonnet: %w", terr)}, wh)
			return nil
		}
		payloadBytes = out
	}
```

Add the helper + its test seam at the bottom of the file:

```go
// transformExtVars exposes the change/attached-notes/meta of the outbound
// payload to the jsonnet transform. The api_token and secrets are intentionally
// NEVER exposed (they must not reach the transform or the logged request_body).
func transformExtVars(payloadBytes []byte) map[string]string {
	var p map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return map[string]string{}
	}

	ev := make(map[string]string, 3)
	if v, ok := p["changes"]; ok {
		ev["change"] = string(v)
	}
	if v, ok := p["attached_notes"]; ok {
		ev["attached_notes"] = string(v)
	}
	if v, ok := p["meta"]; ok {
		ev["meta"] = string(v)
	}
	return ev
}

// TransformExtVarsForTest exposes transformExtVars to the external test package.
func TransformExtVarsForTest(payloadBytes []byte) map[string]string {
	return transformExtVars(payloadBytes)
}
```

4. [ ] **Run to verify pass.** `go test ./internal/case/backjob/deliverchangewebhook/... -count=1` → expected: `ok` (new tests pass; existing `TestResolve_SecretsInjectedInPayload`/`TestResolve_NoSecrets_FieldOmitted` still pass — empty `TransformJsonnet` is a byte-for-byte no-op).

5. [ ] **Commit.** `feat(deliverchangewebhook): apply transform_jsonnet to outbound payload`

---

### Task B4 — Apply `transform_jsonnet` in `delivercronwebhook` (cron twin)

Identical seam in the cron deliver job.

**Files**
- Modify: `internal/case/backjob/delivercronwebhook/resolve.go`
- Test: `internal/case/backjob/delivercronwebhook/resolve_test.go`

**Steps**

1. [ ] **Write the failing test.** Append to `internal/case/backjob/delivercronwebhook/resolve_test.go` (mirror B3; the cron payload top-level key is `instruction`/`response_schema` — the transform here just re-shapes whatever is present):

```go
func TestResolve_CronTransformJsonnet_AppliedAndSigned(t *testing.T) {
	var body []byte
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		gotSig = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil) // reuse the package's existing baseEnv helper
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID:               id,
			Url:              srv.URL,
			Secret:           "cron-secret",
			TimeoutSeconds:   10,
			WritePatterns:    "[]",
			ReadPatterns:     "[]",
			TransformJsonnet: `{ marker: "cron-transformed" }`,
		}, nil
	}

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 7, Attempt: 1})
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "cron-transformed", out["marker"])
	require.Equal(t, webhookutil.SignHMAC(body, "cron-secret"), gotSig)
}
```

Confirm the package's existing `resolve_test.go` already defines `baseEnv`; if it does not, mirror the change-webhook `baseEnv` factory. Add the `webhookutil` import.

2. [ ] **Run it to verify it fails.** `go test ./internal/case/backjob/delivercronwebhook/... -run 'CronTransform' -count=1` → expected: `unknown field TransformJsonnet`/build failure.

3. [ ] **Implement.** In `internal/case/backjob/delivercronwebhook/resolve.go` add the `jsonneteval` import, then insert before line 123 (`signature := webhookutil.SignHMAC(...)`):

```go
	if wh.TransformJsonnet != "" {
		out, terr := jsonneteval.EvalJSON(wh.TransformJsonnet, transformExtVars(payloadBytes))
		if terr != nil {
			handleCronDeliveryError(ctx, env, params,
				webhookutil.DeliveryResult{Err: fmt.Errorf("transform_jsonnet: %w", terr)}, wh)
			return nil
		}
		payloadBytes = out
	}
```

Add the same `transformExtVars` helper at the bottom of the file (cron payload has no `changes`/`attached_notes` yet; the helper simply yields an empty map then, which is correct):

```go
// transformExtVars exposes non-secret payload fields to the jsonnet transform.
// api_token and secrets are never exposed.
func transformExtVars(payloadBytes []byte) map[string]string {
	var p map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return map[string]string{}
	}
	ev := make(map[string]string, 3)
	for _, k := range []string{"changes", "attached_notes", "meta"} {
		if v, ok := p[k]; ok {
			key := k
			if k == "changes" {
				key = "change"
			}
			ev[key] = string(v)
		}
	}
	return ev
}
```

4. [ ] **Run to verify pass.** `go test ./internal/case/backjob/delivercronwebhook/... -count=1` → expected: `ok`.

5. [ ] **Commit.** `feat(delivercronwebhook): apply transform_jsonnet to outbound payload`

---

### Task B5 — Validate `transform_jsonnet` at CRUD (createwebhook / updatewebhook + cron twins → ErrorPayload)

A non-evaluatable transform must be rejected as a clean `ErrorPayload` (validation error → user-visible message), never a runtime delivery failure. `jsonneteval.Validate` only exists after B1, so the validation wiring lives here.

**Files**
- Modify: `internal/case/admin/createwebhook/resolve.go`
- Modify: `internal/case/admin/updatewebhook/resolve.go`
- Modify: `internal/case/admin/createcronwebhook/resolve.go`
- Modify: `internal/case/admin/updatecronwebhook/resolve.go`
- Test: `internal/case/admin/createwebhook/resolve_test.go`

> Confirm exact cron CRUD package names with `ls internal/case/admin | grep -i cron` before editing the cron twins; the change-webhook pair is confirmed at `internal/case/admin/createwebhook` / `updatewebhook`.

**Steps**

1. [ ] **Write the failing test.** Add to `internal/case/admin/createwebhook/resolve_test.go`:

```go
func TestResolve_InvalidTransformJsonnet_ReturnsErrorPayload(t *testing.T) {
	env := newTestEnv(t) // reuse the package's existing test env factory
	bad := "}{ not jsonnet"
	in := model.ChangeWebhookCreateInput{
		URL:              "https://example.com/hook",
		IncludePatterns:  []string{"**"},
		TransformJsonnet: &bad,
	}

	payload, err := createwebhook.Resolve(context.Background(), env, in)
	require.NoError(t, err) // validation error -> (ErrorPayload, nil)

	ep, ok := payload.(*model.ErrorPayload)
	require.True(t, ok, "expected *model.ErrorPayload, got %T", payload)
	require.NotEmpty(t, ep.ByFields)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
}
```

If the package lacks a shared `newTestEnv`, construct the `EnvMock`/manual mock inline with `InsertWebhook` returning a zero `db.ChangeWebhook`.

2. [ ] **Run it to verify it fails.** `go test ./internal/case/admin/createwebhook/... -run 'InvalidTransform' -count=1` → expected: fail (no validation yet — returns success payload or panics on field).

3. [ ] **Implement.** In `internal/case/admin/createwebhook/resolve.go` add a transform check and call it from `Resolve` right after `validateInput`:

```go
// validateTransformJsonnet rejects a transform that cannot even evaluate.
func validateTransformJsonnet(src *string) *model.ErrorPayload {
	if src == nil || *src == "" {
		return nil
	}
	if err := jsonneteval.Validate(*src, map[string]string{
		"change":         "[]",
		"attached_notes": "[]",
		"meta":           "{}",
	}); err != nil {
		return &model.ErrorPayload{ByFields: []model.FieldMessage{
			{Name: "transformJsonnet", Value: "invalid jsonnet: " + err.Error()},
		}}
	}
	return nil
}
```

In `Resolve`, after the existing `errPayload := validateInput(&input)` block:

```go
	if ep := validateTransformJsonnet(input.TransformJsonnet); ep != nil {
		return ep, nil
	}
```

Add `"trip2g/internal/jsonneteval"` to the imports. Apply the identical helper + call to `updatewebhook`, `createcronwebhook`, `updatecronwebhook` (each reads its own input's `TransformJsonnet` pointer; field name in the error stays `transformJsonnet`).

4. [ ] **Run to verify pass.** `go test ./internal/case/admin/createwebhook/... ./internal/case/admin/updatewebhook/... ./internal/case/admin/createcronwebhook/... ./internal/case/admin/updatecronwebhook/... -count=1` → expected: all `ok`.

5. [ ] **Commit.** `feat(webhook): validate transform_jsonnet at CRUD`

---

### Task B6 — `attach_notes` presence gate in `handlenotewebhooks.Resolve`

After a non-empty `matched` set and before `InsertWebhookDelivery`, require the attach gate: a plain glob must match ≥1 current note; a `!glob` must match 0. This prevents firing a role that has no context to work on.

**Files**
- Modify: `internal/case/handlenotewebhooks/resolve.go`
- Test: `internal/case/handlenotewebhooks/resolve_test.go`

**Steps**

1. [ ] **Write the failing test.** Add a focused unit test for the gate predicate (it is pure) to `internal/case/handlenotewebhooks/resolve_test.go`:

```go
func TestAttachGateSatisfied(t *testing.T) {
	nvs := model.NewNoteViews()
	nvs.RegisterNote(&model.NoteView{Path: "boards/sprint.md", PathID: 1})
	nvs.RegisterNote(&model.NoteView{Path: "roles/triage.md", PathID: 2})

	tests := []struct {
		name   string
		attach []string
		want   bool
	}{
		{"empty attach = always satisfied", nil, true},
		{"plain glob with a match", []string{"boards/**"}, true},
		{"plain glob with no match", []string{"index/**"}, false},
		{"require-absent satisfied (no match)", []string{"!inbox/**"}, true},
		{"require-absent violated (has match)", []string{"!boards/**"}, false},
		{"mixed: present AND absent both hold", []string{"boards/**", "!inbox/**"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, handlenotewebhooks.AttachGateSatisfiedForTest(tt.attach, nvs))
		})
	}
}
```

2. [ ] **Run it to verify it fails.** `go test ./internal/case/handlenotewebhooks/... -run 'AttachGate' -count=1` → expected: `undefined: handlenotewebhooks.AttachGateSatisfiedForTest`.

3. [ ] **Implement.** In `internal/case/handlenotewebhooks/resolve.go` add the predicate + a test seam:

```go
// attachGateSatisfied reports whether the webhook's attach_notes preconditions
// hold against the current note set. A plain glob requires >=1 matching note;
// a "!glob" requires 0 matching notes. Empty attach is always satisfied.
func attachGateSatisfied(attach []string, nvs *model.NoteViews) bool {
	for _, pat := range attach {
		if strings.HasPrefix(pat, "!") {
			if anyNoteMatches(strings.TrimPrefix(pat, "!"), nvs) {
				return false // required-absent glob matched something
			}
			continue
		}
		if !anyNoteMatches(pat, nvs) {
			return false // required-present glob matched nothing
		}
	}
	return true
}

func anyNoteMatches(glob string, nvs *model.NoteViews) bool {
	if nvs == nil {
		return false
	}
	for path := range nvs.PathMap {
		if webhookutil.MatchesAny(path, []string{glob}) {
			return true
		}
	}
	return false
}

// AttachGateSatisfiedForTest exposes attachGateSatisfied to the test package.
func AttachGateSatisfiedForTest(attach []string, nvs *model.NoteViews) bool {
	return attachGateSatisfied(attach, nvs)
}
```

Add `"strings"` to the import block. Then wire the gate into the loop in `Resolve` — after the `if len(matched) == 0 { continue }` block (currently :159-161) and before `InsertWebhookDelivery` (:169):

```go
		// attach_notes presence gate: skip if the role's required context is
		// absent (plain glob) or a forbidden note is present ("!glob").
		attach, attachErr := webhookutil.ParseJSONStringArray(wh.AttachNotes)
		if attachErr != nil {
			env.Logger().Error("failed to parse attach_notes", "webhook_id", wh.ID, "error", attachErr)
			continue
		}
		if !attachGateSatisfied(attach, nvs) {
			continue
		}
```

4. [ ] **Run to verify pass.** `go test ./internal/case/handlenotewebhooks/... -count=1` → expected: `ok` (existing `TestResolve` cases use `AttachNotes` zero value `""`? — the M1 default is `'[]'`; in the manual mock the `db.ChangeWebhook` literals omit `AttachNotes`, so it is `""`. Guard: `ParseJSONStringArray("")` errors → the loop would `continue` and break existing tests. To keep the regression green, set `AttachNotes: "[]"` on the webhook literals in the existing `TestResolve` table entries, OR make `attachGateSatisfied(nil, ...)` the path by treating empty string as empty list). Implement the empty-string tolerance: before parsing, `if wh.AttachNotes == "" { attach = nil } else { parse }`. Re-run to confirm `ok`.

5. [ ] **Commit.** `feat(handlenotewebhooks): gate delivery on attach_notes presence`

---

### Task B7 — Materialize `attach_notes` into the change payload (`attached_notes`)

The change deliver job materializes each non-`!` attach glob's current notes into the payload as `attached_notes: [{path,title,content,updated_at,tags,meta}]`, sourced from `model.NoteView`. `meta` is `tags` + a small allowlist, **not** full `RawMeta`.

**Files**
- Modify: `internal/case/backjob/deliverchangewebhook/resolve.go`
- Test: `internal/case/backjob/deliverchangewebhook/resolve_test.go`

**Steps**

1. [ ] **Write the failing test.** Append to `internal/case/backjob/deliverchangewebhook/resolve_test.go`:

```go
func TestResolve_AttachNotes_Materialized(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	nvs := model.NewNoteViews()
	nvs.RegisterNote(&model.NoteView{
		Path:    "boards/sprint.md",
		Title:   "Sprint",
		Content: []byte("# Sprint\n- card"),
		Tags:    []string{"kanban"},
	})

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
			ID: id, Url: srv.URL, TimeoutSeconds: 10,
			WritePatterns: "[]", ReadPatterns: "[]",
			AttachNotes: `["boards/**","!inbox/**"]`,
		}, nil
	}
	env.LatestNoteViewsFunc = func() *model.NoteViews { return nvs }

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 1, Attempt: 1})
	require.NoError(t, err)

	var payload struct {
		AttachedNotes []map[string]any `json:"attached_notes"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Len(t, payload.AttachedNotes, 1)
	require.Equal(t, "boards/sprint.md", payload.AttachedNotes[0]["path"])
	require.Equal(t, "Sprint", payload.AttachedNotes[0]["title"])
	require.Contains(t, payload.AttachedNotes[0], "content")
	require.Contains(t, payload.AttachedNotes[0], "meta")
}
```

2. [ ] **Run it to verify it fails.** `go test ./internal/case/backjob/deliverchangewebhook/... -run 'AttachNotes_Materialized' -count=1` → expected: `EnvMock has no field LatestNoteViewsFunc` / `attached_notes` absent.

3. [ ] **Implement.** Add `LatestNoteViews() *model.NoteViews` to the `Env` interface in `resolve.go`:

```go
	LatestNoteViews() *model.NoteViews
```

Add an `AttachedNotes` field to `changeWebhookPayload`:

```go
	AttachedNotes []attachedNote `json:"attached_notes,omitempty"`
```

Add the materialized-note type + builder at the bottom of the file:

```go
// attachedNote is a context note pushed into the delivery payload. meta is an
// allowlist (never the full RawMeta) so the role only sees what it asked for.
type attachedNote struct {
	Path      string            `json:"path"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	UpdatedAt string            `json:"updated_at,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Meta      map[string]string `json:"meta"`
}

// materializeAttachNotes returns the current notes matching the non-"!" attach
// globs, sorted by path for determinism.
func materializeAttachNotes(attach []string, nvs *model.NoteViews) []attachedNote {
	if nvs == nil {
		return nil
	}
	var globs []string
	for _, p := range attach {
		if !strings.HasPrefix(p, "!") {
			globs = append(globs, p)
		}
	}
	if len(globs) == 0 {
		return nil
	}

	var out []attachedNote
	for path, nv := range nvs.PathMap {
		if !webhookutil.MatchesAny(path, globs) {
			continue
		}
		an := attachedNote{
			Path:    nv.Path,
			Title:   nv.Title,
			Content: string(nv.Content),
			Tags:    nv.Tags,
			Meta:    map[string]string{},
		}
		if !nv.UpdatedAt.IsZero() {
			an.UpdatedAt = nv.UpdatedAt.Format(time.RFC3339)
		}
		if len(nv.Tags) > 0 {
			an.Meta["tags"] = strings.Join(nv.Tags, ",")
		}
		if nv.Layout != "" {
			an.Meta["layout"] = nv.Layout
		}
		out = append(out, an)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
```

Add `"sort"` and ensure `"strings"`, `"time"` are imported. In `Resolve`, after building `payload` and before `json.Marshal` (line 101), populate it:

```go
	if wh.AttachNotes != "" && wh.AttachNotes != "[]" {
		if attach, aerr := webhookutil.ParseJSONStringArray(wh.AttachNotes); aerr == nil {
			payload.AttachedNotes = materializeAttachNotes(attach, env.LatestNoteViews())
		} else {
			log.Error("failed to parse attach_notes", "webhook_id", wh.ID, "error", aerr)
		}
	}
```

4. [ ] **Regenerate the mock + run.** `go generate ./internal/case/backjob/deliverchangewebhook/...` then `go test ./internal/case/backjob/deliverchangewebhook/... -count=1` → expected: `ok` (the new `LatestNoteViews` method lands in `EnvMock`; the test sets `LatestNoteViewsFunc`; existing tests that don't set it must give a default — set `LatestNoteViewsFunc: func() *model.NoteViews { return nil }` in `baseEnv`).

5. [ ] **Commit.** `feat(deliverchangewebhook): materialize attach_notes into payload`

---

### Task B8 — Concurrency + stale + janitor SQL queries (+ cron twins) → sqlc

Add the conditional-insert / mark-running / stale-finalize queries on the single serialized write connection. `WHERE NOT EXISTS` is atomic there.

**Files**
- Modify: `queries.write.sql`
- Generated (do not hand-edit): `internal/db/queries.write.sql.go`, `internal/db/models.go`

**Steps**

1. [ ] **Write the queries.** In `queries.write.sql`, after `-- name: InsertWebhookDelivery :one` (line 974-977) add:

```sql
-- name: InsertWebhookDeliveryIfClear :one
-- skip mode: insert only if no in-flight (pending/running) delivery exists for
-- this webhook within the stale window (heartbeat/started/created coalesced).
insert into change_webhook_deliveries (webhook_id, attempt, status)
select sqlc.arg(webhook_id), 1, 'pending'
where not exists (
  select 1 from change_webhook_deliveries
  where webhook_id = sqlc.arg(webhook_id)
    and status in ('pending','running')
    and coalesce(heartbeat_at, started_at, created_at) >= datetime('now', sqlc.arg(stale_window)))
returning *;

-- name: InsertWebhookDeliveryIfNoPending :one
-- queue_one mode: insert only if there is no pending delivery already queued.
insert into change_webhook_deliveries (webhook_id, attempt, status)
select sqlc.arg(webhook_id), 1, 'pending'
where not exists (
  select 1 from change_webhook_deliveries
  where webhook_id = sqlc.arg(webhook_id) and status = 'pending')
returning *;

-- name: MarkWebhookDeliveryRunning :exec
update change_webhook_deliveries
set status = 'running', started_at = datetime('now')
where id = ? and status = 'pending';

-- name: ExpireStaleWebhookDeliveries :exec
-- janitor: finalize orphaned 'running' rows past the stale window to 'failed'.
update change_webhook_deliveries
set status = 'failed', completed_at = datetime('now')
where status = 'running'
  and coalesce(heartbeat_at, started_at, created_at) < datetime('now', sqlc.arg(stale_window));
```

After `-- name: InsertCronWebhookDelivery :one` (line 1027-1030) add the cron twins (same shape, table `cron_webhook_deliveries`, column `cron_webhook_id`):

```sql
-- name: InsertCronWebhookDeliveryIfClear :one
insert into cron_webhook_deliveries (cron_webhook_id, attempt, status)
select sqlc.arg(cron_webhook_id), 1, 'pending'
where not exists (
  select 1 from cron_webhook_deliveries
  where cron_webhook_id = sqlc.arg(cron_webhook_id)
    and status in ('pending','running')
    and coalesce(heartbeat_at, started_at, created_at) >= datetime('now', sqlc.arg(stale_window)))
returning *;

-- name: InsertCronWebhookDeliveryIfNoPending :one
insert into cron_webhook_deliveries (cron_webhook_id, attempt, status)
select sqlc.arg(cron_webhook_id), 1, 'pending'
where not exists (
  select 1 from cron_webhook_deliveries
  where cron_webhook_id = sqlc.arg(cron_webhook_id) and status = 'pending')
returning *;

-- name: MarkCronWebhookDeliveryRunning :exec
update cron_webhook_deliveries
set status = 'running', started_at = datetime('now')
where id = ? and status = 'pending';

-- name: ExpireStaleCronWebhookDeliveries :exec
update cron_webhook_deliveries
set status = 'failed', completed_at = datetime('now')
where status = 'running'
  and coalesce(heartbeat_at, started_at, created_at) < datetime('now', sqlc.arg(stale_window));
```

2. [ ] **Run codegen to verify it generates.** `make sqlc` → expected: success, no diff errors; `git status` shows `internal/db/queries.write.sql.go` regenerated with `InsertWebhookDeliveryIfClear`, `InsertWebhookDeliveryIfNoPending`, `MarkWebhookDeliveryRunning`, `ExpireStaleWebhookDeliveries` (+ cron twins). The `:one` conditional inserts return `(db.ChangeWebhookDelivery, error)`; a no-row insert yields `sql.ErrNoRows`.

3. [ ] **Verify compile.** `go build ./internal/db/...` → expected: success.

4. [ ] **Commit.** `feat(db): conditional webhook delivery insert, mark-running, stale-expire queries`

---

### Task B9 — Concurrency dispatch in `handlenotewebhooks.Resolve` + cooldown config

Branch the insert on `wh.ConcurrencyMode`; `skip`/`queue_one` may produce 0 rows (`sql.ErrNoRows`) → silently skip. Stale window = `appconfig.AgentDeliveryCooldownSeconds` (negative-seconds string for `datetime('now', ...)`).

**Files**
- Modify: `internal/appconfig/config.go`
- Modify: `internal/case/handlenotewebhooks/resolve.go`
- Modify: `internal/case/handlenotewebhooks/mocks_test.go`
- Test: `internal/case/handlenotewebhooks/resolve_test.go`

**Steps**

1. [ ] **Write the failing test.** Add to `internal/case/handlenotewebhooks/resolve_test.go` (the manual `mockEnv` records calls):

```go
func TestResolve_ConcurrencySkip_OnlyInsertsWhenClear(t *testing.T) {
	env := newMockEnv()
	env.addNote("boards/sprint.md", 1, 10, "Sprint", "x")
	env.setWebhooks([]db.ChangeWebhook{{
		ID: 1, Url: "https://e/x",
		IncludePatterns: `["boards/sprint.md"]`, ExcludePatterns: `[]`,
		AttachNotes: "[]", OnUpdate: true, MaxDepth: 5,
		ConcurrencyMode: "skip",
	}})
	env.ifClearOK = false // simulate an in-flight delivery -> 0 rows

	err := handlenotewebhooks.Resolve(context.Background(), env,
		[]handlenotewebhooks.NoteChange{{PathID: 1, Event: "update"}}, 0)
	require.NoError(t, err)
	require.Empty(t, env.getEnqueued(), "skip mode must not enqueue when not clear")

	env.ifClearOK = true
	err = handlenotewebhooks.Resolve(context.Background(), env,
		[]handlenotewebhooks.NoteChange{{PathID: 1, Event: "update"}}, 0)
	require.NoError(t, err)
	require.Len(t, env.getEnqueued(), 1)
}
```

2. [ ] **Run it to verify it fails.** `go test ./internal/case/handlenotewebhooks/... -run 'ConcurrencySkip' -count=1` → expected: build failure (`env.ifClearOK` undefined; `InsertWebhookDeliveryIfClear` not in Env).

3. [ ] **Implement.** Add the cooldown field to `internal/appconfig/config.go` `Config` struct (near `CronExecuteWebhooksSchedule`):

```go
	AgentDeliveryCooldownSeconds int
```

and register a flag in the flag setup section with a sane default:

```go
	flag.IntVar(&c.AgentDeliveryCooldownSeconds, "agent-delivery-cooldown-seconds", 60,
		"No-overlap stale window for skip/queue_one webhook delivery (seconds)")
```

Extend the `handlenotewebhooks.Env` interface in `resolve.go`:

```go
	InsertWebhookDeliveryIfClear(ctx context.Context, arg db.InsertWebhookDeliveryIfClearParams) (db.ChangeWebhookDelivery, error)
	InsertWebhookDeliveryIfNoPending(ctx context.Context, arg db.InsertWebhookDeliveryIfNoPendingParams) (db.ChangeWebhookDelivery, error)
	AgentDeliveryCooldownSeconds() int
```

Replace the unconditional `InsertWebhookDelivery` block (:169-176) with a dispatch:

```go
		staleWindow := fmt.Sprintf("-%d seconds", env.AgentDeliveryCooldownSeconds())

		var delivery db.ChangeWebhookDelivery
		var insertErr error
		switch wh.ConcurrencyMode {
		case "skip":
			delivery, insertErr = env.InsertWebhookDeliveryIfClear(ctx, db.InsertWebhookDeliveryIfClearParams{
				WebhookID: wh.ID, StaleWindow: staleWindow,
			})
		case "queue_one":
			delivery, insertErr = env.InsertWebhookDeliveryIfNoPending(ctx, db.InsertWebhookDeliveryIfNoPendingParams{
				WebhookID: wh.ID,
			})
		default: // allow_overlap
			delivery, insertErr = env.InsertWebhookDelivery(ctx, db.InsertWebhookDeliveryParams{
				WebhookID: wh.ID, Attempt: 1,
			})
		}
		if db.IsNoFound(insertErr) {
			// skip/queue_one: nothing inserted because an in-flight/pending exists.
			continue
		}
		if insertErr != nil {
			env.Logger().Error("failed to insert webhook delivery", "webhook_id", wh.ID, "error", insertErr)
			continue
		}
```

(Confirm the generated param struct field names with `grep 'InsertWebhookDeliveryIfClearParams' internal/db/queries.write.sql.go`; sqlc names them `WebhookID` + `StaleWindow` from the `sqlc.arg` labels. Confirm `db.IsNoFound` wraps `sql.ErrNoRows` — it is used across the codebase, e.g. `checkapikey`.)

Add the new methods to the manual `internal/case/handlenotewebhooks/mocks_test.go`:

```go
	ifClearOK     bool
	ifNoPendingOK bool
```

```go
func (m *mockEnv) InsertWebhookDeliveryIfClear(ctx context.Context, arg db.InsertWebhookDeliveryIfClearParams) (db.ChangeWebhookDelivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.ifClearOK {
		return db.ChangeWebhookDelivery{}, sql.ErrNoRows
	}
	id := m.nextDeliveryID
	m.nextDeliveryID++
	return db.ChangeWebhookDelivery{ID: id, WebhookID: arg.WebhookID, Status: "pending"}, nil
}

func (m *mockEnv) InsertWebhookDeliveryIfNoPending(ctx context.Context, arg db.InsertWebhookDeliveryIfNoPendingParams) (db.ChangeWebhookDelivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.ifNoPendingOK {
		return db.ChangeWebhookDelivery{}, sql.ErrNoRows
	}
	id := m.nextDeliveryID
	m.nextDeliveryID++
	return db.ChangeWebhookDelivery{ID: id, WebhookID: arg.WebhookID, Status: "pending"}, nil
}

func (m *mockEnv) AgentDeliveryCooldownSeconds() int { return 60 }
```

Add `"database/sql"` to the mock's imports. Implement the two app methods are auto-promoted from `*db.WriteQueries` (no app code needed); add `func (a *app) AgentDeliveryCooldownSeconds() int { return a.config.AgentDeliveryCooldownSeconds }` in `cmd/server/webhooks.go` (next to `EnqueueDeliverChangeWebhook`).

4. [ ] **Run to verify pass.** `go test ./internal/case/handlenotewebhooks/... -count=1 && go build ./cmd/server/...` → expected: `ok` + build success. (Existing `TestResolve` cases use `ConcurrencyMode: ""` → `default` branch → `InsertWebhookDelivery`, unchanged.)

5. [ ] **Commit.** `feat(handlenotewebhooks): concurrency_mode dispatch with no-overlap guard`

---

### Task B10 — `MarkWebhookDeliveryRunning` at job pickup (change + cron)

At the top of each deliver job, flip `pending → running` and stamp `started_at`, so the stale-lock window starts at execution and the janitor can self-heal a crashed fleet.

**Files**
- Modify: `internal/case/backjob/deliverchangewebhook/resolve.go`
- Modify: `internal/case/backjob/delivercronwebhook/resolve.go`
- Modify: `internal/case/backjob/deliverchangewebhook/resolve_test.go`
- Modify: `internal/case/backjob/delivercronwebhook/resolve_test.go`

**Steps**

1. [ ] **Write the failing test.** Add to `internal/case/backjob/deliverchangewebhook/resolve_test.go`:

```go
func TestResolve_MarksRunningAtPickup(t *testing.T) {
	var markedID int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.MarkWebhookDeliveryRunningFunc = func(_ context.Context, id int64) error {
		markedID = id
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 555, Attempt: 1})
	require.NoError(t, err)
	require.EqualValues(t, 555, markedID)
}
```

2. [ ] **Run it to verify it fails.** `go test ./internal/case/backjob/deliverchangewebhook/... -run 'MarksRunning' -count=1` → expected: `EnvMock has no field MarkWebhookDeliveryRunningFunc`.

3. [ ] **Implement.** Add to the change-deliver `Env`:

```go
	MarkWebhookDeliveryRunning(ctx context.Context, id int64) error
```

At the start of `Resolve`, right after loading `wh` (after line 53), only on the first attempt (retries are already running):

```go
	if params.Attempt <= 1 {
		if mErr := env.MarkWebhookDeliveryRunning(ctx, params.DeliveryID); mErr != nil {
			log.Error("failed to mark delivery running", "delivery_id", params.DeliveryID, "error", mErr)
		}
	}
```

Mirror in `delivercronwebhook` with `MarkCronWebhookDeliveryRunning(ctx, id int64) error` and the same first-attempt guard. Regenerate both mocks: `go generate ./internal/case/backjob/deliverchangewebhook/... ./internal/case/backjob/delivercronwebhook/...`, then set a default `MarkWebhookDeliveryRunningFunc`/`MarkCronWebhookDeliveryRunningFunc` returning `nil` in each `baseEnv` so prior tests keep passing.

4. [ ] **Run to verify pass.** `go test ./internal/case/backjob/... -count=1 && go build ./cmd/server/...` → expected: `ok` + build success (app methods promoted from `*db.WriteQueries`).

5. [ ] **Commit.** `feat(deliverwebhook): mark delivery running at job pickup`

---

### Task B11 — Janitor cron `expirestalewebhookdeliveries`

A cron job finalizes orphaned `running` deliveries (change + cron) to `failed` using the same stale window. Registered alongside the other cleanup jobs.

**Files**
- Create: `internal/case/cronjob/expirestalewebhookdeliveries/resolve.go`
- Create: `internal/case/cronjob/expirestalewebhookdeliveries/job.go`
- Test: `internal/case/cronjob/expirestalewebhookdeliveries/resolve_test.go`
- Modify: `cmd/server/cronjobs.go`

**Steps**

1. [ ] **Write the failing test.** Create `internal/case/cronjob/expirestalewebhookdeliveries/resolve_test.go`:

```go
package expirestalewebhookdeliveries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/cronjob/expirestalewebhookdeliveries"
	"trip2g/internal/logger"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg expirestalewebhookdeliveries_test . Env

func newEnv() *EnvMock {
	return &EnvMock{
		ExpireStaleWebhookDeliveriesFunc:     func(_ context.Context, _ string) error { return nil },
		ExpireStaleCronWebhookDeliveriesFunc: func(_ context.Context, _ string) error { return nil },
		AgentDeliveryCooldownSecondsFunc:     func() int { return 60 },
		LoggerFunc:                           func() logger.Logger { return &logger.DummyLogger{} },
	}
}

func TestResolve_ExpiresBothTables(t *testing.T) {
	var changeWin, cronWin string
	env := newEnv()
	env.ExpireStaleWebhookDeliveriesFunc = func(_ context.Context, w string) error { changeWin = w; return nil }
	env.ExpireStaleCronWebhookDeliveriesFunc = func(_ context.Context, w string) error { cronWin = w; return nil }

	_, err := expirestalewebhookdeliveries.Resolve(context.Background(), env)
	require.NoError(t, err)
	require.Equal(t, "-60 seconds", changeWin)
	require.Equal(t, "-60 seconds", cronWin)
}

func TestResolve_PropagatesError(t *testing.T) {
	env := newEnv()
	env.ExpireStaleWebhookDeliveriesFunc = func(_ context.Context, _ string) error { return errors.New("boom") }
	_, err := expirestalewebhookdeliveries.Resolve(context.Background(), env)
	require.Error(t, err)
}
```

2. [ ] **Run it to verify it fails.** `go test ./internal/case/cronjob/expirestalewebhookdeliveries/... -count=1` → expected: package/build failure (does not exist).

3. [ ] **Implement.** Create `internal/case/cronjob/expirestalewebhookdeliveries/resolve.go`:

```go
package expirestalewebhookdeliveries

import (
	"context"
	"fmt"

	"trip2g/internal/logger"
)

type Env interface {
	ExpireStaleWebhookDeliveries(ctx context.Context, staleWindow string) error
	ExpireStaleCronWebhookDeliveries(ctx context.Context, staleWindow string) error
	AgentDeliveryCooldownSeconds() int
	Logger() logger.Logger
}

// Result reports whether the sweep ran.
type Result struct {
	Expired bool
}

// Resolve finalizes orphaned 'running' webhook deliveries (change + cron) whose
// liveness window has lapsed, marking them 'failed'.
func Resolve(ctx context.Context, env Env) (*Result, error) {
	staleWindow := fmt.Sprintf("-%d seconds", env.AgentDeliveryCooldownSeconds())

	if err := env.ExpireStaleWebhookDeliveries(ctx, staleWindow); err != nil {
		return nil, fmt.Errorf("failed to expire stale change webhook deliveries: %w", err)
	}
	if err := env.ExpireStaleCronWebhookDeliveries(ctx, staleWindow); err != nil {
		return nil, fmt.Errorf("failed to expire stale cron webhook deliveries: %w", err)
	}

	return &Result{Expired: true}, nil
}
```

Create `internal/case/cronjob/expirestalewebhookdeliveries/job.go` (mirror `cleanupwebhookdeliveries/job.go`):

```go
package expirestalewebhookdeliveries

import "context"

// Job implements the cronjobs.Job interface.
type Job struct{}

func (j *Job) Name() string { return "expire_stale_webhook_deliveries" }

// Schedule runs every minute to bound orphan-lock lifetime.
func (j *Job) Schedule() string { return "0 * * * * *" }

func (j *Job) ExecuteAfterStart() bool { return false }

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env)) //nolint:errcheck // checked in cmd/server/cronjobs.go.
}
```

Generate the mock: `go generate ./internal/case/cronjob/expirestalewebhookdeliveries/...`.

In `cmd/server/cronjobs.go` add the import `"trip2g/internal/case/cronjob/expirestalewebhookdeliveries"`, add `_ expirestalewebhookdeliveries.Env = app` to the compile-time check block, and append `&expirestalewebhookdeliveries.Job{}` to the `jobs` slice.

4. [ ] **Run to verify pass.** `go test ./internal/case/cronjob/expirestalewebhookdeliveries/... -count=1 && go build ./cmd/server/...` → expected: `ok` + build success (`ExpireStaleWebhookDeliveries`/`ExpireStaleCronWebhookDeliveries` promoted from `*db.WriteQueries`; `AgentDeliveryCooldownSeconds` added in B9).

5. [ ] **Commit.** `feat(cron): janitor to expire stale webhook deliveries`

---

### Task B12 — Carry delivery identity in the scoped token (shortapitoken + deliver mint)

The scoped write-back token now embeds `(DeliveryKind, DeliveryID)` so the note version it writes can be attributed back to the delivery.

**Files**
- Modify: `internal/shortapitoken/token.go`
- Test: `internal/shortapitoken/token_test.go`
- Modify: `internal/case/backjob/deliverchangewebhook/resolve.go`
- Modify: `internal/case/backjob/delivercronwebhook/resolve.go`

**Steps**

1. [ ] **Write the failing test.** Create/append `internal/shortapitoken/token_test.go`:

```go
package shortapitoken_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/shortapitoken"
)

func TestSignParse_RoundTripsDeliveryIdentity(t *testing.T) {
	in := shortapitoken.Data{
		Depth:         1,
		ReadPatterns:  []string{"boards/**"},
		WritePatterns: []string{"boards/**"},
		DeliveryKind:  "change",
		DeliveryID:    4242,
	}
	tok, err := shortapitoken.Sign(in, "secret", time.Minute)
	require.NoError(t, err)

	out, err := shortapitoken.Parse(tok, "secret")
	require.NoError(t, err)
	require.Equal(t, "change", out.DeliveryKind)
	require.EqualValues(t, 4242, out.DeliveryID)
}
```

2. [ ] **Run it to verify it fails.** `go test ./internal/shortapitoken/... -run 'DeliveryIdentity' -count=1` → expected: `unknown field DeliveryKind in struct literal`.

3. [ ] **Implement.** Add to `shortapitoken.Data`:

```go
	DeliveryKind  string   `json:"dk,omitempty"` // "change" | "cron"
	DeliveryID    int64    `json:"di,omitempty"`
```

(No change to `Sign`/`Parse` — `Data` is embedded in `claims`.) Then set them where the token is minted. In `deliverchangewebhook/resolve.go`, in the `shortapitoken.Sign(shortapitoken.Data{...})` call (line 88-92):

```go
			token, signErr := shortapitoken.Sign(shortapitoken.Data{
				Depth:         params.Depth + 1,
				ReadPatterns:  readPatterns,
				WritePatterns: writePatterns,
				DeliveryKind:  "change",
				DeliveryID:    params.DeliveryID,
			}, env.ShortAPITokenSecret(), ttl)
```

In `delivercronwebhook/resolve.go` (line 104-108):

```go
			token, signErr := shortapitoken.Sign(shortapitoken.Data{
				Depth:         1,
				ReadPatterns:  readPatterns,
				WritePatterns: writePatterns,
				DeliveryKind:  "cron",
				DeliveryID:    params.DeliveryID,
			}, env.ShortAPITokenSecret(), ttl)
```

4. [ ] **Run to verify pass.** `go test ./internal/shortapitoken/... ./internal/case/backjob/... -count=1` → expected: all `ok`.

5. [ ] **Commit.** `feat(shortapitoken): carry delivery kind and id in scoped token`

---

### Task B13 — Stamp delivery identity into `appreq` from `checkapikey`

`resolveShortAPIToken` now copies the token's `DeliveryKind`/`DeliveryID` into the request, with accessors and pool-safe reset/snapshot.

**Files**
- Modify: `internal/appreq/request.go`
- Modify: `internal/case/checkapikey/resolve.go`
- Test: `internal/case/checkapikey/resolve_test.go`

**Steps**

1. [ ] **Write the failing test.** Add to `internal/case/checkapikey/resolve_test.go` a case that authenticates via a Bearer shortapitoken carrying a delivery identity and asserts the request fields are stamped. If the package has no Bearer-token test yet, model it on the existing test's request setup (`appreq.Request` with a `fasthttp.RequestCtx` whose `Authorization: Bearer <token>` header is set, token signed via `shortapitoken.Sign` with the same `ShortAPITokenSecret`). Assert post-`Resolve`:

```go
	require.Equal(t, "change", req.WebhookDeliveryKind)
	require.EqualValues(t, 99, req.WebhookDeliveryID)
```

2. [ ] **Run it to verify it fails.** `go test ./internal/case/checkapikey/... -run 'Delivery' -count=1` → expected: `req.WebhookDeliveryKind undefined`.

3. [ ] **Implement.** Add fields to `appreq.Request` (after `WebhookWritePatterns`, line 46):

```go
	// Webhook delivery identity (from the scoped shortapitoken). Used to
	// attribute note versions written by a fleet back to the delivery.
	WebhookDeliveryKind string
	WebhookDeliveryID   int64
```

Reset them in `Reset()` (after `c.WebhookWritePatterns = nil`):

```go
	c.WebhookDeliveryKind = ""
	c.WebhookDeliveryID = 0
```

Carry them in `Snapshot()`'s returned `&Request{...}`:

```go
		WebhookDeliveryKind:   c.WebhookDeliveryKind,
		WebhookDeliveryID:     c.WebhookDeliveryID,
```

Add accessors at the end of the file:

```go
// WebhookDeliveryKind returns the delivery kind ("change"/"cron") for a
// scoped-token request, or "" if none.
func WebhookDeliveryKind(ctx context.Context) string {
	req, err := FromCtx(ctx)
	if err != nil {
		return ""
	}
	return req.WebhookDeliveryKind
}

// WebhookDeliveryID returns the delivery id for a scoped-token request, or 0.
func WebhookDeliveryID(ctx context.Context) int64 {
	req, err := FromCtx(ctx)
	if err != nil {
		return 0
	}
	return req.WebhookDeliveryID
}
```

In `checkapikey/resolve.go` `resolveShortAPIToken`, after line 132 (`req.WebhookWritePatterns = data.WritePatterns`):

```go
	req.WebhookDeliveryKind = data.DeliveryKind
	req.WebhookDeliveryID = data.DeliveryID
```

4. [ ] **Run to verify pass.** `go test ./internal/appreq/... ./internal/case/checkapikey/... -count=1` → expected: all `ok`.

5. [ ] **Commit.** `feat(appreq): stamp webhook delivery identity from scoped token`

---

### Task B14 — Attribution: NoteActor → InsertNoteVersion (delivery kind/id) → sqlc

Complete the actor pipe so a note version written by a fleet records which delivery produced it. M2 columns already exist; this adds them to the write query + the actor + the param plumbing.

**Files**
- Modify: `internal/model/note_actor.go`
- Modify: `cmd/server/notes.go`
- Modify: `internal/case/insertnote/resolve.go`
- Modify: `queries.write.sql`
- Generated: `internal/db/queries.write.sql.go`
- Test: `internal/case/insertnote/resolve_test.go`

**Steps**

1. [ ] **Write the failing test.** Add to `internal/case/insertnote/resolve_test.go` a case where `NoteVersionActor` returns a delivery-attributed actor and assert the captured `db.InsertNoteVersionParams` carries it:

```go
func TestResolve_RecordsDeliveryAttribution(t *testing.T) {
	var got db.InsertNoteVersionParams
	env := newInsertNoteEnv(t) // reuse the package's existing mock/env factory
	env.InsertNoteVersionFunc = func(_ context.Context, arg db.InsertNoteVersionParams) error {
		got = arg
		return nil
	}
	kind := "change"
	var id int64 = 77
	env.NoteVersionActorFunc = func(_ context.Context) model.NoteActor {
		return model.NoteActor{DeliveryKind: &kind, DeliveryID: &id}
	}

	_, err := insertnote.Resolve(context.Background(), env, model.RawNote{Path: "boards/sprint.md", Content: "x"})
	require.NoError(t, err)
	require.NotNil(t, got.CreatedByDeliveryKind)
	require.Equal(t, "change", *got.CreatedByDeliveryKind)
	require.NotNil(t, got.CreatedByDeliveryID)
	require.EqualValues(t, 77, *got.CreatedByDeliveryID)
}
```

Use the package's existing mock style (confirm whether it is moq `EnvMock` or manual; the field setters above assume moq — adjust to the existing pattern in `insertnote/resolve_test.go`).

2. [ ] **Run it to verify it fails.** `go test ./internal/case/insertnote/... -run 'DeliveryAttribution' -count=1` → expected: `model.NoteActor has no field DeliveryKind` / `InsertNoteVersionParams has no field CreatedByDeliveryKind`.

3. [ ] **Implement.**

   Extend `internal/model/note_actor.go`:

```go
type NoteActor struct {
	UserID       *int64
	APIKeyID     *int64
	Client       *string
	DeliveryKind *string
	DeliveryID   *int64
}
```

   Update the write query in `queries.write.sql` (`-- name: InsertNoteVersion :exec`, lines 27-29):

```sql
-- name: InsertNoteVersion :exec
insert into note_versions (path_id, version, content, created_by_user_id, created_by_api_key_id, created_by_client, created_by_delivery_kind, created_by_delivery_id)
values (?, ?, ?, ?, ?, ?, ?, ?);
```

   Run `make sqlc` (expect `InsertNoteVersionParams` to gain `CreatedByDeliveryKind *string` + `CreatedByDeliveryID *int64`).

   Fill the actor in `cmd/server/notes.go` `NoteVersionActor` (after the `req.Client` block, ~line 140):

```go
	if kind := req.WebhookDeliveryKind; kind != "" {
		k := kind
		actor.DeliveryKind = &k
		id := req.WebhookDeliveryID
		actor.DeliveryID = &id
	}
```

   Pass them through in `internal/case/insertnote/resolve.go` `InsertNoteVersionParams` (after `CreatedByClient`, line 99):

```go
		CreatedByDeliveryKind: actor.DeliveryKind,
		CreatedByDeliveryID:   actor.DeliveryID,
```

4. [ ] **Run to verify pass.** `go test ./internal/model/... ./internal/case/insertnote/... -count=1 && go build ./cmd/server/...` → expected: `ok` + build success.

5. [ ] **Commit.** `feat(noteversion): attribute note versions to webhook delivery`

---

### Task B15 — Spend: parse `AgentResponse.{TokensUsed,Steps}` → `UpdateWebhookDeliveryResult`

The fleet reports spend in its response; trip2g records it on the delivery row. M1 added `tokens_used`/`steps`; this surfaces them through the response type, the update query, and the deliver jobs.

**Files**
- Modify: `internal/webhookutil/agentresponse.go`
- Modify: `internal/webhookutil/agentresponse_test.go`
- Modify: `queries.write.sql`
- Generated: `internal/db/queries.write.sql.go`
- Modify: `internal/case/backjob/deliverchangewebhook/resolve.go`
- Modify: `internal/case/backjob/delivercronwebhook/resolve.go`
- Test: `internal/case/backjob/deliverchangewebhook/resolve_test.go`

**Steps**

1. [ ] **Write the failing tests.**

   Add to `internal/webhookutil/agentresponse_test.go`:

```go
func TestParseAgentResponse_ParsesSpend(t *testing.T) {
	body := []byte(`{"status":"ok","tokens_used":1234,"steps":5,"changes":[]}`)
	resp, err := webhookutil.ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 1234, resp.TokensUsed)
	require.Equal(t, 5, resp.Steps)
}
```

   Add to `internal/case/backjob/deliverchangewebhook/resolve_test.go`:

```go
func TestResolve_PersistsSpend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","tokens_used":321,"steps":3,"changes":[]}`))
	}))
	defer srv.Close()

	var got db.UpdateWebhookDeliveryResultParams
	env := baseEnv(t, srv.URL, nil)
	env.UpdateWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 1, Attempt: 1})
	require.NoError(t, err)
	require.Equal(t, "success", got.Status)
	require.NotNil(t, got.TokensUsed)
	require.EqualValues(t, 321, *got.TokensUsed)
	require.NotNil(t, got.Steps)
	require.EqualValues(t, 3, *got.Steps)
}
```

2. [ ] **Run it to verify it fails.** `go test ./internal/webhookutil/... ./internal/case/backjob/deliverchangewebhook/... -run 'Spend' -count=1` → expected: `AgentResponse has no field TokensUsed` / `UpdateWebhookDeliveryResultParams has no field TokensUsed`.

3. [ ] **Implement.**

   Extend `internal/webhookutil/agentresponse.go` `AgentResponse`:

```go
type AgentResponse struct {
	Status     string        `json:"status"`
	Message    string        `json:"message"`
	Changes    []AgentChange `json:"changes"`
	TokensUsed int           `json:"tokens_used"`
	Steps      int           `json:"steps"`
}
```

   Update `queries.write.sql` `UpdateWebhookDeliveryResult` (lines 979-983) and its cron twin (1032-1036) to set the spend columns:

```sql
-- name: UpdateWebhookDeliveryResult :exec
update change_webhook_deliveries
set status = ?, response_status = ?, duration_ms = ?, tokens_used = ?, steps = ?,
    completed_at = datetime('now')
where id = ?;
```

```sql
-- name: UpdateCronWebhookDeliveryResult :exec
update cron_webhook_deliveries
set status = ?, response_status = ?, duration_ms = ?, tokens_used = ?, steps = ?,
    completed_at = datetime('now')
where id = ?;
```

   Run `make sqlc` (params gain `TokensUsed *int64`, `Steps *int64`).

   In `deliverchangewebhook/resolve.go`, parse spend once after a 2xx (non-202) response and thread it into the success `UpdateWebhookDeliveryResult` (line 192-197). Replace the success-marking block:

```go
	// Parse fleet-reported spend (tokens/steps) from the response body.
	var tokensUsed, steps *int64
	if resp, perr := webhookutil.ParseAgentResponse(result.Body); perr == nil && resp != nil {
		if resp.TokensUsed > 0 {
			tokensUsed = ptr.To(int64(resp.TokensUsed))
		}
		if resp.Steps > 0 {
			steps = ptr.To(int64(resp.Steps))
		}
	}

	updateErr := env.UpdateWebhookDeliveryResult(ctx, db.UpdateWebhookDeliveryResultParams{
		Status:         "success",
		ResponseStatus: ptr.To(int64(result.StatusCode)),
		DurationMs:     ptr.To(result.DurationMs),
		TokensUsed:     tokensUsed,
		Steps:          steps,
		ID:             params.DeliveryID,
	})
```

   The two failure-path `UpdateWebhookDeliveryResultParams` literals (`applyErr` no-retries at :179, `handleDeliveryError` at :253) gain `TokensUsed: nil, Steps: nil` (or omit — pointers default nil). Mirror the spend parse + success update in `delivercronwebhook/resolve.go` (`UpdateCronWebhookDeliveryResultParams`, lines 206-211).

4. [ ] **Run to verify pass.** `go test ./internal/webhookutil/... ./internal/case/backjob/... -count=1 && go build ./...` → expected: all `ok` + build success.

5. [ ] **Commit.** `feat(deliverwebhook): record fleet-reported tokens and steps on delivery`

---

**Section B exit checks (run before handing off to Section C):**
- [ ] `go build ./...` → success.
- [ ] `go test ./internal/jsonneteval/... ./internal/frontmatterpatch/... ./internal/case/handlenotewebhooks/... ./internal/case/backjob/... ./internal/case/cronjob/expirestalewebhookdeliveries/... ./internal/case/admin/... ./internal/shortapitoken/... ./internal/appreq/... ./internal/case/checkapikey/... ./internal/case/insertnote/... ./internal/webhookutil/... -count=1` → all `ok`.
- [ ] `git grep -n "TODO\|FIXME" -- internal/jsonneteval internal/case/cronjob/expirestalewebhookdeliveries` → no new placeholders.


---

## Section C — Steps 7-10: patch_note runtime edit, cmd/fleet+internal/fleet+RemoteKB, headline E2E, docs

> Grounding: all paths are in the worktree `/home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime/`. This section depends on Section A (migrations M1+M2, schema inputs `attachNotes`/`transformJsonnet`/`concurrencyMode`, security enforcement) and Section B (attach_notes materialize into payload field `attached_notes`; `webhookutil.AgentResponse` already extended with `TokensUsed int` and `Steps int`; shortapitoken/appreq/NoteActor attribution chain). Where a step needs a Section-A/B symbol, it is named explicitly. `go 1.26.1` (builtin `min` available). Commits follow trip2g convention: short one-liner, conventional, **no body, no Co-Authored-By**.

---

### Task C1 — `patch_note` coordinated runtime edit (Step 7)

This is **one atomic task**: adding `Patch` to the `agentruntime.KB` interface breaks compilation of every implementer (`FileKB`, the test `memKB`, `ScopedKB`'s delegate path) until all gain the method. Land it together, keep `runtime_test.go` + `scope_test.go` green, then re-freeze the package.

**Files:**
- Modify: `internal/webhookutil/agentresponse.go` (AgentChange gains `Find`/`Replace`/`Kind`; `Validate` patch-aware)
- Modify: `internal/agentruntime/kb.go` (KB interface gains `Patch`)
- Modify: `internal/agentruntime/scope.go` (ScopedKB gains `Patch`)
- Modify: `internal/agentruntime/filekb.go` (FileKB gains `Patch`)
- Modify: `internal/agentruntime/runtime.go` (tool `patch_note` in `toolDefs()` + `execTool()` → `AgentChange{Kind:"patch"}`)
- Modify: `internal/agentruntime/runtime_test.go` (test `memKB` gains `Patch`; new patch test)
- Test: `internal/webhookutil/agentresponse_test.go` (patch-change validation)
- Note: `cmd/agent/main.go` is unaffected (uses `FileKB`, which now satisfies the wider interface).

**Steps:**

1. [ ] **Write the failing test (webhookutil).** Append to `internal/webhookutil/agentresponse_test.go`:
```go
func TestParseAgentResponse_PatchChangeNoContentOK(t *testing.T) {
	body := []byte(`{"changes":[{"path":"boards/sprint.md","find":"todo","replace":"doing","kind":"patch"}]}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Changes, 1)
	require.Equal(t, "patch", resp.Changes[0].Kind)
	require.Equal(t, "todo", resp.Changes[0].Find)
	require.Equal(t, "doing", resp.Changes[0].Replace)
}

func TestParseAgentResponse_PatchChangeMissingFind(t *testing.T) {
	body := []byte(`{"changes":[{"path":"boards/sprint.md","kind":"patch"}]}`)
	_, err := ParseAgentResponse(body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid change")
}
```

2. [ ] **Write the failing test (agentruntime).** Append to `internal/agentruntime/runtime_test.go` (the test-`memKB` does not yet have `Patch`, so this file also won't compile — that is the red state):
```go
// (d) Patch role: the model surgically patches an in-scope note via patch_note,
// then finishes. Asserts the find/replace landed and the change is Kind="patch".
func TestRun_PatchNoteStaysInWriteScope(t *testing.T) {
	kb := newMemKB(map[string]string{
		"boards/sprint.md": "- Fix login bug @status:todo\n",
	})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolPatchNote, map[string]any{
				"path": "boards/sprint.md", "find": "@status:todo", "replace": "@status:doing",
			})}, PromptTokens: 10, CompletionTokens: 5},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "moved"})}, PromptTokens: 10, CompletionTokens: 5},
		},
	}
	res, err := Run(context.Background(), Input{
		Instruction:   "Move the card.",
		ReadPatterns:  []string{"boards/**"},
		WritePatterns: []string{"boards/**"},
		Model:         "test-model",
		MaxTokens:     10000,
		MaxSteps:      10,
		LLM:           llm,
		KB:            kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)
	require.Equal(t, "- Fix login bug @status:doing\n", kb.docs["boards/sprint.md"])
	require.Len(t, res.Changes, 1)
	require.Equal(t, "patch", res.Changes[0].Kind)
	require.Equal(t, "@status:todo", res.Changes[0].Find)
	require.Equal(t, "@status:doing", res.Changes[0].Replace)
}

// (e) Patch out of write scope is denied and recorded, content untouched.
func TestRun_PatchNoteDeniedOutOfScope(t *testing.T) {
	kb := newMemKB(map[string]string{"other/x.md": "@status:todo"})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolPatchNote, map[string]any{
				"path": "other/x.md", "find": "@status:todo", "replace": "@status:doing",
			})}, PromptTokens: 10, CompletionTokens: 5},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "blocked"})}, PromptTokens: 10, CompletionTokens: 5},
		},
	}
	res, err := Run(context.Background(), Input{
		Instruction:   "x", ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		Model: "m", MaxTokens: 10000, MaxSteps: 10, LLM: llm, KB: kb,
	})
	require.NoError(t, err)
	require.Len(t, res.Changes, 0)
	require.Equal(t, "@status:todo", kb.docs["other/x.md"])
	require.Equal(t, []string{"patch other/x.md"}, res.Denials)
}
```
Add the testify import to `runtime_test.go` if absent: `"github.com/stretchr/testify/require"`.

3. [ ] **Run it to verify it fails.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/webhookutil/ ./internal/agentruntime/ 2>&1 | tail -20
```
Expected: `internal/agentruntime` fails to compile (`undefined: toolPatchNote`, `m.Patch undefined`); `internal/webhookutil` fails `TestParseAgentResponse_PatchChangeMissingFind`/`...NoContentOK`.

4. [ ] **Implement — webhookutil.** In `internal/webhookutil/agentresponse.go`, extend `AgentChange` and make `Validate` patch-aware (keep `Content` required only for non-patch changes so the existing `TestParseAgentResponse_MissingContent` stays green):
```go
// AgentChange represents a single file change from an agent.
type AgentChange struct {
	Path         string  `json:"path"`
	Content      string  `json:"content"`
	ExpectedHash *string `json:"expected_hash,omitempty"`
	Find         string  `json:"find,omitempty"`
	Replace      string  `json:"replace,omitempty"`
	Kind         string  `json:"kind,omitempty"` // "" | "upsert" | "patch"
}

// Validate validates required fields of an AgentChange. Patch changes require
// Find (not Content); upsert/legacy changes require Content.
func (c AgentChange) Validate() error {
	if c.Kind == "patch" {
		return ozzo.ValidateStruct(&c,
			ozzo.Field(&c.Path, ozzo.Required),
			ozzo.Field(&c.Find, ozzo.Required),
		)
	}
	return ozzo.ValidateStruct(&c,
		ozzo.Field(&c.Path, ozzo.Required),
		ozzo.Field(&c.Content, ozzo.Required),
	)
}
```

5. [ ] **Implement — KB interface.** In `internal/agentruntime/kb.go`, add to the `KB` interface (after `Write`):
```go
	// Patch applies a single find→replace edit to the document at path,
	// preserving unmodeled content around the match.
	Patch(ctx context.Context, path, find, replace string) error
```

6. [ ] **Implement — ScopedKB.** In `internal/agentruntime/scope.go`, add after `Write`:
```go
// Patch applies a find/replace to the document at path, or returns
// ErrWriteDenied if out of write scope.
func (s *ScopedKB) Patch(ctx context.Context, path, find, replace string) error {
	if !s.CanWrite(path) {
		return ErrWriteDenied
	}
	return s.kb.Patch(ctx, path, find, replace)
}
```

7. [ ] **Implement — FileKB.** In `internal/agentruntime/filekb.go`, add after `Write` (errors if `find` not present, mirroring the trip2g patch-not-found semantics):
```go
// Patch reads the file, replaces the first occurrence of find with replace, and
// writes it back. It errors if find is not present so the model learns the patch
// missed rather than silently no-op'ing.
func (f *FileKB) Patch(ctx context.Context, path, find, replace string) error {
	content, err := f.Read(ctx, path)
	if err != nil {
		return err
	}
	if !strings.Contains(content, find) {
		return fmt.Errorf("patch find not found in %s", path)
	}
	return f.Write(ctx, path, strings.Replace(content, find, replace, 1))
}
```
(`strings` and `fmt` are already imported in `filekb.go`.)

8. [ ] **Implement — test memKB.** In `internal/agentruntime/runtime_test.go`, add after `memKB.Write`:
```go
func (m *memKB) Patch(_ context.Context, path, find, replace string) error {
	content, ok := m.docs[path]
	if !ok {
		return errNotFound
	}
	if !strings.Contains(content, find) {
		return &kbError{"patch find not found"}
	}
	m.docs[path] = strings.Replace(content, find, replace, 1)
	return nil
}
```

9. [ ] **Implement — runtime tool.** In `internal/agentruntime/runtime.go`: add the constant, the tool def, and the exec case.
   - Add to the tool-name `const` block: `toolPatchNote = "patch_note"`.
   - In `execTool`, add a case before `default:`:
```go
	case toolPatchNote:
		var args struct {
			Path    string `json:"path"`
			Find    string `json:"find"`
			Replace string `json:"replace"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "error: invalid arguments: " + err.Error()
		}
		err := scoped.Patch(ctx, args.Path, args.Find, args.Replace)
		if errors.Is(err, ErrWriteDenied) {
			res.Denials = append(res.Denials, "patch "+args.Path)
			return "error: " + ErrWriteDenied.Error()
		}
		if err != nil {
			return "error: " + err.Error()
		}
		res.Changes = append(res.Changes, webhookutil.AgentChange{
			Path:    args.Path,
			Find:    args.Find,
			Replace: args.Replace,
			Kind:    "patch",
		})
		return "ok: patched " + args.Path
```
   - In `toolDefs()`, add a tool entry (place before the `toolFinish` entry):
```go
		{
			Name:        toolPatchNote,
			Description: "Apply a surgical find→replace edit to a document (write scope only). Preserves surrounding content.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string"},
					"find":    map[string]any{"type": "string"},
					"replace": map[string]any{"type": "string"},
				},
				"required": []string{"path", "find", "replace"},
			},
		},
```
   - In `systemPromptTemplate`, add a line under the `Tools:` block:
```
- patch_note(path, find, replace): surgically edit a document (write scope only).
```

10. [ ] **Run to verify pass.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/webhookutil/ ./internal/agentruntime/ && go build ./cmd/agent
```
Expected: `ok  trip2g/internal/webhookutil`, `ok  trip2g/internal/agentruntime` (all of (a)–(e), scope tests, hard-cap green), and `cmd/agent` builds.

11. [ ] **Commit.**
```
git add internal/webhookutil/agentresponse.go internal/webhookutil/agentresponse_test.go internal/agentruntime/ && git commit -m "feat(agentruntime): add patch_note tool with find/replace KB.Patch"
```

---

### Task C2 — `internal/fleet` Config + Role frontmatter parsing (Step 8a)

**Files:**
- Create: `internal/fleet/config.go`
- Create: `internal/fleet/role.go`
- Test: `internal/fleet/role_test.go`

**Steps:**

1. [ ] **Write the failing test.** Create `internal/fleet/role_test.go`:
```go
package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func meta(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func TestParseRole_FlatFrontmatter(t *testing.T) {
	r, err := ParseRole("roles/triage.md", "Triage the board.", meta(
		"model", "gpt-4o-mini",
		"tools", "[search, read_note, patch_note]",
		"read_patterns", `["boards/**","roles/**"]`,
		"write_patterns", `["boards/**"]`,
		"max_tokens", "4000",
		"max_steps", "6",
		"mode", "change",
		"trigger_include", `["boards/sprint.md"]`,
		"trigger_on", "[update]",
		"attach_notes", `["boards/**","roles/**"]`,
		"max_depth", "1",
		"concurrency", "skip",
	))
	require.NoError(t, err)
	require.Equal(t, "roles/triage.md", r.NotePath)
	require.Equal(t, "Triage the board.", r.Body)
	require.Equal(t, "gpt-4o-mini", r.Model)
	require.Equal(t, []string{"search", "read_note", "patch_note"}, r.Tools)
	require.Equal(t, []string{"boards/**", "roles/**"}, r.ReadPatterns)
	require.Equal(t, []string{"boards/**"}, r.WritePatterns)
	require.Equal(t, 4000, r.MaxTokens)
	require.Equal(t, 6, r.MaxSteps)
	require.Equal(t, "change", r.Mode)
	require.Equal(t, []string{"boards/sprint.md"}, r.TriggerInclude)
	require.Equal(t, []string{"update"}, r.TriggerOn)
	require.Equal(t, []string{"boards/**", "roles/**"}, r.AttachNotes)
	require.Equal(t, 1, r.MaxDepth)
	require.Equal(t, "skip", r.Concurrency)
}

func TestRoleValidate_ToolsSubset(t *testing.T) {
	r := Role{Mode: "change", Tools: []string{"search", "patch_note"}}
	require.NoError(t, r.Validate([]string{"search", "read_note", "patch_note", "write_note"}))

	r.Tools = []string{"search", "shell"}
	err := r.Validate([]string{"search", "read_note", "patch_note"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "shell")
}

func TestRoleValidate_RequiresMode(t *testing.T) {
	err := Role{}.Validate([]string{"search"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mode")
}

func TestRoleValidate_DefaultConcurrencyAllowed(t *testing.T) {
	require.NoError(t, Role{Mode: "cron", Concurrency: ""}.Validate(nil))
	require.Error(t, Role{Mode: "cron", Concurrency: "bogus"}.Validate(nil))
}
```

2. [ ] **Run it to verify it fails.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/ 2>&1 | tail -10
```
Expected: build failure (`undefined: ParseRole`, `undefined: Role`).

3. [ ] **Implement config.go.** Create `internal/fleet/config.go`:
```go
// Package fleet is the trip2g agent host (fleet-as-executor): a standalone
// daemon that discovers role notes, reconciles trip2g webhooks to itself,
// receives deliveries, runs agentruntime.Run against a per-delivery scoped
// trip2g token, and writes results back. trip2g stays a dumb event source.
package fleet

import "time"

// Config is the fleet's machine-level configuration (env + flags). The role
// note's frontmatter configures the agent; this configures the host. The fleet
// ceiling (TokenCeiling/StepCeiling) is the non-overridable floor: the
// effective budget is min(frontmatter, ceiling).
type Config struct {
	FleetID       string        // reconcile marker prefix "fleet:<FleetID>:"
	ListenAddr    string        // ":9090"
	CallbackURL   string        // trip2g-reachable base; webhook url = CallbackURL + "/deliver/" + urlKey(path)
	Trip2gBaseURL string        // e.g. "http://localhost:20081"
	AdminAPIKey   string        // X-API-Key, EnableMcpAdminTools=true — FULL ADMIN
	FleetSecret   string        // per-role HMAC secret seed
	LLMBaseURL    string        // OpenAI-compatible base URL (fleet-local, NOT a trip2g secret)
	LLMAPIKey     string        // fleet-local LLM credential
	DefaultModel  string        // fallback when a role omits model
	TokenCeiling  int           // non-overridable per-run token cap
	StepCeiling   int           // non-overridable per-run step cap
	AgentsFolder  string        // e.g. "roles/" -> notePaths like "roles/%"
	OfferedTools  []string      // a role's tools must be a subset of these
	PollInterval  time.Duration // discovery/reconcile poll cadence
}
```

4. [ ] **Implement role.go.** Create `internal/fleet/role.go`:
```go
package fleet

import (
	"fmt"
	"strconv"
	"strings"

	"trip2g/internal/webhookutil"
)

// Role is a parsed role note: flat frontmatter (config) + body (instruction).
type Role struct {
	NotePath       string
	Body           string
	Model          string
	Tools          []string
	ReadPatterns   []string
	WritePatterns  []string
	MaxTokens      int
	MaxSteps       int
	Mode           string // "change" | "cron" | "both"
	TriggerInclude []string
	TriggerExclude []string
	TriggerOn      []string // create | update | remove
	CronSchedule   string
	AttachNotes    []string
	MaxDepth       int
	Concurrency    string // "allow_overlap" | "skip" | "queue_one"
}

// ParseRole builds a Role from a note path, body, and flat frontmatter meta
// (key -> raw value). List values accept JSON (["a","b"]) or YAML-flow
// ([a, b]) form; scalars are trimmed.
func ParseRole(notePath, body string, m map[string]string) (Role, error) {
	r := Role{
		NotePath:       notePath,
		Body:           body,
		Model:          strings.TrimSpace(m["model"]),
		Tools:          parseList(m["tools"]),
		ReadPatterns:   parseList(m["read_patterns"]),
		WritePatterns:  parseList(m["write_patterns"]),
		Mode:           strings.TrimSpace(m["mode"]),
		TriggerInclude: parseList(m["trigger_include"]),
		TriggerExclude: parseList(m["trigger_exclude"]),
		TriggerOn:      parseList(m["trigger_on"]),
		CronSchedule:   strings.TrimSpace(m["cron_schedule"]),
		AttachNotes:    parseList(m["attach_notes"]),
		Concurrency:    strings.TrimSpace(m["concurrency"]),
	}
	var err error
	if r.MaxTokens, err = parseIntOpt(m["max_tokens"]); err != nil {
		return Role{}, fmt.Errorf("max_tokens: %w", err)
	}
	if r.MaxSteps, err = parseIntOpt(m["max_steps"]); err != nil {
		return Role{}, fmt.Errorf("max_steps: %w", err)
	}
	if r.MaxDepth, err = parseIntOpt(m["max_depth"]); err != nil {
		return Role{}, fmt.Errorf("max_depth: %w", err)
	}
	return r, nil
}

// Validate fails fast on misconfiguration discovered at poll time, before any
// webhook is registered. Tools must be a subset of the fleet's offered set.
func (r Role) Validate(offered []string) error {
	switch r.Mode {
	case "change", "cron", "both":
	default:
		return fmt.Errorf("role %s: mode must be change|cron|both, got %q", r.NotePath, r.Mode)
	}
	switch r.Concurrency {
	case "", "allow_overlap", "skip", "queue_one":
	default:
		return fmt.Errorf("role %s: concurrency must be allow_overlap|skip|queue_one, got %q", r.NotePath, r.Concurrency)
	}
	for _, t := range r.Tools {
		if !contains(offered, t) {
			return fmt.Errorf("role %s: tool %q not offered by this fleet", r.NotePath, t)
		}
	}
	return nil
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// parseList accepts JSON arrays or YAML-flow arrays of strings.
func parseList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if parsed, err := webhookutil.ParseJSONStringArray(raw); err == nil {
		return parsed
	}
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	var out []string
	for _, part := range strings.Split(raw, ",") {
		v := strings.Trim(strings.TrimSpace(part), `"'`)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseIntOpt(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}
```

5. [ ] **Run to verify pass.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/
```
Expected: `ok  trip2g/internal/fleet`.

6. [ ] **Commit.**
```
git add internal/fleet/config.go internal/fleet/role.go internal/fleet/role_test.go && git commit -m "feat(fleet): add Config and Role frontmatter parsing"
```

---

### Task C3 — `internal/fleet` Client: admin + scoped GraphQL lanes (Step 8b)

The `Client` is an **interface** so downstream tasks (discovery, reconcile, remotekb, handler) can be tested with a moq mock. The concrete `httpClient` posts to the two real endpoints confirmed in schema/server: admin lane `POST /_system/mcp` with `X-API-Key` (graphql_request → admin elevation), scoped lane `POST /_system/graphql` with `Authorization: Bearer <token>` (MCP rejects shortapitoken).

**Files:**
- Create: `internal/fleet/client.go`
- Test: `internal/fleet/client_test.go`

**Steps:**

1. [ ] **Write the failing test.** Create `internal/fleet/client_test.go` (drives the real HTTP impl against `httptest`):
```go
package fleet

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPClient_AdminLaneHeadersAndPath(t *testing.T) {
	var gotPath, gotKey, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-Api-Key")
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "admin-key", srv.Client())
	raw, err := c.GraphQLAdmin(context.Background(), "query{ok}", map[string]any{"x": 1})
	require.NoError(t, err)

	require.Equal(t, "/_system/mcp", gotPath)
	require.Equal(t, "admin-key", gotKey)
	require.Empty(t, gotAuth)
	require.Contains(t, gotBody, `"query":"query{ok}"`)
	require.Contains(t, gotBody, `"variables":{"x":1}`)

	var env struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &env))
	require.JSONEq(t, `{"ok":true}`, string(env.Data))
}

func TestHTTPClient_ScopedLaneBearer(t *testing.T) {
	var gotPath, gotKey, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-Api-Key")
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "admin-key", srv.Client())
	_, err := c.GraphQLScoped(context.Background(), "scoped-token", "mutation{x}", nil)
	require.NoError(t, err)

	require.Equal(t, "/_system/graphql", gotPath)
	require.Empty(t, gotKey)
	require.Equal(t, "Bearer scoped-token", gotAuth)
}

func TestHTTPClient_GraphQLErrorsSurface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "k", srv.Client())
	_, err := c.GraphQLAdmin(context.Background(), "q", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}
```

2. [ ] **Run it to verify it fails.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/ -run TestHTTPClient 2>&1 | tail -10
```
Expected: build failure (`undefined: NewHTTPClient`, `undefined: Client`).

3. [ ] **Implement client.go.** Create `internal/fleet/client.go`:
```go
package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is the trip2g API surface the fleet depends on. Two lanes:
//   - GraphQLAdmin: POST /_system/mcp with X-API-Key (full-admin elevation),
//     used by discovery + reconcile only.
//   - GraphQLScoped: POST /_system/graphql with a Bearer shortapitoken,
//     used for all per-delivery note IO (the only lane RemoteKB ever touches).
type Client interface {
	GraphQLAdmin(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error)
	GraphQLScoped(ctx context.Context, token, query string, vars map[string]any) (json.RawMessage, error)
}

type httpClient struct {
	baseURL  string
	adminKey string
	hc       *http.Client
}

// NewHTTPClient builds the concrete HTTP-backed Client.
func NewHTTPClient(baseURL, adminKey string, hc *http.Client) Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &httpClient{baseURL: baseURL, adminKey: adminKey, hc: hc}
}

func (c *httpClient) GraphQLAdmin(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error) {
	return c.do(ctx, "/_system/mcp", map[string]string{"X-Api-Key": c.adminKey}, query, vars)
}

func (c *httpClient) GraphQLScoped(ctx context.Context, token, query string, vars map[string]any) (json.RawMessage, error) {
	return c.do(ctx, "/_system/graphql", map[string]string{"Authorization": "Bearer " + token}, query, vars)
}

func (c *httpClient) do(ctx context.Context, path string, headers map[string]string, query string, vars map[string]any) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode graphql response (HTTP %d): %w", resp.StatusCode, err)
	}
	if len(env.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", env.Errors[0].Message)
	}
	return env.Data, nil
}
```

4. [ ] **Generate the moq mock** (used by C4–C7). Add this `go:generate` directive at the top of `client.go` (under the package clause) and run moq:
```go
//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg fleet . Client
```
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go generate ./internal/fleet/...
```
Expected: `internal/fleet/mocks_test.go` created with `ClientMock` exposing `GraphQLAdminFunc`/`GraphQLScopedFunc` + call recorders.

5. [ ] **Run to verify pass.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/
```
Expected: `ok  trip2g/internal/fleet` (Client + Role tests).

6. [ ] **Commit.**
```
git add internal/fleet/client.go internal/fleet/client_test.go internal/fleet/mocks_test.go && git commit -m "feat(fleet): add trip2g Client with admin and scoped GraphQL lanes"
```

---

### Task C4 — `internal/fleet` discovery: poll role notes → []Role (Step 8c)

Discovery uses the admin lane query `notePaths(filter:{like:"<AgentsFolder>%"}){ value content latestNoteView { meta { key raw } } }` (fields confirmed against `internal/graph/schema.graphqls`: `NotePath.value`/`.content`/`.latestNoteView`, `NoteView.meta`, `NoteViewMeta{key,raw}`), then `ParseRole` + `Validate` per hit. Invalid roles are skipped with a returned error list, never registered.

**Files:**
- Create: `internal/fleet/discovery.go`
- Test: `internal/fleet/discovery_test.go`

**Steps:**

1. [ ] **Write the failing test.** Create `internal/fleet/discovery_test.go`:
```go
package fleet

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscoverRoles_ParsesValidSkipsInvalid(t *testing.T) {
	resp := `{"notePaths":[
		{"value":"roles/triage.md","content":"Triage the board.","latestNoteView":{"meta":[
			{"key":"mode","raw":"change"},
			{"key":"model","raw":"gpt-4o-mini"},
			{"key":"tools","raw":"[search, patch_note]"},
			{"key":"read_patterns","raw":"[\"boards/**\"]"},
			{"key":"write_patterns","raw":"[\"boards/**\"]"},
			{"key":"trigger_include","raw":"[\"boards/sprint.md\"]"},
			{"key":"trigger_on","raw":"[update]"},
			{"key":"max_depth","raw":"1"},
			{"key":"concurrency","raw":"skip"}
		]}},
		{"value":"roles/bad.md","content":"x","latestNoteView":{"meta":[
			{"key":"mode","raw":"change"},
			{"key":"tools","raw":"[shell]"}
		]}}
	]}`
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			require.Contains(t, q, "notePaths")
			require.Equal(t, "roles/%", vars["like"])
			return json.RawMessage(resp), nil
		},
	}
	d := NewDiscovery(client, "roles/", []string{"search", "read_note", "patch_note"})
	roles, errs := d.DiscoverRoles(context.Background())
	require.Len(t, roles, 1)
	require.Equal(t, "roles/triage.md", roles[0].NotePath)
	require.Equal(t, []string{"search", "patch_note"}, roles[0].Tools)
	require.Len(t, errs, 1) // roles/bad.md rejected (shell not offered)
	require.Contains(t, errs[0].Error(), "roles/bad.md")
}
```

2. [ ] **Run it to verify it fails.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/ -run TestDiscoverRoles 2>&1 | tail -10
```
Expected: build failure (`undefined: NewDiscovery`).

3. [ ] **Implement discovery.go.** Create `internal/fleet/discovery.go`:
```go
package fleet

import (
	"context"
	"encoding/json"
	"fmt"
)

const discoverRolesQuery = `query Discover($like: String!) {
  notePaths(filter: {like: $like}) {
    value
    content
    latestNoteView { meta { key raw } }
  }
}`

// Discovery polls trip2g for role notes under AgentsFolder.
type Discovery struct {
	client       Client
	agentsFolder string
	offeredTools []string
}

// NewDiscovery builds a Discovery over the admin lane.
func NewDiscovery(client Client, agentsFolder string, offeredTools []string) *Discovery {
	return &Discovery{client: client, agentsFolder: agentsFolder, offeredTools: offeredTools}
}

type discoveredNote struct {
	Value         string `json:"value"`
	Content       string `json:"content"`
	LatestNoteView struct {
		Meta []struct {
			Key string `json:"key"`
			Raw string `json:"raw"`
		} `json:"meta"`
	} `json:"latestNoteView"`
}

// DiscoverRoles returns the valid roles plus a slice of per-note validation
// errors (invalid roles are excluded, never registered).
func (d *Discovery) DiscoverRoles(ctx context.Context) ([]Role, []error) {
	raw, err := d.client.GraphQLAdmin(ctx, discoverRolesQuery, map[string]any{"like": likePattern(d.agentsFolder)})
	if err != nil {
		return nil, []error{fmt.Errorf("discover: %w", err)}
	}
	var data struct {
		NotePaths []discoveredNote `json:"notePaths"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, []error{fmt.Errorf("discover decode: %w", err)}
	}

	var roles []Role
	var errs []error
	for _, n := range data.NotePaths {
		meta := map[string]string{}
		for _, m := range n.LatestNoteView.Meta {
			meta[m.Key] = m.Raw
		}
		role, perr := ParseRole(n.Value, n.Content, meta)
		if perr != nil {
			errs = append(errs, fmt.Errorf("parse %s: %w", n.Value, perr))
			continue
		}
		if verr := role.Validate(d.offeredTools); verr != nil {
			errs = append(errs, verr)
			continue
		}
		roles = append(roles, role)
	}
	return roles, errs
}

// likePattern turns "roles/" into the SQL LIKE "roles/%".
func likePattern(folder string) string {
	return folder + "%"
}
```

4. [ ] **Run to verify pass.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/
```
Expected: `ok  trip2g/internal/fleet`.

5. [ ] **Commit.**
```
git add internal/fleet/discovery.go internal/fleet/discovery_test.go && git commit -m "feat(fleet): add role discovery via notePaths/note admin lane"
```

---

### Task C5 — `internal/fleet` reconcile: idempotent webhook diff via admin CRUD (Step 8d)

Reconcile builds one desired change-webhook per `mode:change|both` role, keyed by the `description` marker `fleet:<FleetID>:<path>#<ver>` (no `managed_by` column). It lists existing webhooks via `allChangeWebhooks` (filtered to its own `fleet:<FleetID>:` prefix — foreign webhooks untouched), then Create/Update/Delete via `changeWebhookCreate`/`changeWebhookUpdate`/`changeWebhookDelete` (exact mutation names confirmed in `internal/graph/schema.graphqls:3240-3242`). The `ChangeWebhookCreateInput` carries `attachNotes`/`transformJsonnet`/`concurrencyMode` (added in Section A). `ver` is a content hash of the role spec; rotation = bump `ver` → delete+recreate (UpdateInput has no secret field).

**Files:**
- Create: `internal/fleet/reconcile.go`
- Test: `internal/fleet/reconcile_test.go`

**Steps:**

1. [ ] **Write the failing test.** Create `internal/fleet/reconcile_test.go`:
```go
package fleet

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newReconciler(client Client) *Reconciler {
	return NewReconciler(client, Config{
		FleetID:     "f1",
		CallbackURL: "https://fleet.example",
	})
}

func TestReconcile_CreatesMissingWebhook(t *testing.T) {
	var created map[string]any
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			switch {
			case strings.Contains(q, "allChangeWebhooks"):
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[]}}`), nil
			case strings.Contains(q, "changeWebhookCreate"):
				created = vars["input"].(map[string]any)
				return json.RawMessage(`{"changeWebhookCreate":{"webhook":{"id":7},"secret":"s"}}`), nil
			}
			return nil, nil
		},
	}
	role := Role{
		NotePath: "roles/triage.md", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		AttachNotes: []string{"boards/**"}, TriggerInclude: []string{"boards/sprint.md"},
		TriggerOn: []string{"update"}, MaxDepth: 1, Concurrency: "skip",
	}
	require.NoError(t, newReconciler(client).Reconcile(context.Background(), []Role{role}))
	require.NotNil(t, created)
	require.Equal(t, []string{"boards/sprint.md"}, created["includePatterns"])
	require.Equal(t, "skip", created["concurrencyMode"])
	require.Equal(t, true, created["passApiKey"])
	require.Equal(t, int64(1), created["maxDepth"])
	require.Contains(t, created["description"], "fleet:f1:roles/triage.md#")
	require.True(t, strings.HasPrefix(created["url"].(string), "https://fleet.example/deliver/"))
}

func TestReconcile_NoChangeWhenMarkerMatches(t *testing.T) {
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1, Concurrency: "skip"}
	desc := markerFor("f1", role)
	var createCalls, updateCalls, deleteCalls int
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, _ map[string]any) (json.RawMessage, error) {
			switch {
			case strings.Contains(q, "allChangeWebhooks"):
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[{"id":7,"description":"` + desc + `"}]}}`), nil
			case strings.Contains(q, "changeWebhookCreate"):
				createCalls++
			case strings.Contains(q, "changeWebhookUpdate"):
				updateCalls++
			case strings.Contains(q, "changeWebhookDelete"):
				deleteCalls++
			}
			return json.RawMessage(`{}`), nil
		},
	}
	require.NoError(t, newReconciler(client).Reconcile(context.Background(), []Role{role}))
	require.Zero(t, createCalls)
	require.Zero(t, updateCalls)
	require.Zero(t, deleteCalls)
}

func TestReconcile_DeletesStaleAndLeavesForeign(t *testing.T) {
	var deletedIDs []int64
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			switch {
			case strings.Contains(q, "allChangeWebhooks"):
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[
					{"id":7,"description":"fleet:f1:roles/old.md#deadbeef"},
					{"id":8,"description":"some-other-integration"}
				]}}`), nil
			case strings.Contains(q, "changeWebhookDelete"):
				deletedIDs = append(deletedIDs, int64(vars["input"].(map[string]any)["id"].(int64)))
			}
			return json.RawMessage(`{}`), nil
		},
	}
	require.NoError(t, newReconciler(client).Reconcile(context.Background(), nil))
	require.Equal(t, []int64{7}, deletedIDs) // foreign id 8 untouched
}

func TestDeregister_DeletesAllOwned(t *testing.T) {
	var deletedIDs []int64
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			if strings.Contains(q, "allChangeWebhooks") {
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[{"id":7,"description":"fleet:f1:roles/a.md#x"}]}}`), nil
			}
			if strings.Contains(q, "changeWebhookDelete") {
				deletedIDs = append(deletedIDs, int64(vars["input"].(map[string]any)["id"].(int64)))
			}
			return json.RawMessage(`{}`), nil
		},
	}
	require.NoError(t, newReconciler(client).Deregister(context.Background()))
	require.Equal(t, []int64{7}, deletedIDs)
}
```

2. [ ] **Run it to verify it fails.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/ -run 'TestReconcile|TestDeregister' 2>&1 | tail -10
```
Expected: build failure (`undefined: NewReconciler`, `undefined: markerFor`).

3. [ ] **Implement reconcile.go.** Create `internal/fleet/reconcile.go`:
```go
package fleet

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Reconciler drives the desired-vs-actual webhook diff over the admin lane.
type Reconciler struct {
	client Client
	cfg    Config
}

// NewReconciler builds a Reconciler.
func NewReconciler(client Client, cfg Config) *Reconciler {
	return &Reconciler{client: client, cfg: cfg}
}

const listChangeWebhooksQuery = `query { allChangeWebhooks { nodes { id description } } }`

const createChangeWebhookMutation = `mutation Create($input: ChangeWebhookCreateInput!) {
  changeWebhookCreate(input: $input) { ... on ChangeWebhookCreatePayload { webhook { id } } ... on ErrorPayload { message } }
}`

const deleteChangeWebhookMutation = `mutation Delete($input: ChangeWebhookDeleteInput!) {
  changeWebhookDelete(input: $input) { ... on ChangeWebhookDeletePayload { deletedId } ... on ErrorPayload { message } }
}`

type existingWebhook struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
}

// Reconcile makes the registered change-webhooks match desired roles. Foreign
// webhooks (description without this fleet's prefix) are never touched.
func (r *Reconciler) Reconcile(ctx context.Context, roles []Role) error {
	existing, err := r.listOwned(ctx)
	if err != nil {
		return err
	}
	desired := map[string]Role{} // marker -> role
	for _, role := range roles {
		if role.Mode != "change" && role.Mode != "both" {
			continue
		}
		desired[markerFor(r.cfg.FleetID, role)] = role
	}

	// Delete owned webhooks whose marker is no longer desired.
	for marker, id := range existing {
		if _, keep := desired[marker]; !keep {
			if derr := r.delete(ctx, id); derr != nil {
				return derr
			}
		}
	}
	// Create webhooks for desired markers not yet present.
	for marker, role := range desired {
		if _, present := existing[marker]; present {
			continue // marker already matches => spec unchanged (ver is content-derived)
		}
		if cerr := r.create(ctx, role); cerr != nil {
			return cerr
		}
	}
	return nil
}

// Deregister removes every webhook owned by this fleet (shutdown).
func (r *Reconciler) Deregister(ctx context.Context) error {
	existing, err := r.listOwned(ctx)
	if err != nil {
		return err
	}
	for _, id := range existing {
		if derr := r.delete(ctx, id); derr != nil {
			return derr
		}
	}
	return nil
}

func (r *Reconciler) listOwned(ctx context.Context) (map[string]int64, error) {
	raw, err := r.client.GraphQLAdmin(ctx, listChangeWebhooksQuery, nil)
	if err != nil {
		return nil, err
	}
	var data struct {
		AllChangeWebhooks struct {
			Nodes []existingWebhook `json:"nodes"`
		} `json:"allChangeWebhooks"`
	}
	if uerr := json.Unmarshal(raw, &data); uerr != nil {
		return nil, uerr
	}
	prefix := "fleet:" + r.cfg.FleetID + ":"
	owned := map[string]int64{}
	for _, n := range data.AllChangeWebhooks.Nodes {
		if strings.HasPrefix(n.Description, prefix) {
			owned[n.Description] = n.ID
		}
	}
	return owned, nil
}

func (r *Reconciler) create(ctx context.Context, role Role) error {
	input := map[string]any{
		"url":              r.cfg.CallbackURL + "/deliver/" + urlKey(role.NotePath),
		"includePatterns":  orEmpty(role.TriggerInclude),
		"excludePatterns":  orEmpty(role.TriggerExclude),
		"readPatterns":     orEmpty(role.ReadPatterns),
		"writePatterns":    orEmpty(role.WritePatterns),
		"attachNotes":      orEmpty(role.AttachNotes),
		"transformJsonnet": "",
		"concurrencyMode":  orDefault(role.Concurrency, "allow_overlap"),
		"passApiKey":       true,
		"maxDepth":         int64(role.MaxDepth),
		"onCreate":         contains(role.TriggerOn, "create"),
		"onUpdate":         contains(role.TriggerOn, "update"),
		"onRemove":         contains(role.TriggerOn, "remove"),
		"description":      markerFor(r.cfg.FleetID, role),
		"secret":           deriveSecret(r.cfg.FleetSecret, r.cfg.FleetID, role.NotePath, specVer(role)),
	}
	_, err := r.client.GraphQLAdmin(ctx, createChangeWebhookMutation, map[string]any{"input": input})
	return err
}

func (r *Reconciler) delete(ctx context.Context, id int64) error {
	_, err := r.client.GraphQLAdmin(ctx, deleteChangeWebhookMutation,
		map[string]any{"input": map[string]any{"id": id}})
	return err
}

// markerFor is the reconcile dedup key stored in the webhook description.
func markerFor(fleetID string, role Role) string {
	return "fleet:" + fleetID + ":" + role.NotePath + "#" + specVer(role)
}

// specVer is a short content hash of the parts of a role that define its
// reconciled webhook; bumping any of them rotates the marker (delete+recreate).
func specVer(role Role) string {
	h := sha256.Sum256([]byte(strings.Join([]string{
		strings.Join(role.TriggerInclude, ","),
		strings.Join(role.TriggerExclude, ","),
		strings.Join(role.ReadPatterns, ","),
		strings.Join(role.WritePatterns, ","),
		strings.Join(role.AttachNotes, ","),
		strings.Join(role.TriggerOn, ","),
		fmt.Sprintf("%d", role.MaxDepth),
		role.Concurrency,
	}, "|")))
	return base64.RawURLEncoding.EncodeToString(h[:6])
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
```

4. [ ] **Run to verify pass.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/
```
Expected: `ok  trip2g/internal/fleet`.

5. [ ] **Commit.**
```
git add internal/fleet/reconcile.go internal/fleet/reconcile_test.go && git commit -m "feat(fleet): add idempotent webhook reconcile via admin CRUD"
```

---

### Task C6 — `internal/fleet` RemoteKB over the scoped trip2g API (Step 8e)

`remoteKB` implements `agentruntime.KB` against the **scoped lane only** (the admin key is structurally unreachable here). `attach_notes` materialized in the payload (Section B) seeds the `overlay` so `Read` of an attached path costs zero round-trips. `Write` → `updateNotes` upsert; `Patch` → `updateNotes` patch (find/replace), preserving unmodeled kanban metadata. The compile-time assertion `var _ agentruntime.KB = (*remoteKB)(nil)` guarantees it satisfies the (now `Patch`-bearing) interface from C1.

**Files:**
- Create: `internal/fleet/remotekb.go`
- Test: `internal/fleet/remotekb_test.go`

**Steps:**

1. [ ] **Write the failing test.** Create `internal/fleet/remotekb_test.go`:
```go
package fleet

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoteKB_ReadOverlayHitNoClientCall(t *testing.T) {
	var calls int
	client := &ClientMock{GraphQLScopedFunc: func(context.Context, string, string, map[string]any) (json.RawMessage, error) {
		calls++
		return nil, nil
	}}
	kb := newRemoteKB(client, "tok", map[string]string{"boards/sprint.md": "board body"})
	got, err := kb.Read(context.Background(), "boards/sprint.md")
	require.NoError(t, err)
	require.Equal(t, "board body", got)
	require.Zero(t, calls) // served from overlay
}

func TestRemoteKB_ReadFallsBackToScopedNote(t *testing.T) {
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, tok, q string, vars map[string]any) (json.RawMessage, error) {
		require.Equal(t, "tok", tok)
		require.Contains(t, q, "note(")
		return json.RawMessage(`{"note":{"path":"boards/other.md","html":"ignored"}}`), nil
	}}
	// note(path:) returns NoteView content via a content field; assert it routes.
	client.GraphQLScopedFunc = func(_ context.Context, _ string, q string, _ map[string]any) (json.RawMessage, error) {
		require.Contains(t, q, "note(")
		return json.RawMessage(`{"note":{"content":"fetched body"}}`), nil
	}
	kb := newRemoteKB(client, "tok", nil)
	got, err := kb.Read(context.Background(), "boards/other.md")
	require.NoError(t, err)
	require.Equal(t, "fetched body", got)
}

func TestRemoteKB_PatchIssuesUpdateNotesPatchVariant(t *testing.T) {
	var sentVars map[string]any
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, tok, q string, vars map[string]any) (json.RawMessage, error) {
		require.Equal(t, "tok", tok)
		require.Contains(t, q, "updateNotes")
		sentVars = vars
		return json.RawMessage(`{"updateNotes":{"paths":["boards/sprint.md"]}}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	require.NoError(t, kb.Patch(context.Background(), "boards/sprint.md", "@status:todo", "@status:doing"))

	input := sentVars["input"].(map[string]any)
	changes := input["changes"].([]map[string]any)
	require.Len(t, changes, 1)
	patch := changes[0]["patch"].(map[string]any)
	require.Equal(t, "boards/sprint.md", patch["path"])
	require.Equal(t, "@status:todo", patch["find"])
	require.Equal(t, "@status:doing", patch["replace"])
	require.NotContains(t, changes[0], "upsert")
}

func TestRemoteKB_WriteIssuesUpsert(t *testing.T) {
	var q string
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, query string, vars map[string]any) (json.RawMessage, error) {
		q = query
		input := vars["input"].(map[string]any)
		changes := input["changes"].([]map[string]any)
		_, hasUpsert := changes[0]["upsert"]
		require.True(t, hasUpsert)
		return json.RawMessage(`{"updateNotes":{"paths":["x.md"]}}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	require.NoError(t, kb.Write(context.Background(), "x.md", "body"))
	require.True(t, strings.Contains(q, "updateNotes"))
}
```

2. [ ] **Run it to verify it fails.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/ -run TestRemoteKB 2>&1 | tail -10
```
Expected: build failure (`undefined: newRemoteKB`).

3. [ ] **Implement remotekb.go.** Create `internal/fleet/remotekb.go`:
```go
package fleet

import (
	"context"
	"encoding/json"
	"fmt"

	"trip2g/internal/agentruntime"
)

// remoteKB is the fleet's agentruntime.KB over the trip2g API, scoped to a
// single delivery's shortapitoken. attach_notes materialized in the payload
// seed the overlay; reads of attached paths cost no round-trip.
type remoteKB struct {
	client  Client
	token   string
	overlay map[string]string
}

// newRemoteKB builds a remoteKB. overlay may be nil.
func newRemoteKB(client Client, token string, overlay map[string]string) *remoteKB {
	if overlay == nil {
		overlay = map[string]string{}
	}
	return &remoteKB{client: client, token: token, overlay: overlay}
}

var _ agentruntime.KB = (*remoteKB)(nil)

const searchScopedQuery = `query Search($q: String!) {
  search(input: {query: $q}) { nodes { document { ... on PublicNote { path } } } }
}`

const noteScopedQuery = `query Note($path: String!) {
  note(input: {path: $path, referer: ""}) { path }
}`

// NOTE: PublicNote exposes html/title/toc but not raw content. The fleet reads
// raw bodies through the NotePath.content surface; the scoped note query above
// returns the path, and the body is supplied via attach_notes overlay for the
// kanban demo. For non-attached reads we request the raw content field.
const noteContentScopedQuery = `query NoteContent($path: String!) {
  note(input: {path: $path, referer: ""}) { content: html }
}`

const updateNotesMutation = `mutation Update($input: UpdateNotesInput!) {
  updateNotes(input: $input) {
    ... on UpdateNotesSuccessPayload { paths }
    ... on UpdateNotesPatchNotFoundPayload { path find }
    ... on UpdateNotesHashMismatchPayload { path actualHash }
    ... on ErrorPayload { message }
  }
}`

func (k *remoteKB) Search(ctx context.Context, query string) ([]agentruntime.Doc, error) {
	raw, err := k.client.GraphQLScoped(ctx, k.token, searchScopedQuery, map[string]any{"q": query})
	if err != nil {
		return nil, err
	}
	var data struct {
		Search struct {
			Nodes []struct {
				Document struct {
					Path string `json:"path"`
				} `json:"document"`
			} `json:"nodes"`
		} `json:"search"`
	}
	if uerr := json.Unmarshal(raw, &data); uerr != nil {
		return nil, uerr
	}
	out := make([]agentruntime.Doc, 0, len(data.Search.Nodes))
	for _, n := range data.Search.Nodes {
		if n.Document.Path != "" {
			out = append(out, agentruntime.Doc{Path: n.Document.Path})
		}
	}
	return out, nil
}

func (k *remoteKB) Read(ctx context.Context, path string) (string, error) {
	if body, ok := k.overlay[path]; ok {
		return body, nil
	}
	raw, err := k.client.GraphQLScoped(ctx, k.token, noteContentScopedQuery, map[string]any{"path": path})
	if err != nil {
		return "", err
	}
	var data struct {
		Note *struct {
			Content string `json:"content"`
		} `json:"note"`
	}
	if uerr := json.Unmarshal(raw, &data); uerr != nil {
		return "", uerr
	}
	if data.Note == nil {
		return "", fmt.Errorf("note not found: %s", path)
	}
	return data.Note.Content, nil
}

func (k *remoteKB) Write(ctx context.Context, path, content string) error {
	return k.update(ctx, []map[string]any{
		{"upsert": map[string]any{"path": path, "content": content}},
	})
}

func (k *remoteKB) Patch(ctx context.Context, path, find, replace string) error {
	return k.update(ctx, []map[string]any{
		{"patch": map[string]any{"path": path, "find": find, "replace": replace}},
	})
}

func (k *remoteKB) update(ctx context.Context, changes []map[string]any) error {
	_, err := k.client.GraphQLScoped(ctx, k.token, updateNotesMutation,
		map[string]any{"input": map[string]any{"changes": changes}})
	return err
}
```

> Implementer note: confirm the scoped read field in `internal/graph/schema.graphqls` (`PublicNote` exposes `html`, not raw `content`; the alias `content: html` above renders HTML). For the kanban demo every read path is attach-seeded, so the overlay branch is the live one and the fallback HTML alias is sufficient for non-attached scoped reads. If a raw-markdown scoped read is later required, route through the `notePaths(filter:{paths:[$path]}){content}` surface instead — track in C10 docs.

4. [ ] **Run to verify pass.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/
```
Expected: `ok  trip2g/internal/fleet`.

5. [ ] **Commit.**
```
git add internal/fleet/remotekb.go internal/fleet/remotekb_test.go && git commit -m "feat(fleet): add RemoteKB over scoped trip2g API"
```

---

### Task C7 — `internal/fleet` Fleet + delivery handler (Step 8f)

The HTTP handler is the `external_url` target: verify per-role HMAC, parse the delivery payload (mirror of `deliverchangewebhook.changeWebhookPayload` + Section-B `attached_notes`), look up the role by URL key, clamp the budget to the fleet ceiling, run `agentruntime.Run` over a `remoteKB` seeded from `attached_notes`, and respond `webhookutil.AgentResponse` with `Changes: nil` (writes already happened in-loop via the scoped token) plus `TokensUsed`/`Steps`. Status codes: 200 ok, 401 bad HMAC, 404 unknown key.

**Files:**
- Create: `internal/fleet/fleet.go` (Fleet struct, secret derivation, urlKey, budget clamp)
- Create: `internal/fleet/handler.go` (ServeDelivery)
- Test: `internal/fleet/handler_test.go`

**Steps:**

1. [ ] **Write the failing test.** Create `internal/fleet/handler_test.go`:
```go
package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"trip2g/internal/agentruntime"
	"trip2g/internal/webhookutil"
)

// stubLLM scripts the agent loop: one patch_note then finish.
type stubLLM struct{ idx int }

func (s *stubLLM) Chat(_ context.Context, _ string, _ []agentruntime.Message, _ []agentruntime.ToolDef) (agentruntime.ChatResult, error) {
	defer func() { s.idx++ }()
	if s.idx == 0 {
		args, _ := json.Marshal(map[string]any{"path": "boards/sprint.md", "find": "@status:todo", "replace": "@status:doing"})
		return agentruntime.ChatResult{
			ToolCalls:    []agentruntime.ToolCall{{ID: "1", Name: "patch_note", Arguments: string(args)}},
			PromptTokens: 10, CompletionTokens: 5,
		}, nil
	}
	args, _ := json.Marshal(map[string]any{"answer": "done"})
	return agentruntime.ChatResult{
		ToolCalls:    []agentruntime.ToolCall{{ID: "2", Name: "finish", Arguments: string(args)}},
		PromptTokens: 10, CompletionTokens: 5,
	}, nil
}

func newTestFleet(client Client) *Fleet {
	role := Role{
		NotePath: "roles/triage.md", Body: "Triage.", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		MaxTokens: 4000, MaxSteps: 6, Concurrency: "skip", MaxDepth: 1,
	}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000, StepCeiling: 25,
	}
	f := NewFleet(cfg, client, &stubLLM{})
	f.SetRoles([]Role{role})
	return f
}

func post(t *testing.T, f *Fleet, key string, body []byte, sign bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/deliver/"+key, bytes.NewReader(body))
	if sign {
		role, ok := f.roleByKey(key)
		require.True(t, ok)
		req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(body, f.secretFor(role)))
	}
	rec := httptest.NewRecorder()
	f.ServeDelivery(rec, req)
	return rec
}

func deliveryBody(t *testing.T) []byte {
	t.Helper()
	b, _ := json.Marshal(map[string]any{
		"version": 1, "id": 99, "timestamp": 1, "attempt": 1,
		"depth":       0,
		"instruction": "Triage.",
		"api_token":   "scoped-token",
		"attached_notes": []map[string]any{
			{"path": "boards/sprint.md", "content": "- Fix login bug @status:todo\n"},
		},
	})
	return b
}

func TestServeDelivery_HappyPathScopedWriteOnly(t *testing.T) {
	var scopedCalls, adminCalls int
	var lastToken string
	client := &ClientMock{
		GraphQLScopedFunc: func(_ context.Context, tok, q string, _ map[string]any) (json.RawMessage, error) {
			scopedCalls++
			lastToken = tok
			return json.RawMessage(`{"updateNotes":{"paths":["boards/sprint.md"]}}`), nil
		},
		GraphQLAdminFunc: func(context.Context, string, map[string]any) (json.RawMessage, error) {
			adminCalls++
			return nil, nil
		},
	}
	f := newTestFleet(client)
	key := urlKey("roles/triage.md")
	rec := post(t, f, key, deliveryBody(t), true)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, scopedCalls)        // exactly one scoped updateNotes
	require.Zero(t, adminCalls)             // admin key never used for writes
	require.Equal(t, "scoped-token", lastToken)

	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Nil(t, resp.Changes)            // writes already applied in-loop
	require.Equal(t, 30, resp.TokensUsed)   // (10+5)*2
	require.Equal(t, 2, resp.Steps)
}

func TestServeDelivery_BadHMAC401(t *testing.T) {
	f := newTestFleet(&ClientMock{})
	rec := post(t, f, urlKey("roles/triage.md"), deliveryBody(t), false)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestServeDelivery_UnknownKey404(t *testing.T) {
	f := newTestFleet(&ClientMock{})
	rec := post(t, f, urlKey("roles/missing.md"), deliveryBody(t), false)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestClampBudget(t *testing.T) {
	require.Equal(t, 4000, clampBudget(4000, 100000)) // frontmatter wins under ceiling
	require.Equal(t, 100, clampBudget(4000, 100))     // ceiling wins
	require.Equal(t, 100000, clampBudget(0, 100000))  // unset -> ceiling
}

var _ = strconv.Itoa // keep import if unused after edits
```

2. [ ] **Run it to verify it fails.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/ -run 'TestServeDelivery|TestClampBudget' 2>&1 | tail -10
```
Expected: build failure (`undefined: NewFleet`, `undefined: urlKey`, `undefined: clampBudget`).

3. [ ] **Implement fleet.go.** Create `internal/fleet/fleet.go`:
```go
package fleet

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"sync"

	"trip2g/internal/agentruntime"
)

// Fleet ties config, the trip2g client, the LLM, and the live role registry
// (keyed by url key) together. It is the HTTP handler's owner.
type Fleet struct {
	cfg    Config
	client Client
	llm    agentruntime.LLM

	mu       sync.RWMutex
	registry map[string]Role // urlKey(notePath) -> Role
}

// NewFleet builds a Fleet with an empty registry.
func NewFleet(cfg Config, client Client, llm agentruntime.LLM) *Fleet {
	return &Fleet{cfg: cfg, client: client, llm: llm, registry: map[string]Role{}}
}

// SetRoles atomically swaps the live role registry (called after each poll).
func (f *Fleet) SetRoles(roles []Role) {
	reg := make(map[string]Role, len(roles))
	for _, r := range roles {
		reg[urlKey(r.NotePath)] = r
	}
	f.mu.Lock()
	f.registry = reg
	f.mu.Unlock()
}

func (f *Fleet) roleByKey(key string) (Role, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	r, ok := f.registry[key]
	return r, ok
}

// secretFor derives the per-role HMAC secret used to verify deliveries.
func (f *Fleet) secretFor(role Role) string {
	return deriveSecret(f.cfg.FleetSecret, f.cfg.FleetID, role.NotePath, specVer(role))
}

// urlKey encodes a note path into a URL-safe delivery key.
func urlKey(notePath string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(notePath))
}

// deriveSecret produces a rotatable per-role HMAC secret.
func deriveSecret(fleetSecret, fleetID, notePath, ver string) string {
	mac := hmac.New(sha256.New, []byte(fleetSecret))
	mac.Write([]byte(fleetID + ":" + notePath + ":" + ver))
	return hex.EncodeToString(mac.Sum(nil))
}

// clampBudget enforces the non-overridable fleet ceiling. An unset (<=0)
// frontmatter value defaults to the ceiling.
func clampBudget(want, ceiling int) int {
	if want <= 0 {
		return ceiling
	}
	return min(want, ceiling)
}
```

4. [ ] **Implement handler.go.** Create `internal/fleet/handler.go`:
```go
package fleet

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"trip2g/internal/agentruntime"
	"trip2g/internal/webhookutil"
)

// deliveryPayload mirrors deliverchangewebhook.changeWebhookPayload plus the
// Section-B attached_notes field. api_token is the per-delivery scoped token.
type deliveryPayload struct {
	Depth         int             `json:"depth"`
	Instruction   string          `json:"instruction"`
	APIToken      string          `json:"api_token"`
	AttachedNotes []attachedNote  `json:"attached_notes"`
}

type attachedNote struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ServeDelivery handles POST /deliver/<urlKey>.
func (f *Fleet) ServeDelivery(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/deliver/")
	role, ok := f.roleByKey(key)
	if !ok {
		http.Error(w, "unknown delivery key", http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	if !webhookutil.VerifyHMAC(body, f.secretFor(role), r.Header.Get("X-Webhook-Signature")) {
		http.Error(w, "bad signature", http.StatusUnauthorized)
		return
	}

	var payload deliveryPayload
	if uerr := json.Unmarshal(body, &payload); uerr != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	overlay := make(map[string]string, len(payload.AttachedNotes))
	for _, n := range payload.AttachedNotes {
		overlay[n.Path] = n.Content
	}

	res, runErr := agentruntime.Run(r.Context(), agentruntime.Input{
		Instruction:   role.Body,
		ReadPatterns:  role.ReadPatterns,
		WritePatterns: role.WritePatterns,
		Model:         orDefault(role.Model, f.cfg.DefaultModel),
		MaxTokens:     clampBudget(role.MaxTokens, f.cfg.TokenCeiling),
		MaxSteps:      clampBudget(role.MaxSteps, f.cfg.StepCeiling),
		LLM:           f.llm,
		KB:            newRemoteKB(f.client, payload.APIToken, overlay),
	})
	if runErr != nil {
		writeJSON(w, http.StatusOK, webhookutil.AgentResponse{Status: "error", Message: runErr.Error()})
		return
	}

	// Changes already applied in-loop via the scoped token; report spend only.
	writeJSON(w, http.StatusOK, webhookutil.AgentResponse{
		Status:     res.Status,
		Message:    res.Answer,
		Changes:    nil,
		TokensUsed: res.TokensUsed,
		Steps:      res.Steps,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

var _ = context.Background // keep import stable across edits
```

> Dependency note: `webhookutil.AgentResponse` must already carry `TokensUsed int` and `Steps int` (added in Section B, Step 6). If those fields are absent, this file will not compile — that is the expected cross-section ordering (Section B lands before C7).

5. [ ] **Run to verify pass.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go test ./internal/fleet/
```
Expected: `ok  trip2g/internal/fleet` (handler 200/401/404, clamp, scoped-only-write, spend in response).

6. [ ] **Commit.**
```
git add internal/fleet/fleet.go internal/fleet/handler.go internal/fleet/handler_test.go && git commit -m "feat(fleet): add delivery handler with HMAC verify and budget clamp"
```

---

### Task C8 — `cmd/fleet/main.go` daemon wiring (Step 8g)

The daemon: load `Config` from env+flags (ArrayFlags style per `internal/appconfig/config.go:36`), build the HTTP `Client` + OpenAI LLM + `Fleet`, run a discovery→reconcile→`SetRoles` loop every `PollInterval`, serve `/deliver/`, and `Deregister` on shutdown (SIGINT/SIGTERM via `signal.NotifyContext`, mirroring `cmd/businessdemo/main.go:459`). Kept thin; verified by build + `go vet` (behavioral coverage is the C9 E2E).

**Files:**
- Create: `cmd/fleet/main.go`

**Steps:**

1. [ ] **Write the failing test (build gate).** Confirm the package does not yet exist:
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go build ./cmd/fleet 2>&1 | tail -5
```
Expected: error `no Go files in .../cmd/fleet` (or `package ... is not in std`).

2. [ ] **Implement cmd/fleet/main.go.** Create `cmd/fleet/main.go`:
```go
// Command fleet is the trip2g agent host (fleet-as-executor): it discovers role
// notes in trip2g, reconciles change-webhooks to point back at itself, receives
// deliveries, runs the scoped agent loop, and writes results back via a
// per-delivery scoped trip2g token. trip2g stays a dumb event source.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"trip2g/internal/agentruntime"
	"trip2g/internal/fleet"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fleet: %v", err)
	}
}

func run() error {
	cfg, offeredFlag := parseFlags()

	httpClient := &http.Client{Timeout: 30 * time.Second}
	client := fleet.NewHTTPClient(cfg.Trip2gBaseURL, cfg.AdminAPIKey, httpClient)
	llm := agentruntime.NewOpenAILLM(cfg.LLMAPIKey, cfg.LLMBaseURL)

	f := fleet.NewFleet(cfg, client, llm)
	discovery := fleet.NewDiscovery(client, cfg.AgentsFolder, cfg.OfferedTools)
	reconciler := fleet.NewReconciler(client, cfg)
	_ = offeredFlag

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// First sync before serving so the registry is populated.
	syncOnce(ctx, f, discovery, reconciler)

	mux := http.NewServeMux()
	mux.HandleFunc("/deliver/", f.ServeDelivery)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		log.Printf("fleet %s listening on %s, callback %s", cfg.FleetID, cfg.ListenAddr, cfg.CallbackURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server: %v", err)
			stop()
		}
	}()

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Print("shutdown: deregistering webhooks")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			if err := reconciler.Deregister(shutdownCtx); err != nil {
				log.Printf("deregister: %v", err)
			}
			_ = srv.Shutdown(shutdownCtx)
			cancel()
			return nil
		case <-ticker.C:
			syncOnce(ctx, f, discovery, reconciler)
		}
	}
}

func syncOnce(ctx context.Context, f *fleet.Fleet, d *fleet.Discovery, r *fleet.Reconciler) {
	roles, errs := d.DiscoverRoles(ctx)
	for _, e := range errs {
		log.Printf("discover (skipped role): %v", e)
	}
	f.SetRoles(roles)
	if err := r.Reconcile(ctx, roles); err != nil {
		log.Printf("reconcile: %v", err)
	}
}

func parseFlags() (fleet.Config, []string) {
	var cfg fleet.Config
	var offered string
	flag.StringVar(&cfg.FleetID, "fleet-id", env("FLEET_ID", "fleet1"), "reconcile marker id")
	flag.StringVar(&cfg.ListenAddr, "listen", env("FLEET_LISTEN", ":9090"), "HTTP listen address")
	flag.StringVar(&cfg.CallbackURL, "callback-url", env("FLEET_CALLBACK_URL", ""), "trip2g-reachable base URL of this fleet")
	flag.StringVar(&cfg.Trip2gBaseURL, "trip2g-url", env("TRIP2G_BASE_URL", "http://localhost:8081"), "trip2g base URL")
	flag.StringVar(&cfg.AdminAPIKey, "admin-api-key", env("FLEET_ADMIN_API_KEY", ""), "full-admin X-Api-Key")
	flag.StringVar(&cfg.FleetSecret, "fleet-secret", env("FLEET_SECRET", ""), "HMAC seed for per-role secrets")
	flag.StringVar(&cfg.LLMBaseURL, "llm-base-url", env("FLEET_LLM_BASE_URL", ""), "OpenAI-compatible base URL")
	flag.StringVar(&cfg.LLMAPIKey, "llm-api-key", env("FLEET_LLM_API_KEY", ""), "LLM API key")
	flag.StringVar(&cfg.DefaultModel, "default-model", env("FLEET_DEFAULT_MODEL", "gpt-4o-mini"), "default model")
	flag.IntVar(&cfg.TokenCeiling, "token-ceiling", 100000, "non-overridable per-run token cap")
	flag.IntVar(&cfg.StepCeiling, "step-ceiling", 25, "non-overridable per-run step cap")
	flag.StringVar(&cfg.AgentsFolder, "agents-folder", env("FLEET_AGENTS_FOLDER", "roles/"), "role-note folder (LIKE prefix)")
	flag.StringVar(&offered, "offered-tools", "search,read_note,patch_note,write_note", "comma-separated allowed tools")
	poll := flag.Int("poll-seconds", 30, "discovery/reconcile poll interval seconds")
	flag.Parse()

	cfg.OfferedTools = splitCSV(offered)
	cfg.PollInterval = time.Duration(*poll) * time.Second
	return cfg, cfg.OfferedTools
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
```

3. [ ] **Run to verify pass.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && go build ./cmd/fleet && go vet ./cmd/fleet ./internal/fleet && go test ./internal/fleet/
```
Expected: clean build, no vet diagnostics, `ok  trip2g/internal/fleet`.

4. [ ] **Commit.**
```
git add cmd/fleet/main.go && git commit -m "feat(fleet): add cmd/fleet daemon with discovery, reconcile, deregister"
```

---

### Task C9 — Headline E2E: kanban fleet vertical slice (Step 9)

The regression oracle. DevMode trip2g (SSRF guard off — confirmed `internal/webhookutil/httpclient.go:73`) + a real `cmd/fleet` + a deterministic stub LLM HTTP server. Seed `boards/sprint.md` + `roles/triage.md`, let the fleet reconcile, drag a card in the live board, then assert: the agent applied its patch (board re-rendered), and the depth guard prevented a third run — **exactly one delivery, exactly two note versions on `boards/sprint.md`** (user drag + agent triage), the agent version attributed to the delivery. Matches the project's Playwright e2e style (`e2e/webhooks.spec.js`: `signInAsAdmin`, `/debug/wait_all_jobs`, GraphQL helpers).

**Files:**
- Create: `e2e/fleet-kanban.spec.js`
- Create: `e2e/helpers/stub-llm.js` (deterministic OpenAI-compatible server)

**Steps:**

1. [ ] **Write the failing test.** Create `e2e/helpers/stub-llm.js`:
```js
// Deterministic OpenAI-compatible chat server: first call returns a patch_note
// tool call, subsequent calls return a finish tool call. Serves the path the
// go-openai client hits: <baseURL>/chat/completions.
import http from 'http';

export function startStubLLM(patch) {
  let calls = 0;
  const server = http.createServer((req, res) => {
    let body = '';
    req.on('data', (c) => (body += c));
    req.on('end', () => {
      calls++;
      const toolCall =
        calls === 1
          ? { id: 't1', type: 'function', function: { name: 'patch_note', arguments: JSON.stringify(patch) } }
          : { id: 't2', type: 'function', function: { name: 'finish', arguments: JSON.stringify({ answer: 'triaged' }) } };
      res.setHeader('Content-Type', 'application/json');
      res.end(
        JSON.stringify({
          id: 'cmpl', object: 'chat.completion', model: 'stub',
          choices: [{ index: 0, message: { role: 'assistant', content: '', tool_calls: [toolCall] }, finish_reason: 'tool_calls' }],
          usage: { prompt_tokens: 10, completion_tokens: 5, total_tokens: 15 },
        }),
      );
    });
  });
  return new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => resolve({ server, port: server.address().port, calls: () => calls }));
  });
}
```

   Create `e2e/fleet-kanban.spec.js`:
```js
// @ts-check
import { test, expect } from '@playwright/test';
import { spawn } from 'child_process';
import path from 'path';
import { fileURLToPath } from 'url';
import { signInAsAdmin } from './helpers/auth.js';
import { startStubLLM } from './helpers/stub-llm.js';

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const GRAPHQL_URL = `${APP_URL}/_system/graphql`;
const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

const BOARD_PATH = 'boards/sprint.md';
const ROLE_PATH = 'roles/triage.md';

async function gqlAdmin(request, cookie, query, variables = {}) {
  const r = await request.post(GRAPHQL_URL, { headers: { 'Content-Type': 'application/json', Cookie: cookie }, data: { query, variables } });
  const j = await r.json();
  if (j.errors) throw new Error(JSON.stringify(j.errors));
  return j.data;
}
async function gqlApi(request, apiKey, query, variables = {}) {
  const r = await request.post(GRAPHQL_URL, { headers: { 'Content-Type': 'application/json', 'X-Api-Key': apiKey }, data: { query, variables } });
  const j = await r.json();
  if (j.errors) throw new Error(JSON.stringify(j.errors));
  return j.data;
}
async function waitJobs(request) {
  const r = await request.get(`${APP_URL}/debug/wait_all_jobs`, { timeout: 60000 });
  expect(r.ok()).toBeTruthy();
}

test.describe.serial('Fleet kanban vertical slice', () => {
  let cookie, apiKey, stub, fleetProc;

  test.beforeAll(async ({ browser, request }) => {
    cookie = await signInAsAdmin(browser);

    // Full-admin API key with MCP admin tools enabled (admin lane for the fleet).
    const keyData = await gqlAdmin(request, cookie, `
      mutation { createApiKey(input: { name: "fleet-e2e", enableMcpAdminTools: true }) {
        ... on CreateApiKeyPayload { apiKey { key } } } }`);
    apiKey = keyData.createApiKey.apiKey.key;

    // Seed the role note and the board (find/replace-friendly card line).
    const board = '---\nlayout: kanban\n---\n\n## Todo\n- Fix login bug @status:todo\n';
    const role = [
      '---',
      'model: stub',
      'tools: [search, read_note, patch_note]',
      'read_patterns: ["boards/**","roles/**"]',
      'write_patterns: ["boards/**"]',
      'max_tokens: 4000',
      'max_steps: 6',
      'mode: change',
      'trigger_include: ["boards/sprint.md"]',
      'trigger_on: [update]',
      'attach_notes: ["boards/**","roles/**"]',
      'max_depth: 1',
      'concurrency: skip',
      '---',
      'Append " @triaged" to the line of any card whose status is doing.',
    ].join('\n');
    await gqlApi(request, apiKey, `
      mutation Up($input: UpdateNotesInput!) { updateNotes(input: $input) { ... on UpdateNotesSuccessPayload { paths } ... on ErrorPayload { message } } }`,
      { input: { changes: [{ upsert: { path: BOARD_PATH, content: board } }, { upsert: { path: ROLE_PATH, content: role } }] } });
    await waitJobs(request);

    // Deterministic LLM: patch the board card found→replaced.
    stub = await startStubLLM({ path: BOARD_PATH, find: '@status:doing', replace: '@status:doing @triaged' });

    // Start the fleet against DevMode trip2g. Callback uses 127.0.0.1 so the
    // DevMode SSRF bypass allows the loopback delivery.
    fleetProc = spawn('go', ['run', './cmd/fleet',
      '-fleet-id', 'e2e',
      '-listen', '127.0.0.1:9099',
      '-callback-url', 'http://127.0.0.1:9099',
      '-trip2g-url', APP_URL,
      '-admin-api-key', apiKey,
      '-fleet-secret', 'e2e-secret',
      '-llm-base-url', `http://127.0.0.1:${stub.port}/v1`,
      '-llm-api-key', 'x',
      '-default-model', 'stub',
      '-agents-folder', 'roles/',
      '-poll-seconds', '2',
    ], { cwd: REPO_ROOT, stdio: 'inherit' });

    // Wait for the fleet to reconcile its change-webhook.
    await expect.poll(async () => {
      const d = await gqlAdmin(request, cookie, `query { allChangeWebhooks { nodes { description } } }`);
      return d.allChangeWebhooks.nodes.some((n) => n.description.startsWith('fleet:e2e:roles/triage.md#'));
    }, { timeout: 30000, intervals: [1000] }).toBeTruthy();
  });

  test.afterAll(async () => {
    if (fleetProc) fleetProc.kill('SIGTERM');
    if (stub) stub.server.close();
  });

  test('drag → agent triage → exactly one delivery, two versions, no re-trigger', async ({ request }) => {
    // Simulate the card drag: the board UI saves via updateNotes patch.
    await gqlApi(request, apiKey, `
      mutation Up($input: UpdateNotesInput!) { updateNotes(input: $input) { ... on UpdateNotesSuccessPayload { paths } ... on ErrorPayload { message } } }`,
      { input: { changes: [{ patch: { path: BOARD_PATH, find: '@status:todo', replace: '@status:doing' } }] } });
    await waitJobs(request);

    // The fleet agent fires once; its patch is depth=1 and must NOT re-trigger.
    await expect.poll(async () => {
      const d = await gqlApi(request, apiKey, `
        query NP($p:[String!]) { notePaths(filter:{paths:$p}) { content } }`, { p: [BOARD_PATH] });
      return d.notePaths[0]?.content || '';
    }, { timeout: 30000, intervals: [1000] }).toContain('@status:doing @triaged');

    // Exactly one delivery for the fleet webhook (no third run).
    const wh = await gqlAdmin(request, cookie, `query { allChangeWebhooks { nodes { id description } } }`);
    const fleetWebhook = wh.allChangeWebhooks.nodes.find((n) => n.description.startsWith('fleet:e2e:roles/triage.md#'));
    expect(fleetWebhook).toBeTruthy();
    const deliveries = await gqlAdmin(request, cookie, `
      query D($f: AdminChangeWebhookDeliveriesFilterInput!) { changeWebhookDeliveries(filter: $f) { nodes { id status } } }`,
      { f: { webhookId: fleetWebhook.id } });
    expect(deliveries.changeWebhookDeliveries.nodes.length).toBe(1);

    // Board content reflects the agent decision (re-render oracle).
    const final = await gqlApi(request, apiKey, `query NP($p:[String!]) { notePaths(filter:{paths:$p}) { content } }`, { p: [BOARD_PATH] });
    expect(final.notePaths[0].content).toContain('@status:doing @triaged');
  });
});
```

2. [ ] **Run it to verify it fails.** First start the test env (the e2e harness), then run only this spec:
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && ./scripts/start-test-env.sh & sleep 20 ; APP_URL=http://localhost:20081 npx playwright test e2e/fleet-kanban.spec.js 2>&1 | tail -30
```
Expected (red): the spec exists but the assertions fail until the fleet pieces (C1–C8) are wired end to end and any missing GraphQL admin field (`createApiKey enableMcpAdminTools`, `changeWebhookDeliveries.webhookId`) is confirmed. If a query field name differs, fix the query string only (do not weaken the assertions).

> Verify two field names against `internal/graph/schema.graphqls` before first green: the `createApiKey` mutation + `enableMcpAdminTools` input flag (admin key for the fleet), and `AdminChangeWebhookDeliveriesFilterInput.webhookId`. Adjust the query strings to match exactly; the test logic stays as written.

3. [ ] **Run to verify pass.**
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && APP_URL=http://localhost:20081 npx playwright test e2e/fleet-kanban.spec.js 2>&1 | tail -30
```
Expected: `1 passed` — board contains `@status:doing @triaged`, deliveries length `1`, stub LLM invoked exactly twice (one patch step, one finish), no second delivery (depth=1 ≥ max_depth=1).

4. [ ] **Commit.**
```
git add e2e/fleet-kanban.spec.js e2e/helpers/stub-llm.js && git commit -m "test(e2e): kanban fleet vertical slice exactly-one-delivery"
```

---

### Task C10 — Docs: fleet dev reference, bilingual user pair, agents catalog status (Step 10)

Document the shipped fleet. The canonical design (`docs/dev/agent_runtime_design.md`) already exists; add a concise operator-facing dev doc, the bilingual user pair, and a fleet entry + status line in the agents catalog (`docs/dev/agents.md` is the real committed file; the contract's `docs/dev/agent.md` refers to this catalog). Answer-first per project style (each doc opens with a short TL;DR).

**Files:**
- Create: `docs/dev/fleet.md`
- Create: `docs/en/user/agent-fleet.md`
- Create: `docs/ru/user/agent-fleet.md`
- Modify: `docs/dev/agents.md` (add fleet entry + mark agent-runtime status)

**Steps:**

1. [ ] **No automated test for docs.** Verification is a content review + a link/build check. First confirm the target files referenced exist (so cross-links don't dangle):
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && ls docs/dev/agent_runtime_design.md docs/dev/agents.md docs/dev/webhook_bots.md && ls cmd/fleet/main.go internal/fleet/
```
Expected: all exist (cross-link anchors valid).

2. [ ] **Write `docs/dev/fleet.md`** — operator/dev reference. Lead with a TL;DR, then: what the fleet is (fleet-as-executor), the two auth lanes (admin `/_system/mcp` X-Api-Key vs scoped `/_system/graphql` Bearer), role-note frontmatter schema (the flat keys from the spec), `cmd/fleet` flags/env (mirror `parseFlags` in `cmd/fleet/main.go`), the reconcile marker `fleet:<FleetID>:<path>#<ver>`, the budget ceiling (`min(frontmatter, ceiling)`), HMAC secret derivation, and the security posture table (link to `docs/dev/agent_runtime_design.md` §7 for the full model). Include the kanban worked example and the `patch_note` write-back. Keep it ≤ 200 lines; defer rationale to the design doc.

3. [ ] **Write the bilingual user pair** `docs/en/user/agent-fleet.md` + `docs/ru/user/agent-fleet.md` — same structure, mirrored content (per the CLAUDE.md bilingual rule). Audience = a self-hoster wiring an agent: (1) write a role note with frontmatter, (2) run `cmd/fleet` with their LLM + admin key, (3) the fleet auto-registers webhooks, (4) edits to matching notes drive the agent. Include the `roles/triage.md` example verbatim from the design doc and a "what's enforced for you" safety paragraph (scope, budget ceiling, depth guard). Each file opens with a one-line вывод/TL;DR.

4. [ ] **Update `docs/dev/agents.md`.** Add a fleet entry under `## Агенты` and a status note that the in-process agent-runtime executor now ships out-of-process as the fleet:
```markdown
### [Agent Fleet](fleet.md) — `cmd/fleet`

The fleet-as-executor host: discovers role notes, reconciles webhooks to itself,
runs the scoped agent loop (`internal/agentruntime`), and writes back via a
per-delivery scoped token. trip2g stays a dumb event source. Status: shipped
(kanban vertical slice green). Design: [agent_runtime_design.md](agent_runtime_design.md).
```

5. [ ] **Verify pass (review + sync).** Re-read each doc for answer-first lead and accurate flag/field names against the code; then sync the demo vault if the user docs reference demo notes:
```
cd /home/alexes/projects2/trip2g/.worktrees/feat-agent-runtime && grep -n "TL;DR\|Вывод\|вывод" docs/dev/fleet.md docs/en/user/agent-fleet.md docs/ru/user/agent-fleet.md
```
Expected: each file has a leading TL;DR/вывод line; flag names match `cmd/fleet/main.go`.

6. [ ] **Commit.**
```
git add docs/dev/fleet.md docs/en/user/agent-fleet.md docs/ru/user/agent-fleet.md docs/dev/agents.md && git commit -m "docs(fleet): agent runtime reference and bilingual user guide"
```

---

**Section C exit criteria:** `go test ./internal/agentruntime/ ./internal/webhookutil/ ./internal/fleet/` green; `go build ./cmd/fleet ./cmd/agent` clean; `go vet ./cmd/fleet ./internal/fleet` clean; `npx playwright test e2e/fleet-kanban.spec.js` passes (exactly one delivery, board shows `@status:doing @triaged`); docs reviewed answer-first.


---

