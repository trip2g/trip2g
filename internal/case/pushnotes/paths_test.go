package pushnotes_test

import (
	"context"
	"testing"

	"trip2g/internal/case/pushnotes"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestResolve_PathsStayInsideVault(t *testing.T) {
	rejected := []string{
		"../x.md",
		"a/../../x.md",
		"/../x.md",
		"..",
		"a/../..",
		"./../x.md",
	}

	for _, path := range rejected {
		t.Run("rejects "+path, func(t *testing.T) {
			env := &EnvMock{
				LoggerFunc:             func() logger.Logger { return &logger.TestLogger{} },
				CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) { return "", nil },
			}

			input := model.PushNotesInput{Updates: []model.PushNoteInput{{Path: path, Content: "x"}}}

			result, err := pushnotes.Resolve(context.Background(), env, input)
			require.NoError(t, err)
			require.IsType(t, &model.ErrorPayload{}, result)
			require.Empty(t, env.InsertNoteCalls())
		})
	}
}

func TestResolve_PathsAreNormalized(t *testing.T) {
	cases := map[string]string{
		"/abs.md":       "abs.md",
		"./rel.md":      "rel.md",
		"a/./b/../c.md": "a/c.md",
	}

	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			var stored string

			env := &EnvMock{
				LoggerFunc:             func() logger.Logger { return &logger.TestLogger{} },
				CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) { return "", nil },
				InsertNoteFunc: func(_ context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
					stored = note.Path
					return appmodel.NoteSaveResult{PathID: 1, VersionID: 1}, nil
				},
				PrepareLatestNotesFunc: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
					nvs := appmodel.NewNoteViews()
					nvs.ExtractNoteList()
					return nvs, nil
				},
				HandleLatestNotesAfterSaveFunc: func(_ context.Context, _ []int64) error { return nil },
				LayoutsFunc:                    func() *appmodel.Layouts { return &appmodel.Layouts{Map: map[string]appmodel.Layout{}} },
				PublicURLFunc:                  func() string { return "" },
				LatestNoteViewsFunc:            appmodel.NewNoteViews,
			}

			input := model.PushNotesInput{Updates: []model.PushNoteInput{{Path: raw, Content: "x"}}}

			result, err := pushnotes.Resolve(context.Background(), env, input)
			require.NoError(t, err)
			require.IsType(t, &model.PushNotesPayload{}, result)
			require.Equal(t, want, stored)
		})
	}
}
