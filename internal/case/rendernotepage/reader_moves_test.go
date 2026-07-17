package rendernotepage_test

import (
	"testing"

	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

func readerMoveTestViews() *model.NoteViews {
	note, views := cacheTestNote()
	note.Free = false // authenticated render path, no anonymous page cache

	referrer := &model.NoteView{
		Path:          "/ref-note",
		Title:         "Referrer Note",
		PathID:        2,
		VersionID:     43,
		Content:       []byte("# Ref"),
		HTML:          "<h1>Ref</h1>",
		Permalink:     "/ref-note",
		InLinks:       map[string]struct{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]*model.NoteAssetReplace{},
	}
	views.Map["/ref-note"] = referrer
	views.List = append(views.List, referrer)
	return views
}

// Idle property: with no movement subscribers the render path must do nothing
// beyond the ReaderMovesActive check. PublishReaderMoveFunc is intentionally
// left nil — any call would panic the test.
func TestReaderMove_IdleSkipsPublish(t *testing.T) {
	views := readerMoveTestViews()
	env, _, _ := cacheTestEnv(views, nil)
	// cacheTestEnv stubs ReaderMovesActiveFunc to false; PublishReaderMoveFunc stays nil.

	ctx := newReqCtx(reqOpts{})
	ctx.Request.Header.Set("Referer", "https://example.com/ref-note")
	runHandle(t, env, ctx, &usertoken.Data{ID: 7})

	require.NotEmpty(t, env.ReaderMovesActiveCalls(), "render path must consult the subscriber-count guard")
	require.Empty(t, env.PublishReaderMoveCalls(), "no publish when nobody is subscribed")
}

// With a subscriber attached, a signed-in render publishes the path-level
// from→to pair and the raw viewer id (anonymized later, inside the bus).
func TestReaderMove_PublishesFromToPair(t *testing.T) {
	views := readerMoveTestViews()
	env, _, _ := cacheTestEnv(views, nil)
	env.ReaderMovesActiveFunc = func() bool { return true }
	env.PublishReaderMoveFunc = func(fromPathID *int64, toPathID int64, viewerID string) {}

	ctx := newReqCtx(reqOpts{})
	ctx.Request.Header.Set("Referer", "https://example.com/ref-note")
	runHandle(t, env, ctx, &usertoken.Data{ID: 7})

	calls := env.PublishReaderMoveCalls()
	require.Len(t, calls, 1)
	require.NotNil(t, calls[0].FromPathID)
	require.EqualValues(t, 2, *calls[0].FromPathID)
	require.EqualValues(t, 1, calls[0].ToPathID)
	require.Equal(t, "7", calls[0].ViewerID)
}

// A direct entry (no referrer, or referrer outside the KB) publishes a nil
// FromPathID — the entry-edge signal.
func TestReaderMove_EntryHasNilFrom(t *testing.T) {
	views := readerMoveTestViews()
	env, _, _ := cacheTestEnv(views, nil)
	env.ReaderMovesActiveFunc = func() bool { return true }
	env.PublishReaderMoveFunc = func(fromPathID *int64, toPathID int64, viewerID string) {}

	ctx := newReqCtx(reqOpts{})
	ctx.Request.Header.Set("Referer", "https://google.com/search")
	runHandle(t, env, ctx, &usertoken.Data{ID: 7})

	calls := env.PublishReaderMoveCalls()
	require.Len(t, calls, 1)
	require.Nil(t, calls[0].FromPathID)
	require.EqualValues(t, 1, calls[0].ToPathID)
}
