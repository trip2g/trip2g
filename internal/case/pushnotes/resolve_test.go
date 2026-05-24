package pushnotes_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/pushnotes"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg pushnotes_test . Env

type Env interface {
	Logger() logger.Logger
	InsertNote(ctx context.Context, update appmodel.RawNote) (int64, error)
	InsertUncommittedPath(ctx context.Context, notePathID int64) error
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error
	Layouts() *appmodel.Layouts
	LatestNoteViews() *appmodel.NoteViews
	CheckStorageLimits(ctx context.Context, additionalAssetBytes int64) (string, error)
	PublicURL() string
}

// newEnvMock returns an EnvMock with safe defaults for all methods.
// Individual tests override only the methods they care about.
func newEnvMock(log logger.Logger) *EnvMock {
	return &EnvMock{
		LoggerFunc:             func() logger.Logger { return log },
		CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) { return "", nil },
		PublicURLFunc:          func() string { return "" },
	}
}

func TestResolve(t *testing.T) {
	ctx := context.Background()
	mockLogger := &logger.TestLogger{}

	tests := []struct {
		name     string
		input    model.PushNotesInput
		setupEnv func() *EnvMock
		wantErr  bool
		validate func(t *testing.T, result model.PushNotesOrErrorPayload)
	}{
		{
			name: "unsupported file extension",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.txt", Content: "content"},
				},
			},
			setupEnv: func() *EnvMock {
				return newEnvMock(mockLogger)
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				errPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Contains(t, errPayload.Message, ".canvas")
				require.Contains(t, errPayload.Message, ".base")
			},
		},
		{
			name: "canvas file accepted",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "demo.canvas", Content: `{"nodes":[],"edges":[]}`},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(_ context.Context, note appmodel.RawNote) (int64, error) {
					require.Equal(t, "demo.canvas", note.Path)
					return 1, nil
				}
				env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
					nvs := appmodel.NewNoteViews()
					nvs.ExtractNoteList()
					return nvs, nil
				}
				env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error { return nil }
				env.LayoutsFunc = func() *appmodel.Layouts {
					return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				_, ok := result.(*model.PushNotesPayload)
				require.True(t, ok)
			},
		},
		{
			name: "excalidraw file accepted",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "example.excalidraw", Content: `{"type":"excalidraw","version":2,"source":"https://excalidraw.com","elements":[],"appState":{},"files":{}}`},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(_ context.Context, note appmodel.RawNote) (int64, error) {
					require.Equal(t, "example.excalidraw", note.Path)
					return 3, nil
				}
				env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
					nvs := appmodel.NewNoteViews()
					nvs.ExtractNoteList()
					return nvs, nil
				}
				env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error { return nil }
				env.LayoutsFunc = func() *appmodel.Layouts {
					return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				_, ok := result.(*model.PushNotesPayload)
				require.True(t, ok)
			},
		},
		{
			name: "base file accepted",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "mybase.base", Content: "type: base\nname: My Base\n"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(_ context.Context, note appmodel.RawNote) (int64, error) {
					require.Equal(t, "mybase.base", note.Path)
					return 2, nil
				}
				env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
					nvs := appmodel.NewNoteViews()
					nvs.ExtractNoteList()
					return nvs, nil
				}
				env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error { return nil }
				env.LayoutsFunc = func() *appmodel.Layouts {
					return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				_, ok := result.(*model.PushNotesPayload)
				require.True(t, ok)
			},
		},
		{
			name: "successful push with md file",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.md", Content: "# Hello World"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(ctx context.Context, note appmodel.RawNote) (int64, error) {
					return 1, nil
				}
				env.PrepareLatestNotesFunc = func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) {
					return &appmodel.NoteViews{
						List: []*appmodel.NoteView{
							{
								Path:      "test.md",
								PathID:    1,
								VersionID: 100,
								Assets:    map[string]struct{}{},
							},
						},
						Subgraphs: map[string]*appmodel.NoteSubgraph{},
					}, nil
				}
				env.HandleLatestNotesAfterSaveFunc = func(ctx context.Context, changedPathIDs []int64) error {
					return nil
				}
				env.LayoutsFunc = func() *appmodel.Layouts {
					return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				payload, ok := result.(*model.PushNotesPayload)
				require.True(t, ok)
				require.Len(t, payload.Notes, 1)
				require.Equal(t, "test.md", payload.Notes[0].Path)
			},
		},
		{
			name: "insert note fails",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.md", Content: "# Hello"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(ctx context.Context, note appmodel.RawNote) (int64, error) {
					return 0, errors.New("database error")
				}
				return env
			},
			wantErr: true,
		},
		{
			name: "storage limit exceeded",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.md", Content: "# Hello"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.CheckStorageLimitsFunc = func(_ context.Context, _ int64) (string, error) {
					return "database storage limit exceeded", nil
				}
				return env
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				errPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Equal(t, "database storage limit exceeded", errPayload.Message)
			},
		},
		{
			name: "prepare latest notes fails",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.md", Content: "# Hello"},
				},
			},
			setupEnv: func() *EnvMock {
				env := newEnvMock(mockLogger)
				env.InsertNoteFunc = func(ctx context.Context, note appmodel.RawNote) (int64, error) {
					return 1, nil
				}
				env.PrepareLatestNotesFunc = func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) {
					return nil, errors.New("prepare error")
				}
				return env
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()

			result, err := pushnotes.Resolve(ctx, env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestResolve_UpdatedNotes(t *testing.T) {
	ctx := context.Background()
	mockLogger := &logger.TestLogger{}

	makeNVS := func() *appmodel.NoteViews {
		nvs := appmodel.NewNoteViews()
		note := &appmodel.NoteView{
			PathID:    42,
			VersionID: 1,
			Path:      "my-note.md",
			Permalink: "/my-note",
		}
		note.AddWarning(appmodel.NoteWarningWarning, "broken link to [[missing]]")
		nvs.RegisterNote(note)
		nvs.ExtractNoteList()
		return nvs
	}

	t.Run("updated contains pushed notes with url and warnings", func(t *testing.T) {
		env := newEnvMock(mockLogger)
		env.InsertNoteFunc = func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			return 42, nil
		}
		env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			return makeNVS(), nil
		}
		env.HandleLatestNotesAfterSaveFunc = func(_ context.Context, _ []int64) error {
			return nil
		}
		env.LayoutsFunc = func() *appmodel.Layouts {
			return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
		}
		env.PublicURLFunc = func() string { return "https://example.com" }

		input := model.PushNotesInput{
			Updates: []model.PushNoteInput{
				{Path: "my-note.md", Content: "# Hello"},
			},
		}

		result, err := pushnotes.Resolve(ctx, env, input)
		require.NoError(t, err)

		payload, ok := result.(*model.PushNotesPayload)
		require.True(t, ok)

		require.Len(t, payload.Updated, 1)
		require.Equal(t, "my-note.md", payload.Updated[0].Path)
		require.NotNil(t, payload.Updated[0].URL)
		require.Equal(t, "https://example.com/my-note", *payload.Updated[0].URL)
		require.Len(t, payload.Updated[0].Warnings, 1)
		require.Equal(t, "broken link to [[missing]]", payload.Updated[0].Warnings[0].Message)

		require.Len(t, payload.Notes, 1)
		require.NotNil(t, payload.Notes[0].URL)
		require.Equal(t, "https://example.com/my-note", *payload.Notes[0].URL)
	})

	t.Run("updated is empty when no updates provided", func(t *testing.T) {
		env := newEnvMock(mockLogger)
		env.LatestNoteViewsFunc = func() *appmodel.NoteViews {
			return makeNVS()
		}
		env.LayoutsFunc = func() *appmodel.Layouts {
			return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
		}
		env.PublicURLFunc = func() string { return "https://example.com" }

		input := model.PushNotesInput{Updates: []model.PushNoteInput{}}

		result, err := pushnotes.Resolve(ctx, env, input)
		require.NoError(t, err)

		payload, ok := result.(*model.PushNotesPayload)
		require.True(t, ok)
		require.Empty(t, payload.Updated)
	})
}
