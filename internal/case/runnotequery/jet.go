package runnotequery

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/CloudyKit/jet/v6"

	"trip2g/internal/graph/model"
)

// JetFunc gives a layout noteQuery(query, params?): a named note query
// run with the viewer's access, returning its result as plain maps and lists.
// A failing query stops the render with the query's message.
func JetFunc(ctx context.Context, env Env) jet.Func {
	return func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("noteQuery", 1, 2)

		query, ok := a.Get(0).Interface().(string)
		if !ok {
			a.Panicf("noteQuery: the query must be a string")
		}

		input := model.NoteQueryInput{Query: query}
		if a.NumOfArguments() == 2 {
			input.Params = a.Get(1).Interface()
		}

		res, err := Resolve(ctx, env, input)
		if err != nil {
			a.Panicf("noteQuery(%q): %v", query, err)
		}

		switch p := res.(type) {
		case *model.ErrorPayload:
			a.Panicf("noteQuery(%q): %s", query, p.Message)
		case *model.NoteQueryPayload:
			raw, _ := p.RawResult.(json.RawMessage)
			var v any
			err = json.Unmarshal(raw, &v)
			if err != nil {
				a.Panicf("noteQuery(%q): %v", query, err)
			}
			// Keep the interface type so a null result is still a valid value.
			return reflect.ValueOf(&v).Elem()
		}

		a.Panicf("noteQuery(%q): unexpected result %T", query, res)
		return reflect.Value{}
	}
}
