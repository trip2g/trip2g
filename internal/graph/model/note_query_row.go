package model

import (
	"encoding/json"

	"trip2g/internal/model"
)

// NoteQueryRowNotes finds the note a note query row names, for the viewer who
// ran the query. It returns nil when the row names no note the viewer can read.
type NoteQueryRowNotes interface {
	RowNote(data json.RawMessage) (*model.NoteView, error)
}
