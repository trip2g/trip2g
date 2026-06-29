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
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"trip2g/internal/agentruntime"
	"trip2g/internal/appconfig"
	"trip2g/internal/fleet"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fleet: %v", err)
	}
}

// cliFlags holds the parsed command-line state for a single invocation.
type cliFlags struct {
	cfg        fleet.Config
	dryRun     bool
	oncePath   string // non-empty → one-shot mode; daemon must NOT start
	vaultDir   string // KB root for --once (default ".")
	targetPath string // optional note in the vault to use as change_file context
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

	httpClient := &http.Client{Timeout: 30 * time.Second}
	client := fleet.NewHTTPClient(cli.cfg.Trip2gBaseURL, httpClient)
	adminGQL := fleet.NewAdminGraphQLClient(cli.cfg.Trip2gBaseURL, cli.cfg.JWTSecret, cli.cfg.AdminEmail, httpClient)
	llm := agentruntime.NewOpenAILLM(cli.cfg.LLMAPIKey, cli.cfg.LLMBaseURL)

	f := fleet.NewFleet(cli.cfg, client, llm)
	discovery := fleet.NewDiscovery(adminGQL, cli.cfg.AgentsFolder, cli.cfg.OfferedTools)

	// --dry-run: connect, print + flag each role's resolved config, then exit
	// WITHOUT registering/reconciling any webhooks (eyeball roles before go-live).
	if cli.dryRun {
		runDryRun(ctx, discovery, cli.cfg)
		return nil
	}

	reconciler := fleet.NewReconciler(adminGQL, cli.cfg)

	// First sync before serving so the registry is populated.
	syncOnce(ctx, f, discovery, reconciler)

	mux := http.NewServeMux()
	mux.HandleFunc("/deliver/", f.ServeDelivery)
	srv := &http.Server{Addr: cli.cfg.ListenAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	srvErrCh := make(chan error, 1)
	go func() {
		log.Printf("fleet %s listening on %s, callback %s", cli.cfg.FleetID, cli.cfg.ListenAddr, cli.cfg.CallbackURL)
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
			log.Print("shutdown: deregistering webhooks")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			deregErr := reconciler.Deregister(shutdownCtx)
			if deregErr != nil {
				log.Printf("deregister: %v", deregErr)
			}
			_ = srv.Shutdown(shutdownCtx)
			cancel()
			// Check if the server goroutine also reported an error.
			select {
			case srvListenErr := <-srvErrCh:
				return fmt.Errorf("http server: %w", srvListenErr)
			default:
				return nil
			}
		case <-ticker.C:
			syncOnce(ctx, f, discovery, reconciler)
		}
	}
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

// normalizeCallbackURL strips trailing slashes so webhook URLs assemble cleanly.
func normalizeCallbackURL(u string) string {
	return strings.TrimRight(u, "/")
}

// validateConfig returns an error if any required field is missing or invalid.
func validateConfig(cfg fleet.Config) error {
	missing := []string{}
	if cfg.CallbackURL == "" {
		missing = append(missing, "CallbackURL (--callback-url / TRIP2G_CALLBACK_URL)")
	}
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWTSecret (--jwt-secret / TRIP2G_JWT_SECRET)")
	}
	if cfg.FleetSecret == "" {
		missing = append(missing, "FleetSecret (--fleet-secret / TRIP2G_FLEET_SECRET)")
	}
	if cfg.LLMAPIKey == "" {
		missing = append(missing, "LLMAPIKey (--llm-api-key / TRIP2G_LLM_API_KEY)")
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

func syncOnce(ctx context.Context, f *fleet.Fleet, d *fleet.Discovery, r *fleet.Reconciler) {
	roles, errs := d.DiscoverRoles(ctx)
	for _, e := range errs {
		log.Printf("discover (skipped role): %v", e)
	}
	f.SetRoles(roles)
	if err := r.Reconcile(ctx, roles); err != nil {
		log.Printf("reconcile: %v", err)
	}
}

// parseFlags registers all fleet flags on a dedicated FlagSet wired to
// appconfig.EnvFlag so every flag gets a TRIP2G_<FLAG_NAME> environment
// variable fallback (standard kebab-to-SCREAMING_SNAKE mapping).
func parseFlags(ctx context.Context) (cliFlags, error) {
	var cli cliFlags
	var offered string
	var poll int

	fs := flag.NewFlagSet("fleet", flag.ContinueOnError)

	// Daemon flags.
	fs.BoolVar(&cli.dryRun, "dry-run", false,
		"discover roles, print + flag their resolved config, then exit without registering webhooks")
	fs.StringVar(&cli.cfg.FleetID, "fleet-id", "fleet1",
		"reconcile marker id")
	fs.StringVar(&cli.cfg.ListenAddr, "listen", ":9090",
		"HTTP listen address")
	fs.StringVar(&cli.cfg.CallbackURL, "callback-url", "",
		"trip2g-reachable base URL of this fleet (required for daemon mode)")
	fs.StringVar(&cli.cfg.Trip2gBaseURL, "trip2g-url", "http://localhost:8081",
		"trip2g base URL")
	fs.StringVar(&cli.cfg.AdminAPIKey, "admin-api-key", "",
		"DEPRECATED/unused: legacy full-admin X-Api-Key")
	fs.StringVar(&cli.cfg.JWTSecret, "jwt-secret", "",
		"shared user-token/JWT secret for minting admin HATs (required for daemon mode)")
	fs.StringVar(&cli.cfg.AdminEmail, "admin-email", "fleet@local",
		"admin email the fleet self-provisions via HAT")
	fs.StringVar(&cli.cfg.FleetSecret, "fleet-secret", "",
		"HMAC seed for per-role secrets (required for daemon mode)")
	fs.StringVar(&cli.cfg.LLMBaseURL, "llm-base-url", "",
		"OpenAI-compatible base URL")
	fs.StringVar(&cli.cfg.LLMAPIKey, "llm-api-key", "",
		"LLM API key (falls back to OPENAI_API_KEY)")
	fs.StringVar(&cli.cfg.DefaultModel, "default-model", "gpt-4o-mini",
		"default model when a role omits model")
	fs.IntVar(&cli.cfg.TokenCeiling, "token-ceiling", 100000,
		"non-overridable per-run token cap")
	fs.IntVar(&cli.cfg.StepCeiling, "step-ceiling", 25,
		"non-overridable per-run step cap")
	fs.StringVar(&cli.cfg.AgentsFolder, "agents-folder", "roles/",
		"role-note folder (LIKE prefix)")
	fs.StringVar(&offered, "offered-tools", "search,read_note,patch_note,write_note",
		"comma-separated allowed tools")
	fs.IntVar(&poll, "poll-seconds", 30,
		"discovery/reconcile poll interval seconds")

	// One-shot offline harness flags.
	fs.StringVar(&cli.oncePath, "once", "",
		"path to a role-note .md file; runs it once without a trip2g connection and exits")
	fs.StringVar(&cli.vaultDir, "vault", ".",
		"vault directory for the local file KB (used with --once)")
	fs.StringVar(&cli.targetPath, "target", "",
		"note path in the vault to populate change_file context (used with --once)")

	ef := appconfig.New(appconfig.EnvFlagConfig{
		FlagSet:           fs,
		EnvPrefix:         "TRIP2G_",
		ShowEnvKeyInUsage: true,
		ShowEnvValInUsage: true,
	})

	if err := ef.Parse(ctx, os.Args[1:]); err != nil {
		return cliFlags{}, err
	}

	// OPENAI_API_KEY fallback for --once offline convenience.
	if cli.cfg.LLMAPIKey == "" {
		cli.cfg.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
	}

	cli.cfg.OfferedTools = splitCSV(offered)
	cli.cfg.PollInterval = time.Duration(poll) * time.Second
	return cli, nil
}

// runDryRun discovers (parses) every role over the admin lane, prints each
// role's resolved config with a flag on any that fail Validate, and returns
// without touching webhooks. It is the fleet's pre-flight doctor.
func runDryRun(ctx context.Context, d *fleet.Discovery, cfg fleet.Config) {
	roles, errs := d.DiscoverParsed(ctx)
	for _, e := range errs {
		log.Printf("dry-run parse error: %v", e)
	}
	fmt.Fprint(os.Stdout, reportRoles(roles, cfg.OfferedTools, cfg.DefaultModel))
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

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
