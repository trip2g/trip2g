package mcp

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type federationACLEnvMock struct {
	noteViews   *model.NoteViews
	canReadFunc func(ctx context.Context, note *model.NoteView) (bool, error)
}

func (m federationACLEnvMock) LatestNoteViews() *model.NoteViews {
	return m.noteViews
}

func (m federationACLEnvMock) CanReadNote(ctx context.Context, note *model.NoteView) (bool, error) {
	return m.canReadFunc(ctx, note)
}

func TestAccessibleKBNotes(t *testing.T) {
	allowedNote := &model.NoteView{PathID: 1, MCPFederationKBURL: "https://allowed.example/_system/mcp"}
	deniedNote := &model.NoteView{PathID: 2, MCPFederationKBURL: "https://denied.example/_system/mcp"}
	nvs := model.NewNoteViews()
	nvs.MCPFederationNotes = []*model.MCPFederationNote{
		model.NewMCPFederationNote(allowedNote),
		model.NewMCPFederationNote(deniedNote),
	}

	got, err := accessibleKBNotes(context.Background(), federationACLEnvMock{
		noteViews: nvs,
		canReadFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
			return note.PathID == allowedNote.PathID, nil
		},
	})

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, allowedNote, got[0].Note)
}

// An API key is admin everywhere else on the MCP surface (canReadMCPNote), and
// a KB-note is a note like any other. Before this test accessibleKBNotes called
// CanReadNote directly, so a KB-note without free: true was invisible to an
// API-keyed caller and every federated_* call on it answered
// federation_not_configured — the same "not configured" a kb_id that does not
// exist gets, which is why it read as a routing problem rather than an ACL one.
func TestAccessibleKBNotesAPIKeySeesGatedKBNote(t *testing.T) {
	gated := &model.NoteView{PathID: 1, MCPFederationKBURL: "https://gated.example/_system/mcp"}
	nvs := model.NewNoteViews()
	nvs.MCPFederationNotes = []*model.MCPFederationNote{model.NewMCPFederationNote(gated)}

	env := federationACLEnvMock{
		noteViews: nvs,
		canReadFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
			return false, errors.New("CanReadNote must not decide for an API key")
		},
	}

	got, err := accessibleKBNotes(contextWithMCPAPIKeyAuth(context.Background(), false), env)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, gated, got[0].Note)
}

func TestAccessibleKBNotesFederatedPeerKeepsItsSubgraphScope(t *testing.T) {
	inScope := &model.NoteView{
		PathID:             1,
		SubgraphNames:      []string{"shared"},
		MCPFederationKBURL: "https://in.example/_system/mcp",
	}
	outOfScope := &model.NoteView{
		PathID:             2,
		SubgraphNames:      []string{"private"},
		MCPFederationKBURL: "https://out.example/_system/mcp",
	}
	nvs := model.NewNoteViews()
	nvs.MCPFederationNotes = []*model.MCPFederationNote{
		model.NewMCPFederationNote(inScope),
		model.NewMCPFederationNote(outOfScope),
	}

	env := federationACLEnvMock{
		noteViews: nvs,
		canReadFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
			return false, errors.New("CanReadNote must not decide for a federated peer")
		},
	}
	ctx := contextWithFederationAuth(context.Background(), federationAuth{KID: "peer", AllowedSubgraphs: []string{"shared"}})

	got, err := accessibleKBNotes(ctx, env)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, inScope, got[0].Note)
}

func TestAccessibleKBNotesReturnsCanReadError(t *testing.T) {
	wantErr := errors.New("boom")
	note := &model.NoteView{PathID: 1, MCPFederationKBURL: "https://allowed.example/_system/mcp"}
	nvs := model.NewNoteViews()
	nvs.MCPFederationNotes = []*model.MCPFederationNote{model.NewMCPFederationNote(note)}

	_, err := accessibleKBNotes(context.Background(), federationACLEnvMock{
		noteViews: nvs,
		canReadFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
			return false, wantErr
		},
	})

	require.ErrorIs(t, err, wantErr)
}
