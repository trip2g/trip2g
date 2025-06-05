package createredirect

import (
	"context"
	"errors"
	"testing"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.CreateRedirectInput
	}

	tests := []struct {
		name          string
		env           Env
		args          args
		want          model.CreateRedirectOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful redirect creation",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				InsertRedirectFunc: func(ctx context.Context, params db.InsertRedirectParams) (db.Redirect, error) {
					return db.Redirect{
						ID:         1,
						CreatedAt:  time.Now(),
						CreatedBy:  123,
						Pattern:    "/old-page",
						IgnoreCase: true,
						IsRegex:    false,
						Target:     "/new-page",
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateRedirectInput{
					Pattern:    "/old-page",
					IgnoreCase: true,
					IsRegex:    false,
					Target:     "/new-page",
				},
			},
			want: &model.CreateRedirectPayload{
				Redirect: &db.Redirect{
					ID:         1,
					CreatedBy:  123,
					Pattern:    "/old-page",
					IgnoreCase: true,
					IsRegex:    false,
					Target:     "/new-page",
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.InsertRedirectCalls()))

				params := mockEnv.InsertRedirectCalls()[0].Params
				require.Equal(t, int64(123), params.CreatedBy)
				require.Equal(t, "/old-page", params.Pattern)
				require.True(t, params.IgnoreCase)
				require.False(t, params.IsRegex)
				require.Equal(t, "/new-page", params.Target)
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
				input: model.CreateRedirectInput{
					Pattern:    "/test",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/target",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.InsertRedirectCalls()))
			},
		},
		{
			name: "database error on insert",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 456}, nil
				},
				InsertRedirectFunc: func(ctx context.Context, params db.InsertRedirectParams) (db.Redirect, error) {
					return db.Redirect{}, errors.New("database connection failed")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateRedirectInput{
					Pattern:    "/error-test",
					IgnoreCase: true,
					IsRegex:    false,
					Target:     "/error-target",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.InsertRedirectCalls()))

				params := mockEnv.InsertRedirectCalls()[0].Params
				require.Equal(t, int64(456), params.CreatedBy)
				require.Equal(t, "/error-test", params.Pattern)
			},
		},
		{
			name: "create regex redirect",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 789}, nil
				},
				InsertRedirectFunc: func(ctx context.Context, params db.InsertRedirectParams) (db.Redirect, error) {
					return db.Redirect{
						ID:         2,
						CreatedAt:  time.Now(),
						CreatedBy:  789,
						Pattern:    "^/blog/([0-9]+)$",
						IgnoreCase: false,
						IsRegex:    true,
						Target:     "/posts/$1",
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateRedirectInput{
					Pattern:    "^/blog/([0-9]+)$",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/posts/$1",
				},
			},
			want: &model.CreateRedirectPayload{
				Redirect: &db.Redirect{
					ID:         2,
					CreatedBy:  789,
					Pattern:    "^/blog/([0-9]+)$",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/posts/$1",
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.InsertRedirectCalls()))

				params := mockEnv.InsertRedirectCalls()[0].Params
				require.Equal(t, int64(789), params.CreatedBy)
				require.Equal(t, "^/blog/([0-9]+)$", params.Pattern)
				require.False(t, params.IgnoreCase)
				require.True(t, params.IsRegex)
				require.Equal(t, "/posts/$1", params.Target)
			},
		},
		{
			name: "create case sensitive redirect",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 999}, nil
				},
				InsertRedirectFunc: func(ctx context.Context, params db.InsertRedirectParams) (db.Redirect, error) {
					return db.Redirect{
						ID:         3,
						CreatedAt:  time.Now(),
						CreatedBy:  999,
						Pattern:    "/CaseSensitive",
						IgnoreCase: false,
						IsRegex:    false,
						Target:     "/case-sensitive",
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateRedirectInput{
					Pattern:    "/CaseSensitive",
					IgnoreCase: false,
					IsRegex:    false,
					Target:     "/case-sensitive",
				},
			},
			want: &model.CreateRedirectPayload{
				Redirect: &db.Redirect{
					ID:         3,
					CreatedBy:  999,
					Pattern:    "/CaseSensitive",
					IgnoreCase: false,
					IsRegex:    false,
					Target:     "/case-sensitive",
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.InsertRedirectCalls()))

				params := mockEnv.InsertRedirectCalls()[0].Params
				require.Equal(t, int64(999), params.CreatedBy)
				require.Equal(t, "/CaseSensitive", params.Pattern)
				require.False(t, params.IgnoreCase)
				require.False(t, params.IsRegex)
				require.Equal(t, "/case-sensitive", params.Target)
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
				if tt.want != nil {
					gotPayload := got.(*model.CreateRedirectPayload)
					wantPayload := tt.want.(*model.CreateRedirectPayload)
					// Compare without time fields for simplicity
					require.Equal(t, wantPayload.Redirect.ID, gotPayload.Redirect.ID)
					require.Equal(t, wantPayload.Redirect.CreatedBy, gotPayload.Redirect.CreatedBy)
					require.Equal(t, wantPayload.Redirect.Pattern, gotPayload.Redirect.Pattern)
					require.Equal(t, wantPayload.Redirect.IgnoreCase, gotPayload.Redirect.IgnoreCase)
					require.Equal(t, wantPayload.Redirect.IsRegex, gotPayload.Redirect.IsRegex)
					require.Equal(t, wantPayload.Redirect.Target, gotPayload.Redirect.Target)
				}
			}

			if tt.afterCallback != nil {
				mockEnv, ok := tt.env.(*envMock)
				require.True(t, ok, "env should be a mock for callback tests")
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}
