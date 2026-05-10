package renderpreview

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"

	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"
)

type Env interface {
	Logger() logger.Logger
	LatestNoteViews() *model.NoteViews
	LoadPreviewLayout(main model.LayoutSourceFile, extra map[string]string) (model.Layout, []string)
	PreviewBuffer() *PreviewBuffer
}

// Resolve renders a layout and stores the result in the preview buffer.
// Auth is the caller's responsibility (HTTP endpoint or GraphQL admin mutation).
func Resolve(ctx context.Context, env Env, input graphmodel.RenderLayoutInput) (*graphmodel.RenderLayoutPayload, error) {
	if input.Layout == nil || input.Layout.Path == "" {
		return nil, errors.New("layout.path is required")
	}

	// 1. Normalise layout path (Jet requires leading slash).
	layoutPath := input.Layout.Path
	if !strings.HasPrefix(layoutPath, "/") {
		layoutPath = "/" + layoutPath
	}

	layoutSrc := ""
	if input.Layout.Src != nil {
		layoutSrc = *input.Layout.Src
	}

	mainSource := model.LayoutSourceFile{
		ID:      layoutPath,
		Path:    layoutPath,
		Content: layoutSrc, // empty → LoadPreview fills from server snapshot
	}

	// Build override files map. Caller-supplied content takes priority over server.
	overrides := make(map[string]string)
	for _, f := range input.OverrideFiles {
		p := f.Path
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if f.Src != nil {
			overrides[p] = *f.Src
		}
	}
	if layoutSrc != "" {
		overrides[layoutPath] = layoutSrc
	}

	// 2. Compile layout.
	layout, layoutWarnings := env.LoadPreviewLayout(mainSource, overrides)

	warnings := &graphmodel.RenderLayoutWarnings{
		Layout: make([]string, 0, len(layoutWarnings)),
		Note:   []string{},
		Files:  []graphmodel.RenderLayoutFileWarning{},
	}
	warnings.Layout = append(warnings.Layout, layoutWarnings...)

	// 3. Resolve note.
	nvs := env.LatestNoteViews()
	var noteView *templateviews.Note

	if input.Note != nil {
		switch {
		case input.Note.Path != "":
			nv := nvs.GetByPath(input.Note.Path)
			if nv == nil {
				return nil, fmt.Errorf("note not found: %s", input.Note.Path)
			}
			noteView = templateviews.NewNote(nv)
		case input.Note.Src != nil && *input.Note.Src != "":
			noteView = templateviews.NewNote(renderMarkdownNote(*input.Note.Src))
		}
	}

	// 4. Execute template.
	var buf bytes.Buffer
	if layout.View != nil {
		vars := make(jet.VarMap)
		if noteView != nil {
			vars["note"] = reflect.ValueOf(noteView)
		}
		vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(nvs, "latest"))
		if err := layout.View.Execute(&buf, vars, nil); err != nil {
			warnings.Layout = append(warnings.Layout, "runtime: "+err.Error())
		}
	}

	// 5. Store in buffer (preserving flat warning list for buffer metadata).
	flatWarnings := append([]string{}, warnings.Layout...)
	entry := env.PreviewBuffer().Push(buf.String(), flatWarnings)

	return &graphmodel.RenderLayoutPayload{
		PreviewID:  entry.ID,
		PreviewURL: "/_system/renderlayout?preview_id=" + entry.ID,
		Warnings:   warnings,
	}, nil
}

func renderMarkdownNote(content string) *model.NoteView {
	md := goldmark.New()
	src := []byte(content)
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)
	var buf bytes.Buffer
	_ = md.Renderer().Render(&buf, src, doc)
	return &model.NoteView{
		Content: src,
		HTML:    template.HTML(buf.String()), //nolint:gosec // goldmark output is trusted; content is user-supplied markdown rendered server-side
	}
}
