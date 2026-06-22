package federationtopology_test

import (
	"context"
	"testing"
	"time"

	"trip2g/internal/case/system/federationtopology"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestBuildTopology(t *testing.T) {
	kbURL := "https://bob.team.io/_system/mcp"
	env := &EnvMock{
		PublicURLFunc:      func() string { return "https://alice.team.io" },
		LoadSiteConfigFunc: func(ctx context.Context) (model.SiteConfig, error) { return model.SiteConfig{}, nil },
		ListAllSubgraphsFunc: func(ctx context.Context) ([]db.Subgraph, error) {
			return []db.Subgraph{{ID: 1, Name: "internal"}}, nil
		},
		ListFederationSecretsFunc: func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
			return []db.ListFederationSecretsRow{
				{ID: 12, Kid: "bob-2026", KbUrl: &kbURL}, // outbound (has kb_url)
				{ID: 7, Kid: "carol-2026"},               // inbound (no kb_url)
			}, nil
		},
		ListAllFederationSecretScopesFunc: func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
			return []db.ListAllFederationSecretScopesRow{
				{Kid: "bob-2026", SubgraphID: 1, SubgraphName: "internal"},
			}, nil
		},
		LatestNoteViewsFunc: func() *model.NoteViews {
			return &model.NoteViews{MCPFederationNotes: []*model.MCPFederationNote{
				{URL: kbURL, ID: "bob"},
			}}
		},
	}

	out, err := federationtopology.Resolve(context.Background(), env)
	require.NoError(t, err)

	require.Equal(t, "https://alice.team.io/_system/mcp", out.Self.MCPURL)
	require.Equal(t, "alice.team.io", out.Self.KBID)
	require.Len(t, out.Self.Subgraphs, 1)

	require.Len(t, out.Outbound, 1)
	require.Equal(t, kbURL, *out.Outbound[0].KBURL)
	require.Equal(t, []string{"internal"}, namesOf(out.Outbound[0].Subgraphs))

	require.Len(t, out.Inbound, 1)
	require.Nil(t, out.Inbound[0].KBURL)

	require.Len(t, out.KBNotes, 1)
	require.Equal(t, kbURL, out.KBNotes[0].KBURL)
}

func TestRevokedAndEmptyScope(t *testing.T) {
	now := time.Now()
	kbURL := "https://x/_system/mcp"
	env := &EnvMock{
		PublicURLFunc:        func() string { return "https://me.io" },
		LoadSiteConfigFunc:   func(ctx context.Context) (model.SiteConfig, error) { return model.SiteConfig{}, nil },
		ListAllSubgraphsFunc: func(ctx context.Context) ([]db.Subgraph, error) { return nil, nil },
		ListFederationSecretsFunc: func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
			return []db.ListFederationSecretsRow{{ID: 1, Kid: "k", KbUrl: &kbURL, RevokedAt: &now}}, nil
		},
		ListAllFederationSecretScopesFunc: func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
			return nil, nil
		},
		LatestNoteViewsFunc: func() *model.NoteViews { return &model.NoteViews{} },
	}
	out, err := federationtopology.Resolve(context.Background(), env)
	require.NoError(t, err)
	require.NotNil(t, out.Outbound[0].RevokedAt)
	require.NotNil(t, out.Outbound[0].Subgraphs)
	require.Empty(t, out.Outbound[0].Subgraphs)
}

func TestSelfKBIDFromConfigAndFallback(t *testing.T) {
	newEnv := func(kbID string) *EnvMock {
		return &EnvMock{
			PublicURLFunc:      func() string { return "https://alice.team.io" },
			LoadSiteConfigFunc: func(ctx context.Context) (model.SiteConfig, error) { return model.SiteConfig{KBID: kbID}, nil },
			ListAllSubgraphsFunc: func(ctx context.Context) ([]db.Subgraph, error) {
				return nil, nil
			},
			ListFederationSecretsFunc: func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
				return nil, nil
			},
			ListAllFederationSecretScopesFunc: func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
				return nil, nil
			},
			LatestNoteViewsFunc: func() *model.NoteViews { return &model.NoteViews{} },
		}
	}

	// Explicit config value wins.
	out, err := federationtopology.Resolve(context.Background(), newEnv("alice"))
	require.NoError(t, err)
	require.Equal(t, "alice", out.Self.KBID)
	require.Equal(t, "alice", out.Self.Name)

	// Empty config falls back to the public URL host.
	out, err = federationtopology.Resolve(context.Background(), newEnv("  "))
	require.NoError(t, err)
	require.Equal(t, "alice.team.io", out.Self.KBID)
	require.Equal(t, "alice.team.io", out.Self.Name)
}

func namesOf(ss []federationtopology.Subgraph) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, s.Name)
	}
	return out
}
