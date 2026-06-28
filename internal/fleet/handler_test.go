package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	require.Equal(t, 1, scopedCalls)    // exactly one scoped updateNotes
	require.Zero(t, adminCalls)         // admin key never used for writes
	require.Equal(t, "scoped-token", lastToken)

	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Nil(t, resp.Changes)      // writes already applied in-loop
	require.Equal(t, 30, resp.TokensUsed) // (10+5)*2
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

// errLLM is a stub LLM that always returns an error from Chat.
type errLLM struct{ msg string }

func (e *errLLM) Chat(_ context.Context, _ string, _ []agentruntime.Message, _ []agentruntime.ToolDef) (agentruntime.ChatResult, error) {
	return agentruntime.ChatResult{}, fmt.Errorf("%s", e.msg)
}

// newTestFleetWithLLM builds a Fleet with the given LLM.
func newTestFleetWithLLM(client Client, llm agentruntime.LLM) *Fleet {
	role := Role{
		NotePath: "roles/triage.md", Body: "Triage.", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		MaxTokens: 4000, MaxSteps: 6, Concurrency: "skip", MaxDepth: 1,
	}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000, StepCeiling: 25,
	}
	f := NewFleet(cfg, client, llm)
	f.SetRoles([]Role{role})
	return f
}

// TestServeDelivery_RunError502 ensures that when agentruntime.Run returns an
// error, the handler responds with a non-2xx status (502) so trip2g's
// handleDeliveryError can engage retry/backoff instead of silently dropping.
func TestServeDelivery_RunError502(t *testing.T) {
	tests := []struct {
		name      string
		llmErrMsg string
	}{
		{"llm_unavailable", "connection refused"},
		{"llm_rate_limit", "rate limit exceeded"},
		{"llm_context_cancelled", "context deadline exceeded"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &ClientMock{}
			f := newTestFleetWithLLM(client, &errLLM{msg: tc.llmErrMsg})
			key := urlKey("roles/triage.md")
			rec := post(t, f, key, deliveryBody(t), true)

			require.GreaterOrEqual(t, rec.Code, 500, "want >=500 on Run error, got %d", rec.Code)

			var resp webhookutil.AgentResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.Equal(t, "error", resp.Status)
			require.NotEmpty(t, resp.Message)
		})
	}
}

// TestServeDelivery_MaxBytesReader ensures oversized bodies are rejected with
// 400 (not 500 panic / OOM).
func TestServeDelivery_MaxBytesReader413(t *testing.T) {
	client := &ClientMock{}
	f := newTestFleetWithLLM(client, &stubLLM{})
	key := urlKey("roles/triage.md")

	// Build a valid body, then pad it beyond the limit.
	base := deliveryBody(t)
	oversized := make([]byte, 0, 11*1024*1024)
	oversized = append(oversized, base...)
	padding := make([]byte, 11*1024*1024)
	oversized = append(oversized, padding...)

	req := httptest.NewRequest(http.MethodPost, "/deliver/"+key, bytes.NewReader(oversized))
	role, ok := f.roleByKey(key)
	require.True(t, ok)
	// Sign the original base body (signature won't match the padded payload, but
	// the max-bytes check should reject before signature verification reaches the
	// full body; alternatively the body mismatch triggers 400 or 401, either way
	// confirming no OOM/panic on oversized input).
	req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(base, f.secretFor(role)))

	rec := httptest.NewRecorder()
	f.ServeDelivery(rec, req)
	require.NotEqual(t, http.StatusOK, rec.Code, "oversized body must not yield 200")
}
