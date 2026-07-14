// Package appconfig loads codellm's runtime configuration. It mirrors the
// STYLE of internal/appconfig (typed Config, a Get() that layers env vars and
// flags on top of defaults, ozzo validation) but is deliberately lightweight:
// codellm is one small binary with half a dozen settings, so it does not pull
// in the monolith's dependency surface (envflag's TRIP2G_ global-flag binding,
// MinIO/Patreon/audit-log sub-configs, etc.) — just the typed-Config + Get() +
// validate shape, sized to this service.
package appconfig

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/agentruntime"
)

// Config holds codellm's runtime configuration.
type Config struct {
	// Addr is the listen address for the OpenAI-compatible API. Defaults to
	// loopback: codellm's Auth/TokenCheck seams are no-op in Phase 1, so
	// binding to all interfaces must be an explicit operator opt-in, not the
	// default.
	Addr string

	// AllowedPrograms is the interpreter allowlist (python, bash, node, ...).
	// Empty disables code execution (every request then fails 422).
	AllowedPrograms []string

	// Sandbox is the OS-level isolation mode for each executed block.
	Sandbox agentruntime.SandboxMode

	// Timeout bounds a single completion's code run; 0 = request-context bound.
	Timeout time.Duration

	// MaxStdoutBytes caps each block's captured stdout; 0 = 1 MiB default.
	MaxStdoutBytes int

	// ChannelToken is a placeholder for the future fleet<->codellm locked-channel
	// check (shared token / mTLS material, per docs/dev/codellm_extraction.md).
	// Not enforced yet — a later PR wires channel auth using this value.
	ChannelToken string

	// MonolithURL is the trip2g monolith base URL used by the delegated-admin
	// gate on the browser-facing endpoints (/v1 via the browser proxy and
	// /graphql): each request's session cookie is forwarded to the monolith's
	// viewer{role} query. Defaults to loopback (the monolith on the same box,
	// mirroring the Caddy SSE proxy). Required and non-empty so the browser gate
	// is always wired (fail-closed).
	MonolithURL string
}

// Defaults.
const (
	DefaultAddr            = "127.0.0.1:8082"
	DefaultAllowedPrograms = "python,bash,node"
	DefaultTimeout         = 300 * time.Second
	DefaultMonolithURL     = "http://127.0.0.1:8081"
)

// DefaultConfig returns Config's baseline values, before env/flag overrides.
func DefaultConfig() Config {
	return Config{
		Addr:            DefaultAddr,
		AllowedPrograms: splitCSV(DefaultAllowedPrograms),
		Sandbox:         agentruntime.SandboxNative,
		Timeout:         DefaultTimeout,
		MaxStdoutBytes:  0,
		MonolithURL:     DefaultMonolithURL,
	}
}

// Get loads Config from defaults, then CODELLM_* environment variables, then
// os.Args command-line flags (highest precedence — matching internal/appconfig's
// "env overrides defaults, flags override env" layering), and validates it.
func Get() (*Config, error) {
	return GetArgs(os.Args[1:])
}

// GetArgs is Get with an explicit argv, so tests can drive flag parsing without
// touching the process's real os.Args.
func GetArgs(args []string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.applyEnv()

	if err := cfg.defineAndParseFlags(args); err != nil {
		return nil, fmt.Errorf("appconfig: parse flags: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("appconfig: invalid configuration: %w", err)
	}
	return &cfg, nil
}

// applyEnv overrides defaults from CODELLM_* environment variables. Flags
// (defineAndParseFlags), parsed afterward with these values as their defaults,
// take precedence when explicitly passed.
func (c *Config) applyEnv() {
	if v := os.Getenv("CODELLM_ADDR"); v != "" {
		c.Addr = v
	}
	if v := os.Getenv("CODELLM_ALLOWED_PROGRAMS"); v != "" {
		c.AllowedPrograms = splitCSV(v)
	}
	if v := os.Getenv("CODELLM_SANDBOX"); v != "" {
		c.Sandbox = agentruntime.SandboxMode(v)
	}
	if v := os.Getenv("CODELLM_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
	if v := os.Getenv("CODELLM_MAX_STDOUT_BYTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxStdoutBytes = n
		}
	}
	if v := os.Getenv("CODELLM_CHANNEL_TOKEN"); v != "" {
		c.ChannelToken = v
	}
	if v := os.Getenv("CODELLM_MONOLITH_URL"); v != "" {
		c.MonolithURL = v
	}
}

// defineAndParseFlags registers flags seeded from the env-resolved config (so
// an unset flag keeps the env/default value) and parses args. A dedicated
// FlagSet (not flag.CommandLine) keeps Get() safely repeatable in tests.
func (c *Config) defineAndParseFlags(args []string) error {
	fs := flag.NewFlagSet("codellm", flag.ContinueOnError)

	allowedPrograms := strings.Join(c.AllowedPrograms, ",")
	sandbox := string(c.Sandbox)

	fs.StringVar(&c.Addr, "addr", c.Addr, "listen address for the OpenAI-compatible API; defaults to loopback since auth is a no-op seam")
	fs.StringVar(&allowedPrograms, "allowed-programs", allowedPrograms, "comma-separated interpreter allowlist; empty disables code execution")
	fs.StringVar(&sandbox, "sandbox", sandbox, "sandbox mode: native | besteffort | off")
	fs.DurationVar(&c.Timeout, "timeout", c.Timeout, "per-completion code-run timeout; 0 = request-context bound")
	fs.IntVar(&c.MaxStdoutBytes, "max-stdout-bytes", c.MaxStdoutBytes, "stdout cap per code block; 0 = 1 MiB default")
	fs.StringVar(&c.ChannelToken, "channel-token", c.ChannelToken, "shared fleet<->codellm channel token (not yet enforced)")
	fs.StringVar(&c.MonolithURL, "monolith-url", c.MonolithURL, "trip2g monolith base URL for the delegated-admin gate on browser-facing endpoints")

	if err := fs.Parse(args); err != nil {
		return err
	}

	c.AllowedPrograms = splitCSV(allowedPrograms)
	c.Sandbox = agentruntime.SandboxMode(sandbox)
	return nil
}

func (c *Config) validate() error {
	return ozzo.ValidateStruct(c,
		ozzo.Field(&c.Addr, ozzo.Required),
		ozzo.Field(&c.MonolithURL, ozzo.Required),
		ozzo.Field(&c.MaxStdoutBytes, ozzo.Min(0)),
		ozzo.Field(&c.Timeout, ozzo.By(nonNegativeDuration)),
		ozzo.Field(&c.Sandbox, ozzo.In(agentruntime.SandboxNative, agentruntime.SandboxBestEffort, agentruntime.SandboxOff)),
	)
}

// nonNegativeDuration rejects a negative timeout (0 is valid: request-context
// bound).
func nonNegativeDuration(value any) error {
	d, ok := value.(time.Duration)
	if !ok {
		return errors.New("not a duration")
	}
	if d < 0 {
		return errors.New("must not be negative")
	}
	return nil
}

// splitCSV splits a comma-separated value into a trimmed, empty-free slice.
func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
