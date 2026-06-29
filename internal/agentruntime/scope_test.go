package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopedKB_ReadEnforcement(t *testing.T) {
	kb := newMemKB(map[string]string{
		"subjects/subj1/profile.md": "subj1 profile",
		"subjects/subj2/profile.md": "subj2 profile",
	})
	scoped := NewScopedKB(kb, []string{"subjects/subj1/**"}, nil)

	got, err := scoped.Read(context.Background(), "subjects/subj1/profile.md")
	if err != nil {
		t.Fatalf("in-scope read failed: %v", err)
	}
	if got != "subj1 profile" {
		t.Fatalf("unexpected content: %q", got)
	}

	_, err = scoped.Read(context.Background(), "subjects/subj2/profile.md")
	if !errors.Is(err, ErrReadDenied) {
		t.Fatalf("out-of-scope read should be denied, got %v", err)
	}
}

func TestScopedKB_SearchFiltersOutOfScope(t *testing.T) {
	kb := newMemKB(map[string]string{
		"subjects/subj1/profile.md": "shared keyword",
		"subjects/subj2/profile.md": "shared keyword",
	})
	scoped := NewScopedKB(kb, []string{"subjects/subj1/**"}, nil)

	docs, err := scoped.Search(context.Background(), "shared")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(docs) != 1 || docs[0].Path != "subjects/subj1/profile.md" {
		t.Fatalf("search must return only in-scope docs, got %+v", docs)
	}
}

func TestScopedKB_WriteEnforcement(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, []string{"subjects/subj1/**"}, []string{"subjects/subj1/**"})

	err := scoped.Write(context.Background(), "subjects/subj1/note.md", "ok")
	if err != nil {
		t.Fatalf("in-scope write failed: %v", err)
	}
	if kb.docs["subjects/subj1/note.md"] != "ok" {
		t.Fatalf("write did not persist")
	}

	err = scoped.Write(context.Background(), "subjects/subj2/note.md", "nope")
	if !errors.Is(err, ErrWriteDenied) {
		t.Fatalf("out-of-scope write should be denied, got %v", err)
	}
}

func TestScopedKB_EmptyWritePatternsIsReadOnly(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, []string{"subjects/subj1/**"}, nil)

	err := scoped.Write(context.Background(), "subjects/subj1/note.md", "x")
	if !errors.Is(err, ErrWriteDenied) {
		t.Fatalf("empty write patterns must deny all writes, got %v", err)
	}
}

// TestScopedKB_PatchUniqueness is the regression test for G5: Patch must error
// when find is absent (0 occurrences) or ambiguous (>1 occurrences), and must
// patch correctly when find is unique (exactly 1 occurrence). The duplicate-match
// case would silently patch the wrong card before the fix.
func TestScopedKB_PatchUniqueness(t *testing.T) {
	ctx := context.Background()
	const path = "boards/sprint.md"
	patterns := []string{"boards/**"}

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
			initial: "alpha\nbeta\ngamma\n",
			find:    "beta",
			replace: "BETA",
			wantErr: false,
			wantDoc: "alpha\nBETA\ngamma\n",
		},
		{
			name:    "missing find returns error",
			initial: "alpha\nbeta\ngamma\n",
			find:    "delta",
			replace: "DELTA",
			wantErr: true,
			wantDoc: "alpha\nbeta\ngamma\n", // unchanged
		},
		{
			name:    "duplicate find returns error without patching",
			initial: "- task @status:todo\n- other @status:todo\n",
			find:    "@status:todo",
			replace: "@status:doing",
			wantErr: true,
			wantDoc: "- task @status:todo\n- other @status:todo\n", // unchanged
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kb := newMemKB(map[string]string{path: tc.initial})
			scoped := NewScopedKB(kb, patterns, patterns)

			err := scoped.Patch(ctx, path, tc.find, tc.replace)
			if tc.wantErr {
				require.Error(t, err, "expected error for case %q", tc.name)
			} else {
				require.NoError(t, err, "unexpected error for case %q", tc.name)
			}
			require.Equal(t, tc.wantDoc, kb.docs[path], "document content mismatch for case %q", tc.name)
		})
	}
}
