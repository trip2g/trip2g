// Command codellm-standalone runs the REAL internal/codellm service in its
// documented standalone (no-auth) mode — the same mode its own tests use
// (Config.Auth nil -> no-op passthrough, see internal/codellm/server.go).
//
// Why not cmd/codellm directly? That binary hard-wires the delegated-admin
// browser gate (session cookie -> monolith viewer{role}), and its fleet-lane
// TokenCheck seam is explicitly NOT built yet. So /v1/chat/completions returns
// 401 for any server-to-server caller unless the trip2g monolith is up AND an
// admin cookie is forwarded. This driver instantiates the identical service
// package (same handleChatCompletions / ExecCode / buildResponse path), only
// skipping the auth wrapper, so the demo can exercise code execution end-to-end
// without standing up the monolith. It does NOT modify codellm's Go code.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"trip2g/internal/agentruntime"
	"trip2g/internal/codellm"
)

func main() {
	// The per-block sandbox re-execs THIS binary as a confined child; run the
	// marker check first (no-op when the marker env var is absent), exactly as
	// cmd/codellm/main.go does.
	agentruntime.MaybeRunSandboxChild()

	addr := envOr("CODELLM_ADDR", "127.0.0.1:8092")
	programs := strings.Split(envOr("CODELLM_ALLOWED_PROGRAMS", "python"), ",")
	sandbox := agentruntime.SandboxMode(envOr("CODELLM_SANDBOX", "off"))

	srv := &http.Server{
		Addr: addr,
		Handler: codellm.New(codellm.Config{
			AllowedPrograms: programs,
			Sandbox:         agentruntime.SandboxPolicy{Mode: sandbox},
			Timeout:         300 * time.Second,
			// Auth left nil -> no-op passthrough (standalone mode).
		}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("codellm-standalone listening on %s (sandbox=%s, programs=%v)", addr, sandbox, programs)
	log.Fatal(srv.ListenAndServe())
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
