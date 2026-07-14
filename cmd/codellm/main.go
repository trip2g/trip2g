// Command codellm is a standalone OpenAI-compatible chat-completions service
// that executes the fenced code in incoming markdown and returns the writes as
// OpenAI tool_calls. See docs/dev/codellm_extraction.md (Phase 1).
//
// SECURITY: an unauthenticated /v1/chat/completions is RCE-as-a-service. The
// fleet↔codellm channel MUST be locked (mTLS / shared token / loopback) before
// exposure — the auth/token seams in internal/codellm are wired by a later PR.
package main

import (
	"log"
	"net/http"
	"time"

	"trip2g/internal/agentruntime"
	"trip2g/internal/codellm"

	"trip2g/cmd/codellm/appconfig"
)

func main() {
	// The per-block sandbox works by re-execing THIS binary as a confined child
	// (marker env var). It must run first so a re-exec lands in the child branch
	// and never falls through to start a second server. No-op when the marker is
	// absent. Phase 2 completes the sandbox story (move + tests).
	agentruntime.MaybeRunSandboxChild()

	cfg, err := appconfig.Get()
	if err != nil {
		log.Fatalf("codellm: config: %v", err)
	}

	srvCfg := codellm.Config{
		AllowedPrograms: cfg.AllowedPrograms,
		Sandbox:         agentruntime.SandboxPolicy{Mode: cfg.Sandbox},
		MaxStdoutBytes:  cfg.MaxStdoutBytes,
		Timeout:         cfg.Timeout,
		// Auth/TokenCheck seams intentionally left nil (no-op) in Phase 1 — a
		// separate PR builds the shared delegated-admin auth helper and the
		// channel token/mTLS check (cfg.ChannelToken is the placeholder for it).
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           codellm.New(srvCfg).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("codellm listening on %s (sandbox=%s, programs=%v)", cfg.Addr, cfg.Sandbox, cfg.AllowedPrograms)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("codellm: server error: %v", err)
	}
}
