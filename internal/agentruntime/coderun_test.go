package agentruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunCode_EndToEnd runs a bash script that emits a write change and asserts
// the change is applied via ScopedKB.
func TestRunCode_EndToEnd(t *testing.T) {
	skipIfSandboxUnsupported(t)
	kb := newMemKB(nil)
	body := "```bash\necho '{\"changes\":[{\"path\":\"notes/a.md\",\"content\":\"generated\"}],\"answer\":\"ok\"}'\n```"

	res, err := RunCode(context.Background(), CodeInput{
		Body:            body,
		WritePatterns:   []string{"notes/**"},
		KB:              kb,
		AllowedPrograms: []string{"bash"},
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)
	require.Equal(t, 1, res.Steps)
	require.Equal(t, 0, res.TokensUsed)
	require.Equal(t, "ok", res.Answer)
	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/a.md", res.Changes[0].Path)
	require.Equal(t, "generated", kb.docs["notes/a.md"])
}

// TestRunCode_ScopeEnforcement asserts writes outside write_patterns are denied
// and not applied — exec/code is NOT a scope bypass.
func TestRunCode_ScopeEnforcement(t *testing.T) {
	skipIfSandboxUnsupported(t)
	kb := newMemKB(nil)
	// Script tries to write two files: one in scope and one out of scope.
	script := `echo '{"changes":[{"path":"notes/in.md","content":"in"},{"path":"other/out.md","content":"out"}],"answer":"tried"}'`
	body := "```bash\n" + script + "\n```"

	res, err := RunCode(context.Background(), CodeInput{
		Body:            body,
		WritePatterns:   []string{"notes/**"},
		KB:              kb,
		AllowedPrograms: []string{"bash"},
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)
	require.Len(t, res.Changes, 1, "only in-scope write must succeed")
	require.Equal(t, "notes/in.md", res.Changes[0].Path)
	require.Len(t, res.Denials, 1, "out-of-scope write must be recorded as a denial")
	require.Contains(t, res.Denials[0], "other/out.md")
	_, leaked := kb.docs["other/out.md"]
	require.False(t, leaked, "out-of-scope path must not appear in KB")
}

// TestRunCode_ProgramNotInAllowlist asserts a program not in allowed_programs
// is rejected before execution.
func TestRunCode_ProgramNotInAllowlist(t *testing.T) {
	kb := newMemKB(nil)
	body := "```python\nprint('{\"changes\":[],\"answer\":\"hi\"}')\n```"

	_, err := RunCode(context.Background(), CodeInput{
		Body:            body,
		WritePatterns:   []string{"notes/**"},
		KB:              kb,
		AllowedPrograms: []string{"bash"}, // python not allowed
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in --allowed-programs")
}

// TestRunCode_AllowlistEmpty asserts code execution is disabled when
// AllowedPrograms is empty (the default off-by-default state).
func TestRunCode_AllowlistEmpty(t *testing.T) {
	kb := newMemKB(nil)
	body := "```bash\necho '{\"changes\":[],\"answer\":\"hi\"}'\n```"

	_, err := RunCode(context.Background(), CodeInput{
		Body:            body,
		WritePatterns:   []string{"notes/**"},
		KB:              kb,
		AllowedPrograms: nil, // empty = disabled
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "disabled")
}

// TestRunCode_NoFencedBlock asserts RunCode fails cleanly when the body has no
// fenced code block.
func TestRunCode_NoFencedBlock(t *testing.T) {
	kb := newMemKB(nil)
	_, err := RunCode(context.Background(), CodeInput{
		Body:            "just text, no code block",
		WritePatterns:   []string{"**"},
		KB:              kb,
		AllowedPrograms: []string{"bash"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no fenced code block")
}

// TestRunCode_Pipeline_TwoBlocks pipes block 1's free-form stdout into block 2's
// stdin; only the LAST block's stdout is parsed as the {changes,answer} contract.
func TestRunCode_Pipeline_TwoBlocks(t *testing.T) {
	skipIfSandboxUnsupported(t)
	kb := newMemKB(nil)
	body := "```bash\necho world\n```\n" +
		"```bash\nread -r v; echo \"{\\\"changes\\\":[{\\\"path\\\":\\\"notes/p.md\\\",\\\"content\\\":\\\"$v\\\"}],\\\"answer\\\":\\\"got $v\\\"}\"\n```"

	res, err := RunCode(context.Background(), CodeInput{
		Body:            body,
		WritePatterns:   []string{"notes/**"},
		KB:              kb,
		AllowedPrograms: []string{"bash"},
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)
	require.Equal(t, 2, res.Steps, "Steps must equal number of blocks run")
	require.Equal(t, "got world", res.Answer)
	require.Equal(t, "world", kb.docs["notes/p.md"])
}

// TestRunCode_Pipeline_MixedLanguages pipes bash stdout into a python block:
// each block resolves its own interpreter from its fence language.
func TestRunCode_Pipeline_MixedLanguages(t *testing.T) {
	skipIfSandboxUnsupported(t)
	kb := newMemKB(nil)
	body := "```bash\necho 41\n```\n" +
		"```python\nimport sys, json\nn = int(sys.stdin.read()) + 1\nprint(json.dumps({\"changes\": [], \"answer\": str(n)}))\n```"

	res, err := RunCode(context.Background(), CodeInput{
		Body:            body,
		WritePatterns:   []string{"notes/**"},
		KB:              kb,
		AllowedPrograms: []string{"bash", "python"},
	})
	require.NoError(t, err)
	require.Equal(t, 2, res.Steps)
	require.Equal(t, "42", res.Answer)
}

// TestRunCode_Pipeline_MiddleBlockFails asserts a non-zero exit in the middle
// stops the pipeline, surfaces the failing block index + stderr, and applies
// no changes.
func TestRunCode_Pipeline_MiddleBlockFails(t *testing.T) {
	skipIfSandboxUnsupported(t)
	kb := newMemKB(nil)
	body := "```bash\necho start\n```\n" +
		"```bash\necho boom >&2; exit 1\n```\n" +
		"```bash\necho '{\"changes\":[{\"path\":\"notes/x.md\",\"content\":\"never\"}],\"answer\":\"never\"}'\n```"

	_, err := RunCode(context.Background(), CodeInput{
		Body:            body,
		WritePatterns:   []string{"notes/**"},
		KB:              kb,
		AllowedPrograms: []string{"bash"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "block 2/3")
	require.Contains(t, err.Error(), "boom")
	require.Empty(t, kb.docs, "no changes may be applied when the pipeline fails")
}

// TestRunCode_Pipeline_AllowlistCheckedUpfront asserts a disallowed program in
// a LATER block fails the run before block 1 executes (no partial side effects).
func TestRunCode_Pipeline_AllowlistCheckedUpfront(t *testing.T) {
	kb := newMemKB(nil)
	body := "```bash\necho start\n```\n```python\nprint('x')\n```"

	_, err := RunCode(context.Background(), CodeInput{
		Body:            body,
		WritePatterns:   []string{"notes/**"},
		KB:              kb,
		AllowedPrograms: []string{"bash"}, // python not allowed
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "block 2")
	require.Contains(t, err.Error(), "not in --allowed-programs")
}

// TestRunCode_Pipeline_InputBagAllBlocks asserts the delivery bag ($FLEET_INPUT)
// is visible to every block in the pipeline, not just the first.
func TestRunCode_Pipeline_InputBagAllBlocks(t *testing.T) {
	skipIfSandboxUnsupported(t)
	kb := newMemKB(nil)
	body := "```bash\ncat \"$FLEET_INPUT\"\n```\n" +
		"```bash\nread -r first; bag=$(cat \"$FLEET_INPUT\"); if [ \"$first\" = \"$bag\" ]; then a=same; else a=diff; fi; echo \"{\\\"changes\\\":[],\\\"answer\\\":\\\"$a\\\"}\"\n```"

	res, err := RunCode(context.Background(), CodeInput{
		Body:            body,
		WritePatterns:   []string{"notes/**"},
		KB:              kb,
		AllowedPrograms: []string{"bash"},
		Input:           []byte(`{"depth":7}`),
	})
	require.NoError(t, err)
	require.Equal(t, "same", res.Answer)
}

// TestRunCode_RequiresKB asserts RunCode returns an error when KB is nil.
func TestRunCode_RequiresKB(t *testing.T) {
	_, err := RunCode(context.Background(), CodeInput{
		Body:            "```bash\necho hi\n```",
		AllowedPrograms: []string{"bash"},
		KB:              nil,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "KB is required")
}

// TestRun_ExecToolGatedByAllowedPrograms asserts:
//  1. exec is NOT in the offered tool set when AllowedPrograms is empty.
//  2. exec IS in the offered set when AllowedPrograms is non-empty.
func TestRun_ExecToolGatedByAllowedPrograms(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/a.md": "content"})

	t.Run("absent when AllowedPrograms empty", func(t *testing.T) {
		llm := &stubLLM{
			script: []ChatResult{
				{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "done"})},
					PromptTokens: 5, CompletionTokens: 5},
			},
		}
		_, err := Run(context.Background(), Input{
			Instruction:     "x",
			ReadPatterns:    []string{"**"},
			WritePatterns:   []string{"**"},
			Model:           "m",
			MaxTokens:       10000,
			MaxSteps:        5,
			AllowedPrograms: nil, // empty → exec disabled
			LLM:             llm,
			KB:              kb,
		})
		require.NoError(t, err)
		for _, td := range llm.seenTools {
			require.NotEqual(t, toolExec, td.Name,
				"exec must NOT be offered when AllowedPrograms is empty")
		}
	})

	t.Run("present when AllowedPrograms non-empty", func(t *testing.T) {
		llm := &stubLLM{
			script: []ChatResult{
				{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "done"})},
					PromptTokens: 5, CompletionTokens: 5},
			},
		}
		_, err := Run(context.Background(), Input{
			Instruction:     "x",
			ReadPatterns:    []string{"**"},
			WritePatterns:   []string{"**"},
			Model:           "m",
			MaxTokens:       10000,
			MaxSteps:        5,
			AllowedPrograms: []string{"bash"},
			LLM:             llm,
			KB:              kb,
		})
		require.NoError(t, err)
		found := false
		for _, td := range llm.seenTools {
			if td.Name == toolExec {
				found = true
			}
		}
		require.True(t, found, "exec must be offered when AllowedPrograms is non-empty")
	})
}

// TestRun_ExecToolRunsAndApplies drives Run with a stubLLM that calls exec
// with a bash one-liner, then finishes. Asserts the write is applied via
// ScopedKB and recorded in res.Changes.
func TestRun_ExecToolRunsAndApplies(t *testing.T) {
	skipIfSandboxUnsupported(t)
	kb := newMemKB(nil)
	bashCode := `echo '{"changes":[{"path":"notes/exec.md","content":"from exec"}],"answer":"wrote"}'`

	llm := &stubLLM{
		script: []ChatResult{
			{
				ToolCalls: []ToolCall{toolCall("1", toolExec, map[string]any{
					"program": "bash",
					"code":    bashCode,
				})},
				PromptTokens: 10, CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})},
				PromptTokens: 5, CompletionTokens: 5},
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:     "run some code",
		ReadPatterns:    []string{"**"},
		WritePatterns:   []string{"notes/**"},
		Model:           "m",
		MaxTokens:       10000,
		MaxSteps:        5,
		AllowedPrograms: []string{"bash"},
		LLM:             llm,
		KB:              kb,
	})
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, res.Status)
	require.Equal(t, "from exec", kb.docs["notes/exec.md"],
		"exec tool write must land in KB")
	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/exec.md", res.Changes[0].Path)
}

// TestRun_ExecToolDeniedOutOfScope drives Run with a stubLLM that calls exec
// with a write targeting a path outside write_patterns. The write must be
// denied (recorded in res.Denials) and not reach the KB.
func TestRun_ExecToolDeniedOutOfScope(t *testing.T) {
	skipIfSandboxUnsupported(t)
	kb := newMemKB(nil)
	bashCode := `echo '{"changes":[{"path":"secret/x.md","content":"leak"}],"answer":"tried"}'`

	llm := &stubLLM{
		script: []ChatResult{
			{
				ToolCalls: []ToolCall{toolCall("1", toolExec, map[string]any{
					"program": "bash",
					"code":    bashCode,
				})},
				PromptTokens: 10, CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})},
				PromptTokens: 5, CompletionTokens: 5},
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:     "try to leak",
		ReadPatterns:    []string{"**"},
		WritePatterns:   []string{"notes/**"}, // secret/** not allowed
		Model:           "m",
		MaxTokens:       10000,
		MaxSteps:        5,
		AllowedPrograms: []string{"bash"},
		LLM:             llm,
		KB:              kb,
	})
	require.NoError(t, err)
	require.Empty(t, res.Changes, "out-of-scope exec write must not appear in Changes")
	require.Len(t, res.Denials, 1)
	require.Contains(t, res.Denials[0], "secret/x.md")
	_, leaked := kb.docs["secret/x.md"]
	require.False(t, leaked, "out-of-scope path must not appear in KB")
}

// TestRun_ExecToolProgramNotInAllowlist asserts that when the model calls exec
// with a program not in AllowedPrograms, the tool returns an error string and
// does not run any code.
func TestRun_ExecToolProgramNotInAllowlist(t *testing.T) {
	kb := newMemKB(nil)

	llm := &stubLLM{
		script: []ChatResult{
			{
				ToolCalls: []ToolCall{toolCall("1", toolExec, map[string]any{
					"program": "node", // not in allowlist
					"code":    `console.log(JSON.stringify({changes:[],answer:"hi"}))`,
				})},
				PromptTokens: 10, CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})},
				PromptTokens: 5, CompletionTokens: 5},
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:     "use node",
		ReadPatterns:    []string{"**"},
		WritePatterns:   []string{"**"},
		Model:           "m",
		MaxTokens:       10000,
		MaxSteps:        5,
		AllowedPrograms: []string{"bash"}, // only bash allowed
		LLM:             llm,
		KB:              kb,
	})
	require.NoError(t, err)
	require.Empty(t, res.Changes, "disallowed program must not produce changes")
	// The model should have received an error message about the program.
	var gotError bool
	for _, msg := range llm.seen {
		if strings.Contains(msg.Content, "not in allowed_programs") {
			gotError = true
		}
	}
	require.True(t, gotError, "model must receive error about disallowed program")
}

// TestRun_DefaultToolCountUnchangedWithoutAllowedPrograms asserts that with
// AllowedPrograms=nil, the default tool set is still exactly 5 (backward-compat).
func TestRun_DefaultToolCountUnchangedWithoutAllowedPrograms(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/a.md": "x"})
	llm := &stubLLM{
		script: []ChatResult{
			{ToolCalls: []ToolCall{toolCall("1", toolFinish, map[string]any{"answer": "done"})},
				PromptTokens: 5, CompletionTokens: 5},
		},
	}
	_, err := Run(context.Background(), Input{
		Instruction:     "x",
		ReadPatterns:    []string{"**"},
		Model:           "m",
		MaxTokens:       10000,
		MaxSteps:        5,
		AllowedPrograms: nil,
		LLM:             llm,
		KB:              kb,
	})
	require.NoError(t, err)
	require.Len(t, llm.seenTools, 5,
		"AllowedPrograms=nil must not add exec to the offered set")
}
