// Package appconfig is cmd/fleet's own typed configuration layer: env + flags
// -> validated Config, mirroring internal/appconfig's Config+Get() pattern at
// a scale proportional to fleet — a small standalone binary, not the
// monolith. It owns the settings named in docs/dev/fleet_graphql.md's fleet
// GraphQL work: discovery scope, the run/reconcile addresses, the default
// model, the offered-tools allowlist, and the fleet GraphQL listen address.
// Fleet's secrets/ceilings/sandbox knobs stay defined inline in
// cmd/fleet/main.go's fleet.Config wiring; folding those in too is a larger,
// separate refactor.
package appconfig

import (
	"context"
	"flag"
	"strings"

	baseappconfig "trip2g/internal/appconfig"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

// Defaults mirror the flag defaults cmd/fleet has always used.
const (
	DefaultAgentsFolder  = "roles/"
	DefaultListenAddr    = ":9090"
	DefaultTrip2gBaseURL = "http://localhost:8081"
	DefaultModel         = "gpt-4o-mini"
	DefaultOfferedTools  = "search,read_note,patch_note,write_note"

	// DefaultGraphQLAddr binds the fleet GraphQL API to loopback by default so
	// exposure is an explicit operator opt-in, with Caddy /_fleet/* as the sole
	// ingress (mirrors codellm's loopback default). The API is gated by the
	// delegated-admin middleware regardless of bind.
	DefaultGraphQLAddr = "127.0.0.1:9093"

	// DefaultMetricsAddr binds the internal listener (Prometheus /metrics, pprof,
	// liveness/readiness) to loopback: none of it is authenticated, exactly like
	// the monolith's internal listener. The 18xxx band keeps the standalone
	// binaries clear of the 19xxx internal ports infra/site.yml assigns one per
	// site ("MUST be unique per service"; 19090 is already reserved there):
	// codellm 8087 -> 18087, fleet 9090 -> 18090.
	DefaultMetricsAddr = "127.0.0.1:18090"
)

// Config holds the subset of fleet's machine-level settings layered via
// env+flags through this package. ListenAddr/CallbackURL/Trip2gBaseURL are the
// run/reconcile addresses; the rest select discovery scope, the default
// model, and the GraphQL surface.
type Config struct {
	AgentsFolder  string   // role-note folder (LIKE prefix), e.g. "roles/"
	ListenAddr    string   // fleet's delivery HTTP listen address (run)
	CallbackURL   string   // trip2g-reachable base URL of this fleet (reconcile: webhook target)
	Trip2gBaseURL string   // trip2g base URL fleet talks to (reconcile: discovery/registration)
	GraphQLAddr   string   // fleet's own GraphQL read API (roles + roleGraph); empty = disabled
	MetricsAddr   string   // internal listener: /metrics, pprof, probes; loopback-only, empty = disabled
	DefaultModel  string   // fallback model when a role omits model
	OfferedTools  []string // allowed tools a role may declare

	offeredToolsCSV string // raw flag value; Prepare splits it into OfferedTools
}

// DefaultConfig returns a Config with default values, matching the flag
// defaults fleet has always used.
func DefaultConfig() *Config {
	return &Config{
		AgentsFolder:    DefaultAgentsFolder,
		ListenAddr:      DefaultListenAddr,
		Trip2gBaseURL:   DefaultTrip2gBaseURL,
		GraphQLAddr:     DefaultGraphQLAddr,
		MetricsAddr:     DefaultMetricsAddr,
		DefaultModel:    DefaultModel,
		offeredToolsCSV: DefaultOfferedTools,
	}
}

// DefineFlags registers this Config's flags on fs. It does not parse: fleet
// shares one FlagSet across all its flags (this package's and the rest defined
// directly in cmd/fleet/main.go) and parses once, so DefineFlags only binds —
// callers run Prepare and Validate themselves after that single Parse.
func (c *Config) DefineFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.AgentsFolder, "agents-folder", c.AgentsFolder,
		"role-note folder (LIKE prefix)")
	fs.StringVar(&c.ListenAddr, "listen", c.ListenAddr,
		"HTTP listen address")
	fs.StringVar(&c.CallbackURL, "callback-url", c.CallbackURL,
		"trip2g-reachable base URL of this fleet (required for daemon mode)")
	fs.StringVar(&c.Trip2gBaseURL, "trip2g-url", c.Trip2gBaseURL,
		"trip2g base URL")
	fs.StringVar(&c.GraphQLAddr, "graphql-addr", c.GraphQLAddr,
		"listen address serving the fleet GraphQL read API (POST /graphql: roles + roleGraph); "+
			"defaults to loopback (127.0.0.1:9093), empty = disabled. The whole port is gated by "+
			"the delegated-admin middleware (forwards the caller's cookie to the monolith's "+
			"viewer{role}; admin -> serve, else 401, monolith-unreachable -> fail-closed). Front it "+
			"with Caddy /_fleet/* as the sole ingress; expose a non-loopback bind only behind that proxy")
	fs.StringVar(&c.MetricsAddr, "metrics-addr", c.MetricsAddr,
		"loopback listen address serving Prometheus /metrics, pprof and the liveness/readiness probes; "+
			"unauthenticated by design, so never bind it off-box. Empty = disabled")
	fs.StringVar(&c.DefaultModel, "default-model", c.DefaultModel,
		"default model when a role omits model")
	fs.StringVar(&c.offeredToolsCSV, "offered-tools", c.offeredToolsCSV,
		"comma-separated allowed tools")
}

// Prepare finalizes derived fields after flags/env have been parsed (splits
// the offered-tools CSV). Call once, after Parse and before Validate.
func (c *Config) Prepare() {
	c.OfferedTools = SplitCSV(c.offeredToolsCSV)
}

// Validate checks the fields this package owns. AgentsFolder and OfferedTools
// are required for discovery to do anything. GraphQLAddr/CallbackURL/
// Trip2gBaseURL/DefaultModel have no hard requirement here:
// CallbackURL's daemon-mode requirement is enforced by cmd/fleet's own
// validateConfig (fleet.Config), and the GraphQLAddr non-loopback caution
// stays a documentation note (see DefineFlags), not a hard rule, since an
// operator may front it with their own auth/proxy.
func (c *Config) Validate() error {
	return ozzo.ValidateStruct(c,
		ozzo.Field(&c.AgentsFolder, ozzo.Required),
		ozzo.Field(&c.OfferedTools, ozzo.Required),
	)
}

// Get loads a Config from env + flags on its own dedicated FlagSet: define,
// parse (env first, then args, via internal/appconfig's TRIP2G_FLEET_ prefix
// convention), prepare, validate. A standalone entry point for tests and any
// future caller that doesn't need to share a FlagSet with the rest of fleet's
// flags; cmd/fleet's main() instead calls DefineFlags directly on its own
// shared FlagSet (so all of fleet's flags parse in one pass) and calls
// Prepare/Validate itself after that single Parse.
func Get(ctx context.Context, args []string) (*Config, error) {
	fs := flag.NewFlagSet("fleet-appconfig", flag.ContinueOnError)
	cfg := DefaultConfig()
	cfg.DefineFlags(fs)

	ef := baseappconfig.New(baseappconfig.EnvFlagConfig{
		FlagSet:           fs,
		EnvPrefix:         "TRIP2G_FLEET_",
		ShowEnvKeyInUsage: true,
	})
	if err := ef.Parse(ctx, args); err != nil {
		return nil, err
	}
	cfg.Prepare()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SplitCSV splits a comma-separated flag value into a trimmed, non-empty
// slice of parts (blank entries dropped).
func SplitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
