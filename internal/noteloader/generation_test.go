package noteloader_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/mdloader"
	"trip2g/internal/noteloader"
)

// The published views carry the Load that built them, so a cache keyed by
// it can't mix a snapshot with another Load's generation.
func TestLoader_NoteViewsCarryGeneration(t *testing.T) {
	notes := []noteloader.RawNote{{Path: "a.md", PathID: 1, VersionID: 11, Content: "a"}}
	loader := noteloader.New("test", makeMinimalEnv(notes, func() bool { return false }), mdloader.Config{})
	ctx := context.Background()

	for range 2 {
		require.NoError(t, loader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}))
		require.Equal(t, loader.Generation(), loader.NoteViews().Generation)
	}
	require.Equal(t, uint64(2), loader.NoteViews().Generation)
}
