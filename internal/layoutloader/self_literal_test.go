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

func TestScanSelfLiteral_DidNotFlagged(t *testing.T) {
	// The @did check is gone: a dash id is just a word, and layouts named after an
	// HTML element (rss, table, main) matched their own tags. A hardcoded BEM class
	// no longer warns — @lid is the only self-literal still tracked.
	src := "line1\n" +
		`.@did__nav { color: red; }` + "\n" +
		`document.querySelector('.mesh-bar__nav')` + "\n" +
		`a .mesh-bar { }` + "\n" +
		`b .mesh-bar--mod { }` + "\n"
	ws := scanSelfLiteral(src, "/mesh/bar")
	require.Empty(t, ws)
}

func TestScanSelfLiteral_XMLRootElementNotFlagged(t *testing.T) {
	// The onboarding vault's _layouts/rss.html: <rss> is the RSS 2.0 root element,
	// not a reference to the layout's own name.
	src := `<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">` + "\n" +
		`</rss>` + "\n"
	ws := scanSelfLiteral(src, "/rss.html")
	require.Empty(t, ws)
}

func TestScanSelfLiteral_OtherComponentNotFlagged(t *testing.T) {
	// index.html composing other components by their expanded names is correct.
	src := `{{ yield mesh_bar() }}` + "\n"
	ws := scanSelfLiteral(src, "/mesh/index")
	require.Empty(t, ws)
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
	src := `{{ block mesh_bar() }}{{ yield mesh_bar() }}` + "\n"
	ws := scanSelfLiteral(src, "/mesh/bar")
	require.Len(t, ws, 1)
}

func TestScanSelfLiteral_HTMLCommentNotFlagged(t *testing.T) {
	// Usage documented in a comment is prose, not a real call site.
	src := `<!-- Render with {{ yield kanban() }} from any note. -->` + "\n"
	ws := scanSelfLiteral(src, "/kanban")
	require.Empty(t, ws)
}

func TestScanSelfLiteral_RealViolationStillCaughtNearComments(t *testing.T) {
	src := `<!-- Render with {{ yield kanban() }} from any note. -->` + "\n" +
		`{{ block kanban() }}` + "\n"
	ws := scanSelfLiteral(src, "/kanban")
	require.Len(t, warnLines(t, ws, "@lid"), 1)
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
		Content: `{{ block mesh_bar() }}<div class="@did__nav"></div>{{ end }}`,
	}}
	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, sources, Options{})
	require.NoError(t, err)
	got := layouts.Map["/mesh/bar"].Warnings
	require.NotEmpty(t, got)
	require.Contains(t, got[0].Message, `literal "mesh_bar"`)
}

func TestLoadPreview_SelfLiteralWarning(t *testing.T) {
	env := &testEnv{logger: &logger.TestLogger{}}
	layouts, err := Load(env, nil, Options{})
	require.NoError(t, err)

	main := model.LayoutSourceFile{
		ID:      "/mesh/bar",
		Path:    "_layouts/mesh/bar.html",
		Content: `{{ block mesh_bar() }}<div class="@did__nav"></div>{{ end }}`,
	}
	_, warnings := layouts.LoadPreview(main, nil)

	var found bool
	for _, w := range warnings {
		if strings.Contains(w, `literal "mesh_bar"`) {
			found = true
		}
	}
	require.True(t, found, "expected self-literal warning in preview, got %v", warnings)
}
