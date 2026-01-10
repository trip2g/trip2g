package renderlayoutpreview

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"
	"trip2g/internal/usertoken"

	"github.com/CloudyKit/jet/v6"
)

type Env interface {
	Logger() logger.Logger
	LatestNoteViews() *model.NoteViews
	LoadLatestLayout(source model.LayoutSourceFile) model.Layout
}

type Request struct {
	UserToken *usertoken.Data
	NotePath  string
	Layout    layoutloader.JSONLayout
}

type Response struct {
	Error string
	HTML  string
}

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	if !request.UserToken.IsAdmin() {
		return nil, errors.New("admin authorization required")
	}

	notes := env.LatestNoteViews()

	note := notes.GetByPath(request.NotePath)
	if note == nil {
		return nil, fmt.Errorf("note not found: %s", request.NotePath)
	}

	// Serialize layout back to JSON for Load()
	layoutJSON, err := json.Marshal(request.Layout)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize layout: %w", err)
	}

	// Use Load() to compile the template (auto-converts .html.json)
	source := model.LayoutSourceFile{
		ID:      "/_preview",
		Path:    "/_preview.html.json",
		Content: string(layoutJSON),
	}

	layout := env.LoadLatestLayout(source)
	if layout.View == nil {
		return &Response{Error: "failed to load layout"}, nil
	}

	// Execute template
	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(templateviews.NewNote(note))
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(notes, "latest"))

	var buf bytes.Buffer
	err = layout.View.Execute(&buf, vars, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return &Response{HTML: buf.String()}, nil
}
