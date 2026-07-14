package appconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.Equal(t, DefaultAgentsFolder, cfg.AgentsFolder)
	require.Equal(t, DefaultListenAddr, cfg.ListenAddr)
	require.Equal(t, DefaultTrip2gBaseURL, cfg.Trip2gBaseURL)
	require.Equal(t, DefaultModel, cfg.DefaultModel)
	require.Equal(t, DefaultGraphQLAddr, cfg.GraphQLAddr) // loopback by default
	require.Empty(t, cfg.CodellmBaseURL)

	cfg.Prepare()
	require.Equal(t, []string{"search", "read_note", "patch_note", "write_note"}, cfg.OfferedTools)
}

func TestGetDefaults(t *testing.T) {
	cfg, err := Get(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, DefaultAgentsFolder, cfg.AgentsFolder)
	require.Equal(t, []string{"search", "read_note", "patch_note", "write_note"}, cfg.OfferedTools)
}

func TestGetFlagsOverrideDefaults(t *testing.T) {
	cfg, err := Get(context.Background(), []string{
		"-agents-folder", "custom-roles/",
		"-graphql-addr", "127.0.0.1:9093",
		"-default-model", "gpt-5",
		"-offered-tools", "search, read_note",
		"-codellm-base-url", "http://localhost:9094",
	})
	require.NoError(t, err)
	require.Equal(t, "custom-roles/", cfg.AgentsFolder)
	require.Equal(t, "127.0.0.1:9093", cfg.GraphQLAddr)
	require.Equal(t, "gpt-5", cfg.DefaultModel)
	require.Equal(t, []string{"search", "read_note"}, cfg.OfferedTools)
	require.Equal(t, "http://localhost:9094", cfg.CodellmBaseURL)
}

func TestGetEnvOverridesDefaults(t *testing.T) {
	t.Setenv("TRIP2G_FLEET_AGENTS_FOLDER", "env-roles/")
	t.Setenv("TRIP2G_FLEET_OFFERED_TOOLS", "read_note")

	cfg, err := Get(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, "env-roles/", cfg.AgentsFolder)
	require.Equal(t, []string{"read_note"}, cfg.OfferedTools)
}

func TestGetFlagOverridesEnv(t *testing.T) {
	t.Setenv("TRIP2G_FLEET_AGENTS_FOLDER", "env-roles/")

	cfg, err := Get(context.Background(), []string{"-agents-folder", "flag-roles/"})
	require.NoError(t, err)
	require.Equal(t, "flag-roles/", cfg.AgentsFolder)
}

func TestValidateRejectsEmptyRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"empty_agents_folder", func(c *Config) { c.AgentsFolder = "" }, "AgentsFolder"},
		{"empty_offered_tools", func(c *Config) { c.OfferedTools = nil }, "OfferedTools"},
		{"all_present", func(c *Config) {}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Prepare()
			tt.mutate(cfg)

			err := cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single", "search", []string{"search"}},
		{"multiple", "search,read_note,write_note", []string{"search", "read_note", "write_note"}},
		{"whitespace trimmed", " search , read_note ", []string{"search", "read_note"}},
		{"blank entries dropped", "search,,read_note", []string{"search", "read_note"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, SplitCSV(tt.in))
		})
	}
}
