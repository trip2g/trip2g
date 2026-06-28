package agentruntime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
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

var errNotFound = &kbError{"not found"}

type kbError struct{ msg string }

func (e *kbError) Error() string { return e.msg }

// stubLLM returns a scripted sequence of responses, then a fixed fallback. It
// records every message batch it receives so a test can assert what the model
// was (or was not) shown.
type stubLLM struct {
	script   []ChatResult
	fallback ChatResult
	idx      int
	seen     []Message
}

func (s *stubLLM) Chat(_ context.Context, _ string, messages []Message, _ []ToolDef) (ChatResult, error) {
	s.seen = append(s.seen, messages...)
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
			{ToolCalls: []ToolCall{toolCall("3", toolWriteNote, map[string]any{"path": "subjects/subj1/profile.md", "content": "tampered"})}, PromptTokens: 10, CompletionTokens: 5},
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
			{ToolCalls: []ToolCall{toolCall("2", toolWriteNote, map[string]any{"path": "subjects/subj1/segments/a.md", "content": "topic A"})}, PromptTokens: 10, CompletionTokens: 5},
			{ToolCalls: []ToolCall{toolCall("3", toolWriteNote, map[string]any{"path": "subjects/subj1/segments/b.md", "content": "topic B"})}, PromptTokens: 10, CompletionTokens: 5},
			{ToolCalls: []ToolCall{toolCall("4", toolWriteNote, map[string]any{"path": "subjects/subj2/segments/x.md", "content": "leak"})}, PromptTokens: 10, CompletionTokens: 5},
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
