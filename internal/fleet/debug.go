package fleet

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"trip2g/internal/agentruntime"
)

// Debug surface: a localhost-only HTTP handler for stepping through a code
// role's block pipeline one block at a time. It runs a single fenced block
// with a caller-supplied stdin and returns stdout/stderr/exit + timing, so a
// developer can inspect intermediate pipe output. Stateless by design: each
// request carries everything needed to run one block.
//
// Enabled only via --debug-listen (loopback-only, see ValidateLoopbackAddr);
// never mounted on the public delivery listener. It reuses RunBlock, so the
// sandbox and secret-scrubbed env apply exactly as in real deliveries.

// debugRunBlockDefaultTimeout bounds an inline-body block run; role-based runs
// use the role's own timeout.
const debugRunBlockDefaultTimeout = 60 * time.Second

// debugRunBlockRequest selects one block to run: either from a registered
// role's body (Role = note path, unrendered) or from an inline markdown Body.
type debugRunBlockRequest struct {
	Role  string          `json:"role"`  // registered role note path; takes precedence over Body
	Body  string          `json:"body"`  // inline markdown with fenced code blocks
	Block int             `json:"block"` // 0-based block index
	Stdin string          `json:"stdin"` // piped to the block's stdin (prior block's stdout when stepping)
	Input json.RawMessage `json:"input"` // optional delivery bag JSON exposed via $FLEET_INPUT
}

// debugRunBlockResponse reports one block run. OK=false with a 200 status is a
// block-level failure (non-zero exit / timeout) — the thing being debugged.
type debugRunBlockResponse struct {
	Block       int    `json:"block"`
	BlocksTotal int    `json:"blocks_total"`
	Lang        string `json:"lang"`
	Program     string `json:"program"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	OK          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
	DurationMs  int64  `json:"duration_ms"`
}

// DebugHandler returns the mux served on the localhost debug listener.
func (f *Fleet) DebugHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/run-block", f.serveDebugRunBlock)
	return mux
}

// serveDebugRunBlock handles POST /debug/run-block.
func (f *Fleet) serveDebugRunBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req debugRunBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	body := req.Body
	timeout := debugRunBlockDefaultTimeout
	var envPassthrough, envPrefix []string
	if req.Role != "" {
		role, ok := f.roleByKey(urlKey(req.Role))
		if !ok {
			http.Error(w, "unknown role "+req.Role, http.StatusNotFound)
			return
		}
		// The role body is used unrendered (no Jet evaluation): debug stepping
		// targets the static fenced blocks. POST an inline body for a rendered
		// variant.
		body = role.Body
		envPassthrough, envPrefix = role.EnvPassthrough, role.EnvPrefix
		timeout = time.Duration(role.EffectiveTimeoutSeconds()) * time.Second
	}

	blocks := agentruntime.ExtractFencedBlocks(body)
	if len(blocks) == 0 {
		http.Error(w, "no fenced code block found", http.StatusBadRequest)
		return
	}
	if req.Block < 0 || req.Block >= len(blocks) {
		http.Error(w, fmt.Sprintf("block index %d out of range (have %d blocks)", req.Block, len(blocks)), http.StatusBadRequest)
		return
	}
	block := blocks[req.Block]

	program := agentruntime.ProgramForFenceLang(block.Lang)
	if program == "" {
		http.Error(w, fmt.Sprintf("fence language %q not supported", block.Lang), http.StatusBadRequest)
		return
	}
	if !isAllowedProgram(program, f.cfg.AllowedPrograms) {
		http.Error(w, fmt.Sprintf("program %q not in allowed programs %v", program, f.cfg.AllowedPrograms), http.StatusBadRequest)
		return
	}

	start := time.Now()
	stdout, stderr, _, runErr := agentruntime.RunBlock(r.Context(), agentruntime.CodeSpec{
		Program:        program,
		Code:           block.Code,
		Stdin:          []byte(req.Stdin),
		Input:          req.Input,
		Timeout:        timeout,
		EnvPassthrough: envPassthrough,
		EnvPrefix:      envPrefix,
		MaxStdoutBytes: f.cfg.MaxStdoutBytes,
		Sandbox:        f.sandboxPolicy(),
	})

	resp := debugRunBlockResponse{
		Block:       req.Block,
		BlocksTotal: len(blocks),
		Lang:        block.Lang,
		Program:     program,
		Stdout:      stdout,
		Stderr:      stderr,
		OK:          runErr == nil,
		DurationMs:  time.Since(start).Milliseconds(),
	}
	if runErr != nil {
		resp.Error = runErr.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}

// isAllowedProgram mirrors agentruntime's allowlist check (empty = disabled).
func isAllowedProgram(program string, allowed []string) bool {
	for _, p := range allowed {
		if p == program {
			return true
		}
	}
	return false
}

// ValidateLoopbackAddr ensures addr binds a loopback interface only
// (127.0.0.0/8, [::1], or the literal "localhost"). The debug surface must
// never be reachable from other hosts.
func ValidateLoopbackAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("debug listen address %q: %w", addr, err)
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("debug listener must bind loopback (127.0.0.1, [::1], or localhost), got %q", addr)
	}
	return nil
}
