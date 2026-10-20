package doclint

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/case/runnotequery"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"
	"trip2g/internal/templateviews"
	"trip2g/internal/usertoken"
)

// guestQueryEnv runs note queries as a guest over the docs vault.
type guestQueryEnv struct{ nvs *model.NoteViews }

func (e guestQueryEnv) LatestNoteViews() *model.NoteViews {
	return e.nvs
}

func (guestQueryEnv) CurrentUserToken(context.Context) (*usertoken.Data, error) {
	return nil, nil
}

func (guestQueryEnv) CurrentFederatedScope(context.Context) ([]string, bool) {
	return nil, false
}

func (guestQueryEnv) ListActiveUserSubgraphs(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (guestQueryEnv) ObserveNoteQuery(string, string, time.Duration, int) {}

func (guestQueryEnv) CachedQueryNotes(_ uint64, _ canreadnote.Class, _ bool, build func() ([]byte, error)) ([]byte, error) {
	return build()
}

func renderDocsLanding(t *testing.T, permalink string) string {
	t.Helper()

	env := newFsEnv("../../docs", &logger.DummyLogger{})
	ldr := noteloader.New("landing", env, mdloader.Config{})
	err := ldr.Load(context.Background(), noteloader.LoadOptions{SkipSearchIndex: true})
	require.NoError(t, err)

	nvs := ldr.NoteViews()
	page := nvs.GetByPath(permalink)
	require.NotNil(t, page, "no note at %s", permalink)

	layout, ok := ldr.Layouts().Map["/"+page.Layout]
	require.True(t, ok, "layout %q not loaded", page.Layout)
	require.NotNil(t, layout.View, "layout %q did not parse: %+v", page.Layout, layout.Warnings)

	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(templateviews.NewNote(page))
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(nvs, ""))
	vars["title"] = reflect.ValueOf(page.Title)
	vars["publicURL"] = reflect.ValueOf("https://trip2g.com")
	vars["htmlInjectionsHead"] = reflect.ValueOf([]db.HtmlInjection{})
	vars["htmlInjectionsBodyEnd"] = reflect.ValueOf([]db.HtmlInjection{})
	vars["noteQueryRows"] = reflect.ValueOf(runnotequery.JetRowsFunc(context.Background(), guestQueryEnv{nvs: nvs}, ""))

	var out strings.Builder
	err = layout.View.Execute(&out, vars, nil)
	require.NoError(t, err)

	for _, w := range layout.Warnings {
		require.NotEqual(t, model.NoteWarningCritical, w.Level, w.Message)
	}

	return out.String()
}

func TestLanding_RendersHeroAndSections(t *testing.T) {
	tests := []struct {
		permalink string
		want      []string
	}{
		{
			permalink: "/",
			want: []string{
				"<title>Agents write it. People read it in a browser — trip2g</title>",
				"Agents write it. <em>People read it in a browser.</em>",
				"open source · MIT · one Go process on your own server",
				`class="mesh-week"`,
				"One week, one trip2g",
				`href="/en/user/agent_memory"`,
				`href="/en/user/default_template"`,
				"Five jobs. One process.",
				`class="mesh-knife"`,
				"A markdown Swiss army knife",
				`href="/en/thoughts/anatomy_15_months"`,
				".mesh-week h2 {",
				".mesh-knife h2 {",
			},
		},
		{
			permalink: "/ru",
			want: []string{
				"<title>Агенты пишут — люди читают в браузере — trip2g</title>",
				"Агенты пишут\u00a0— <em>люди читают в браузере</em>",
				"open source · MIT · один Go-процесс на вашем сервере",
				`class="mesh-week"`,
				"Одна неделя, один trip2g",
				`href="/ru/user/agent_memory"`,
				`href="/ru/user/default_template"`,
				"Пять задач. Один процесс.",
				`class="mesh-knife"`,
				"Швейцарский нож на markdown",
				`href="/ru/thoughts/anatomy_15_months"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.permalink, func(t *testing.T) {
			out := renderDocsLanding(t, tt.permalink)
			for _, w := range tt.want {
				require.Contains(t, out, w)
			}
			order := []string{
				`class="mesh-hero_walk"`,
				`class="mesh-week"`,
				`class="mesh-knife"`,
				`class="how"`,
				`class="mesh-capabilities"`,
			}
			prev := -1
			for _, marker := range order {
				at := strings.Index(out, marker)
				require.Greater(t, at, prev, "%s out of order", marker)
				prev = at
			}
		})
	}
}

// TestWheel_PointsComeFromTheQuery checks the wheel lists its folder's labelled
// notes in `order`, through queries/circle_points.md, on the page and on a
// point's own URL.
func TestWheel_PointsComeFromTheQuery(t *testing.T) {
	want := `data-slug="garden" data-slug="channel" data-slug="subscription" data-slug="memory" ` +
		`data-slug="agents" data-slug="company" data-slug="bot" data-slug="calls" data-slug="files"`
	slug := regexp.MustCompile(`data-slug="[a-z]+"`)

	for _, permalink := range []string{"/en/circle", "/ru/circle", "/en/circle/garden"} {
		t.Run(permalink, func(t *testing.T) {
			out := renderDocsLanding(t, permalink)
			require.Equal(t, want, strings.Join(slug.FindAllString(out, -1), " "))
			require.Equal(t, 9, strings.Count(out, `role="tabpanel"`))
			require.NotContains(t, out, `class="wrap"`, "the wheel is full width")
		})
	}
}
