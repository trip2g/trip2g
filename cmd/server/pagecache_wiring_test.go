package main

import (
	"context"
	"testing"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"
	"trip2g/internal/pagecache"

	"github.com/stretchr/testify/require"
)

// emptyNoteLoaderEnv is a minimal noteloader.Env that loads zero notes/assets,
// so noteloader.Load succeeds without a database. It lets the wiring tests call
// the real (*app).PrepareLatestNotes / PrepareLiveNotes and observe the
// ClearPageCache() each performs after a successful reload.
type emptyNoteLoaderEnv struct{}

func (emptyNoteLoaderEnv) RawNotes(context.Context) ([]noteloader.RawNote, error) { return nil, nil }
func (emptyNoteLoaderEnv) RawAssets(context.Context) ([]noteloader.RawAsset, error) {
	return nil, nil
}
func (emptyNoteLoaderEnv) RawNoteChunks(context.Context) ([]noteloader.RawNoteChunk, error) {
	return nil, nil
}
func (emptyNoteLoaderEnv) NoteAssetExists(context.Context, db.NoteAsset) (bool, error) {
	return false, nil
}
func (emptyNoteLoaderEnv) NoteAssetURL(context.Context, db.NoteAsset) (model.PresignedURL, error) {
	return model.PresignedURL{}, nil
}
func (emptyNoteLoaderEnv) NoteAssetPath(db.NoteAsset) string { return "" }
func (emptyNoteLoaderEnv) PublicURL() string                 { return "https://example.com" }
func (emptyNoteLoaderEnv) Logger() logger.Logger             { return &logger.DummyLogger{} }
func (emptyNoteLoaderEnv) Now() time.Time                    { return time.Now() }
func (emptyNoteLoaderEnv) IsDevMode() bool                   { return false }
func (emptyNoteLoaderEnv) LoadFrontmatterPatches(context.Context) ([]frontmatterpatch.CompiledPatch, error) {
	return nil, nil
}
func (emptyNoteLoaderEnv) LoadSiteConfig(context.Context) (model.SiteConfig, error) {
	return model.SiteConfig{}, nil
}
func (emptyNoteLoaderEnv) ListAllSubgraphs(context.Context) ([]db.Subgraph, error) {
	return nil, nil
}

// appWithEmptyLoaders builds an app whose note loaders load nothing and whose
// page cache holds one entry. Calling a Prepare* method must empty the cache.
func appWithEmptyLoaders() *app {
	a := &app{appState: &appState{pageCache: pagecache.New()}}
	a.latestNoteLoader = noteloader.New("latest", emptyNoteLoaderEnv{}, mdloader.Config{})
	a.liveNoteLoader = noteloader.New("live", emptyNoteLoaderEnv{}, mdloader.Config{})
	return a
}

func seedOnePage(a *app) {
	a.StoreCachedPage(pagecache.Key{Path: "/seed", Host: "example.com"}, []byte("gz"))
}

// TestPrepareLatestNotes_ClearsPageCache pins the reload->invalidation wiring:
// PrepareLatestNotes (main.go) calls a.ClearPageCache() after a successful load,
// so the anonymous page cache is emptied. Removing that call leaves the stale
// entry and fails this test. partial=true skips the bleve search index.
func TestPrepareLatestNotes_ClearsPageCache(t *testing.T) {
	a := appWithEmptyLoaders()
	seedOnePage(a)
	require.Equal(t, 1, a.pageCache.Len())

	_, err := a.PrepareLatestNotes(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, 0, a.pageCache.Len(),
		"PrepareLatestNotes must ClearPageCache() after reload")
}

// TestPrepareLiveNotes_ClearsPageCache is the same wiring assertion for the live
// reload path: PrepareLiveNotes (main.go) calls a.ClearPageCache() after load.
func TestPrepareLiveNotes_ClearsPageCache(t *testing.T) {
	a := appWithEmptyLoaders()
	seedOnePage(a)
	require.Equal(t, 1, a.pageCache.Len())

	_, err := a.PrepareLiveNotes(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, a.pageCache.Len(),
		"PrepareLiveNotes must ClearPageCache() after reload")
}
