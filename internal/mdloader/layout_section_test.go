package mdloader

import (
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestParseLayoutSections_HeaderOnly(t *testing.T) {
	meta := map[string]interface{}{
		"header_includes": []interface{}{"blog/**"},
	}
	entries := parseLayoutSections(meta, "layouts/header.md")
	require.Len(t, entries, 1)
	require.Equal(t, "header", entries[0].Section)
	require.Equal(t, "layouts/header.md", entries[0].NotePath)
	require.Equal(t, []string{"blog/**"}, entries[0].Includes)
}

func TestParseLayoutSections_HeaderAndFooter(t *testing.T) {
	meta := map[string]interface{}{
		"header_includes": []interface{}{"blog/**"},
		"footer_includes": []interface{}{"**/*.md"},
	}
	entries := parseLayoutSections(meta, "layouts/hf.md")
	require.Len(t, entries, 2)

	sections := make(map[string]model.LayoutSectionEntry)
	for _, e := range entries {
		sections[e.Section] = e
	}
	require.Contains(t, sections, "header")
	require.Contains(t, sections, "footer")
}

func TestParseLayoutSections_AllFourSections(t *testing.T) {
	meta := map[string]interface{}{
		"header_includes":        []interface{}{"**/*.md"},
		"footer_includes":        []interface{}{"**/*.md"},
		"left_sidebar_includes":  []interface{}{"**/*.md"},
		"right_sidebar_includes": []interface{}{"**/*.md"},
	}
	entries := parseLayoutSections(meta, "layouts/all.md")
	require.Len(t, entries, 4)

	sections := make(map[string]bool)
	for _, e := range entries {
		sections[e.Section] = true
	}
	require.True(t, sections["header"])
	require.True(t, sections["footer"])
	require.True(t, sections["left_sidebar"])
	require.True(t, sections["right_sidebar"])
}

func TestParseLayoutSections_NoIncludesReturnsNil(t *testing.T) {
	meta := map[string]interface{}{
		"title": "Just a regular note",
	}
	entries := parseLayoutSections(meta, "notes/regular.md")
	require.Nil(t, entries)
}

func TestParseLayoutSections_SingleStringInclude(t *testing.T) {
	meta := map[string]interface{}{
		"header_includes": "blog/**",
	}
	entries := parseLayoutSections(meta, "layouts/header.md")
	require.Len(t, entries, 1)
	require.Equal(t, []string{"blog/**"}, entries[0].Includes)
}

func TestParseLayoutSections_PriorityDefaultsToZero(t *testing.T) {
	meta := map[string]interface{}{
		"header_includes": []interface{}{"**/*.md"},
	}
	entries := parseLayoutSections(meta, "layouts/header.md")
	require.Len(t, entries, 1)
	require.Equal(t, 0, entries[0].Priority)
}

func TestParseLayoutSections_PriorityParsed(t *testing.T) {
	meta := map[string]interface{}{
		"header_includes": []interface{}{"**/*.md"},
		"header_priority": 5,
	}
	entries := parseLayoutSections(meta, "layouts/header.md")
	require.Len(t, entries, 1)
	require.Equal(t, 5, entries[0].Priority)
}

func TestParseLayoutSections_IncludePropertyParsed(t *testing.T) {
	meta := map[string]interface{}{
		"header_includes":         []interface{}{"**/*.md"},
		"header_include_property": "published",
	}
	entries := parseLayoutSections(meta, "layouts/header.md")
	require.Len(t, entries, 1)
	require.Equal(t, "published", entries[0].IncludeProperty)
}

func TestParseLayoutSections_ExcludesParsed(t *testing.T) {
	meta := map[string]interface{}{
		"header_includes": []interface{}{"**/*.md"},
		"header_excludes": []interface{}{"drafts/**", "private/**"},
	}
	entries := parseLayoutSections(meta, "layouts/header.md")
	require.Len(t, entries, 1)
	require.Equal(t, []string{"drafts/**", "private/**"}, entries[0].Excludes)
}

func TestParseLayoutSections_ExcludePropertyParsed(t *testing.T) {
	meta := map[string]interface{}{
		"header_includes":         []interface{}{"**/*.md"},
		"header_exclude_property": "draft",
	}
	entries := parseLayoutSections(meta, "layouts/header.md")
	require.Len(t, entries, 1)
	require.Equal(t, "draft", entries[0].ExcludeProperty)
}
