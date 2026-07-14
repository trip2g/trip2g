package resolvewikilinks_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/resolvewikilinks"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func scopedCtx(readPatterns []string) context.Context {
	return appreq.NewContext(context.Background(), &appreq.Request{
		WebhookScoped:       true,
		WebhookReadPatterns: readPatterns,
	})
}

// viewsResolvingTo builds NoteViews where bare link "foo" resolves to target.
func viewsResolvingTo(target *appmodel.NoteView) *appmodel.NoteViews {
	return &appmodel.NoteViews{
		Map:         map[string]*appmodel.NoteView{},
		BasenameMap: map[string][]*appmodel.NoteView{"foo": {target}},
	}
}

func TestResolve_NilNoteViews(t *testing.T) {
	env := &resolvewikilinks.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nil },
	}
	out, err := resolvewikilinks.Resolve(context.Background(), env, model.ResolveWikilinksFilter{Links: []string{"foo"}})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "foo", out[0].Link)
	require.Nil(t, out[0].Path)
	require.Nil(t, out[0].URL)
}

func TestResolve_Unresolved(t *testing.T) {
	env := &resolvewikilinks.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return &appmodel.NoteViews{Map: map[string]*appmodel.NoteView{}, BasenameMap: map[string][]*appmodel.NoteView{}}
		},
	}
	out, err := resolvewikilinks.Resolve(context.Background(), env, model.ResolveWikilinksFilter{Links: []string{"missing"}})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Nil(t, out[0].Path)
	require.Nil(t, out[0].URL)
}

func TestResolve_ResolvedPermalink(t *testing.T) {
	target := &appmodel.NoteView{Path: "boards/foo.md", Permalink: "/foo", PermalinkOriginal: "/foo-orig"}
	env := &resolvewikilinks.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return viewsResolvingTo(target) },
	}
	out, err := resolvewikilinks.Resolve(context.Background(), env, model.ResolveWikilinksFilter{Links: []string{"foo"}})
	require.NoError(t, err)
	require.NotNil(t, out[0].Path)
	require.Equal(t, "boards/foo.md", *out[0].Path)
	require.NotNil(t, out[0].URL)
	require.Equal(t, "/foo", *out[0].URL) // Slug empty -> Permalink
}

func TestResolve_ResolvedSlugUsesPermalinkOriginal(t *testing.T) {
	target := &appmodel.NoteView{Path: "boards/foo.md", Slug: "foo", Permalink: "/foo", PermalinkOriginal: "/foo-orig"}
	env := &resolvewikilinks.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return viewsResolvingTo(target) },
	}
	out, err := resolvewikilinks.Resolve(context.Background(), env, model.ResolveWikilinksFilter{Links: []string{"foo"}})
	require.NoError(t, err)
	require.NotNil(t, out[0].URL)
	require.Equal(t, "/foo-orig", *out[0].URL) // Slug set -> PermalinkOriginal
}

func TestResolve_ScopedAllowed(t *testing.T) {
	target := &appmodel.NoteView{Path: "boards/foo.md", Permalink: "/foo"}
	env := &resolvewikilinks.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return viewsResolvingTo(target) },
	}
	out, err := resolvewikilinks.Resolve(scopedCtx([]string{"boards/**"}), env, model.ResolveWikilinksFilter{Links: []string{"foo"}})
	require.NoError(t, err)
	require.NotNil(t, out[0].Path)
	require.NotNil(t, out[0].URL)
}

func TestResolve_ScopedDeniedDoesNotLeakExistence(t *testing.T) {
	target := &appmodel.NoteView{Path: "secret/foo.md", Permalink: "/foo"}
	env := &resolvewikilinks.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return viewsResolvingTo(target) },
	}
	out, err := resolvewikilinks.Resolve(scopedCtx([]string{"boards/**"}), env, model.ResolveWikilinksFilter{Links: []string{"foo"}})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "foo", out[0].Link)
	require.Nil(t, out[0].Path) // existence must not leak
	require.Nil(t, out[0].URL)
}

func TestResolve_ScopedEmptyPatternsFailClosed(t *testing.T) {
	target := &appmodel.NoteView{Path: "boards/foo.md", Permalink: "/foo"}
	env := &resolvewikilinks.EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return viewsResolvingTo(target) },
	}
	out, err := resolvewikilinks.Resolve(scopedCtx(nil), env, model.ResolveWikilinksFilter{Links: []string{"foo"}})
	require.NoError(t, err)
	require.Nil(t, out[0].Path)
	require.Nil(t, out[0].URL)
}
