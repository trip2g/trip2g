package graph

import (
	"errors"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// TestResolveNotePathContent_FallsBackToStoredBytesOnMapMiss reproduces the
// data-loss bug: a path listed in notePaths with a non-empty latest_content_hash
// but absent from both Layouts().Map and LatestNoteViews().PathMap must still
// return its real stored bytes, never "". Returning "" makes the sync plugin
// wipe the local file.
func TestResolveNotePathContent_FallsBackToStoredBytesOnMapMiss(t *testing.T) {
	const path = "demo/_layouts/json-test.html.json"
	const realContent = `{"meta":{},"body":[{"type":"html","html":"<h1>real</h1>"}]}`

	obj := &db.NotePath{Value: path, LatestContentHash: "nonempty-hash"}
	layouts := &model.Layouts{Map: map[string]model.Layout{}}
	nvs := &model.NoteViews{PathMap: map[string]*model.NoteView{}}

	got, err := resolveNotePathContent(obj, layouts, nvs, func() (string, bool, error) {
		return realContent, true, nil
	})

	require.NoError(t, err)
	require.NotEmpty(t, got, "must never return empty content for a path with a non-empty latest_content_hash")
	require.JSONEq(t, realContent, got)
}

// TestResolveNotePathContent_RegisteredLayoutWins keeps the fast path: a path
// present in Layouts().Map returns its OriginalContent without hitting the DB.
func TestResolveNotePathContent_RegisteredLayoutWins(t *testing.T) {
	const path = "_layouts/index.html"
	const original = "<html>{{ yield body() }}</html>"

	obj := &db.NotePath{Value: path}
	layouts := &model.Layouts{Map: map[string]model.Layout{
		"index": {Path: path, OriginalContent: original},
	}}
	nvs := &model.NoteViews{PathMap: map[string]*model.NoteView{}}

	got, err := resolveNotePathContent(obj, layouts, nvs, func() (string, bool, error) {
		t.Fatal("fetchLatest must not be called when the layout is registered")
		return "", false, nil
	})

	require.NoError(t, err)
	require.Equal(t, original, got)
}

// TestResolveNotePathContent_FetchErrorPropagates ensures a DB error surfaces as
// an error (plugin skips on error) rather than being swallowed into "".
func TestResolveNotePathContent_FetchErrorPropagates(t *testing.T) {
	obj := &db.NotePath{Value: "demo/_layouts/json-test.html.json"}
	layouts := &model.Layouts{Map: map[string]model.Layout{}}
	nvs := &model.NoteViews{PathMap: map[string]*model.NoteView{}}

	wantErr := errors.New("db down")
	got, err := resolveNotePathContent(obj, layouts, nvs, func() (string, bool, error) {
		return "", false, wantErr
	})

	require.ErrorIs(t, err, wantErr)
	require.Empty(t, got)
}
