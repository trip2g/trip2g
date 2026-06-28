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
