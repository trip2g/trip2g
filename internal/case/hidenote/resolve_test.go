package hidenote

import (
	"context"
	"errors"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate moq -out mocks_test.go . Env

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.HideNoteInput
	}

	tests := []struct {
		name          string
		env           Env
		args          args
		want          model.HideNoteOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful hide note",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				HideNotePathFunc: func(ctx context.Context, params db.HideNotePathParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.HideNoteInput{
					Path: "/test/note.md",
				},
			},
			want: &model.HideNotePayload{
				Success: true,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.HideNotePathCalls()))

				hideParams := mockEnv.HideNotePathCalls()[0].Params
				require.Equal(t, "/test/note.md", hideParams.Value)
				require.True(t, hideParams.HiddenBy.Valid)
				require.Equal(t, int64(123), hideParams.HiddenBy.Int64)
			},
		},
		{
			name: "admin authorization failure",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("user not authenticated")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.HideNoteInput{
					Path: "/test/note.md",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.HideNotePathCalls()))
			},
		},
		{
			name: "database error when hiding note",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 456}, nil
				},
				HideNotePathFunc: func(ctx context.Context, params db.HideNotePathParams) error {
					return errors.New("database connection failed")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.HideNoteInput{
					Path: "/another/note.md",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.HideNotePathCalls()))

				hideParams := mockEnv.HideNotePathCalls()[0].Params
				require.Equal(t, "/another/note.md", hideParams.Value)
				require.True(t, hideParams.HiddenBy.Valid)
				require.Equal(t, int64(456), hideParams.HiddenBy.Int64)
			},
		},
		{
			name: "hide note with empty path",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 789}, nil
				},
				HideNotePathFunc: func(ctx context.Context, params db.HideNotePathParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.HideNoteInput{
					Path: "",
				},
			},
			want: &model.HideNotePayload{
				Success: true,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.HideNotePathCalls()))

				hideParams := mockEnv.HideNotePathCalls()[0].Params
				require.Equal(t, "", hideParams.Value)
				require.True(t, hideParams.HiddenBy.Valid)
				require.Equal(t, int64(789), hideParams.HiddenBy.Int64)
			},
		},
		{
			name: "hide note with special characters in path",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 999}, nil
				},
				HideNotePathFunc: func(ctx context.Context, params db.HideNotePathParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.HideNoteInput{
					Path: "/folder with spaces/note-with-dashes/file_with_underscores.md",
				},
			},
			want: &model.HideNotePayload{
				Success: true,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.HideNotePathCalls()))

				hideParams := mockEnv.HideNotePathCalls()[0].Params
				require.Equal(t, "/folder with spaces/note-with-dashes/file_with_underscores.md", hideParams.Value)
				require.True(t, hideParams.HiddenBy.Valid)
				require.Equal(t, int64(999), hideParams.HiddenBy.Int64)
			},
		},
		{
			name: "database constraint violation",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 111}, nil
				},
				HideNotePathFunc: func(ctx context.Context, params db.HideNotePathParams) error {
					return errors.New("UNIQUE constraint failed: note_paths.value")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.HideNoteInput{
					Path: "/duplicate/note.md",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.HideNotePathCalls()))

				hideParams := mockEnv.HideNotePathCalls()[0].Params
				require.Equal(t, "/duplicate/note.md", hideParams.Value)
				require.True(t, hideParams.HiddenBy.Valid)
				require.Equal(t, int64(111), hideParams.HiddenBy.Int64)
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