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
	"context"
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

	"github.com/99designs/gqlgen/graphql/playground"
	goopenai "github.com/sashabaranov/go-openai"

	"trip2g/cmd/codellm/internal/codellm/codellmgql"
	"trip2g/cmd/codellm/internal/codellmmetrics"
	"trip2g/cmd/codellm/internal/coderun"
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
	// (exact) and name prefixes codellm exposes from its OWN environment to
	// executed code, on every run. codellm owns the secrets, so codellm alone
	// decides what to expose — the request carries nothing about env. Both empty
	// (the default) means codellm exposes nothing: the secret-scrubbed
	// PATH+FLEET_INPUT child env.
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

	// Metrics is the Prometheus sink served on the internal listener. nil
	// disables recording entirely (every call site is nil-safe), which is what
	// tests and a --metrics-addr="" run get.
	Metrics *codellmmetrics.Metrics
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
	return BrowserAuthWithMetrics(admin, tokenCheck, nil)
}

// BrowserAuthWithMetrics is BrowserAuth plus auth accounting: which lane
// admitted (or rejected) each request. The cookie lane's verdict is only
// visible from the response status the admin middleware wrote, so it is read
// back off the wrapped writer. m may be nil.
func BrowserAuthWithMetrics(
	admin func(http.Handler) http.Handler,
	tokenCheck func(*http.Request) error,
	m *codellmmetrics.Metrics,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		gated := admin(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tokenCheck != nil {
				if err := tokenCheck(r); err == nil {
					m.RecordAuth(codellmmetrics.LaneAPIKey, codellmmetrics.Allowed)
					next.ServeHTTP(w, r)
					return
				}
			}
			rec := &authStatusWriter{ResponseWriter: w, status: http.StatusOK}
			gated.ServeHTTP(rec, r)
			result := codellmmetrics.Allowed
			if rec.status == http.StatusUnauthorized {
				result = codellmmetrics.Denied
			}
			m.RecordAuth(codellmmetrics.LaneCookie, result)
		})
	}
}

// authStatusWriter captures the status the admin gate wrote so the cookie
// lane's allow/deny verdict can be counted.
type authStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *authStatusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Server is the codellm HTTP service.
type Server struct {
	cfg Config
}

type blockRunner struct{ cfg Config }

func (r blockRunner) RunBlocks(ctx context.Context, req codellmgql.BlockRunRequest) (codellmgql.BlockRunResult, error) {
	observe, blocks := blockObserver(r.cfg.Metrics)
	out, debug, err := coderun.ExecBlocksDebug(ctx, coderun.CodeInput{
		Body: req.Body, Input: req.FleetInput,
		AllowedPrograms: r.cfg.AllowedPrograms,
		Sandbox:         r.cfg.Sandbox, MaxStdoutBytes: r.cfg.MaxStdoutBytes,
		Timeout: r.cfg.Timeout, EnvPassthrough: r.cfg.ExposeEnv,
		EnvPrefix: r.cfg.ExposeEnvPrefix,
		Observe:   observe,
	}, req.MaxSteps)
	r.cfg.Metrics.ObserveRequestBlocks(*blocks)
	if err != nil {
		r.cfg.Metrics.RecordExecError(err)
		return codellmgql.BlockRunResult{}, err
	}
	results := make([]codellmgql.BlockResult, 0, len(debug))
	for _, d := range debug {
		results = append(results, codellmgql.BlockResult{
			Index: d.Index, ExitCode: d.ExitCode, Stdout: d.Stdout,
			Stderr: d.Stderr, DurationMs: d.DurationMs, MaxRSSBytes: d.MaxRSSBytes,
		})
	}
	return codellmgql.BlockRunResult{Output: out, Results: results}, nil
}

// New builds a Server from cfg, filling in nil seams with no-op defaults.
func New(cfg Config) *Server {
	if cfg.Auth == nil {
		cfg.Auth = func(next http.Handler) http.Handler { return next }
	}
	return &Server{cfg: cfg}
}

// Handler returns the HTTP handler for the service. The two BROWSER-facing
// endpoints — execution (/v1/chat/completions) and the markdown-structure
// GraphQL (/_system/codellm/graphql) — are wrapped by the
// auth seam (cfg.Auth); everything else (liveness /healthz, /v1/models for
// client compat) is open. See the two-auth-regime note on Config.Auth.
func (s *Server) Handler() http.Handler {
	const graphqlPrefix = "/_system/codellm/graphql"

	m := s.cfg.Metrics
	mux := http.NewServeMux()
	graphqlHandler := codellmgql.NewHTTPHandler(nil, blockRunner{cfg: s.cfg})
	gqlAuthHandler := s.cfg.Auth(graphqlHandler)
	// Auth-gated (browser cookie / fleet channel token — see cfg.Auth).
	mux.Handle("POST /v1/chat/completions",
		m.Instrument("chat_completions", s.cfg.Auth(http.HandlerFunc(s.handleChatCompletions))))
	mux.Handle("POST "+graphqlPrefix, m.Instrument("graphql", gqlAuthHandler))
	mux.Handle("GET "+graphqlPrefix,
		m.Instrument("graphql_playground", s.cfg.Auth(playground.Handler("CodeLLM GraphQL Playground", graphqlPrefix))))
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

	body, bag := extractBodyAndBag(req.Messages)

	// codellm alone decides what env to expose: its operator allowlist
	// (ExposeEnv/ExposeEnvPrefix). buildChildEnv sources the VALUES from codellm's
	// OWN environment for those names; the request carries nothing about env.
	s.logExposedEnv()
	observe, blocks := blockObserver(s.cfg.Metrics)
	in := coderun.CodeInput{
		Body:            body,
		AllowedPrograms: s.cfg.AllowedPrograms,
		Sandbox:         s.cfg.Sandbox,
		MaxStdoutBytes:  s.cfg.MaxStdoutBytes,
		Timeout:         s.cfg.Timeout,
		Input:           bag,
		EnvPassthrough:  s.cfg.ExposeEnv,
		EnvPrefix:       s.cfg.ExposeEnvPrefix,
		Observe:         observe,
	}

	changes, answer, err := coderun.ExecCode(r.Context(), in)
	s.cfg.Metrics.ObserveRequestBlocks(*blocks)
	if err != nil {
		// Apply-error semantics = hard-fail 422 (decision b): a failing block,
		// timeout, or unparseable stdout is a deterministic failure, not a soft
		// skip. 422 (not 5xx) so fleet's OpenAILLM does not retry it.
		s.cfg.Metrics.RecordExecError(err)
		writeError(w, http.StatusUnprocessableEntity, "code_execution_error", err.Error())
		return
	}
	for _, ch := range changes {
		s.cfg.Metrics.RecordChange(ch.Kind)
	}

	resp := buildResponse(req.Model, changes, answer)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// extractBodyAndBag concatenates the message contents that may contain fenced
// code (everything except the reserved fleet_input message) into the body to
// scan, and returns the fleet_input message content as the delivery bag. The
// system-prompt wrapper has no fences, so only the role body's blocks are found.
func extractBodyAndBag(messages []goopenai.ChatCompletionMessage) (string, []byte) {
	var sb strings.Builder
	var bag []byte
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

// logExposedEnv records, on each run with a non-empty allowlist, which env var
// NAMES codellm exposes from its own env to the code child (never the values).
// The allowlist IS the whole decision — the request carries nothing about env.
func (s *Server) logExposedEnv() {
	if len(s.cfg.ExposeEnv) == 0 && len(s.cfg.ExposeEnvPrefix) == 0 {
		return
	}
	//nolint:sloglint // codellm has no logger instance; global slog is intentional here
	slog.Info("codellm: exposing allowlisted env to code child (names only)",
		"passthrough", s.cfg.ExposeEnv, "prefix", s.cfg.ExposeEnvPrefix)
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

// blockObserver builds the coderun observe hook that feeds the block metrics,
// plus the per-request block counter it increments. The counter is request-
// local: coderun calls the observer from the request's own goroutine.
func blockObserver(m *codellmmetrics.Metrics) (func(coderun.BlockStats), *int) {
	n := 0
	return func(st coderun.BlockStats) {
		n++
		m.RecordBlock(st)
	}, &n
}
