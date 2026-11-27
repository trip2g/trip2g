package rendernotepage

import (
	"bytes"
	"context"
	"html/template"
	"os"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/embedguard"
	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

type freeOnlyReader struct{}

func (freeOnlyReader) CanReadNote(_ context.Context, note *model.NoteView) (bool, error) {
	return note.IsAnonymouslyReadable(), nil
}

func TestRSSLayout_FeedItemDoesNotCarryAClosedEmbed(t *testing.T) {
	layoutBytes, err := os.ReadFile(shippedRSSLayoutPath)
	require.NoError(t, err)

	created := time.Date(2024, 3, 2, 10, 30, 0, 0, time.UTC)
	closed := &model.NoteView{
		Path: "private/closed.md", Title: "Closed", Permalink: "/private/closed", PathID: 2,
		HTML: template.HTML("<p>zzsecretzz</p>"),
	}
	free := &model.NoteView{
		Path: "blog/free.md", Title: "Free Note", Permalink: "/blog/free", PathID: 1,
		Free: true, CreatedAt: created, Description: strptr("A free note"),
		HTML: template.HTML("<p>intro</p>" + embedguard.Wrap(2, string(closed.HTML))),
	}

	nvs := model.NewNoteViews()
	nvs.PathMap[free.Path] = free
	nvs.PathMap[closed.Path] = closed

	feed := &model.NoteView{
		Path: "feed.md", Title: "Feed", Permalink: "/feed.xml", Free: true,
		RawMeta: map[string]interface{}{"rss_glob": "blog/*.md", "rss_limit": 20},
	}

	layouts, err := layoutloader.Load(&loaderTestEnv{log: &logger.TestLogger{}}, []model.LayoutSourceFile{{
		ID: "/rss", Path: "_layouts/rss.html", Content: string(layoutBytes),
	}}, layoutloader.Options{})
	require.NoError(t, err)

	viewer := embedguard.NewViewer(context.Background(), freeOnlyReader{}, nvs)

	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(templateviews.NewNote(feed))
	vars["nvs"] = reflect.ValueOf(templateviews.NVSWithGuard(templateviews.NewNVS(nvs, "live"), viewer.Strip))
	vars["publicURL"] = reflect.ValueOf("https://example.com")

	var buf bytes.Buffer
	require.NoError(t, layouts.Map["/rss"].View.Execute(&buf, vars, nil))

	out := buf.String()
	require.Contains(t, out, "intro")
	require.NotContains(t, out, "zzsecretzz")
	require.NotContains(t, out, "t2e:")
}
