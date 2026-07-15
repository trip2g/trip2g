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

	"trip2g/internal/coderun"
)

// Config holds codellm's runtime configuration.
type Config struct {
	// Addr is the listen address for the OpenAI-compatible API. Defaults to
	// loopback: codellm's auth is the api_key check plus the browser
	// delegated-admin cookie gate (see APIKey), so binding to all interfaces
	// must be an explicit operator opt-in, not the default.
	Addr string

	// AllowedPrograms is the interpreter allowlist (python, bash, node, ...).
	// Empty disables code execution (every request then fails 422).
	AllowedPrograms []string

	// Sandbox is the OS-level isolation mode for each executed block.
	Sandbox coderun.SandboxMode

	// Timeout bounds a single completion's code run; 0 = request-context bound.
	Timeout time.Duration

	// MaxStdoutBytes caps each block's captured stdout; 0 = 1 MiB default.
	MaxStdoutBytes int

	// APIKey is codellm's own OpenAI-standard api_key — the same credential shape
	// any OpenAI-compatible endpoint has. codellm accepts a caller presenting it
	// as `Authorization: Bearer <api_key>` (constant-time compared), bypassing
	// the browser cookie gate. Empty disables key auth (fail-safe): only the
	// browser delegated-admin gate then admits requests. When non-empty, must be
	// at least minAPIKeyLength chars (a short key would let key auth guard
	// against nothing).
	APIKey string

	// Trip2gBaseURL is the trip2g base URL used by the delegated-admin gate on
	// the browser-facing endpoints (/v1 via the browser proxy and /graphql):
	// each request's session cookie is forwarded to trip2g's viewer{role}
	// query. Defaults to loopback (trip2g on the same box, mirroring the Caddy
	// SSE proxy). Required and non-empty so the browser gate is always wired
	// (fail-closed). Named to match fleet's appconfig.Trip2gBaseURL.
	Trip2gBaseURL string

	// ExposeEnv / ExposeEnvPrefix are the operator's allowlist of env var NAMES
	// (exact) and name prefixes codellm exposes from its OWN environment to
	// executed code, on every run. codellm owns the secrets, so codellm alone
	// decides what to expose — the request carries nothing about env. Both empty
	// (the default) = expose nothing. Holding the secret VALUES here (deploy-time
	// env / mounted secret) is the point: fleet holds no secrets.
	ExposeEnv       []string
	ExposeEnvPrefix []string
}

// Defaults.
const (
	DefaultAddr            = "127.0.0.1:8087"
	DefaultAllowedPrograms = "python,bash,node"
	DefaultTimeout         = 300 * time.Second
	DefaultTrip2gBaseURL   = "http://127.0.0.1:8081"
)

// minAPIKeyLength is the minimum length a non-empty APIKey must have. Key auth
// guards an RCE-capable endpoint (code execution); a short key is guessable and
// makes that guard worthless. Empty is still allowed — it means key auth is off.
const minAPIKeyLength = 32

// DefaultConfig returns Config's baseline values, before env/flag overrides.
func DefaultConfig() Config {
	return Config{
		Addr:            DefaultAddr,
		AllowedPrograms: splitCSV(DefaultAllowedPrograms),
		Sandbox:         coderun.SandboxNative,
		Timeout:         DefaultTimeout,
		MaxStdoutBytes:  0,
		Trip2gBaseURL:   DefaultTrip2gBaseURL,
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
		c.Sandbox = coderun.SandboxMode(v)
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
	if v := os.Getenv("CODELLM_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("CODELLM_TRIP2G_URL"); v != "" {
		c.Trip2gBaseURL = v
	}
	if v := os.Getenv("CODELLM_EXPOSE_ENV"); v != "" {
		c.ExposeEnv = splitCSV(v)
	}
	if v := os.Getenv("CODELLM_EXPOSE_ENV_PREFIX"); v != "" {
		c.ExposeEnvPrefix = splitCSV(v)
	}
}

// defineAndParseFlags registers flags seeded from the env-resolved config (so
// an unset flag keeps the env/default value) and parses args. A dedicated
// FlagSet (not flag.CommandLine) keeps Get() safely repeatable in tests.
func (c *Config) defineAndParseFlags(args []string) error {
	fs := flag.NewFlagSet("codellm", flag.ContinueOnError)

	allowedPrograms := strings.Join(c.AllowedPrograms, ",")
	sandbox := string(c.Sandbox)
	exposeEnv := strings.Join(c.ExposeEnv, ",")
	exposeEnvPrefix := strings.Join(c.ExposeEnvPrefix, ",")

	fs.StringVar(&c.Addr, "addr", c.Addr, "listen address for the OpenAI-compatible API; defaults to loopback since auth is a no-op seam")
	fs.StringVar(&allowedPrograms, "allowed-programs", allowedPrograms, "comma-separated interpreter allowlist; empty disables code execution")
	fs.StringVar(&sandbox, "sandbox", sandbox, "sandbox mode: native | besteffort | off")
	fs.DurationVar(&c.Timeout, "timeout", c.Timeout, "per-completion code-run timeout; 0 = request-context bound")
	fs.IntVar(&c.MaxStdoutBytes, "max-stdout-bytes", c.MaxStdoutBytes, "stdout cap per code block; 0 = 1 MiB default")
	fs.StringVar(&c.APIKey, "api-key", c.APIKey, "codellm's OpenAI-standard api_key (Bearer); empty disables key auth")
	fs.StringVar(&c.Trip2gBaseURL, "trip2g-url", c.Trip2gBaseURL, "base URL of the trip2g instance that answers viewer{role}")
	fs.StringVar(&exposeEnv, "expose-env", exposeEnv,
		"comma-separated allowlist of env var NAMES codellm may expose from its own env to code "+
			"(a request can never exceed this); empty = expose nothing")
	fs.StringVar(&exposeEnvPrefix, "expose-env-prefix", exposeEnvPrefix,
		"comma-separated allowlist of env var name PREFIXES codellm may expose from its own env to code")

	if err := fs.Parse(args); err != nil {
		return err
	}

	c.AllowedPrograms = splitCSV(allowedPrograms)
	c.Sandbox = coderun.SandboxMode(sandbox)
	c.ExposeEnv = splitCSV(exposeEnv)
	c.ExposeEnvPrefix = splitCSV(exposeEnvPrefix)
	return nil
}

func (c *Config) validate() error {
	return ozzo.ValidateStruct(c,
		ozzo.Field(&c.Addr, ozzo.Required),
		ozzo.Field(&c.Trip2gBaseURL, ozzo.Required),
		ozzo.Field(&c.MaxStdoutBytes, ozzo.Min(0)),
		ozzo.Field(&c.Timeout, ozzo.By(nonNegativeDuration)),
		ozzo.Field(&c.Sandbox, ozzo.In(coderun.SandboxNative, coderun.SandboxBestEffort, coderun.SandboxOff)),
		ozzo.Field(&c.APIKey, ozzo.By(minAPIKeyLengthIfSet)),
	)
}

// minAPIKeyLengthIfSet allows an empty APIKey (key auth off) but rejects a
// non-empty one shorter than minAPIKeyLength (a guessable key would make key
// auth on an RCE-capable endpoint worthless).
func minAPIKeyLengthIfSet(value any) error {
	s, ok := value.(string)
	if !ok {
		return errors.New("not a string")
	}
	if s == "" {
		return nil
	}
	if len(s) < minAPIKeyLength {
		return fmt.Errorf("must be at least %d characters", minAPIKeyLength)
	}
	return nil
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
