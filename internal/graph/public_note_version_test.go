package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func TestPublicNoteVersionID(t *testing.T) {
	r := &publicNoteResolver{&Resolver{}}

	got, err := r.VersionID(context.Background(), &model.PublicNote{
		NoteView: &appmodel.NoteView{VersionID: 42},
	})
	require.NoError(t, err)
	require.Equal(t, int64(42), got)

	got, err = r.VersionID(context.Background(), &model.PublicNote{})
	require.NoError(t, err)
	require.Equal(t, int64(0), got)
}
