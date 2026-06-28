package fleet

import (
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
	Depth         int            `json:"depth"`
	Instruction   string         `json:"instruction"`
	APIToken      string         `json:"api_token"`
	AttachedNotes []attachedNote `json:"attached_notes"`
}

type attachedNote struct {
	Path    string `json:"path"`
	Content string `json:"content"`
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

	res, runErr := agentruntime.Run(r.Context(), agentruntime.Input{
		Instruction:   role.Body,
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
		writeJSON(w, http.StatusBadGateway, webhookutil.AgentResponse{Status: "error", Message: runErr.Error()})
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
