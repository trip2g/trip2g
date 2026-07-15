// Command codellm is a standalone OpenAI-compatible chat-completions service
// that executes the fenced code in incoming markdown and returns the writes as
// OpenAI tool_calls. See docs/dev/codellm_extraction.md (Phase 1).
//
// SECURITY: an unauthenticated /v1/chat/completions is RCE-as-a-service. codellm
// is locked by two auth lanes (see internal/codellm): its own OpenAI-standard
// api_key (CODELLM_API_KEY, Authorization: Bearer — the same credential shape
// any OpenAI endpoint has) and the browser delegated-admin cookie gate. An unset
// key disables key auth (fail-safe), leaving the cookie gate as the only way in.
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
		log.Fatalf("config: %v", err)
	}

	// Delegated-admin gate for the browser-facing endpoints: each request's
	// session cookie is forwarded to the monolith's viewer{role}; admin → serve,
	// else 401, monolith-unreachable → fail-closed. BrowserAuth composes it with
	// the api_key TokenCheck seam below, so a caller can authenticate either way.
	admin, err := delegatedadmin.New(delegatedadmin.Config{MonolithBaseURL: cfg.Trip2gBaseURL})
	if err != nil {
		log.Fatalf("delegated admin: %v", err)
	}

	// wait air restarts
	// for i := 3; i >= 0; i-- {
	// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//
	// 	// check trip2g state, fail fast if it's down
	// 	role, fetchErr := admin.FetchViewerRole(ctx, "")
	// 	cancel()
	// 	if fetchErr != nil {
	// 		time.Sleep(time.Second)
	//
	// 		if i == 0 {
	// 			log.Fatalf("failed to FetchViewerRole: %v", fetchErr)
	// 		}
	// 	}
	//
	// 	if role != string(model.RoleGuest) {
	// 		log.Printf("unexpected role: %s", role)
	// 	}
	// }

	srvCfg := codellm.Config{
		AllowedPrograms: cfg.AllowedPrograms,
		Sandbox:         coderun.SandboxPolicy{Mode: cfg.Sandbox},
		MaxStdoutBytes:  cfg.MaxStdoutBytes,
		Timeout:         cfg.Timeout,
		// Secret VALUES live here (codellm's own env); this allowlist is the whole
		// decision of what to expose to the child. The request carries no env.
		ExposeEnv:       cfg.ExposeEnv,
		ExposeEnvPrefix: cfg.ExposeEnvPrefix,
		// codellm's own OpenAI-standard api_key (Authorization: Bearer <api_key>),
		// constant-time compared. An empty CODELLM_API_KEY disables key auth
		// (fail-safe), leaving the browser delegated-admin gate as the only way in.
		TokenCheck: codellm.APIKeyCheck(cfg.APIKey),
	}
	srvCfg.Auth = codellm.BrowserAuth(admin.Wrap, srvCfg.TokenCheck)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           codellm.New(srvCfg).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("codellm listening on %s (sandbox=%s, programs=%v)", cfg.Addr, cfg.Sandbox, cfg.AllowedPrograms)
	if err = srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
