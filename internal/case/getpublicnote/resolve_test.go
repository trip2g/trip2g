package getpublicnote_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/getpublicnote"
	"trip2g/internal/case/rendernotepage"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func ptr[T any](v T) *T { return &v }

func scopedCtx(readPatterns []string) context.Context {
	return appreq.NewContext(context.Background(), &appreq.Request{
		WebhookScoped:       true,
		WebhookReadPatterns: readPatterns,
	})
}

func noteViews(views ...*appmodel.NoteView) *appmodel.NoteViews {
	nv := &appmodel.NoteViews{Map: map[string]*appmodel.NoteView{}}
	for _, v := range views {
		nv.List = append(nv.List, v)
		nv.Map[v.Permalink] = v
	}
	return nv
}

func TestResolve_PathIDHit(t *testing.T) {
	view := &appmodel.NoteView{PathID: 7, Path: "boards/task.md", Permalink: "/task"}
	var gotPath string
	env := &getpublicnote.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return noteViews(view) },
		RenderNotePageFunc: func(ctx context.Context, req rendernotepage.Request) (*rendernotepage.Response, error) {
			gotPath = req.Path
			return &rendernotepage.Response{Note: &appmodel.NoteView{PathID: 7, Permalink: "/task"}}, nil
		},
	}

	out, err := getpublicnote.Resolve(context.Background(), env, model.NoteInput{PathID: ptr(int64(7))}, nil)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, "/task", gotPath)
	require.Equal(t, int64(7), out.PathID)
}

func TestResolve_PathIDMiss(t *testing.T) {
	view := &appmodel.NoteView{PathID: 1, Path: "a.md", Permalink: "/a"}
	var gotPath string
	env := &getpublicnote.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return noteViews(view) },
		RenderNotePageFunc: func(ctx context.Context, req rendernotepage.Request) (*rendernotepage.Response, error) {
			gotPath = req.Path
			return &rendernotepage.Response{Note: nil}, nil
		},
	}

	out, err := getpublicnote.Resolve(context.Background(), env, model.NoteInput{PathID: ptr(int64(999))}, nil)
	require.NoError(t, err)
	require.Nil(t, out)
	require.Equal(t, "", gotPath) // no view matched, no input.Path -> empty path
}

func TestResolve_ScopedAllowed(t *testing.T) {
	view := &appmodel.NoteView{PathID: 7, Path: "boards/task.md", Permalink: "/task"}
	called := false
	env := &getpublicnote.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return noteViews(view) },
		RenderNotePageFunc: func(ctx context.Context, req rendernotepage.Request) (*rendernotepage.Response, error) {
			called = true
			return &rendernotepage.Response{Note: &appmodel.NoteView{PathID: 7}}, nil
		},
	}

	out, err := getpublicnote.Resolve(scopedCtx([]string{"boards/**"}), env, model.NoteInput{PathID: ptr(int64(7))}, nil)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.True(t, called)
}

func TestResolve_ScopedDenied(t *testing.T) {
	view := &appmodel.NoteView{PathID: 7, Path: "boards/task.md", Permalink: "/task"}
	env := &getpublicnote.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return noteViews(view) },
		RenderNotePageFunc: func(ctx context.Context, req rendernotepage.Request) (*rendernotepage.Response, error) {
			t.Fatal("RenderNotePage must not be called when scope denies")
			return nil, nil
		},
	}

	out, err := getpublicnote.Resolve(scopedCtx([]string{"posts/**"}), env, model.NoteInput{PathID: ptr(int64(7))}, nil)
	require.NoError(t, err)
	require.Nil(t, out)
}

func TestResolve_ScopedEmptyPatternsFailClosed(t *testing.T) {
	view := &appmodel.NoteView{PathID: 7, Path: "boards/task.md", Permalink: "/task"}
	env := &getpublicnote.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return noteViews(view) },
		RenderNotePageFunc: func(ctx context.Context, req rendernotepage.Request) (*rendernotepage.Response, error) {
			t.Fatal("RenderNotePage must not be called when read_patterns empty")
			return nil, nil
		},
	}

	out, err := getpublicnote.Resolve(scopedCtx(nil), env, model.NoteInput{PathID: ptr(int64(7))}, nil)
	require.NoError(t, err)
	require.Nil(t, out)
}

func TestResolve_PathFallbackWhenFsPathUnknown(t *testing.T) {
	// input.Path supplied but not present in NoteViews.Map -> fsPath unknown,
	// scope check falls back to the URL path.
	var gotCheckAllowed bool
	env := &getpublicnote.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return noteViews() },
		RenderNotePageFunc: func(ctx context.Context, req rendernotepage.Request) (*rendernotepage.Response, error) {
			gotCheckAllowed = true
			return &rendernotepage.Response{Note: &appmodel.NoteView{PathID: 3}}, nil
		},
	}

	out, err := getpublicnote.Resolve(scopedCtx([]string{"**"}), env, model.NoteInput{Path: ptr("/unknown-url")}, nil)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.True(t, gotCheckAllowed)
}

func TestResolve_RenderError(t *testing.T) {
	env := &getpublicnote.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return noteViews() },
		RenderNotePageFunc: func(ctx context.Context, req rendernotepage.Request) (*rendernotepage.Response, error) {
			return nil, errors.New("boom")
		},
	}

	out, err := getpublicnote.Resolve(context.Background(), env, model.NoteInput{Path: ptr("/x")}, nil)
	require.Error(t, err)
	require.Nil(t, out)
}
