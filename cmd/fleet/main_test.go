package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"trip2g/internal/fleet"
	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
)

func validConfig() fleet.Config {
	return fleet.Config{
		CallbackURL:  "https://fleet.example.com",
		JWTSecret:    "jwt-secret",
		AdminEmail:   "fleet@local",
		FleetSecret:  "secret",
		LLMAPIKey:    "llm-key",
		FleetID:      "fleet1",
		DefaultModel: "gpt-4o-mini",
		TokenCeiling: 100000,
		StepCeiling:  25,
		OfferedTools: []string{"search", "read_note"},
	}
}

// TestValidateConfig_RejectsMissingFields ensures that each required field
// triggers a non-nil error when absent, so the daemon fails fast before
// running the first syncOnce with broken config.
func TestValidateConfig_RejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*fleet.Config)
		wantErr string
	}{
		{
			name:    "missing_callback_url",
			mutate:  func(c *fleet.Config) { c.CallbackURL = "" },
			wantErr: "CallbackURL",
		},
		{
			name:    "missing_jwt_secret",
			mutate:  func(c *fleet.Config) { c.JWTSecret = "" },
			wantErr: "JWTSecret",
		},
		{
			name:    "missing_fleet_secret",
			mutate:  func(c *fleet.Config) { c.FleetSecret = "" },
			wantErr: "FleetSecret",
		},
		{
			name:    "missing_llm_api_key",
			mutate:  func(c *fleet.Config) { c.LLMAPIKey = "" },
			wantErr: "LLMAPIKey",
		},
		{
			name:    "token_ceiling_zero",
			mutate:  func(c *fleet.Config) { c.TokenCeiling = 0 },
			wantErr: "TokenCeiling",
		},
		{
			name:    "step_ceiling_zero",
			mutate:  func(c *fleet.Config) { c.StepCeiling = 0 },
			wantErr: "StepCeiling",
		},
		{
			name:    "offered_tools_empty",
			mutate:  func(c *fleet.Config) { c.OfferedTools = nil },
			wantErr: "OfferedTools",
		},
		{
			name:    "sandbox_invalid_value",
			mutate:  func(c *fleet.Config) { c.Sandbox = "chroot" },
			wantErr: "Sandbox",
		},
		{
			name:    "sandbox_off_ok",
			mutate:  func(c *fleet.Config) { c.Sandbox = "off" },
			wantErr: "",
		},
		{
			name:    "debug_listen_non_loopback",
			mutate:  func(c *fleet.Config) { c.DebugListenAddr = "0.0.0.0:9091" },
			wantErr: "loopback",
		},
		{
			name:    "debug_listen_loopback_ok",
			mutate:  func(c *fleet.Config) { c.DebugListenAddr = "127.0.0.1:9091" },
			wantErr: "",
		},
		{
			name:    "all_fields_present",
			mutate:  func(c *fleet.Config) {},
			wantErr: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig()
			tc.mutate(&cfg)
			err := validateConfig(cfg)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr,
					"expected error to mention %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// TestReportRoles_FormatsAndFlags verifies the --dry-run resolver/printer:
// it renders each role's resolved config (trigger_on -> on* flags, defaulted
// model and timeout) and FLAGS any role that fails Validate. This tests the
// formatting/flagging logic, not the live admin-lane connect.
func TestReportRoles_FormatsAndFlags(t *testing.T) {
	roles := []fleet.Role{
		{
			NotePath:       "roles/good.md",
			Mode:           "change",
			TriggerOn:      []string{"update"},
			TriggerInclude: []string{"boards/**"},
			ReadPatterns:   []string{"boards/**"},
			WritePatterns:  []string{"boards/**"},
			Tools:          []string{"search", "patch_note"},
		},
		{
			NotePath: "roles/bad.md",
			Mode:     "change",
			// empty trigger_on -> fires on nothing -> must be FLAGGED
			TriggerInclude: []string{"boards/**"},
		},
	}
	out := reportRoles(roles, []string{"search", "read_note", "patch_note"}, "gpt-4o-mini")

	// Resolved config for the good role.
	require.Contains(t, out, "roles/good.md")
	require.Contains(t, out, "onCreate=false onUpdate=true onRemove=false")
	require.Contains(t, out, "gpt-4o-mini (default)") // model omitted -> default shown
	require.Contains(t, out, "300 (default)")         // timeout unset -> resolved default
	require.Contains(t, out, "STATUS: OK")

	// The misconfigured role is flagged with its failure reason.
	require.Contains(t, out, "roles/bad.md")
	require.Contains(t, out, "FLAGGED")
	require.Contains(t, out, "trigger_on")
}

// TestReportRoles_Empty reports clearly when no roles were discovered.
func TestReportRoles_Empty(t *testing.T) {
	out := reportRoles(nil, []string{"search"}, "gpt-4o-mini")
	require.Contains(t, out, "no roles discovered")
}

// TestParseFrontmatter verifies that parseFrontmatter splits a role-note into
// its flat meta map and body, handling both present and absent frontmatter.
func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMeta map[string]string
		wantBody string
		wantErr  bool
	}{
		{
			name:  "full_frontmatter",
			input: "---\nmodel: gpt-4o-mini\ntools: [search, read_note]\nmode: change\n---\nBody text here.\n",
			wantMeta: map[string]string{
				"model": "gpt-4o-mini",
				"tools": "[search, read_note]",
				"mode":  "change",
			},
			wantBody: "Body text here.\n",
		},
		{
			name:     "no_frontmatter",
			input:    "Just a body with no frontmatter.\n",
			wantMeta: map[string]string{},
			wantBody: "Just a body with no frontmatter.\n",
		},
		{
			name:    "unclosed_frontmatter",
			input:   "---\nmodel: gpt-4o\n",
			wantErr: true,
		},
		{
			name:     "empty_body_after_frontmatter",
			input:    "---\nmode: change\n---\n",
			wantMeta: map[string]string{"mode": "change"},
			wantBody: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta, body, err := parseFrontmatter(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantMeta, meta)
			require.Equal(t, tc.wantBody, body)
		})
	}
}

// TestTrailingSlashNormalization verifies that normalizeCallbackURL (called by
// run() before validateConfig) strips trailing slashes from CallbackURL.
func TestTrailingSlashNormalization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://fleet.example.com/", "https://fleet.example.com"},
		{"https://fleet.example.com///", "https://fleet.example.com"},
		{"https://fleet.example.com", "https://fleet.example.com"},
		{"http://localhost:9090/", "http://localhost:9090"},
	}
	for _, tc := range tests {
		got := normalizeCallbackURL(tc.input)
		require.Equal(t, tc.want, got, "input: %s", tc.input)
	}
}

// newIdleTestServer starts a real HTTP server on a random loopback port using
// the provided handler. The caller must call srv.Shutdown or srv.Close when done.
func newIdleTestServer(t *testing.T, handler http.Handler) (*http.Server, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &http.Server{Handler: handler}
	go func() { _ = srv.Serve(ln) }()
	return srv, ln.Addr().String()
}

// waitShutdownDraining blocks until srv.Shutdown has closed the listener (a new
// dial is refused), proving Shutdown has been entered and is in its drain loop
// waiting for the in-flight handler. Releasing the handler only after this makes
// the subsequent 200 a genuine drain guarantee: a force-close (srv.Close) would
// instead drop the in-flight connection and the request would not get 200.
func waitShutdownDraining(t *testing.T, addr string) {
	t.Helper()
	require.Eventually(t, func() bool {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return true // refused → listener closed → Shutdown is draining
		}
		_ = c.Close()
		return false
	}, 3*time.Second, 5*time.Millisecond)
}

// TestGracefulShutdown_DrainsInFlightAndDeregisters asserts that gracefulShutdown
// waits for an active in-flight request to complete (drain) before returning, and
// then calls the deregister function exactly once when keepWebhooks is false.
func TestGracefulShutdown_DrainsInFlightAndDeregisters(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusOK)
	})

	srv, addr := newIdleTestServer(t, mux)

	// Fire a request in a background goroutine; capture the status code.
	var respCode int
	reqDone := make(chan struct{})
	go func() {
		defer close(reqDone)
		resp, httpErr := http.Get("http://" + addr + "/")
		if httpErr == nil {
			respCode = resp.StatusCode
			resp.Body.Close()
		}
	}()

	<-entered // handler is now blocking inside the server

	var deregCount atomic.Int32
	stubDereg := func(_ context.Context) error {
		if n := deregCount.Add(1); n < 0 {
			return fmt.Errorf("unexpected: deregCount overflowed to %d", n)
		}
		return nil
	}

	shutdownErr := make(chan error, 1)
	go func() {
		shutdownErr <- gracefulShutdown(&logger.DummyLogger{}, srv, stubDereg, false, 5*time.Second)
	}()

	// Block until Shutdown has closed the listener and is draining the active
	// request, so releasing the handler now proves Shutdown WAITED for it.
	waitShutdownDraining(t, addr)
	close(release) // unblock the handler → it writes 200

	<-reqDone
	err := <-shutdownErr

	require.NoError(t, err, "gracefulShutdown must return nil when drain succeeds")
	require.Equal(t, http.StatusOK, respCode, "in-flight request must drain and complete with 200")
	require.Equal(t, int32(1), deregCount.Load(), "deregister must be called exactly once")
}

// TestGracefulShutdown_KeepWebhooksSkipsDeregister asserts that when
// keepWebhooks is true the deregister function is never called, but the
// in-flight request is still drained normally (200 response).
func TestGracefulShutdown_KeepWebhooksSkipsDeregister(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusOK)
	})

	srv, addr := newIdleTestServer(t, mux)

	var respCode int
	reqDone := make(chan struct{})
	go func() {
		defer close(reqDone)
		resp, httpErr := http.Get("http://" + addr + "/")
		if httpErr == nil {
			respCode = resp.StatusCode
			resp.Body.Close()
		}
	}()

	<-entered

	var deregCount atomic.Int32
	stubDereg := func(_ context.Context) error {
		if n := deregCount.Add(1); n < 0 {
			return fmt.Errorf("unexpected: deregCount overflowed to %d", n)
		}
		return nil
	}

	shutdownErr := make(chan error, 1)
	go func() {
		shutdownErr <- gracefulShutdown(&logger.DummyLogger{}, srv, stubDereg, true, 5*time.Second)
	}()

	// Same drain-ordering guarantee as the deregister test (see waitShutdownDraining).
	waitShutdownDraining(t, addr)
	close(release)

	<-reqDone
	err := <-shutdownErr

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, respCode, "in-flight request must still drain with keepWebhooks=true")
	require.Equal(t, int32(0), deregCount.Load(), "deregister must NOT be called when keepWebhooks=true")
}

// TestGracefulShutdown_DeregisterNilSafe asserts that passing a nil deregister
// function does not panic and returns without error on an idle server.
func TestGracefulShutdown_DeregisterNilSafe(t *testing.T) {
	srv, _ := newIdleTestServer(t, http.NewServeMux())
	err := gracefulShutdown(&logger.DummyLogger{}, srv, nil, false, time.Second)
	require.NoError(t, err)
}

// TestEffectiveGrace_FloorsNonPositive guards the floor that protects the drain
// path: a zero/negative grace must become 30s, never an already-expired context
// (which would make srv.Shutdown force-close in-flight runs). This floor is the
// only thing keeping --shutdown-grace-seconds=0 (incl. via env) from skipping
// the drain entirely, so it is tested directly.
func TestEffectiveGrace_FloorsNonPositive(t *testing.T) {
	require.Equal(t, 30*time.Second, effectiveGrace(0), "0 must floor to 30s")
	require.Equal(t, 30*time.Second, effectiveGrace(-5), "negative must floor to 30s")
	require.Equal(t, 45*time.Second, effectiveGrace(45), "positive must pass through")
}

// TestGracefulShutdown_DrainFailureReturnsError asserts that a drain that
// cannot finish within the grace window surfaces a non-nil error (run()
// propagates it so a supervisor can tell a failed drain from a clean exit).
func TestGracefulShutdown_DrainFailureReturnsError(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	defer close(release)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusOK)
	})

	srv, addr := newIdleTestServer(t, mux)

	go func() {
		resp, httpErr := http.Get("http://" + addr + "/")
		if httpErr == nil {
			resp.Body.Close()
		}
	}()
	<-entered // handler now stuck past any 50ms grace

	err := gracefulShutdown(&logger.DummyLogger{}, srv, nil, false, 50*time.Millisecond)
	require.Error(t, err, "drain incomplete within grace must be reported, not swallowed")
}
