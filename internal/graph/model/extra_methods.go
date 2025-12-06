package model

import (
	"errors"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/model"
)

func NewFieldError(field string, message string) *ErrorPayload {
	return &ErrorPayload{
		ByFields: []FieldMessage{{Name: field, Value: message}},
	}
}

func NewOzzoError(err error) *ErrorPayload {
	if err == nil {
		return nil
	}

	var ozzoErrors ozzo.Errors
	if !errors.As(err, &ozzoErrors) {
		return &ErrorPayload{Message: err.Error()}
	}

	payload := ErrorPayload{}

	for key, fieldErr := range ozzoErrors {
		payload.ByFields = append(payload.ByFields, FieldMessage{
			Name:  key,
			Value: fieldErr.Error(),
		})
	}

	return &payload
}

func ConvertNoteToPublic(note *model.NoteView) *PublicNote {
	return &PublicNote{
		PathID: note.PathID,
		Title:  note.Title,
		HTML:   string(note.HTML),
		Toc:    prepareTOC(note),

		NoteView: note,
	}
}

func prepareTOC(note *model.NoteView) []NoteTocItem {
	toc := make([]NoteTocItem, 0, len(note.TOC()))
	for _, heading := range note.TOC() {
		level := heading.Level
		if level > 2147483647 {
			level = 2147483647 // Cap at max int32 value
		}
		toc = append(toc, NoteTocItem{
			ID:    heading.ID,
			Title: heading.Text,
			Level: int32(level),
		})
	}

	return toc
}
