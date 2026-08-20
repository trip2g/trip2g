package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/cmd/codellm/appconfig"
	"trip2g/cmd/codellm/internal/coderun"
)

// TestStartMetricsServer_DisabledByEmptyAddr asserts an empty --metrics-addr
// starts nothing and yields the nil sink every call site tolerates.
func TestStartMetricsServer_DisabledByEmptyAddr(t *testing.T) {
	cfg := appconfig.DefaultConfig()
	cfg.MetricsAddr = ""

	require.Nil(t, startMetricsServer(&cfg))
}

// TestStartMetricsServer_PublishesExecutionPosture asserts an enabled listener
// comes up with the sandbox posture already published, so an operator can read
// it from a scrape instead of from the box.
func TestStartMetricsServer_PublishesExecutionPosture(t *testing.T) {
	cfg := appconfig.DefaultConfig()
	cfg.MetricsAddr = freeAddr(t)
	cfg.Sandbox = coderun.SandboxNative
	cfg.SandboxNetwork = true
	cfg.AllowedPrograms = []string{"python", "bash"}

	m := startMetricsServer(&cfg)
	require.NotNil(t, m)

	rec := httptest.NewRecorder()
	m.Handler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(),
		`codellm_config_info{allowed_programs="python,bash",sandbox_mode="native",sandbox_network="true"} 1`)
}

// freeAddr returns a loopback address that was free a moment ago. The listener
// under test binds it itself, so it is released first.
func freeAddr(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	addr := srv.Listener.Addr().String()
	srv.Close()
	return addr
}
