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
}

// newEnvMock returns an EnvMock with safe defaults for all methods.
// Individual tests override only the methods they care about.
func newEnvMock(log logger.Logger) *EnvMock {
	return &EnvMock{
		LoggerFunc:             func() logger.Logger { return log },
		CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) { return "", nil },
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
				require.Contains(t, errPayload.Message, ".md, .html, and .html.json")
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
