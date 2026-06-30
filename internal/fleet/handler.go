package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"

	"trip2g/internal/agentruntime"
	"trip2g/internal/webhookutil"
)

// cronDeliveryPayload mirrors trip2g's cron-webhook delivery body
// (delivercronwebhook.cronWebhookPayload). Only the fields the fleet needs are
// decoded; instruction, response_schema, secrets, and previous_error are owned
// by trip2g's retry logic and are not forwarded to the role body.
type cronDeliveryPayload struct {
	Depth         int            `json:"depth"`
	APIToken      string         `json:"api_token"`
	AttachedNotes []attachedNote `json:"attached_notes"`
}

// deliveryPayload mirrors deliverchangewebhook.changeWebhookPayload plus the
// Section-B attached_notes field. api_token is the per-delivery scoped token.
type deliveryPayload struct {
	Depth         int            `json:"depth"`
	Instruction   string         `json:"instruction"`
	APIToken      string         `json:"api_token"`
	Changes       []changeInfo   `json:"changes"`
	AttachedNotes []attachedNote `json:"attached_notes"`
}

// changeInfo mirrors handlenotewebhooks.ChangeInfo — the per-note trigger data
// trip2g sends in the delivery payload's changes[] array.
type changeInfo struct {
	Path    string `json:"path"`
	Event   string `json:"event"`
	PathID  int64  `json:"path_id"`
	Version int64  `json:"version"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// attachedNote mirrors webhookutil.AttachedNote — a context note materialized
// by trip2g via attach_notes. Meta is the key-allowlist trip2g sends (not the
// full RawMeta).
type attachedNote struct {
	Path      string            `json:"path"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	UpdatedAt string            `json:"updated_at"`
	Tags      []string          `json:"tags"`
	Meta      map[string]string `json:"meta"`
}

// maxBodyBytes caps the delivery payload size to guard against DoS.
const maxBodyBytes = 10 * 1024 * 1024 // 10 MiB

// inputKeyDepth is the JSON key for the depth field in the $FLEET_INPUT bag.
const inputKeyDepth = "depth"

// statusError is the AgentResponse status value for hard-failure responses.
const statusError = "error"

// ServeDelivery handles POST /deliver/<urlKey> (change deliveries) and
// POST /deliver/cron/<urlKey> (cron-triggered deliveries).
func (f *Fleet) ServeDelivery(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/deliver/")
	isCron := strings.HasPrefix(rawPath, "cron/")
	key := rawPath
	if isCron {
		key = strings.TrimPrefix(rawPath, "cron/")
	}

	role, ok := f.roleByKey(key)
	if !ok {
		http.Error(w, "unknown delivery key", http.StatusNotFound)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	// Select the HMAC secret that matches this delivery type.
	secret := f.secretFor(role)
	if isCron {
		secret = f.cronSecretFor(role)
	}
	if !webhookutil.VerifyHMAC(body, secret, r.Header.Get("X-Webhook-Signature")) {
		http.Error(w, "bad signature", http.StatusUnauthorized)
		return
	}

	if isCron {
		f.serveCronDelivery(w, r, role, body)
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

	base := renderCtx{
		ChangedFiles:  payload.Changes,
		AttachedNotes: payload.AttachedNotes,
		Depth:         payload.Depth,
	}

	// Zero-item fan-out (e.g. for_each:attached_notes with no attached notes, or
	// for_each:changed_files with an empty changes[]) is a no-op, not a failure.
	// Returning 200 stops trip2g from retrying the empty batch to exhaustion and
	// marking the delivery failed.
	items := fanOut(role.ForEach, base)
	if len(items) == 0 {
		writeJSON(w, http.StatusOK, webhookutil.AgentResponse{
			Status:     agentruntime.StatusCompleted,
			Message:    "no " + role.ForEach + " items to process",
			TokensUsed: 0,
			Steps:      0,
		})
		return
	}

	// Detach the agent run from the request context. trip2g closes the delivery
	// connection when its change-webhook timeoutSeconds elapses, but an LLM agent
	// run can outlive that. Tying Run to r.Context() would let a premature
	// client/connection close cancel the run mid-flight and lose the write-back.
	// Instead, bound the run by the role's own timeout off a fresh background
	// context. No request-scoped values are needed here (the scoped api_token
	// rides in the payload, not the context), so nothing is carried over.
	runCtx, cancel := context.WithTimeout(context.Background(),
		time.Duration(role.EffectiveTimeoutSeconds())*time.Second)
	defer cancel()

	// Sequential fan-out, continue-on-error: one Run per item, accumulate spend,
	// collect per-item errors instead of aborting the batch. Each Run reuses the
	// same scoped api_token, so per-delivery attribution (all items reuse the one
	// scoped delivery token) is preserved across the batch.
	var totalTokens, totalSteps, successCount int
	var aggStatus string
	var answers, errMsgs []string
	for i, rc := range items {
		instruction, rerr := renderInstruction(role.Body, rc)
		if rerr != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("item %d: render: %v", i+1, rerr))
			continue
		}
		// Code executor: body already rendered to a program; run it deterministically.
		// The run context (runCtx) already carries the role timeout — Timeout: 0
		// means "use ctx only" so we don't double-apply the deadline.
		res, runErr := f.execRole(execRoleInput{
			Ctx:      runCtx,
			Role:     role,
			Instr:    instruction,
			GQL:      NewScopedGraphQLClient(f.cfg.Trip2gBaseURL, payload.APIToken, f.hc),
			Overlay:  overlay,
			InputBag: buildInputBag(rc),
		})
		if runErr != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("item %d: %v", i+1, runErr))
			continue
		}
		totalTokens += res.TokensUsed
		totalSteps += res.Steps
		successCount++
		aggStatus = mergeStatus(aggStatus, res.Status)
		if res.Answer != "" {
			answers = append(answers, res.Answer)
		}
	}

	// Whole batch failed: surface a non-2xx so trip2g's retry/backoff engages.
	if successCount == 0 {
		writeJSON(w, http.StatusBadGateway, webhookutil.AgentResponse{
			Status:  statusError,
			Message: strings.Join(errMsgs, "; "),
		})
		return
	}

	status := aggStatus
	message := strings.Join(answers, "\n")
	if len(errMsgs) > 0 {
		status = "partial"
		if message != "" {
			message += "\n"
		}
		message += "errors: " + strings.Join(errMsgs, "; ")
		// trip2g ignores the status field and records partials as success, so a
		// permanent per-item failure is otherwise invisible. Log it warn-level so
		// partials are discoverable from the fleet side.
		//nolint:sloglint // Fleet has no logger instance; global slog is intentional here
		slog.WarnContext(r.Context(), "fleet: partial fan-out", "role", role.NotePath,
			"failed", len(errMsgs), "total", len(items), "errors", strings.Join(errMsgs, "; "))
	}

	// Changes already applied in-loop via the scoped token; report spend only.
	writeJSON(w, http.StatusOK, webhookutil.AgentResponse{
		Status:     status,
		Message:    message,
		Changes:    nil,
		TokensUsed: totalTokens,
		Steps:      totalSteps,
	})
}

// buildInputBag marshals the trigger render context into the JSON bag delivered
// to code programs via $FLEET_INPUT. It carries only non-secret trigger data
// (the same fields exposed to Jet templates). The scoped write token is never
// included.
func buildInputBag(rc renderCtx) []byte {
	bag := map[string]any{
		forEachChangedFiles:  rc.ChangedFiles,
		"change_file":        rc.ChangeFile,
		forEachAttachedNotes: rc.AttachedNotes,
		inputKeyDepth:        rc.Depth,
	}
	data, _ := json.Marshal(bag)
	return data
}

// fanOut expands the base render context into one context per for_each item,
// sequentially. Empty for_each yields a single context (change_file=nil, full
// changed_files/attached_notes lists). for_each:attached_notes scopes the
// attached_notes var to the current note (the var bag has no singular note slot,
// so the current item is exposed as a one-element attached_notes list).
func fanOut(mode string, base renderCtx) []renderCtx {
	switch mode {
	case forEachChangedFiles:
		out := make([]renderCtx, 0, len(base.ChangedFiles))
		for i := range base.ChangedFiles {
			rc := base
			rc.ChangeFile = &base.ChangedFiles[i]
			out = append(out, rc)
		}
		return out
	case forEachAttachedNotes:
		out := make([]renderCtx, 0, len(base.AttachedNotes))
		for i := range base.AttachedNotes {
			rc := base
			rc.AttachedNotes = base.AttachedNotes[i : i+1]
			out = append(out, rc)
		}
		return out
	default:
		return []renderCtx{base}
	}
}

// mergeStatus folds a fan-out item's run status into the batch aggregate,
// preferring a cap status (capped/max_steps) over completed so a capped item is
// never hidden by a later completed one.
func mergeStatus(agg, next string) string {
	if statusSeverity(next) >= statusSeverity(agg) {
		return next
	}
	return agg
}

// statusSeverity ranks run statuses so the most severe wins in mergeStatus.
func statusSeverity(status string) int {
	switch status {
	case agentruntime.StatusCapped:
		return 2
	case agentruntime.StatusMaxSteps:
		return 1
	default: // completed / empty
		return 0
	}
}

// serveCronDelivery handles a cron-triggered POST /deliver/cron/<key>.
// Unlike change delivery, there are no changed_files; the role body is rendered
// once with an empty change context and the wall-clock `now` variable.
func (f *Fleet) serveCronDelivery(w http.ResponseWriter, r *http.Request, role Role, body []byte) {
	var payload cronDeliveryPayload
	if uerr := json.Unmarshal(body, &payload); uerr != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	overlay := make(map[string]string, len(payload.AttachedNotes))
	for _, n := range payload.AttachedNotes {
		overlay[n.Path] = n.Content
	}

	rc := renderCtx{
		AttachedNotes: payload.AttachedNotes,
		Depth:         payload.Depth,
		Now:           time.Now(),
	}

	instruction, rerr := renderInstruction(role.Body, rc)
	if rerr != nil {
		writeJSON(w, http.StatusBadRequest, webhookutil.AgentResponse{
			Status:  statusError,
			Message: fmt.Sprintf("render: %v", rerr),
		})
		return
	}

	// Detach the run from the request context (same rationale as change delivery).
	runCtx, cancel := context.WithTimeout(context.Background(),
		time.Duration(role.EffectiveTimeoutSeconds())*time.Second)
	defer cancel()

	res, runErr := f.execRole(execRoleInput{
		Ctx:      runCtx,
		Role:     role,
		Instr:    instruction,
		GQL:      NewScopedGraphQLClient(f.cfg.Trip2gBaseURL, payload.APIToken, f.hc),
		Overlay:  overlay,
		InputBag: buildCronInputBag(rc),
	})
	if runErr != nil {
		//nolint:sloglint // Fleet has no logger instance; global slog is intentional here
		slog.WarnContext(r.Context(), "fleet: cron run error", "role", role.NotePath, "error", runErr)
		writeJSON(w, http.StatusBadGateway, webhookutil.AgentResponse{
			Status:  statusError,
			Message: runErr.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, webhookutil.AgentResponse{
		Status:     res.Status,
		Message:    res.Answer,
		TokensUsed: res.TokensUsed,
		Steps:      res.Steps,
	})
}

// buildCronInputBag marshals the cron render context into the JSON bag delivered
// to code programs via $FLEET_INPUT. Exposes now as an RFC3339 string.
func buildCronInputBag(rc renderCtx) []byte {
	bag := map[string]any{
		forEachAttachedNotes: rc.AttachedNotes,
		inputKeyDepth:        rc.Depth,
		"now":                rc.Now.Format(time.RFC3339),
	}
	data, _ := json.Marshal(bag)
	return data
}

// execRoleInput bundles the parameters for a single role execution dispatch.
type execRoleInput struct {
	Ctx      context.Context
	Role     Role
	Instr    string
	GQL      graphql.Client // scoped genqlient client for the delivery's KB lane
	Overlay  map[string]string
	InputBag []byte // JSON bag for code executor ($FLEET_INPUT); nil for LLM executor
}

// execRole dispatches a single rendered instruction through the role's configured
// executor (LLM agent run or deterministic code runner). Called from both change
// delivery (with buildInputBag) and cron delivery (with buildCronInputBag).
func (f *Fleet) execRole(p execRoleInput) (*agentruntime.Result, error) {
	kb := newRemoteKB(p.GQL, p.Overlay)
	if p.Role.Executor == executorCode {
		return f.codeRunner(p.Ctx, agentruntime.CodeInput{
			Body:            p.Instr,
			WritePatterns:   p.Role.WritePatterns,
			KB:              kb,
			AllowedPrograms: f.cfg.AllowedPrograms,
			Input:           p.InputBag,
			EnvPassthrough:  p.Role.EnvPassthrough,
			EnvPrefix:       p.Role.EnvPrefix,
			MaxStdoutBytes:  f.cfg.MaxStdoutBytes,
		})
	}
	return agentruntime.Run(p.Ctx, agentruntime.Input{
		Instruction:     p.Instr,
		ReadPatterns:    p.Role.ReadPatterns,
		WritePatterns:   p.Role.WritePatterns,
		Tools:           p.Role.Tools,
		Model:           orDefault(p.Role.Model, f.cfg.DefaultModel),
		MaxTokens:       clampBudget(p.Role.MaxTokens, f.cfg.TokenCeiling),
		MaxSteps:        clampBudget(p.Role.MaxSteps, f.cfg.StepCeiling),
		AllowedPrograms: f.cfg.AllowedPrograms,
		LLM:             f.llm,
		KB:              kb,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
