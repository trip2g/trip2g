package updateredirect

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
		input model.UpdateRedirectInput
	}

	tests := []struct {
		name          string
		env           Env
		args          args
		want          model.UpdateRedirectOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful redirect update",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				UpdateRedirectFunc: func(ctx context.Context, params db.UpdateRedirectParams) (db.Redirect, error) {
					return db.Redirect{
						ID:         1,
						CreatedAt:  time.Now(),
						CreatedBy:  123,
						Pattern:    "/updated-page",
						IgnoreCase: false,
						IsRegex:    true,
						Target:     "/updated-target",
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateRedirectInput{
					ID:         1,
					Pattern:    "/updated-page",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/updated-target",
				},
			},
			want: &model.UpdateRedirectPayload{
				Redirect: &db.Redirect{
					ID:         1,
					CreatedBy:  123,
					Pattern:    "/updated-page",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/updated-target",
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateRedirectCalls()))

				params := mockEnv.UpdateRedirectCalls()[0].Params
				require.Equal(t, int64(1), params.ID)
				require.Equal(t, "/updated-page", params.Pattern)
				require.False(t, params.IgnoreCase)
				require.True(t, params.IsRegex)
				require.Equal(t, "/updated-target", params.Target)
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
				input: model.UpdateRedirectInput{
					ID:         1,
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
				require.Equal(t, 0, len(mockEnv.UpdateRedirectCalls()))
			},
		},
		{
			name: "redirect not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 456}, nil
				},
				UpdateRedirectFunc: func(ctx context.Context, params db.UpdateRedirectParams) (db.Redirect, error) {
					return db.Redirect{}, errors.New("no rows in result set")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateRedirectInput{
					ID:         999,
					Pattern:    "/nonexistent",
					IgnoreCase: true,
					IsRegex:    false,
					Target:     "/nowhere",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateRedirectCalls()))

				params := mockEnv.UpdateRedirectCalls()[0].Params
				require.Equal(t, int64(999), params.ID)
				require.Equal(t, "/nonexistent", params.Pattern)
			},
		},
		{
			name: "update to regex pattern",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 789}, nil
				},
				UpdateRedirectFunc: func(ctx context.Context, params db.UpdateRedirectParams) (db.Redirect, error) {
					return db.Redirect{
						ID:         2,
						CreatedAt:  time.Now(),
						CreatedBy:  123,
						Pattern:    "^/api/v([0-9]+)/.*$",
						IgnoreCase: false,
						IsRegex:    true,
						Target:     "/api/v$1/",
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateRedirectInput{
					ID:         2,
					Pattern:    "^/api/v([0-9]+)/.*$",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/api/v$1/",
				},
			},
			want: &model.UpdateRedirectPayload{
				Redirect: &db.Redirect{
					ID:         2,
					CreatedBy:  123,
					Pattern:    "^/api/v([0-9]+)/.*$",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/api/v$1/",
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateRedirectCalls()))

				params := mockEnv.UpdateRedirectCalls()[0].Params
				require.Equal(t, int64(2), params.ID)
				require.Equal(t, "^/api/v([0-9]+)/.*$", params.Pattern)
				require.False(t, params.IgnoreCase)
				require.True(t, params.IsRegex)
				require.Equal(t, "/api/v$1/", params.Target)
			},
		},
		{
			name: "change case sensitivity",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 111}, nil
				},
				UpdateRedirectFunc: func(ctx context.Context, params db.UpdateRedirectParams) (db.Redirect, error) {
					return db.Redirect{
						ID:         3,
						CreatedAt:  time.Now(),
						CreatedBy:  111,
						Pattern:    "/Case-Insensitive",
						IgnoreCase: true,
						IsRegex:    false,
						Target:     "/case-insensitive",
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateRedirectInput{
					ID:         3,
					Pattern:    "/Case-Insensitive",
					IgnoreCase: true,
					IsRegex:    false,
					Target:     "/case-insensitive",
				},
			},
			want: &model.UpdateRedirectPayload{
				Redirect: &db.Redirect{
					ID:         3,
					CreatedBy:  111,
					Pattern:    "/Case-Insensitive",
					IgnoreCase: true,
					IsRegex:    false,
					Target:     "/case-insensitive",
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateRedirectCalls()))

				params := mockEnv.UpdateRedirectCalls()[0].Params
				require.Equal(t, int64(3), params.ID)
				require.Equal(t, "/Case-Insensitive", params.Pattern)
				require.True(t, params.IgnoreCase)
				require.False(t, params.IsRegex)
				require.Equal(t, "/case-insensitive", params.Target)
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
					gotPayload := got.(*model.UpdateRedirectPayload)
					wantPayload := tt.want.(*model.UpdateRedirectPayload)
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
