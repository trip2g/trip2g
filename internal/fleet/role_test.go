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

// validChangeRole is a minimal change-mode role that passes Validate: it has a
// non-empty trigger_on and trigger_include (both now required for change mode).
// Tests tweak one field to isolate the rejection under test.
func validChangeRole() Role {
	return Role{
		NotePath:       "roles/a.md",
		Mode:           "change",
		TriggerOn:      []string{"update"},
		TriggerInclude: []string{"boards/**"},
	}
}

func TestParseRole_TimeoutSeconds(t *testing.T) {
	r, err := ParseRole("roles/a.md", "body", meta("mode", "change", "timeout_seconds", "120"))
	require.NoError(t, err)
	require.Equal(t, 120, r.TimeoutSeconds)
	require.Equal(t, 120, r.EffectiveTimeoutSeconds())

	// Unset stays 0 raw and resolves to the generous default.
	r2, err := ParseRole("roles/a.md", "body", meta("mode", "change"))
	require.NoError(t, err)
	require.Equal(t, 0, r2.TimeoutSeconds)
	require.Equal(t, defaultTimeoutSeconds, r2.EffectiveTimeoutSeconds())

	// Non-numeric is a parse error.
	_, err = ParseRole("roles/a.md", "body", meta("mode", "change", "timeout_seconds", "soon"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "timeout_seconds")
}

func TestRoleValidate_TimeoutSecondsNonNegative(t *testing.T) {
	r := validChangeRole()
	r.TimeoutSeconds = 0 // unset -> allowed (defaults later)
	require.NoError(t, r.Validate(nil))
	r.TimeoutSeconds = 300
	require.NoError(t, r.Validate(nil))

	r.TimeoutSeconds = -1
	err := r.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timeout_seconds")
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
	for _, fe := range []string{"", "changed_files", "attached_notes"} {
		r := validChangeRole()
		r.ForEach = fe
		require.NoError(t, r.Validate(nil))
	}

	bad := validChangeRole()
	bad.ForEach = "bogus"
	err := bad.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "for_each")
}

func TestRoleValidate_ToolsSubset(t *testing.T) {
	r := validChangeRole()
	r.Tools = []string{"search", "patch_note"}
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
	ok := validChangeRole()
	ok.Concurrency = ""
	require.NoError(t, ok.Validate(nil))

	bad := validChangeRole()
	bad.Concurrency = "bogus"
	require.Error(t, bad.Validate(nil))
}

// TestRoleValidate_CronModeAccepted verifies that cron and both modes are now
// accepted by Validate. Previously they were rejected as "not yet supported";
// the cron-mode implementation removes that restriction.
func TestRoleValidate_CronModeAccepted(t *testing.T) {
	// A minimal valid cron-only role (no trigger_on/include needed).
	cronRole := func() Role {
		return Role{
			NotePath:     "roles/a.md",
			Mode:         "cron",
			CronSchedule: "*/5 * * * *",
		}
	}
	// A minimal valid both-mode role needs change fields + cron_schedule.
	bothRole := func() Role {
		return Role{
			NotePath:       "roles/a.md",
			Mode:           "both",
			CronSchedule:   "0 * * * *",
			TriggerOn:      []string{"update"},
			TriggerInclude: []string{"boards/**"},
		}
	}

	t.Run("cron_with_schedule_passes", func(t *testing.T) {
		require.NoError(t, cronRole().Validate(nil))
	})
	t.Run("cron_without_schedule_fails", func(t *testing.T) {
		r := cronRole()
		r.CronSchedule = ""
		err := r.Validate(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cron_schedule")
	})
	t.Run("both_with_all_required_fields_passes", func(t *testing.T) {
		require.NoError(t, bothRole().Validate(nil))
	})
	t.Run("both_without_schedule_fails", func(t *testing.T) {
		r := bothRole()
		r.CronSchedule = ""
		err := r.Validate(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cron_schedule")
	})
	t.Run("both_without_trigger_on_fails", func(t *testing.T) {
		r := bothRole()
		r.TriggerOn = nil
		err := r.Validate(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "trigger_on")
	})
	t.Run("both_without_trigger_include_fails", func(t *testing.T) {
		r := bothRole()
		r.TriggerInclude = nil
		err := r.Validate(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "trigger_include")
	})
	t.Run("change_still_passes", func(t *testing.T) {
		require.NoError(t, validChangeRole().Validate(nil))
	})
}

// TestRoleValidate_RejectsEmptyTriggerOn: a change-mode role with no trigger_on
// registers a webhook that fires on no events (the silent-misconfig footgun).
func TestRoleValidate_RejectsEmptyTriggerOn(t *testing.T) {
	r := validChangeRole()
	r.TriggerOn = nil
	err := r.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "trigger_on")
}

// TestRoleValidate_RejectsEmptyTriggerInclude: an empty trigger_include matches
// no paths, so the webhook never fires.
func TestRoleValidate_RejectsEmptyTriggerInclude(t *testing.T) {
	r := validChangeRole()
	r.TriggerInclude = nil
	err := r.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "trigger_include")
}

// TestRoleValidate_RejectsChangeFileWithoutForEach: a body that references
// change_file but isn't fanned out per changed file renders against nil and
// every delivery fails — the documented footgun. for_each:changed_files fixes it.
func TestRoleValidate_RejectsChangeFileWithoutForEach(t *testing.T) {
	r := validChangeRole()
	r.Body = "Handle {{ change_file.Path }}."
	r.ForEach = "" // not changed_files
	err := r.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "change_file")

	// Correct usage (for_each:changed_files) passes.
	r.ForEach = "changed_files"
	require.NoError(t, r.Validate(nil))

	// Bodies that only mention changed_files (plural) are not flagged.
	ok := validChangeRole()
	ok.Body = "Files:{{ range changed_files }} {{ .Path }}{{ end }}."
	require.NoError(t, ok.Validate(nil))
}

// validCodeRole is a minimal code-executor role that passes Validate.
func validCodeRole() Role {
	return Role{
		NotePath:       "roles/code.md",
		Mode:           "change",
		Executor:       "code",
		TriggerOn:      []string{"update"},
		TriggerInclude: []string{"transcripts/**"},
		WritePatterns:  []string{"notes/**"},
		Body:           "```bash\necho '{\"changes\":[],\"answer\":\"ok\"}'\n```",
	}
}

func TestParseRole_Executor(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"code", "code"},
		{"llm", "llm"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run("executor="+tc.raw, func(t *testing.T) {
			r, err := ParseRole("roles/a.md", "body", meta("mode", "change", "executor", tc.raw))
			require.NoError(t, err)
			require.Equal(t, tc.want, r.Executor)
		})
	}
}

func TestRoleValidate_ExecutorDefault(t *testing.T) {
	// Omitted executor (empty) is treated as llm — backward-compatible.
	r := validChangeRole()
	r.Executor = ""
	require.NoError(t, r.Validate(nil))

	r.Executor = "llm"
	require.NoError(t, r.Validate(nil))
}

func TestRoleValidate_ExecutorInvalidValue(t *testing.T) {
	r := validChangeRole()
	r.Executor = "shell"
	err := r.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "executor must be llm|code")
}

func TestRoleValidate_CodeExecutorAllowsEmptyWritePatterns(t *testing.T) {
	r := validCodeRole()
	r.WritePatterns = nil
	require.NoError(t, r.Validate(nil))
}

func TestRoleValidate_CodeExecutorRequiresFencedBlock(t *testing.T) {
	r := validCodeRole()
	r.Body = "no code block here"
	err := r.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fenced code block")
}

func TestRoleValidate_CodeExecutorUnknownFenceLang(t *testing.T) {
	r := validCodeRole()
	r.Body = "```haskell\nputs 'hi'\n```"
	err := r.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fence language")
	require.Contains(t, err.Error(), "haskell")
}

func TestRoleValidate_CodeExecutorSupportedFenceLangs(t *testing.T) {
	langs := []string{"python", "py", "bash", "sh", "js", "javascript", "node", "ruby", "rb", "php", "pl", "perl"}
	for _, lang := range langs {
		t.Run("lang="+lang, func(t *testing.T) {
			r := validCodeRole()
			r.Body = "```" + lang + "\necho 'hi'\n```"
			require.NoError(t, r.Validate(nil))
		})
	}
}

// captureLogger records Warn calls for assertions.
type captureLogger struct{ warns []string }

func (c *captureLogger) Info(msg string, _ ...interface{})  {}
func (c *captureLogger) Error(msg string, _ ...interface{}) {}
func (c *captureLogger) Debug(msg string, _ ...interface{}) {}
func (c *captureLogger) Warn(msg string, _ ...interface{})  { c.warns = append(c.warns, msg) }

func TestRole_WarnIfWriteScopeMisconfigured(t *testing.T) {
	tests := []struct {
		name     string
		tools    []string
		patterns []string
		wantWarn bool
	}{
		{"write tool + empty patterns warns", []string{"search", "write_note"}, nil, true},
		{"patch tool + empty patterns warns", []string{"patch_note"}, nil, true},
		{"write tool + non-empty patterns silent", []string{"write_note"}, []string{"concepts/**"}, false},
		{"read-only role silent", []string{"search", "read_note"}, nil, false},
		{"no tools silent", nil, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Role{NotePath: "roles/x.md", Tools: tc.tools, WritePatterns: tc.patterns}
			lg := &captureLogger{}
			r.WarnIfWriteScopeMisconfigured(lg)
			if tc.wantWarn {
				require.Len(t, lg.warns, 1)
				require.Contains(t, lg.warns[0], "write_patterns is empty")
				require.Contains(t, lg.warns[0], "roles/x.md")
			} else {
				require.Empty(t, lg.warns)
			}
		})
	}
}

func TestParseRole_EnvPassthrough(t *testing.T) {
	r, err := ParseRole("roles/a.md", "```bash\necho hi\n```", meta(
		"mode", "change",
		"executor", "code",
		"write_patterns", `["notes/**"]`,
		"env_passthrough", `["MY_TOKEN", "API_KEY"]`,
		"env_prefix", `["KRISP_"]`,
	))
	require.NoError(t, err)
	require.Equal(t, []string{"MY_TOKEN", "API_KEY"}, r.EnvPassthrough)
	require.Equal(t, []string{"KRISP_"}, r.EnvPrefix)
}

func TestRoleValidate_CodeExecutorEmptyEnvPrefixRejected(t *testing.T) {
	r := validCodeRole()
	r.EnvPrefix = []string{""} // empty prefix matches all vars → rejected
	err := r.Validate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "env_prefix")
	require.Contains(t, err.Error(), "empty entry")
}

func TestRoleValidate_CodeExecutorValidEnvPassthrough(t *testing.T) {
	r := validCodeRole()
	r.EnvPassthrough = []string{"MY_TOKEN"}
	r.EnvPrefix = []string{"KRISP_"}
	require.NoError(t, r.Validate(nil))
}
