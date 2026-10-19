package noteloader

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

type smokeTestEnv struct{ log logger.Logger }

func (e *smokeTestEnv) Logger() logger.Logger { return e.log }
func (e *smokeTestEnv) IsDevMode() bool       { return false }

func smokeLoadLayouts(t *testing.T, sources []model.LayoutSourceFile) *model.Layouts {
	t.Helper()
	layouts, err := layoutloader.Load(&smokeTestEnv{log: &logger.TestLogger{}}, sources, layoutloader.Options{})
	require.NoError(t, err)
	return layouts
}

// Layout parses but blows up at render time (unknown field on note).
// Smoke render must turn that into a NoteWarning on the layout.
func TestSmokeRenderLayouts_RuntimeErrorBecomesWarning(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/buggy", Path: "_layouts/buggy.html",
			Content: `<html>{{ note.NoSuchField }}</html>`},
	}
	layouts := smokeLoadLayouts(t, sources)
	require.NotNil(t, layouts.Map["/buggy"].View, "layout should parse OK")

	nvs := &model.NoteViews{List: []*model.NoteView{
		{Path: "page.md", Layout: "buggy", Title: "p"},
	}}

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	warnings := layouts.Map["/buggy"].Warnings
	require.NotEmpty(t, warnings, "expected smoke render warning")

	var found bool
	for _, w := range warnings {
		if strings.Contains(w.Message, "smoke render") {
			found = true
		}
	}
	require.True(t, found, "expected 'smoke render' warning, got: %+v", warnings)
}

// No note uses this layout — smoke must skip it (no spurious warnings).
func TestSmokeRenderLayouts_NoMatchingNotes_NoWarning(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/orphan", Path: "_layouts/orphan.html",
			Content: `<html>{{ note.NoSuchField }}</html>`},
	}
	layouts := smokeLoadLayouts(t, sources)

	nvs := &model.NoteViews{List: []*model.NoteView{
		{Path: "page.md", Layout: "other", Title: "p"},
	}}

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	require.Empty(t, layouts.Map["/orphan"].Warnings,
		"no notes => no smoke run => no warning")
}

// Limit must cap how many notes we render per layout. A trigger note placed
// beyond the limit must NOT cause a warning.
func TestSmokeRenderLayouts_LimitFirstN(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/cond", Path: "_layouts/cond.html",
			Content: `{{ if note.Title == "trigger" }}{{ note.NoSuchField }}{{ end }}`},
	}
	layouts := smokeLoadLayouts(t, sources)

	notes := make([]*model.NoteView, 0, 12)
	for range 10 {
		notes = append(notes, &model.NoteView{
			Path: "ok.md", Layout: "cond", Title: "ok",
		})
	}
	notes = append(notes, &model.NoteView{
		Path: "boom.md", Layout: "cond", Title: "trigger",
	})
	nvs := &model.NoteViews{List: notes}

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	require.Empty(t, layouts.Map["/cond"].Warnings,
		"11th note must not be smoke-tested when limit=10")
}

// Single-variable Jet range (`range inj := slice`) assigns the loop INDEX (int),
// not the element — the same footgun as Go's `for i := range slice`.
// The smoke render must catch this when the injection slice is non-empty.
func TestSmokeRenderLayouts_SingleVarRangeOverInjectionsIsDetected(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/broken", Path: "_layouts/broken.html",
			Content: `{{ range inj := htmlInjectionsBodyEnd }}{{ inj.Content | unsafe }}{{ end }}`},
	}
	layouts := smokeLoadLayouts(t, sources)
	nvs := &model.NoteViews{List: []*model.NoteView{
		{Path: "p.md", Layout: "broken", Title: "p"},
	}}
	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	var found bool
	for _, w := range layouts.Map["/broken"].Warnings {
		if strings.Contains(w.Message, "smoke render") {
			found = true
		}
	}
	require.True(t, found,
		"single-var range over injections must produce smoke warning, got: %+v",
		layouts.Map["/broken"].Warnings)
}

// Two-variable range (`range i, inj := slice`) correctly binds the struct value.
func TestSmokeRenderLayouts_TwoVarRangeOverInjectionsIsOK(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/ok", Path: "_layouts/ok.html",
			Content: `{{ range i, inj := htmlInjectionsBodyEnd }}{{ inj.Content | unsafe }}{{ end }}`},
	}
	layouts := smokeLoadLayouts(t, sources)
	nvs := &model.NoteViews{List: []*model.NoteView{
		{Path: "p.md", Layout: "ok", Title: "p"},
	}}
	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	for _, w := range layouts.Map["/ok"].Warnings {
		if strings.Contains(w.Message, "smoke render") {
			t.Fatalf("two-var range over injections must not produce smoke warning, got: %s", w.Message)
		}
	}
}

// Parse-broken layout already has Critical warning and View == nil.
// Smoke must not touch it (no duplicate / crash).
func TestSmokeRenderLayouts_SkipsLayoutsWithParseError(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/broken", Path: "_layouts/broken.html",
			Content: `<html>{{ unterminated`},
	}
	layouts := smokeLoadLayouts(t, sources)
	require.Nil(t, layouts.Map["/broken"].View, "should be nil after parse error")
	parseWarnings := len(layouts.Map["/broken"].Warnings)
	require.Positive(t, parseWarnings, "should have parse-time warning")

	nvs := &model.NoteViews{List: []*model.NoteView{
		{Path: "p.md", Layout: "broken", Title: "p"},
	}}

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	require.Len(t, layouts.Map["/broken"].Warnings, parseWarnings,
		"smoke must not add warnings to layouts with parse errors")
}

// A page a guest can open renders its queries as a guest: an object result
// renders, and a members-only query warns. A page only signed-in viewers open
// renders them as a reader of every query, so a members-only query is fine and
// a missing one still warns.
func TestSmokeRenderLayouts_RunsNoteQueryAsTheViewer(t *testing.T) {
	sources := []model.LayoutSourceFile{
		{ID: "/dash", Path: "_layouts/dash.html",
			Content: `{{ c := noteQuery("queries/counts.md") }}{{ c.posts }}`},
		{ID: "/paid", Path: "_layouts/paid.html",
			Content: `{{ noteQuery("queries/paid.md") }}`},
		{ID: "/club", Path: "_layouts/club.html",
			Content: `{{ c := noteQuery("queries/paid.md") }}{{ len(c) }}`},
		{ID: "/typo", Path: "_layouts/typo.html",
			Content: `{{ noteQuery("queries/nope.md") }}`},
	}
	layouts := smokeLoadLayouts(t, sources)

	srcs := []mdloader.SourceFile{
		{Path: "queries/counts.md", Content: []byte("---\ntype: note-query\nfree: true\n---\n```jsonnet\n" +
			"{posts: std.length(notes)}\n```\n")},
		{Path: "queries/paid.md", Content: []byte("---\ntype: note-query\nsubgraphs: [club]\n---\n```jsonnet\n[]\n```\n")},
		{Path: "dash.md", Content: []byte("---\nfree: true\nlayout: dash\n---\nd")},
		{Path: "paid_page.md", Content: []byte("---\nfree: true\nlayout: paid\n---\np")},
		{Path: "club_page.md", Content: []byte("---\nsubgraphs: [club]\nlayout: club\n---\nc")},
		{Path: "typo_page.md", Content: []byte("---\nsubgraphs: [club]\nlayout: typo\n---\nt")},
	}
	for i := range srcs {
		srcs[i].PathID = int64(i + 1)
	}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &logger.TestLogger{}})
	require.NoError(t, err)

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	require.Empty(t, layouts.Map["/dash"].Warnings)
	require.Empty(t, layouts.Map["/club"].Warnings, "a signed-in viewer runs the members-only query")

	paid := layouts.Map["/paid"].Warnings
	require.Len(t, paid, 1, "a guest can't run a members-only query")
	require.Contains(t, paid[0].Message, `noteQuery("queries/paid.md"): query not found`)

	typo := layouts.Map["/typo"].Warnings
	require.Len(t, typo, 1, "a missing query warns on a members-only page too")
	require.Contains(t, typo[0].Message, `noteQuery("queries/nope.md"): query not found`)
}

// A layout used by more members-only pages than the sample holds still
// samples the page a guest can open, and warns for its members-only query.
func TestSmokeRenderLayouts_SamplesGuestPagesFirst(t *testing.T) {
	layouts := smokeLoadLayouts(t, []model.LayoutSourceFile{
		{ID: "/mixed", Path: "_layouts/mixed.html", Content: `{{ noteQuery("queries/paid.md") }}`},
	})

	srcs := []mdloader.SourceFile{
		{Path: "queries/paid.md", Content: []byte("---\ntype: note-query\nsubgraphs: [club]\n---\n```jsonnet\n[]\n```\n")},
	}
	for i := range smokeRenderLimit {
		srcs = append(srcs, mdloader.SourceFile{
			Path:    "club/" + strconv.Itoa(i) + ".md",
			Content: []byte("---\nsubgraphs: [club]\nlayout: mixed\n---\nc"),
		})
	}
	srcs = append(srcs, mdloader.SourceFile{Path: "zz_public.md", Content: []byte("---\nfree: true\nlayout: mixed\n---\np")})
	for i := range srcs {
		srcs[i].PathID = int64(i + 1)
	}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &logger.TestLogger{}})
	require.NoError(t, err)

	smokeRenderLayouts(layouts, nvs, &logger.TestLogger{})

	warnings := layouts.Map["/mixed"].Warnings
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0].Message, `"zz_public.md"`)
}

// A layout sampled for many notes runs each distinct noteQuery call once,
// failures included.
func TestSmokeRenderForLayout_RunsEachNoteQueryOnce(t *testing.T) {
	layouts := smokeLoadLayouts(t, []model.LayoutSourceFile{
		{ID: "/list", Path: "_layouts/list.html",
			Content: `{{ noteQuery("a") }}{{ noteQuery("a") }}{{ noteQuery("a", map("tag", "go")) }}` +
				`{{ noteQuery("a", map("tag", "web")) }}{{ noteQuery("b") }}{{ noteQuery("broken") }}`},
	})

	runs := map[string]int{}
	next := func(a jet.Arguments) reflect.Value {
		key, _ := noteQueryKey(a)
		runs[key]++
		if a.Get(0).Interface() == "broken" {
			a.Panicf("boom")
		}
		return reflect.ValueOf("ok")
	}
	noteQuery := memoNoteQuery(next)

	const sampled = 5
	nvsWrap := templateviews.NewNVS(&model.NoteViews{}, "")
	var failures int
	for i := range sampled {
		note := &model.NoteView{Path: "n" + strconv.Itoa(i) + ".md", Layout: "list"}
		if executeSmoke(layouts.Map["/list"].View, note, nvsWrap, noteQuery) != nil {
			failures++
		}
	}

	require.Equal(t, map[string]int{
		`["a"]`:               1,
		`["a",{"tag":"go"}]`:  1,
		`["a",{"tag":"web"}]`: 1,
		`["b"]`:               1,
		`["broken"]`:          1,
	}, runs)
	require.Equal(t, sampled, failures, "a failing query still fails every sampled note")
}
