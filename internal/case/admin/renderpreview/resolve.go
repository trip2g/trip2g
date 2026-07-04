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
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
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
		vars["title"] = reflect.ValueOf(previewTitle(noteView))
		vars["publicURL"] = reflect.ValueOf("")
		// Set empty injection slices to prevent "identifier not available" errors
		// in layouts that call {{ range injection := htmlInjectionsHead }}.
		vars["htmlInjectionsHead"] = reflect.ValueOf([]struct{}{})
		vars["htmlInjectionsBodyEnd"] = reflect.ValueOf([]struct{}{})
		// Stub namespaces the real page render injects (rendernotepage). They need
		// a live HTTP request context to populate; no-op values keep layouts that
		// call defaultTemplate.Styles() / currentUser.IsAdmin() previewable.
		vars["defaultTemplate"] = reflect.ValueOf(map[string]interface{}{
			"UserSpaceScripts": func() string { return "" },
			"Header":           func() string { return "" },
			"Footer":           func() string { return "" },
			"Styles":           func() string { return "" },
		})
		vars["currentUser"] = reflect.ValueOf(map[string]interface{}{
			"IsAdmin": func() bool { return false },
		})
		if err := executePreview(env, layout.View, &buf, vars); err != nil {
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

// executePreview runs a user-authored layout. Jet's Execute re-panics
// runtime errors and non-error panics, so without a recover a bad layout
// would kill the server process (same guard as rendernotepage.renderLayout).
func executePreview(env Env, view *jet.Template, buf *bytes.Buffer, vars jet.VarMap) (err error) {
	defer func() {
		if r := recover(); r != nil {
			env.Logger().Error("template panic", "layout", "preview", "error", r)
			err = fmt.Errorf("template panic: %v", r)
		}
	}()
	return view.Execute(buf, vars, nil)
}

func previewTitle(noteView *templateviews.Note) string {
	if noteView == nil {
		return ""
	}
	return noteView.Title()
}

func renderMarkdownNote(content string) *model.NoteView {
	md := goldmark.New(goldmark.WithExtensions(meta.Meta))
	src := []byte(content)
	ctx := parser.NewContext()
	doc := md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))
	var buf bytes.Buffer
	_ = md.Renderer().Render(&buf, src, doc)
	nv := &model.NoteView{
		Content: src,
		HTML:    template.HTML(buf.String()), //nolint:gosec // goldmark output is trusted; content is user-supplied markdown rendered server-side
		RawMeta: meta.Get(ctx),
	}
	nv.SetAst(doc)
	nv.Title = nv.ExtractTitle()
	return nv
}
