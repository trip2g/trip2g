// Package hermesllm implements an OpenAI-compatible chat-completions shim in
// front of a Hermes agent, so cmd/fleet can drive Hermes as if it were an LLM.
//
// Hermes' own /v1/chat/completions IGNORES the OpenAI `tools` array and never
// returns tool_calls — it is a server-side agent that answers with final text.
// fleet, on the other hand, runs its loop purely off
// choices[0].message.tool_calls. The shim bridges the two: it renders fleet's
// tool schemas into a synthetic system preamble that asks Hermes for a single
// JSON object, forwards the flattened transcript, and translates the JSON back
// into real OpenAI tool_calls.
package hermesllm

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	goopenai "github.com/sashabaranov/go-openai"
)

// defaultModelID is the model advertised by GET /v1/models when Config.Model is
// unset. Hermes pins its own model, so this is only a label for clients.
const defaultModelID = "hermes-agent"

// Config configures a hermesllm Server.
type Config struct {
	// HermesURL is the BASE URL of the Hermes agent; requests go to
	// {HermesURL}/v1/chat/completions.
	HermesURL string

	// HermesKey is the upstream credential, sent as `Authorization: Bearer`.
	HermesKey string

	// Model is echoed in responses and advertised by GET /v1/models.
	Model string

	// Timeout bounds a single upstream Hermes call; 0 = request-context bound.
	Timeout time.Duration

	// Auth gates POST /v1/chat/completions. In production it is
	// APIKeyAuth(cfg.APIKey); nil defaults to a no-op passthrough.
	Auth func(http.Handler) http.Handler
}

// Server is the hermesllm HTTP service.
type Server struct {
	cfg    Config
	client *http.Client
}

// New builds a Server from cfg, filling in nil seams with no-op defaults.
func New(cfg Config) *Server {
	if cfg.Auth == nil {
		cfg.Auth = func(next http.Handler) http.Handler { return next }
	}
	if cfg.Model == "" {
		cfg.Model = defaultModelID
	}
	return &Server{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}
}

// Handler returns the HTTP handler for the service. Only the completions
// endpoint is gated; liveness and client-compatibility routes stay open.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /v1/chat/completions", s.cfg.Auth(http.HandlerFunc(s.handleChatCompletions)))
	mux.HandleFunc("GET /v1/models", s.handleModels)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	return mux
}

// handleChatCompletions translates one OpenAI chat request into a Hermes call
// and the Hermes answer back into an OpenAI completion.
func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "read request: "+err.Error())
		return
	}

	var req goopenai.ChatCompletionRequest
	if err = json.Unmarshal(raw, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "decode request: "+err.Error())
		return
	}
	if req.Stream {
		// Hermes runs an agent per call; there is nothing to stream incrementally.
		writeError(w, http.StatusBadRequest, "invalid_request_error", "streaming is not supported")
		return
	}

	content, usage, err := s.askHermes(r.Context(), flattenMessages(buildPreamble(req.Tools), req.Messages))
	if err != nil {
		var hermesErr *hermesRunError
		if errors.As(err, &hermesErr) {
			// Hermes ran and failed. 422 is terminal on fleet's side (it retries
			// only 429/5xx/transport), which is what a failed agent run deserves.
			writeError(w, http.StatusUnprocessableEntity, "hermes_error", hermesErr.msg)
			return
		}
		writeError(w, http.StatusBadGateway, "hermes_upstream_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.buildResponse(content, usage))
}

// buildResponse maps Hermes' answer text onto an OpenAI completion: a parseable
// tool_calls object becomes real tool_calls, anything else becomes plain content
// with a stop finish reason — which fleet treats as a completed run, a safe
// terminal state.
//
// Usage is passed through untouched: these are real upstream subscription tokens
// and fleet's spend ceilings are computed from them.
func (s *Server) buildResponse(content string, usage goopenai.Usage) goopenai.ChatCompletionResponse {
	// One random token per response, shared by the completion id and the ids of
	// its tool calls, so ids stay unique across turns of the same run.
	id := randomID()

	message := goopenai.ChatCompletionMessage{Role: goopenai.ChatMessageRoleAssistant}
	finishReason := goopenai.FinishReasonStop
	if calls := extractToolCalls(content, id); len(calls) > 0 {
		message.ToolCalls = calls
		finishReason = goopenai.FinishReasonToolCalls
	} else {
		message.Content = content
	}

	return goopenai.ChatCompletionResponse{
		ID:      "hermesllm-" + id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   s.cfg.Model,
		Choices: []goopenai.ChatCompletionChoice{{
			Index:        0,
			Message:      message,
			FinishReason: finishReason,
		}},
		Usage: usage,
	}
}

// hermesMessage is one turn of the Hermes conversation. Hermes accepts only
// system/user/assistant roles and no tool_calls, so everything is flattened into
// text.
type hermesMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type hermesRequest struct {
	Messages []hermesMessage `json:"messages"`
	// Hermes pins its own model via its own config, so no model is sent.
	Stream bool `json:"stream"`
}

type hermesResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Hermes *struct {
		Failed bool            `json:"failed"`
		Error  json.RawMessage `json:"error"`
	} `json:"hermes"`
}

// hermesRunError marks a run that reached Hermes and failed there (as opposed to
// a transport/protocol problem), so the handler can answer 422 instead of 502.
type hermesRunError struct{ msg string }

func (e *hermesRunError) Error() string { return e.msg }

// askHermes posts the flattened conversation to {HermesURL}/v1/chat/completions
// and returns the assistant's text plus the tokens the run really spent.
func (s *Server) askHermes(ctx context.Context, messages []hermesMessage) (string, goopenai.Usage, error) {
	var none goopenai.Usage

	body, err := json.Marshal(hermesRequest{Messages: messages, Stream: false})
	if err != nil {
		return "", none, fmt.Errorf("encode hermes request: %w", err)
	}

	url := strings.TrimSuffix(s.cfg.HermesURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", none, fmt.Errorf("build hermes request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.HermesKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", none, fmt.Errorf("call hermes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", none, fmt.Errorf("read hermes response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", none, fmt.Errorf("hermes returned %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var out hermesResponse
	if err = json.Unmarshal(raw, &out); err != nil {
		return "", none, fmt.Errorf("decode hermes response: %w", err)
	}
	if out.Hermes != nil && out.Hermes.Failed {
		return "", none, &hermesRunError{msg: "hermes run failed: " + hermesErrorText(out.Hermes.Error)}
	}
	if len(out.Choices) == 0 {
		return "", none, errors.New("hermes returned no choices")
	}
	if out.Choices[0].FinishReason == "error" {
		return "", none, &hermesRunError{msg: "hermes run failed: finish_reason=error"}
	}

	usage := goopenai.Usage{
		PromptTokens:     out.Usage.PromptTokens,
		CompletionTokens: out.Usage.CompletionTokens,
		TotalTokens:      out.Usage.TotalTokens,
	}
	return out.Choices[0].Message.Content, usage, nil
}

// hermesErrorText renders Hermes' error field, which may be a plain string or a
// structured object.
func hermesErrorText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "no details"
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

// buildPreamble renders the synthetic system prompt that makes a text-only agent
// behave like a tool-calling model: the tool list plus the JSON protocol Hermes
// must answer in. Empty tools means pass-through — no preamble at all.
func buildPreamble(tools []goopenai.Tool) string {
	var lines []string
	for _, t := range tools {
		if t.Function == nil || t.Function.Name == "" {
			continue
		}
		line := "- " + t.Function.Name + "(" + strings.Join(schemaParams(t.Function.Parameters), ", ") + ")"
		if t.Function.Description != "" {
			line += " — " + t.Function.Description
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}

	return "You are driving a tool-calling agent. These tools are available:\n\n" +
		strings.Join(lines, "\n") + "\n\n" +
		"Reply with a SINGLE JSON object and nothing else:\n" +
		`{"tool_calls":[{"name":"<tool>","arguments":{...}}]}` + "\n\n" +
		"Rules:\n" +
		"- No prose, no explanations, no markdown code fences — the JSON object only.\n" +
		"- Use only the tool names listed above, with exactly their arguments.\n" +
		"- Results of your previous calls appear in the conversation; keep calling tools until the work is done.\n" +
		"- Call finish when the work is done."
}

// schemaParams renders a JSON-Schema parameters object as "name: type" entries:
// required ones first (in schema order), then the rest sorted, so the preamble
// is deterministic.
func schemaParams(parameters any) []string {
	if parameters == nil {
		return nil
	}
	raw, err := json.Marshal(parameters)
	if err != nil {
		return nil
	}
	var schema struct {
		Properties map[string]struct {
			Type any `json:"type"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err = json.Unmarshal(raw, &schema); err != nil || len(schema.Properties) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(schema.Properties))
	var names []string
	for _, name := range schema.Required {
		if _, ok := schema.Properties[name]; ok && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	var rest []string
	for name := range schema.Properties {
		if !seen[name] {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)

	out := make([]string, 0, len(schema.Properties))
	for _, name := range append(names, rest...) {
		out = append(out, name+": "+schemaTypeName(schema.Properties[name].Type))
	}
	return out
}

// schemaTypeName renders a JSON-Schema "type", which may be a string or a union
// array.
func schemaTypeName(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		parts := make([]string, 0, len(t))
		for _, p := range t {
			if s, ok := p.(string); ok {
				parts = append(parts, s)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "|")
		}
	}
	return "any"
}

// flattenMessages turns fleet's OpenAI transcript into the system/user/assistant
// text Hermes accepts. The preamble is prepended to the merged system text; tool
// results become user messages ("Result of <name>: ...") and an assistant turn's
// tool_calls become a description of the calls it made — that is what lets
// Hermes see its own prior results and eventually emit finish, since fleet
// re-sends the whole transcript every turn.
func flattenMessages(preamble string, messages []goopenai.ChatCompletionMessage) []hermesMessage {
	var systemParts []string
	if preamble != "" {
		systemParts = append(systemParts, preamble)
	}
	for _, m := range messages {
		if m.Role == goopenai.ChatMessageRoleSystem && m.Content != "" {
			systemParts = append(systemParts, m.Content)
		}
	}

	var out []hermesMessage
	if len(systemParts) > 0 {
		out = append(out, hermesMessage{Role: "system", Content: strings.Join(systemParts, "\n\n")})
	}

	// Tool results carry only a call id, so remember which call each name came
	// from while walking the transcript.
	callNames := map[string]string{}
	for _, m := range messages {
		switch m.Role {
		case goopenai.ChatMessageRoleSystem:
			continue
		case goopenai.ChatMessageRoleTool:
			out = append(out, hermesMessage{
				Role:    "user",
				Content: "Result of " + toolName(m, callNames) + ": " + m.Content,
			})
		case goopenai.ChatMessageRoleAssistant:
			var parts []string
			if m.Content != "" {
				parts = append(parts, m.Content)
			}
			for _, tc := range m.ToolCalls {
				callNames[tc.ID] = tc.Function.Name
				parts = append(parts, "Called "+tc.Function.Name+" with "+orEmptyObject(tc.Function.Arguments))
			}
			if len(parts) > 0 {
				out = append(out, hermesMessage{Role: "assistant", Content: strings.Join(parts, "\n")})
			}
		default:
			if m.Content != "" {
				out = append(out, hermesMessage{Role: "user", Content: m.Content})
			}
		}
	}
	return out
}

// toolName resolves the tool a result belongs to: the message's own name, else
// the name recorded for its call id, else a generic label.
func toolName(m goopenai.ChatCompletionMessage, callNames map[string]string) string {
	if m.Name != "" {
		return m.Name
	}
	if name := callNames[m.ToolCallID]; name != "" {
		return name
	}
	return "tool"
}

func orEmptyObject(s string) string {
	if s == "" {
		return "{}"
	}
	return s
}

// rawToolCall is one entry of Hermes' JSON answer. Arguments may be an object or
// an already-serialized string.
type rawToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// extractToolCalls finds the tool calls in Hermes' answer text. Hermes is asked
// for a bare JSON object, but LLM output drifts: markdown fences, leading prose,
// a single call instead of the wrapper. So the scan walks every `{`/`[`, takes
// the outermost balanced span starting there (respecting string literals and
// escapes) and returns the first span that parses into at least one named call.
// Nothing parseable returns nil, and the caller falls back to plain content.
func extractToolCalls(text, idPrefix string) []goopenai.ToolCall {
	for i := range len(text) {
		if text[i] != '{' && text[i] != '[' {
			continue
		}
		end := matchBalanced(text, i)
		if end < 0 {
			continue
		}
		if calls := parseToolCalls([]byte(text[i:end+1]), idPrefix); len(calls) > 0 {
			return calls
		}
	}
	return nil
}

// parseToolCalls accepts the three shapes Hermes emits: the wrapper object, a
// bare array of calls, and a single call object.
func parseToolCalls(raw []byte, idPrefix string) []goopenai.ToolCall {
	var wrapper struct {
		ToolCalls []rawToolCall `json:"tool_calls"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && len(wrapper.ToolCalls) > 0 {
		return toOpenAIToolCalls(wrapper.ToolCalls, idPrefix)
	}
	var list []rawToolCall
	if err := json.Unmarshal(raw, &list); err == nil && len(list) > 0 {
		return toOpenAIToolCalls(list, idPrefix)
	}
	// The single-object form has no `tool_calls` marker, so it is only accepted
	// when the object is shaped exactly like a call — otherwise a final answer
	// such as {"name":"Alice","age":30} would be mistaken for one.
	if !isToolCallObject(raw) {
		return nil
	}
	var single rawToolCall
	if err := json.Unmarshal(raw, &single); err == nil && single.Name != "" {
		return toOpenAIToolCalls([]rawToolCall{single}, idPrefix)
	}
	return nil
}

// isToolCallObject reports whether raw is a bare tool call: a non-empty string
// `name`, an `arguments` object or string, and no key beyond the id/type an
// OpenAI-shaped call may also carry.
func isToolCallObject(raw []byte) bool {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}

	var name string
	if err := json.Unmarshal(obj["name"], &name); err != nil || name == "" {
		return false
	}
	args := bytes.TrimSpace(obj["arguments"])
	if len(args) == 0 || (args[0] != '{' && args[0] != '"') {
		return false
	}
	for key := range obj {
		switch key {
		case "name", "arguments", "id", "type":
		default:
			return false
		}
	}
	return true
}

// toOpenAIToolCalls converts parsed calls into OpenAI tool_calls, dropping
// nameless entries (a call fleet could never dispatch).
func toOpenAIToolCalls(calls []rawToolCall, idPrefix string) []goopenai.ToolCall {
	var out []goopenai.ToolCall
	for _, c := range calls {
		if c.Name == "" {
			continue
		}
		out = append(out, goopenai.ToolCall{
			ID:   fmt.Sprintf("call_%s_%d", idPrefix, len(out)),
			Type: goopenai.ToolTypeFunction,
			Function: goopenai.FunctionCall{
				Name:      c.Name,
				Arguments: argumentsString(c.Arguments),
			},
		})
	}
	return out
}

// argumentsString normalises an arguments value to the compact JSON string
// OpenAI clients expect: an object is compacted, a string passes through, an
// absent value becomes an empty object.
func argumentsString(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return "{}"
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err == nil {
			return s
		}
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, trimmed); err != nil {
		return "{}"
	}
	return buf.String()
}

// matchBalanced returns the index closing the bracket opened at start, or -1 if
// the span is unbalanced. Brackets inside string literals (and escaped quotes)
// do not count.
func matchBalanced(s string, start int) int {
	var stack []byte
	inString, escaped := false, false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, c)
		case '}', ']':
			if len(stack) == 0 {
				return -1
			}
			open := stack[len(stack)-1]
			if (c == '}' && open != '{') || (c == ']' && open != '[') {
				return -1
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return i
			}
		}
	}
	return -1
}

func (s *Server) handleModels(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data": []map[string]any{{
			"id":       s.cfg.Model,
			"object":   "model",
			"owned_by": "trip2g",
		}},
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// apiError is the OpenAI-shaped error envelope.
type apiError struct {
	Error apiErrorBody `json:"error"`
}

type apiErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func writeError(w http.ResponseWriter, status int, errType, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{Error: apiErrorBody{Message: msg, Type: errType}})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// randomID returns a short hex token for the completion id.
func randomID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "000000000000"
	}
	return hex.EncodeToString(b[:])
}
