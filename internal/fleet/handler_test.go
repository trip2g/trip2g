package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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
	require.Equal(t, 1, scopedCalls) // exactly one scoped updateNotes
	require.Zero(t, adminCalls)      // admin key never used for writes
	require.Equal(t, "scoped-token", lastToken)

	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Nil(t, resp.Changes)          // writes already applied in-loop
	require.Equal(t, 30, resp.TokensUsed) // (10+5)*2
	require.Equal(t, 2, resp.Steps)
}

// TestDeliveryPayload_DecodesTriggerContext asserts the widened delivery payload
// decodes trip2g's changes[] array and the attached-note metadata (title/tags/
// meta/updated_at) into the fleet's receiving structs.
func TestDeliveryPayload_DecodesTriggerContext(t *testing.T) {
	body, err := json.Marshal(map[string]any{
		"depth":       2,
		"instruction": "Triage.",
		"api_token":   "scoped-token",
		"changes": []map[string]any{
			{"path": "boards/sprint.md", "event": "update", "path_id": 7, "version": 42, "title": "Sprint", "content": "- a @status:todo\n"},
			{"path": "boards/backlog.md", "event": "create", "path_id": 8, "version": 1, "title": "Backlog", "content": "x"},
		},
		"attached_notes": []map[string]any{
			{"path": "roles/triage.md", "content": "role body", "title": "Triage role", "tags": []string{"role", "triage"}, "meta": map[string]string{"layout": "doc"}, "updated_at": "2026-06-29T10:00:00Z"},
		},
	})
	require.NoError(t, err)

	var p deliveryPayload
	require.NoError(t, json.Unmarshal(body, &p))

	require.Equal(t, 2, p.Depth)
	require.Len(t, p.Changes, 2)
	require.Equal(t, "boards/sprint.md", p.Changes[0].Path)
	require.Equal(t, "update", p.Changes[0].Event)
	require.Equal(t, int64(7), p.Changes[0].PathID)
	require.Equal(t, int64(42), p.Changes[0].Version)
	require.Equal(t, "Sprint", p.Changes[0].Title)
	require.Equal(t, "- a @status:todo\n", p.Changes[0].Content)

	require.Len(t, p.AttachedNotes, 1)
	an := p.AttachedNotes[0]
	require.Equal(t, "roles/triage.md", an.Path)
	require.Equal(t, "Triage role", an.Title)
	require.Equal(t, []string{"role", "triage"}, an.Tags)
	require.Equal(t, "doc", an.Meta["layout"])
	require.Equal(t, "2026-06-29T10:00:00Z", an.UpdatedAt)
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

// recordLLM finishes each run in one step and records the system prompt (which
// embeds the rendered instruction) so tests can assert per-item rendering.
type recordLLM struct{ systems []string }

func (r *recordLLM) Chat(_ context.Context, _ string, msgs []agentruntime.Message, _ []agentruntime.ToolDef) (agentruntime.ChatResult, error) {
	if len(msgs) > 0 {
		r.systems = append(r.systems, msgs[0].Content)
	}
	args, _ := json.Marshal(map[string]any{"answer": "ok"})
	return agentruntime.ChatResult{
		ToolCalls:    []agentruntime.ToolCall{{ID: "1", Name: "finish", Arguments: string(args)}},
		PromptTokens: 3, CompletionTokens: 2,
	}, nil
}

func fanOutFleet(t *testing.T, llm agentruntime.LLM, forEach, body string) *Fleet {
	t.Helper()
	role := Role{
		NotePath: "roles/triage.md", Body: body, Mode: "change", ForEach: forEach,
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		MaxTokens: 4000, MaxSteps: 6, Concurrency: "skip", MaxDepth: 1,
	}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000, StepCeiling: 25,
	}
	f := NewFleet(cfg, &ClientMock{}, llm)
	f.SetRoles([]Role{role})
	return f
}

func changesBody(t *testing.T) []byte {
	t.Helper()
	b, _ := json.Marshal(map[string]any{
		"depth": 0, "api_token": "scoped-token",
		"changes": []map[string]any{
			{"path": "boards/sprint.md", "event": "update", "title": "Sprint", "content": "x"},
			{"path": "boards/backlog.md", "event": "update", "title": "Backlog", "content": "y"},
		},
	})
	return b
}

// TestServeDelivery_ForEachChangedFiles asserts a for_each:changed_files role
// runs once per change, each with the instruction rendered for that change_file.
func TestServeDelivery_ForEachChangedFiles(t *testing.T) {
	llm := &recordLLM{}
	f := fanOutFleet(t, llm, "changed_files", "Handle file {{ change_file.Path }}.")
	rec := post(t, f, urlKey("roles/triage.md"), changesBody(t), true)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, llm.systems, 2)
	require.Contains(t, llm.systems[0], "Handle file boards/sprint.md.")
	require.Contains(t, llm.systems[1], "Handle file boards/backlog.md.")

	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 10, resp.TokensUsed) // (3+2)*2 runs
	require.Equal(t, 2, resp.Steps)       // 1 step per run
}

// TestServeDelivery_NoForEach_SingleRunAllChanges asserts the legacy mode runs
// once with the full changed_files list in context.
func TestServeDelivery_NoForEach_SingleRunAllChanges(t *testing.T) {
	llm := &recordLLM{}
	f := fanOutFleet(t, llm, "", "Files:{{ range changed_files }} {{ .Path }}{{ end }}.")
	rec := post(t, f, urlKey("roles/triage.md"), changesBody(t), true)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, llm.systems, 1)
	require.Contains(t, llm.systems[0], "Files: boards/sprint.md boards/backlog.md.")
}

// flakyLLM succeeds on the first run and errors on the second, simulating a
// partial fan-out failure.
type flakyLLM struct{ call int }

func (l *flakyLLM) Chat(_ context.Context, _ string, _ []agentruntime.Message, _ []agentruntime.ToolDef) (agentruntime.ChatResult, error) {
	l.call++
	if l.call == 2 {
		return agentruntime.ChatResult{}, fmt.Errorf("item 2 boom")
	}
	args, _ := json.Marshal(map[string]any{"answer": "ok"})
	return agentruntime.ChatResult{
		ToolCalls:    []agentruntime.ToolCall{{ID: "1", Name: "finish", Arguments: string(args)}},
		PromptTokens: 3, CompletionTokens: 2,
	}, nil
}

// TestServeDelivery_ForEach_ContinueOnError asserts a partial fan-out failure
// does NOT abort the batch: item 1 still runs, spend is summed over the
// successful run(s), per-item errors are reported, and the response is 200
// (not a hard 500).
func TestServeDelivery_ForEach_ContinueOnError(t *testing.T) {
	llm := &flakyLLM{}
	f := fanOutFleet(t, llm, "changed_files", "Handle {{ change_file.Path }}.")
	rec := post(t, f, urlKey("roles/triage.md"), changesBody(t), true)

	require.Equal(t, http.StatusOK, rec.Code, "partial failure must not be a hard 500")

	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 5, resp.TokensUsed) // only item 1 succeeded (3+2)
	require.Equal(t, 1, resp.Steps)      // only item 1
	require.Contains(t, resp.Message, "item 2 boom", "per-item error must be reported")
}

// TestServeDelivery_ForEach_ZeroItems200NoOp asserts that a fan-out with no
// items to iterate (empty changes[] for changed_files, empty attached_notes for
// attached_notes) is a 200 no-op, not a 502. A 502 makes trip2g retry the empty
// batch to exhaustion and mark the delivery failed.
func TestServeDelivery_ForEach_ZeroItems200NoOp(t *testing.T) {
	emptyBody := func(t *testing.T) []byte {
		t.Helper()
		b, _ := json.Marshal(map[string]any{
			"depth": 0, "api_token": "scoped-token",
			"changes":        []map[string]any{},
			"attached_notes": []map[string]any{},
		})
		return b
	}
	for _, mode := range []string{"changed_files", "attached_notes"} {
		t.Run(mode, func(t *testing.T) {
			llm := &recordLLM{}
			f := fanOutFleet(t, llm, mode, "Body {{ depth }}.")
			rec := post(t, f, urlKey("roles/triage.md"), emptyBody(t), true)

			require.Equal(t, http.StatusOK, rec.Code, "zero-item fan-out must be a 200 no-op")
			require.Empty(t, llm.systems, "no items means the agent loop never runs")

			var resp webhookutil.AgentResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.Equal(t, "completed", resp.Status)
			require.Equal(t, 0, resp.TokensUsed)
			require.Equal(t, 0, resp.Steps)
			require.Contains(t, resp.Message, mode, "no-op message should name the empty collection")
		})
	}
}

// TestServeDelivery_ForEach_AllErrors502 asserts that when every fan-out item
// fails, the whole batch is a non-2xx so trip2g's retry/backoff engages.
func TestServeDelivery_ForEach_AllErrors502(t *testing.T) {
	f := fanOutFleet(t, &errLLM{msg: "always boom"}, "changed_files", "Handle {{ change_file.Path }}.")
	rec := post(t, f, urlKey("roles/triage.md"), changesBody(t), true)

	require.GreaterOrEqual(t, rec.Code, 500)
	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "error", resp.Status)
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

// TestServeDelivery_MaxBytesReader413 is a true regression test for the
// MaxBytesReader DoS guard.
//
// The body is a valid, correctly-signed JSON payload whose size exceeds
// maxBodyBytes. With MaxBytesReader in place io.ReadAll stops at the limit
// and the handler returns 400. Without it the body is fully read, HMAC
// verifies, JSON parses, the agent runs, and the response is 200 — so
// removing MaxBytesReader makes this test fail.
func TestServeDelivery_MaxBytesReader413(t *testing.T) {
	client := &ClientMock{
		GraphQLScopedFunc: func(_ context.Context, _, _ string, _ map[string]any) (json.RawMessage, error) {
			return json.RawMessage(`{"updateNotes":{"paths":["boards/sprint.md"]}}`), nil
		},
	}
	f := newTestFleetWithLLM(client, &stubLLM{})
	key := urlKey("roles/triage.md")

	// Construct a valid JSON body that is larger than maxBodyBytes. The
	// instruction field carries a long string so json.Unmarshal would succeed
	// and the agent would run (returning 200) if MaxBytesReader were absent.
	big := strings.Repeat("a", maxBodyBytes) // JSON body ~= maxBodyBytes + 50 bytes
	oversized, err := json.Marshal(map[string]any{
		"depth":       0,
		"instruction": big,
		"api_token":   "scoped-token",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/deliver/"+key, bytes.NewReader(oversized))
	role, ok := f.roleByKey(key)
	require.True(t, ok)
	req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(oversized, f.secretFor(role)))

	rec := httptest.NewRecorder()
	f.ServeDelivery(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code,
		"oversized body must be rejected by MaxBytesReader before the agent runs")
}
