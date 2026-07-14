package codellm

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"

	"trip2g/internal/agentruntime"
)

// chatDebugResponse mirrors debugResponse for decoding on the client side.
type chatDebugResponse struct {
	goopenai.ChatCompletionResponse
	XFleetDebug *struct {
		Blocks []struct {
			Index      int    `json:"index"`
			Stdout     string `json:"stdout"`
			PipeBuffer string `json:"pipeBuffer"`
		} `json:"blocks"`
	} `json:"x_fleet_debug"`
}

// doChatWithDebug posts a chat request carrying the x_fleet_debug extension flag.
func doChatWithDebug(t *testing.T, srv *Server, messages []goopenai.ChatCompletionMessage) *httptest.ResponseRecorder {
	t.Helper()
	payload := map[string]any{
		"model":         "codellm",
		"messages":      messages,
		"x_fleet_debug": true,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

// TestChatCompletions_XFleetDebug_SingleBlock asserts the opt-in extension adds
// the x_fleet_debug field with per-block stdout for a single block.
func TestChatCompletions_XFleetDebug_SingleBlock(t *testing.T) {
	srv := newTestServer()
	script := `echo '{"changes":[],"answer":"solo"}'`
	rec := doChatWithDebug(t, srv, []goopenai.ChatCompletionMessage{bashBody(script)})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp chatDebugResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Standard completion still intact (finish present, usage zero).
	require.Equal(t, "finish", resp.Choices[0].Message.ToolCalls[0].Function.Name)
	require.Equal(t, 0, resp.Usage.TotalTokens)

	require.NotNil(t, resp.XFleetDebug, "x_fleet_debug must be present when opted in")
	require.Len(t, resp.XFleetDebug.Blocks, 1)
	require.Equal(t, 0, resp.XFleetDebug.Blocks[0].Index)
	require.Empty(t, resp.XFleetDebug.Blocks[0].PipeBuffer)
	require.Contains(t, resp.XFleetDebug.Blocks[0].Stdout, `"answer":"solo"`)
}

// TestChatCompletions_XFleetDebug_Pipeline asserts the inter-block pipe buffer is
// surfaced per block for a multi-block pipeline.
func TestChatCompletions_XFleetDebug_Pipeline(t *testing.T) {
	srv := newTestServer()
	body := goopenai.ChatCompletionMessage{
		Role: goopenai.ChatMessageRoleSystem,
		Content: "agent\n```bash\necho hello\n```\n" +
			"```bash\nv=$(cat); echo \"{\\\"changes\\\":[],\\\"answer\\\":\\\"$v\\\"}\"\n```",
	}
	rec := doChatWithDebug(t, srv, []goopenai.ChatCompletionMessage{body})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp chatDebugResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.XFleetDebug)
	require.Len(t, resp.XFleetDebug.Blocks, 2)

	require.Equal(t, "hello\n", resp.XFleetDebug.Blocks[0].PipeBuffer, "inter-block pipe buffer")
	require.Equal(t, "hello\n", resp.XFleetDebug.Blocks[0].Stdout)
	require.Empty(t, resp.XFleetDebug.Blocks[1].PipeBuffer, "last block has no downstream")
	require.Contains(t, resp.XFleetDebug.Blocks[1].Stdout, `"answer":"hello"`)
}

// TestChatCompletions_NoDebug_ByteIdentical asserts a request WITHOUT the
// x_fleet_debug flag produces no x_fleet_debug field (normal path unchanged).
func TestChatCompletions_NoDebug_ByteIdentical(t *testing.T) {
	srv := newTestServer()
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{bashBody(`echo '{"changes":[],"answer":""}'`)})
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), "x_fleet_debug", "normal requests must not carry the debug field")
}

// TestBrowserAuth_TokenLaneBypass asserts BrowserAuth lets a valid fleet-lane
// token bypass the (failing) browser admin gate.
func TestBrowserAuth_TokenLaneBypass(t *testing.T) {
	rejectAdmin := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
	}
	tokenOK := func(r *http.Request) error {
		if r.Header.Get("X-Channel-Token") == "secret" {
			return nil
		}
		return errAuth
	}
	srv := New(Config{
		AllowedPrograms: []string{"bash"},
		Sandbox:         agentruntime.SandboxPolicy{Mode: agentruntime.SandboxOff},
		TokenCheck:      tokenOK,
		Auth:            BrowserAuth(rejectAdmin, tokenOK),
	})

	// No token → browser admin gate rejects.
	rec := doChat(t, srv, []goopenai.ChatCompletionMessage{bashBody(`echo '{"changes":[],"answer":""}'`)})
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// With the channel token → fleet lane bypass → served.
	body, _ := json.Marshal(goopenai.ChatCompletionRequest{
		Model:    "codellm",
		Messages: []goopenai.ChatCompletionMessage{bashBody(`echo '{"changes":[],"answer":"ok"}'`)},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("X-Channel-Token", "secret")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

var errAuth = &authError{}

type authError struct{}

func (*authError) Error() string { return "no token" }
