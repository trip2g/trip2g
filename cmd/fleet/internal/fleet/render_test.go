package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderInstruction_ContextVars(t *testing.T) {
	ctx := renderCtx{
		ChangedFiles: []changeInfo{
			{Path: "boards/sprint.md", Title: "Sprint"},
			{Path: "boards/backlog.md", Title: "Backlog"},
		},
		ChangeFile: &changeInfo{Path: "boards/sprint.md", Title: "Sprint", Content: "- a @status:todo\n"},
		Depth:      1,
	}
	body := "File: {{ change_file.Path }}\nContent: {{ change_file.Content }}\nTitles:{{ range changed_files }} {{ .Title }}{{ end }}\nDepth: {{ depth }}"

	out, err := renderInstruction(body, ctx)
	require.NoError(t, err)
	require.Contains(t, out, "File: boards/sprint.md")
	require.Contains(t, out, "Content: - a @status:todo")
	require.Contains(t, out, "Titles: Sprint Backlog")
	require.Contains(t, out, "Depth: 1")
}

// TestRenderInstruction_AttachedNotesHint asserts that when attached_notes are
// present but the role body doesn't reference them, the rendered instruction
// still names the attached note paths so the model knows they're readable.
func TestRenderInstruction_AttachedNotesHint(t *testing.T) {
	ctx := renderCtx{
		AttachedNotes: []attachedNote{
			{Path: "roles/triage.md", Title: "Triage role"},
			{Path: "boards/policy.md", Title: "Policy"},
		},
	}
	out, err := renderInstruction("Do the work.", ctx)
	require.NoError(t, err)
	require.Contains(t, out, "Do the work.")
	require.Contains(t, out, "Attached notes available:")
	require.Contains(t, out, "roles/triage.md")
	require.Contains(t, out, "boards/policy.md")
}

// TestRenderInstruction_AttachedNotesHint_SkipWhenReferenced asserts the hint is
// not appended for notes the template already names.
func TestRenderInstruction_AttachedNotesHint_SkipWhenReferenced(t *testing.T) {
	ctx := renderCtx{
		AttachedNotes: []attachedNote{{Path: "roles/triage.md", Title: "Triage role"}},
	}
	out, err := renderInstruction("See roles/triage.md for policy.", ctx)
	require.NoError(t, err)
	require.NotContains(t, out, "Attached notes available:")
}

// TestRenderInstruction_NoSecretLeakage asserts secrets/api_token are never
// exposed as template vars: referencing them errors rather than leaking.
func TestRenderInstruction_NoSecretLeakage(t *testing.T) {
	for _, body := range []string{"{{ secrets }}", "{{ api_token }}", "{{ secrets.openai }}"} {
		out, err := renderInstruction(body, renderCtx{})
		require.Error(t, err, "referencing %q must error (var not exposed)", body)
		require.Empty(t, out)
	}
}
