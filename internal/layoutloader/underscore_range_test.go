package layoutloader

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// `{{ range _, x := ... }}` parses in Jet but its AST walker panics with
// "unexpected node _". The panic must be contained per-layout: the bad layout
// gets a critical warning, other layouts load, Load does not panic.
// Regression for: one bad layout crashing the whole pushNotes batch.
func TestLoadUnderscoreRangePanicContained(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{
			ID:      "/bad/index",
			Path:    "_layouts/bad/index.html",
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

	bad := layouts.Map["/bad/index"]
	require.Nil(t, bad.View)
	require.NotEmpty(t, bad.Warnings)
	require.Equal(t, model.NoteWarningCritical, bad.Warnings[0].Level)
	require.Contains(t, bad.Warnings[0].Message, "unexpected node _")

	good := layouts.Map["/good/index"]
	require.NotNil(t, good.View)
	require.Empty(t, good.Warnings)
}

// A broken layout must not poison OTHER layouts: buildBlockRegistry walks
// every other template's AST during each page's load, so with the good page
// first the bad component's walker panic would surface inside the good page.
func TestLoadUnderscoreRangeDoesNotPoisonOtherLayouts(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{
			ID:      "/good/index",
			Path:    "_layouts/good/index.html",
			Content: `<html>ok</html>`,
		},
		{
			ID:      "/bad/index",
			Path:    "_layouts/bad/index.html",
			Content: `<html>{{ range _, x := note.List() }}{{ x }}{{ end }}</html>`,
		},
	}

	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)

	good := layouts.Map["/good/index"]
	require.NotNil(t, good.View)
	require.Empty(t, good.Warnings)

	require.Nil(t, layouts.Map["/bad/index"].View)
}

// A page explicitly importing a broken layout must not crash Load: the
// imported-template walks in processTemplates hit the same walker panic.
func TestLoadUnderscoreRangeImportedPanicContained(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{
			ID:      "/page/index",
			Path:    "_layouts/page/index.html",
			Content: `{{ import "/bad/index" }}<html>ok</html>`,
		},
		{
			ID:      "/bad/index",
			Path:    "_layouts/bad/index.html",
			Content: `{{ range _, x := note.List() }}{{ x }}{{ end }}`,
		},
	}

	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
	require.NoError(t, err)
	require.NotNil(t, layouts.Map["/page/index"].View)
	require.Nil(t, layouts.Map["/bad/index"].View)
}

// The same panic must be contained in the preview compile path.
func TestLoadPreviewUnderscoreRangePanicContained(t *testing.T) {
	layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, nil, Options{})
	require.NoError(t, err)

	layout, warnings := layouts.LoadPreview(model.LayoutSourceFile{
		ID:      "/bad/index",
		Path:    "/_layouts/bad/index.html",
		Content: `<html>{{ range _, x := note.List() }}{{ x }}{{ end }}</html>`,
	}, nil)

	require.Nil(t, layout.View)
	require.NotEmpty(t, warnings)
	require.Contains(t, warnings[0], "unexpected node _")
}
