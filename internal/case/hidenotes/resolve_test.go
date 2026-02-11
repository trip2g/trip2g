package hidenotes

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	internalmodel "trip2g/internal/model"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type envMock = EnvMock

func newTestEnv(hideFunc func(ctx context.Context, params db.HideNotePathParams) error) *envMock {
	return &envMock{
		HideNotePathFunc:    hideFunc,
		LatestNoteViewsFunc: internalmodel.NewNoteViews,
		LoggerFunc: func() logger.Logger {
			return slog.Default()
		},
	}
}

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.HideNotesInput
	}

	tests := []struct {
		name          string
		env           Env
		args          args
		want          model.HideNotesOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful hide single note",
			env: newTestEnv(func(ctx context.Context, params db.HideNotePathParams) error {
				return nil
			}),
			args: args{
				ctx: context.Background(),
				input: model.HideNotesInput{
					Paths: []string{"/test/note.md"},
					ApiKey: db.ApiKey{
						CreatedBy: 123,
					},
				},
			},
			want: &model.HideNotesPayload{
				Success: true,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.HideNotePathCalls(), 1)

				hideParams := mockEnv.HideNotePathCalls()[0].Params
				require.Equal(t, "/test/note.md", hideParams.Value)
				require.NotNil(t, hideParams.HiddenBy)
				require.Equal(t, int64(123), *hideParams.HiddenBy)
			},
		},
		{
			name: "successful hide multiple notes",
			env: newTestEnv(func(ctx context.Context, params db.HideNotePathParams) error {
				return nil
			}),
			args: args{
				ctx: context.Background(),
				input: model.HideNotesInput{
					Paths: []string{"/test/note1.md", "/test/note2.md", "/folder/note3.md"},
					ApiKey: db.ApiKey{
						CreatedBy: 456,
					},
				},
			},
			want: &model.HideNotesPayload{
				Success: true,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.HideNotePathCalls(), 3)

				expectedPaths := []string{"/test/note1.md", "/test/note2.md", "/folder/note3.md"}
				for i, call := range mockEnv.HideNotePathCalls() {
					require.Equal(t, expectedPaths[i], call.Params.Value)
					require.NotNil(t, call.Params.HiddenBy)
					require.Equal(t, int64(456), *call.Params.HiddenBy)
				}
			},
		},
		{
			name: "database error when hiding first note",
			env: newTestEnv(func(ctx context.Context, params db.HideNotePathParams) error {
				return errors.New("database connection failed")
			}),
			args: args{
				ctx: context.Background(),
				input: model.HideNotesInput{
					Paths: []string{"/another/note.md", "/second/note.md"},
					ApiKey: db.ApiKey{
						CreatedBy: 789,
					},
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.HideNotePathCalls(), 1)

				hideParams := mockEnv.HideNotePathCalls()[0].Params
				require.Equal(t, "/another/note.md", hideParams.Value)
				require.NotNil(t, hideParams.HiddenBy)
				require.Equal(t, int64(789), *hideParams.HiddenBy)
			},
		},
		{
			name: "hide notes with empty paths array",
			env: newTestEnv(func(ctx context.Context, params db.HideNotePathParams) error {
				return nil
			}),
			args: args{
				ctx: context.Background(),
				input: model.HideNotesInput{
					Paths: []string{},
					ApiKey: db.ApiKey{
						CreatedBy: 999,
					},
				},
			},
			want: &model.HideNotesPayload{
				Success: true,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.HideNotePathCalls())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.args.ctx, tt.env, tt.args.input)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got, pretty.Diff(tt.want, got))
			}

			if tt.afterCallback != nil {
				mockEnv, ok := tt.env.(*envMock)
				require.True(t, ok, "env should be a mock for callback tests")
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}
