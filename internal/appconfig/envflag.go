// Package appconfig provides utilities for parsing command-line flags
// with environment variable fallbacks using modern Go practices.
package appconfig

import (
	"context"
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
	showEnvValInUsage bool
	logger            logger.Logger
}

// EnvFlagConfig contains configuration options for EnvFlag.
type EnvFlagConfig struct {
	FlagSet           *flag.FlagSet
	MinLength         int
	EnvFlagDict       map[string]string
	ShowEnvKeyInUsage bool
	ShowEnvValInUsage bool
	Logger            logger.Logger
}

// DefaultEnvFlagConfig returns a default configuration.
func DefaultEnvFlagConfig() EnvFlagConfig {
	return EnvFlagConfig{
		FlagSet:           flag.CommandLine,
		MinLength:         3,
		EnvFlagDict:       make(map[string]string),
		ShowEnvKeyInUsage: true,
		ShowEnvValInUsage: true,
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
		showEnvValInUsage: cfg.ShowEnvValInUsage,
		logger:            cfg.Logger,
	}
}

// ProcessError represents an error that occurred during flag processing.
type ProcessError struct {
	Flag  string
	Value string
	Err   error
}

func (e *ProcessError) Error() string {
	return fmt.Sprintf("error setting flag %q to %q: %v", e.Flag, e.Value, e.Err)
}

func (e *ProcessError) Unwrap() error {
	return e.Err
}

// ErrAlreadyParsed is returned when attempting to process flags that have already been parsed.
var ErrAlreadyParsed = fmt.Errorf("flags have already been parsed")

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
			envKey = flagToEnv(f.Name)
		}

		envPrefix := fmt.Sprintf("[%s]", envKey)
		if strings.HasPrefix(f.Usage, envPrefix) {
			return // Already updated
		}

		f.Usage = fmt.Sprintf("%s %s", envPrefix, f.Usage)
	})
}

// processEnvironmentVariables processes all environment variables and sets corresponding flags.
func (ef *EnvFlag) processEnvironmentVariables(ctx context.Context, flagEnvMap map[string]string) error {
	for _, envLine := range os.Environ() {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := ef.processEnvLine(envLine, flagEnvMap); err != nil {
			return err
		}
	}
	return nil
}

// processEnvLine processes a single environment variable line.
func (ef *EnvFlag) processEnvLine(envLine string, flagEnvMap map[string]string) error {
	envKV := strings.SplitN(envLine, "=", 2)
	if len(envKV) == 0 {
		return nil
	}

	key := envKV[0]
	if len(key) < ef.minLength {
		ef.logger.Debug("skipping environment variable (too short)", "key", key, "minLength", ef.minLength)
		return nil
	}

	value := ""
	if len(envKV) > 1 {
		value = envKV[1]
	}

	flagKey := ef.getFlagKey(key)
	f := ef.flagSet.Lookup(flagKey)
	if f == nil {
		ef.logger.Debug("skipping environment variable (flag not defined)", "envKey", key, "flagKey", flagKey)
		return nil
	}

	ef.logger.Debug("processing environment variable", "envKey", key, "flagKey", flagKey, "value", value)

	if ef.showEnvValInUsage {
		f.DefValue = value
	}

	if err := ef.flagSet.Set(flagKey, value); err != nil {
		ef.logger.Error("failed to set flag from environment variable", "envKey", key, "flagKey", flagKey, "value", value, "error", err)
		return &ProcessError{
			Flag:  flagKey,
			Value: value,
			Err:   err,
		}
	}

	ef.logger.Info("set flag from environment variable", "envKey", key, "flagKey", flagKey)
	return nil
}

// getFlagKey returns the flag name for the given environment variable key.
func (ef *EnvFlag) getFlagKey(envKey string) string {
	if flagName, exists := ef.envFlagDict[envKey]; exists {
		return flagName
	}
	return envToFlag(envKey)
}

// Parse processes environment variables and then parses command-line arguments.
// Environment variable values can be overridden by command-line arguments.
func (ef *EnvFlag) Parse(ctx context.Context, args []string) error {
	ef.logger.Info("starting flag parsing", "args", len(args))

	if err := ef.ProcessWithEnv(ctx); err != nil {
		ef.logger.Error("failed to process environment variables", "error", err)
		return fmt.Errorf("processing environment variables: %w", err)
	}

	ef.logger.Debug("parsing command-line arguments", "args", args)
	if err := ef.flagSet.Parse(args); err != nil {
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

// Standard instance for package-level functions
var std = New(DefaultEnvFlagConfig())

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

// SetShowEnvValInUsage controls whether environment variable values are shown as defaults.
func SetShowEnvValInUsage(v bool) {
	std.showEnvValInUsage = v
}

// SetLogger sets the logger for the standard instance.
func SetLogger(log logger.Logger) {
	std.SetLogger(log)
}

// envToFlag converts SCREAMING_SNAKE_CASE to kebab-case.
func envToFlag(env string) string {
	return strings.ReplaceAll(strings.ToLower(env), "_", "-")
}

// flagToEnv converts kebab-case to SCREAMING_SNAKE_CASE.
func flagToEnv(flag string) string {
	return strings.ReplaceAll(strings.ToUpper(flag), "-", "_")
}