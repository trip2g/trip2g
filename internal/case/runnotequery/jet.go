package runnotequery

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/CloudyKit/jet/v6"

	"trip2g/internal/graph/model"
	"trip2g/internal/templateviews"
)

// JetFunc gives a layout noteQuery(query, params?): a named note query
// run with the viewer's access, returning its result as plain maps and lists.
// A failing query stops the render with the query's message.
func JetFunc(ctx context.Context, env Env) jet.Func {
	return func(a jet.Arguments) reflect.Value {
		query, payload := runJet(ctx, env, "noteQuery", a)

		raw, _ := payload.Result.(json.RawMessage)
		var v any
		err := json.Unmarshal(raw, &v)
		if err != nil {
			a.Panicf("noteQuery(%q): %v", query, err)
		}
		// Keep the interface type so a null result is still a valid value.
		return reflect.ValueOf(&v).Elem()
	}
}

// JetRowsFunc gives a layout noteQueryRows(query, params?): the same query,
// whose result must be a list, as rows with Data() and Note(). Notes render
// for domainHost, like the page around them.
func JetRowsFunc(ctx context.Context, env Env, domainHost string) jet.Func {
	return func(a jet.Arguments) reflect.Value {
		query, payload := runJet(ctx, env, "noteQueryRows", a)

		rows, err := Rows(payload)
		if err != nil {
			a.Panicf("noteQueryRows(%q): %v", query, err)
		}
		if rows == nil {
			a.Panicf("noteQueryRows(%q): the result is not a list", query)
		}

		out := make([]*JetRow, len(rows))
		for i := range rows {
			out[i] = &JetRow{row: rows[i], domainHost: domainHost}
		}
		return reflect.ValueOf(out)
	}
}

// runJet runs the query a template function was called with and stops the
// render on any failure.
func runJet(ctx context.Context, env Env, name string, a jet.Arguments) (string, *model.NoteQueryPayload) {
	a.RequireNumOfArguments(name, 1, 2)

	query, ok := a.Get(0).Interface().(string)
	if !ok {
		a.Panicf("%s: the query must be a string", name)
	}

	input := model.NoteQueryInput{Query: query}
	if a.NumOfArguments() == 2 {
		input.Params = a.Get(1).Interface()
	}

	res, err := Resolve(ctx, env, input)
	if err != nil {
		a.Panicf("%s(%q): %v", name, query, err)
	}

	switch p := res.(type) {
	case *model.ErrorPayload:
		a.Panicf("%s(%q): %s", name, query, p.Message)
	case *model.NoteQueryPayload:
		return query, p
	}

	a.Panicf("%s(%q): unexpected result %T", name, query, res)
	return query, nil
}

// JetRow is one item of a list result in a template.
type JetRow struct {
	row        model.NoteQueryRow
	domainHost string
}

// Data is the item as the program returned it, as plain maps and lists.
func (r *JetRow) Data() any {
	raw, _ := r.row.Data.(json.RawMessage)
	var v any
	err := json.Unmarshal(raw, &v)
	if err != nil {
		panic(err)
	}
	return v
}

// Note is the note the item names (by version_id, path_id or path), or an
// untyped nil when it names none the viewer can read.
func (r *JetRow) Note() any {
	data, _ := r.row.Data.(json.RawMessage)
	note, err := r.row.Notes.RowNote(data)
	if err != nil {
		panic(err)
	}
	if note == nil {
		return nil
	}
	return templateviews.NewNoteWithDomain(note, r.domainHost)
}
