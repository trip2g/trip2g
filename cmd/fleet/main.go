// Command fleet is the trip2g agent host (fleet-as-executor): it discovers role
// notes in trip2g, reconciles change-webhooks to point back at itself, receives
// deliveries, runs the scoped agent loop, and writes results back via a
// per-delivery scoped trip2g token. trip2g stays a dumb event source.
//
// Use --once <role-note.md> to run a single role-note offline without a trip2g
// connection (local KB from --vault, optional target note via --target).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	fleetappconfig "trip2g/cmd/fleet/appconfig"
	"trip2g/cmd/fleet/internal/agentruntime"
	"trip2g/cmd/fleet/internal/fleet"
	"trip2g/cmd/fleet/internal/fleet/fleetgql"
	"trip2g/cmd/fleet/internal/fleet/graph"
	"trip2g/cmd/fleet/internal/fleetmetrics"
	"trip2g/internal/appconfig"
	"trip2g/internal/delegatedadmin"
	"trip2g/internal/logger"
	"trip2g/internal/zerologger"

	"github.com/Khan/genqlient/graphql"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fleet:", err)
		os.Exit(1)
	}
}

// cliFlags holds the parsed command-line state for a single invocation.
type cliFlags struct {
	cfg    fleet.Config
	appCfg *fleetappconfig.Config // discovery scope, run/reconcile addrs, default model, offered tools, GraphQL addr

	dryRun     bool
	oncePath   string // non-empty → one-shot mode; daemon must NOT start
	vaultDir   string // KB root for --once (default ".")
	targetPath string // optional note in the vault to use as change_file context
	graphAddr  string // non-empty → serve the dependency-graph debug UI/JSON; loopback-only
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cli, err := parseFlags(ctx)
	if err != nil {
		return err
	}

	// --once: run a single role-note offline, then exit. No daemon, no trip2g.
	if cli.oncePath != "" {
		return runOnce(ctx, cli)
	}

	// Normalize: strip trailing slash so webhook URLs assemble cleanly.
	cli.cfg.CallbackURL = normalizeCallbackURL(cli.cfg.CallbackURL)

	err = validateConfig(cli.cfg)
	if err != nil {
		return err
	}
	if err = cli.appCfg.Validate(); err != nil {
		return fmt.Errorf("fleet: invalid config: %w", err)
	}

	lg := zerologger.New(cli.cfg.LogLevel, false)

	// Metrics live on their own loopback listener (mirrors the monolith's
	// internal listener). It starts before the first sync so a scrape can see the
	// fleet warming up.
	var firstSyncDone atomic.Bool
	metrics := startMetricsServer(lg, cli, firstSyncDone.Load)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	adminGQL := fleet.NewAdminGraphQLClient(cli.cfg.Trip2gBaseURL, cli.cfg.Trip2gAdminPersonalToken, httpClient)
	llm := agentruntime.NewOpenAILLM(cli.cfg.LLMAPIKey, cli.cfg.LLMBaseURL)
	llm.SetMetrics(metrics, fleetmetrics.LaneLLM)

	exec := execLLM(cli.cfg)
	if exec != nil {
		exec.SetMetrics(metrics, fleetmetrics.LaneExec)
	}

	f := fleet.NewFleet(cli.cfg, httpClient, llm, execAsLLM(exec))
	f.SetMetrics(metrics)
	discovery := fleet.NewDiscovery(adminGQL, cli.cfg.FleetID, cli.cfg.AgentsFolder, cli.cfg.OfferedTools)

	// --dry-run: connect, print + flag each role's resolved config, then exit
	// WITHOUT registering/reconciling any webhooks (eyeball roles before go-live).
	if cli.dryRun {
		runDryRun(ctx, lg, discovery, cli.cfg, os.Stdout)
		return nil
	}

	reconciler := fleet.NewReconciler(adminGQL, cli.cfg)
	reconciler.SetMetrics(metrics)

	// --graph-addr: localhost-only debug surface serving the dependency graph.
	graphSrv, err := startGraphDebugServer(lg, discovery, adminGQL, cli)
	if err != nil {
		return err
	}
	if graphSrv != nil {
		defer func() { _ = graphSrv.Close() }()
	}

	// --graphql-addr: the fleet's own GraphQL read API (roles + roleGraph).
	// Reuses fleet.ParseRole via the same Discovery. The ENTIRE browser-facing
	// mux on this port is gated by the delegated-admin middleware (forwards the
	// caller's cookie to the monolith's viewer{role}; admin -> serve, else 401,
	// monolith-unreachable -> fail-closed). This is a separate server from the
	// webhook delivery listener (cli.cfg.ListenAddr, /_fleet/<h>/webhook/,
	// HMAC-authed), so the cookie gate never touches the monolith->fleet path.
	graphqlSrv, err := startFleetGraphQLServer(lg, discovery, cli)
	if err != nil {
		return err
	}
	if graphqlSrv != nil {
		defer func() { _ = graphqlSrv.Close() }()
	}

	// First sync before serving so the registry is populated.
	syncOnce(ctx, lg, f, discovery, reconciler, metrics)
	firstSyncDone.Store(true)

	mux := http.NewServeMux()
	mux.HandleFunc(f.WebhookPath(), f.ServeDelivery)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: cli.cfg.ListenAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	srvErrCh := make(chan error, 1)
	go func() {
		if cli.cfg.AllowRoleAuthoring {
			lg.Warn("role-authoring guard is OFF (--allow-role-authoring): agents may create and " +
				"edit role notes, and a role declares its own write_patterns")
		}
		lg.Info("fleet listening", "fleet_id", cli.cfg.FleetID, "addr", cli.cfg.ListenAddr, "callback", cli.cfg.CallbackURL)
		listenErr := srv.ListenAndServe()
		if listenErr != nil && listenErr != http.ErrServerClosed {
			srvErrCh <- listenErr
			stop()
		}
	}()

	ticker := time.NewTicker(cli.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case srvListenErr := <-srvErrCh:
			return fmt.Errorf("http server: %w", srvListenErr)
		case <-ctx.Done():
			shutdownErr := gracefulShutdown(lg, srv, reconciler.Deregister, cli.cfg.KeepWebhooksOnShutdown, cli.cfg.ShutdownGrace)
			select {
			case srvListenErr := <-srvErrCh:
				return fmt.Errorf("http server: %w", srvListenErr)
			default:
				if shutdownErr != nil {
					return fmt.Errorf("graceful shutdown: %w", shutdownErr)
				}
				return nil
			}
		case <-ticker.C:
			syncOnce(ctx, lg, f, discovery, reconciler, metrics)
		}
	}
}

// startGraphDebugServer starts the localhost-only dependency-graph debug UI
// (GET / = UI, GET /graph.json = machine JSON) when --graph-addr is set. It is
// an internal introspection tool, so a non-loopback bind is refused. Returns
// nil when disabled; the caller owns Close via defer.
func startGraphDebugServer(
	lg logger.Logger,
	discovery *fleet.Discovery,
	adminGQL graphql.Client,
	cli cliFlags,
) (*http.Server, error) {
	if cli.graphAddr == "" {
		return nil, nil
	}
	if err := validateLoopbackAddr(cli.graphAddr); err != nil {
		return nil, fmt.Errorf("--graph-addr: %w", err)
	}
	gs := graph.NewServer(discovery, adminGQL, cli.cfg)
	srv := &http.Server{Addr: cli.graphAddr, Handler: gs.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		lg.Info("fleet graph debug UI listening", "addr", cli.graphAddr)
		if gerr := srv.ListenAndServe(); gerr != nil && gerr != http.ErrServerClosed {
			lg.Error("graph debug server failed", "err", gerr)
		}
	}()
	return srv, nil
}

// startFleetGraphQLServer starts the fleet's own GraphQL read API (roles +
// roleGraph) when --graphql-addr is set. The entire mux is gated by the
// delegated-admin middleware. Returns nil when disabled; the caller owns Close.
func startFleetGraphQLServer(
	lg logger.Logger,
	discovery *fleet.Discovery,
	cli cliFlags,
) (*http.Server, error) {
	if cli.appCfg.GraphQLAddr == "" {
		return nil, nil
	}
	warnIfGraphQLAddrNonLoopback(lg, cli.appCfg.GraphQLAddr)
	gqlMux, err := newFleetGraphQLHandler(discovery, cli.cfg.Trip2gBaseURL)
	if err != nil {
		return nil, fmt.Errorf("fleet: graphql auth: %w", err)
	}
	srv := &http.Server{Addr: cli.appCfg.GraphQLAddr, Handler: gqlMux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		lg.Info("fleet GraphQL API listening", "addr", cli.appCfg.GraphQLAddr)
		if gerr := srv.ListenAndServe(); gerr != nil && gerr != http.ErrServerClosed {
			lg.Error("graphql server failed", "err", gerr)
		}
	}()
	return srv, nil
}

// execLLM builds the exec-tool client (codellm) when --exec-base-url is set.
// nil disables the exec tool; program allowlisting and sandboxing are the
// codellm operator's concern — fleet executes no code in-process.
func execLLM(cfg fleet.Config) *agentruntime.OpenAILLM {
	if cfg.ExecBaseURL == "" {
		return nil
	}
	return agentruntime.NewOpenAILLM(cfg.ExecAPIKey, cfg.ExecBaseURL)
}

// execAsLLM converts the (possibly nil) exec client to the LLM interface. A
// typed nil in an interface is NOT nil, and the exec tool is enabled by a
// non-nil LLM, so the conversion has to be explicit.
func execAsLLM(exec *agentruntime.OpenAILLM) agentruntime.LLM {
	if exec == nil {
		return nil
	}
	return exec
}

// newFleetGraphQLHandler builds the fleet GraphQL server's HTTP handler: a mux
// serving POST /graphql (roles + roleGraph), with the ENTIRE mux wrapped by the
// delegated-admin middleware so every browser-facing path on this port — not
// just /graphql — is gated on the caller being a verified trip2g admin.
// monolithBaseURL is where viewer{role} is asked (fleet's Trip2gBaseURL). The
// webhook delivery path lives on a different server and is intentionally NOT
// wrapped here (it authenticates via HMAC, not the admin cookie).
func newFleetGraphQLHandler(roles fleetgql.RoleSource, monolithBaseURL string) (http.Handler, error) {
	da, err := delegatedadmin.New(delegatedadmin.Config{MonolithBaseURL: monolithBaseURL})
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/graphql", fleetgql.NewHTTPHandler(roles, nil))
	return da.Wrap(mux), nil
}

// runOnce reads cli.oncePath as a role-note, parses it, validates against the
// fleet's offered tools, renders the instruction, and runs agentruntime.Run
// against a local file KB. Prints the Result as indented JSON. No trip2g
// connection is required; LLM credentials must still be set.
func runOnce(ctx context.Context, cli cliFlags) error {
	raw, err := os.ReadFile(cli.oncePath)
	if err != nil {
		return fmt.Errorf("once: read role file: %w", err)
	}

	meta, body, err := parseFrontmatter(string(raw))
	if err != nil {
		return fmt.Errorf("once: parse frontmatter: %w", err)
	}

	role, err := fleet.ParseRole(cli.oncePath, body, meta)
	if err != nil {
		return fmt.Errorf("once: parse role: %w", err)
	}

	err = role.Validate(cli.cfg.OfferedTools)
	if err != nil {
		return fmt.Errorf("once: validate role: %w", err)
	}

	// Surface the deny-all trap loudly: write tools declared but no write_patterns.
	// Without this the model paraphrases the per-tool "access denied" as its own
	// refusal, hiding the config gap from the operator.
	role.WarnIfWriteScopeMisconfigured(zerologger.New(cli.cfg.LogLevel, false))

	// Load target note content if --target was given.
	var targetContent string
	if cli.targetPath != "" {
		data, readErr := os.ReadFile(cli.targetPath)
		if readErr != nil {
			return fmt.Errorf("once: read target file: %w", readErr)
		}
		targetContent = string(data)
	}

	instruction, err := fleet.RenderRoleInstruction(role, cli.targetPath, targetContent)
	if err != nil {
		return fmt.Errorf("once: render instruction: %w", err)
	}

	model := role.Model
	if model == "" {
		model = cli.cfg.DefaultModel
	}
	maxTokens := role.MaxTokens
	if maxTokens <= 0 || (cli.cfg.TokenCeiling > 0 && maxTokens > cli.cfg.TokenCeiling) {
		maxTokens = cli.cfg.TokenCeiling
	}
	if maxTokens <= 0 {
		maxTokens = 100000
	}
	maxSteps := role.MaxSteps
	if maxSteps <= 0 || (cli.cfg.StepCeiling > 0 && maxSteps > cli.cfg.StepCeiling) {
		maxSteps = cli.cfg.StepCeiling
	}
	if maxSteps <= 0 {
		maxSteps = 25
	}

	llm := agentruntime.NewOpenAILLM(cli.cfg.LLMAPIKey, cli.cfg.LLMBaseURL)
	kb := agentruntime.NewFileKB(cli.vaultDir)

	runCtx, cancel := context.WithTimeout(ctx,
		time.Duration(role.EffectiveTimeoutSeconds())*time.Second)
	defer cancel()

	result, runErr := agentruntime.Run(runCtx, agentruntime.Input{
		Instruction:   instruction,
		ReadPatterns:  role.ReadPatterns,
		WritePatterns: role.WritePatterns,
		Tools:         role.Tools,
		Model:         model,
		MaxTokens:     maxTokens,
		MaxSteps:      maxSteps,
		ExecLLM:       execLLM(cli.cfg),
		LLM:           llm,
		KB:            kb,
	})
	if runErr != nil {
		return fmt.Errorf("once: run: %w", runErr)
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(out))
	return nil
}

// parseFrontmatter splits a markdown file into a raw-value meta map and body.
// The frontmatter block is the leading ---\n...\n---\n section (YAML, flat
// key: value lines). Values are kept as raw strings so fleet.ParseRole's
// parseList handles array forms (JSON ["a"] or YAML-flow [a, b]) correctly.
// Files without a leading --- block return an empty meta map and the full
// content as body.
func parseFrontmatter(content string) (map[string]string, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return map[string]string{}, content, nil
	}
	rest := content[4:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil, "", errors.New("unclosed frontmatter block (missing closing ---)")
	}
	block := rest[:idx]
	after := rest[idx+4:]
	if strings.HasPrefix(after, "\r\n") {
		after = after[2:]
	} else if strings.HasPrefix(after, "\n") {
		after = after[1:]
	}

	meta := make(map[string]string)
	for _, line := range strings.Split(block, "\n") {
		key, val, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		k := strings.TrimSpace(key)
		if k == "" || strings.HasPrefix(k, "#") {
			continue
		}
		meta[k] = strings.TrimSpace(val)
	}
	return meta, after, nil
}

// validateLoopbackAddr rejects any graph debug bind that is not loopback: the
// graph surface has no auth and must never be reachable off-box.
func validateLoopbackAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("must be host:port: %w", err)
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("host %q is not loopback; bind 127.0.0.1, ::1 or localhost", host)
	}
	return nil
}

// warnIfGraphQLAddrNonLoopback logs a loud warning when --graphql-addr is
// bound off loopback. Unlike --graph-addr, a non-loopback
// GraphQL bind is not hard-blocked — a remote-fleet-behind-its-own-Caddy setup
// is a legitimate topology, and the delegated-admin gate still authenticates
// every request — but the operator should see the exposure explicitly.
func warnIfGraphQLAddrNonLoopback(lg logger.Logger, addr string) {
	if isLoopbackAddr(addr) {
		return
	}
	lg.Warn("fleet GraphQL bound non-loopback; ensure Caddy is the sole ingress + delegated-admin gate", "addr", addr)
}

// normalizeCallbackURL strips trailing slashes so webhook URLs assemble cleanly.
func normalizeCallbackURL(u string) string {
	return strings.TrimRight(u, "/")
}

// validateConfig returns an error if any required field is missing or invalid.
func validateConfig(cfg fleet.Config) error {
	missing := []string{}
	if cfg.FleetID == "" {
		missing = append(missing, "FleetID (--fleet-id / TRIP2G_FLEET_FLEET_ID)")
	}
	if cfg.CallbackURL == "" {
		missing = append(missing, "CallbackURL (--callback-url / TRIP2G_FLEET_CALLBACK_URL)")
	}
	if cfg.Trip2gAdminPersonalToken == "" {
		missing = append(missing, "Trip2gAdminPersonalToken (--trip2g-admin-personal-token / TRIP2G_FLEET_TRIP2G_ADMIN_PERSONAL_TOKEN)")
	}
	if cfg.FleetSecret == "" {
		missing = append(missing, "FleetSecret (--fleet-secret / TRIP2G_FLEET_FLEET_SECRET)")
	}
	if cfg.LLMAPIKey == "" {
		missing = append(missing, "LLMAPIKey (--llm-api-key / TRIP2G_FLEET_LLM_API_KEY)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("fleet: missing required config: %s", strings.Join(missing, ", "))
	}
	if cfg.TokenCeiling <= 0 {
		return fmt.Errorf("fleet: TokenCeiling must be > 0 (got %d); use --token-ceiling", cfg.TokenCeiling)
	}
	if cfg.StepCeiling <= 0 {
		return fmt.Errorf("fleet: StepCeiling must be > 0 (got %d); use --step-ceiling", cfg.StepCeiling)
	}
	if len(cfg.OfferedTools) == 0 {
		return errors.New("fleet: OfferedTools must be non-empty; use --offered-tools")
	}
	return nil
}

func syncOnce(ctx context.Context, lg logger.Logger, f *fleet.Fleet, d *fleet.Discovery, r *fleet.Reconciler, m *fleetmetrics.Metrics) {
	started := time.Now()
	roles, errs := d.DiscoverRoles(ctx)
	for _, e := range errs {
		lg.Warn("discover: skipped role", "err", e)
	}
	m.AddRolesSkipped(len(errs))
	for _, role := range roles {
		role.WarnIfWriteScopeMisconfigured(lg)
	}
	f.SetRoles(roles)

	reconcileErr := r.Reconcile(ctx, roles)
	if reconcileErr != nil {
		lg.Error("reconcile failed", "err", reconcileErr)
	}
	m.RecordSync(syncStatus(reconcileErr, len(errs)), time.Since(started).Seconds())
}

// syncStatus grades one poll cycle. A reconcile failure outranks everything: the
// cycle did not land. Otherwise dropped role notes make it partial — the
// registry WAS refreshed, so freshness still advances, and the dropped notes are
// counted separately by fleet_roles_skipped_total.
func syncStatus(reconcileErr error, skipped int) string {
	switch {
	case reconcileErr != nil:
		return fleetmetrics.StatusError
	case skipped > 0:
		return fleetmetrics.StatusPartial
	default:
		return fleetmetrics.StatusOK
	}
}

// gracefulShutdown drains in-flight deliveries (srv.Shutdown waits for active
// handlers — runs are synchronous, so this drains runs too) within the grace
// window, then deregisters owned webhooks unless keepWebhooks is set (rolling
// deploys keep them so trip2g retries against the next instance). deregister may
// be nil (no-op). Returns the drain error, if any.
//
// Drain runs BEFORE deregister so in-flight writes always finish; the tradeoff
// is that on a clean (non-keep) shutdown trip2g may see connection-refused for
// up to grace until deregister runs — those deliveries are retried, and not
// losing in-flight writes is the priority.
func gracefulShutdown(lg logger.Logger, srv *http.Server, deregister func(context.Context) error, keepWebhooks bool, grace time.Duration) error {
	lg.Info("shutdown: draining in-flight runs", "grace", grace.String())
	drainCtx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	shutdownErr := srv.Shutdown(drainCtx)
	if shutdownErr != nil {
		lg.Error("shutdown: drain incomplete", "err", shutdownErr)
	}

	if keepWebhooks {
		lg.Info("shutdown: keeping webhooks (keep-webhooks-on-shutdown)")
		return shutdownErr
	}
	if deregister != nil {
		lg.Info("shutdown: deregistering webhooks")
		deregCtx, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel2()
		if deregErr := deregister(deregCtx); deregErr != nil {
			lg.Error("shutdown: deregister failed", "err", deregErr)
		}
	}
	return shutdownErr
}

// effectiveGrace converts the --shutdown-grace-seconds flag to a duration,
// flooring non-positive values to 30s so the drain path is never skipped (a
// zero/expired grace would make srv.Shutdown force-close in-flight runs).
func effectiveGrace(seconds int) time.Duration {
	d := time.Duration(seconds) * time.Second
	if d <= 0 {
		return 30 * time.Second
	}
	return d
}

// parseFlags registers all fleet flags on a dedicated FlagSet wired to
// appconfig.EnvFlag so every flag gets a TRIP2G_FLEET_<FLAG_NAME> environment
// variable fallback (standard kebab-to-SCREAMING_SNAKE mapping). The
// TRIP2G_FLEET_ prefix distinguishes fleet config from the trip2g app's
// TRIP2G_ namespace when both run in the same environment.
func parseFlags(ctx context.Context) (cliFlags, error) {
	var cli cliFlags
	var poll int
	var graceSeconds int

	fs := flag.NewFlagSet("fleet", flag.ContinueOnError)

	// Discovery scope, run/reconcile addrs, default model, offered tools, and
	// the GraphQL listen addr: defined by cmd/fleet/appconfig on this
	// same shared FlagSet, so they parse in the one Parse call below alongside
	// the rest of fleet's flags.
	cli.appCfg = fleetappconfig.DefaultConfig()
	cli.appCfg.DefineFlags(fs)

	// Daemon flags.
	fs.BoolVar(&cli.dryRun, "dry-run", false,
		"discover roles, print + flag their resolved config, then exit without registering webhooks")
	fs.StringVar(&cli.cfg.FleetID, "fleet-id", "",
		"REQUIRED fleet identity: partition key for role fleet_id + the /_fleet/<sha256(\"fleet:\"+id)>/webhook delivery path")
	fs.StringVar(&cli.cfg.Trip2gAdminPersonalToken, "trip2g-admin-personal-token", "",
		"trip2g personal token (t2g_*) of an admin user; trip2g seeds it from OWNER_PERSONAL_TOKEN_VALUE (required for daemon mode)")
	fs.StringVar(&cli.cfg.FleetSecret, "fleet-secret", "",
		"HMAC seed for per-role secrets (required for daemon mode)")
	fs.StringVar(&cli.cfg.LLMBaseURL, "llm-base-url", "",
		"OpenAI-compatible base URL")
	fs.StringVar(&cli.cfg.LLMAPIKey, "llm-api-key", "",
		"LLM API key (falls back to OPENAI_API_KEY)")
	fs.StringVar(&cli.cfg.ExecBaseURL, "exec-base-url", "",
		"OpenAI-compatible base URL the exec tool routes code to (codellm); "+
			"empty = exec tool disabled. Program allowlisting and sandboxing are codellm's concern")
	fs.StringVar(&cli.cfg.ExecAPIKey, "exec-api-key", "",
		"API key for --exec-base-url")
	fs.IntVar(&cli.cfg.TokenCeiling, "token-ceiling", 100000,
		"non-overridable per-run token cap")
	fs.IntVar(&cli.cfg.StepCeiling, "step-ceiling", 25,
		"non-overridable per-run step cap")
	fs.IntVar(&poll, "poll-seconds", 30,
		"discovery/reconcile poll interval seconds")
	fs.IntVar(&graceSeconds, "shutdown-grace-seconds", 30,
		"max seconds to drain in-flight runs on shutdown before forcing close")
	fs.BoolVar(&cli.cfg.KeepWebhooksOnShutdown, "keep-webhooks-on-shutdown", false,
		"do not deregister webhooks on shutdown (rolling deploys: trip2g keeps them and retries)")
	fs.BoolVar(&cli.cfg.AllowRoleAuthoring, "allow-role-authoring", false,
		"let agents create and edit role notes (fleet_id in frontmatter). Off by default: a role "+
			"declares its own write_patterns, so authoring one escalates scope")
	fs.StringVar(&cli.cfg.LogLevel, "log-level", "info",
		"log level: debug|info|warn|error")
	fs.StringVar(&cli.graphAddr, "graph-addr", "",
		"loopback-only debug listen address serving the fleet dependency graph "+
			"(GET / = UI, GET /graph.json = JSON), e.g. 127.0.0.1:9092; empty = disabled")

	// One-shot offline harness flags.
	fs.StringVar(&cli.oncePath, "once", "",
		"path to a role-note .md file; runs it once without a trip2g connection and exits")
	fs.StringVar(&cli.vaultDir, "vault", ".",
		"vault directory for the local file KB (used with --once)")
	fs.StringVar(&cli.targetPath, "target", "",
		"note path in the vault to populate change_file context (used with --once)")

	ef := appconfig.New(appconfig.EnvFlagConfig{
		FlagSet:           fs,
		EnvPrefix:         "TRIP2G_FLEET_",
		ShowEnvKeyInUsage: true,
	})

	if err := ef.Parse(ctx, os.Args[1:]); err != nil {
		return cliFlags{}, err
	}

	// OPENAI_API_KEY fallback for --once offline convenience.
	if cli.cfg.LLMAPIKey == "" {
		cli.cfg.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
	}

	cli.appCfg.Prepare()
	cli.cfg.ListenAddr = cli.appCfg.ListenAddr
	cli.cfg.CallbackURL = cli.appCfg.CallbackURL
	cli.cfg.Trip2gBaseURL = cli.appCfg.Trip2gBaseURL
	cli.cfg.DefaultModel = cli.appCfg.DefaultModel
	cli.cfg.AgentsFolder = cli.appCfg.AgentsFolder
	cli.cfg.OfferedTools = cli.appCfg.OfferedTools
	cli.cfg.PollInterval = time.Duration(poll) * time.Second
	cli.cfg.ShutdownGrace = effectiveGrace(graceSeconds)
	return cli, nil
}

// runDryRun discovers (parses) every role over the admin lane, prints each
// role's resolved config with a flag on any that fail Validate, and returns
// without touching webhooks. It is the fleet's pre-flight doctor.
func runDryRun(ctx context.Context, lg logger.Logger, d *fleet.Discovery, cfg fleet.Config, out io.Writer) {
	roles, errs := d.DiscoverParsed(ctx)
	for _, e := range errs {
		lg.Warn("dry-run: parse error", "err", e)
	}
	fmt.Fprint(out, reportRoles(roles, cfg.OfferedTools, cfg.DefaultModel))
}

// reportRoles renders a human-readable resolved-config report for each role,
// flagging any that fail Validate. offered is the fleet's offered tool set used
// for the Validate check; defaultModel is shown when a role omits model.
func reportRoles(roles []fleet.Role, offered []string, defaultModel string) string {
	if len(roles) == 0 {
		return "no roles discovered\n"
	}
	var b strings.Builder
	for _, r := range roles {
		model := r.Model
		if model == "" {
			model = defaultModel + " (default)"
		}
		forEach := r.ForEach
		if forEach == "" {
			forEach = "(single run)"
		}
		timeoutNote := ""
		if r.TimeoutSeconds <= 0 {
			timeoutNote = " (default)"
		}

		fmt.Fprintf(&b, "%s\n", r.NotePath)
		fmt.Fprintf(&b, "  mode:            %s\n", r.Mode)
		fmt.Fprintf(&b, "  trigger_on:      %v -> onCreate=%t onUpdate=%t onRemove=%t\n",
			r.TriggerOn,
			slices.Contains(r.TriggerOn, "create"),
			slices.Contains(r.TriggerOn, "update"),
			slices.Contains(r.TriggerOn, "remove"))
		fmt.Fprintf(&b, "  trigger_include: %v\n", r.TriggerInclude)
		fmt.Fprintf(&b, "  trigger_exclude: %v\n", r.TriggerExclude)
		fmt.Fprintf(&b, "  read_patterns:   %v\n", r.ReadPatterns)
		fmt.Fprintf(&b, "  write_patterns:  %v\n", r.WritePatterns)
		fmt.Fprintf(&b, "  model:           %s\n", model)
		fmt.Fprintf(&b, "  tools:           %v\n", r.Tools)
		fmt.Fprintf(&b, "  for_each:        %s\n", forEach)
		fmt.Fprintf(&b, "  timeout_seconds: %d%s\n", r.EffectiveTimeoutSeconds(), timeoutNote)
		if verr := r.Validate(offered); verr != nil {
			fmt.Fprintf(&b, "  STATUS: FLAGGED: %v\n", verr)
		} else {
			fmt.Fprint(&b, "  STATUS: OK\n")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// startMetricsServer brings up the internal listener when --metrics-addr is set
// and returns the sink every component records into (nil when disabled, which
// all the call sites tolerate). A bind failure is logged, not fatal: losing
// observability must not take the fleet down. ready reports whether the first
// discovery sync has been ATTEMPTED, not whether it succeeded, so a fleet whose
// first poll found trip2g down still answers deliveries for whatever it knows
// rather than parking itself. Staleness is reported by
// fleet_last_successful_sync_timestamp_seconds instead.
func startMetricsServer(lg logger.Logger, cli cliFlags, ready func() bool) *fleetmetrics.Metrics {
	if cli.appCfg.MetricsAddr == "" {
		return nil
	}
	warnIfMetricsAddrNonLoopback(lg, cli.appCfg.MetricsAddr)
	m := fleetmetrics.New()
	m.SetConfigInfo(cli.cfg.FleetID, cli.cfg.DefaultModel, strconv.FormatBool(cli.cfg.ExecBaseURL != ""))

	srv := &http.Server{Addr: cli.appCfg.MetricsAddr, Handler: m.Handler(ready), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		lg.Info("fleet metrics listening", "addr", cli.appCfg.MetricsAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			lg.Error("metrics server failed", "err", err)
		}
	}()
	return m
}

// warnIfMetricsAddrNonLoopback logs a loud warning when the internal listener
// is bound off loopback. It is not blocked — scraping a containerized fleet
// requires binding the container's interface — but /metrics and pprof are
// unauthenticated, so the exposure must be visible in the log.
func warnIfMetricsAddrNonLoopback(lg logger.Logger, addr string) {
	if isLoopbackAddr(addr) {
		return
	}
	lg.Warn("fleet metrics bound non-loopback: /metrics and pprof are unauthenticated", "addr", addr)
}

// isLoopbackAddr reports whether a host:port binds only the loopback interface.
func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
