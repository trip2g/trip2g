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

// TestScopedKB_LeadingSlashNormalization is the regression test for the
// leading-slash matching bug: small models sometimes prepend "/" or "./" to the
// path they write, so a candidate like "/concepts/x.md" was wrongly denied
// against pattern "concepts/**". Normalization must ALLOW those while keeping
// traversal/absolute-escape paths DENIED.
func TestScopedKB_LeadingSlashNormalization(t *testing.T) {
	patterns := []string{"concepts/**"}
	scoped := NewScopedKB(newMemKB(nil), patterns, patterns)

	allowed := []string{
		"concepts/x.md",
		"/concepts/x.md",
		"./concepts/x.md",
		"concepts/sub/y.md",
		"/concepts/sub/y.md",
	}
	for _, p := range allowed {
		if !scoped.CanWrite(p) {
			t.Errorf("expected ALLOW for %q under %v, got DENY", p, patterns)
		}
		if !scoped.CanRead(p) {
			t.Errorf("expected read ALLOW for %q under %v, got DENY", p, patterns)
		}
	}

	denied := []string{
		"../x.md",
		"concepts/../x.md",
		"/concepts/../secrets.md",
		"/etc/passwd",
		"../../etc/passwd",
		"concepts/../../etc/passwd",
		"secrets/x.md",
	}
	for _, p := range denied {
		if scoped.CanWrite(p) {
			t.Errorf("expected DENY for %q under %v, got ALLOW", p, patterns)
		}
	}
}

// TestScopedKB_BackslashTraversalDenied: FileKB.resolve treats "\" as a path
// separator (ReplaceAll to "/"), but scope matching used path.Clean where "\"
// is an ordinary character — so `concepts/..\secrets/x.md` passed the
// "concepts/**" glob yet resolved to secrets/x.md, outside the scope. Both
// layers must agree: backslash traversal is denied.
func TestScopedKB_BackslashTraversalDenied(t *testing.T) {
	patterns := []string{"concepts/**"}
	scoped := NewScopedKB(newMemKB(nil), patterns, patterns)

	denied := []string{
		`concepts/..\secrets/x.md`,
		`..\secrets\x.md`,
		`concepts\..\..\etc\passwd`,
	}
	for _, p := range denied {
		if scoped.CanRead(p) {
			t.Errorf("expected read DENY for %q under %v, got ALLOW", p, patterns)
		}
		if scoped.CanWrite(p) {
			t.Errorf("expected write DENY for %q under %v, got ALLOW", p, patterns)
		}
	}

	// Backslash-separated in-scope paths must still resolve and match, same as
	// FileKB.resolve would.
	if !scoped.CanRead(`concepts\x.md`) {
		t.Errorf("expected read ALLOW for %q under %v, got DENY", `concepts\x.md`, patterns)
	}
}

// TestScopedKB_WritePassesNormalizedPathToKB: normalization must apply to the
// path handed DOWNSTREAM, not just to glob matching. Today Write("/concepts/x.md")
// passes the scope check but forwards the raw slash path to the underlying KB,
// so the doc lands at "/concepts/x.md" — downstream lookups keyed by the
// normalized path ("concepts/x.md") miss and a duplicate ghost note appears.
func TestScopedKB_WritePassesNormalizedPathToKB(t *testing.T) {
	kb := newMemKB(nil)
	patterns := []string{"concepts/**"}
	scoped := NewScopedKB(kb, patterns, patterns)

	err := scoped.Write(context.Background(), "/concepts/x.md", "content")
	require.NoError(t, err)

	require.Contains(t, kb.docs, "concepts/x.md",
		"KB must receive the normalized path, not the raw slash-prefixed one")
	require.NotContains(t, kb.docs, "/concepts/x.md",
		"raw slash path must not leak into the KB")
}

// TestScopedKB_LeadingSlashPatternsStillMatch: patterns themselves are never
// normalized, so write_patterns like ["/concepts/**"] silently match NOTHING
// (candidates are normalized slash-less) — a deny-all lockout. Desired: a
// leading-slash pattern matches the same paths as its slash-less form.
func TestScopedKB_LeadingSlashPatternsStillMatch(t *testing.T) {
	patterns := []string{"/concepts/**"}
	scoped := NewScopedKB(newMemKB(nil), patterns, patterns)

	require.True(t, scoped.CanWrite("concepts/x.md"),
		"pattern %q must match %q after pattern normalization", patterns[0], "concepts/x.md")
	require.True(t, scoped.CanRead("concepts/x.md"),
		"read pattern %q must match %q after pattern normalization", patterns[0], "concepts/x.md")
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
