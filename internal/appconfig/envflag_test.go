package appconfig

import (
	"bytes"
	"context"
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
)

// Environment-backed secrets may configure a flag, but they must never become
// that flag's printable default: flag.PrintDefaults is what powers --help.
func TestProcessWithEnv_HelpDoesNotExposeSecretValue(t *testing.T) {
	const secret = "ENV-SECRET-CANARY"
	t.Setenv("REGRESSION_JWT_SECRET", secret)

	var usage bytes.Buffer
	flags := flag.NewFlagSet("env-help", flag.ContinueOnError)
	flags.SetOutput(&usage)
	flags.String("jwt-secret", "", "JWT signing secret")

	cfg := DefaultEnvFlagConfig()
	cfg.FlagSet = flags
	cfg.EnvPrefix = "REGRESSION_"
	envFlags := New(cfg)

	require.NoError(t, envFlags.ProcessWithEnv(context.Background()))
	flags.PrintDefaults()
	require.NotContains(t, usage.String(), secret,
		"--help output must not expose secret values loaded from the environment")
}
