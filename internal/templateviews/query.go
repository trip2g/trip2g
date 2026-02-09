package templateviews

import (
	"cmp"
	"reflect"
	"sort"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

// sortField represents a single sort criterion.
type sortField struct {
	field string
	desc  bool
	meta  bool
}

// NoteQuery is a lazy query builder for filtering and sorting notes.
// Operations are accumulated and executed only when a terminal method is called.
type NoteQuery struct {
	nvs    *NVS
	glob   string
	sorts  []sortField
	limit  int
	offset int
}

// SortBy adds a sort criterion by a Note method name (Title, CreatedAt, Permalink).
// Default order is ascending. Use Desc() to change to descending.
func (q *NoteQuery) SortBy(field string) *NoteQuery {
	q.sorts = append(q.sorts, sortField{field: field})
	return q
}

// SortByMeta adds a sort criterion by a frontmatter meta field.
// Default order is ascending. Use Desc() to change to descending.
func (q *NoteQuery) SortByMeta(field string) *NoteQuery {
	q.sorts = append(q.sorts, sortField{field: field, meta: true})
	return q
}

// Desc sets the last added sort criterion to descending order.
func (q *NoteQuery) Desc() *NoteQuery {
	if len(q.sorts) > 0 {
		q.sorts[len(q.sorts)-1].desc = true
	}
	return q
}

// Asc sets the last added sort criterion to ascending order (default).
func (q *NoteQuery) Asc() *NoteQuery {
	if len(q.sorts) > 0 {
		q.sorts[len(q.sorts)-1].desc = false
	}
	return q
}

// Limit sets the maximum number of notes to return.
func (q *NoteQuery) Limit(n int) *NoteQuery {
	q.limit = n
	return q
}

// Offset sets the number of notes to skip before returning results.
func (q *NoteQuery) Offset(n int) *NoteQuery {
	q.offset = n
	return q
}

// All executes the query and returns all matching notes.
func (q *NoteQuery) All() []*Note {
	if q.nvs == nil || q.nvs.nvs == nil {
		return nil
	}

	// Step 1: Filter by glob pattern
	var notes []*Note
	for path, nv := range q.nvs.nvs.PathMap {
		if q.glob != "" {
			// TODO: handle error and compile pattern once
			match, _ := doublestar.Match(q.glob, path)
			if !match {
				continue
			}
		}
		notes = append(notes, NewNote(nv))
	}

	// Step 2: Sort
	if len(q.sorts) > 0 {
		q.sortNotes(notes)
	}

	// Step 3: Apply offset
	if q.offset > 0 {
		if q.offset >= len(notes) {
			return nil
		}
		notes = notes[q.offset:]
	}

	// Step 4: Apply limit
	if q.limit > 0 && q.limit < len(notes) {
		notes = notes[:q.limit]
	}

	return notes
}

// First executes the query and returns the first matching note, or nil.
func (q *NoteQuery) First() *Note {
	q.limit = 1
	notes := q.All()
	if len(notes) == 0 {
		return nil
	}
	return notes[0]
}

// Last executes the query and returns the last matching note, or nil.
func (q *NoteQuery) Last() *Note {
	notes := q.All()
	if len(notes) == 0 {
		return nil
	}
	return notes[len(notes)-1]
}

// sortNotes sorts notes by the configured sort criteria.
func (q *NoteQuery) sortNotes(notes []*Note) {
	sort.SliceStable(notes, func(i, j int) bool {
		for _, sf := range q.sorts {
			cmp := q.compareNotes(notes[i], notes[j], sf)
			if cmp != 0 {
				if sf.desc {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return false
	})
}

// compareNotes compares two notes by a sort field.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func (q *NoteQuery) compareNotes(a, b *Note, sf sortField) int {
	if sf.meta {
		return q.compareByMeta(a, b, sf.field)
	}
	return q.compareByField(a, b, sf.field)
}

// normalizeFieldName maps snake_case to method names.
func normalizeFieldName(field string) string {
	switch field {
	case "title":
		return "Title"
	case "created_at":
		return "CreatedAt"
	case "permalink", "path":
		return "Permalink"
	default:
		return field
	}
}

// compareByField compares notes by a Note method using reflection.
func (q *NoteQuery) compareByField(a, b *Note, field string) int {
	// Normalize field name
	field = normalizeFieldName(field)

	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	ma := va.MethodByName(field)
	mb := vb.MethodByName(field)

	if !ma.IsValid() || !mb.IsValid() {
		return 0
	}

	resA := ma.Call(nil)
	resB := mb.Call(nil)

	if len(resA) == 0 || len(resB) == 0 {
		return 0
	}

	return compareValues(resA[0].Interface(), resB[0].Interface())
}

// compareByMeta compares notes by a frontmatter meta field.
func (q *NoteQuery) compareByMeta(a, b *Note, field string) int {
	metaA := a.M()
	metaB := b.M()

	if metaA == nil || metaB == nil {
		return 0
	}

	valA := metaA.Get(field)
	valB := metaB.Get(field)

	return compareValues(valA, valB)
}

// compareValues compares two interface{} values.
// Supports string, int, int64, float64, time.Time.
func compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return cmp.Compare(va, vb)
		}
	case int:
		if vb, ok := b.(int); ok {
			return cmp.Compare(va, vb)
		}
	case int64:
		if vb, ok := b.(int64); ok {
			return cmp.Compare(va, vb)
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return cmp.Compare(va, vb)
		}
	case time.Time:
		if vb, ok := b.(time.Time); ok {
			return va.Compare(vb)
		}
	}

	return 0
}
