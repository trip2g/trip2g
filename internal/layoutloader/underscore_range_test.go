package layoutloader

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

// `{{ range _, x := ... }}` used to panic Jet's AST walker with
// "unexpected node _", which load() turned into a critical layout warning.
// The walker now treats _ as a leaf, so such layouts load like any other —
// as the page itself, as a component walked for other pages, as an import
// and in preview.
func TestLoadUnderscoreRangeLoads(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{
			ID:      "/under/index",
			Path:    "_layouts/under/index.html",
			Content: `<html>{{ range _, x := note.List() }}{{ x }}{{ end }}</html>`,
		},
		{
			ID:      "/good/index",
			Path:    "_layouts/good/index.html",
			Content: `<html>ok</html>`,
		},
	}

	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)

	for _, id := range []string{"/under/index", "/good/index"} {
		require.NotNil(t, layouts.Map[id].View, id)
		require.Empty(t, layouts.Map[id].Warnings, id)
	}
}

func TestLoadUnderscoreRangeImported(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{
			ID:      "/page/index",
			Path:    "_layouts/page/index.html",
			Content: `{{ import "/under/index" }}<html>{{ yield list(items=items) }}</html>`,
		},
		{
			ID:      "/under/index",
			Path:    "_layouts/under/index.html",
			Content: `{{ block list(items=nil) }}{{ range _, x := items }}[{{ x }}]{{ end }}{{ end }}`,
		},
	}

	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)

	vars := jet.VarMap{}
	vars.Set("items", []string{"a", "b"})
	out, err := renderWith(t, layouts, "/page/index", vars)
	require.NoError(t, err)
	require.Contains(t, out, "<html>[a][b]</html>")
}

func TestLoadPreviewUnderscoreRange(t *testing.T) {
	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, nil, Options{})
	require.NoError(t, err)

	layout, warnings := layouts.LoadPreview(model.LayoutSourceFile{
		ID:      "/under/index",
		Path:    "/_layouts/under/index.html",
		Content: `<html>{{ range _, x := note.List() }}{{ x }}{{ end }}</html>`,
	}, nil)

	require.NotNil(t, layout.View)
	require.Empty(t, warnings)
}
