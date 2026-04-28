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
