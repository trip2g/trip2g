package layoutloader

import (
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestDerivePlaceholderIDs(t *testing.T) {
	cases := []struct {
		id      string
		wantLid string
		wantDid string
	}{
		{"/mesh/bar", "mesh_bar", "mesh-bar"},
		{"/mesh/bar.html", "mesh_bar", "mesh-bar"},
		{"mesh/bar", "mesh_bar", "mesh-bar"},
		{"/mesh/index", "mesh_index", "mesh-index"},
	}
	for _, c := range cases {
		lid, did := derivePlaceholderIDs(c.id)
		require.Equal(t, c.wantLid, lid, "lid for %s", c.id)
		require.Equal(t, c.wantDid, did, "did for %s", c.id)
	}
}

// lines returns the 1-based line numbers referenced by the warnings, for terse assertions.
func warnLines(t *testing.T, ws []model.NoteWarning, kind string) []string {
	t.Helper()
	var out []string
	for _, w := range ws {
		if strings.Contains(w.Message, "this file's "+kind) {
			out = append(out, w.Message)
		}
	}
	return out
}

func TestScanSelfLiteral_DidInCSSAndJS(t *testing.T) {
	// bar.html style: @did placeholders are fine, but a hardcoded mesh-bar class
	// (here in a JS querySelector string) is the drift to catch.
	src := "line1\n" +
		`.@did__nav { color: red; }` + "\n" +
		`document.querySelector('.mesh-bar__nav')` + "\n"
	ws := scanSelfLiteral(src, "/mesh/bar")
	require.Len(t, ws, 1)
	require.Contains(t, ws[0].Message, "line 3")
	require.Contains(t, ws[0].Message, `literal "mesh-bar"`)
	require.Contains(t, ws[0].Message, "this file's @did")
	require.Equal(t, model.NoteWarningWarning, ws[0].Level)
}

func TestScanSelfLiteral_EscapedPlaceholderNotFlagged(t *testing.T) {
	// @@did is an escape that stays literal "@did" in the raw source, so it never
	// matches the expanded "mesh-bar". Even when the expanded literal appears on the
	// SAME line elsewhere, only the true literal is flagged.
	src := `<div class="@did__x @@did">` + "\n" +
		`<div class="@did__y mesh-bar__z @@did">` + "\n"
	ws := scanSelfLiteral(src, "/mesh/bar")
	// Only line 2 has a real literal (mesh-bar__z); the @@did escapes are ignored.
	require.Len(t, ws, 1)
	require.Contains(t, ws[0].Message, "line 2")
}

func TestScanSelfLiteral_OtherComponentNotFlagged(t *testing.T) {
	// index.html composing other components by their expanded names is correct.
	src := `{{ yield mesh_bar() }}` + "\n" +
		`<div class="mesh-bar__nav"></div>` + "\n"
	ws := scanSelfLiteral(src, "/mesh/index")
	require.Empty(t, ws)
}

func TestScanSelfLiteral_DidWordBoundary(t *testing.T) {
	// mesh-bar must not match inside mesh-barometer.
	src := `.mesh-barometer { x: 1 }` + "\n"
	ws := scanSelfLiteral(src, "/mesh/bar")
	require.Empty(t, ws)
}

func TestScanSelfLiteral_DidBoundaryVariants(t *testing.T) {
	src := `a .mesh-bar { }` + "\n" + // whitespace after
		`b .mesh-bar__el { }` + "\n" + // __ suffix
		`c .mesh-bar--mod { }` + "\n" + // -- suffix
		`d "mesh-bar"` + "\n" + // quote after
		`e mesh-bar` + "\n" // end of line
	ws := scanSelfLiteral(src, "/mesh/bar")
	require.Len(t, warnLines(t, ws, "@did"), 5)
}

func TestScanSelfLiteral_LidCallForms(t *testing.T) {
	src := `{{ block mesh_bar() }}` + "\n" + // <lid>(
		`{{ block mesh_bar_ru() }}` + "\n" + // <lid>_ru(
		`{{ block _style_mesh_bar() }}` + "\n" + // _style_<lid>
		`{{ yield mesh_bar() }}` + "\n" // yield <lid>
	ws := scanSelfLiteral(src, "/mesh/bar")
	require.Len(t, warnLines(t, ws, "@lid"), 4)
}

func TestScanSelfLiteral_LidWordBoundary(t *testing.T) {
	// mesh_art must not match inside mesh_article.
	src := `{{ yield mesh_article() }}` + "\n" +
		`{{ block _style_mesh_articles() }}` + "\n"
	ws := scanSelfLiteral(src, "/mesh/art")
	require.Empty(t, ws)
}

func TestScanSelfLiteral_DedupePerLine(t *testing.T) {
	// Two literals on one line collapse to a single warning for that line/kind.
	src := `<a class="mesh-bar__x mesh-bar__y">` + "\n"
	ws := scanSelfLiteral(src, "/mesh/bar")
	require.Len(t, ws, 1)
}

func TestScanSelfLiteral_Clean(t *testing.T) {
	src := `{{ block @lid() }}<div class="@did__x">@@did</div>{{ end }}` + "\n"
	ws := scanSelfLiteral(src, "/mesh/bar")
	require.Empty(t, ws)
}

// Load surfaces self-literal warnings on the layout (which doclint reads) and
// preview returns them as strings.
func TestLoad_SurfacesSelfLiteralWarning(t *testing.T) {
	sources := []model.LayoutSourceFile{{
		ID:      "/mesh/bar",
		Path:    "_layouts/mesh/bar.html",
		Content: `{{ block @lid() }}<div class="mesh-bar__nav"></div>{{ end }}`,
	}}
	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)
	got := layouts.Map["/mesh/bar"].Warnings
	require.NotEmpty(t, got)
	require.Contains(t, got[0].Message, `literal "mesh-bar"`)
}

func TestLoadPreview_SelfLiteralWarning(t *testing.T) {
	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, nil, Options{})
	require.NoError(t, err)

	main := model.LayoutSourceFile{
		ID:      "/mesh/bar",
		Path:    "_layouts/mesh/bar.html",
		Content: `{{ block @lid() }}<div class="mesh-bar__nav"></div>{{ end }}`,
	}
	_, warnings := layouts.LoadPreview(main, nil)

	var found bool
	for _, w := range warnings {
		if strings.Contains(w, `literal "mesh-bar"`) {
			found = true
		}
	}
	require.True(t, found, "expected self-literal warning in preview, got %v", warnings)
}
