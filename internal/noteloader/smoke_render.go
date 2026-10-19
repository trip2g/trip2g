package noteloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"time"
	"trip2g/internal/case/runnotequery"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"
	"trip2g/internal/usertoken"

	"github.com/CloudyKit/jet/v6"
)

const smokeRenderLimit = 10

// smokeQueryEnv runs noteQuery during the smoke render as a guest over the
// notes being loaded.
type smokeQueryEnv struct{ nvs *model.NoteViews }

func (e smokeQueryEnv) LatestNoteViews() *model.NoteViews {
	return e.nvs
}

func (smokeQueryEnv) CurrentUserToken(context.Context) (*usertoken.Data, error) {
	return nil, nil
}

func (smokeQueryEnv) CurrentFederatedScope(context.Context) ([]string, bool) {
	return nil, false
}

func (smokeQueryEnv) ListActiveUserSubgraphs(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (smokeQueryEnv) ObserveNoteQuery(string, string, time.Duration, int) {}

// smokeRenderLayouts executes each parsed layout against up to smokeRenderLimit
// notes that select it via frontmatter `layout:`. Runtime errors and panics
// from Jet's Execute are turned into NoteWarning entries on the layout so that
// CLI / pushNotes flows surface them without needing a browser request.
//
// Skipped:
//   - layouts with parse errors (View == nil) — already have Critical warning
//   - layouts no note uses — nothing meaningful to render against
func smokeRenderLayouts(layouts *model.Layouts, nvs *model.NoteViews, log logger.Logger) {
	if layouts == nil || nvs == nil {
		return
	}

	byLayout := make(map[string][]*model.NoteView)
	for _, n := range nvs.List {
		if n == nil || n.Layout == "" {
			continue
		}
		key := "/" + n.Layout
		if len(byLayout[key]) >= smokeRenderLimit {
			continue
		}
		byLayout[key] = append(byLayout[key], n)
	}

	for id, layout := range layouts.Map {
		if layout.View == nil {
			continue
		}
		notes, ok := byLayout[id]
		if !ok {
			continue
		}
		warnings := smokeRenderForLayout(layout.View, notes, nvs, log, id)
		if len(warnings) > 0 {
			layout.Warnings = append(layout.Warnings, warnings...)
			layouts.Map[id] = layout
		}
	}
}

func smokeRenderForLayout(view *jet.Template, notes []*model.NoteView, nvs *model.NoteViews, log logger.Logger, layoutID string) []model.NoteWarning {
	var warnings []model.NoteWarning
	nvsWrap := templateviews.NewNVS(nvs, "")
	noteQuery := memoNoteQuery(runnotequery.JetFunc(context.Background(), smokeQueryEnv{nvs: nvs}))
	for _, n := range notes {
		if w := executeSmoke(view, n, nvsWrap, noteQuery); w != nil {
			log.Warn("layout smoke render failed",
				"layout", layoutID, "note", n.Path, "msg", w.Message)
			warnings = append(warnings, *w)
		}
	}
	return warnings
}

type noteQueryOutcome struct {
	value reflect.Value
	panic any
}

// memoNoteQuery runs each distinct (query, params) once per smoke render: every
// sampled note gets the same guest answer, so re-running it only costs time.
// A failure is remembered too and raised again for each note.
func memoNoteQuery(next jet.Func) jet.Func {
	seen := make(map[string]noteQueryOutcome)
	return func(a jet.Arguments) reflect.Value {
		key, ok := noteQueryKey(a)
		if !ok {
			return next(a)
		}
		outcome, done := seen[key]
		if !done {
			outcome = callNoteQuery(next, a)
			seen[key] = outcome
		}
		if outcome.panic != nil {
			panic(outcome.panic)
		}
		return outcome.value
	}
}

func noteQueryKey(a jet.Arguments) (string, bool) {
	if a.NumOfArguments() < 1 || a.NumOfArguments() > 2 {
		return "", false
	}
	args := make([]any, a.NumOfArguments())
	for i := range args {
		v := a.Get(i)
		if !v.IsValid() {
			return "", false
		}
		args[i] = v.Interface()
	}
	key, err := json.Marshal(args)
	if err != nil {
		return "", false
	}
	return string(key), true
}

//nolint:nonamedreturns // named return required so deferred recover can set it
func callNoteQuery(next jet.Func, a jet.Arguments) (outcome noteQueryOutcome) {
	defer func() {
		if r := recover(); r != nil {
			outcome.panic = r
		}
	}()
	return noteQueryOutcome{value: next(a)}
}

//nolint:nonamedreturns // named return required so deferred recover can set it
func executeSmoke(view *jet.Template, note *model.NoteView, nvsWrap *templateviews.NVS, noteQuery jet.Func) (w *model.NoteWarning) {
	defer func() {
		if r := recover(); r != nil {
			w = &model.NoteWarning{
				Level:   model.NoteWarningWarning,
				Message: fmt.Sprintf("smoke render panic on %q: %v", note.Path, r),
			}
		}
	}()

	vars := make(jet.VarMap)
	// Wrap in templateviews.Note so smoke render matches request-time rendering,
	// which exposes methods like M() that layouts call. Passing the raw
	// *model.NoteView here caused false "can't use M as field name" warnings.
	vars["note"] = reflect.ValueOf(templateviews.NewNote(note))
	vars["nvs"] = reflect.ValueOf(nvsWrap)
	vars["title"] = reflect.ValueOf(note.Title)
	vars["publicURL"] = reflect.ValueOf("")
	vars["htmlInjectionsHead"] = reflect.ValueOf([]db.HtmlInjection{{}})
	vars["htmlInjectionsBodyEnd"] = reflect.ValueOf([]db.HtmlInjection{{}})
	// Stub namespaces that custom layouts call but that require a live HTTP
	// request context to populate. No-op functions return zero values so
	// layouts render without errors during smoke-checks.
	vars["defaultTemplate"] = reflect.ValueOf(map[string]interface{}{
		"UserSpaceScripts": func() model.SafeHTML { return "" },
		"Header":           func() model.SafeHTML { return "" },
		"Footer":           func() model.SafeHTML { return "" },
		"Styles":           func() model.SafeHTML { return "" },
	})
	vars["currentUser"] = reflect.ValueOf(map[string]interface{}{
		"IsAdmin": func() bool { return false },
	})
	vars["noteQuery"] = reflect.ValueOf(noteQuery)

	if err := view.Execute(io.Discard, vars, nil); err != nil {
		return &model.NoteWarning{
			Level:   model.NoteWarningWarning,
			Message: fmt.Sprintf("smoke render error on %q: %s", note.Path, err.Error()),
		}
	}
	return nil
}
