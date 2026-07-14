package codellm

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"

	"trip2g/internal/coderun"
)

// newTestServer builds a codellm server that runs bash blocks unsandboxed, so
// the wire-protocol tests are portable (no unprivileged-userns dependency). The
// sandbox itself is exercised in coderun's own tests.
func newTestServer() *Server {
	return New(Config{
		AllowedPrograms: []string{"bash"},
		Sandbox:         coderun.SandboxPolicy{Mode: coderun.SandboxOff},
	})
}

// doChat posts a chat-completions request built from the given messages and
// returns the raw recorder.
func doChat(t *testing.T, srv *Server, messages []goopenai.ChatCompletionMessage) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(goopenai.ChatCompletionRequest{Model: "codellm", Messages: messages})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

// bashBody wraps a bash script as a fenced markdown block inside the system
// prompt, mirroring how fleet delivers a rendered code-role body.
func bashBody(script string) goopenai.ChatCompletionMessage {
	return goopenai.ChatCompletionMessage{
		Role:    goopenai.ChatMessageRoleSystem,
		Content: "You are a scoped micro-agent.\n```bash\n" + script + "\n```",
	}
}

func decodeResp(t *testing.T, rec *httptest.ResponseRecorder) goopenai.ChatCompletionResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var resp goopenai.ChatCompletionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

// finishArgs finds the finish tool call and returns its answer argument.
func finishArgs(t *testing.T, calls []goopenai.ToolCall) string {
	t.Helper()
	var answer string
	var count int
	for _, c := range calls {
		if c.Function.Name == "finish" {
			count++
			var a struct {
				Answer string `json:"answer"`
			}
			require.NoError(t, json.Unmarshal([]byte(c.Function.Arguments), &a))
			answer = a.Answer
		}
	}
	require.Equal(t, 1, count, "exactly one finish tool_call required")
	return answer
}

func TestChatCompletions_WriteNoteAndFinish(t *testing.T) {
	srv := newTestServer()
	script := `echo '{"changes":[{"path":"notes/a.md","content":"generated"}],"answer":"done"}'`
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{
		bashBody(script),
		{Role: goopenai.ChatMessageRoleUser, Content: "Begin."},
	})
	resp := decodeResp(t, rec)

	require.Len(t, resp.Choices, 1)
	choice := resp.Choices[0]
	require.Equal(t, goopenai.FinishReasonToolCalls, choice.FinishReason)
	require.Equal(t, 0, resp.Usage.TotalTokens)
	require.Equal(t, 0, resp.Usage.PromptTokens)
	require.Equal(t, 0, resp.Usage.CompletionTokens)

	calls := choice.Message.ToolCalls
	require.Len(t, calls, 2, "one write_note + trailing finish")

	require.Equal(t, "write_note", calls[0].Function.Name)
	var wargs struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal([]byte(calls[0].Function.Arguments), &wargs))
	require.Equal(t, "notes/a.md", wargs.Path)
	require.Equal(t, "generated", wargs.Content)

	// finish must be LAST.
	require.Equal(t, "finish", calls[len(calls)-1].Function.Name)
	require.Equal(t, "done", finishArgs(t, calls))
}

func TestChatCompletions_PatchNote(t *testing.T) {
	srv := newTestServer()
	script := `echo '{"changes":[{"path":"index.md","find":"a","replace":"b"}],"answer":"patched"}'`
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{bashBody(script)})
	resp := decodeResp(t, rec)

	calls := resp.Choices[0].Message.ToolCalls
	require.Len(t, calls, 2)
	require.Equal(t, "patch_note", calls[0].Function.Name)
	var pargs struct {
		Path    string `json:"path"`
		Find    string `json:"find"`
		Replace string `json:"replace"`
	}
	require.NoError(t, json.Unmarshal([]byte(calls[0].Function.Arguments), &pargs))
	require.Equal(t, "index.md", pargs.Path)
	require.Equal(t, "a", pargs.Find)
	require.Equal(t, "b", pargs.Replace)
	require.Equal(t, "finish", calls[1].Function.Name)
}

// TestChatCompletions_AlwaysFinish_EmptyAnswer is the core conformance guard:
// even with zero changes and an empty answer, the completion ends with exactly
// one finish(""). Missing finish would let fleet's stateless loop re-run the
// blocks and re-apply every write.
func TestChatCompletions_AlwaysFinish_EmptyAnswer(t *testing.T) {
	srv := newTestServer()
	script := `echo '{"changes":[],"answer":""}'`
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{bashBody(script)})
	resp := decodeResp(t, rec)

	calls := resp.Choices[0].Message.ToolCalls
	require.Len(t, calls, 1, "no changes → only the mandatory finish")
	require.Equal(t, "finish", calls[0].Function.Name)
	require.Empty(t, finishArgs(t, calls))
}

// TestEveryCompletionEndsWithFinish asserts the always-finish invariant across a
// spread of bodies: no successful completion omits finish, and finish is last.
func TestEveryCompletionEndsWithFinish(t *testing.T) {
	srv := newTestServer()
	scripts := []string{
		`echo '{"changes":[],"answer":""}'`,
		`echo '{"changes":[],"answer":"just an answer"}'`,
		`echo '{"changes":[{"path":"a.md","content":"x"}],"answer":"one write"}'`,
		`echo '{"changes":[{"path":"a.md","content":"x"},{"path":"b.md","find":"p","replace":"q"}],"answer":"mixed"}'`,
	}
	for _, script := range scripts {
		rec := doChat(t, srv, []goopenai.ChatCompletionMessage{bashBody(script)})
		resp := decodeResp(t, rec)
		calls := resp.Choices[0].Message.ToolCalls
		require.NotEmpty(t, calls, "script %q: expected tool_calls", script)
		require.Equal(t, "finish", calls[len(calls)-1].Function.Name, "script %q: finish must be last", script)
		_ = finishArgs(t, calls) // asserts exactly one finish
	}
}

// TestChatCompletions_FleetInputBag verifies the delivery bag rides as the
// fleet_input system message and is exposed to code as $FLEET_INPUT.
func TestChatCompletions_FleetInputBag(t *testing.T) {
	srv := newTestServer()
	script := `printf '{"changes":[],"answer":"%s"}' "$(cat "$FLEET_INPUT")"`
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{
		bashBody(script),
		{Role: goopenai.ChatMessageRoleSystem, Name: fleetInputMessageName, Content: "hello-bag"},
	})
	resp := decodeResp(t, rec)
	require.Equal(t, "hello-bag", finishArgs(t, resp.Choices[0].Message.ToolCalls))
}

// TestChatCompletions_ParseError422 asserts unparseable stdout is a hard 422
// (decision b: preserve today's hard-fail parity, not a soft skip).
func TestChatCompletions_ParseError422(t *testing.T) {
	srv := newTestServer()
	script := `echo 'not json at all'`
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{bashBody(script)})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())

	var e apiError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e))
	require.Equal(t, "code_execution_error", e.Error.Type)
	require.NotEmpty(t, e.Error.Message)
}

// TestChatCompletions_ExecError422 asserts a non-zero block exit is a hard 422.
func TestChatCompletions_ExecError422(t *testing.T) {
	srv := newTestServer()
	script := `echo 'boom' >&2; exit 3`
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{bashBody(script)})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
}

// TestChatCompletions_NoFencedBlock422 asserts a request carrying no fenced code
// is a hard 422 (nothing to execute), not a silent empty success.
func TestChatCompletions_NoFencedBlock422(t *testing.T) {
	srv := newTestServer()
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{
		{Role: goopenai.ChatMessageRoleUser, Content: "Begin."},
	})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
}

func TestExtractBodyAndBag(t *testing.T) {
	body, bag := extractBodyAndBag([]goopenai.ChatCompletionMessage{
		{Role: "system", Content: "role body ```code```"},
		{Role: "system", Name: fleetInputMessageName, Content: `{"changed_files":[]}`},
		{Role: "user", Content: "Begin."},
	})
	require.Contains(t, body, "role body")
	require.Contains(t, body, "Begin.")
	require.NotContains(t, body, "changed_files", "fleet_input must not be scanned for code")
	require.JSONEq(t, `{"changed_files":[]}`, string(bag))
}

func TestModelsAndHealthz(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), modelID)
}

// TestAuthSeam asserts the auth middleware seam is honored (a custom middleware
// can reject requests before they reach the handler).
func TestAuthSeam(t *testing.T) {
	srv := New(Config{
		AllowedPrograms: []string{"bash"},
		Sandbox:         coderun.SandboxPolicy{Mode: coderun.SandboxOff},
		Auth: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			})
		},
	})
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{bashBody(`echo '{"changes":[],"answer":""}'`)})
	require.Equal(t, http.StatusForbidden, rec.Code)
}
