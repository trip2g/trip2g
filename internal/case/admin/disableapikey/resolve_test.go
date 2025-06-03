package disableapikey_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"trip2g/internal/case/admin/disableapikey"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg disableapikey_test . Env

type Env interface {
	DisableApiKey(ctx context.Context, params db.DisableApiKeyParams) (db.ApiKey, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock


func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.DisableAPIKeyInput
	}
	tests := []struct {
		name          string
		env           disableapikey.Env
		args          args
		want          model.DisableAPIKeyOrErrorPayload
		wantErr       bool
		wantErrText   string
		wantCallCount int
	}{
		{
			name: "successful disable",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123, Role: "admin"}, nil
				},
				DisableApiKeyFunc: func(ctx context.Context, params db.DisableApiKeyParams) (db.ApiKey, error) {
					return db.ApiKey{
						ID:         1,
						Value:      "api-key-12345",
						CreatedBy:  456,
						DisabledBy: sql.NullInt64{Valid: true, Int64: 123},
					}, nil
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.DisableAPIKeyInput{ID: 1},
			},
			want: &model.DisableAPIKeyPayload{
				APIKey: &db.ApiKey{
					ID:         1,
					Value:      "api-key-12345",
					CreatedBy:  456,
					DisabledBy: sql.NullInt64{Valid: true, Int64: 123},
				},
			},
			wantCallCount: 1,
		},
		{
			name: "unauthorized user",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("user is not admin")
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.DisableAPIKeyInput{ID: 1},
			},
			wantErr:     true,
			wantErrText: "failed to get current user token",
		},
		{
			name: "no token in context",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("no token")
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.DisableAPIKeyInput{ID: 1},
			},
			wantErr:     true,
			wantErrText: "failed to get current user token",
		},
		{
			name: "database error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123, Role: "admin"}, nil
				},
				DisableApiKeyFunc: func(ctx context.Context, params db.DisableApiKeyParams) (db.ApiKey, error) {
					return db.ApiKey{}, errors.New("database error")
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.DisableAPIKeyInput{ID: 1},
			},
			wantErr:       true,
			wantErrText:   "database error",
			wantCallCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := disableapikey.Resolve(tt.args.ctx, tt.env, tt.args.input)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrText)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)

			if env, ok := tt.env.(*envMock); ok && tt.wantCallCount > 0 {
				require.Len(t, env.DisableApiKeyCalls(), tt.wantCallCount)
				call := env.DisableApiKeyCalls()[0]
				require.Equal(t, int64(1), call.Params.ID)
				require.True(t, call.Params.DisabledBy.Valid)
				require.Equal(t, int64(123), call.Params.DisabledBy.Int64)
			}
		})
	}
}