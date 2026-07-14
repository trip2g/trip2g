package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"trip2g/internal/agentruntime"
	"trip2g/internal/webhookutil"

	"github.com/stretchr/testify/require"
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

func newTestFleet(hc *http.Client) *Fleet {
	role := Role{
		NotePath: "roles/triage.md", Body: "Triage.", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		MaxTokens: 4000, MaxSteps: 6, Concurrency: "skip", MaxDepth: 1,
	}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000, StepCeiling: 25,
	}
	f := NewFleet(cfg, hc, &stubLLM{})
	f.SetRoles([]Role{role})
	return f
}

// newScopedKBServer starts an httptest server that handles scoped /_system/graphql
// requests. respond is called with the Authorization header value and must return
// JSON to put under {"data": ...}. A nil respond returns an empty UpdateNotes success.
func newScopedKBServer(t *testing.T, respond func(auth, body string) string) (*httptest.Server, *http.Client) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		rawBody, _ := io.ReadAll(r.Body)
		var data string
		if respond != nil {
			data = respond(r.Header.Get("Authorization"), string(rawBody))
		} else {
			data = `{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":[]}}`
		}
		_, _ = w.Write([]byte(`{"data":` + data + `}`))
	}))
	t.Cleanup(srv.Close)
	return srv, srv.Client()
}

func post(t *testing.T, f *Fleet, key string, body []byte, sign bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, f.WebhookPath()+key, bytes.NewReader(body))
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
	var scopedCalls atomic.Int32
	var lastToken atomic.Value
	srv, hc := newScopedKBServer(t, func(auth, _ string) string {
		scopedCalls.Add(1)
		lastToken.Store(auth)
		return `{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["boards/sprint.md"]}}`
	})

	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		Trip2gBaseURL: srv.URL, TokenCeiling: 100000, StepCeiling: 25,
	}
	role := Role{
		NotePath: "roles/triage.md", Body: "Triage.", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		MaxTokens: 4000, MaxSteps: 6, Concurrency: "skip", MaxDepth: 1,
	}
	f := NewFleet(cfg, hc, &stubLLM{})
	f.SetRoles([]Role{role})

	key := urlKey("roles/triage.md")
	rec := post(t, f, key, deliveryBody(t), true)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int32(1), scopedCalls.Load()) // exactly one scoped updateNotes
	require.Equal(t, "Bearer scoped-token", lastToken.Load())

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
			{
				"path":       "roles/triage.md",
				"content":    "role body",
				"title":      "Triage role",
				"tags":       []string{"role", "triage"},
				"meta":       map[string]string{"layout": "doc"},
				"updated_at": "2026-06-29T10:00:00Z",
			},
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

// ctxProbeLLM blocks the first Chat call until the test signals it (after the
// test has cancelled the request context), then records whether the run context
// it received is still alive. A live (nil-error) run context proves the run was
// detached from the request context. It writes once via patch_note, then
// finishes, so the test can also assert the write-back survived the cancel.
type ctxProbeLLM struct {
	started   chan struct{}
	proceed   chan struct{}
	runCtxErr error
	idx       int
}

func (l *ctxProbeLLM) Chat(ctx context.Context, _ string, _ []agentruntime.Message, _ []agentruntime.ToolDef) (agentruntime.ChatResult, error) {
	defer func() { l.idx++ }()
	if l.idx == 0 {
		close(l.started)
		<-l.proceed
		l.runCtxErr = ctx.Err()
		args, _ := json.Marshal(map[string]any{"path": "boards/sprint.md", "find": "todo", "replace": "doing"})
		return agentruntime.ChatResult{
			ToolCalls:    []agentruntime.ToolCall{{ID: "1", Name: "patch_note", Arguments: string(args)}},
			PromptTokens: 1, CompletionTokens: 1,
		}, nil
	}
	args, _ := json.Marshal(map[string]any{"answer": "done"})
	return agentruntime.ChatResult{
		ToolCalls:    []agentruntime.ToolCall{{ID: "2", Name: "finish", Arguments: string(args)}},
		PromptTokens: 1, CompletionTokens: 1,
	}, nil
}

// TestServeDelivery_RunDetachedFromRequestContext is the regression test for
// finding #4: trip2g closes the delivery connection when its change-webhook
// timeoutSeconds (default 60s) elapses, which cancels the inbound request
// context. The agent run MUST NOT be tied to that context — otherwise a slow
// run is aborted mid-flight and its write-back is lost. Here the request context
// is cancelled while the agent is mid-run; the run must still see a live context
// and complete + write.
func TestServeDelivery_RunDetachedFromRequestContext(t *testing.T) {
	var scopedCalls atomic.Int32
	srv, hc := newScopedKBServer(t, func(_, _ string) string {
		scopedCalls.Add(1)
		return `{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["boards/sprint.md"]}}`
	})

	llm := &ctxProbeLLM{started: make(chan struct{}), proceed: make(chan struct{})}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		Trip2gBaseURL: srv.URL, TokenCeiling: 100000, StepCeiling: 25,
	}
	role := Role{
		NotePath: "roles/triage.md", Body: "Triage.", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		MaxTokens: 4000, MaxSteps: 6, Concurrency: "skip", MaxDepth: 1,
	}
	f := NewFleet(cfg, hc, llm)
	f.SetRoles([]Role{role})

	key := urlKey("roles/triage.md")
	body := deliveryBody(t)
	reqCtx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, f.WebhookPath()+key, bytes.NewReader(body)).WithContext(reqCtx)
	r, ok := f.roleByKey(key)
	require.True(t, ok)
	req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(body, f.secretFor(r)))
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		f.ServeDelivery(rec, req)
		close(done)
	}()

	<-llm.started      // the handler has entered the agent run
	cancel()           // simulate trip2g closing the delivery at its timeout
	close(llm.proceed) // let the run continue
	<-done

	require.NoError(t, llm.runCtxErr,
		"agent run context must NOT be cancelled by the inbound request context")
	require.Equal(t, http.StatusOK, rec.Code, "run must complete despite request cancel")
	require.Equal(t, int32(1), scopedCalls.Load(), "write-back must still happen after request cancel")
}

func TestServeDelivery_BadHMAC401(t *testing.T) {
	f := newTestFleet(http.DefaultClient)
	rec := post(t, f, urlKey("roles/triage.md"), deliveryBody(t), false)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestServeDelivery_UnknownKey404(t *testing.T) {
	f := newTestFleet(http.DefaultClient)
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
	f := NewFleet(cfg, http.DefaultClient, llm)
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
		return agentruntime.ChatResult{}, errors.New("item 2 boom")
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

// capThenCompleteLLM caps item 1's run (a non-finish, non-permitted tool call
// whose tokens exceed the per-run cap, so the next loop step trips the hard-cap
// without touching the KB) then completes every later run.
type capThenCompleteLLM struct{ call int }

func (l *capThenCompleteLLM) Chat(_ context.Context, _ string, _ []agentruntime.Message, _ []agentruntime.ToolDef) (agentruntime.ChatResult, error) {
	l.call++
	if l.call == 1 {
		args, _ := json.Marshal(map[string]any{})
		return agentruntime.ChatResult{
			ToolCalls:    []agentruntime.ToolCall{{ID: "1", Name: "noop", Arguments: string(args)}},
			PromptTokens: 5000, CompletionTokens: 0,
		}, nil
	}
	args, _ := json.Marshal(map[string]any{"answer": "ok"})
	return agentruntime.ChatResult{
		ToolCalls:    []agentruntime.ToolCall{{ID: "2", Name: "finish", Arguments: string(args)}},
		PromptTokens: 3, CompletionTokens: 2,
	}, nil
}

// TestServeDelivery_ForEach_AggregatesCappedStatus asserts that when one fan-out
// item is capped and another completes, the aggregate status reflects the cap
// (not the last run's "completed").
func TestServeDelivery_ForEach_AggregatesCappedStatus(t *testing.T) {
	f := fanOutFleet(t, &capThenCompleteLLM{}, "changed_files", "Handle {{ change_file.Path }}.")
	rec := post(t, f, urlKey("roles/triage.md"), changesBody(t), true)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, agentruntime.StatusCapped, resp.Status,
		"aggregate status must reflect the capped item, not the last completed one")
}

// newCodeRoleFleet builds a Fleet with a code-executor role and a stub codeRunner.
func newCodeRoleFleet(stubRunner func(context.Context, agentruntime.CodeInput) (*agentruntime.Result, error)) *Fleet {
	role := Role{
		NotePath:      "roles/code.md",
		Mode:          "change",
		Executor:      "code",
		WritePatterns: []string{"boards/**"},
		Body:          "```bash\necho hi\n```",
	}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000, StepCeiling: 25,
		AllowedPrograms: []string{"bash"},
	}
	f := NewFleet(cfg, http.DefaultClient, nil) // llm=nil is fine for code roles
	f.codeRunner = stubRunner
	f.SetRoles([]Role{role})
	return f
}

// TestServeDelivery_CodeRole_DispatchesRunCode asserts that a role with
// executor:code invokes the codeRunner (not agentruntime.Run), and that the
// response reports TokensUsed=0 (code roles have no LLM spend).
func TestServeDelivery_CodeRole_DispatchesRunCode(t *testing.T) {
	var runCodeCalled bool
	var receivedWritePatterns []string

	stub := func(_ context.Context, in agentruntime.CodeInput) (*agentruntime.Result, error) {
		runCodeCalled = true
		receivedWritePatterns = in.WritePatterns
		return &agentruntime.Result{
			Status:     agentruntime.StatusCompleted,
			Answer:     "code done",
			TokensUsed: 0,
			Steps:      1,
		}, nil
	}

	f := newCodeRoleFleet(stub)
	key := urlKey("roles/code.md")
	body, _ := json.Marshal(map[string]any{
		"depth":       0,
		"instruction": "run code",
		"api_token":   "scoped-token",
	})
	req := httptest.NewRequest(http.MethodPost, f.WebhookPath()+key, bytes.NewReader(body))
	role, ok := f.roleByKey(key)
	require.True(t, ok)
	req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(body, f.secretFor(role)))
	rec := httptest.NewRecorder()
	f.ServeDelivery(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, runCodeCalled, "code executor role must invoke codeRunner, not agentruntime.Run")
	require.Equal(t, []string{"boards/**"}, receivedWritePatterns)

	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.TokensUsed, "code roles have zero LLM token spend")
	require.Equal(t, agentruntime.StatusCompleted, resp.Status)
}

// TestServeDelivery_CodeRole_PassesEnvPassthrough asserts that env_passthrough
// and env_prefix declared on the role are threaded into the CodeInput.
func TestServeDelivery_CodeRole_PassesEnvPassthrough(t *testing.T) {
	var receivedEnvPassthrough []string
	var receivedEnvPrefix []string

	stub := func(_ context.Context, in agentruntime.CodeInput) (*agentruntime.Result, error) {
		receivedEnvPassthrough = in.EnvPassthrough
		receivedEnvPrefix = in.EnvPrefix
		return &agentruntime.Result{Status: agentruntime.StatusCompleted}, nil
	}

	role := Role{
		NotePath:       "roles/envcode.md",
		Mode:           "change",
		Executor:       "code",
		WritePatterns:  []string{"notes/**"},
		Body:           "```bash\necho hi\n```",
		EnvPassthrough: []string{"MY_TOKEN"},
		EnvPrefix:      []string{"KRISP_"},
	}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000, StepCeiling: 25,
		AllowedPrograms: []string{"bash"},
	}
	f := NewFleet(cfg, http.DefaultClient, nil)
	f.codeRunner = stub
	f.SetRoles([]Role{role})

	key := urlKey("roles/envcode.md")
	body, _ := json.Marshal(map[string]any{
		"depth": 0, "api_token": "tok",
	})
	req := httptest.NewRequest(http.MethodPost, f.WebhookPath()+key, bytes.NewReader(body))
	r, ok := f.roleByKey(key)
	require.True(t, ok)
	req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(body, f.secretFor(r)))
	rec := httptest.NewRecorder()
	f.ServeDelivery(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"MY_TOKEN"}, receivedEnvPassthrough)
	require.Equal(t, []string{"KRISP_"}, receivedEnvPrefix)
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
	return agentruntime.ChatResult{}, errors.New(e.msg)
}

// -- Cron delivery tests --

// newCronFleet builds a Fleet with a cron-mode role for delivery tests.
func newCronFleet(llm agentruntime.LLM) *Fleet {
	role := Role{
		NotePath:      "roles/kb-refresh.md",
		Body:          `Refresh the KB at {{ now.Format("2006-01-02") }}.`,
		Mode:          "cron",
		CronSchedule:  "0 */6 * * *",
		ReadPatterns:  []string{"kb/**"},
		WritePatterns: []string{"kb/**"},
		MaxTokens:     4000,
		MaxSteps:      6,
	}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000, StepCeiling: 25,
	}
	f := NewFleet(cfg, http.DefaultClient, llm)
	f.SetRoles([]Role{role})
	return f
}

// cronDeliveryBody returns a minimal cron delivery payload (no changes[]).
func cronDeliveryBody(t *testing.T) []byte {
	t.Helper()
	b, _ := json.Marshal(map[string]any{
		"depth":          0,
		"api_token":      "cron-token",
		"attached_notes": []map[string]any{},
	})
	return b
}

// postCron sends a POST to /deliver/cron/<key> with a cron-signed HMAC.
func postCron(t *testing.T, f *Fleet, key string, body []byte, sign bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, f.WebhookPath()+"cron/"+key, bytes.NewReader(body))
	if sign {
		role, ok := f.roleByKey(key)
		require.True(t, ok)
		req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(body, f.cronSecretFor(role)))
	}
	rec := httptest.NewRecorder()
	f.ServeDelivery(rec, req)
	return rec
}

// TestServeCronDelivery_HappyPath asserts that a cron delivery (no changes[])
// triggers a single LLM run and returns 200 with spend data.
func TestServeCronDelivery_HappyPath(t *testing.T) {
	llm := &recordLLM{}
	f := newCronFleet(llm)
	key := urlKey("roles/kb-refresh.md")
	rec := postCron(t, f, key, cronDeliveryBody(t), true)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, llm.systems, 1, "cron delivery must trigger exactly one LLM run")

	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, agentruntime.StatusCompleted, resp.Status)
	require.Equal(t, 5, resp.TokensUsed) // (3+2)*1
}

// TestServeCronDelivery_NowVarRendered asserts the `now` template variable is
// non-zero and renders into the instruction (proof the var is injected).
func TestServeCronDelivery_NowVarRendered(t *testing.T) {
	llm := &recordLLM{}
	f := newCronFleet(llm)
	key := urlKey("roles/kb-refresh.md")
	rec := postCron(t, f, key, cronDeliveryBody(t), true)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, llm.systems, 1)
	// The body template is "Refresh the KB at {{ now.Format \"2006-01-02\" }}."
	// A rendered date looks like "2026-06-30"; it must not be the zero-value "0001-01-01".
	require.NotContains(t, llm.systems[0], "0001-01-01", "now must not be zero")
	require.Regexp(t, `\d{4}-\d{2}-\d{2}`, llm.systems[0], "now must render as a date")
}

// TestServeCronDelivery_BadHMAC401 asserts that a wrong HMAC is rejected.
func TestServeCronDelivery_BadHMAC401(t *testing.T) {
	f := newCronFleet(&recordLLM{})
	key := urlKey("roles/kb-refresh.md")
	rec := postCron(t, f, key, cronDeliveryBody(t), false) // unsigned
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestServeCronDelivery_ChangeHMACRejected asserts the change-webhook HMAC is
// NOT accepted for a cron delivery (the two secrets are derived differently).
func TestServeCronDelivery_ChangeHMACRejected(t *testing.T) {
	llm := &recordLLM{}
	f := newCronFleet(llm)
	key := urlKey("roles/kb-refresh.md")
	body := cronDeliveryBody(t)

	// Sign with the change secret (not the cron secret).
	role, ok := f.roleByKey(key)
	require.True(t, ok)
	req := httptest.NewRequest(http.MethodPost, f.WebhookPath()+"cron/"+key, bytes.NewReader(body))
	req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(body, f.secretFor(role)))
	rec := httptest.NewRecorder()
	f.ServeDelivery(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code,
		"change-webhook HMAC must not be valid for cron delivery")
}

// TestServeCronDelivery_WithAttachedNotes asserts that attached_notes in the
// cron payload are accessible via the overlay/KB.
func TestServeCronDelivery_WithAttachedNotes(t *testing.T) {
	role := Role{
		NotePath:      "roles/notes-role.md",
		Body:          "Process {{ range attached_notes }}{{ .Path }} {{ end }}.",
		Mode:          "cron",
		CronSchedule:  "*/5 * * * *",
		ReadPatterns:  []string{"**"},
		WritePatterns: []string{"**"},
		MaxSteps:      6,
	}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000, StepCeiling: 25,
	}
	llm := &recordLLM{}
	f := NewFleet(cfg, http.DefaultClient, llm)
	f.SetRoles([]Role{role})

	key := urlKey(role.NotePath)
	body, _ := json.Marshal(map[string]any{
		"depth":     0,
		"api_token": "tok",
		"attached_notes": []map[string]any{
			{"path": "kb/a.md", "content": "content A"},
			{"path": "kb/b.md", "content": "content B"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, f.WebhookPath()+"cron/"+key, bytes.NewReader(body))
	req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(body, f.cronSecretFor(role)))
	rec := httptest.NewRecorder()
	f.ServeDelivery(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, llm.systems, 1)
	require.Contains(t, llm.systems[0], "kb/a.md")
	require.Contains(t, llm.systems[0], "kb/b.md")
}

// TestServeCronDelivery_RunError502 asserts a cron run error produces 502.
func TestServeCronDelivery_RunError502(t *testing.T) {
	f := newCronFleet(&errLLM{msg: "llm unavailable"})
	key := urlKey("roles/kb-refresh.md")
	rec := postCron(t, f, key, cronDeliveryBody(t), true)

	require.GreaterOrEqual(t, rec.Code, 500)
	var resp webhookutil.AgentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "error", resp.Status)
}

// newTestFleetWithLLM builds a Fleet with the given LLM (no KB writes expected).
func newTestFleetWithLLM(llm agentruntime.LLM) *Fleet {
	role := Role{
		NotePath: "roles/triage.md", Body: "Triage.", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		MaxTokens: 4000, MaxSteps: 6, Concurrency: "skip", MaxDepth: 1,
	}
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000, StepCeiling: 25,
	}
	f := NewFleet(cfg, http.DefaultClient, llm)
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
			f := newTestFleetWithLLM(&errLLM{msg: tc.llmErrMsg})
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
	// Body is too large to even parse — rejected before any LLM or KB call.
	f := newTestFleetWithLLM(&stubLLM{})
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

	req := httptest.NewRequest(http.MethodPost, f.WebhookPath()+key, bytes.NewReader(oversized))
	role, ok := f.roleByKey(key)
	require.True(t, ok)
	req.Header.Set("X-Webhook-Signature", webhookutil.SignHMAC(oversized, f.secretFor(role)))

	rec := httptest.NewRecorder()
	f.ServeDelivery(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code,
		"oversized body must be rejected by MaxBytesReader before the agent runs")
}
