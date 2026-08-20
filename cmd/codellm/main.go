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
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trip2g/cmd/codellm/internal/codellm"
	"trip2g/cmd/codellm/internal/codellmmetrics"
	"trip2g/cmd/codellm/internal/coderun"
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

	// Metrics live on their own loopback listener (mirrors the monolith's
	// internal listener): /metrics, pprof and the probes are unauthenticated, so
	// the port must never leave the box.
	metrics := startMetricsServer(cfg)

	srvCfg := codellm.Config{
		AllowedPrograms: cfg.AllowedPrograms,
		Metrics:         metrics,
		Sandbox: coderun.SandboxPolicy{
			Mode:    cfg.Sandbox,
			Network: cfg.SandboxNetwork,
		},
		MaxStdoutBytes: cfg.MaxStdoutBytes,
		Timeout:        cfg.Timeout,
		// Secret VALUES live here (codellm's own env); this allowlist is the whole
		// decision of what to expose to the child. The request carries no env.
		ExposeEnv:       cfg.ExposeEnv,
		ExposeEnvPrefix: cfg.ExposeEnvPrefix,
		// codellm's own OpenAI-standard api_key (Authorization: Bearer <api_key>),
		// constant-time compared. An empty CODELLM_API_KEY disables key auth
		// (fail-safe), leaving the browser delegated-admin gate as the only way in.
		TokenCheck: codellm.APIKeyCheck(cfg.APIKey),
	}
	srvCfg.Auth = codellm.BrowserAuthWithMetrics(admin.Wrap, srvCfg.TokenCheck, metrics)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           codellm.New(srvCfg).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("codellm listening on %s (sandbox=%s, network=%t, programs=%v)", cfg.Addr, cfg.Sandbox, cfg.SandboxNetwork, cfg.AllowedPrograms)
	if err = srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// startMetricsServer brings up the internal listener when --metrics-addr is
// set and returns the sink the service records into (nil when disabled, which
// every record call site tolerates). A failure to bind it is logged, not fatal:
// losing observability must not take the service down.
func startMetricsServer(cfg *appconfig.Config) *codellmmetrics.Metrics {
	if cfg.MetricsAddr == "" {
		return nil
	}
	warnIfMetricsAddrNonLoopback(cfg.MetricsAddr)
	m := codellmmetrics.New()
	m.SetConfigInfo(string(cfg.Sandbox), strconv.FormatBool(cfg.SandboxNetwork), strings.Join(cfg.AllowedPrograms, ","))

	srv := &http.Server{
		Addr: cfg.MetricsAddr,
		// codellm is ready as soon as it listens: it holds no warm-up state.
		Handler:           m.Handler(nil),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("codellm metrics listening on %s", cfg.MetricsAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("codellm metrics server error: %v", err)
		}
	}()
	return m
}

// warnIfMetricsAddrNonLoopback logs a loud warning when the internal listener
// is bound off loopback. It is not blocked — scraping a containerized codellm
// requires binding the container's interface — but /metrics and pprof are
// unauthenticated, so the exposure must be visible in the log.
func warnIfMetricsAddrNonLoopback(addr string) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	if host == "localhost" {
		return
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return
	}
	log.Printf("WARNING: codellm metrics bound non-loopback (%s): /metrics and pprof are unauthenticated", addr)
}
