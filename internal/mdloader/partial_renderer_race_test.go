package mdloader_test

import (
	"sync"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// TestPartialRendererConcurrent renders partials of different notes and of the
// same note from many goroutines, the way concurrent page requests do for the
// footer and cards. Run with -race.
func TestPartialRendererConcurrent(t *testing.T) {
	sources := []mdloader.SourceFile{
		{Path: "a.md", Content: []byte("![[pic.png]]\n\n## One\n\ntext\n\n## Two\n\n![[pic.png]]"),
			Assets: map[string]*model.NoteAssetReplace{"pic.png": {URL: "/a.png"}}},
		{Path: "b.md", Content: []byte("![[pic.png]]\n\n## One\n\ntext\n\n## Two\n\n![[pic.png]]"),
			Assets: map[string]*model.NoteAssetReplace{"pic.png": {URL: "/b.png"}}},
	}
	notes, err := mdloader.Load(mdloader.Options{Sources: sources, Log: &logger.TestLogger{}})
	require.NoError(t, err)

	a := notes.PathMap["a.md"]
	b := notes.PathMap["b.md"]

	var wg sync.WaitGroup
	for _, n := range []*model.NoteView{a, b, a, b} {
		wg.Add(1)
		go func(n *model.NoteView) {
			defer wg.Done()
			for range 50 {
				n.PartialRenderer.Introduce()
				n.PartialRenderer.Sections(2)
				n.PartialRenderer.Section("Two")
			}
		}(n)
	}
	wg.Wait()

	require.Contains(t, a.PartialRenderer.Introduce().ContentHTML, "/a.png")
	require.Contains(t, b.PartialRenderer.Introduce().ContentHTML, "/b.png")
	require.Contains(t, a.PartialRenderer.Sections(2)[1].ContentHTML, "/a.png")
	require.Contains(t, b.PartialRenderer.Sections(2)[1].ContentHTML, "/b.png")
}
