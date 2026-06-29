package agentruntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFileKB_PatchUniqueness is the filesystem-level regression test for G5:
// FileKB.Patch must error on an absent find (0 occurrences) or an ambiguous
// find (>1 occurrences), and must apply cleanly on a unique find (1 occurrence).
func TestFileKB_PatchUniqueness(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		initial string
		find    string
		replace string
		wantErr bool
		wantDoc string
	}{
		{
			name:    "unique find patches correctly",
			initial: "line one\nline two\nline three\n",
			find:    "line two",
			replace: "LINE TWO",
			wantErr: false,
			wantDoc: "line one\nLINE TWO\nline three\n",
		},
		{
			name:    "missing find returns error",
			initial: "line one\nline two\n",
			find:    "line four",
			replace: "LINE FOUR",
			wantErr: true,
			wantDoc: "line one\nline two\n", // unchanged
		},
		{
			name:    "duplicate find returns error without writing",
			initial: "- card A @status:todo\n- card B @status:todo\n",
			find:    "@status:todo",
			replace: "@status:doing",
			wantErr: true,
			wantDoc: "- card A @status:todo\n- card B @status:todo\n", // unchanged
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := "note.md"
			absPath := filepath.Join(dir, path)
			require.NoError(t, os.WriteFile(absPath, []byte(tc.initial), 0o644))

			kb := NewFileKB(dir)
			err := kb.Patch(ctx, path, tc.find, tc.replace)

			if tc.wantErr {
				require.Error(t, err, "expected error for case %q", tc.name)
			} else {
				require.NoError(t, err, "unexpected error for case %q", tc.name)
			}

			got, readErr := os.ReadFile(absPath)
			require.NoError(t, readErr)
			require.Equal(t, tc.wantDoc, string(got), "file content mismatch for case %q", tc.name)
		})
	}
}
