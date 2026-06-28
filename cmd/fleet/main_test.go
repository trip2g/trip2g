package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"trip2g/internal/fleet"
)

func validConfig() fleet.Config {
	return fleet.Config{
		CallbackURL:  "https://fleet.example.com",
		AdminAPIKey:  "admin-key",
		FleetSecret:  "secret",
		LLMAPIKey:    "llm-key",
		FleetID:      "fleet1",
		DefaultModel: "gpt-4o-mini",
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
			name:    "missing_admin_api_key",
			mutate:  func(c *fleet.Config) { c.AdminAPIKey = "" },
			wantErr: "AdminAPIKey",
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
				require.True(t, strings.Contains(err.Error(), tc.wantErr),
					"expected error to mention %q, got: %v", tc.wantErr, err)
			}
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
