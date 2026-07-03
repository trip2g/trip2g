package renderpreview

import (
	"context"
	"reflect"
	"testing"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"

	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

type testEnv struct {
	log     logger.Logger
	nvs     *model.NoteViews
	preview func(main model.LayoutSourceFile, extra map[string]string) (model.Layout, []string)
	buffer  *PreviewBuffer
}

func (e *testEnv) Logger() logger.Logger             { return e.log }
func (e *testEnv) LatestNoteViews() *model.NoteViews { return e.nvs }
func (e *testEnv) PreviewBuffer() *PreviewBuffer     { return e.buffer }
func (e *testEnv) LoadPreviewLayout(main model.LayoutSourceFile, extra map[string]string) (model.Layout, []string) {
	return e.preview(main, extra)
}

type loaderEnv struct{ log logger.Logger }

func (e *loaderEnv) Logger() logger.Logger { return e.log }
func (e *loaderEnv) IsDevMode() bool       { return false }

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	layouts, err := layoutloader.Load(&loaderEnv{log: &logger.TestLogger{}}, nil, layoutloader.Options{})
	require.NoError(t, err)
	return &testEnv{
		log:     &logger.TestLogger{},
		nvs:     &model.NoteViews{},
		preview: layouts.LoadPreview,
		buffer:  NewPreviewBuffer(DefaultConfig()),
	}
}

func src(s string) *string { return &s }

// A layout that panics at render time (Jet re-panics runtime.Error and
// non-error panics from Execute) must return a clean warning payload, not
// crash the process. Regression for: preview panic killing the server.
func TestResolveRenderPanicRecovered(t *testing.T) {
	env := newTestEnv(t)
	// Build a view whose execution panics with a non-error value, which Jet's
	// own Execute recover re-panics instead of returning as error (same for
	// runtime.Error panics like out-of-range indexing).
	loader := jet.NewInMemLoader()
	loader.Set("/boom", `{{ boom() }}`)
	set := jet.NewSet(loader)
	set.AddGlobalFunc("boom", func(a jet.Arguments) reflect.Value {
		panic("boom: exec-time panic") // non-error panic -> Jet Execute re-panics it
	})
	view, err := set.GetTemplate("/boom")
	require.NoError(t, err)

	env.preview = func(model.LayoutSourceFile, map[string]string) (model.Layout, []string) {
		return model.Layout{View: view}, nil
	}

	payload, err := Resolve(context.Background(), env, graphmodel.RenderLayoutInput{
		Layout: &graphmodel.RenderLayoutFileInput{Path: "boom.html", Src: src(`{{ boom() }}`)},
	})
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.NotEmpty(t, payload.Warnings.Layout)
	require.Contains(t, payload.Warnings.Layout[0], "panic")
}

// Preview must expose the same defaultTemplate / currentUser / title vars the
// real page render injects, so layouts using them can be previewed.
func TestResolveInjectsUserSpaceVars(t *testing.T) {
	env := newTestEnv(t)

	payload, err := Resolve(context.Background(), env, graphmodel.RenderLayoutInput{
		Layout: &graphmodel.RenderLayoutFileInput{
			Path: "parity/index.html",
			Src: src(`<head>{{ defaultTemplate.Styles() }}{{ title }}</head>` +
				`<body>{{ defaultTemplate.Header() }}admin:{{ currentUser.IsAdmin() }}` +
				`{{ defaultTemplate.Footer() }}{{ defaultTemplate.UserSpaceScripts() }}</body>`),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Empty(t, payload.Warnings.Layout)

	entry, ok := env.buffer.GetByID(payload.PreviewID)
	require.True(t, ok)
	require.Contains(t, entry.HTML, "admin:false")
}
