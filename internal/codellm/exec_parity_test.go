package codellm_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/agentruntime"
	"trip2g/internal/codellm"
	"trip2g/internal/coderun"
)

// execScriptLLM scripts the MAIN agent loop: first calls exec(program, code),
// then finishes. It records every message it is shown.
type execScriptLLM struct {
	program string
	code    string
	idx     int
	seen    []agentruntime.Message
}

func (s *execScriptLLM) Chat(_ context.Context, _ string, messages []agentruntime.Message, _ []agentruntime.ToolDef) (agentruntime.ChatResult, error) {
	s.seen = append(s.seen, messages...)
	defer func() { s.idx++ }()
	if s.idx == 0 {
		args, _ := json.Marshal(map[string]string{"program": s.program, "code": s.code})
		return agentruntime.ChatResult{
			ToolCalls:    []agentruntime.ToolCall{{ID: "1", Name: "exec", Arguments: string(args)}},
			PromptTokens: 10, CompletionTokens: 5,
		}, nil
	}
	args, _ := json.Marshal(map[string]string{"answer": "done"})
	return agentruntime.ChatResult{
		ToolCalls:    []agentruntime.ToolCall{{ID: "2", Name: "finish", Arguments: string(args)}},
		PromptTokens: 5, CompletionTokens: 5,
	}, nil
}

// newExecServer starts a codellm test server allowing only bash, sandbox off
// (portable: no Linux namespace privileges needed in the test environment).
func newExecServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(codellm.New(codellm.Config{
		AllowedPrograms: []string{"bash"},
		Sandbox:         coderun.SandboxPolicy{Mode: coderun.SandboxOff},
	}).Handler())
	t.Cleanup(srv.Close)
	return srv
}

// TestExecTool_EndToEndThroughCodellm is the exec-routing parity test: the
// agent loop's exec(program, code) tool call travels through the real
// OpenAILLM client to a real codellm server, which executes the block and
// returns write_note+finish tool_calls; the runtime applies the write through
// the scoped KB — same observable behavior as the old in-process RunBlock path.
func TestExecTool_EndToEndThroughCodellm(t *testing.T) {
	srv := newExecServer(t)
	kb := agentruntime.NewFileKB(t.TempDir())
	llm := &execScriptLLM{
		program: "bash",
		code:    `echo '{"changes":[{"path":"notes/exec.md","content":"from exec"}],"answer":"wrote"}'`,
	}

	res, err := agentruntime.Run(context.Background(), agentruntime.Input{
		Instruction:   "run some code",
		ReadPatterns:  []string{"**"},
		WritePatterns: []string{"notes/**"},
		Model:         "m",
		MaxTokens:     10000,
		MaxSteps:      5,
		ExecLLM:       agentruntime.NewOpenAILLM("test-key", srv.URL+"/v1"),
		LLM:           llm,
		KB:            kb,
	})
	require.NoError(t, err)
	require.Equal(t, agentruntime.StatusCompleted, res.Status)
	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/exec.md", res.Changes[0].Path)

	content, err := kb.Read(context.Background(), "notes/exec.md")
	require.NoError(t, err)
	require.Equal(t, "from exec", content)

	// The model sees the same summary shape as the old in-process exec.
	var summary string
	for _, m := range llm.seen {
		if m.Role == agentruntime.RoleTool && m.Name == "exec" {
			summary = m.Content
		}
	}
	require.Equal(t, "ok: ran bash, 1 write(s); wrote", summary)
}

// TestExecTool_DisallowedProgramThroughCodellm asserts allowlisting is
// codellm-authoritative: a program outside codellm's --allowed-programs comes
// back as a deterministic 422 and reaches the model as a soft tool error, with
// no changes applied.
func TestExecTool_DisallowedProgramThroughCodellm(t *testing.T) {
	srv := newExecServer(t) // allows bash only
	kb := agentruntime.NewFileKB(t.TempDir())
	llm := &execScriptLLM{
		program: "python",
		code:    `print("nope")`,
	}

	res, err := agentruntime.Run(context.Background(), agentruntime.Input{
		Instruction:   "use python",
		ReadPatterns:  []string{"**"},
		WritePatterns: []string{"notes/**"},
		Model:         "m",
		MaxTokens:     10000,
		MaxSteps:      5,
		ExecLLM:       agentruntime.NewOpenAILLM("test-key", srv.URL+"/v1"),
		LLM:           llm,
		KB:            kb,
	})
	require.NoError(t, err)
	require.Empty(t, res.Changes)

	var gotError bool
	for _, m := range llm.seen {
		if m.Role == agentruntime.RoleTool && strings.Contains(m.Content, "not in --allowed-programs") {
			gotError = true
		}
	}
	require.True(t, gotError, "model must receive codellm's allowlist error")
}

// TestExecTool_FailingBlockThroughCodellm asserts a non-zero exit stays a soft
// tool error (the model can self-correct), exactly like the old in-process
// exec's RunBlock failure path.
func TestExecTool_FailingBlockThroughCodellm(t *testing.T) {
	srv := newExecServer(t)
	kb := agentruntime.NewFileKB(t.TempDir())
	llm := &execScriptLLM{
		program: "bash",
		code:    "echo boom >&2; exit 1",
	}

	res, err := agentruntime.Run(context.Background(), agentruntime.Input{
		Instruction:   "fail",
		ReadPatterns:  []string{"**"},
		WritePatterns: []string{"notes/**"},
		Model:         "m",
		MaxTokens:     10000,
		MaxSteps:      5,
		ExecLLM:       agentruntime.NewOpenAILLM("test-key", srv.URL+"/v1"),
		LLM:           llm,
		KB:            kb,
	})
	require.NoError(t, err)
	require.Empty(t, res.Changes)

	var gotError bool
	for _, m := range llm.seen {
		if m.Role == agentruntime.RoleTool && strings.HasPrefix(m.Content, "error:") {
			gotError = true
		}
	}
	require.True(t, gotError, "block failure must surface as a soft tool error")
}
