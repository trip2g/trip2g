package mcp

import (
	"context"
	"fmt"

	"trip2g/internal/model"
)

type federationACLEnv interface {
	LatestNoteViews() *model.NoteViews
	CanReadNote(ctx context.Context, note *model.NoteView) (bool, error)
}

func accessibleKBNotes(ctx context.Context, env federationACLEnv) ([]*model.MCPFederationNote, error) {
	nvs := env.LatestNoteViews()
	if nvs == nil {
		return nil, nil
	}

	result := make([]*model.MCPFederationNote, 0, len(nvs.MCPFederationNotes))
	for _, kb := range nvs.MCPFederationNotes {
		if kb == nil || kb.Note == nil {
			continue
		}

		ok, err := env.CanReadNote(ctx, kb.Note)
		if err != nil {
			return nil, fmt.Errorf("check federation kb-note access: %w", err)
		}
		if ok {
			result = append(result, kb)
		}
	}
	return result, nil
}
