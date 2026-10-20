package runnotequery_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"trip2g/internal/case/runnotequery"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

// BenchmarkResolveDocs runs a simple query over the repo's docs/ vault
// (~850 notes). Access checks are mocked, so a signed-in reader's
// per-note subscription lookup is not included.
//
//	go test ./internal/case/runnotequery -run '^$' -bench . -benchmem
func BenchmarkResolveDocs(b *testing.B) {
	var srcs []mdloader.SourceFile
	err := filepath.WalkDir("../../../docs", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		content, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel("../../../docs", p)
		srcs = append(srcs, mdloader.SourceFile{Path: filepath.ToSlash(rel), Content: content})
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
	srcs = append(srcs, mdloader.SourceFile{
		Path:    "queries/en_user.md",
		Content: []byte(query(`[{title: n.title, url: n.permalink} for n in notes if std.startsWith(n.path, 'en/user/')]`)),
	})

	log := logger.DummyLogger{}
	nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log})
	if err != nil {
		b.Fatal(err)
	}
	env := &runnotequery.EnvMock{
		LatestNoteViewsFunc:  func() *appmodel.NoteViews { return nvs },
		CachedQueryNotesFunc: runnotequery.NewNotesCache().CachedQueryNotes,
		ObserveNoteQueryFunc: func(string, string, time.Duration, int) {},
		CurrentUserTokenFunc: func(context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 1, Role: "admin"}, nil
		},
		CurrentFederatedScopeFunc: func(context.Context) ([]string, bool) { return nil, false },
	}

	for b.Loop() {
		res, resErr := runnotequery.Resolve(context.Background(), env, model.NoteQueryInput{Query: "queries/en_user.md"})
		if resErr != nil {
			b.Fatal(resErr)
		}
		if _, ok := res.(*model.NoteQueryPayload); !ok {
			b.Fatalf("unexpected %#v", res)
		}
	}
}
