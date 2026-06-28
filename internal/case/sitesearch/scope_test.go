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

	req := &appreq.Request{WebhookReadPatterns: []string{"boards/**"}}
	ctx := appreq.NewContext(context.Background(), req)

	conn, err := sitesearch.Resolve(ctx, env, model.SearchInput{Query: "x"})
	require.NoError(t, err)
	require.Len(t, conn.Nodes, 1)
	require.Equal(t, "boards/sprint.md", conn.Nodes[0].NoteView.Path)
}
