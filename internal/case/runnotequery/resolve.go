// Package runnotequery runs a named query: a note with type: note-query whose
// jsonnet block turns the notes the caller can read, plus params, into JSON.
package runnotequery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/graph/model"
	"trip2g/internal/jsonneteval"
	"trip2g/internal/mdloader"
	appmodel "trip2g/internal/model"
	"trip2g/internal/yamlutil"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type Env interface {
	canreadnote.Env
	LatestNoteViews() *appmodel.NoteViews
	// ObserveNoteQuery records the call's duration for metrics and logs a
	// slow one. query is the query note's path, or "unknown".
	ObserveNoteQuery(query, result string, elapsed time.Duration, paramsBytes int)
}

const maxParamsBytes = 16 << 10

// queryPrelude gives the query `notes`, `params` and `currentUser` as values.
// It shares the block's first line, so jsonnet error line numbers match the
// block.
const queryPrelude = `local notes = std.parseJson(std.extVar("notes")), params = std.parseJson(std.extVar("params")), currentUser = std.parseJson(std.extVar("currentUser")); `

type queryNote struct {
	Path      string         `json:"path"`
	Title     string         `json:"title"`
	Permalink string         `json:"permalink"`
	Meta      map[string]any `json:"meta"`
	// Readable is set only for `all_notes: true` queries.
	Readable *bool `json:"readable,omitempty"`
}

func Resolve(ctx context.Context, env Env, input model.NoteQueryInput) (model.NoteQueryOrErrorPayload, error) {
	start := time.Now()
	query := "unknown"

	params, paramsErr := encodeParams(input.Params)

	var res model.NoteQueryOrErrorPayload
	var err error
	if paramsErr != nil {
		res = paramsErr
	} else {
		res, err = resolve(ctx, env, input.Query, params, &query)
	}

	result := "ok"
	if p, isErr := res.(*model.ErrorPayload); isErr {
		result = "error"
		if p.Message == errQueryNotFound {
			result = "not_found"
		}
	}
	if err != nil {
		result = "failed"
	}

	env.ObserveNoteQuery(query, result, time.Since(start), len(params))

	return res, err
}

const errQueryNotFound = "query not found"

// encodeParams turns the caller's params into the JSON the program reads.
func encodeParams(params any) (string, *model.ErrorPayload) {
	if params == nil {
		return "{}", nil
	}

	b, err := json.Marshal(params)
	if err != nil {
		return "", &model.ErrorPayload{Message: "params must be JSON"}
	}
	if len(b) > maxParamsBytes {
		return "", &model.ErrorPayload{Message: "params is larger than 16 KB"}
	}
	return string(b), nil
}

// resolve does the work; it sets *query to the query note's path once the
// caller is known to be allowed to see it.
func resolve(ctx context.Context, env Env, queryName, params string, query *string) (model.NoteQueryOrErrorPayload, error) {
	nvs := env.LatestNoteViews()

	note := findQuery(nvs, queryName)
	if note == nil {
		return &model.ErrorPayload{Message: errQueryNotFound}, nil
	}

	read, err := canreadnote.NewReader(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve the viewer: %w", err)
	}

	// A query the caller can't read must look the same as a missing one.
	canRead, err := read(note)
	if err != nil {
		return nil, fmt.Errorf("failed to check query access: %w", err)
	}
	if !canRead {
		return &model.ErrorPayload{Message: errQueryNotFound}, nil
	}

	if typ, _ := note.RawMeta["type"].(string); typ != "note-query" {
		return &model.ErrorPayload{Message: note.Path + " is not a note-query"}, nil
	}
	*query = note.Path

	blocks, err := mdloader.JsonnetBlocks(note)
	if err != nil {
		return &model.ErrorPayload{Message: note.Path + ": " + err.Error()}, nil //nolint:nilerr // a broken query note is shown to the caller as a payload.
	}
	if len(blocks) != 1 {
		return &model.ErrorPayload{Message: note.Path + ": a note-query needs exactly one jsonnet block"}, nil
	}

	// all_notes: true hands the program every note with `readable`; the author
	// filters. Without it the program sees only readable notes.
	allNotes := note.RawMeta["all_notes"] == true

	notesJSON, err := queryNotes(read, nvs, allNotes)
	if err != nil {
		return nil, err
	}

	user, err := currentUser(ctx, env)
	if err != nil {
		return nil, err
	}

	out, err := jsonneteval.EvalJSON(queryPrelude+blocks[0], map[string]string{
		"notes":       string(notesJSON),
		"params":      params,
		"currentUser": user,
	})
	if err != nil {
		msg := strings.TrimPrefix(err.Error(), "evaluate jsonnet: ")
		msg, _, _ = strings.Cut(msg, "\n")
		return &model.ErrorPayload{Message: note.Path + ": " + msg}, nil
	}

	var compact bytes.Buffer
	err = json.Compact(&compact, out)
	if err != nil {
		return nil, fmt.Errorf("failed to compact query result: %w", err)
	}

	return &model.NoteQueryPayload{RawResult: json.RawMessage(compact.Bytes())}, nil
}

// findQuery looks the query up by vault path, then like a wikilink.
func findQuery(nvs *appmodel.NoteViews, query string) *appmodel.NoteView {
	name := strings.TrimSpace(query)
	name = strings.TrimPrefix(name, "[[")
	name = strings.TrimSuffix(name, "]]")
	name, _, _ = strings.Cut(name, "|")
	name = strings.TrimSuffix(strings.TrimSpace(name), ".md")

	if note, ok := nvs.PathMap[name+".md"]; ok {
		return note
	}
	return nvs.ResolveWikilinkTarget(nil, name)
}

// queryNotes is the query's `notes` in path order, with frontmatter after
// patches: the notes the caller can read, or with all set every note plus
// `readable`. System notes (`_…`) are left out, as in other listings.
func queryNotes(read canreadnote.Reader, nvs *appmodel.NoteViews, all bool) ([]byte, error) {
	paths := make([]string, 0, len(nvs.PathMap))
	for path := range nvs.PathMap {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	notes := make([]queryNote, 0, len(paths))
	for _, path := range paths {
		note := nvs.PathMap[path]
		if note.IsSystem() || note.Ast() == nil {
			continue
		}

		canRead, err := read(note)
		if err != nil {
			return nil, fmt.Errorf("failed to check note access: %w", err)
		}
		if !canRead && !all {
			continue
		}

		meta, _ := finite(yamlutil.Normalize(note.RawMeta)).(map[string]any)
		if meta == nil {
			meta = map[string]any{}
		}

		qn := queryNote{
			Path:      note.Path,
			Title:     note.Title,
			Permalink: note.Permalink,
			Meta:      meta,
		}
		if all {
			qn.Readable = &canRead
		}
		notes = append(notes, qn)
	}

	out, err := json.Marshal(notes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notes: %w", err)
	}
	return out, nil
}

// finite replaces NaN and ±Inf, which YAML allows and JSON doesn't, with nil,
// so one odd note can't break every query.
func finite(v any) any {
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return nil
		}
	case map[string]any:
		for k, item := range t {
			t[k] = finite(item)
		}
	case []any:
		for i, item := range t {
			t[i] = finite(item)
		}
	}
	return v
}

// currentUser is the query's `currentUser`: who is asking. A guest has a null
// id and role.
func currentUser(ctx context.Context, env Env) (string, error) {
	token, err := env.CurrentUserToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	user := map[string]any{
		"id":       nil,
		"role":     nil,
		"isAdmin":  token.IsAdmin(),
		"signedIn": token != nil,
	}
	if token != nil {
		user["id"] = token.ID
		user["role"] = token.Role
	}

	b, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("failed to marshal current user: %w", err)
	}
	return string(b), nil
}
