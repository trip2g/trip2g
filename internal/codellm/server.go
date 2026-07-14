// Package codellm implements a standalone OpenAI-compatible chat-completions
// service that *pretends to be an LLM*: it receives chat messages containing
// markdown-with-fenced-code, executes the fenced blocks, and returns the writes
// as OpenAI tool_calls (write_note / patch_note / finish).
//
// Per docs/dev/codellm_extraction.md: the wire protocol + the {changes}→tool_calls
// mapping + the hard invariants (always-finish, 422 on apply/parse failure). It
// uses internal/coderun for execution (RunBlock / pipeline / sandbox), which
// Phase 2 moved out of internal/agentruntime; the fleet cutover (Phase 3) is
// deliberately NOT part of this service yet. codellm holds no vault, no KB, no
// auth, no secrets; scope enforcement stays in fleet.
package codellm

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	goopenai "github.com/sashabaranov/go-openai"

	"trip2g/internal/codellm/codellmgql"
	"trip2g/internal/coderun"
	"trip2g/internal/webhookutil"
)

// fleetInputMessageName is the reserved message name that carries the delivery
// bag (the JSON exposed to executed code as $FLEET_INPUT). It rides as a system
// message per the wire protocol; its content is NOT scanned for fenced blocks.
const fleetInputMessageName = "fleet_input"

// fleetEnvMessageName is the reserved message name carrying the code role's
// declared env var NAMES (JSON: {"passthrough":[...],"prefix":[...]}). codellm
// intersects them with its own expose-allowlist and forwards the surviving vars
// from ITS OWN env to the code child. Names ride the wire; values never do. Its
// content is NOT scanned for fenced blocks.
const fleetEnvMessageName = "fleet_env"

// fleetEnvNames is the JSON payload of the fleet_env message.
type fleetEnvNames struct {
	Passthrough []string `json:"passthrough,omitempty"`
	Prefix      []string `json:"prefix,omitempty"`
}

// modelID is the synthetic model advertised by GET /v1/models. codellm echoes
// whatever model the request asks for, so this is only for client compatibility.
const modelID = "codellm"

// Config configures a codellm Server. codellm holds no vault/KB/secrets; the
// only knobs are the code-execution policy and the auth seam.
type Config struct {
	// AllowedPrograms is the interpreter allowlist (python, bash, node, ...).
	// Empty disables code execution (every request then fails 422).
	AllowedPrograms []string

	// Sandbox is the OS-level isolation policy for each executed block. The zero
	// value is the safe default (native, enforcing). main.go calls
	// coderun.MaybeRunSandboxChild() first and defaults --sandbox to native, so
	// each executed block gets the full native posture.
	Sandbox coderun.SandboxPolicy

	// MaxStdoutBytes caps each block's captured stdout; 0 → 1 MiB default.
	MaxStdoutBytes int

	// ExposeEnv / ExposeEnvPrefix are the operator's allowlist of env var NAMES
	// (exact) and name prefixes that codellm MAY expose from its OWN environment
	// to executed code. A request's fleet_env names are INTERSECTED with these —
	// a request can never reach beyond the allowlist. Both empty (the default)
	// means codellm exposes nothing: the secret-scrubbed PATH+FLEET_INPUT child
	// env, as before.
	ExposeEnv       []string
	ExposeEnvPrefix []string

	// Timeout bounds a single completion's code run; 0 → bounded by the request
	// context only.
	Timeout time.Duration

	// Auth gates the two BROWSER-facing endpoints (/v1/chat/completions and
	// /graphql). In production it is BrowserAuth(delegatedAdmin, TokenCheck): a
	// caller presenting codellm's own OpenAI-standard api_key (Authorization:
	// Bearer) is served directly via TokenCheck; otherwise the request falls
	// through to the delegated-admin cookie check (cookie → monolith
	// viewer{role}; admin → serve, else 401). nil defaults to a no-op passthrough
	// (tests / standalone).
	Auth func(http.Handler) http.Handler

	// TokenCheck is the seam for codellm's own api_key check. In production it is
	// APIKeyCheck(cfg.APIKey): codellm's OpenAI-standard api_key, presented as
	// `Authorization: Bearer <api_key>` and compared in constant time — the same
	// credential shape any OpenAI-compatible endpoint has, so any client
	// (fleet's OpenAILLM included) authenticates identically to a real LLM
	// endpoint. An empty configured key disables key auth (fail-safe). nil → no
	// key auth, every request must pass the browser admin-cookie gate (the
	// secure default). Kept distinct from Auth so the transport-lock and the
	// caller-identity concerns wire independently.
	TokenCheck func(*http.Request) error
}

// BrowserAuth composes codellm's two auth regimes into one middleware for the
// browser-facing endpoints:
//
//   - API-key lane (any OpenAI-compatible client, fleet included): when
//     tokenCheck is non-nil AND passes — the caller presented codellm's own
//     api_key (Authorization: Bearer) — the request is served directly. This is
//     the standard "base_url + api_key" shape any OpenAI endpoint has; codellm
//     is no different, which keeps it interchangeable with a real LLM.
//   - Browser lane (Caddy /_codellm/* forwards the session cookie): gated by the
//     admin middleware — the delegated-admin check (cookie → monolith
//     viewer{role}; admin → serve, else 401, monolith-unreachable → fail-closed).
//
// tokenCheck nil → no key auth; every request must pass the browser admin gate
// (the secure default until an api_key is configured).
func BrowserAuth(admin func(http.Handler) http.Handler, tokenCheck func(*http.Request) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		gated := admin(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tokenCheck != nil {
				if err := tokenCheck(r); err == nil {
					next.ServeHTTP(w, r)
					return
				}
			}
			gated.ServeHTTP(w, r)
		})
	}
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

// Handler returns the HTTP handler for the service. The two BROWSER-facing
// endpoints — execution (/v1/chat/completions, incl. the x_fleet_debug
// extension) and the markdown-structure GraphQL (/graphql) — are wrapped by the
// auth seam (cfg.Auth); everything else (liveness /healthz, /v1/models for
// client compat) is open. See the two-auth-regime note on Config.Auth.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// Auth-gated (browser cookie / fleet channel token — see cfg.Auth).
	mux.Handle("POST /v1/chat/completions", s.cfg.Auth(http.HandlerFunc(s.handleChatCompletions)))
	mux.Handle("/graphql", s.cfg.Auth(codellmgql.NewHTTPHandler(nil)))
	// Open: liveness + client compatibility, never gated.
	mux.HandleFunc("GET /v1/models", s.handleModels)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	return mux
}

// handleChatCompletions is the OpenAI-compatible surface. It decodes the chat
// request, extracts the markdown-with-code and the delivery bag, executes the
// fenced blocks, and returns the writes as tool_calls ending in exactly one
// finish. Execution/parse failures are a deterministic 422 (not retried).
func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	// Auth is composed entirely in cfg.Auth (BrowserAuth): the api_key check and
	// the browser cookie gate are both decided before this handler runs. A second
	// TokenCheck here would be contradictory — it would REQUIRE the key on a
	// request the cookie lane already admitted (no key), breaking the browser
	// debugger lane.
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

	// x_fleet_debug is an opt-in EXTENSION field a real LLM ignores. When set, the
	// response carries a non-standard x_fleet_debug block (per-block stdout + the
	// inter-block pipe buffer). It is decoded separately because the standard
	// ChatCompletionRequest struct drops unknown fields.
	var ext struct {
		XFleetDebug bool `json:"x_fleet_debug"`
	}
	_ = json.Unmarshal(raw, &ext)

	body, bag, envReq := extractBodyAndBag(req.Messages)

	// The request names env vars it wants; codellm exposes only those on its
	// operator allowlist, sourcing the VALUES from its OWN env (buildChildEnv).
	// Shared by the normal and x_fleet_debug paths (both use `in`).
	passthrough, prefix := s.exposedEnv(envReq)

	in := coderun.CodeInput{
		Body:            body,
		AllowedPrograms: s.cfg.AllowedPrograms,
		Sandbox:         s.cfg.Sandbox,
		MaxStdoutBytes:  s.cfg.MaxStdoutBytes,
		Timeout:         s.cfg.Timeout,
		Input:           bag,
		// Env values are supplied from codellm's OWN environment for the
		// allowlist-intersected names only; the request carried names, not values.
		EnvPassthrough: passthrough,
		EnvPrefix:      prefix,
	}

	if ext.XFleetDebug {
		s.respondWithDebug(w, r, req.Model, in)
		return
	}

	changes, answer, err := coderun.ExecCode(r.Context(), in)
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

// respondWithDebug runs the code with per-block capture (ExecCodeDebug) and
// returns the standard completion PLUS the non-standard x_fleet_debug field.
// A normal OpenAI client ignores the extra field; the debugger reads it to show
// each block's stdout and the editable inter-block pipe buffer.
func (s *Server) respondWithDebug(w http.ResponseWriter, r *http.Request, model string, in coderun.CodeInput) {
	changes, answer, blocks, err := coderun.ExecCodeDebug(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "code_execution_error", err.Error())
		return
	}

	resp := debugResponse{
		ChatCompletionResponse: buildResponse(model, changes, answer),
		XFleetDebug:            &fleetDebug{Blocks: toDebugBlocks(blocks)},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// debugResponse is a standard completion plus the x_fleet_debug extension. The
// embedded ChatCompletionResponse marshals its fields inline, so a normal client
// sees an ordinary completion and simply ignores the extra x_fleet_debug key.
type debugResponse struct {
	goopenai.ChatCompletionResponse
	XFleetDebug *fleetDebug `json:"x_fleet_debug,omitempty"`
}

type fleetDebug struct {
	Blocks []fleetDebugBlock `json:"blocks"`
}

type fleetDebugBlock struct {
	Index      int    `json:"index"`
	Stdout     string `json:"stdout"`
	PipeBuffer string `json:"pipeBuffer"`
}

func toDebugBlocks(blocks []coderun.BlockDebug) []fleetDebugBlock {
	out := make([]fleetDebugBlock, len(blocks))
	for i, b := range blocks {
		out[i] = fleetDebugBlock{Index: b.Index, Stdout: b.Stdout, PipeBuffer: b.PipeBuffer}
	}
	return out
}

// extractBodyAndBag concatenates the message contents that may contain fenced
// code (everything except the reserved fleet_input / fleet_env messages) into
// the body to scan, and returns the fleet_input content as the delivery bag plus
// the fleet_env content as the requested env NAMES. The system-prompt wrapper
// has no fences, so only the role body's blocks are found.
func extractBodyAndBag(messages []goopenai.ChatCompletionMessage) (string, []byte, fleetEnvNames) {
	var sb strings.Builder
	var bag []byte
	var env fleetEnvNames
	for _, m := range messages {
		switch m.Name {
		case fleetInputMessageName:
			bag = []byte(m.Content)
			continue
		case fleetEnvMessageName:
			// Malformed fleet_env is non-fatal: no names → nothing exposed.
			_ = json.Unmarshal([]byte(m.Content), &env)
			continue
		}
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}
	return sb.String(), bag, env
}

// exposedEnv intersects a request's declared env NAMES (from fleet_env) with
// codellm's operator allowlist (ExposeEnv exact / ExposeEnvPrefix prefixes).
// Only allowlisted entries survive — a request can never reach an env var the
// operator did not permit. buildChildEnv then supplies the VALUES from codellm's
// own environment. The surviving NAMES are logged (never values).
func (s *Server) exposedEnv(req fleetEnvNames) ([]string, []string) {
	allowName := make(map[string]struct{}, len(s.cfg.ExposeEnv))
	for _, n := range s.cfg.ExposeEnv {
		allowName[n] = struct{}{}
	}
	var passthrough []string
	for _, n := range req.Passthrough {
		if _, ok := allowName[n]; ok {
			passthrough = append(passthrough, n)
		}
	}

	allowPrefix := make(map[string]struct{}, len(s.cfg.ExposeEnvPrefix))
	for _, p := range s.cfg.ExposeEnvPrefix {
		allowPrefix[p] = struct{}{}
	}
	var prefix []string
	for _, p := range req.Prefix {
		if _, ok := allowPrefix[p]; ok {
			prefix = append(prefix, p)
		}
	}

	if len(passthrough) > 0 || len(prefix) > 0 {
		//nolint:sloglint // codellm has no logger instance; global slog is intentional here
		slog.Info("codellm: exposing allowlisted env to code child (names only)",
			"passthrough", passthrough, "prefix", prefix)
	}
	return passthrough, prefix
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
func changeToToolCall(ch webhookutil.AgentChange) (string, string) {
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
