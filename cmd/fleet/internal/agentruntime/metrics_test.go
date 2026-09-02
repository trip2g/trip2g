package agentruntime

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// recordedRun is one RecordRun call.
type recordedRun struct {
	role, status string
	steps        int
}

// recordedTokens is one RecordTokens call.
type recordedTokens struct {
	model, role, kind string
	n                 int
}

// fakeMetrics captures what the loop reports, so a test asserts on the
// recorded facts rather than on a Prometheus registry.
type fakeMetrics struct {
	runs    []recordedRun
	tokens  []recordedTokens
	tools   map[string]string // tool -> last outcome
	denials []string
	applies []string
}

func newFakeMetrics() *fakeMetrics {
	return &fakeMetrics{tools: map[string]string{}}
}

func (f *fakeMetrics) RecordRun(role, status string, steps int, _ float64) {
	f.runs = append(f.runs, recordedRun{role: role, status: status, steps: steps})
}

func (f *fakeMetrics) RecordTokens(model, role, kind string, n int) {
	f.tokens = append(f.tokens, recordedTokens{model: model, role: role, kind: kind, n: n})
}

func (f *fakeMetrics) RecordToolCall(tool, outcome string) { f.tools[tool] = outcome }

func (f *fakeMetrics) RecordDenial(kind string) { f.denials = append(f.denials, kind) }

func (f *fakeMetrics) RecordApplyFailure(_, tool string) { f.applies = append(f.applies, tool) }

// totalTokens sums the recorded spend, mirroring what Result.TokensUsed counts.
// The cached kind is deliberately excluded: it is a share of the prompt tokens
// already counted, not spend on top of them.
func (f *fakeMetrics) totalTokens() int {
	sum := 0
	for _, t := range f.tokens {
		if t.kind == tokenKindCached {
			continue
		}
		sum += t.n
	}
	return sum
}

// tokensOfKind sums the recorded tokens of one kind.
func (f *fakeMetrics) tokensOfKind(kind string) int {
	sum := 0
	for _, t := range f.tokens {
		if t.kind == kind {
			sum += t.n
		}
	}
	return sum
}

// TestRun_RecordsSpendStatusAndDenials drives one run through a denied write,
// a successful write and finish, then asserts every lane was reported: the
// terminal status, the token spend split by kind, the tool outcomes and the
// scope denial.
func TestRun_RecordsSpendStatusAndDenials(t *testing.T) {
	kb := newMemKB(map[string]string{"boards/a.md": "todo"})
	llm := &stubLLM{script: []ChatResult{
		{ToolCalls: []ToolCall{toolCall("1", toolWriteNote, map[string]any{"path": "secrets/x.md", "content": "nope"})}, PromptTokens: 10, CompletionTokens: 5},
		{ToolCalls: []ToolCall{toolCall("2", toolWriteNote, map[string]any{"path": "boards/a.md", "content": "doing"})}, PromptTokens: 10, CompletionTokens: 5},
		{ToolCalls: []ToolCall{toolCall("3", toolFinish, map[string]any{"answer": "ok"})}, PromptTokens: 10, CompletionTokens: 5},
	}}
	m := newFakeMetrics()

	res, err := Run(context.Background(), Input{
		Instruction:   "Move the card.",
		ReadPatterns:  []string{"boards/**"},
		WritePatterns: []string{"boards/**"},
		Model:         "test-model",
		Role:          "roles/triage.md",
		Metrics:       m,
		MaxTokens:     10000,
		MaxSteps:      10,
		LLM:           llm,
		KB:            kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)

	require.Len(t, m.runs, 1)
	require.Equal(t, recordedRun{role: "roles/triage.md", status: StatusCompleted, steps: res.Steps}, m.runs[0])

	// Token spend is reported per model+role and must match the run's own tally.
	require.Equal(t, res.TokensUsed, m.totalTokens())
	require.Equal(t, "test-model", m.tokens[0].model)
	require.Equal(t, "roles/triage.md", m.tokens[0].role)

	// The denied write is the last write_note outcome before the allowed one, so
	// assert the denial lane explicitly rather than the overwritten tool entry.
	require.Equal(t, []string{denialWrite}, m.denials)
	require.Equal(t, outcomeOK, m.tools[toolWriteNote])
}

// TestRun_RecordsCappedStatus asserts a run stopped by the token hard-cap is
// reported as capped — the runaway-spend signal.
func TestRun_RecordsCappedStatus(t *testing.T) {
	kb := newMemKB(nil)
	llm := &stubLLM{fallback: ChatResult{
		ToolCalls:    []ToolCall{toolCall("1", toolSearch, map[string]any{"query": "x"})},
		PromptTokens: 60, CompletionTokens: 60,
	}}
	m := newFakeMetrics()

	res, err := Run(context.Background(), Input{
		Instruction:  "Search forever.",
		ReadPatterns: []string{"**"},
		Model:        "test-model",
		Role:         "roles/loop.md",
		Metrics:      m,
		MaxTokens:    100,
		MaxSteps:     10,
		LLM:          llm,
		KB:           kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusCapped, res.Status)
	require.Len(t, m.runs, 1)
	require.Equal(t, StatusCapped, m.runs[0].status)
}

// TestRun_RecordsApplyFailure asserts a genuine apply failure is counted (and,
// under HardFailApply, reported as an errored run rather than a completed one).
func TestRun_RecordsApplyFailure(t *testing.T) {
	kb := newMemKB(map[string]string{"boards/a.md": "todo todo"})
	llm := &stubLLM{fallback: ChatResult{
		ToolCalls: []ToolCall{toolCall("1", toolPatchNote, map[string]any{
			"path": "boards/a.md", "find": "todo", "replace": "doing",
		})},
		PromptTokens: 10, CompletionTokens: 5,
	}}
	m := newFakeMetrics()

	_, err := Run(context.Background(), Input{
		Instruction:   "Patch it.",
		ReadPatterns:  []string{"boards/**"},
		WritePatterns: []string{"boards/**"},
		Model:         "test-model",
		Role:          "roles/patch.md",
		Metrics:       m,
		HardFailApply: true,
		MaxTokens:     10000,
		MaxSteps:      5,
		LLM:           llm,
		KB:            kb,
	})
	require.Error(t, err) // ambiguous find: the patch cannot be applied

	require.Equal(t, []string{toolPatchNote}, m.applies)
	require.Equal(t, outcomeApplyFailed, m.tools[toolPatchNote])
	require.Len(t, m.runs, 1)
	require.Equal(t, statusErrorForRun, m.runs[0].status)
}

// TestRun_RecordsNotPermittedTool asserts a tool outside the role's allowlist
// is counted as such — the hallucination/injection signal.
func TestRun_RecordsNotPermittedTool(t *testing.T) {
	kb := newMemKB(nil)
	llm := &stubLLM{script: []ChatResult{
		{ToolCalls: []ToolCall{toolCall("1", toolWriteNote, map[string]any{"path": "boards/a.md", "content": "x"})}, PromptTokens: 5, CompletionTokens: 5},
		{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})}, PromptTokens: 5, CompletionTokens: 5},
	}}
	m := newFakeMetrics()

	_, err := Run(context.Background(), Input{
		Instruction:  "Read only.",
		ReadPatterns: []string{"boards/**"},
		Tools:        []string{toolReadNote},
		Model:        "test-model",
		Role:         "roles/ro.md",
		Metrics:      m,
		MaxTokens:    10000,
		MaxSteps:     5,
		LLM:          llm,
		KB:           kb,
	})
	require.NoError(t, err)
	// The rejected name is model-supplied, so it is bucketed: a hallucinated or
	// injected tool name must never mint a metric series of its own.
	require.Equal(t, outcomeNotPermitted, m.tools[unknownTool])
	require.NotContains(t, m.tools, toolWriteNote)
	require.Equal(t, []string{denialNotPermitted}, m.denials)
}

// TestRun_RecordsFinish asserts the ordinary successful termination is counted:
// finish is the tool call most runs end on.
func TestRun_RecordsFinish(t *testing.T) {
	llm := &stubLLM{fallback: ChatResult{
		ToolCalls:    []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "done"})},
		PromptTokens: 5, CompletionTokens: 5,
	}}
	m := newFakeMetrics()

	_, err := Run(context.Background(), Input{
		Instruction: "Finish.", Model: "test-model", Role: "roles/f.md", Metrics: m,
		MaxTokens: 1000, MaxSteps: 3, LLM: llm, KB: newMemKB(nil),
	})
	require.NoError(t, err)
	require.Equal(t, outcomeOK, m.tools[toolFinish])
}

// TestRun_FailedRunKeepsItsSteps asserts a run that dies mid-loop still reports
// the steps and spend it already burned — otherwise a failing role looks free.
func TestRun_FailedRunKeepsItsSteps(t *testing.T) {
	kb := newMemKB(map[string]string{"boards/a.md": "todo todo"})
	llm := &stubLLM{fallback: ChatResult{
		ToolCalls: []ToolCall{toolCall("1", toolPatchNote, map[string]any{
			"path": "boards/a.md", "find": "todo", "replace": "doing",
		})},
		PromptTokens: 10, CompletionTokens: 5,
	}}
	m := newFakeMetrics()

	res, err := Run(context.Background(), Input{
		Instruction: "Patch it.", ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		Model: "test-model", Role: "roles/patch.md", Metrics: m, HardFailApply: true,
		MaxTokens: 10000, MaxSteps: 5, LLM: llm, KB: kb,
	})
	require.Error(t, err)
	require.Nil(t, res, "the caller contract is nil result on error")

	require.Len(t, m.runs, 1)
	require.Equal(t, statusErrorForRun, m.runs[0].status)
	require.Equal(t, 1, m.runs[0].steps, "the step it burned before failing must still be reported")
	require.Equal(t, 15, m.totalTokens())
}

// llmMetricsRecorder captures the provider-call lane.
type llmMetricsRecorder struct {
	requests []string // "lane|model|status"
	retries  []string // "lane|reason"
}

func (r *llmMetricsRecorder) RecordLLMRequest(lane, model, status string, _ float64) {
	r.requests = append(r.requests, lane+"|"+model+"|"+status)
}

func (r *llmMetricsRecorder) RecordLLMRetry(lane, reason string) {
	r.retries = append(r.retries, lane+"|"+reason)
}

// TestOpenAILLM_RecordsRetriesByReason asserts each retried attempt is
// attributed to what caused it — the earliest upstream-instability signal —
// and that the surrounding call is recorded once, as a success.
func TestOpenAILLM_RecordsRetriesByReason(t *testing.T) {
	llm, _ := llmStubServer(t, func(attempt int, _ map[string]any) (int, string) {
		switch attempt {
		case 1:
			return http.StatusTooManyRequests, `{"error":{"message":"rate limited","type":"rate_limit_exceeded"}}`
		case 2:
			return http.StatusInternalServerError, `{"error":{"message":"boom","type":"server_error"}}`
		default:
			return http.StatusOK, okCompletion
		}
	})
	rec := &llmMetricsRecorder{}
	llm.SetMetrics(rec, "llm")

	_, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.NoError(t, err)

	require.Equal(t, []string{"llm|429", "llm|5xx"}, rec.retries)
	require.Equal(t, []string{"llm|test-model|ok"}, rec.requests)
}

// TestOpenAILLM_RecordsTerminalFailureReason asserts a non-retryable failure is
// recorded with its HTTP class rather than as a generic error.
func TestOpenAILLM_RecordsTerminalFailureReason(t *testing.T) {
	llm, _ := llmStubServer(t, func(int, map[string]any) (int, string) {
		return http.StatusUnprocessableEntity, `{"error":{"message":"bad code","type":"code_execution_error"}}`
	})
	rec := &llmMetricsRecorder{}
	llm.SetMetrics(rec, "exec")

	_, err := llm.Chat(context.Background(), "codellm", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.Error(t, err)

	require.Empty(t, rec.retries)
	require.Equal(t, []string{"exec|codellm|4xx"}, rec.requests)
}

// TestOpenAILLM_ExhaustedRetriesRecordOnce asserts the final failure is the
// request's outcome and not also counted as a retry — otherwise the retry rate
// overstates by one per failed call.
func TestOpenAILLM_ExhaustedRetriesRecordOnce(t *testing.T) {
	llm, calls := llmStubServer(t, func(int, map[string]any) (int, string) {
		return http.StatusInternalServerError, `{"error":{"message":"boom","type":"server_error"}}`
	})
	rec := &llmMetricsRecorder{}
	llm.SetMetrics(rec, "llm")

	_, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.Error(t, err)

	require.Equal(t, int32(llmMaxAttempts), calls.Load())
	require.Len(t, rec.retries, llmMaxAttempts-1, "the last attempt is the outcome, not a retry")
	require.Equal(t, []string{"llm|test-model|5xx"}, rec.requests)
}

// TestOpenAILLM_EmptyCompletionIsNotSuccess asserts a 200 carrying no choices
// is recorded as the provider-side failure it is.
func TestOpenAILLM_EmptyCompletionIsNotSuccess(t *testing.T) {
	llm, _ := llmStubServer(t, func(int, map[string]any) (int, string) {
		return http.StatusOK, `{"id":"1","object":"chat.completion","choices":[],"usage":{"prompt_tokens":7,"completion_tokens":0}}`
	})
	rec := &llmMetricsRecorder{}
	llm.SetMetrics(rec, "llm")

	_, err := llm.Chat(context.Background(), "test-model", []Message{{Role: RoleUser, Content: "x"}}, nil)
	require.ErrorIs(t, err, ErrEmptyCompletion)
	require.Equal(t, []string{"llm|test-model|empty_completion"}, rec.requests)
}

// TestRun_RecordsCachedPromptTokens: the cache-hit share is reported as its own
// kind and must NOT inflate the run's spend — it is part of the prompt tokens
// already counted, so double-counting it would make the budget bind early.
func TestRun_RecordsCachedPromptTokens(t *testing.T) {
	llm := &stubLLM{script: []ChatResult{
		{
			ToolCalls:          []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "ok"})},
			PromptTokens:       100,
			CompletionTokens:   10,
			CachedPromptTokens: 80,
		},
	}}
	m := newFakeMetrics()

	res, err := Run(context.Background(), Input{
		Instruction: "go", Model: "m", Role: "roles/r.md",
		MaxTokens: 1000, MaxSteps: 5,
		Metrics: m, LLM: llm, KB: newMemKB(nil),
	})
	require.NoError(t, err)
	require.Equal(t, 110, res.TokensUsed)
	require.Equal(t, 80, m.tokensOfKind(tokenKindCached))
	require.Equal(t, res.TokensUsed, m.totalTokens())
}
