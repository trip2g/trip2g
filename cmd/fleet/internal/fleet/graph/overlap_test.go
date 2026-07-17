package graph

import (
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/stretchr/testify/require"
)

func TestOverlap(t *testing.T) {
	tests := []struct {
		name string
		p, q string
		want bool
		// witness: for want=true cases, a path that must match BOTH patterns
		// under doublestar.Match — validates the fixture against the real
		// delivery-matching semantics. Empty = skip (conservative-true cases).
		witness string
	}{
		// literals
		{"equal literals", "roles/a.md", "roles/a.md", true, "roles/a.md"},
		{"different literals", "roles/a.md", "roles/b.md", false, ""},
		{"different first segment", "roles/*", "transcripts/**", false, ""},
		{"different depth literals", "a/b", "a", false, ""},

		// single star
		{"star vs literal", "logs/*", "logs/run.md", true, "logs/run.md"},
		{"star vs star", "logs/*.md", "logs/run-*", true, "logs/run-1.md"},
		{"star suffix mismatch", "logs/*.md", "logs/*.txt", false, ""},
		{"star does not cross slash", "logs/*", "logs/a/b", false, ""},
		{"embedded stars", "a*c", "ab*", true, "abc"},

		// doublestar
		{"doublestar vs deep literal", "wiki/**", "wiki/topics/x.md", true, "wiki/topics/x.md"},
		{"doublestar vs star", "wiki/**", "wiki/topics/*.md", true, "wiki/topics/a.md"},
		{"doublestar prefix mismatch", "wiki/**", "logs/**", false, ""},
		{"bare doublestar matches all", "**", "any/deep/path.md", true, "any/deep/path.md"},
		{"doublestar both sides", "a/**/z.md", "**/z.md", true, "a/z.md"},
		{"doublestar mid-pattern", "a/**/c", "a/b/c", true, "a/b/c"},
		{"doublestar mid-pattern mismatch", "a/**/c", "a/b/d", false, ""},
		{"trailing doublestar zero segments", "a/**", "a/*", true, "a/x"},

		// ? and classes
		{"question vs literal", "logs/r?n.md", "logs/run.md", true, "logs/run.md"},
		{"class vs literal in range", "logs/run-[0-9].md", "logs/run-5.md", true, "logs/run-5.md"},
		{"class vs literal out of range", "logs/run-[0-9].md", "logs/run-x.md", false, ""},
		{"class vs star", "logs/*.md", "logs/run-[0-9]*.md", true, "logs/run-1.md"},
		{"disjoint classes", "x/[a-c].md", "x/[d-f].md", false, ""},
		{"overlapping classes", "x/[a-e].md", "x/[c-z].md", true, "x/d.md"},
		{"negated class conservative", "x/[!a].md", "x/[a-z].md", true, ""},

		// braces
		{"brace expansion hit", "docs/{en,ru}/*.md", "docs/ru/intro.md", true, "docs/ru/intro.md"},
		{"brace expansion miss", "docs/{en,ru}/*.md", "docs/de/intro.md", false, ""},
		{"braces both sides", "a/{b,c}/x", "a/{c,d}/x", true, "a/c/x"},

		// escapes
		{"escaped star literal", `a/\*`, "a/b", false, ""},
		{"escaped star vs itself", `a/\*`, `a/\*`, true, `a/*`},

		// degenerate / conservative
		{"empty vs empty", "", "", true, ""},
		{"empty vs pattern", "", "a", false, ""},
		{"unclosed class conservative", "a/[x", "a/anything", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Overlap(tt.p, tt.q), "Overlap(%q, %q)", tt.p, tt.q)
			require.Equal(t, tt.want, Overlap(tt.q, tt.p), "Overlap(%q, %q) (symmetric)", tt.q, tt.p)
			if tt.witness != "" {
				mp, err := doublestar.Match(tt.p, tt.witness)
				require.NoError(t, err)
				mq, err := doublestar.Match(tt.q, tt.witness)
				require.NoError(t, err)
				require.True(t, mp && mq, "witness %q must match both patterns under doublestar", tt.witness)
			}
		})
	}
}
