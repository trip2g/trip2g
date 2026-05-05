package commitnotes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/commitnotes"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg commitnotes_test . Env

func TestResolve_Updated(t *testing.T) {
	ctx := context.Background()

	makeNVS := func() *appmodel.NoteViews {
		nvs := appmodel.NewNoteViews()
		note := &appmodel.NoteView{
			PathID:    7,
			VersionID: 3,
			Path:      "commit-note.md",
			Permalink: "/commit-note",
		}
		nvs.RegisterNote(note)
		nvs.ExtractNoteList()
		return nvs
	}

	env := &EnvMock{
		ListUncommittedPathsFunc: func(_ context.Context) ([]int64, error) {
			return []int64{7}, nil
		},
		PrepareLatestNotesFunc: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			return makeNVS(), nil
		},
		HandleLatestNotesAfterSaveFunc: func(_ context.Context, _ []int64) error {
			return nil
		},
		ClearUncommittedPathsFunc: func(_ context.Context) error {
			return nil
		},
		PublicURLFunc: func() string { return "https://site.com" },
	}

	result, err := commitnotes.Resolve(ctx, env)
	require.NoError(t, err)

	payload, ok := result.(*model.CommitNotesPayload)
	require.True(t, ok)
	require.True(t, payload.Success)

	require.Len(t, payload.Updated, 1)
	require.Equal(t, "commit-note.md", payload.Updated[0].Path)
	require.NotNil(t, payload.Updated[0].URL)
	require.Equal(t, "https://site.com/commit-note", *payload.Updated[0].URL)
	require.Empty(t, payload.Updated[0].Warnings)
}
