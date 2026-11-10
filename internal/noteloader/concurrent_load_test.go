package noteloader_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"trip2g/internal/mdloader"
	"trip2g/internal/noteloader"

	"github.com/stretchr/testify/require"
)

// Two overlapping Loads used to write the search index maps at the same time
// (a fatal "concurrent map writes"), and the one that finished last published
// its snapshot even when it had read older notes. Run with -race.
func TestLoader_ConcurrentLoads(t *testing.T) {
	var mu sync.Mutex
	reads := 0

	env := makeMinimalEnv(nil, func() bool { return false })
	env.RawNotesFunc = func(_ context.Context) ([]noteloader.RawNote, error) {
		mu.Lock()
		reads++
		read := reads
		mu.Unlock()

		notes := make([]noteloader.RawNote, 50)
		for i := range notes {
			notes[i] = noteloader.RawNote{
				Path:      fmt.Sprintf("n%d.md", i),
				PathID:    int64(i + 1),
				VersionID: int64(i + 100),
				Content:   fmt.Sprintf("# n%d\nword%d read%d", i, i, read),
			}
		}
		return notes, nil
	}

	loader := noteloader.New("test", env, mdloader.Config{})

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = loader.Load(context.Background(), noteloader.LoadOptions{})
		}()
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	note := loader.NoteByPath("/n0")
	require.NotNil(t, note)
	require.True(t, strings.HasSuffix(string(note.Content), "read2"),
		"published snapshot must come from the last read, got %q", note.Content)
}
