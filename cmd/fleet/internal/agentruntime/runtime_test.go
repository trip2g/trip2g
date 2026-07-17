package agentruntime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"trip2g/internal/webhookutil"

	"github.com/stretchr/testify/require"
)

// memKB is an in-memory KB for deterministic, offline tests.
type memKB struct {
	docs map[string]string
}

func newMemKB(seed map[string]string) *memKB {
	docs := map[string]string{}
	for k, v := range seed {
		docs[k] = v
	}
	return &memKB{docs: docs}
}

func (m *memKB) Search(_ context.Context, query string) ([]Doc, error) {
	q := strings.ToLower(query)
	var out []Doc
	for path, content := range m.docs {
		if q == "" || strings.Contains(strings.ToLower(content), q) {
			out = append(out, Doc{Path: path, Content: content})
		}
	}
	return out, nil
}

func (m *memKB) Read(_ context.Context, path string) (string, error) {
	content, ok := m.docs[path]
	if !ok {
		return "", errNotFound
	}
	return content, nil
}

func (m *memKB) Write(_ context.Context, path string, content string) error {
	m.docs[path] = content
	return nil
}

func (m *memKB) Patch(_ context.Context, path, find, replace string) error {
	content, ok := m.docs[path]
	if !ok {
		return errNotFound
	}
	idx := strings.Index(content, find)
	if idx == -1 {
		return &kbError{"patch find not found"}
	}
	if strings.Contains(content[idx+len(find):], find) {
		return &kbError{"patch find is ambiguous (multiple occurrences)"}
	}
	m.docs[path] = content[:idx] + replace + content[idx+len(find):]
	return nil
}

var errNotFound = &kbError{"not found"}

type kbError struct{ msg string }

func (e *kbError) Error() string { return e.msg }

// stubLLM returns a scripted sequence of responses, then a fixed fallback. It
// records every message batch it receives so a test can assert what the model
// was (or was not) shown. seenTools records the tool list from the first Chat
// call so tests can assert which tools were offered to the model.
type stubLLM struct {
	script    []ChatResult
	fallback  ChatResult
	idx       int
	seen      []Message
	seenTools []ToolDef
}

func (s *stubLLM) Chat(_ context.Context, _ string, messages []Message, tools []ToolDef) (ChatResult, error) {
	s.seen = append(s.seen, messages...)
	if s.seenTools == nil {
		s.seenTools = tools
	}
	if s.idx < len(s.script) {
		r := s.script[s.idx]
		s.idx++
		return r, nil
	}
	return s.fallback, nil
}

func toolCall(id, name string, args map[string]any) ToolCall {
	raw, _ := json.Marshal(args)
	return ToolCall{ID: id, Name: name, Arguments: string(raw)}
}

// (a) Read-only role: read_patterns scoped to one subject, write_patterns empty.
// The model reads its subject (ok), tries to read another subject (denied),
// tries to write (denied), then finishes. Asserts the foreign content never
// reached the model and no change was recorded.
func TestRun_ReadOnlyRoleStaysInReadScope(t *testing.T) {
	kb := newMemKB(map[string]string{
		"subjects/subj1/profile.md": "subj1 is a backend engineer",
		"subjects/subj2/profile.md": "SUBJ2-SECRET-CONTENT",
	})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolReadNote, map[string]any{"path": "subjects/subj1/profile.md"})}, PromptTokens: 10, CompletionTokens: 5},
			{ToolCalls: []ToolCall{toolCall("2", toolReadNote, map[string]any{"path": "subjects/subj2/profile.md"})}, PromptTokens: 10, CompletionTokens: 5},
			{
				ToolCalls:        []ToolCall{toolCall("3", toolWriteNote, map[string]any{"path": "subjects/subj1/profile.md", "content": "tampered"})},
				PromptTokens:     10,
				CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("4", toolFinish, map[string]any{"answer": "subj1 is a backend engineer"})}, PromptTokens: 10, CompletionTokens: 5},
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:   "Answer a question about the subject from their profile.",
		ReadPatterns:  []string{"subjects/subj1/**"},
		WritePatterns: nil,
		Model:         "test-model",
		MaxTokens:     10000,
		MaxSteps:      10,
		LLM:           llm,
		KB:            kb,
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if res.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", res.Status)
	}
	if len(res.Changes) != 0 {
		t.Fatalf("read-only role must produce no changes, got %+v", res.Changes)
	}
	// The foreign subject's content must never have been delivered to the model.
	for _, m := range llm.seen {
		if strings.Contains(m.Content, "SUBJ2-SECRET-CONTENT") {
			t.Fatalf("out-of-scope content leaked to the model: %q", m.Content)
		}
	}
	// Both the foreign read and the write must be recorded as denials.
	if len(res.Denials) != 2 {
		t.Fatalf("expected 2 denials (foreign read + write), got %v", res.Denials)
	}
}

// (b) Write role: read+write scoped to one subject. The model reads a
// transcript, files two segments in scope (ok), tries to write outside scope
// (denied). Asserts only in-scope writes were recorded.
func TestRun_WriteRoleStaysInWriteScope(t *testing.T) {
	kb := newMemKB(map[string]string{
		"subjects/subj1/transcript.md": "topic A ... topic B ...",
	})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolReadNote, map[string]any{"path": "subjects/subj1/transcript.md"})}, PromptTokens: 10, CompletionTokens: 5},
			{
				ToolCalls:        []ToolCall{toolCall("2", toolWriteNote, map[string]any{"path": "subjects/subj1/segments/a.md", "content": "topic A"})},
				PromptTokens:     10,
				CompletionTokens: 5,
			},
			{
				ToolCalls:        []ToolCall{toolCall("3", toolWriteNote, map[string]any{"path": "subjects/subj1/segments/b.md", "content": "topic B"})},
				PromptTokens:     10,
				CompletionTokens: 5,
			},
			{
				ToolCalls:        []ToolCall{toolCall("4", toolWriteNote, map[string]any{"path": "subjects/subj2/segments/x.md", "content": "leak"})},
				PromptTokens:     10,
				CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("5", toolFinish, map[string]any{"answer": "filed 2 segments"})}, PromptTokens: 10, CompletionTokens: 5},
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:   "Segment the transcript by topic and file each segment.",
		ReadPatterns:  []string{"subjects/subj1/**"},
		WritePatterns: []string{"subjects/subj1/**"},
		Model:         "test-model",
		MaxTokens:     10000,
		MaxSteps:      10,
		LLM:           llm,
		KB:            kb,
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if len(res.Changes) != 2 {
		t.Fatalf("expected 2 in-scope changes, got %+v", res.Changes)
	}
	for _, c := range res.Changes {
		if !strings.HasPrefix(c.Path, "subjects/subj1/") {
			t.Fatalf("change outside write scope persisted: %s", c.Path)
		}
	}
	if _, leaked := kb.docs["subjects/subj2/segments/x.md"]; leaked {
		t.Fatalf("out-of-scope write reached the KB")
	}
	if len(res.Denials) != 1 {
		t.Fatalf("expected 1 write denial, got %v", res.Denials)
	}
}

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
		Instruction: "x", ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		Model: "m", MaxTokens: 10000, MaxSteps: 10, LLM: llm, KB: kb,
	})
	require.NoError(t, err)
	require.Empty(t, res.Changes)
	require.Equal(t, "@status:todo", kb.docs["other/x.md"])
	require.Equal(t, []string{"patch other/x.md"}, res.Denials)
}

// (f) Tools allowlist: when Input.Tools is ["search","finish"], the tool list
// passed to the LLM must contain only those two tools; write_note, patch_note,
// and read_note must be absent. An empty Input.Tools must expose the full
// default set (backward-compat).
func TestRun_ToolsAllowlistRestrictsLLMTools(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/a.md": "hello"})

	t.Run("restricted to search+finish", func(t *testing.T) {
		llm := &stubLLM{
			script: []ChatResult{
				{ToolCalls: []ToolCall{toolCall("1", toolSearch, map[string]any{"query": "hello"})}, PromptTokens: 10, CompletionTokens: 5},
				{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})}, PromptTokens: 10, CompletionTokens: 5},
			},
		}
		res, err := Run(context.Background(), Input{
			Instruction:  "search only",
			ReadPatterns: []string{"notes/**"},
			Model:        "m", MaxTokens: 10000, MaxSteps: 10,
			Tools: []string{toolSearch, toolFinish},
			LLM:   llm, KB: kb,
		})
		require.NoError(t, err)
		require.Equal(t, StatusCompleted, res.Status)

		offered := make(map[string]bool, len(llm.seenTools))
		for _, td := range llm.seenTools {
			offered[td.Name] = true
		}
		require.True(t, offered[toolSearch], "search must be offered")
		require.True(t, offered[toolFinish], "finish must always be offered")
		require.False(t, offered[toolWriteNote], "write_note must NOT be offered when not in allowlist")
		require.False(t, offered[toolPatchNote], "patch_note must NOT be offered when not in allowlist")
		require.False(t, offered[toolReadNote], "read_note must NOT be offered when not in allowlist")
		require.Len(t, llm.seenTools, 2, "exactly 2 tools should be offered")
	})

	t.Run("empty Tools exposes full default set", func(t *testing.T) {
		llm := &stubLLM{
			script: []ChatResult{
				{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "done"})}, PromptTokens: 5, CompletionTokens: 5},
			},
		}
		_, err := Run(context.Background(), Input{
			Instruction:  "all tools",
			ReadPatterns: []string{"notes/**"},
			Model:        "m", MaxTokens: 10000, MaxSteps: 10,
			Tools: nil, // empty = back-compat full set
			LLM:   llm, KB: kb,
		})
		require.NoError(t, err)
		require.Len(t, llm.seenTools, 5, "empty Tools must expose all 5 default tools")
	})
}

// (f2) Tools allowlist at execution time: when Input.Tools=["search","finish"],
// a model call to write_note must be rejected (denial recorded, no change
// persisted). This guards the trust boundary — allowlist at execution not just
// advertisement. See SECURITY finding F3.
func TestRun_ToolsAllowlistBlocksUndeclaredToolAtExecution(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/a.md": "hello"})

	llm := &stubLLM{
		script: []ChatResult{
			// Model calls write_note even though it was NOT offered (jailbreak / hallucination).
			{
				ToolCalls:        []ToolCall{toolCall("1", toolWriteNote, map[string]any{"path": "notes/a.md", "content": "INJECTED"})},
				PromptTokens:     10,
				CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})}, PromptTokens: 10, CompletionTokens: 5},
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:   "search only",
		ReadPatterns:  []string{"notes/**"},
		WritePatterns: []string{"notes/**"}, // write scope exists but tool not allowed
		Model:         "m", MaxTokens: 10000, MaxSteps: 10,
		Tools: []string{toolSearch, toolFinish}, // write_note not in allowlist
		LLM:   llm, KB: kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)
	// The write must be denied, not executed.
	require.Empty(t, res.Changes, "undeclared write_note call must produce no change")
	require.Len(t, res.Denials, 1, "undeclared tool call must be recorded as denial")
	// The original content must be untouched.
	require.Equal(t, "hello", kb.docs["notes/a.md"], "KB must not be mutated by denied tool")
}

// G9 regression: write_note must emit AgentChangeKindWrite (not empty string).
func TestRun_WriteNoteKindIsExplicit(t *testing.T) {
	kb := newMemKB(map[string]string{})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolWriteNote, map[string]any{
				"path": "notes/a.md", "content": "hello",
			})}, PromptTokens: 10, CompletionTokens: 5},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})},
				PromptTokens: 5, CompletionTokens: 5},
		},
	}
	res, err := Run(context.Background(), Input{
		Instruction:   "write a note",
		ReadPatterns:  []string{"notes/**"},
		WritePatterns: []string{"notes/**"},
		Model:         "m", MaxTokens: 10000, MaxSteps: 10,
		LLM: llm, KB: kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)
	require.Len(t, res.Changes, 1)
	require.Equal(t, webhookutil.AgentChangeKindWrite, res.Changes[0].Kind,
		"write_note must set AgentChangeKindWrite, not emit empty Kind")
}

// (c) Hard-cap: the model loops forever (always a tool call). A low MaxTokens
// with a high MaxSteps must stop the loop via the cap, not the step guard.
func TestRun_TokenHardCapStopsLoop(t *testing.T) {
	kb := newMemKB(map[string]string{"subjects/subj1/profile.md": "x"})
	llm := &stubLLM{
		fallback: ChatResult{
			ToolCalls:        []ToolCall{toolCall("loop", toolReadNote, map[string]any{"path": "subjects/subj1/profile.md"})},
			PromptTokens:     100,
			CompletionTokens: 0,
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:   "Loop.",
		ReadPatterns:  []string{"subjects/subj1/**"},
		WritePatterns: nil,
		Model:         "test-model",
		MaxTokens:     250,
		MaxSteps:      1000,
		LLM:           llm,
		KB:            kb,
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if res.Status != StatusCapped {
		t.Fatalf("expected capped status, got %s", res.Status)
	}
	if res.TokensUsed < 250 {
		t.Fatalf("cap should trip only after exceeding budget, used %d", res.TokensUsed)
	}
	if res.Steps >= 1000 {
		t.Fatalf("loop stopped by step guard, not the cap (steps=%d)", res.Steps)
	}
}

// TestRun_MaxStepsExhaustionStatus asserts the step guard: a model that keeps
// calling tools without ever finishing stops at MaxSteps with StatusMaxSteps.
func TestRun_MaxStepsExhaustionStatus(t *testing.T) {
	kb := newMemKB(map[string]string{"subjects/subj1/profile.md": "x"})
	llm := &stubLLM{
		fallback: ChatResult{
			ToolCalls:        []ToolCall{toolCall("loop", toolReadNote, map[string]any{"path": "subjects/subj1/profile.md"})},
			PromptTokens:     1,
			CompletionTokens: 1,
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:  "Loop forever.",
		ReadPatterns: []string{"subjects/subj1/**"},
		Model:        "test-model",
		MaxTokens:    100000,
		MaxSteps:     3,
		LLM:          llm,
		KB:           kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusMaxSteps, res.Status)
	require.Equal(t, 3, res.Steps)
}

// budgetRecordingLLM wraps stubLLM and records the per-call remaining budget
// passed through the BudgetedLLM hook.
type budgetRecordingLLM struct {
	stubLLM
	budgets []int
}

func (b *budgetRecordingLLM) ChatWithBudget(
	ctx context.Context, model string, messages []Message, tools []ToolDef, maxCompletionTokens int,
) (ChatResult, error) {
	b.budgets = append(b.budgets, maxCompletionTokens)
	return b.Chat(ctx, model, messages, tools)
}

// TestRun_PassesRemainingBudgetToBudgetedLLM asserts the loop hands each call
// the run's remaining token budget (MaxTokens minus tokens already spent).
func TestRun_PassesRemainingBudgetToBudgetedLLM(t *testing.T) {
	kb := newMemKB(nil)
	llm := &budgetRecordingLLM{
		stubLLM: stubLLM{
			script: []ChatResult{
				{ToolCalls: []ToolCall{toolCall("1", toolSearch, map[string]any{"query": "x"})}, PromptTokens: 100, CompletionTokens: 50},
				{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})}, PromptTokens: 10, CompletionTokens: 5},
			},
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:  "Do a thing.",
		ReadPatterns: []string{"**"},
		Model:        "test-model",
		MaxTokens:    1000,
		MaxSteps:     10,
		LLM:          llm,
		KB:           kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)
	require.Equal(t, []int{1000, 850}, llm.budgets)
}

// TestRun_InputBagRidesAsFleetInputMessage asserts InputBag is delivered as a
// system message named "fleet_input" (the codellm $FLEET_INPUT seam), inserted
// between the system prompt and the "Begin." user turn, and never scanned as a
// role instruction.
func TestRun_InputBagRidesAsFleetInputMessage(t *testing.T) {
	kb := newMemKB(nil)
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "ok"})}, PromptTokens: 1, CompletionTokens: 1},
		},
	}
	bag := []byte(`{"changed_files":[{"path":"a.md"}],"depth":1}`)
	res, err := Run(context.Background(), Input{
		Instruction: "```bash\necho hi\n```",
		Model:       "m", MaxTokens: 1000, MaxSteps: 5,
		InputBag: bag,
		LLM:      llm, KB: kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)

	// The first batch the LLM saw: [system prompt, fleet_input, user "Begin."].
	require.GreaterOrEqual(t, len(llm.seen), 3)
	require.Equal(t, RoleSystem, llm.seen[0].Role)
	require.Empty(t, llm.seen[0].Name)
	require.Equal(t, RoleSystem, llm.seen[1].Role)
	require.Equal(t, "fleet_input", llm.seen[1].Name)
	require.Equal(t, string(bag), llm.seen[1].Content)
	require.Equal(t, RoleUser, llm.seen[2].Role)
	require.Equal(t, "Begin.", llm.seen[2].Content)
}

// TestRun_NoInputBagOmitsFleetInputMessage asserts that without an InputBag no
// fleet_input message is injected (real-LLM roles are unchanged).
func TestRun_NoInputBagOmitsFleetInputMessage(t *testing.T) {
	kb := newMemKB(nil)
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "ok"})}, PromptTokens: 1, CompletionTokens: 1},
		},
	}
	_, err := Run(context.Background(), Input{
		Instruction: "hi", Model: "m", MaxTokens: 1000, MaxSteps: 5, LLM: llm, KB: kb,
	})
	require.NoError(t, err)
	for _, m := range llm.seen {
		require.NotEqual(t, "fleet_input", m.Name, "no fleet_input message without InputBag")
	}
}

// TestRun_CodeStyleSingleTurnAppliesToolCalls asserts the codellm shape: one
// completion returns write_note + patch_note + finish, they are applied through
// ScopedKB in a SINGLE loop turn (always-finish invariant → Steps == 1) even
// though MaxSteps allows more.
func TestRun_CodeStyleSingleTurnAppliesToolCalls(t *testing.T) {
	kb := newMemKB(map[string]string{"boards/b.md": "@status:todo"})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{
				toolCall("1", toolWriteNote, map[string]any{"path": "boards/new.md", "content": "hello"}),
				toolCall("2", toolPatchNote, map[string]any{"path": "boards/b.md", "find": "@status:todo", "replace": "@status:done"}),
				toolCall("3", toolFinish, map[string]any{"answer": "done"}),
			}, PromptTokens: 0, CompletionTokens: 0},
		},
	}
	res, err := Run(context.Background(), Input{
		Instruction:   "```python\n...\n```",
		WritePatterns: []string{"boards/**"},
		Model:         "codellm", MaxTokens: 1000, MaxSteps: 25,
		InputBag:      []byte(`{"depth":1}`),
		HardFailApply: true,
		LLM:           llm, KB: kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)
	require.Equal(t, 1, res.Steps, "always-finish stops the loop in one turn")
	require.Len(t, res.Changes, 2)
	require.Equal(t, "hello", kb.docs["boards/new.md"])
	require.Equal(t, "@status:done", kb.docs["boards/b.md"])
	require.Equal(t, "done", res.Answer)
}

// TestRun_HardFailApply_BadPatchFailsRun asserts the executor:code/codellm
// semantics: a patch whose find is absent is a genuine apply error, and with
// HardFailApply set the whole run fails (all-or-nothing, matching RunCode).
func TestRun_HardFailApply_BadPatchFailsRun(t *testing.T) {
	kb := newMemKB(map[string]string{"boards/b.md": "content without the token"})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{
				toolCall("1", toolPatchNote, map[string]any{"path": "boards/b.md", "find": "MISSING", "replace": "x"}),
				toolCall("2", toolFinish, map[string]any{"answer": "done"}),
			}, PromptTokens: 0, CompletionTokens: 0},
		},
	}
	res, err := Run(context.Background(), Input{
		Instruction:   "```python\n...\n```",
		WritePatterns: []string{"boards/**"},
		Model:         "codellm", MaxTokens: 1000, MaxSteps: 25,
		HardFailApply: true,
		LLM:           llm, KB: kb,
	})
	require.Error(t, err, "a failed apply must fail the run when HardFailApply is set")
	require.Nil(t, res)
	require.Contains(t, err.Error(), "apply patch_note")
}

// TestRun_SoftApply_RealLLMSelfCorrects asserts the DEFAULT (real-LLM) path is
// UNCHANGED: the same bad-find patch is a soft, self-correctable tool result —
// the model sees the error, retries with a valid find, and the run completes.
func TestRun_SoftApply_RealLLMSelfCorrects(t *testing.T) {
	kb := newMemKB(map[string]string{"boards/b.md": "@status:todo"})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolPatchNote, map[string]any{
				"path": "boards/b.md", "find": "MISSING", "replace": "x",
			})}, PromptTokens: 5, CompletionTokens: 5},
			{ToolCalls: []ToolCall{toolCall("2", toolPatchNote, map[string]any{
				"path": "boards/b.md", "find": "@status:todo", "replace": "@status:done",
			})}, PromptTokens: 5, CompletionTokens: 5},
			{ToolCalls: []ToolCall{toolCall("3", toolFinish, map[string]any{"answer": "recovered"})}, PromptTokens: 5, CompletionTokens: 5},
		},
	}
	res, err := Run(context.Background(), Input{
		Instruction:   "fix it",
		WritePatterns: []string{"boards/**"},
		Model:         "gpt-4o", MaxTokens: 1000, MaxSteps: 25,
		// HardFailApply defaults false for real-LLM roles.
		LLM: llm, KB: kb,
	})
	require.NoError(t, err, "real-LLM roles keep soft, self-correctable apply errors")
	require.Equal(t, StatusCompleted, res.Status)
	require.Equal(t, "@status:done", kb.docs["boards/b.md"])
	require.Len(t, res.Changes, 1, "only the successful retry is recorded")
	require.Equal(t, "recovered", res.Answer)
}
