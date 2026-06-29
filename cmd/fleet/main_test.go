package main

import (
	"testing"

	"trip2g/internal/fleet"

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
