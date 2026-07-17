package agentruntime

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Exec-tool wire parity tests: Run's exec(program, code) tool driven through
// the real OpenAILLM client against a minimal httptest stand-in for codellm's
// /v1/chat/completions contract. They replace the old cross-package
// internal/codellm/exec_parity_test.go so neither side imports the other:
// codellm's actual /v1 behavior (block execution, allowlist 422, failing-block
// 422) is asserted in codellm's own server tests, and the real
// agentruntime→codellm→sandbox path is covered end-to-end by the docker e2e
// suite.

// codellmWriteAndFinish is the completion codellm returns for a block that
// wrote one note: write_note + trailing finish tool_calls, zero usage.
const codellmWriteAndFinish = `{"id":"1","object":"chat.completion","model":"codellm","choices":[{"index":0,"message":{"role":"assistant","tool_calls":[` +
	`{"id":"c0","type":"function","function":{"name":"write_note","arguments":"{\"path\":\"notes/exec.md\",\"content\":\"from exec\"}"}},` +
	`{"id":"c1","type":"function","function":{"name":"finish","arguments":"{\"answer\":\"wrote\"}"}}` +
	`]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`

// TestExecTool_WireParityWithCodellm asserts the exec tool call travels the
// real OpenAILLM client as codellm expects it — one /v1 chat request, model
// "codellm", a single user message carrying the code as one fenced block — and
// that the returned write_note/finish tool_calls map back into applied changes
// and the usual exec summary.
func TestExecTool_WireParityWithCodellm(t *testing.T) {
	var gotBody map[string]any
	exec, calls := llmStubServer(t, func(_ int, body map[string]any) (int, string) {
		gotBody = body
		return http.StatusOK, codellmWriteAndFinish
	})
	kb := newMemKB(nil)

	res, llm := runWithExec(t, kb, []string{"notes/**"}, "bash", "echo hi", exec)

	require.Equal(t, StatusCompleted, res.Status)
	require.Len(t, res.Changes, 1)
	require.Equal(t, "notes/exec.md", res.Changes[0].Path)
	require.Equal(t, "from exec", kb.docs["notes/exec.md"])

	// Wire shape.
	require.Equal(t, int32(1), calls.Load())
	require.Equal(t, execModel, gotBody["model"])
	msgs, ok := gotBody["messages"].([]any)
	require.True(t, ok, "messages must be an array")
	require.Len(t, msgs, 1)
	msg, ok := msgs[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "user", msg["role"])
	require.Equal(t, "```bash\necho hi\n```", msg["content"])

	// The model sees the same summary shape as the old in-process exec.
	var summary string
	for _, m := range llm.seen {
		if m.Role == RoleTool && m.Name == toolExec {
			summary = m.Content
		}
	}
	require.Equal(t, "ok: ran bash, 1 write(s); wrote", summary)
}

// TestExecTool_Codellm422IsSoftToolError asserts codellm's deterministic 422s
// (disallowed program, failing block) travel the wire client as a terminal
// error — no retry — and reach the model as a soft tool error carrying
// codellm's message, with no changes applied. The 422 production itself is
// asserted in codellm's own server tests.
func TestExecTool_Codellm422IsSoftToolError(t *testing.T) {
	cases := []struct {
		name    string
		program string
		code    string
		message string // as it rides the 422 error JSON
		want    string // substring the model must see
	}{
		{
			name:    "disallowed program",
			program: "python",
			code:    `print("nope")`,
			message: `coderun: program \"python\" not in --allowed-programs [bash]`,
			want:    "not in --allowed-programs",
		},
		{
			name:    "failing block",
			program: "bash",
			code:    "echo boom >&2; exit 1",
			message: `coderun: non-zero exit: exit status 1: boom`,
			want:    "non-zero exit",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exec, calls := llmStubServer(t, func(_ int, _ map[string]any) (int, string) {
				return http.StatusUnprocessableEntity,
					`{"error":{"message":"` + tc.message + `","type":"code_execution_error"}}`
			})
			kb := newMemKB(nil)

			res, llm := runWithExec(t, kb, []string{"notes/**"}, tc.program, tc.code, exec)

			require.Empty(t, res.Changes)
			require.Equal(t, int32(1), calls.Load(), "422 must be terminal, not retried")
			var gotError bool
			for _, m := range llm.seen {
				if m.Role == RoleTool && strings.HasPrefix(m.Content, "error:") &&
					strings.Contains(m.Content, tc.want) {
					gotError = true
				}
			}
			require.True(t, gotError, "model must receive codellm's error as a soft tool error")
		})
	}
}
