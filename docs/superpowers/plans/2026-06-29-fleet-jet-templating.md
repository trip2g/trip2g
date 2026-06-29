# Fleet #13 (Jet role-note templating + for_each) + runnable-kit — Implementation Plan

> **For agentic workers:** TDD task-by-task. Steps use `- [ ]`. Work in worktree `feat/agent-runtime`. Do NOT push (PR #53 updated separately). cmd/server can't build here (missing embed assets) — build `./cmd/fleet/... ./internal/fleet/... ./internal/agentruntime/...` and run their tests.

**Goal:** Make the fleet a general "notes-trigger-agents" runtime: the role-note body becomes a Jet template rendered with the trigger context, with `for_each` fan-out — and ship a runnable kit (example role/board + internal run guide + working demo e2e).

**Architecture:** Templating hooks in at ONE seam — `internal/fleet/handler.go`, between payload decode and `agentruntime.Run`. `Instruction: role.Body` → `Instruction: render(role.Body, ctx)`. Reuse the EXISTING `CloudyKit/jet/v6` (already a dep; pattern in `internal/layoutloader/loader.go:439`). trip2g needs NO change to its delivery payload (it already sends `changes[]` + `attached_notes`); only the fleet's structs widen to receive them. One config wrinkle: the fleet reconciler must request note content.

## Locked decisions (approved 2026-06-29)
- **Var bag exposed to the template:** `changed_files` (list), `change_file` (current item under for_each), `attached_notes` (list), `depth` (int). Each file/note exposes `Path, Event, Title, Content, Tags, Meta, UpdatedAt` (whatever trip2g sends). **NEVER exposed:** `secrets`, `api_token`, the raw scoped token.
- **for_each:** enum `{"", "changed_files", "attached_notes"}`. Empty = legacy single run. Non-empty = one `agentruntime.Run` per item, **sequential**.
- **Aggregation:** sum `tokens`/`steps` across runs; **continue-on-error** (collect per-item errors, don't abort the batch); per-item delivery attribution preserved.
- **Docs:** INTERNAL only (docs/dev). No public/user docs, no announcement.

---

### Task 1: Widen the delivery payload to receive trigger context

**Files:** Modify `internal/fleet/handler.go` (the `deliveryPayload`/`attachedNote` structs, ~lines 15-25); Test `internal/fleet/handler_test.go`.

- [ ] **Step 1 — failing test:** post a delivery payload JSON containing a `changes` array (`[{path,event,path_id,version,title,content}]`) and an `attached_notes` entry with `title/tags/meta/updated_at`; assert the handler decodes them into the widened structs (assert `len(changes)==N` and the metadata fields are populated). Run, confirm FAIL (fields don't exist).
- [ ] **Step 2 — implement:** add `Changes []changeInfo` to `deliveryPayload`; define `changeInfo{Path, Event string; PathID int64; Version int64; Title, Content string}` matching trip2g's `ChangeInfo` (see `internal/case/backjob/deliverchangewebhook/resolve.go:39`). Enrich `attachedNote` with `Title string; Tags []string; Meta map[string]any; UpdatedAt string` (match what trip2g sends — verify against the delivery resolver). Keep existing fields.
- [ ] **Step 3 — pass + commit:** `go test ./internal/fleet/...`; commit `feat(fleet): receive changes[] + attached-note metadata in delivery payload`.

### Task 2: Reconciler requests note content

**Files:** Modify the fleet reconciler that creates/updates the change webhook (grep `internal/fleet` for where it sets webhook fields / `include_content`); Test alongside.

- [ ] **Step 1 — verify the gap:** confirm the reconciler does not set `include_content` (so `changes[].content` arrives empty). If trip2g gates content on a webhook flag, set it true; if content is always sent, note that and skip.
- [ ] **Step 2 — failing test then implement:** assert the reconciled webhook payload/spec has content enabled. Implement the flag.
- [ ] **Step 3 — commit:** `fix(fleet): request note content in reconciled change webhook`.

### Task 3: Add `ForEach` to role parsing + validation

**Files:** Modify `internal/fleet/role.go` (`ParseRole`/`Validate`); Test `internal/fleet/role_test.go`.

- [ ] **Step 1 — failing test:** table test — `for_each: changed_files` and `for_each: attached_notes` parse into `role.ForEach`; an invalid value (`for_each: bogus`) fails validation; absent = `""` (legacy).
- [ ] **Step 2 — implement:** add `ForEach string` to the role struct + frontmatter key `for_each`; validate against the enum in `Validate`.
- [ ] **Step 3 — commit:** `feat(fleet): role for_each fan-out mode (changed_files|attached_notes)`.

### Task 4: Render step (Jet)

**Files:** Create `internal/fleet/render.go`; Test `internal/fleet/render_test.go`.

- [ ] **Step 1 — failing test:** `renderInstruction(body string, ctx renderCtx) (string, error)` — a template referencing `{{ change_file.Path }}` and `{{ range changed_files }}{{ .Title }}{{ end }}` renders the expected string; a template referencing a non-exposed var (e.g. `secrets`) renders empty/errors (assert no secret leakage). Run, FAIL.
- [ ] **Step 2 — implement:** build a `renderCtx` struct `{ChangedFiles []changeInfo; ChangeFile *changeInfo; AttachedNotes []attachedNote; Depth int}`. Use `jet/v6` (mirror `internal/layoutloader/loader.go:439`): an in-memory loader, set vars via `VarMap` (`changed_files`, `change_file`, `attached_notes`, `depth`). Do NOT register any secret/token var. Return the rendered string.
- [ ] **Step 3 — pass + commit:** `feat(fleet): jet render of role body with trigger-context bag (no secrets)`.

### Task 5: Wire render + fan-out into the handler

**Files:** Modify `internal/fleet/handler.go` (~line 60-70, the `agentruntime.Run` call); Test `internal/fleet/handler_test.go`.

- [ ] **Step 1 — failing test:** with a stub LLM/KB, post a payload with 2 changes + a role whose `for_each: changed_files`; assert `agentruntime.Run` is invoked twice (once per change), each with an instruction rendered for that `change_file`. Also a no-for_each case: 1 run with all changes in `changed_files`. Run, FAIL.
- [ ] **Step 2 — implement:** replace `Instruction: role.Body` with rendered instruction. If `role.ForEach == ""`: render once (full `changed_files`/`attached_notes` in ctx, `change_file=nil`), one Run. Else: pick the collection, loop sequentially, set `change_file` per item, render, Run per item.
- [ ] **Step 3 — pass + commit:** `feat(fleet): render instruction + sequential for_each fan-out at handler seam`.

### Task 6: Aggregate N run results

**Files:** Modify `internal/fleet/handler.go` + the response type it returns (grep `AgentResponse`); Test alongside.

- [ ] **Step 1 — failing test:** 2-item fan-out where item 2's Run errors — assert the batch continues, the aggregate response sums tokens/steps of the successful run(s), and per-item errors are reported (not a hard 500). Run, FAIL.
- [ ] **Step 2 — implement:** accumulate per-item `{tokens, steps, err}`; continue-on-error; return an aggregate that sums tokens/steps and lists per-item outcomes. Preserve per-item delivery attribution (each Run still stamps `created_by_delivery_*`).
- [ ] **Step 3 — commit:** `feat(fleet): aggregate for_each run results (sum spend, continue-on-error)`.

### Task 7: Surface attached_notes existence to the model

**Files:** Modify `internal/fleet/render.go` or the system-prompt assembly; Test alongside.

- [ ] **Step 1 — failing test:** with attached_notes present and a role body that does NOT reference them, assert the final instruction/system context still names the attached note paths (so the model knows they're available to read).
- [ ] **Step 2 — implement:** append a short "Attached notes available: <paths>" line to the rendered instruction when attached_notes is non-empty and the template didn't already include them.
- [ ] **Step 3 — commit:** `feat(fleet): tell the model which attached notes are available`.

### Task 8: Example role-note + board (runnable kit)

**Files:** Create `docs/demo/fleet/roles/triage.md`, `docs/demo/fleet/boards/sprint.md` (or align with the e2e's seeded paths).

- [ ] Author a `triage` role using the template: frontmatter (`model`, `tools: [search, patch_note]`, `read_patterns`, `write_patterns`, `mode: change`, `trigger_include`, `attach_notes`, `max_depth: 1`, `concurrency: skip`, optionally `for_each: changed_files`); body is a Jet template that references `{{ change_file.Path }}` / `{{ change_file.Content }}`. Add a board note. Commit `docs(demo): example fleet triage role + board`.

### Task 9: Internal run guide

**Files:** Create `docs/dev/fleet_run.md` (INTERNAL).

- [ ] Document running the fleet as a core feature: `cmd/fleet` flags (from `cmd/fleet/main.go parseFlags()`), DevMode requirement (loopback SSRF bypass), full-admin API key (`createApiKey` + `setApiKeyMcpAdminTools`), networking (host: `127.0.0.1:9099`; docker: `host.docker.internal:9099`), LLM endpoint flags, and the trigger→delivery→write-back walkthrough. Commit `docs(dev): internal fleet run guide`.

### Task 10: Make the demo e2e runnable

**Files:** Modify `e2e/fleet-kanban.spec.js` + `scripts/test-e2e.sh` (note: test-e2e.sh edits stay local/unstaged per owner's convention unless told).

- [ ] Fix host-networking so the spec works under the compose setup (use `host.docker.internal` callback); make the role body use the new template; optionally subscribe to `noteChanges` SSE to prove the "observed" half. Wire it into `test-e2e.sh` (or document how to run it standalone). Confirm it passes with the stub LLM. Commit the spec changes (leave test-e2e.sh per convention).

---

## Self-review checklist
- Every exposed template var is non-secret (Task 4 asserts no secret leakage).
- for_each sequential, continue-on-error, per-item attribution (Tasks 5-6).
- trip2g unchanged except content-flag request (Task 2); all hooks are fleet-side.
- Build `./cmd/fleet/... ./internal/fleet/... ./internal/agentruntime/...` green; cmd/server embed failure is expected in-worktree.
