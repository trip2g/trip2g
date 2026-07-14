// Command codellm is a standalone OpenAI-compatible chat-completions service
// that executes the fenced code in incoming markdown and returns the writes as
// OpenAI tool_calls. See docs/dev/codellm_extraction.md (Phase 1).
//
// SECURITY: an unauthenticated /v1/chat/completions is RCE-as-a-service. The
// fleet↔codellm channel is locked by two auth lanes (see internal/codellm):
// the browser delegated-admin cookie gate and the server-to-server channel-token
// lane (CODELLM_CHANNEL_TOKEN, Authorization: Bearer). An unset token disables
// the fleet lane (fail-safe), leaving the cookie gate as the only way in.
package main

import (
	"log"
	"net/http"
	"time"

	"trip2g/internal/codellm"
	"trip2g/internal/coderun"
	"trip2g/internal/delegatedadmin"

	"trip2g/cmd/codellm/appconfig"
)

func main() {
	// The per-block sandbox works by re-execing THIS binary as a confined child
	// (marker env var). It must run first so a re-exec lands in the child branch
	// and never falls through to start a second server. No-op when the marker is
	// absent. Phase 2 completes the sandbox story (move + tests).
	coderun.MaybeRunSandboxChild()

	cfg, err := appconfig.Get()
	if err != nil {
		log.Fatalf("codellm: config: %v", err)
	}

	// Delegated-admin gate for the browser-facing endpoints: each request's
	// session cookie is forwarded to the monolith's viewer{role}; admin → serve,
	// else 401, monolith-unreachable → fail-closed. BrowserAuth composes it with
	// the fleet-lane TokenCheck seam (nil here — mTLS/shared-token is a deploy
	// concern), so the fleet server-to-server /v1 lane stays a separate regime.
	admin, err := delegatedadmin.New(delegatedadmin.Config{MonolithBaseURL: cfg.Trip2gBaseURL})
	if err != nil {
		log.Fatalf("codellm: delegated admin: %v", err)
	}

	srvCfg := codellm.Config{
		AllowedPrograms: cfg.AllowedPrograms,
		Sandbox:         coderun.SandboxPolicy{Mode: cfg.Sandbox},
		MaxStdoutBytes:  cfg.MaxStdoutBytes,
		Timeout:         cfg.Timeout,
		// Fleet lane: authenticate a server-to-server caller by the shared channel
		// token (Authorization: Bearer <token>), constant-time compared. An empty
		// CODELLM_CHANNEL_TOKEN disables the lane (fail-safe), leaving the browser
		// delegated-admin gate as the only way in.
		TokenCheck: codellm.ChannelTokenCheck(cfg.ChannelToken),
	}
	srvCfg.Auth = codellm.BrowserAuth(admin.Wrap, srvCfg.TokenCheck)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           codellm.New(srvCfg).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("codellm listening on %s (sandbox=%s, programs=%v)", cfg.Addr, cfg.Sandbox, cfg.AllowedPrograms)
	if err = srv.ListenAndServe(); err != nil {
		log.Fatalf("codellm: server error: %v", err)
	}
}
