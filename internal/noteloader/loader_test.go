package noteloader_test

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg noteloader_test . Env

import (
	"context"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"

	"github.com/stretchr/testify/require"
)

// makeMinimalEnv returns an EnvMock wired with no-op defaults.
// Override RequireSigninFunc to simulate DB subgraph changes between loads.
func makeMinimalEnv(notes []noteloader.RawNote, requireSignin func() bool) *EnvMock {
	return &EnvMock{
		RawNotesFunc:        func(_ context.Context) ([]noteloader.RawNote, error) { return notes, nil },
		RawAssetsFunc:       func(_ context.Context) ([]noteloader.RawAsset, error) { return nil, nil },
		RawNoteChunksFunc:   func(_ context.Context) ([]noteloader.RawNoteChunk, error) { return nil, nil },
		NoteAssetExistsFunc: func(_ context.Context, _ db.NoteAsset) (bool, error) { return false, nil },
		NoteAssetPathFunc:   func(_ db.NoteAsset) string { return "" },
		PublicURLFunc:       func() string { return "https://example.com" },
		LoggerFunc:          func() logger.Logger { return &logger.TestLogger{} },
		IsDevModeFunc:       func() bool { return false },
		LoadFrontmatterPatchesFunc: func(_ context.Context) ([]frontmatterpatch.CompiledPatch, error) {
			return nil, nil
		},
		LoadSiteConfigFunc: func(_ context.Context) (model.SiteConfig, error) {
			return model.SiteConfig{}, nil
		},
		ListAllSubgraphsFunc: func(_ context.Context) ([]db.Subgraph, error) {
			return []db.Subgraph{{Name: "mysg", RequireSignin: requireSignin()}}, nil
		},
	}
}

// TestLoader_RawFiles_DontPanicSearchIndex is the regression test for a
// nil-AST panic in buildSearchIndex when the vault contains .canvas, .base
// or .excalidraw files. Those NoteViews are stored verbatim (no markdown
// parse), so Ast() is nil — feeding it to extractText panicked. Search
// index must skip them gracefully and still index the .md notes alongside.
func TestLoader_RawFiles_DontPanicSearchIndex(t *testing.T) {
	notes := []noteloader.RawNote{
		{
			Path:      "intro.md",
			PathID:    1,
			VersionID: 1,
			Content:   "---\ntitle: Intro\n---\nReal markdown body.",
		},
		{
			Path:      "diagram.canvas",
			PathID:    2,
			VersionID: 2,
			Content:   `{"nodes":[{"id":"a","type":"text","text":"hi","x":0,"y":0,"width":50,"height":50}],"edges":[]}`,
		},
		{
			Path:      "data.base",
			PathID:    3,
			VersionID: 3,
			Content:   "views:\n  - type: table\n    name: T\n",
		},
		{
			Path:      "sketch.excalidraw",
			PathID:    4,
			VersionID: 4,
			Content:   `{"type":"excalidraw","version":2,"source":"x","elements":[],"appState":{},"files":{}}`,
		},
	}

	env := makeMinimalEnv(notes, func() bool { return false })
	loader := noteloader.New("test", env, mdloader.Config{})

	// SkipSearchIndex must be false so buildSearchIndex actually runs.
	require.NotPanics(t, func() {
		require.NoError(t, loader.Load(context.Background(), noteloader.LoadOptions{}))
	})

	nvs := loader.NoteViews()

	require.NotNil(t, nvs.PathMap["intro.md"], "markdown note should be in PathMap")
	require.NotNil(t, nvs.PathMap["diagram.canvas"], "canvas should be in PathMap")
	require.NotNil(t, nvs.PathMap["data.base"], "base should be in PathMap")
	require.NotNil(t, nvs.PathMap["sketch.excalidraw"], "excalidraw should be in PathMap")

	// Raw files keep their bytes verbatim — no markdown processing.
	require.Equal(t, notes[1].Content, string(nvs.PathMap["diagram.canvas"].Content))
	require.Equal(t, notes[2].Content, string(nvs.PathMap["data.base"].Content))
	require.Equal(t, notes[3].Content, string(nvs.PathMap["sketch.excalidraw"].Content))

	// Raw files have no AST — assert that's still the case (guards rely on it).
	require.Nil(t, nvs.PathMap["diagram.canvas"].Ast())
	require.Nil(t, nvs.PathMap["data.base"].Ast())
	require.Nil(t, nvs.PathMap["sketch.excalidraw"].Ast())
}

// TestLoader_RequireSignin_UpdatesAfterCacheHit is the regression test for the
// stale-pointer bug: when note content is unchanged (NoteCache returns the old
// NoteView), a subsequent Load must still propagate the updated RequireSignin
// flag from DB into the note's Subgraphs map.
func TestLoader_RequireSignin_UpdatesAfterCacheHit(t *testing.T) {
	notes := []noteloader.RawNote{
		{
			Path:      "wall.md",
			PathID:    1,
			VersionID: 1,
			Content:   "---\nsubgraphs: mysg\nfree: false\ntitle: Wall\n---\nContent unchanged",
		},
	}

	signinEnabled := false
	env := makeMinimalEnv(notes, func() bool { return signinEnabled })
	loader := noteloader.New("test", env, mdloader.Config{})
	ctx := context.Background()

	// First load: subgraph has RequireSignin=false
	require.NoError(t, loader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}))
	nv1 := loader.NoteViews()
	require.Len(t, nv1.List, 1)
	require.NotNil(t, nv1.List[0].Subgraphs["mysg"])
	require.False(t, nv1.List[0].Subgraphs["mysg"].RequireSignin)

	// Simulate updateSubgraph(requireSignin=true) — note content is unchanged.
	signinEnabled = true

	// Second load: NoteCache hits (same content) → old NoteView reused.
	// Bug: without rebindSubgraphPointers, the stale pointer keeps RequireSignin=false.
	require.NoError(t, loader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}))
	nv2 := loader.NoteViews()
	require.Len(t, nv2.List, 1)
	require.NotNil(t, nv2.List[0].Subgraphs["mysg"])
	require.True(t, nv2.List[0].Subgraphs["mysg"].RequireSignin,
		"RequireSignin must be true after subgraph update even when note content is cached")
}
