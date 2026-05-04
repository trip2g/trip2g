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

// TestAutoImport_YieldWithoutExplicitImport verifies that a page can {{ yield X() }}
// a block defined in a sibling component file WITHOUT an explicit {{ import }}.
// This is the core auto-import contract: HTML blocks (not just CSS via yield_blocks)
// must be available transparently.
func TestAutoImport_YieldWithoutExplicitImport(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{
			ID:   "/mesh/ru_index",
			Path: "mesh/ru_index.html",
			// No explicit {{ import "/mesh/bar" }} — yield must still resolve.
			Content: `<html><body>{{yield mesh_bar_ru()}}</body></html>`,
		},
		{
			ID:      "/mesh/bar",
			Path:    "mesh/bar.html",
			Content: `{{block mesh_bar_ru()}}<header class="bar-ru">RU</header>{{end}}`,
		},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/mesh/ru_index")
	require.Contains(t, out, `<header class="bar-ru">RU</header>`,
		"yield mesh_bar_ru() must resolve via auto-import (no explicit import statement)")
}

// TestAutoImport_TransitiveYield verifies that auto-import follows transitive
// yields: page yields A, A yields B, B's block must be parsed too.
func TestAutoImport_TransitiveYield(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{
			ID:      "/page",
			Path:    "page.html",
			Content: `<html>{{yield outer()}}</html>`,
		},
		{
			ID:      "/comp_outer",
			Path:    "comp_outer.html",
			Content: `{{block outer()}}<div class="outer">{{yield inner()}}</div>{{end}}`,
		},
		{
			ID:      "/comp_inner",
			Path:    "comp_inner.html",
			Content: `{{block inner()}}<span class="inner">x</span>{{end}}`,
		},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/page")
	require.Contains(t, out, `<div class="outer">`)
	require.Contains(t, out, `<span class="inner">x</span>`,
		"transitive yield must be resolved via BFS auto-import")
}

// TestYieldBlocks_StyleMeshTransitive is the exact scenario from the bug report:
// page yields mesh_hero (via @lid), hero yields mesh_button (via @lid),
// yield_blocks("_style_mesh_") must collect CSS from BOTH hero and button.
func TestYieldBlocks_StyleMeshTransitive(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{
			ID:   "/mesh/index",
			Path: "mesh/index.html",
			Content: `<html><head><style>{{yield_blocks("_style_mesh_")}}</style></head>` +
				`<body>{{yield mesh_hero()}}</body></html>`,
		},
		{
			ID:   "/mesh/hero",
			Path: "mesh/hero.html",
			Content: `{{block _style_@lid()}}.mesh-hero{color:red}{{end}}` +
				`{{block @lid()}}<section>{{yield mesh_button()}}</section>{{end}}`,
		},
		{
			ID:   "/mesh/button",
			Path: "mesh/button.html",
			Content: `{{block _style_@lid()}}.mesh-button{display:inline-flex}{{end}}` +
				`{{block @lid()}}<button class="mesh-button">click</button>{{end}}`,
		},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/mesh/index")
	require.Contains(t, out, ".mesh-hero{color:red}", "hero CSS must appear via yield_blocks")
	require.Contains(t, out, ".mesh-button{display:inline-flex}",
		"button CSS must appear via yield_blocks — transitive dep through hero")
}

func TestExpandBlockName_Integration(t *testing.T) {
	// Regression: jl.load() must apply expandBlockName so that @lid placeholders
	// are resolved before Jet parses the template. Without the fix, block @lid()
	// stays literal and yield mesh_bar() / yield_blocks("_style_") find nothing.
	sources := []model.LayoutSourceFile{
		{
			ID:   "/mesh/index",
			Path: "mesh/index.html",
			Content: `{{ import "/mesh/bar" }}` + "\n" +
				`<style>{{yield_blocks("_style_")}}</style>` + "\n" +
				`{{yield mesh_bar()}}`,
		},
		{
			ID:   "/mesh/bar",
			Path: "mesh/bar.html",
			Content: `{{block _style_@lid()}}.bar{}{{end}}` + "\n" +
				`{{block @lid()}}<header class="bar"></header>{{end}}`,
		},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/mesh/index")
	require.Contains(t, out, ".bar{}", "component CSS must be injected via yield_blocks")
	require.Contains(t, out, `<header class="bar">`, "component HTML must render via yield mesh_bar()")
}

func TestExpandBlockName(t *testing.T) {
	tests := []struct {
		sourceID string
		content  string
		want     string
	}{
		{"/components/button.html", "{{block @lid()}}", "{{block components_button()}}"},
		{"card.html", "_style_@lid", "_style_card"},
		{"/a/b/c.html", "@lid", "a_b_c"},
		{"/a/b/c.html", "@did", "a-b-c"},
		// escape: @@lid → literal @lid
		{"button.html", "var @@lid = 1;", "var @lid = 1;"},
		// CSS usage with @did
		{"button.html", ".@did__nav { }", ".button__nav { }"},
		// no placeholder → unchanged
		{"button.html", "no placeholder", "no placeholder"},
		// both in same string
		{"button.html", "@lid and @@lid", "button and @lid"},
	}
	for _, tt := range tests {
		got := expandBlockName(tt.content, tt.sourceID)
		if got != tt.want {
			t.Errorf("expandBlockName(%q, %q) = %q, want %q", tt.sourceID, tt.content, got, tt.want)
		}
	}
}

func TestYieldBlocks_ButtonCSSViaTransitiveDep(t *testing.T) {
	// page → hero → button (transitive). yield_blocks("_style_mesh_") must include button CSS.
	sources := []model.LayoutSourceFile{
		{
			ID: "/mesh/page.html", Path: "/mesh/page.html",
			Content: `{{yield mesh_hero()}}` + "\n" + `<style>{{yield_blocks("_style_mesh_")}}</style>`,
		},
		{
			ID: "/mesh/hero.html", Path: "/mesh/hero.html",
			Content: `{{block _style_@lid()}}.@did{color:red}{{end}}` + "\n" +
				`{{block @lid()}}{{yield mesh_button(label="start",href="/",variant="primary")}}{{end}}`,
		},
		{
			ID: "/mesh/button.html", Path: "/mesh/button.html",
			Content: `{{block _style_@lid()}}.@did{border:1px solid white}.@did--primary{background:green}{{end}}` + "\n" +
				`{{block @lid(label="",href="",variant="",modifier="")}}` +
				`<a class="@did{{if variant}} @did--{{variant}}{{end}}" href="{{href}}">{{label}}</a>{{end}}`,
		},
	}
	layouts := testLoadLayouts(t, sources)
	out := renderLayout(t, layouts, "/mesh/page.html")
	require.Contains(t, out, ".mesh-button", "base button CSS must be in output")
	require.Contains(t, out, ".mesh-button--primary", "primary button CSS must be in output")
	require.Contains(t, out, "background:green", "primary green background must be in output")
}
