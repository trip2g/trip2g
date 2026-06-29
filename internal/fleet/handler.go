package fleet

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"trip2g/internal/agentruntime"
	"trip2g/internal/webhookutil"
)

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

// ServeDelivery handles POST /deliver/<urlKey>.
func (f *Fleet) ServeDelivery(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/deliver/")
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
		res, runErr := agentruntime.Run(r.Context(), agentruntime.Input{
			Instruction:   instruction,
			ReadPatterns:  role.ReadPatterns,
			WritePatterns: role.WritePatterns,
			Tools:         role.Tools,
			Model:         orDefault(role.Model, f.cfg.DefaultModel),
			MaxTokens:     clampBudget(role.MaxTokens, f.cfg.TokenCeiling),
			MaxSteps:      clampBudget(role.MaxSteps, f.cfg.StepCeiling),
			LLM:           f.llm,
			KB:            newRemoteKB(f.client, payload.APIToken, overlay),
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
			Status:  "error",
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

// fanOut expands the base render context into one context per for_each item,
// sequentially. Empty for_each yields a single context (change_file=nil, full
// changed_files/attached_notes lists). for_each:attached_notes scopes the
// attached_notes var to the current note (the var bag has no singular note slot,
// so the current item is exposed as a one-element attached_notes list).
func fanOut(mode string, base renderCtx) []renderCtx {
	switch mode {
	case "changed_files":
		out := make([]renderCtx, 0, len(base.ChangedFiles))
		for i := range base.ChangedFiles {
			rc := base
			rc.ChangeFile = &base.ChangedFiles[i]
			out = append(out, rc)
		}
		return out
	case "attached_notes":
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

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
