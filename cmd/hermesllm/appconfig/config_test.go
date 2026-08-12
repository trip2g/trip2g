package appconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// withKey prefixes the minimal argv that satisfies the required upstream key, so
// each test varies exactly the setting it cares about.
func withKey(args ...string) []string {
	return append([]string{"-hermes-key", "upstream-key"}, args...)
}

// fillerKey builds an n-char api key.
func fillerKey(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

func TestGetArgs_Defaults(t *testing.T) {
	cfg, err := GetArgs(withKey())
	require.NoError(t, err)
	require.Equal(t, DefaultAddr, cfg.Addr, "must default to loopback, not all-interfaces")
	require.Equal(t, DefaultHermesURL, cfg.HermesURL)
	require.Equal(t, DefaultModel, cfg.Model)
	require.Equal(t, DefaultTimeout, cfg.Timeout)
	require.Empty(t, cfg.APIKey)
}

func TestGetArgs_EnvOverridesDefaults(t *testing.T) {
	const apiKey = "secret-api-key-at-least-32-chars-long"
	t.Setenv("HERMESLLM_ADDR", "0.0.0.0:9999")
	t.Setenv("HERMESLLM_HERMES_URL", "http://hermes:8642")
	t.Setenv("HERMESLLM_HERMES_KEY", "upstream-key")
	t.Setenv("HERMESLLM_API_KEY", apiKey)
	t.Setenv("HERMESLLM_MODEL", "custom-agent")
	t.Setenv("HERMESLLM_TIMEOUT", "5s")

	cfg, err := GetArgs(nil)
	require.NoError(t, err)
	require.Equal(t, "0.0.0.0:9999", cfg.Addr)
	require.Equal(t, "http://hermes:8642", cfg.HermesURL)
	require.Equal(t, "upstream-key", cfg.HermesKey)
	require.Equal(t, apiKey, cfg.APIKey)
	require.Equal(t, "custom-agent", cfg.Model)
	require.Equal(t, 5*time.Second, cfg.Timeout)
}

func TestGetArgs_FlagsOverrideEnv(t *testing.T) {
	t.Setenv("HERMESLLM_ADDR", "0.0.0.0:9999")
	t.Setenv("HERMESLLM_HERMES_KEY", "upstream-key")

	cfg, err := GetArgs([]string{"-addr", "127.0.0.1:7777"})
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:7777", cfg.Addr, "explicit flag must win over env")
}

func TestGetArgs_MissingHermesKeyRejected(t *testing.T) {
	_, err := GetArgs(nil)
	require.Error(t, err, "the upstream Hermes key is required")
}

func TestGetArgs_NegativeTimeoutRejected(t *testing.T) {
	_, err := GetArgs(withKey("-timeout", "-1s"))
	require.Error(t, err)
}

func TestGetArgs_EmptyHermesURLRejected(t *testing.T) {
	_, err := GetArgs(withKey("-hermes-url", ""))
	require.Error(t, err)
}

func TestGetArgs_EmptyAPIKeyAllowed(t *testing.T) {
	cfg, err := GetArgs(withKey())
	require.NoError(t, err)
	require.Empty(t, cfg.APIKey, "empty api-key means key auth is off, not a validation error")
}

func TestGetArgs_ShortAPIKeyRejected(t *testing.T) {
	_, err := GetArgs(withKey("-api-key", fillerKey(minAPIKeyLength-1)))
	require.Error(t, err, "a %d-char key is below the minimum and must be rejected", minAPIKeyLength-1)
}

func TestGetArgs_MinLengthAPIKeyAccepted(t *testing.T) {
	key := fillerKey(minAPIKeyLength)
	cfg, err := GetArgs(withKey("-api-key", key))
	require.NoError(t, err)
	require.Equal(t, key, cfg.APIKey)
}
