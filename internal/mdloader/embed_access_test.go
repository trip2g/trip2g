package mdloader_test

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

func loadEmbedAccess(t *testing.T, sources map[string]string) map[string]string {
	t.Helper()

	files := make([]mdloader.SourceFile, 0, len(sources))
	for path, content := range sources {
		files = append(files, mdloader.SourceFile{Path: path, Content: []byte(content)})
	}

	notes, err := mdloader.Load(mdloader.Options{Log: &logger.TestLogger{}, Sources: files})
	require.NoError(t, err)

	res := make(map[string]string, len(sources))
	for path := range sources {
		res[path] = string(notes.PathMap[path].HTML)
	}

	return res
}

func TestEmbedFreeIntoFreeInlines(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\nfree: true\n---\nbefore ![[open]]\n",
		"open.md":   "---\nfree: true\n---\nzzopenzz\n",
	})

	require.Contains(t, html["parent.md"], "zzopenzz")
}

func TestEmbedClosedIntoFreeBecomesLinkOnly(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\nfree: true\n---\nbefore ![[closed]]\n",
		"closed.md": "---\nsubgraph: other\n---\nzzsecretzz\n",
	})

	require.NotContains(t, html["parent.md"], "zzsecretzz")
	require.NotContains(t, html["parent.md"], "embedded-note")
	require.Contains(t, html["parent.md"], `href="/closed"`)
	require.Contains(t, html["parent.md"], "closed</a>")
}

func TestEmbedClosedIntoFreeNestedStaysHidden(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\nfree: true\n---\n![[middle]]\n",
		"middle.md": "---\nfree: true\n---\nzzmiddlezz ![[closed]]\n",
		"closed.md": "---\nsubgraph: other\n---\nzzsecretzz\n",
	})

	require.Contains(t, html["parent.md"], "zzmiddlezz")
	require.NotContains(t, html["parent.md"], "zzsecretzz")
	require.NotContains(t, html["middle.md"], "zzsecretzz")
}

func TestEmbedNestedThroughClosedMiddleStaysHidden(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\nfree: true\n---\n![[middle]]\n",
		"middle.md": "---\nsubgraph: other\n---\nzzmiddlezz ![[deep]]\n",
		"deep.md":   "---\nfree: true\n---\nzzdeepzz\n",
	})

	require.NotContains(t, html["parent.md"], "zzmiddlezz")
	require.NotContains(t, html["parent.md"], "zzdeepzz")
	require.Contains(t, html["middle.md"], "zzdeepzz")
}

func TestEmbedClosedIntoClosedSameSubgraphInlines(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\nsubgraph: a\n---\n![[inner]]\n",
		"inner.md":  "---\nsubgraph: a\n---\nzzinnerzz\n",
	})

	require.Contains(t, html["parent.md"], "zzinnerzz")
}

func TestEmbedClosedIntoClosedOtherSubgraphIsLinkOnly(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\nsubgraph: a\n---\n![[inner]]\n",
		"inner.md":  "---\nsubgraph: b\n---\nzzinnerzz\n",
	})

	require.NotContains(t, html["parent.md"], "zzinnerzz")
	require.Contains(t, html["parent.md"], `href="/inner"`)
}

func TestEmbedWiderParentIntoNarrowerTargetIsLinkOnly(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\nsubgraphs: [a, b]\n---\n![[inner]]\n",
		"inner.md":  "---\nsubgraph: a\n---\nzzinnerzz\n",
	})

	require.NotContains(t, html["parent.md"], "zzinnerzz")
}

func TestEmbedNarrowerParentIntoWiderTargetInlines(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\nsubgraph: a\n---\n![[inner]]\n",
		"inner.md":  "---\nsubgraphs: [a, b]\n---\nzzinnerzz\n",
	})

	require.Contains(t, html["parent.md"], "zzinnerzz")
}

func TestEmbedSubgraphParentIntoUngatedTargetIsLinkOnly(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\n---\n![[inner]]\n",
		"inner.md":  "---\nsubgraph: a\n---\nzzinnerzz\n",
	})

	require.NotContains(t, html["parent.md"], "zzinnerzz")
}

func TestEmbedClosedIntoFreePreviewIsLinkOnly(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md": "---\nsubgraph: a\nfree_paragraphs: 1\n---\n![[inner]]\n",
		"inner.md":  "---\nsubgraph: a\n---\nzzinnerzz\n",
	})

	require.Contains(t, html["parent.md"], "zzinnerzz")

	notes, err := mdloader.Load(mdloader.Options{
		Log: &logger.TestLogger{},
		Sources: []mdloader.SourceFile{
			{Path: "parent.md", Content: []byte("---\nsubgraph: a\nfree_paragraphs: 1\n---\n![[inner]]\n")},
			{Path: "inner.md", Content: []byte("---\nsubgraph: a\n---\nzzinnerzz\n")},
		},
	})
	require.NoError(t, err)
	require.NotContains(t, string(notes.PathMap["parent.md"].FreeHTML), "zzinnerzz")
}

func TestEmbedUngatedSystemPartialIntoFreeInlines(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md":   "---\nfree: true\n---\n![[_partial]]\n",
		"_partial.md": "zzpartialzz\n",
	})

	require.Contains(t, html["parent.md"], "zzpartialzz")
}

func TestEmbedGatedSystemNoteIntoFreeIsLinkOnly(t *testing.T) {
	html := loadEmbedAccess(t, map[string]string{
		"parent.md":   "---\nfree: true\n---\n![[_partial]]\n",
		"_partial.md": "---\nsubgraph: a\n---\nzzpartialzz\n",
	})

	require.NotContains(t, html["parent.md"], "zzpartialzz")
}
