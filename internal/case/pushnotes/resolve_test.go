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
	InsertNote(ctx context.Context, update appmodel.RawNote) error
	InsertSubgraph(ctx context.Context, name string) error
	PrepareLatestNotes(ctx context.Context) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(changedPathIDs []int64) error
	Layouts() *appmodel.Layouts
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
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result model.PushNotesOrErrorPayload) {
				errPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Contains(t, errPayload.Message, ".md and .html")
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
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					InsertNoteFunc: func(ctx context.Context, note appmodel.RawNote) error {
						return nil
					},
					PrepareLatestNotesFunc: func(ctx context.Context) (*appmodel.NoteViews, error) {
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
					},
					HandleLatestNotesAfterSaveFunc: func(changedPathIDs []int64) error {
						return nil
					},
					LayoutsFunc: func() *appmodel.Layouts {
						return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
					},
				}
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
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					InsertNoteFunc: func(ctx context.Context, note appmodel.RawNote) error {
						return errors.New("database error")
					},
				}
			},
			wantErr: true,
		},
		{
			name: "prepare latest notes fails",
			input: model.PushNotesInput{
				Updates: []model.PushNoteInput{
					{Path: "test.md", Content: "# Hello"},
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return mockLogger
					},
					InsertNoteFunc: func(ctx context.Context, note appmodel.RawNote) error {
						return nil
					},
					PrepareLatestNotesFunc: func(ctx context.Context) (*appmodel.NoteViews, error) {
						return nil, errors.New("prepare error")
					},
				}
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
