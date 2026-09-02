package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeclaresRole(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"plain note", "# Title\n\nbody\n", false},
		{"frontmatter without fleet_id", "---\ntitle: x\n---\n\nbody\n", false},
		{"frontmatter with fleet_id", "---\nfleet_id: codellm\nmode: cron\n---\n\nbody\n", true},
		{"empty fleet_id still counts", "---\nfleet_id: \"\"\n---\n", true},
		{"fleet_id only in body", "# doc\n\nset fleet_id: codellm to assign it\n", false},
		{"fenced block mentioning fleet_id", "---\ntitle: x\n---\n\n```yaml\nfleet_id: codellm\n```\n", false},
		{"frontmatter not at start", "intro\n\n---\nfleet_id: codellm\n---\n", false},
		{"leading blank lines then frontmatter", "\n\n---\nfleet_id: codellm\n---\n", true},
		{"crlf line endings", "---\r\nfleet_id: codellm\r\n---\r\nbody\r\n", true},
		// goldmark closes an open block at EOF and parses what it collected, so
		// an unterminated frontmatter still makes the note a role.
		{"unterminated frontmatter", "---\nfleet_id: codellm\n", true},
		{"four-dash fence", "----\nfleet_id: codellm\n----\n", true},
		{"mismatched fence lengths", "---\nfleet_id: codellm\n--------\n", true},
		{"duplicate fleet_id keys", "---\nfleet_id: a\nfleet_id: b\n---\n", true},
		{"list body is not a fence", "- item\n- other\n", false},
		{"empty content", "", false},
		{"nested fleet_id is not a role marker", "---\nmeta:\n  fleet_id: codellm\n---\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, declaresRole(tc.content))
		})
	}
}

func TestApplyPatchPreview(t *testing.T) {
	cases := []struct {
		name          string
		content       string
		find, replace string
		want          string
		ok            bool
	}{
		{"single occurrence", "a b c", "b", "X", "a X c", true},
		{"first and only", "---\ntitle: x\n---\n", "title: x", "fleet_id: codellm", "---\nfleet_id: codellm\n---\n", true},
		{"missing find leaves content", "a b c", "zzz", "X", "a b c", false},
		{"ambiguous find leaves content", "a b a", "a", "X", "a b a", false},
		{"empty find leaves content", "a b c", "", "X", "a b c", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := applyPatchPreview(tc.content, tc.find, tc.replace)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.ok, ok)
		})
	}
}

func TestScopedKB_WriteDeniesRoleAuthoring(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, nil, []string{"**"})

	err := scoped.Write(context.Background(), "notes/evil.md", "---\nfleet_id: codellm\n---\nbody\n")

	require.ErrorIs(t, err, ErrRoleAuthoringDenied)
	_, readErr := kb.Read(context.Background(), "notes/evil.md")
	require.Error(t, readErr, "denied write must not reach the KB")
}

func TestScopedKB_WriteAllowsPlainNote(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, nil, []string{"**"})

	require.NoError(t, scoped.Write(context.Background(), "notes/ok.md", "---\ntitle: x\n---\nbody\n"))
}

func TestScopedKB_WriteAllowsRoleAuthoringWhenPermitted(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, nil, []string{"**"})
	scoped.allowRoleAuthoring = true

	require.NoError(t, scoped.Write(context.Background(), "roles/new.md", "---\nfleet_id: codellm\n---\nbody\n"))
}

// The guard is about role authorship, not folder layout: a role note may live
// anywhere, so the check keys on the fleet_id marker regardless of path.
func TestScopedKB_WriteDeniesRoleAuthoringOutsideRolesFolder(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, nil, []string{"**"})

	err := scoped.Write(context.Background(), "transcripts/2026/x.md", "---\nfleet_id: codellm\n---\n")

	require.ErrorIs(t, err, ErrRoleAuthoringDenied)
}

func TestScopedKB_PatchDeniesTurningNoteIntoRole(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/plain.md": "---\ntitle: x\n---\nbody\n"})
	scoped := NewScopedKB(kb, nil, []string{"**"})

	err := scoped.Patch(context.Background(), "notes/plain.md", "title: x", "fleet_id: codellm")

	require.ErrorIs(t, err, ErrRoleAuthoringDenied)
	got, _ := kb.Read(context.Background(), "notes/plain.md")
	require.Equal(t, "---\ntitle: x\n---\nbody\n", got, "denied patch must not reach the KB")
}

// The substring-on-replace shortcut misses this: retagging an existing role
// note changes only the value, so "fleet_id" never appears in replace.
func TestScopedKB_PatchDeniesRetaggingExistingRole(t *testing.T) {
	kb := newMemKB(map[string]string{"roles/other.md": "---\nfleet_id: otherfleet\n---\nbody\n"})
	scoped := NewScopedKB(kb, nil, []string{"**"})

	err := scoped.Patch(context.Background(), "roles/other.md", "otherfleet", "codellm")

	require.ErrorIs(t, err, ErrRoleAuthoringDenied)
}

func TestScopedKB_PatchDeniesEditingRoleBody(t *testing.T) {
	kb := newMemKB(map[string]string{"roles/existing.md": "---\nfleet_id: codellm\n---\nold\n"})
	scoped := NewScopedKB(kb, nil, []string{"**"})

	err := scoped.Patch(context.Background(), "roles/existing.md", "old", "new")

	require.ErrorIs(t, err, ErrRoleAuthoringDenied)
}

func TestScopedKB_PatchAllowsPlainNote(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/plain.md": "hello world\n"})
	scoped := NewScopedKB(kb, nil, []string{"**"})

	require.NoError(t, scoped.Patch(context.Background(), "notes/plain.md", "world", "there"))
}

// A find that does not match is trip2g's business (it reports PatchNotFound).
// The guard must not turn that into a role-authoring denial.
func TestScopedKB_PatchWithMissingFindIsNotARoleDenial(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/plain.md": "hello world\n"})
	scoped := NewScopedKB(kb, nil, []string{"**"})

	err := scoped.Patch(context.Background(), "notes/plain.md", "absent", "x")

	require.NotErrorIs(t, err, ErrRoleAuthoringDenied)
	require.NotErrorIs(t, err, ErrRoleGuardUnverifiable)
}

// The guard reads through the underlying KB, not the scoped one: a role with
// write scope but no read scope must not thereby escape verification.
func TestScopedKB_PatchGuardReadsOutsideReadScope(t *testing.T) {
	kb := newMemKB(map[string]string{"roles/existing.md": "---\nfleet_id: codellm\n---\nold\n"})
	scoped := NewScopedKB(kb, []string{"nothing/**"}, []string{"**"})

	err := scoped.Patch(context.Background(), "roles/existing.md", "old", "new")

	require.ErrorIs(t, err, ErrRoleAuthoringDenied)
}

func TestScopedKB_PatchFailsClosedWhenReadFails(t *testing.T) {
	kb := &failingReadKB{KB: newMemKB(map[string]string{"notes/plain.md": "hello\n"})}
	scoped := NewScopedKB(kb, nil, []string{"**"})

	err := scoped.Patch(context.Background(), "notes/plain.md", "hello", "bye")

	require.ErrorIs(t, err, ErrRoleGuardUnverifiable)
	require.NotErrorIs(t, err, ErrRoleAuthoringDenied,
		"an unverifiable patch is not an accusation of role authoring")
	require.False(t, kb.patched, "unverifiable patch must not reach the KB")
}

func TestScopedKB_PatchSkipsGuardReadWhenPermitted(t *testing.T) {
	kb := &failingReadKB{KB: newMemKB(map[string]string{"notes/plain.md": "hello\n"})}
	scoped := NewScopedKB(kb, nil, []string{"**"})
	scoped.allowRoleAuthoring = true

	require.NoError(t, scoped.Patch(context.Background(), "notes/plain.md", "hello", "bye"))
	require.True(t, kb.patched, "with the guard off there is no verification read to fail")
}

// Scope is checked before the guard: an out-of-scope write is a scope denial,
// not a role-authoring one, so the existing message stays accurate.
func TestScopedKB_ScopeDenialWinsOverRoleGuard(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, nil, []string{"allowed/**"})

	err := scoped.Write(context.Background(), "elsewhere/evil.md", "---\nfleet_id: codellm\n---\n")

	require.ErrorIs(t, err, ErrWriteDenied)
}

type failingReadKB struct {
	KB
	patched bool
}

func (f *failingReadKB) Read(context.Context, string) (string, error) {
	return "", errors.New("kb unavailable")
}

func (f *failingReadKB) Patch(ctx context.Context, path, find, replace string) error {
	f.patched = true
	return f.KB.Patch(ctx, path, find, replace)
}

// End-to-end through Run: the guard is on by default, the denial is recorded
// for the operator, and the model is told why rather than left to invent a
// reason. Covers the wiring from Input through to ScopedKB, which is the part
// that breaks silently.
func TestRun_DeniesAuthoringARoleNote(t *testing.T) {
	kb := newMemKB(nil)
	llm := &stubLLM{
		script: []ChatResult{
			{
				ToolCalls: []ToolCall{toolCall("1", toolWriteNote, map[string]any{
					"path":    "notes/successor.md",
					"content": "---\nfleet_id: codellm\nmode: cron\nwrite_patterns:\n  - \"**\"\n---\nsteal\n",
				})},
				PromptTokens: 10, CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})}, PromptTokens: 10, CompletionTokens: 5},
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:   "file the transcript",
		WritePatterns: []string{"**"},
		Model:         "test-model",
		MaxTokens:     10000,
		MaxSteps:      10,
		LLM:           llm,
		KB:            kb,
	})

	require.NoError(t, err)
	require.Empty(t, res.Changes, "a denied role note must not be reported as a change")
	require.Len(t, res.Denials, 1)
	require.Contains(t, res.Denials[0], "notes/successor.md")
	require.Contains(t, res.Denials[0], "fleet_id")
	_, leaked := kb.docs["notes/successor.md"]
	require.False(t, leaked, "denied role note reached the KB")
}

func TestRun_AllowRoleAuthoringOptsOut(t *testing.T) {
	kb := newMemKB(nil)
	llm := &stubLLM{
		script: []ChatResult{
			{
				ToolCalls: []ToolCall{toolCall("1", toolWriteNote, map[string]any{
					"path":    "roles/new.md",
					"content": "---\nfleet_id: codellm\n---\nbody\n",
				})},
				PromptTokens: 10, CompletionTokens: 5,
			},
			{ToolCalls: []ToolCall{toolCall("2", toolFinish, map[string]any{"answer": "done"})}, PromptTokens: 10, CompletionTokens: 5},
		},
	}

	res, err := Run(context.Background(), Input{
		Instruction:        "manage the fleet",
		WritePatterns:      []string{"**"},
		AllowRoleAuthoring: true,
		Model:              "test-model",
		MaxTokens:          10000,
		MaxSteps:           10,
		LLM:                llm,
		KB:                 kb,
	})

	require.NoError(t, err)
	require.Len(t, res.Changes, 1)
	require.Empty(t, res.Denials)
}

// Patching a note that does not exist is an ordinary apply failure, not a
// denial. It still fails closed — a scoped reader cannot tell "absent" from
// "outside my read scope" — but it must not be logged as a role-authoring hit,
// or the denial log fills with false accusations and hides trip2g's own reason.
func TestScopedKB_PatchMissingNoteIsNotReportedAsDenial(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, nil, []string{"**"})

	err := scoped.Patch(context.Background(), "notes/absent.md", "a", "b")

	require.ErrorIs(t, err, ErrRoleGuardUnverifiable)
	require.NotErrorIs(t, err, ErrRoleAuthoringDenied)

	_, isDenial := writeDenial(err, "patch", "notes/absent.md")
	require.False(t, isDenial, "must not be classified as a denial")
	require.ErrorContains(t, err, "not found", "the underlying cause must reach the operator")
}

// The two real denials stay denials, with the reason the operator needs.
func TestWriteDenialClassification(t *testing.T) {
	scopeMsg, ok := writeDenial(ErrWriteDenied, "write", "a.md")
	require.True(t, ok)
	require.Equal(t, "write a.md", scopeMsg)

	roleMsg, ok := writeDenial(ErrRoleAuthoringDenied, "write", "a.md")
	require.True(t, ok)
	require.Contains(t, roleMsg, "fleet_id")

	_, ok = writeDenial(errors.New("boom"), "write", "a.md")
	require.False(t, ok)
}
