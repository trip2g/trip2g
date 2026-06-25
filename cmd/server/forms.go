package main

import (
	"context"
	"trip2g/internal/case/submitform"
	"trip2g/internal/db"
)

func (a *app) InsertFormSubmit(ctx context.Context, noteVersionID int64, formID string, userID *int64, ip string) (int64, error) {
	return a.WriteQueries.InsertFormSubmit(ctx, db.InsertFormSubmitParams{
		NoteVersionID: noteVersionID,
		FormID:        formID,
		UserID:        userID,
		Ip:            ip,
	})
}

func (a *app) InsertFormStringValue(ctx context.Context, submitID int64, fieldName, value string) error {
	return a.WriteQueries.InsertFormStringValue(ctx, db.InsertFormStringValueParams{
		SubmitID: submitID, FieldName: fieldName, Value: value,
	})
}

func (a *app) InsertFormIntValue(ctx context.Context, submitID int64, fieldName string, value int64) error {
	return a.WriteQueries.InsertFormIntValue(ctx, db.InsertFormIntValueParams{
		SubmitID: submitID, FieldName: fieldName, Value: value,
	})
}

func (a *app) InsertFormBoolValue(ctx context.Context, submitID int64, fieldName string, value bool) error {
	v := int64(0)
	if value {
		v = 1
	}
	return a.WriteQueries.InsertFormBoolValue(ctx, db.InsertFormBoolValueParams{
		SubmitID: submitID, FieldName: fieldName, Value: v,
	})
}

func (a *app) ListFormSubmits(ctx context.Context, arg db.ListFormSubmitsParams) ([]db.FormSubmit, error) {
	return a.Queries.ListFormSubmits(ctx, arg)
}

func (a *app) CountFormSubmits(ctx context.Context, arg db.CountFormSubmitsParams) (int64, error) {
	return a.Queries.CountFormSubmits(ctx, arg)
}

func (a *app) GetFormStringValuesBySubmitID(ctx context.Context, submitID int64) ([]db.GetFormStringValuesBySubmitIDRow, error) {
	return a.Queries.GetFormStringValuesBySubmitID(ctx, submitID)
}

func (a *app) GetFormIntValuesBySubmitID(ctx context.Context, submitID int64) ([]db.GetFormIntValuesBySubmitIDRow, error) {
	return a.Queries.GetFormIntValuesBySubmitID(ctx, submitID)
}

func (a *app) GetFormBoolValuesBySubmitID(ctx context.Context, submitID int64) ([]db.GetFormBoolValuesBySubmitIDRow, error) {
	return a.Queries.GetFormBoolValuesBySubmitID(ctx, submitID)
}

func (a *app) GetNotesWithFormSubmits(ctx context.Context) ([]db.GetNotesWithFormSubmitsRow, error) {
	return a.Queries.GetNotesWithFormSubmits(ctx)
}

var _ submitform.Env = (*app)(nil)
