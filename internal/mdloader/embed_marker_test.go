package mdloader_test

import (
	"testing"

	"trip2g/internal/embedguard"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func loadWithIDs(t *testing.T, sources []mdloader.SourceFile, options mdloader.Options) *model.NoteViews {
	t.Helper()

	options.Log = &logger.TestLogger{}
	options.Sources = sources

	notes, err := mdloader.Load(options)
	require.NoError(t, err)

	return notes
}

func TestEmbedIsWrappedInMarkerCarryingTheNoteID(t *testing.T) {
	notes := loadWithIDs(t, []mdloader.SourceFile{
		{Path: "parent.md", PathID: 1, Content: []byte("---\nfree: true\n---\n![[open]]\n")},
		{Path: "open.md", PathID: 2, Content: []byte("---\nfree: true\n---\nzzopenzz\n")},
	}, mdloader.Options{})

	html := string(notes.PathMap["parent.md"].HTML)

	require.Contains(t, html, embedguard.Wrap(2, "")[:len("<!--t2e:2-->")])
	require.Contains(t, html, "zzopenzz")
}

func TestUngatedPartialCannotLaunderAClosedEmbed(t *testing.T) {
	notes := loadWithIDs(t, []mdloader.SourceFile{
		{Path: "free.md", PathID: 1, Content: []byte("---\nfree: true\n---\n![[_partial]]\n")},
		{Path: "_partial.md", PathID: 2, Content: []byte("partial ![[closed]]\n")},
		{Path: "closed.md", PathID: 3, Content: []byte("zzsecretzz\n")},
	}, mdloader.Options{})

	html := string(notes.PathMap["free.md"].HTML)

	deny3 := func(id int64) (bool, string) { return id != 3, "<a>closed</a>" }
	stripped := embedguard.Strip(html, deny3)

	require.Contains(t, html, "<!--t2e:3-->", "the closed embed stays marked inside the partial")
	require.Contains(t, stripped, "partial")
	require.NotContains(t, stripped, "zzsecretzz")
}

func TestRequireSigninFlagIsKnownBeforeRendering(t *testing.T) {
	sources := []mdloader.SourceFile{
		{Path: "paid.md", PathID: 1, Content: []byte("---\nsubgraph: paid\n---\n![[members]]\n")},
		{Path: "members.md", PathID: 2, Content: []byte("---\nsubgraph: members\n---\nzzmembersonlyzz\n")},
	}

	withoutFlag := loadWithIDs(t, sources, mdloader.Options{})
	require.NotContains(t, string(withoutFlag.PathMap["paid.md"].HTML), "zzmembersonlyzz")

	withFlag := loadWithIDs(t, sources, mdloader.Options{
		RequireSigninSubgraphs: map[string]bool{"members": true},
	})
	require.Contains(t, string(withFlag.PathMap["paid.md"].HTML), "zzmembersonlyzz")
}
