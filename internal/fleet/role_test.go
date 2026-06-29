package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func meta(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func TestParseRole_FlatFrontmatter(t *testing.T) {
	r, err := ParseRole("roles/triage.md", "Triage the board.", meta(
		"model", "gpt-4o-mini",
		"tools", "[search, read_note, patch_note]",
		"read_patterns", `["boards/**","roles/**"]`,
		"write_patterns", `["boards/**"]`,
		"max_tokens", "4000",
		"max_steps", "6",
		"mode", "change",
		"trigger_include", `["boards/sprint.md"]`,
		"trigger_on", "[update]",
		"attach_notes", `["boards/**","roles/**"]`,
		"max_depth", "1",
		"concurrency", "skip",
	))
	require.NoError(t, err)
	require.Equal(t, "roles/triage.md", r.NotePath)
	require.Equal(t, "Triage the board.", r.Body)
	require.Equal(t, "gpt-4o-mini", r.Model)
	require.Equal(t, []string{"search", "read_note", "patch_note"}, r.Tools)
	require.Equal(t, []string{"boards/**", "roles/**"}, r.ReadPatterns)
	require.Equal(t, []string{"boards/**"}, r.WritePatterns)
	require.Equal(t, 4000, r.MaxTokens)
	require.Equal(t, 6, r.MaxSteps)
	require.Equal(t, "change", r.Mode)
	require.Equal(t, []string{"boards/sprint.md"}, r.TriggerInclude)
	require.Equal(t, []string{"update"}, r.TriggerOn)
	require.Equal(t, []string{"boards/**", "roles/**"}, r.AttachNotes)
	require.Equal(t, 1, r.MaxDepth)
	require.Equal(t, "skip", r.Concurrency)
}

func TestParseRole_ForEach(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"changed_files", "changed_files"},
		{"attached_notes", "attached_notes"},
		{"", ""}, // absent = legacy single run
	}
	for _, tc := range cases {
		t.Run("for_each="+tc.raw, func(t *testing.T) {
			r, err := ParseRole("roles/triage.md", "body", meta("mode", "change", "for_each", tc.raw))
			require.NoError(t, err)
			require.Equal(t, tc.want, r.ForEach)
		})
	}
}

func TestRoleValidate_ForEachEnum(t *testing.T) {
	require.NoError(t, Role{Mode: "change", ForEach: ""}.Validate(nil))
	require.NoError(t, Role{Mode: "change", ForEach: "changed_files"}.Validate(nil))
	require.NoError(t, Role{Mode: "change", ForEach: "attached_notes"}.Validate(nil))

	err := Role{Mode: "change", ForEach: "bogus"}.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "for_each")
}

func TestRoleValidate_ToolsSubset(t *testing.T) {
	r := Role{Mode: "change", Tools: []string{"search", "patch_note"}}
	require.NoError(t, r.Validate([]string{"search", "read_note", "patch_note", "write_note"}))

	r.Tools = []string{"search", "shell"}
	err := r.Validate([]string{"search", "read_note", "patch_note"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "shell")
}

func TestRoleValidate_RequiresMode(t *testing.T) {
	err := Role{}.Validate([]string{"search"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mode")
}

func TestRoleValidate_DefaultConcurrencyAllowed(t *testing.T) {
	require.NoError(t, Role{Mode: "change", Concurrency: ""}.Validate(nil))
	require.Error(t, Role{Mode: "change", Concurrency: "bogus"}.Validate(nil))
}

// TestRoleValidate_CronModeRejected is a regression test for F6: cron-mode
// roles must fail fast at discovery because cron reconcile is not yet
// implemented. mode:change must still pass.
func TestRoleValidate_CronModeRejected(t *testing.T) {
	cases := []struct {
		mode    string
		wantErr bool
	}{
		{"change", false},
		{"cron", true},
		{"both", true},
	}
	for _, tc := range cases {
		t.Run("mode="+tc.mode, func(t *testing.T) {
			err := Role{Mode: tc.mode}.Validate(nil)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "not yet supported")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
