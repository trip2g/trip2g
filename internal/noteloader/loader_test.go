package noteloader_test

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg noteloader_test . Env

import (
	"context"
	"strings"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"
	"trip2g/internal/templateviews"

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

func TestLoader_ValidatesOnlyWhenEnabled(t *testing.T) {
	notes := []noteloader.RawNote{
		{Path: "note.md", PathID: 1, VersionID: 1, Content: "---\nvalidate: \"[[nope]]\"\n---\nbody\n"},
	}
	ctx := context.Background()

	plain := noteloader.New("test", makeMinimalEnv(notes, func() bool { return false }), mdloader.Config{})
	require.NoError(t, plain.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}))
	require.Empty(t, plain.NoteByPath("/note").Warnings)

	validating := noteloader.New("test", makeMinimalEnv(notes, func() bool { return false }), mdloader.Config{})
	validating.EnableValidation()
	require.NoError(t, validating.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}))
	expected := []model.NoteWarning{
		{Level: model.NoteWarningCritical, Message: "validate: target not found: [[nope]]"},
	}
	require.Equal(t, expected, validating.NoteByPath("/note").Warnings)
}

type cancelOnChartRows struct {
	cancel context.CancelFunc
}

func (c cancelOnChartRows) ChartRows(_ int64, _ model.NoteViewChart) model.ChartRowsResult {
	c.cancel()
	return model.ChartRowsResult{}
}

// A deadline that runs out while notes are parsed, validated and rendered must
// not fail the database reads: they all happen before mdloader.Load.
func TestLoader_DeadlineDuringMarkdownDoesNotFailDBReads(t *testing.T) {
	chart := "```datachart\n" +
		`{"data":{"source":"url","url":"http://x"},"config":{"series":[]}}` +
		"\n```\n"
	notes := []noteloader.RawNote{
		{Path: "note.md", PathID: 1, VersionID: 1, Content: chart},
	}

	env := makeMinimalEnv(notes, func() bool { return false })
	env.ListAllSubgraphsFunc = func(ctx context.Context) ([]db.Subgraph, error) {
		return nil, ctx.Err()
	}
	env.RawNoteChunksFunc = func(ctx context.Context) ([]noteloader.RawNoteChunk, error) {
		chunk := noteloader.RawNoteChunk{VersionID: 1, Path: "note.md", Embedding: []byte{0, 0, 128, 63}}
		return []noteloader.RawNoteChunk{chunk}, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loader := noteloader.New("test", env, mdloader.Config{})
	loader.SetChartDataProvider(cancelOnChartRows{cancel: cancel})

	require.NoError(t, loader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}))
	require.Error(t, ctx.Err(), "the chart must have been rendered inside Load")
	require.Len(t, loader.NoteChunks(), 1, "chunks must be read before the deadline")
}

// legacyRSSFeeds is the per-request scan RSSFeeds ran before the list was
// precomputed at load; the loader's result must match it exactly.
func legacyRSSFeeds(nvs *model.NoteViews) []model.RSSFeedLink {
	var feeds []model.RSSFeedLink
	for _, note := range templateviews.NewNVS(nvs, "live").List() {
		ct := strings.ToLower(strings.TrimSpace(note.M().GetString("content_type", "")))
		if !strings.HasPrefix(ct, "application/rss+xml") {
			continue
		}

		title := note.M().GetString("rss_title", "")
		if title == "" {
			title = note.Title()
		}
		if title == "" {
			title = "RSS"
		}

		feeds = append(feeds, model.RSSFeedLink{Title: title, Href: note.Permalink()})
	}
	return feeds
}

func TestLoader_RSSFeedsMatchLegacyScan(t *testing.T) {
	notes := []noteloader.RawNote{
		{Path: "feed.md", PathID: 1, VersionID: 1, Content: "---\ncontent_type: application/rss+xml; charset=utf-8\nrss_title: Main feed\n---\nx"},
		{Path: "blog/feed.md", PathID: 2, VersionID: 2, Content: "---\ncontent_type: \"  Application/RSS+XML \"\ntitle: Blog\n---\nx"},
		{Path: "news.md", PathID: 3, VersionID: 3, Content: "---\ncontent_type: application/rss+xml\n---\nx"},
		{Path: "_system/feed.md", PathID: 4, VersionID: 4, Content: "---\ncontent_type: application/rss+xml\n---\nx"},
		{Path: "atom.md", PathID: 5, VersionID: 5, Content: "---\ncontent_type: application/atom+xml\n---\nx"},
		{Path: "post.md", PathID: 6, VersionID: 6, Content: "---\ntitle: Post\n---\nx"},
	}

	env := makeMinimalEnv(notes, func() bool { return false })
	loader := noteloader.New("test", env, mdloader.Config{})
	require.NoError(t, loader.Load(context.Background(), noteloader.LoadOptions{SkipSearchIndex: true}))

	nvs := loader.NoteViews()
	want := legacyRSSFeeds(nvs)
	require.Len(t, want, 3)
	require.Equal(t, want, nvs.RSSFeeds)
	require.Equal(t, want, nvs.Copy().RSSFeeds)
}
