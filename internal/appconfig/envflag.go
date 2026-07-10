// Package appconfig provides utilities for parsing command-line flags
// with environment variable fallbacks using modern Go practices.
package appconfig

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"trip2g/internal/logger"
)

// EnvFlag provides environment variable integration for command-line flags.
type EnvFlag struct {
	flagSet           *flag.FlagSet
	minLength         int
	envFlagDict       map[string]string
	showEnvKeyInUsage bool
	envPrefix         string
	logger            logger.Logger
}

// EnvFlagConfig contains configuration options for EnvFlag.
type EnvFlagConfig struct {
	FlagSet           *flag.FlagSet
	MinLength         int
	EnvFlagDict       map[string]string
	ShowEnvKeyInUsage bool
	// EnvPrefix, when set, restricts processing to env vars with this prefix.
	// Prefixed vars that don't match any flag are treated as errors (likely typos).
	// Example: "TRIP2G_" makes TRIP2G_LISTEN_ADDR map to flag listen-addr.
	EnvPrefix string
	Logger    logger.Logger
}

// DefaultEnvFlagConfig returns a default configuration.
func DefaultEnvFlagConfig() EnvFlagConfig {
	return EnvFlagConfig{
		FlagSet:           flag.CommandLine,
		MinLength:         3,
		EnvFlagDict:       make(map[string]string),
		ShowEnvKeyInUsage: true,
		EnvPrefix:         "TRIP2G_",
		Logger:            &logger.DummyLogger{},
	}
}

// New creates a new EnvFlag instance with the provided configuration.
func New(cfg EnvFlagConfig) *EnvFlag {
	if cfg.FlagSet == nil {
		cfg.FlagSet = flag.CommandLine
	}
	if cfg.EnvFlagDict == nil {
		cfg.EnvFlagDict = make(map[string]string)
	}
	if cfg.Logger == nil {
		cfg.Logger = &logger.DummyLogger{}
	}

	return &EnvFlag{
		flagSet:           cfg.FlagSet,
		minLength:         cfg.MinLength,
		envFlagDict:       cfg.EnvFlagDict,
		showEnvKeyInUsage: cfg.ShowEnvKeyInUsage,
		envPrefix:         cfg.EnvPrefix,
		logger:            cfg.Logger,
	}
}

// ProcessError represents an error that occurred during flag processing.
// The value is intentionally omitted — env-loaded values may be secrets.
type ProcessError struct {
	Flag string
	Err  error
}

func (e *ProcessError) Error() string {
	return fmt.Sprintf("error setting flag %q: %v", e.Flag, e.Err)
}

func (e *ProcessError) Unwrap() error {
	return e.Err
}

// ErrAlreadyParsed is returned when attempting to process flags that have already been parsed.
var ErrAlreadyParsed = errors.New("flags have already been parsed")

// ProcessWithEnv processes environment variables and updates flag defaults.
// It returns an error if flags have already been parsed or if there's an issue
// setting flag values.
func (ef *EnvFlag) ProcessWithEnv(ctx context.Context) error {
	if ef.flagSet.Parsed() {
		ef.logger.Error("attempted to process environment variables after flags were parsed")
		return ErrAlreadyParsed
	}

	ef.logger.Debug("starting environment variable processing", "minLength", ef.minLength)

	// Create reverse mapping for faster lookups
	flagEnvMap := make(map[string]string, len(ef.envFlagDict))
	for envKey, flagName := range ef.envFlagDict {
		flagEnvMap[flagName] = envKey
	}

	if len(ef.envFlagDict) > 0 {
		ef.logger.Debug("using custom environment variable mappings", "count", len(ef.envFlagDict))
	}

	// Update usage strings to show environment variable names
	if ef.showEnvKeyInUsage {
		ef.logger.Debug("updating flag usage with environment variable names")
		ef.updateUsageWithEnvKeys(flagEnvMap)
	}

	// Process environment variables
	ef.logger.Debug("processing environment variables")
	return ef.processEnvironmentVariables(ctx, flagEnvMap)
}

// updateUsageWithEnvKeys updates flag usage strings to include environment variable names.
func (ef *EnvFlag) updateUsageWithEnvKeys(flagEnvMap map[string]string) {
	ef.flagSet.VisitAll(func(f *flag.Flag) {
		if len(f.Name) < ef.minLength {
			return
		}

		envKey, exists := flagEnvMap[f.Name]
		if !exists {
			envKey = ef.envPrefix + flagToEnv(f.Name)
		}

		envPrefix := fmt.Sprintf("[%s]", envKey)
		if strings.HasPrefix(f.Usage, envPrefix) {
			return // Already updated
		}

		f.Usage = fmt.Sprintf("%s %s", envPrefix, f.Usage)
	})
}

// processEnvironmentVariables processes all environment variables and sets corresponding flags.
// When a prefix is configured two passes are made: unprefixed vars first, then prefixed vars,
// so prefixed values always take precedence. Unknown prefixed vars are aggregated into a single
// error returned after all vars have been processed.
func (ef *EnvFlag) processEnvironmentVariables(ctx context.Context, flagEnvMap map[string]string) error {
	envLines := os.Environ()
	var unknownPrefixed []string

	runPass := func(wantPrefixed bool) error {
		for _, envLine := range envLines {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			key, _, _ := strings.Cut(envLine, "=")
			hasPfx := ef.envPrefix != "" && strings.HasPrefix(key, ef.envPrefix)
			if hasPfx != wantPrefixed {
				continue
			}
			unknown, err := ef.processEnvLine(envLine, flagEnvMap)
			if err != nil {
				return err
			}
			if unknown != "" {
				unknownPrefixed = append(unknownPrefixed, unknown)
			}
		}
		return nil
	}

	// Pass 1: unprefixed (or everything when no prefix is configured).
	if err := runPass(false); err != nil {
		return err
	}
	// Pass 2: prefixed vars override unprefixed ones.
	if ef.envPrefix != "" {
		if err := runPass(true); err != nil {
			return err
		}
	}

	if len(unknownPrefixed) > 0 {
		fmt.Fprintf(os.Stderr,
			"WRN unknown environment variables with prefix %q (possible typos): %s\n",
			ef.envPrefix, strings.Join(unknownPrefixed, ", "),
		)
	}
	return nil
}

// processEnvLine processes a single environment variable line.
// Returns the key if it carries the configured prefix but maps to no known flag (collected by
// the caller for a deferred error), or an error for hard failures like flag.Set.
func (ef *EnvFlag) processEnvLine(envLine string, _ map[string]string) (string, error) {
	envKV := strings.SplitN(envLine, "=", 2)
	if len(envKV) == 0 {
		return "", nil
	}

	key := envKV[0]
	if ef.envPrefix == "" && len(key) < ef.minLength {
		ef.logger.Debug("skipping environment variable (too short)", "key", key, "minLength", ef.minLength)
		return "", nil
	}

	value := ""
	if len(envKV) > 1 {
		value = envKV[1]
	}

	flagKey := ef.getFlagKey(key)
	f := ef.flagSet.Lookup(flagKey)
	if f == nil {
		if ef.envPrefix != "" && strings.HasPrefix(key, ef.envPrefix) {
			ef.logger.Debug("unknown prefixed environment variable", "envKey", key, "flagKey", flagKey)
			return key, nil // deferred error — caller aggregates
		}
		ef.logger.Debug("skipping environment variable (flag not defined)", "envKey", key, "flagKey", flagKey)
		return "", nil
	}

	// Never log or expose the value: env-loaded values may be secrets, and
	// setting them as the flag's printable default would leak them via --help.
	ef.logger.Debug("processing environment variable", "envKey", key, "flagKey", flagKey)

	err := ef.flagSet.Set(flagKey, value)
	if err != nil {
		ef.logger.Error(
			"failed to set flag from environment variable",
			"envKey", key,
			"flagKey", flagKey,
			"error", err,
		)
		return "", &ProcessError{
			Flag: flagKey,
			Err:  err,
		}
	}

	ef.logger.Info("set flag from environment variable", "envKey", key, "flagKey", flagKey)
	return "", nil
}

// getFlagKey returns the flag name for the given environment variable key.
func (ef *EnvFlag) getFlagKey(envKey string) string {
	if flagName, exists := ef.envFlagDict[envKey]; exists {
		return flagName
	}
	key := strings.TrimPrefix(envKey, ef.envPrefix)
	return envToFlag(key)
}

// Parse processes environment variables and then parses command-line arguments.
// Environment variable values can be overridden by command-line arguments.
func (ef *EnvFlag) Parse(ctx context.Context, args []string) error {
	ef.logger.Info("starting flag parsing", "args", len(args))

	err := ef.ProcessWithEnv(ctx)
	if err != nil {
		ef.logger.Error("failed to process environment variables", "error", err)
		return fmt.Errorf("processing environment variables: %w", err)
	}

	ef.logger.Debug("parsing command-line arguments", "args", args)
	err = ef.flagSet.Parse(args)
	if err != nil {
		ef.logger.Error("failed to parse command-line arguments", "error", err, "args", args)
		return fmt.Errorf("parsing command-line arguments: %w", err)
	}

	ef.logger.Info("flag parsing completed successfully")
	return nil
}

// ParseWithTimeout is like Parse but with a timeout for environment processing.
func (ef *EnvFlag) ParseWithTimeout(timeout time.Duration, args []string) error {
	ef.logger.Debug("parsing with timeout", "timeout", timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return ef.Parse(ctx, args)
}

// SetLogger updates the logger for this EnvFlag instance.
func (ef *EnvFlag) SetLogger(log logger.Logger) {
	if log != nil {
		ef.logger = log
	}
}

// Standard instance for package-level functions.
var std = New(DefaultEnvFlagConfig()) //nolint:gochecknoglobals // it's a common pattern

// ProcessWithEnv processes environment variables using the standard instance.
func ProcessWithEnv(ctx context.Context) error {
	return std.ProcessWithEnv(ctx)
}

// Parse parses environment variables and command-line arguments using the standard instance.
func Parse(ctx context.Context) error {
	return std.Parse(ctx, os.Args[1:])
}

// ParseWithTimeout parses with a timeout using the standard instance.
func ParseWithTimeout(timeout time.Duration) error {
	return std.ParseWithTimeout(timeout, os.Args[1:])
}

// SetMinLength sets the minimum length for environment variable processing.
func SetMinLength(v int) {
	std.minLength = v
}

// SetEnvFlagDict sets a custom environment variable to flag name mapping.
func SetEnvFlagDict(v map[string]string) {
	std.envFlagDict = make(map[string]string, len(v))
	for k, v := range v {
		std.envFlagDict[k] = v
	}
}

// SetShowEnvKeyInUsage controls whether environment variable names are shown in usage.
func SetShowEnvKeyInUsage(v bool) {
	std.showEnvKeyInUsage = v
}

// SetLogger sets the logger for the standard instance.
func SetLogger(log logger.Logger) {
	std.SetLogger(log)
}

// SetEnvPrefix sets an environment variable prefix for the standard instance.
// When set, only vars with this prefix are processed; unrecognised prefixed vars
// return an error instead of being silently skipped.
func SetEnvPrefix(v string) {
	std.envPrefix = v
}

// envToFlag converts SCREAMING_SNAKE_CASE to kebab-case.
func envToFlag(env string) string {
	return strings.ReplaceAll(strings.ToLower(env), "_", "-")
}

// flagToEnv converts kebab-case to SCREAMING_SNAKE_CASE.
func flagToEnv(flag string) string {
	return strings.ReplaceAll(strings.ToUpper(flag), "-", "_")
}
