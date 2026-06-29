package sitesearch_test

//go:generate go tool github.com/matryer/moq -out scope_mocks_test.go -pkg sitesearch_test . Env

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/sitesearch"
	"trip2g/internal/features"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/openai"
	"trip2g/internal/usertoken"
)

func TestResolve_ReadPatternsOmitOutOfScope(t *testing.T) {
	inScope := appmodel.SearchResult{URL: "/boards/sprint", NoteView: &appmodel.NoteView{Path: "boards/sprint.md", Permalink: "/boards/sprint"}}
	outScope := appmodel.SearchResult{URL: "/secrets/p", NoteView: &appmodel.NoteView{Path: "secrets/p.md", Permalink: "/secrets/p"}}

	env := &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) { return &usertoken.Data{}, nil },
		SiteConfigFunc:       func(_ context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc:  func(_ string) ([]appmodel.SearchResult, error) { return []appmodel.SearchResult{inScope, outScope}, nil },
		FeaturesFunc:         func() features.Features { return features.Features{} },
		OpenAIFunc:           func() *openai.Client { return nil },
		CanReadNoteFunc:      func(_ context.Context, _ *appmodel.NoteView) (bool, error) { return true, nil },
		LoggerFunc:           func() logger.Logger { return &logger.DummyLogger{} },
	}

	// WebhookScoped must be true: read-pattern enforcement is only triggered for
	// scoped shortapitoken requests. Without it the predicate is fail-open.
	req := &appreq.Request{WebhookScoped: true, WebhookReadPatterns: []string{"boards/**"}}
	ctx := appreq.NewContext(context.Background(), req)

	conn, err := sitesearch.Resolve(ctx, env, model.SearchInput{Query: "x"})
	require.NoError(t, err)
	require.Len(t, conn.Nodes, 1)
	require.Equal(t, "boards/sprint.md", conn.Nodes[0].NoteView.Path)
}

func makeSearchEnv(results []appmodel.SearchResult) *EnvMock {
	return &EnvMock{
		CurrentUserTokenFunc: func(_ context.Context) (*usertoken.Data, error) { return &usertoken.Data{}, nil },
		SiteConfigFunc:       func(_ context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc:  func(_ string) ([]appmodel.SearchResult, error) { return results, nil },
		FeaturesFunc:         func() features.Features { return features.Features{} },
		OpenAIFunc:           func() *openai.Client { return nil },
		CanReadNoteFunc:      func(_ context.Context, _ *appmodel.NoteView) (bool, error) { return true, nil },
		LoggerFunc:           func() logger.Logger { return &logger.DummyLogger{} },
	}
}

// TestResolve_ScopedSearch_FailClosed asserts fail-closed behaviour for scoped
// shortapitoken requests in the search path:
//   - scoped + empty read_patterns  → deny all (return nothing)
//   - scoped + non-empty patterns   → in-scope notes only
//   - unscoped                      → normal (no read-pattern enforcement)
func TestResolve_ScopedSearch_FailClosed(t *testing.T) {
	inScope := appmodel.SearchResult{URL: "/boards/sprint", NoteView: &appmodel.NoteView{Path: "boards/sprint.md", Permalink: "/boards/sprint"}}
	outScope := appmodel.SearchResult{URL: "/secrets/p", NoteView: &appmodel.NoteView{Path: "secrets/p.md", Permalink: "/secrets/p"}}
	both := []appmodel.SearchResult{inScope, outScope}

	tests := []struct {
		name      string
		req       *appreq.Request
		wantLen   int
		wantPaths []string
	}{
		{
			name:    "scoped + empty read_patterns → deny all",
			req:     &appreq.Request{WebhookScoped: true, WebhookReadPatterns: nil},
			wantLen: 0,
		},
		{
			name:      "scoped + non-empty read_patterns → in-scope only",
			req:       &appreq.Request{WebhookScoped: true, WebhookReadPatterns: []string{"boards/**"}},
			wantLen:   1,
			wantPaths: []string{"boards/sprint.md"},
		},
		{
			name:    "unscoped → no read-pattern enforcement",
			req:     &appreq.Request{WebhookScoped: false},
			wantLen: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := appreq.NewContext(context.Background(), tc.req)
			conn, err := sitesearch.Resolve(ctx, makeSearchEnv(both), model.SearchInput{Query: "x"})
			require.NoError(t, err)
			require.Len(t, conn.Nodes, tc.wantLen)
			for i, path := range tc.wantPaths {
				require.Equal(t, path, conn.Nodes[i].NoteView.Path)
			}
		})
	}
}
