package agentruntime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// execStub is a fake exec endpoint (codellm) recording the request it received
// and returning a canned tool_calls completion or an error.
type execStub struct {
	res ChatResult
	err error

	calls    int
	model    string
	messages []Message
}

func (e *execStub) Chat(_ context.Context, model string, messages []Message, _ []ToolDef) (ChatResult, error) {
	e.calls++
	e.model = model
	e.messages = messages
	return e.res, e.err
}

// runWithExec drives Run with a scripted main LLM that first calls
// exec(program, code) and then finishes, against the given exec endpoint.
func runWithExec(t *testing.T, kb *memKB, writePatterns []string, program, code string, exec LLM) (*Result, *stubLLM) {
	t.Helper()
	llm := &stubLLM{
		script: []ChatResult{
			{
				ToolCalls: []ToolCall{toolCall("1", toolExec, map[string]any{
					"program": program,
					"code":    code,
				})},
				PromptTokens: 10, CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})},
				PromptTokens: 5, CompletionTokens: 5},
		},
	}
	res, err := Run(context.Background(), Input{
		Instruction:   "run some code",
		ReadPatterns:  []string{"**"},
		WritePatterns: writePatterns,
		Model:         "m",
		MaxTokens:     10000,
		MaxSteps:      5,
		ExecLLM:       exec,
		LLM:           llm,
		KB:            kb,
	})
	require.NoError(t, err)
	return res, llm
}

// TestRun_ExecToolGatedByExecLLM asserts:
//  1. exec is NOT in the offered tool set when ExecLLM is nil.
//  2. exec IS in the offered set when ExecLLM is non-nil.
func TestRun_ExecToolGatedByExecLLM(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/a.md": "content"})
	finishOnly := func() *stubLLM {
		return &stubLLM{
			script: []ChatResult{
				{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "done"})},
					PromptTokens: 5, CompletionTokens: 5},
			},
		}
	}

	t.Run("absent when ExecLLM nil", func(t *testing.T) {
		llm := finishOnly()
		_, err := Run(context.Background(), Input{
			Instruction:   "x",
			ReadPatterns:  []string{"**"},
			WritePatterns: []string{"**"},
			Model:         "m",
			MaxTokens:     10000,
			MaxSteps:      5,
			ExecLLM:       nil, // nil → exec disabled
			LLM:           llm,
			KB:            kb,
		})
		require.NoError(t, err)
		for _, td := range llm.seenTools {
			require.NotEqual(t, toolExec, td.Name,
				"exec must NOT be offered when ExecLLM is nil")
		}
	})

	t.Run("present when ExecLLM non-nil", func(t *testing.T) {
		llm := finishOnly()
		_, err := Run(context.Background(), Input{
			Instruction:   "x",
			ReadPatterns:  []string{"**"},
			WritePatterns: []string{"**"},
			Model:         "m",
			MaxTokens:     10000,
			MaxSteps:      5,
			ExecLLM:       &execStub{},
			LLM:           llm,
			KB:            kb,
		})
		require.NoError(t, err)
		found := false
		for _, td := range llm.seenTools {
			if td.Name == toolExec {
				found = true
			}
		}
		require.True(t, found, "exec must be offered when ExecLLM is non-nil")
	})
}

// TestRun_ExecToolRoutesThroughExecLLM asserts the exec tool sends the code as
// a one-fenced-block chat completion (fence label = program name) to ExecLLM
// and applies the returned write_note tool call through the scoped KB.
func TestRun_ExecToolRoutesThroughExecLLM(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolWriteNote, map[string]any{"path": "notes/exec.md", "content": "from exec"}),
			toolCall("c1", toolFinish, map[string]any{"answer": "wrote"}),
		}},
	}

	res, llm := runWithExec(t, kb, []string{"notes/**"}, "bash", "echo hi", exec)

	require.Equal(t, StatusCompleted, res.Status)
	require.Equal(t, "from exec", kb.docs["notes/exec.md"], "exec tool write must land in KB")
	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/exec.md", res.Changes[0].Path)

	// Wire shape: exactly one call, one user message, code fenced under the
	// program-name label.
	require.Equal(t, 1, exec.calls)
	require.Len(t, exec.messages, 1)
	require.Equal(t, RoleUser, exec.messages[0].Role)
	require.Equal(t, "```bash\necho hi\n```", exec.messages[0].Content)

	// The model gets the same summary shape as the old in-process exec.
	var summary string
	for _, m := range llm.seen {
		if m.Role == RoleTool && m.Name == toolExec {
			summary = m.Content
		}
	}
	require.Equal(t, "ok: ran bash, 1 write(s); wrote", summary)
}

// TestRun_ExecToolAppliesPatch asserts a patch_note tool call from the exec
// endpoint is applied as a patch through the scoped KB.
func TestRun_ExecToolAppliesPatch(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/p.md": "status: todo"})
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolPatchNote, map[string]any{"path": "notes/p.md", "find": "todo", "replace": "done"}),
			toolCall("c1", toolFinish, map[string]any{"answer": ""}),
		}},
	}

	res, _ := runWithExec(t, kb, []string{"notes/**"}, "python", "patch()", exec)

	require.Equal(t, "status: done", kb.docs["notes/p.md"])
	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/p.md", res.Changes[0].Path)
}

// TestRun_ExecToolDeniedOutOfScope asserts a write returned by the exec
// endpoint targeting a path outside write_patterns is denied (recorded in
// res.Denials) and never reaches the KB.
func TestRun_ExecToolDeniedOutOfScope(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", toolWriteNote, map[string]any{"path": "secret/x.md", "content": "leak"}),
			toolCall("c1", toolFinish, map[string]any{"answer": "tried"}),
		}},
	}

	res, _ := runWithExec(t, kb, []string{"notes/**"}, "bash", "echo leak", exec)

	require.Empty(t, res.Changes, "out-of-scope exec write must not appear in Changes")
	require.Len(t, res.Denials, 1)
	require.Contains(t, res.Denials[0], "secret/x.md")
	_, leaked := kb.docs["secret/x.md"]
	require.False(t, leaked, "out-of-scope path must not appear in KB")
}

// TestRun_ExecToolEndpointError asserts an error from the exec endpoint (e.g.
// codellm's deterministic 422 for a disallowed program or a failing block) is
// surfaced to the model as a soft tool error, producing no changes.
func TestRun_ExecToolEndpointError(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{err: errors.New(`program "node" not in --allowed-programs [bash]`)}

	res, llm := runWithExec(t, kb, []string{"**"}, "node", "console.log(1)", exec)

	require.Empty(t, res.Changes, "failed exec must not produce changes")
	var gotError bool
	for _, msg := range llm.seen {
		if strings.Contains(msg.Content, "not in --allowed-programs") {
			gotError = true
		}
	}
	require.True(t, gotError, "model must receive the exec endpoint's error")
}

// TestRun_ExecToolRejectsFenceInCode asserts code containing a ``` fence
// marker is rejected up front (it cannot ride the one-block wire protocol)
// without calling the exec endpoint.
func TestRun_ExecToolRejectsFenceInCode(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{}

	res, llm := runWithExec(t, kb, []string{"**"}, "bash", "cat <<EOF\n```md\nEOF\n", exec)

	require.Empty(t, res.Changes)
	require.Equal(t, 0, exec.calls, "exec endpoint must not be called")
	var gotError bool
	for _, msg := range llm.seen {
		if msg.Role == RoleTool && strings.HasPrefix(msg.Content, "error:") {
			gotError = true
		}
	}
	require.True(t, gotError, "model must receive a soft error")
}

// TestRun_ExecToolRejectsBadProgram asserts an empty or non-token program name
// is rejected before any endpoint call.
func TestRun_ExecToolRejectsBadProgram(t *testing.T) {
	for _, program := range []string{"", "ba sh", "bash\nrm"} {
		kb := newMemKB(nil)
		exec := &execStub{}

		res, _ := runWithExec(t, kb, []string{"**"}, program, "echo hi", exec)

		require.Empty(t, res.Changes, "program %q must not produce changes", program)
		require.Equal(t, 0, exec.calls, "program %q must not reach the endpoint", program)
	}
}

// TestRun_ExecToolUnexpectedToolCall asserts a tool call other than
// write_note/patch_note/finish coming back from the exec endpoint is a soft
// error, not a silent skip.
func TestRun_ExecToolUnexpectedToolCall(t *testing.T) {
	kb := newMemKB(nil)
	exec := &execStub{
		res: ChatResult{ToolCalls: []ToolCall{
			toolCall("c0", "read_note", map[string]any{"path": "notes/a.md"}),
		}},
	}

	res, llm := runWithExec(t, kb, []string{"**"}, "bash", "echo hi", exec)

	require.Empty(t, res.Changes)
	var gotError bool
	for _, msg := range llm.seen {
		if strings.Contains(msg.Content, "unexpected tool call") {
			gotError = true
		}
	}
	require.True(t, gotError)
}

// TestRun_DefaultToolCountUnchangedWithoutExecLLM asserts that with ExecLLM
// nil, the default tool set is still exactly 5 (backward-compat).
func TestRun_DefaultToolCountUnchangedWithoutExecLLM(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/a.md": "x"})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "done"})},
				PromptTokens: 5, CompletionTokens: 5},
		},
	}
	_, err := Run(context.Background(), Input{
		Instruction:  "x",
		ReadPatterns: []string{"**"},
		Model:        "m",
		MaxTokens:    10000,
		MaxSteps:     5,
		ExecLLM:      nil,
		LLM:          llm,
		KB:           kb,
	})
	require.NoError(t, err)
	require.Len(t, llm.seenTools, 5,
		"ExecLLM=nil must not add exec to the offered set")
}
