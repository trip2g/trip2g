package graph

import (
	"context"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

// parseInterfaceTime converts SQLite's datetime interface{} to time.Time.
func parseInterfaceTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339} {
			if parsed, err := time.Parse(layout, t); err == nil {
				return parsed
			}
		}
	}
	return time.Time{}
}

type formFieldReader interface {
	GetFormStringValuesBySubmitID(context.Context, int64) ([]db.GetFormStringValuesBySubmitIDRow, error)
	GetFormIntValuesBySubmitID(context.Context, int64) ([]db.GetFormIntValuesBySubmitIDRow, error)
	GetFormBoolValuesBySubmitID(context.Context, int64) ([]db.GetFormBoolValuesBySubmitIDRow, error)
}

func loadFormSubmitFields(ctx context.Context, r formFieldReader, submitID int64) ([]model.AdminFormValue, error) {
	var fields []model.AdminFormValue

	strs, err := r.GetFormStringValuesBySubmitID(ctx, submitID)
	if err != nil {
		return nil, err
	}
	for _, s := range strs {
		fields = append(fields, &model.AdminFormStringValue{Name: s.FieldName, Value: s.Value})
	}

	ints, err := r.GetFormIntValuesBySubmitID(ctx, submitID)
	if err != nil {
		return nil, err
	}
	for _, n := range ints {
		fields = append(fields, &model.AdminFormIntValue{Name: n.FieldName, Value: int32(n.Value)})
	}

	bools, err := r.GetFormBoolValuesBySubmitID(ctx, submitID)
	if err != nil {
		return nil, err
	}
	for _, b := range bools {
		fields = append(fields, &model.AdminFormBoolValue{Name: b.FieldName, Value: b.Value != 0})
	}

	return fields, nil
}
