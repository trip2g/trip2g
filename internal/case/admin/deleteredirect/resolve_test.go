package deleteredirect

import (
	"context"
	"errors"
	"testing"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.DeleteRedirectInput
	}

	tests := []struct {
		name          string
		env           Env
		args          args
		want          model.DeleteRedirectOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful redirect deletion",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				DeleteRedirectFunc: func(ctx context.Context, id int64) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.DeleteRedirectInput{
					ID: 1,
				},
			},
			want: &model.DeleteRedirectPayload{
				ID: 1,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.DeleteRedirectCalls()))

				id := mockEnv.DeleteRedirectCalls()[0].ID
				require.Equal(t, int64(1), id)
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
				input: model.DeleteRedirectInput{
					ID: 1,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.DeleteRedirectCalls()))
			},
		},
		{
			name: "redirect not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 456}, nil
				},
				DeleteRedirectFunc: func(ctx context.Context, id int64) error {
					return errors.New("no rows affected")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.DeleteRedirectInput{
					ID: 999,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.DeleteRedirectCalls()))

				id := mockEnv.DeleteRedirectCalls()[0].ID
				require.Equal(t, int64(999), id)
			},
		},
		{
			name: "database error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 789}, nil
				},
				DeleteRedirectFunc: func(ctx context.Context, id int64) error {
					return errors.New("database connection failed")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.DeleteRedirectInput{
					ID: 2,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.DeleteRedirectCalls()))

				id := mockEnv.DeleteRedirectCalls()[0].ID
				require.Equal(t, int64(2), id)
			},
		},
		{
			name: "delete redirect with large ID",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 111}, nil
				},
				DeleteRedirectFunc: func(ctx context.Context, id int64) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.DeleteRedirectInput{
					ID: 9223372036854775807, // max int64
				},
			},
			want: &model.DeleteRedirectPayload{
				ID: 9223372036854775807,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.DeleteRedirectCalls()))

				id := mockEnv.DeleteRedirectCalls()[0].ID
				require.Equal(t, int64(9223372036854775807), id)
			},
		},
		{
			name: "insufficient permissions",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("insufficient permissions")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.DeleteRedirectInput{
					ID: 5,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.DeleteRedirectCalls()))
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