package defaulttemplate

import "testing"

func TestParseContentRefWidgetKeywords(t *testing.T) {
	cases := []struct {
		in   string
		want ContentRefKind
	}{
		{"self", ContentRefSelfContent},
		{"selfcontent", ContentRefSelfContent},
		{"magazine", ContentRefMagazine},
		{"toc", ContentRefTOC},
		{"TOC", ContentRefTOC},
		{"inlinks", ContentRefInLinks},
		{"backlinks", ContentRefInLinks},
		{"outlinks", ContentRefOutLinks},
		{"similar", ContentRefSimilar},
		{"false", ContentRefNone},
		{"[[Some Note]]", ContentRefWikiLink},
		{"docs/page.md", ContentRefFile},
	}
	for _, c := range cases {
		got := parseContentRef(c.in)
		if got.Kind != c.want {
			t.Errorf("parseContentRef(%q).Kind = %v, want %v", c.in, got.Kind, c.want)
		}
	}
}
