package federationtopology_test

import (
	"context"
	"testing"

	"trip2g/internal/case/system/federationtopology"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestBuildTopology(t *testing.T) {
	kbURL := "https://bob.team.io/_system/mcp"
	env := &EnvMock{
		PublicURLFunc: func() string { return "https://alice.team.io" },
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

func namesOf(ss []federationtopology.Subgraph) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, s.Name)
	}
	return out
}
