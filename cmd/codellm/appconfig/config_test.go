package appconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/coderun"
)

func TestGetArgs_Defaults(t *testing.T) {
	cfg, err := GetArgs(nil)
	require.NoError(t, err)
	require.Equal(t, DefaultAddr, cfg.Addr, "must default to loopback, not all-interfaces")
	require.Equal(t, []string{"python", "bash", "node"}, cfg.AllowedPrograms)
	require.Equal(t, coderun.SandboxNative, cfg.Sandbox)
	require.Equal(t, DefaultTimeout, cfg.Timeout)
	require.Equal(t, 0, cfg.MaxStdoutBytes)
	require.Empty(t, cfg.APIKey)
}

func TestGetArgs_EnvOverridesDefaults(t *testing.T) {
	const apiKey = "secret-api-key-at-least-32-chars-long"
	t.Setenv("CODELLM_ADDR", "0.0.0.0:9999")
	t.Setenv("CODELLM_ALLOWED_PROGRAMS", "bash")
	t.Setenv("CODELLM_SANDBOX", "besteffort")
	t.Setenv("CODELLM_TIMEOUT", "5s")
	t.Setenv("CODELLM_MAX_STDOUT_BYTES", "1024")
	t.Setenv("CODELLM_API_KEY", apiKey)

	cfg, err := GetArgs(nil)
	require.NoError(t, err)
	require.Equal(t, "0.0.0.0:9999", cfg.Addr)
	require.Equal(t, []string{"bash"}, cfg.AllowedPrograms)
	require.Equal(t, coderun.SandboxBestEffort, cfg.Sandbox)
	require.Equal(t, 5*time.Second, cfg.Timeout)
	require.Equal(t, 1024, cfg.MaxStdoutBytes)
	require.Equal(t, apiKey, cfg.APIKey)
}

func TestGetArgs_FlagsOverrideEnv(t *testing.T) {
	t.Setenv("CODELLM_ADDR", "0.0.0.0:9999")

	cfg, err := GetArgs([]string{"-addr", "127.0.0.1:7777"})
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:7777", cfg.Addr, "explicit flag must win over env")
}

func TestGetArgs_InvalidSandboxRejected(t *testing.T) {
	_, err := GetArgs([]string{"-sandbox", "bogus"})
	require.Error(t, err)
}

func TestGetArgs_NegativeTimeoutRejected(t *testing.T) {
	_, err := GetArgs([]string{"-timeout", "-1s"})
	require.Error(t, err)
}

func TestGetArgs_NegativeMaxStdoutRejected(t *testing.T) {
	_, err := GetArgs([]string{"-max-stdout-bytes", "-1"})
	require.Error(t, err)
}

func TestGetArgs_EmptyAllowedProgramsDisablesExecution(t *testing.T) {
	cfg, err := GetArgs([]string{"-allowed-programs", ""})
	require.NoError(t, err)
	require.Empty(t, cfg.AllowedPrograms)
}

func TestGetArgs_EmptyAPIKeyAllowed(t *testing.T) {
	cfg, err := GetArgs(nil)
	require.NoError(t, err)
	require.Empty(t, cfg.APIKey, "empty api-key means key auth is off, not a validation error")
}

func TestGetArgs_ShortAPIKeyRejected(t *testing.T) {
	short := make([]byte, minAPIKeyLength-1)
	for i := range short {
		short[i] = 'a'
	}
	_, err := GetArgs([]string{"-api-key", string(short)})
	require.Error(t, err, "a %d-char key is below the minimum and must be rejected", minAPIKeyLength-1)
}

func TestGetArgs_MinLengthAPIKeyAccepted(t *testing.T) {
	key := make([]byte, minAPIKeyLength)
	for i := range key {
		key[i] = 'a'
	}
	cfg, err := GetArgs([]string{"-api-key", string(key)})
	require.NoError(t, err)
	require.Equal(t, string(key), cfg.APIKey)
}
