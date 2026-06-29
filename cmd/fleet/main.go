// Command fleet is the trip2g agent host (fleet-as-executor): it discovers role
// notes in trip2g, reconciles change-webhooks to point back at itself, receives
// deliveries, runs the scoped agent loop, and writes results back via a
// per-delivery scoped trip2g token. trip2g stays a dumb event source.
package main

import (
	"context"
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
	"trip2g/internal/fleet"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fleet: %v", err)
	}
}

func run() error {
	cfg, dryRun := parseFlags()

	// Normalize: strip trailing slash so webhook URLs assemble cleanly.
	cfg.CallbackURL = normalizeCallbackURL(cfg.CallbackURL)

	if err := validateConfig(cfg); err != nil {
		return err
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	client := fleet.NewHTTPClient(cfg.Trip2gBaseURL, httpClient)
	adminGQL := fleet.NewAdminGraphQLClient(cfg.Trip2gBaseURL, cfg.JWTSecret, cfg.AdminEmail, httpClient)
	llm := agentruntime.NewOpenAILLM(cfg.LLMAPIKey, cfg.LLMBaseURL)

	f := fleet.NewFleet(cfg, client, llm)
	discovery := fleet.NewDiscovery(adminGQL, cfg.AgentsFolder, cfg.OfferedTools)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --dry-run: connect, print + flag each role's resolved config, then exit
	// WITHOUT registering/reconciling any webhooks (eyeball roles before go-live).
	if dryRun {
		return runDryRun(ctx, discovery, cfg)
	}

	reconciler := fleet.NewReconciler(adminGQL, cfg)

	// First sync before serving so the registry is populated.
	syncOnce(ctx, f, discovery, reconciler)

	mux := http.NewServeMux()
	mux.HandleFunc("/deliver/", f.ServeDelivery)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	srvErr := make(chan error, 1)
	go func() {
		log.Printf("fleet %s listening on %s, callback %s", cfg.FleetID, cfg.ListenAddr, cfg.CallbackURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srvErr <- err
			stop()
		}
	}()

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case err := <-srvErr:
			return fmt.Errorf("http server: %w", err)
		case <-ctx.Done():
			log.Print("shutdown: deregistering webhooks")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			if err := reconciler.Deregister(shutdownCtx); err != nil {
				log.Printf("deregister: %v", err)
			}
			_ = srv.Shutdown(shutdownCtx)
			cancel()
			// Check if the server goroutine also reported an error.
			select {
			case err := <-srvErr:
				return fmt.Errorf("http server: %w", err)
			default:
				return nil
			}
		case <-ticker.C:
			syncOnce(ctx, f, discovery, reconciler)
		}
	}
}

// normalizeCallbackURL strips trailing slashes so webhook URLs assemble cleanly.
func normalizeCallbackURL(u string) string {
	return strings.TrimRight(u, "/")
}

// validateConfig returns an error if any required field is missing or invalid.
func validateConfig(cfg fleet.Config) error {
	missing := []string{}
	if cfg.CallbackURL == "" {
		missing = append(missing, "CallbackURL (--callback-url / FLEET_CALLBACK_URL)")
	}
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWTSecret (--jwt-secret / FLEET_JWT_SECRET)")
	}
	if cfg.FleetSecret == "" {
		missing = append(missing, "FleetSecret (--fleet-secret / FLEET_SECRET)")
	}
	if cfg.LLMAPIKey == "" {
		missing = append(missing, "LLMAPIKey (--llm-api-key / FLEET_LLM_API_KEY)")
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

func parseFlags() (fleet.Config, bool) {
	var cfg fleet.Config
	var offered string
	var dryRun bool
	flag.BoolVar(&dryRun, "dry-run", false, "discover roles, print + flag their resolved config, then exit without registering webhooks")
	flag.StringVar(&cfg.FleetID, "fleet-id", env("FLEET_ID", "fleet1"), "reconcile marker id")
	flag.StringVar(&cfg.ListenAddr, "listen", env("FLEET_LISTEN", ":9090"), "HTTP listen address")
	flag.StringVar(&cfg.CallbackURL, "callback-url", env("FLEET_CALLBACK_URL", ""), "trip2g-reachable base URL of this fleet")
	flag.StringVar(&cfg.Trip2gBaseURL, "trip2g-url", env("TRIP2G_BASE_URL", "http://localhost:8081"), "trip2g base URL")
	flag.StringVar(&cfg.AdminAPIKey, "admin-api-key", env("FLEET_ADMIN_API_KEY", ""), "DEPRECATED/unused: legacy full-admin X-Api-Key")
	flag.StringVar(&cfg.JWTSecret, "jwt-secret", env("FLEET_JWT_SECRET", ""), "shared user-token/JWT secret for minting admin HATs")
	flag.StringVar(&cfg.AdminEmail, "admin-email", env("FLEET_ADMIN_EMAIL", "fleet@local"), "admin email the fleet self-provisions via HAT")
	flag.StringVar(&cfg.FleetSecret, "fleet-secret", env("FLEET_SECRET", ""), "HMAC seed for per-role secrets")
	flag.StringVar(&cfg.LLMBaseURL, "llm-base-url", env("FLEET_LLM_BASE_URL", ""), "OpenAI-compatible base URL")
	flag.StringVar(&cfg.LLMAPIKey, "llm-api-key", env("FLEET_LLM_API_KEY", ""), "LLM API key")
	flag.StringVar(&cfg.DefaultModel, "default-model", env("FLEET_DEFAULT_MODEL", "gpt-4o-mini"), "default model")
	flag.IntVar(&cfg.TokenCeiling, "token-ceiling", 100000, "non-overridable per-run token cap")
	flag.IntVar(&cfg.StepCeiling, "step-ceiling", 25, "non-overridable per-run step cap")
	flag.StringVar(&cfg.AgentsFolder, "agents-folder", env("FLEET_AGENTS_FOLDER", "roles/"), "role-note folder (LIKE prefix)")
	flag.StringVar(&offered, "offered-tools", "search,read_note,patch_note,write_note", "comma-separated allowed tools")
	poll := flag.Int("poll-seconds", 30, "discovery/reconcile poll interval seconds")
	flag.Parse()

	cfg.OfferedTools = splitCSV(offered)
	cfg.PollInterval = time.Duration(*poll) * time.Second
	return cfg, dryRun
}

// runDryRun discovers (parses) every role over the admin lane, prints each
// role's resolved config with a flag on any that fail Validate, and returns
// without touching webhooks. It is the fleet's pre-flight doctor.
func runDryRun(ctx context.Context, d *fleet.Discovery, cfg fleet.Config) error {
	roles, errs := d.DiscoverParsed(ctx)
	for _, e := range errs {
		log.Printf("dry-run parse error: %v", e)
	}
	fmt.Print(reportRoles(roles, cfg.OfferedTools, cfg.DefaultModel))
	return nil
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

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
