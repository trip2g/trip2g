package embedguard_test

import (
	"context"
	"testing"

	"trip2g/internal/embedguard"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type readerFunc func(note *model.NoteView) bool

func (f readerFunc) CanReadNote(_ context.Context, note *model.NoteView) (bool, error) {
	return f(note), nil
}

func views(notes ...*model.NoteView) *model.NoteViews {
	nvs := model.NewNoteViews()
	for _, note := range notes {
		nvs.PathMap[note.Path] = note
	}

	return nvs
}

func TestViewerJudgesAtRequestTimeNotAtRenderTime(t *testing.T) {
	members := &model.NoteView{
		Path: "members.md", PathID: 2, Title: "Members", Permalink: "/members",
		SubgraphNames: []string{"members"},
	}
	page := embedguard.Wrap(2, "zzmembersonlyzz")

	signedOut := readerFunc(func(note *model.NoteView) bool { return note.IsAnonymouslyReadable() })
	signedIn := readerFunc(func(note *model.NoteView) bool { return true })

	out := embedguard.NewViewer(context.Background(), signedOut, views(members)).Strip(page)
	require.NotContains(t, out, "zzmembersonlyzz")
	require.Contains(t, out, `<a href="/members">Members</a>`)

	out = embedguard.NewViewer(context.Background(), signedIn, views(members)).Strip(page)
	require.Equal(t, "zzmembersonlyzz", out)
}

func TestViewerUnknownNoteIsRefused(t *testing.T) {
	page := embedguard.Wrap(99, "zzsecretzz")

	out := embedguard.NewViewer(context.Background(), readerFunc(func(*model.NoteView) bool { return true }), views()).Strip(page)

	require.Empty(t, out)
}

func TestNilViewerRefusesEveryEmbed(t *testing.T) {
	var viewer *embedguard.Viewer

	require.Equal(t, "ab", viewer.Strip("a"+embedguard.Wrap(1, "secret")+"b"))
}

func TestViewerPartialsAreOpenOnlyWhenAskedFor(t *testing.T) {
	partial := &model.NoteView{Path: "_footer_note.md", PathID: 3, Title: "Partial", Permalink: "/_footer_note"}
	page := embedguard.Wrap(3, "partial text")
	denyAll := readerFunc(func(*model.NoteView) bool { return false })

	strict := embedguard.NewViewer(context.Background(), denyAll, views(partial)).Strip(page)
	require.NotContains(t, strict, "partial text")

	open := embedguard.NewViewer(context.Background(), denyAll, views(partial)).WithOpenPartials().Strip(page)
	require.Equal(t, "partial text", open)
}

func TestViewerOpenPartialsStillJudgeWhatThePartialEmbeds(t *testing.T) {
	partial := &model.NoteView{Path: "_p.md", PathID: 3, Title: "P", Permalink: "/_p"}
	closed := &model.NoteView{Path: "closed.md", PathID: 4, Title: "Closed", Permalink: "/closed"}
	page := embedguard.Wrap(3, "partial text "+embedguard.Wrap(4, "zzsecretzz"))
	denyAll := readerFunc(func(*model.NoteView) bool { return false })

	out := embedguard.NewViewer(context.Background(), denyAll, views(partial, closed)).WithOpenPartials().Strip(page)

	require.Contains(t, out, "partial text")
	require.NotContains(t, out, "zzsecretzz")
}

func TestViewerPartialWithASubgraphGateIsNotOpen(t *testing.T) {
	gated := &model.NoteView{Path: "_g.md", PathID: 3, Title: "G", Permalink: "/_g", SubgraphNames: []string{"paid"}}
	denyAll := readerFunc(func(*model.NoteView) bool { return false })

	out := embedguard.NewViewer(context.Background(), denyAll, views(gated)).WithOpenPartials().Strip(embedguard.Wrap(3, "zzpaidzz"))

	require.NotContains(t, out, "zzpaidzz")
}
