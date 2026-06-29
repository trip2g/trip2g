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
	cfg, offeredFlag := parseFlags()

	// Normalize: strip trailing slash so webhook URLs assemble cleanly.
	cfg.CallbackURL = normalizeCallbackURL(cfg.CallbackURL)

	if err := validateConfig(cfg); err != nil {
		return err
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	client := fleet.NewHTTPClient(cfg.Trip2gBaseURL, cfg.AdminAPIKey, httpClient)
	llm := agentruntime.NewOpenAILLM(cfg.LLMAPIKey, cfg.LLMBaseURL)

	f := fleet.NewFleet(cfg, client, llm)
	discovery := fleet.NewDiscovery(client, cfg.AgentsFolder, cfg.OfferedTools)
	reconciler := fleet.NewReconciler(client, cfg)
	_ = offeredFlag

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	if cfg.AdminAPIKey == "" {
		missing = append(missing, "AdminAPIKey (--admin-api-key / FLEET_ADMIN_API_KEY)")
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

func parseFlags() (fleet.Config, []string) {
	var cfg fleet.Config
	var offered string
	flag.StringVar(&cfg.FleetID, "fleet-id", env("FLEET_ID", "fleet1"), "reconcile marker id")
	flag.StringVar(&cfg.ListenAddr, "listen", env("FLEET_LISTEN", ":9090"), "HTTP listen address")
	flag.StringVar(&cfg.CallbackURL, "callback-url", env("FLEET_CALLBACK_URL", ""), "trip2g-reachable base URL of this fleet")
	flag.StringVar(&cfg.Trip2gBaseURL, "trip2g-url", env("TRIP2G_BASE_URL", "http://localhost:8081"), "trip2g base URL")
	flag.StringVar(&cfg.AdminAPIKey, "admin-api-key", env("FLEET_ADMIN_API_KEY", ""), "full-admin X-Api-Key")
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
	return cfg, cfg.OfferedTools
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
