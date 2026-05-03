package layoutloader

import (
	"os"
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func testLoadLayouts(t *testing.T, sources []model.LayoutSourceFile) *model.Layouts {
	t.Helper()
	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)
	return layouts
}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func renderLayout(t *testing.T, layouts *model.Layouts, sourceID string) string {
	t.Helper()
	layout, ok := layouts.Map[sourceID]
	require.True(t, ok, "layout %q not found", sourceID)
	require.NotNil(t, layout.View, "layout %q has no view (parse error?)", sourceID)
	var buf strings.Builder
	err := layout.View.Execute(&buf, nil, nil)
	require.NoError(t, err)
	return buf.String()
}

func TestYieldBlocks_PrefixPattern(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/page_home.html", Path: "testdata/blocks_inline/page_home.html", Content: readFixture(t, "testdata/blocks_inline/page_home.html")},
		{ID: "/comp_nav.html", Path: "testdata/blocks_inline/comp_nav.html", Content: readFixture(t, "testdata/blocks_inline/comp_nav.html")},
		{ID: "/comp_hero.html", Path: "testdata/blocks_inline/comp_hero.html", Content: readFixture(t, "testdata/blocks_inline/comp_hero.html")},
		{ID: "/comp_footer.html", Path: "testdata/blocks_inline/comp_footer.html", Content: readFixture(t, "testdata/blocks_inline/comp_footer.html")},
		{ID: "/comp_navinner.html", Path: "testdata/blocks_inline/comp_navinner.html", Content: readFixture(t, "testdata/blocks_inline/comp_navinner.html")},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/page_home.html")

	require.Contains(t, out, ".nav {", "nav style expected")
	require.Contains(t, out, ".hero {", "hero style expected")
	require.NotContains(t, out, ".footer {", "footer style must be absent")
}

func TestYieldBlocks_RegexpPattern(t *testing.T) {
	pageContent := `{{ import "/comp_nav.html" }}
{{ import "/comp_navinner.html" }}
<html><body>
{{yield nav()}}
<style>{{yield_blocks("/_style_.*/")}}</style>
</body></html>`

	sources := []model.LayoutSourceFile{
		{ID: "/page.html", Path: "testdata/blocks_inline/page.html", Content: pageContent},
		{ID: "/comp_nav.html", Path: "testdata/blocks_inline/comp_nav.html", Content: readFixture(t, "testdata/blocks_inline/comp_nav.html")},
		{ID: "/comp_navinner.html", Path: "testdata/blocks_inline/comp_navinner.html", Content: readFixture(t, "testdata/blocks_inline/comp_navinner.html")},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/page.html")
	require.Contains(t, out, ".nav {")
	require.Contains(t, out, ".navinner {")
}

func TestYieldBlocks_TransitiveDeps(t *testing.T) {
	// page → nav → nav_inner: nav_inner CSS must appear
	sources := []model.LayoutSourceFile{
		{ID: "/page_home.html", Path: "testdata/blocks_inline/page_home.html", Content: readFixture(t, "testdata/blocks_inline/page_home.html")},
		{ID: "/comp_nav.html", Path: "testdata/blocks_inline/comp_nav.html", Content: readFixture(t, "testdata/blocks_inline/comp_nav.html")},
		{ID: "/comp_hero.html", Path: "testdata/blocks_inline/comp_hero.html", Content: readFixture(t, "testdata/blocks_inline/comp_hero.html")},
		{ID: "/comp_navinner.html", Path: "testdata/blocks_inline/comp_navinner.html", Content: readFixture(t, "testdata/blocks_inline/comp_navinner.html")},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/page_home.html")
	require.Contains(t, out, ".navinner {", "transitive dep style expected")
}

func TestYieldBlocks_InvalidRegexp(t *testing.T) {
	pageContent := `<html><body><style>{{yield_blocks("/[/")}}</style></body></html>`
	sources := []model.LayoutSourceFile{
		{ID: "/page.html", Path: "testdata/blocks_inline/page.html", Content: pageContent},
	}
	layouts := testLoadLayouts(t, sources)
	layout := layouts.Map["/page.html"]
	hasWarn := false
	for _, w := range layout.Warnings {
		if strings.Contains(w.Message, "invalid") || strings.Contains(w.Message, "regexp") {
			hasWarn = true
		}
	}
	require.True(t, hasWarn, "expected NoteWarning for invalid regexp")
	// render must not panic
	if layout.View != nil {
		var buf strings.Builder
		_ = layout.View.Execute(&buf, nil, nil)
	}
}

func TestYieldBlocks_DuplicateBlockName(t *testing.T) {
	comp1 := `{{block same_block()}}comp1{{end}}`
	comp2 := `{{block same_block()}}comp2{{end}}`
	pageContent := `{{yield same_block()}}<style>{{yield_blocks("same")}}</style>`
	sources := []model.LayoutSourceFile{
		{ID: "/page.html", Path: "testdata/blocks_inline/page.html", Content: pageContent},
		{ID: "/comp1.html", Path: "testdata/blocks_inline/comp1.html", Content: comp1},
		{ID: "/comp2.html", Path: "testdata/blocks_inline/comp2.html", Content: comp2},
	}
	layouts := testLoadLayouts(t, sources)
	layout := layouts.Map["/page.html"]
	hasWarn := false
	for _, w := range layout.Warnings {
		if strings.Contains(w.Message, "same_block") {
			hasWarn = true
		}
	}
	require.True(t, hasWarn, "expected NoteWarning for duplicate block name")
}

func TestYieldBlocks_UnknownBlockYielded(t *testing.T) {
	pageContent := `{{yield does_not_exist()}}<style>{{yield_blocks("_style_")}}</style>`
	sources := []model.LayoutSourceFile{
		{ID: "/page.html", Path: "testdata/blocks_inline/page.html", Content: pageContent},
	}
	// Load should not panic
	layouts := testLoadLayouts(t, sources)
	_ = layouts
}

func TestYieldBlocks_PageBlocksNotInRegistry(t *testing.T) {
	// page defines its own block — should still be yieldable, registry should not interfere
	pageContent := `{{block my_block()}}page-content{{end}}{{yield my_block()}}`
	sources := []model.LayoutSourceFile{
		{ID: "/page.html", Path: "testdata/blocks_inline/page.html", Content: pageContent},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/page.html")
	require.Contains(t, out, "page-content")
}

func TestExpandBlockName(t *testing.T) {
	tests := []struct {
		sourceID string
		content  string
		want     string
	}{
		{"/components/button.html", "{{block $fileID()}}", "{{block components_button()}}"},
		{"card.html", "_style_$fileID", "_style_card"},
		{"/a/b/c.html", "$fileID", "a_b_c"},
		// escape: $$fileID → literal $fileID
		{"button.html", "var $$fileID = 1;", "var $fileID = 1;"},
		// no $fileID → unchanged
		{"button.html", "no placeholder", "no placeholder"},
		// both in same string
		{"button.html", "$fileID and $$fileID", "button and $fileID"},
	}
	for _, tt := range tests {
		got := expandBlockName(tt.content, tt.sourceID)
		if got != tt.want {
			t.Errorf("expandBlockName(%q, %q) = %q, want %q", tt.sourceID, tt.content, got, tt.want)
		}
	}
}
