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

// TestRenderInstruction_NoSecretLeakage asserts secrets/api_token are never
// exposed as template vars: referencing them errors rather than leaking.
func TestRenderInstruction_NoSecretLeakage(t *testing.T) {
	for _, body := range []string{"{{ secrets }}", "{{ api_token }}", "{{ secrets.openai }}"} {
		out, err := renderInstruction(body, renderCtx{})
		require.Error(t, err, "referencing %q must error (var not exposed)", body)
		require.Empty(t, out)
	}
}
