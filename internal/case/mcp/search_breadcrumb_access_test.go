package mcp_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"trip2g/internal/case/mcp"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

const closedHeading = "Confidential acquisition"

func breadcrumbVault(t *testing.T) *appmodel.NoteViews {
	t.Helper()

	sources := []mdloader.SourceFile{
		{Path: "free.md", PathID: 1, VersionID: 1, Content: []byte("---\nfree: true\n---\nneedleword in the open text\n\n![[_partial]]\n")},
		{Path: "_partial.md", PathID: 2, VersionID: 2, Content: []byte("![[closed]]\n")},
		{Path: "closed.md", PathID: 3, VersionID: 3, Content: []byte("## " + closedHeading + "\n\nneedleword in the closed text\n")},
	}

	views, err := mdloader.Load(mdloader.Options{Log: &logger.TestLogger{}, Sources: sources})
	require.NoError(t, err)

	return views
}

func breadcrumbSearch(t *testing.T, views *appmodel.NoteViews, snippet string, readable func(*appmodel.NoteView) bool) mcp.SearchResultPayload {
	t.Helper()

	free := views.PathMap["free.md"]
	env := &EnvMock{
		FeaturesFunc:               func() features.Features { return features.Features{} },
		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
		SiteConfigFunc:             func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		LoggerFunc:                 func() logger.Logger { return &logger.DummyLogger{} },
		LiveNoteChunksFunc:         func() []appmodel.NoteChunk { return nil },
		LiveNoteViewsFunc:          func() *appmodel.NoteViews { return views },
		LatestNoteViewsFunc:        func() *appmodel.NoteViews { return views },
		NoteURLFunc:                func(n *appmodel.NoteView) string { return n.Permalink },
		SearchLiveNotesFunc: func(string, int, int) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{
				NoteView:        free,
				URL:             free.Permalink,
				ContentSegments: appmodel.PlainSegments(snippet),
			}}, nil
		},
		CanReadNoteFunc: func(_ context.Context, note *appmodel.NoteView) (bool, error) {
			return readable(note), nil
		},
	}

	args, err := json.Marshal(map[string]any{"query": "needleword", "detail_limit": 1})
	require.NoError(t, err)

	return rerankSearchCallArgs(t, env, string(args))
}

func TestSearchBreadcrumbNeverNamesAHeadingOfAClosedEmbed(t *testing.T) {
	snippets := map[string]string{
		"matched snippet":    "needleword",
		"unmatched fallback": "zzz not in the page zzz",
	}

	for name, snippet := range snippets {
		t.Run(name, func(t *testing.T) {
			views := breadcrumbVault(t)

			payload := breadcrumbSearch(t, views, snippet, func(note *appmodel.NoteView) bool { return note.Free })

			require.Len(t, payload.Results, 1)
			for _, match := range payload.Results[0].Matches {
				require.NotContains(t, match.TOCPath, closedHeading)
				require.NotContains(t, match.SectionURL, "confidential")
			}
		})
	}
}

func TestSearchBreadcrumbNamesTheHeadingOfAnEmbedTheCallerCanRead(t *testing.T) {
	views := breadcrumbVault(t)

	payload := breadcrumbSearch(t, views, "needleword", func(*appmodel.NoteView) bool { return true })

	require.Len(t, payload.Results, 1)
	require.Len(t, payload.Results[0].Matches, 1)
	require.Contains(t, payload.Results[0].Matches[0].TOCPath, closedHeading)
}
