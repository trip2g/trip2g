package hermesllm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
)

// writeNoteTool / finishTool mirror the JSON-schema tool defs fleet advertises.
func writeNoteTool() goopenai.Tool {
	return goopenai.Tool{
		Type: goopenai.ToolTypeFunction,
		Function: &goopenai.FunctionDefinition{
			Name:        "write_note",
			Description: "Create or replace a document (write scope only).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func finishTool() goopenai.Tool {
	return goopenai.Tool{
		Type: goopenai.ToolTypeFunction,
		Function: &goopenai.FunctionDefinition{
			Name:        "finish",
			Description: "End the run with a final answer/summary.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"answer": map[string]any{"type": "string"},
				},
				"required": []string{"answer"},
			},
		},
	}
}

func TestBuildPreamble(t *testing.T) {
	tests := []struct {
		name  string
		tools []goopenai.Tool
		want  []string // substrings the preamble must contain
		empty bool
	}{
		{
			name:  "no tools is pass-through: no preamble at all",
			tools: nil,
			empty: true,
		},
		{
			name:  "tool list and protocol block",
			tools: []goopenai.Tool{writeNoteTool(), finishTool()},
			want: []string{
				"- write_note(path: string, content: string) — Create or replace a document (write scope only).",
				"- finish(answer: string) — End the run with a final answer/summary.",
				`{"tool_calls":[{"name":"<tool>","arguments":{...}}]}`,
				"markdown",
				"finish",
			},
		},
		{
			name: "tool without parameters renders empty arg list",
			tools: []goopenai.Tool{{
				Type:     goopenai.ToolTypeFunction,
				Function: &goopenai.FunctionDefinition{Name: "ping", Description: "Check liveness."},
			}},
			want: []string{"- ping() — Check liveness."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPreamble(tt.tools)
			if tt.empty {
				require.Empty(t, got)
				return
			}
			for _, want := range tt.want {
				require.Contains(t, got, want)
			}
		})
	}
}

func TestBuildPreamble_OptionalParamsAfterRequired(t *testing.T) {
	tool := goopenai.Tool{
		Type: goopenai.ToolTypeFunction,
		Function: &goopenai.FunctionDefinition{
			Name: "search",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"limit": map[string]any{"type": "integer"},
					"深":     map[string]any{"type": "boolean"},
				},
				"required": []string{"query"},
			},
		},
	}
	// Required first (schema order), then the rest sorted — deterministic output.
	require.Contains(t, buildPreamble([]goopenai.Tool{tool}), "- search(query: string, limit: integer, 深: boolean)")
}

func TestFlattenMessages(t *testing.T) {
	msgs := []goopenai.ChatCompletionMessage{
		{Role: goopenai.ChatMessageRoleSystem, Content: "You are an agent."},
		{Role: goopenai.ChatMessageRoleUser, Content: "Sum 10 and 20."},
		{Role: goopenai.ChatMessageRoleAssistant, ToolCalls: []goopenai.ToolCall{{
			ID:       "call_0",
			Type:     goopenai.ToolTypeFunction,
			Function: goopenai.FunctionCall{Name: "write_note", Arguments: `{"path":"a.md"}`},
		}}},
		{Role: goopenai.ChatMessageRoleTool, ToolCallID: "call_0", Content: "written"},
	}

	got := flattenMessages("PREAMBLE", msgs)
	require.Equal(t, []hermesMessage{
		{Role: "system", Content: "PREAMBLE\n\nYou are an agent."},
		{Role: "user", Content: "Sum 10 and 20."},
		{Role: "assistant", Content: `Called write_note with {"path":"a.md"}`},
		{Role: "user", Content: "Result of write_note: written"},
	}, got)
}

func TestFlattenMessages_SystemsMergedAndPreambleOnly(t *testing.T) {
	msgs := []goopenai.ChatCompletionMessage{
		{Role: goopenai.ChatMessageRoleSystem, Content: "first"},
		{Role: goopenai.ChatMessageRoleUser, Content: "go"},
		{Role: goopenai.ChatMessageRoleSystem, Content: "second"},
	}
	require.Equal(t, []hermesMessage{
		{Role: "system", Content: "PRE\n\nfirst\n\nsecond"},
		{Role: "user", Content: "go"},
	}, flattenMessages("PRE", msgs))

	// No preamble (pass-through) keeps the incoming system text untouched.
	require.Equal(t, []hermesMessage{
		{Role: "system", Content: "first\n\nsecond"},
		{Role: "user", Content: "go"},
	}, flattenMessages("", msgs))
}

func TestFlattenMessages_ToolNameFallsBackToMessageName(t *testing.T) {
	msgs := []goopenai.ChatCompletionMessage{
		{Role: goopenai.ChatMessageRoleTool, Name: "search", Content: "3 hits"},
		{Role: goopenai.ChatMessageRoleTool, ToolCallID: "unknown", Content: "orphan"},
	}
	require.Equal(t, []hermesMessage{
		{Role: "user", Content: "Result of search: 3 hits"},
		{Role: "user", Content: "Result of tool: orphan"},
	}, flattenMessages("", msgs))
}

func TestFlattenMessages_AssistantContentAndCalls(t *testing.T) {
	msgs := []goopenai.ChatCompletionMessage{{
		Role:    goopenai.ChatMessageRoleAssistant,
		Content: "thinking",
		ToolCalls: []goopenai.ToolCall{
			{ID: "call_0", Function: goopenai.FunctionCall{Name: "search", Arguments: `{"query":"x"}`}},
			{ID: "call_1", Function: goopenai.FunctionCall{Name: "finish", Arguments: `{"answer":"y"}`}},
		},
	}}
	require.Equal(t, []hermesMessage{{
		Role:    "assistant",
		Content: "thinking\nCalled search with {\"query\":\"x\"}\nCalled finish with {\"answer\":\"y\"}",
	}}, flattenMessages("", msgs))
}

func TestExtractToolCalls(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []goopenai.ToolCall
	}{
		{
			name: "bare object",
			text: `{"tool_calls":[{"name":"finish","arguments":{"answer":"ok"}}]}`,
			want: []goopenai.ToolCall{call(0, "finish", `{"answer":"ok"}`)},
		},
		{
			name: "json fenced",
			text: "```json\n{\"tool_calls\":[{\"name\":\"finish\",\"arguments\":{\"answer\":\"ok\"}}]}\n```",
			want: []goopenai.ToolCall{call(0, "finish", `{"answer":"ok"}`)},
		},
		{
			name: "prose wrapped",
			text: "Sure! Here is the call:\n{\"tool_calls\":[{\"name\":\"finish\",\"arguments\":{\"answer\":\"ok\"}}]}\nHope that helps.",
			want: []goopenai.ToolCall{call(0, "finish", `{"answer":"ok"}`)},
		},
		{
			name: "nested braces in arguments",
			text: `{"tool_calls":[{"name":"write_note","arguments":{"path":"a.md","meta":{"tags":["x"]}}}]}`,
			// Key order is preserved: arguments are compacted, not re-marshalled.
			want: []goopenai.ToolCall{call(0, "write_note", `{"path":"a.md","meta":{"tags":["x"]}}`)},
		},
		{
			name: "braces inside string literals",
			text: `{"tool_calls":[{"name":"write_note","arguments":{"content":"a } b { c \" d"}}]}`,
			want: []goopenai.ToolCall{call(0, "write_note", `{"content":"a } b { c \" d"}`)},
		},
		{
			name: "single object form",
			text: `{"name":"finish","arguments":{"answer":"ok"}}`,
			want: []goopenai.ToolCall{call(0, "finish", `{"answer":"ok"}`)},
		},
		{
			name: "single object with id and type is still a call",
			text: `{"id":"x","type":"function","name":"finish","arguments":"{}"}`,
			want: []goopenai.ToolCall{call(0, "finish", `{}`)},
		},
		{
			// A final answer that merely has a name key must stay plain content.
			name: "single object with foreign keys is not a call",
			text: `{"name":"Alice","age":30}`,
			want: nil,
		},
		{
			name: "single object without arguments is not a call",
			text: `{"name":"Alice"}`,
			want: nil,
		},
		{
			name: "bare array form",
			text: `[{"name":"finish","arguments":{"answer":"ok"}}]`,
			want: []goopenai.ToolCall{call(0, "finish", `{"answer":"ok"}`)},
		},
		{
			name: "arguments already a string",
			text: `{"tool_calls":[{"name":"finish","arguments":"{\"answer\":\"ok\"}"}]}`,
			want: []goopenai.ToolCall{call(0, "finish", `{"answer":"ok"}`)},
		},
		{
			name: "missing arguments becomes empty object",
			text: `{"tool_calls":[{"name":"finish"}]}`,
			want: []goopenai.ToolCall{call(0, "finish", `{}`)},
		},
		{
			name: "several calls are indexed in order",
			text: `{"tool_calls":[{"name":"write_note","arguments":{"path":"a.md"}},{"name":"finish","arguments":{"answer":"ok"}}]}`,
			want: []goopenai.ToolCall{
				call(0, "write_note", `{"path":"a.md"}`),
				call(1, "finish", `{"answer":"ok"}`),
			},
		},
		{
			name: "entries without a name are dropped",
			text: `{"tool_calls":[{"arguments":{"answer":"ok"}},{"name":"finish","arguments":{"answer":"ok"}}]}`,
			want: []goopenai.ToolCall{call(0, "finish", `{"answer":"ok"}`)},
		},
		{
			name: "only nameless entries yields nothing",
			text: `{"tool_calls":[{"arguments":{"answer":"ok"}}]}`,
			want: nil,
		},
		{
			name: "unrelated leading json is skipped",
			text: `Note {"a":1} then {"tool_calls":[{"name":"finish","arguments":{"answer":"ok"}}]}`,
			want: []goopenai.ToolCall{call(0, "finish", `{"answer":"ok"}`)},
		},
		{
			name: "plain prose yields nothing",
			text: "I could not complete the task.",
			want: nil,
		},
		{
			name: "unbalanced json yields nothing",
			text: `{"tool_calls":[{"name":"finish"`,
			want: nil,
		},
		{
			name: "empty tool_calls yields nothing",
			text: `{"tool_calls":[]}`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, extractToolCalls(tt.text, testIDPrefix))
		})
	}
}

const testIDPrefix = "abc123"

func call(i int, name, args string) goopenai.ToolCall {
	return callWithPrefix(testIDPrefix, i, name, args)
}

func callWithPrefix(prefix string, i int, name, args string) goopenai.ToolCall {
	return goopenai.ToolCall{
		ID:       fmt.Sprintf("call_%s_%d", prefix, i),
		Type:     goopenai.ToolTypeFunction,
		Function: goopenai.FunctionCall{Name: name, Arguments: args},
	}
}

// fakeUpstream is a stub Hermes that records what the shim sent it.
type fakeUpstream struct {
	URL    string
	Path   string
	Body   []byte
	Header http.Header
}

// fakeHermes starts a stub upstream replying with the given status and body.
func fakeHermes(t *testing.T, status int, body string) *fakeUpstream {
	t.Helper()
	up := &fakeUpstream{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up.Path = r.URL.Path
		up.Body, _ = io.ReadAll(r.Body)
		up.Header = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	up.URL = srv.URL
	return up
}

// hermesReply wraps content in a minimal Hermes chat-completions response.
// An optional usage block mirrors the real one Hermes returns.
func hermesReply(content string, usage ...map[string]any) string {
	reply := map[string]any{
		"choices": []any{map[string]any{
			"message":       map[string]any{"role": "assistant", "content": content},
			"finish_reason": "stop",
		}},
	}
	if len(usage) > 0 {
		reply["usage"] = usage[0]
	}
	b, _ := json.Marshal(reply)
	return string(b)
}

// chatRequest posts an OpenAI chat request to srv and returns the recorder.
func chatRequest(srv *Server, bearer string, req goopenai.ChatCompletionRequest) *httptest.ResponseRecorder {
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, r)
	return rec
}

func newServer(t *testing.T, hermesURL, apiKey string) *Server {
	t.Helper()
	return New(Config{
		HermesURL: hermesURL,
		HermesKey: "upstream-key",
		Model:     "hermes-agent",
		Auth:      APIKeyAuth(apiKey),
	})
}

func agentRequest() goopenai.ChatCompletionRequest {
	return goopenai.ChatCompletionRequest{
		Model: "hermes-agent",
		Messages: []goopenai.ChatCompletionMessage{
			{Role: goopenai.ChatMessageRoleSystem, Content: "You are an agent."},
			{Role: goopenai.ChatMessageRoleUser, Content: "Sum 10 and 20."},
		},
		Tools: []goopenai.Tool{writeNoteTool(), finishTool()},
	}
}

func decodeChat(t *testing.T, rec *httptest.ResponseRecorder) goopenai.ChatCompletionResponse {
	t.Helper()
	var out goopenai.ChatCompletionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func TestChatCompletions_ToolCallsTranslated(t *testing.T) {
	upstream := fakeHermes(t, http.StatusOK,
		hermesReply(`{"tool_calls":[{"name":"write_note","arguments":{"path":"results/task1.md","content":"answer: 30\n"}}]}`))
	srv := newServer(t, upstream.URL, "")

	rec := chatRequest(srv, "", agentRequest())
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	resp := decodeChat(t, rec)
	require.Equal(t, "hermes-agent", resp.Model)
	require.Equal(t, "chat.completion", resp.Object)
	require.True(t, strings.HasPrefix(resp.ID, "hermesllm-"))
	require.Len(t, resp.Choices, 1)
	require.Equal(t, goopenai.FinishReasonToolCalls, resp.Choices[0].FinishReason)
	require.Empty(t, resp.Choices[0].Message.Content)
	// Call ids carry the response's own random token, so turns never collide.
	require.Equal(t, []goopenai.ToolCall{
		callWithPrefix(strings.TrimPrefix(resp.ID, "hermesllm-"), 0, "write_note", `{"path":"results/task1.md","content":"answer: 30\n"}`),
	}, resp.Choices[0].Message.ToolCalls)

	// Upstream request: no model, streaming off, preamble prepended, key forwarded.
	require.Equal(t, "/v1/chat/completions", upstream.Path)
	require.Equal(t, "Bearer upstream-key", upstream.Header.Get("Authorization"))
	var sent map[string]any
	require.NoError(t, json.Unmarshal(upstream.Body, &sent))
	require.NotContains(t, sent, "model")
	require.Equal(t, false, sent["stream"])
	msgs, ok := sent["messages"].([]any)
	require.True(t, ok)
	require.Len(t, msgs, 2)
	system, ok := msgs[0].(map[string]any)
	require.True(t, ok)
	require.Contains(t, system["content"], "- write_note(path: string, content: string)")
	require.Contains(t, system["content"], "You are an agent.")
}

// Hermes spends real subscription tokens, and fleet's max_tokens / token ceiling
// are computed from what the shim reports — so usage must pass straight through.
func TestChatCompletions_UsagePassthrough(t *testing.T) {
	tests := []struct {
		name  string
		usage []map[string]any
		want  goopenai.Usage
	}{
		{
			name:  "real hermes usage",
			usage: []map[string]any{{"prompt_tokens": 14470, "completion_tokens": 41, "total_tokens": 14511}},
			want:  goopenai.Usage{PromptTokens: 14470, CompletionTokens: 41, TotalTokens: 14511},
		},
		{
			name: "absent usage is zero, not an error",
			want: goopenai.Usage{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := fakeHermes(t, http.StatusOK, hermesReply("done", tt.usage...))
			srv := newServer(t, upstream.URL, "")

			rec := chatRequest(srv, "", agentRequest())
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			require.Equal(t, tt.want, decodeChat(t, rec).Usage)
		})
	}
}

// Different turns of one run must not reuse tool_call ids.
func TestChatCompletions_ToolCallIDsDifferPerResponse(t *testing.T) {
	upstream := fakeHermes(t, http.StatusOK,
		hermesReply(`{"tool_calls":[{"name":"finish","arguments":{"answer":"ok"}}]}`))
	srv := newServer(t, upstream.URL, "")

	first := decodeChat(t, chatRequest(srv, "", agentRequest()))
	second := decodeChat(t, chatRequest(srv, "", agentRequest()))
	require.NotEqual(t,
		first.Choices[0].Message.ToolCalls[0].ID,
		second.Choices[0].Message.ToolCalls[0].ID)
}

func TestChatCompletions_StreamingRejected(t *testing.T) {
	upstream := fakeHermes(t, http.StatusOK, hermesReply("hello"))
	srv := newServer(t, upstream.URL, "")

	req := agentRequest()
	req.Stream = true
	rec := chatRequest(srv, "", req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, upstream.Body, "the request never reaches hermes")
}

func TestChatCompletions_PlainTextBecomesStopAnswer(t *testing.T) {
	upstream := fakeHermes(t, http.StatusOK, hermesReply("I could not find the note."))
	srv := newServer(t, upstream.URL, "")

	rec := chatRequest(srv, "", agentRequest())
	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeChat(t, rec)
	require.Equal(t, goopenai.FinishReasonStop, resp.Choices[0].FinishReason)
	require.Equal(t, "I could not find the note.", resp.Choices[0].Message.Content)
	require.Empty(t, resp.Choices[0].Message.ToolCalls)
}

func TestChatCompletions_PassThroughWithoutTools(t *testing.T) {
	upstream := fakeHermes(t, http.StatusOK, hermesReply("hello"))
	srv := newServer(t, upstream.URL, "")

	req := agentRequest()
	req.Tools = nil
	rec := chatRequest(srv, "", req)
	require.Equal(t, http.StatusOK, rec.Code)

	var sent map[string]any
	require.NoError(t, json.Unmarshal(upstream.Body, &sent))
	msgs, _ := sent["messages"].([]any)
	system, _ := msgs[0].(map[string]any)
	require.Equal(t, "You are an agent.", system["content"], "no tools means no synthetic preamble")
}

func TestChatCompletions_HermesFailureIs422(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "hermes envelope reports failure",
			body: `{"choices":[{"message":{"role":"assistant","content":""},"finish_reason":"stop"}],
			        "hermes":{"failed":true,"error":"agent crashed"}}`,
		},
		{
			name: "finish_reason error",
			body: `{"choices":[{"message":{"role":"assistant","content":"partial"},"finish_reason":"error"}]}`,
		},
		{
			name: "structured hermes error",
			body: `{"choices":[{"message":{"content":""}}],"hermes":{"failed":true,"error":{"code":7}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := fakeHermes(t, http.StatusOK, tt.body)
			srv := newServer(t, upstream.URL, "")

			rec := chatRequest(srv, "", agentRequest())
			require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())

			var out apiError
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
			require.Equal(t, "hermes_error", out.Error.Type)
			require.NotEmpty(t, out.Error.Message)
		})
	}
}

func TestChatCompletions_UpstreamFailureIs502(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "upstream 500", status: http.StatusInternalServerError, body: "boom"},
		{name: "upstream 401", status: http.StatusUnauthorized, body: "nope"},
		{name: "undecodable body", status: http.StatusOK, body: "not json"},
		{name: "no choices", status: http.StatusOK, body: `{"choices":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := fakeHermes(t, tt.status, tt.body)
			srv := newServer(t, upstream.URL, "")

			rec := chatRequest(srv, "", agentRequest())
			require.Equal(t, http.StatusBadGateway, rec.Code, "body: %s", rec.Body.String())

			var out apiError
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
			require.Equal(t, "hermes_upstream_error", out.Error.Type)
		})
	}
}

func TestChatCompletions_UnreachableUpstreamIs502(t *testing.T) {
	srv := newServer(t, "http://127.0.0.1:1", "")
	rec := chatRequest(srv, "", agentRequest())
	require.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestChatCompletions_BadRequestBody(t *testing.T) {
	srv := newServer(t, "http://127.0.0.1:1", "")
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, r)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestModelsAndHealthz(t *testing.T) {
	srv := newServer(t, "http://127.0.0.1:1", "an-api-key-that-is-long-enough!!")

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	require.Equal(t, http.StatusOK, rec.Code, "/v1/models is open, never key-gated")
	var models struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &models))
	require.Equal(t, "list", models.Object)
	require.Len(t, models.Data, 1)
	require.Equal(t, "hermes-agent", models.Data[0].ID)
	require.Equal(t, "model", models.Data[0].Object)
	require.Equal(t, "trip2g", models.Data[0].OwnedBy)

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
}
