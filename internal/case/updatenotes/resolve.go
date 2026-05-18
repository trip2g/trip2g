package updatenotes

import (
	"context"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

type Env interface {
	LatestNoteViews() *appmodel.NoteViews
	InsertNote(ctx context.Context, note appmodel.RawNote) (int64, error)
	HideNotePath(ctx context.Context, params db.HideNotePathParams) error
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(ctx context.Context, pathIDs []int64) error
}

func Resolve(ctx context.Context, env Env, input model.UpdateNotesInput) (model.UpdateNotesOrErrorPayload, error) {
	return nil, nil
}
