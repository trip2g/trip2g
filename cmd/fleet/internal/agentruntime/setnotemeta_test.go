package agentruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/webhookutil"
)

func runSetNoteMeta(t *testing.T, kb KB, writePatterns []string, args map[string]any) (*Result, string) {
	t.Helper()
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolSetNoteMeta, args)}, PromptTokens: 10, CompletionTokens: 5},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})}, PromptTokens: 10, CompletionTokens: 5},
		},
	}
	res, err := Run(context.Background(), Input{
		Instruction:   "mark it done",
		ReadPatterns:  []string{"**"},
		WritePatterns: writePatterns,
		Model:         "m",
		MaxTokens:     10000,
		MaxSteps:      5,
		LLM:           llm,
		KB:            kb,
	})
	require.NoError(t, err)
	var output string
	for _, m := range llm.seen {
		if m.Role == RoleTool {
			output = m.Content
		}
	}
	return res, output
}

func TestRun_SetNoteMeta(t *testing.T) {
	const note = "---\ntitle: Task # keep\nstatus: todo\n---\nBody\n"

	tests := []struct {
		name          string
		writePatterns []string
		args          map[string]any
		wantDoc       string
		wantOutput    string
		wantChanges   []webhookutil.AgentChange
		wantDenial    string
	}{
		{
			name:          "sets and unsets keys in scope",
			writePatterns: []string{"tasks/**"},
			args:          map[string]any{"path": "tasks/a.md", "set": map[string]any{"status": "done", "points": 3}, "unset": []string{"title"}},
			wantDoc:       "---\nstatus: done\npoints: 3\n---\nBody\n",
			wantOutput:    "ok: set meta of tasks/a.md",
			wantChanges: []webhookutil.AgentChange{{
				Path:  "tasks/a.md",
				Set:   map[string]any{"status": "done", "points": float64(3)},
				Unset: []string{"title"},
				Kind:  webhookutil.AgentChangeKindMeta,
			}},
		},
		{
			name:          "out of write scope is denied",
			writePatterns: []string{"other/**"},
			args:          map[string]any{"path": "tasks/a.md", "set": map[string]any{"status": "done"}},
			wantDoc:       note,
			wantOutput:    "error: " + ErrWriteDenied.Error(),
			wantDenial:    "set meta tasks/a.md",
		},
		{
			name:          "setting fleet_id is role authoring",
			writePatterns: []string{"**"},
			args:          map[string]any{"path": "tasks/a.md", "set": map[string]any{"fleet_id": "codellm"}},
			wantDoc:       note,
			wantOutput:    "error: " + ErrRoleAuthoringDenied.Error(),
			wantDenial:    "set meta tasks/a.md: role note (fleet_id) — agents may not author roles",
		},
		{
			name:          "nothing to change is invalid",
			writePatterns: []string{"**"},
			args:          map[string]any{"path": "tasks/a.md"},
			wantDoc:       note,
			wantOutput:    "error: set or unset must name at least one key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kb := newMemKB(map[string]string{"tasks/a.md": note})
			res, output := runSetNoteMeta(t, kb, tt.writePatterns, tt.args)

			require.Equal(t, tt.wantOutput, output)
			require.Equal(t, tt.wantDoc, kb.docs["tasks/a.md"])
			require.Equal(t, tt.wantChanges, res.Changes)
			if tt.wantDenial == "" {
				require.Empty(t, res.Denials)
			} else {
				require.Equal(t, []string{tt.wantDenial}, res.Denials)
			}
		})
	}
}

func TestRun_SetNoteMetaOnRoleNoteIsDenied(t *testing.T) {
	const role = "---\nfleet_id: codellm\nwrite_patterns: [\"notes/**\"]\n---\nbody\n"
	kb := newMemKB(map[string]string{"roles/r.md": role})

	res, _ := runSetNoteMeta(t, kb, []string{"**"}, map[string]any{
		"path": "roles/r.md", "set": map[string]any{"write_patterns": []string{"**"}},
	})

	require.Equal(t, role, kb.docs["roles/r.md"])
	require.Len(t, res.Denials, 1)
	require.Empty(t, res.Changes)
}

type conditionalMetaKB struct {
	KB
	gotHash   string
	gotCalled bool
	plainCall bool
}

func (c *conditionalMetaKB) SetMetaIfUnchanged(_ context.Context, _ string, _ map[string]any, _ []string, expectedHash string) error {
	c.gotCalled = true
	c.gotHash = expectedHash
	return nil
}

func (c *conditionalMetaKB) SetMeta(context.Context, string, map[string]any, []string) error {
	c.plainCall = true
	return nil
}

func TestScopedKB_SetMetaIsConditionalOnVerifiedBytes(t *testing.T) {
	const body = "---\na: 1\n---\n"
	kb := &conditionalMetaKB{KB: newMemKB(map[string]string{"notes/a.md": body})}
	scoped := NewScopedKB(kb, nil, []string{"**"})

	require.NoError(t, scoped.SetMeta(context.Background(), "/notes/a.md", map[string]any{"a": 2}, nil))

	require.True(t, kb.gotCalled)
	require.False(t, kb.plainCall)
	require.Equal(t, contentHash(body), kb.gotHash)
}

func TestScopedKB_SetMetaUnconditionalWhenGuardOff(t *testing.T) {
	kb := &conditionalMetaKB{KB: newMemKB(map[string]string{"notes/a.md": "---\na: 1\n---\n"})}
	scoped := NewScopedKB(kb, nil, []string{"**"})
	scoped.allowRoleAuthoring = true

	require.NoError(t, scoped.SetMeta(context.Background(), "notes/a.md", map[string]any{"fleet_id": "x"}, nil))

	require.False(t, kb.gotCalled)
	require.True(t, kb.plainCall)
}

func TestScopedKB_SetMetaUnverifiableNoteFailsClosed(t *testing.T) {
	inner := newMemKB(map[string]string{"notes/a.md": "---\na: 1\n---\n"})
	kb := &failingReadKB{KB: inner}
	scoped := NewScopedKB(kb, nil, []string{"**"})

	err := scoped.SetMeta(context.Background(), "notes/a.md", map[string]any{"a": 2}, nil)

	require.ErrorIs(t, err, ErrRoleGuardUnverifiable)
	require.Equal(t, "---\na: 1\n---\n", inner.docs["notes/a.md"])
}
