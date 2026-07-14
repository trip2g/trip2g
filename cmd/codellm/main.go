// Command codellm is a standalone OpenAI-compatible chat-completions service
// that executes the fenced code in incoming markdown and returns the writes as
// OpenAI tool_calls. See docs/dev/codellm_extraction.md (Phase 1).
//
// SECURITY: an unauthenticated /v1/chat/completions is RCE-as-a-service. The
// fleet↔codellm channel MUST be locked (mTLS / shared token / loopback) before
// exposure — the auth/token seams in internal/codellm are wired by a later PR.
package main

import (
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	"trip2g/internal/agentruntime"
	"trip2g/internal/codellm"
)

func main() {
	// The per-block sandbox works by re-execing THIS binary as a confined child
	// (marker env var). It must run first so a re-exec lands in the child branch
	// and never falls through to start a second server. No-op when the marker is
	// absent. Phase 2 completes the sandbox story (move + tests).
	agentruntime.MaybeRunSandboxChild()

	addr := flag.String("addr", "127.0.0.1:8082", "listen address for the OpenAI-compatible API; defaults to loopback since auth is a no-op seam — binding to all interfaces is an explicit operator opt-in")
	allowedPrograms := flag.String("allowed-programs", "python,bash,node", "comma-separated interpreter allowlist; empty disables code execution")
	sandboxMode := flag.String("sandbox", string(agentruntime.SandboxNative), "sandbox mode: native | besteffort | off")
	timeout := flag.Duration("timeout", 300*time.Second, "per-completion code-run timeout; 0 = request-context bound")
	maxStdout := flag.Int("max-stdout-bytes", 0, "stdout cap per code block; 0 = 1 MiB default")
	flag.Parse()

	cfg := codellm.Config{
		AllowedPrograms: splitCSV(*allowedPrograms),
		Sandbox:         agentruntime.SandboxPolicy{Mode: agentruntime.SandboxMode(*sandboxMode)},
		MaxStdoutBytes:  *maxStdout,
		Timeout:         *timeout,
		// Auth/TokenCheck seams intentionally left nil (no-op) in Phase 1 — a
		// separate PR builds the shared delegated-admin auth helper and the
		// channel token/mTLS check and wires them here.
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           codellm.New(cfg).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("codellm listening on %s (sandbox=%s, programs=%v)", *addr, *sandboxMode, cfg.AllowedPrograms)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("codellm: server error: %v", err)
	}
}

// splitCSV splits a comma-separated flag value into a trimmed, empty-free slice.
func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
