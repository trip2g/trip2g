package mdloader_test

import (
	"fmt"
	"strings"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func hasWarning(note *model.NoteView, substr string) bool {
	for _, w := range note.Warnings {
		if strings.Contains(w.Message, substr) {
			return true
		}
	}
	return false
}

func TestEmbedOfEmptyNoteKeepsHostContent(t *testing.T) {
	notes, err := mdloader.Load(mdloader.Options{
		Log: &logger.TestLogger{},
		Sources: []mdloader.SourceFile{
			{Path: "a.md", Content: []byte("Before ![[b]] after")},
			{Path: "b.md", Content: []byte("---\ntitle: B\n---\n")},
		},
	})
	require.NoError(t, err)

	html := string(notes.PathMap["a.md"].HTML)
	require.Contains(t, html, "Before")
	require.Contains(t, html, "after")
}

func TestEmbedCycleKeepsContentAndWarns(t *testing.T) {
	notes, err := mdloader.Load(mdloader.Options{
		Log: &logger.TestLogger{},
		Sources: []mdloader.SourceFile{
			{Path: "a.md", Content: []byte("A text ![[b]]")},
			{Path: "b.md", Content: []byte("B text ![[a]]")},
			{Path: "self.md", Content: []byte("Self text ![[self]]")},
		},
	})
	require.NoError(t, err)

	a := notes.PathMap["a.md"]
	b := notes.PathMap["b.md"]
	self := notes.PathMap["self.md"]

	// Paths are rendered in sorted order: a starts first, so b is rendered
	// while a is in progress and b's embed of a is the one that breaks.
	require.Contains(t, string(a.HTML), "A text")
	require.Contains(t, string(a.HTML), "B text")
	require.Contains(t, string(b.HTML), "B text")
	require.True(t, hasWarning(b, "embed cycle"), "warnings: %v", b.Warnings)

	require.Contains(t, string(self.HTML), "Self text")
	require.True(t, hasWarning(self, "embed cycle"), "warnings: %v", self.Warnings)
}

func TestLongEmbedChain(t *testing.T) {
	const n = 40
	for _, reversed := range []bool{false, true} {
		t.Run(fmt.Sprintf("reversed=%v", reversed), func(t *testing.T) {
			var sources []mdloader.SourceFile
			for i := range n {
				content := fmt.Sprintf("note%d ![[n%d]]", i, i+1)
				if i == n-1 {
					content = "End"
				}
				sources = append(sources, mdloader.SourceFile{
					Path:    fmt.Sprintf("n%d.md", i),
					Content: []byte(content),
				})
			}
			if reversed {
				for i, j := 0, len(sources)-1; i < j; i, j = i+1, j-1 {
					sources[i], sources[j] = sources[j], sources[i]
				}
			}

			notes, err := mdloader.Load(mdloader.Options{Sources: sources, Log: &logger.TestLogger{}})
			require.NoError(t, err)

			for i := range n {
				html := string(notes.PathMap[fmt.Sprintf("n%d.md", i)].HTML)
				require.Contains(t, html, "End", "n%d", i)
			}
		})
	}
}

// An absolute-path embed isn't in ResolvedLinks but still resolves at render
// time; it must be rendered before the note that embeds it.
func TestEmbedByAbsolutePath(t *testing.T) {
	notes, err := mdloader.Load(mdloader.Options{
		Log: &logger.TestLogger{},
		Sources: []mdloader.SourceFile{
			{Path: "a.md", Content: []byte("![[/zz]]")},
			{Path: "zz.md", Content: []byte("hello embedded")},
		},
	})
	require.NoError(t, err)
	require.Contains(t, string(notes.PathMap["a.md"].HTML), "hello embedded")
}

func TestEmbedCycleWarnsOnce(t *testing.T) {
	notes, err := mdloader.Load(mdloader.Options{
		Log:    &logger.TestLogger{},
		Config: mdloader.Config{FreeParagraphs: 3},
		Sources: []mdloader.SourceFile{
			{Path: "self.md", Content: []byte("Self text ![[self]]")},
		},
	})
	require.NoError(t, err)

	count := 0
	for _, w := range notes.PathMap["self.md"].Warnings {
		if strings.Contains(w.Message, "embed cycle") {
			count++
		}
	}
	require.Equal(t, 1, count)
}
