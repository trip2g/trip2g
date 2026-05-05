package noteloader_test

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg noteloader_test . Env

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
		NoteAssetURLFunc: func(_ context.Context, _ db.NoteAsset) (model.PresignedURL, error) {
			return model.PresignedURL{}, nil
		},
		NoteAssetPathFunc: func(_ db.NoteAsset) string { return "" },
		PublicURLFunc:     func() string { return "https://example.com" },
		LoggerFunc:        func() logger.Logger { return &logger.TestLogger{} },
		NowFunc:           time.Now,
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
