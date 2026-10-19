package noteloader_test

import (
	"context"
	"testing"
	"trip2g/internal/mdloader"
	"trip2g/internal/noteloader"

	"github.com/stretchr/testify/require"
)

func TestLoader_FreshLoaderServesAnEmptySnapshot(t *testing.T) {
	env := makeMinimalEnv(nil, func() bool { return false })
	env.RawNoteEmbeddingsFunc = func(_ context.Context) ([]noteloader.RawNoteEmbedding, error) {
		return []noteloader.RawNoteEmbedding{{VersionID: 1}}, nil
	}
	loader := noteloader.New("test", env, mdloader.Config{})

	nvs := loader.NoteViews()
	require.NotNil(t, nvs)
	require.Empty(t, nvs.List)
	require.Nil(t, nvs.GetByVersionID(1))
	require.Nil(t, loader.NoteByPath("/note"))

	layouts := loader.Layouts()
	require.NotNil(t, layouts)
	_, ok := layouts.Map["/default"]
	require.False(t, ok)
	require.Empty(t, layouts.Blocks.All())

	require.NoError(t, loader.ReloadEmbeddings(context.Background()))
	require.Empty(t, loader.NoteViews().List)
}
