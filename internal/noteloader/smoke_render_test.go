package noteloader

import (
	"strings"
	"testing"
	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type smokeTestEnv struct{ log logger.Logger }

func (e *smokeTestEnv) Logger() logger.Logger { return e.log }

func smokeLoadLayouts(t *testing.T, sources []model.LayoutSourceFile) *model.Layouts {
	t.Helper()
	layouts, err := layoutloader.Load(&smokeTestEnv{log: &logger.TestLogger{}}, sources, layoutloader.Options{})
	require.NoError(t, err)
	return layouts
}

// Layout parses but blows up at render time (unknown field on note).
// Smoke render must turn that into a NoteWarning on the layout.
func TestSmokeRenderLayouts_RuntimeErrorBecomesWarning(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/buggy", Path: "_layouts/buggy.html",
			Content: `<html>{{ note.NoSuchField }}</html>`},
	}
	layouts := smokeLoadLayouts(t, sources)
	require.NotNil(t, layouts.Map["/buggy"].View, "layout should parse OK")

	nvs := &model.NoteViews{List: []*model.NoteView{
		{Path: "page.md", Layout: "buggy", Title: "p"},
	}}

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	warnings := layouts.Map["/buggy"].Warnings
	require.NotEmpty(t, warnings, "expected smoke render warning")

	var found bool
	for _, w := range warnings {
		if strings.Contains(w.Message, "smoke render") {
			found = true
		}
	}
	require.True(t, found, "expected 'smoke render' warning, got: %+v", warnings)
}

// No note uses this layout — smoke must skip it (no spurious warnings).
func TestSmokeRenderLayouts_NoMatchingNotes_NoWarning(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/orphan", Path: "_layouts/orphan.html",
			Content: `<html>{{ note.NoSuchField }}</html>`},
	}
	layouts := smokeLoadLayouts(t, sources)

	nvs := &model.NoteViews{List: []*model.NoteView{
		{Path: "page.md", Layout: "other", Title: "p"},
	}}

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	require.Empty(t, layouts.Map["/orphan"].Warnings,
		"no notes => no smoke run => no warning")
}

// Limit must cap how many notes we render per layout. A trigger note placed
// beyond the limit must NOT cause a warning.
func TestSmokeRenderLayouts_LimitFirstN(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/cond", Path: "_layouts/cond.html",
			Content: `{{ if note.Title == "trigger" }}{{ note.NoSuchField }}{{ end }}`},
	}
	layouts := smokeLoadLayouts(t, sources)

	notes := make([]*model.NoteView, 0, 12)
	for range 10 {
		notes = append(notes, &model.NoteView{
			Path: "ok.md", Layout: "cond", Title: "ok",
		})
	}
	notes = append(notes, &model.NoteView{
		Path: "boom.md", Layout: "cond", Title: "trigger",
	})
	nvs := &model.NoteViews{List: notes}

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	require.Empty(t, layouts.Map["/cond"].Warnings,
		"11th note must not be smoke-tested when limit=10")
}

// Parse-broken layout already has Critical warning and View == nil.
// Smoke must not touch it (no duplicate / crash).
func TestSmokeRenderLayouts_SkipsLayoutsWithParseError(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/broken", Path: "_layouts/broken.html",
			Content: `<html>{{ unterminated`},
	}
	layouts := smokeLoadLayouts(t, sources)
	require.Nil(t, layouts.Map["/broken"].View, "should be nil after parse error")
	parseWarnings := len(layouts.Map["/broken"].Warnings)
	require.Positive(t, parseWarnings, "should have parse-time warning")

	nvs := &model.NoteViews{List: []*model.NoteView{
		{Path: "p.md", Layout: "broken", Title: "p"},
	}}

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	require.Len(t, layouts.Map["/broken"].Warnings, parseWarnings,
		"smoke must not add warnings to layouts with parse errors")
}
