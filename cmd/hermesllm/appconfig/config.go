// Package appconfig loads hermesllm's runtime configuration. It mirrors
// cmd/codellm/appconfig (typed Config, a Get() that layers env vars and flags on
// top of defaults, ozzo validation) — the same lightweight shape, sized to a
// binary with half a dozen settings and no monolith dependency surface.
package appconfig

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

// Config holds hermesllm's runtime configuration.
type Config struct {
	// Addr is the listen address for the OpenAI-compatible API. Defaults to
	// loopback: the api_key check is optional (see APIKey), so binding to all
	// interfaces must be an explicit operator opt-in.
	Addr string

	// HermesURL is the BASE URL of the Hermes agent; the shim posts to
	// {HermesURL}/v1/chat/completions.
	HermesURL string

	// HermesKey is the upstream credential, sent as `Authorization: Bearer
	// <key>` to Hermes. Required: Hermes never runs unauthenticated.
	HermesKey string

	// APIKey is hermesllm's own OpenAI-standard api_key — the credential clients
	// (fleet's OpenAILLM included) present as `Authorization: Bearer <api_key>`,
	// compared in constant time. Empty DISABLES the check: unlike codellm there is
	// no second (browser cookie) lane here, so an unset key means an open
	// endpoint. When non-empty it must be at least minAPIKeyLength chars.
	APIKey string

	// Model is the id advertised by GET /v1/models and echoed in responses.
	// Hermes pins its own model, so this is only a label for clients.
	Model string

	// Timeout bounds a single upstream Hermes call; 0 = request-context bound.
	// Hermes is a full agent, so the default is generous.
	Timeout time.Duration
}

// Defaults.
const (
	DefaultAddr      = "127.0.0.1:8088"
	DefaultHermesURL = "http://127.0.0.1:8642"
	DefaultModel     = "hermes-agent"
	DefaultTimeout   = 300 * time.Second
)

// minAPIKeyLength is the minimum length a non-empty APIKey must have: a short
// key is guessable and would make the check worthless. Empty is still allowed —
// it means key auth is off.
const minAPIKeyLength = 32

// DefaultConfig returns Config's baseline values, before env/flag overrides.
func DefaultConfig() Config {
	return Config{
		Addr:      DefaultAddr,
		HermesURL: DefaultHermesURL,
		Model:     DefaultModel,
		Timeout:   DefaultTimeout,
	}
}

// Get loads Config from defaults, then HERMESLLM_* environment variables, then
// os.Args command-line flags (highest precedence), and validates it.
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

// applyEnv overrides defaults from HERMESLLM_* environment variables. Flags,
// parsed afterward with these values as their defaults, take precedence when
// explicitly passed.
func (c *Config) applyEnv() {
	if v := os.Getenv("HERMESLLM_ADDR"); v != "" {
		c.Addr = v
	}
	if v := os.Getenv("HERMESLLM_HERMES_URL"); v != "" {
		c.HermesURL = v
	}
	if v := os.Getenv("HERMESLLM_HERMES_KEY"); v != "" {
		c.HermesKey = v
	}
	if v := os.Getenv("HERMESLLM_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("HERMESLLM_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("HERMESLLM_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
}

// defineAndParseFlags registers flags seeded from the env-resolved config (so an
// unset flag keeps the env/default value) and parses args. A dedicated FlagSet
// (not flag.CommandLine) keeps Get() safely repeatable in tests.
func (c *Config) defineAndParseFlags(args []string) error {
	fs := flag.NewFlagSet("hermesllm", flag.ContinueOnError)

	fs.StringVar(&c.Addr, "addr", c.Addr, "listen address for the OpenAI-compatible API")
	fs.StringVar(&c.HermesURL, "hermes-url", c.HermesURL, "base URL of the Hermes agent; requests go to {url}/v1/chat/completions")
	fs.StringVar(&c.HermesKey, "hermes-key", c.HermesKey, "upstream Hermes credential, sent as Authorization: Bearer")
	fs.StringVar(&c.APIKey, "api-key", c.APIKey, "hermesllm's OpenAI-standard api_key (Bearer); empty disables key auth")
	fs.StringVar(&c.Model, "model", c.Model, "model id advertised by /v1/models and echoed in responses")
	fs.DurationVar(&c.Timeout, "timeout", c.Timeout, "per-request upstream Hermes timeout; 0 = request-context bound")

	return fs.Parse(args)
}

func (c *Config) validate() error {
	return ozzo.ValidateStruct(c,
		ozzo.Field(&c.Addr, ozzo.Required),
		ozzo.Field(&c.HermesURL, ozzo.Required),
		ozzo.Field(&c.HermesKey, ozzo.Required),
		ozzo.Field(&c.Model, ozzo.Required),
		ozzo.Field(&c.Timeout, ozzo.By(nonNegativeDuration)),
		ozzo.Field(&c.APIKey, ozzo.By(minAPIKeyLengthIfSet)),
	)
}

// minAPIKeyLengthIfSet allows an empty APIKey (key auth off) but rejects a
// non-empty one shorter than minAPIKeyLength.
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
