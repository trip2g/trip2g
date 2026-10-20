package runnotequery

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

// rowNotes resolves the notes that rows of one query result name. It does
// nothing until a row asks: the viewer and the id indexes are built on the
// first lookup and shared by every row after it.
type rowNotes struct {
	ctx context.Context
	env Env
	nvs *appmodel.NoteViews

	once      sync.Once
	read      canreadnote.Reader
	byVersion map[int64]*appmodel.NoteView
	byPathID  map[int64]*appmodel.NoteView
	err       error
}

var _ model.NoteQueryRowNotes = (*rowNotes)(nil)

func newRowNotes(ctx context.Context, env Env, nvs *appmodel.NoteViews) *rowNotes {
	return &rowNotes{ctx: ctx, env: env, nvs: nvs}
}

// rowRef is what a row may carry to name a note.
type rowRef struct {
	VersionID *int64  `json:"version_id"`
	PathID    *int64  `json:"path_id"`
	Path      *string `json:"path"`
}

func (r *rowNotes) init() {
	read, _, err := canreadnote.NewReader(r.ctx, r.env)
	if err != nil {
		r.err = fmt.Errorf("failed to resolve the viewer: %w", err)
		return
	}
	r.read = read

	r.byVersion = make(map[int64]*appmodel.NoteView, len(r.nvs.PathMap))
	r.byPathID = make(map[int64]*appmodel.NoteView, len(r.nvs.PathMap))
	for _, note := range r.nvs.PathMap {
		r.byVersion[note.VersionID] = note
		r.byPathID[note.PathID] = note
	}
}

// RowNote is the note a row names: by version_id, else path_id, else path.
// Access is checked again here, independently of what the program was given,
// so a program over `all_notes: true` can't hand out a note the viewer can't
// read.
func (r *rowNotes) RowNote(data json.RawMessage) (*appmodel.NoteView, error) {
	var ref rowRef
	err := json.Unmarshal(data, &ref)
	if err != nil {
		return nil, nil //nolint:nilerr // an item that doesn't name a note has no note.
	}

	r.once.Do(r.init)
	if r.err != nil {
		return nil, r.err
	}

	var note *appmodel.NoteView
	switch {
	case ref.VersionID != nil:
		note = r.byVersion[*ref.VersionID]
	case ref.PathID != nil:
		note = r.byPathID[*ref.PathID]
	case ref.Path != nil:
		note = r.nvs.PathMap[*ref.Path]
	}
	if note == nil || !queryable(note) {
		return nil, nil
	}

	canRead, err := r.read(note)
	if err != nil {
		return nil, fmt.Errorf("failed to check note access: %w", err)
	}
	if !canRead {
		return nil, nil
	}
	return note, nil
}

// Rows splits a list result into rows; nil when the result is not a list.
func Rows(payload *model.NoteQueryPayload) ([]model.NoteQueryRow, error) {
	raw, _ := payload.Result.(json.RawMessage)

	var items []json.RawMessage
	err := json.Unmarshal(raw, &items)
	if err != nil || items == nil {
		return nil, nil //nolint:nilerr // a result that isn't a list has no rows.
	}

	rows := make([]model.NoteQueryRow, len(items))
	for i, item := range items {
		rows[i] = model.NoteQueryRow{Data: item, Notes: payload.Notes}
	}
	return rows, nil
}

// RowNote is the row's note as the public note type, or nil.
func RowNote(row *model.NoteQueryRow) (*model.PublicNote, error) {
	if row.Notes == nil {
		return nil, nil
	}

	data, _ := row.Data.(json.RawMessage)
	note, err := row.Notes.RowNote(data)
	if err != nil || note == nil {
		return nil, err
	}
	return model.ConvertNoteToPublic(note), nil
}
