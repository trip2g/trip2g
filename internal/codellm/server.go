// Package codellm implements a standalone OpenAI-compatible chat-completions
// service that *pretends to be an LLM*: it receives chat messages containing
// markdown-with-fenced-code, executes the fenced blocks, and returns the writes
// as OpenAI tool_calls (write_note / patch_note / finish).
//
// This is Phase 1 of docs/dev/codellm_extraction.md: the wire protocol + the
// {changes}→tool_calls mapping + the hard invariants (always-finish, 422 on
// apply/parse failure). It reuses internal/agentruntime for execution (RunBlock /
// pipeline / sandbox) — the sandbox move (Phase 2) and the fleet cutover
// (Phase 3) are deliberately NOT part of this service yet. codellm holds no
// vault, no KB, no auth, no secrets; scope enforcement stays in fleet.
package codellm

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	goopenai "github.com/sashabaranov/go-openai"

	"trip2g/internal/agentruntime"
	"trip2g/internal/webhookutil"
)

// fleetInputMessageName is the reserved message name that carries the delivery
// bag (the JSON exposed to executed code as $FLEET_INPUT). It rides as a system
// message per the wire protocol; its content is NOT scanned for fenced blocks.
const fleetInputMessageName = "fleet_input"

// modelID is the synthetic model advertised by GET /v1/models. codellm echoes
// whatever model the request asks for, so this is only for client compatibility.
const modelID = "codellm"

// Config configures a codellm Server. codellm holds no vault/KB/secrets; the
// only knobs are the code-execution policy and the auth seam.
type Config struct {
	// AllowedPrograms is the interpreter allowlist (python, bash, node, ...).
	// Empty disables code execution (every request then fails 422). Phase 2
	// moves the full --allowed-programs/--sandbox/... flag set here.
	AllowedPrograms []string

	// Sandbox is the OS-level isolation policy for each executed block. The zero
	// value is the safe default (native, enforcing). Phase 2 wires
	// MaybeRunSandboxChild() so native mode is fully operational.
	Sandbox agentruntime.SandboxPolicy

	// MaxStdoutBytes caps each block's captured stdout; 0 → 1 MiB default.
	MaxStdoutBytes int

	// Timeout bounds a single completion's code run; 0 → bounded by the request
	// context only.
	Timeout time.Duration

	// Auth is the middleware SEAM for the delegated-admin / shared-token check.
	// The fleet↔codellm channel MUST be locked down (mTLS / shared token /
	// loopback) before exposure — an unauthenticated /v1/chat/completions is
	// RCE-as-a-service. A separate PR builds the shared auth helper and wires it
	// here; nil defaults to a no-op passthrough for now.
	Auth func(http.Handler) http.Handler

	// TokenCheck is the seam for the fleet↔codellm shared-token/mTLS check.
	// Not built in Phase 1 (see the channel-locking note above); nil → no check.
	// Kept distinct from Auth so the transport-lock and the caller-identity
	// concerns can be wired independently.
	TokenCheck func(*http.Request) error
}

// Server is the codellm HTTP service.
type Server struct {
	cfg Config
}

// New builds a Server from cfg, filling in nil seams with no-op defaults.
func New(cfg Config) *Server {
	if cfg.Auth == nil {
		cfg.Auth = func(next http.Handler) http.Handler { return next }
	}
	return &Server{cfg: cfg}
}

// Handler returns the HTTP handler for the service, with the auth seam applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("GET /v1/models", s.handleModels)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	return s.cfg.Auth(mux)
}

// handleChatCompletions is the OpenAI-compatible surface. It decodes the chat
// request, extracts the markdown-with-code and the delivery bag, executes the
// fenced blocks, and returns the writes as tool_calls ending in exactly one
// finish. Execution/parse failures are a deterministic 422 (not retried).
func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if s.cfg.TokenCheck != nil {
		if err := s.cfg.TokenCheck(r); err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}
	}

	var req goopenai.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "decode request: "+err.Error())
		return
	}

	body, bag := extractBodyAndBag(req.Messages)

	changes, answer, err := agentruntime.ExecCode(r.Context(), agentruntime.CodeInput{
		Body:            body,
		AllowedPrograms: s.cfg.AllowedPrograms,
		Sandbox:         s.cfg.Sandbox,
		MaxStdoutBytes:  s.cfg.MaxStdoutBytes,
		Timeout:         s.cfg.Timeout,
		Input:           bag,
		// No EnvPassthrough/EnvPrefix: env passthrough is dropped by design —
		// no parent env crosses the HTTP boundary.
	})
	if err != nil {
		// Apply-error semantics = hard-fail 422 (decision b): a failing block,
		// timeout, or unparseable stdout is a deterministic failure, not a soft
		// skip. 422 (not 5xx) so fleet's OpenAILLM does not retry it.
		writeError(w, http.StatusUnprocessableEntity, "code_execution_error", err.Error())
		return
	}

	resp := buildResponse(req.Model, changes, answer)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// extractBodyAndBag concatenates the message contents that may contain fenced
// code (everything except the reserved fleet_input message) into the body to
// scan, and returns the fleet_input message content as the delivery bag. The
// system-prompt wrapper has no fences, so only the role body's blocks are found.
func extractBodyAndBag(messages []goopenai.ChatCompletionMessage) (body string, bag []byte) {
	var sb strings.Builder
	for _, m := range messages {
		if m.Name == fleetInputMessageName {
			bag = []byte(m.Content)
			continue
		}
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}
	return sb.String(), bag
}

// buildResponse maps the parsed {changes, answer} onto an OpenAI tool_calls
// completion: content-change → write_note(path, content), find/replace →
// patch_note(path, find, replace), then a trailing finish(answer) kept LAST.
//
// Hard invariant: every successful completion ends with exactly ONE finish
// tool_call — even an empty answer becomes finish(""). This is the
// re-execution defense: fleet's stateless loop re-calls codellm (which
// re-extracts and re-applies every write) unless a finish stops it.
func buildResponse(model string, changes []webhookutil.AgentChange, answer string) goopenai.ChatCompletionResponse {
	toolCalls := make([]goopenai.ToolCall, 0, len(changes)+1)
	for i, ch := range changes {
		name, args := changeToToolCall(ch)
		toolCalls = append(toolCalls, goopenai.ToolCall{
			ID:       fmt.Sprintf("call_%d", i),
			Type:     goopenai.ToolTypeFunction,
			Function: goopenai.FunctionCall{Name: name, Arguments: args},
		})
	}
	// finish is always LAST and always present (even with an empty answer).
	toolCalls = append(toolCalls, goopenai.ToolCall{
		ID:       fmt.Sprintf("call_%d", len(changes)),
		Type:     goopenai.ToolTypeFunction,
		Function: goopenai.FunctionCall{Name: "finish", Arguments: mustJSON(map[string]string{"answer": answer})},
	})

	if model == "" {
		model = modelID
	}
	return goopenai.ChatCompletionResponse{
		ID:      "codellm-" + randomID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []goopenai.ChatCompletionChoice{{
			Index: 0,
			Message: goopenai.ChatCompletionMessage{
				Role:      goopenai.ChatMessageRoleAssistant,
				Content:   "",
				ToolCalls: toolCalls,
			},
			FinishReason: goopenai.FinishReasonToolCalls,
		}},
		// codellm does no token accounting: usage is always zero (matching the
		// old TokensUsed: 0 for in-process code runs).
		Usage: goopenai.Usage{},
	}
}

// changeToToolCall maps one AgentChange to a tool name and JSON arguments.
func changeToToolCall(ch webhookutil.AgentChange) (name, args string) {
	if ch.Kind == webhookutil.AgentChangeKindPatch {
		return "patch_note", mustJSON(map[string]string{
			"path":    ch.Path,
			"find":    ch.Find,
			"replace": ch.Replace,
		})
	}
	return "write_note", mustJSON(map[string]string{
		"path":    ch.Path,
		"content": ch.Content,
	})
}

func (s *Server) handleModels(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data": []map[string]any{{
			"id":       modelID,
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

// randomID returns a short hex token for the completion id.
func randomID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "000000000000"
	}
	return hex.EncodeToString(b[:])
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		// The inputs are plain string maps; marshaling cannot fail.
		panic(errors.New("codellm: marshal tool arguments: " + err.Error()))
	}
	return string(b)
}
