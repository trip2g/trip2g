package enableapikey_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/enableapikey"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg enableapikey_test . Env

type Env interface {
	EnableApiKey(ctx context.Context, id int64) (db.ApiKey, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.EnableAPIKeyInput
	}
	tests := []struct {
		name        string
		env         enableapikey.Env
		args        args
		want        model.EnableAPIKeyOrErrorPayload
		wantErr     bool
		wantErrText string
	}{
		{
			name: "successful enable",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123, Role: "admin"}, nil
				},
				EnableApiKeyFunc: func(ctx context.Context, id int64) (db.ApiKey, error) {
					return db.ApiKey{
						ID:        1,
						Value:     "api-key-12345",
						CreatedBy: 456,
					}, nil
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.EnableAPIKeyInput{ID: 1},
			},
			want: &model.EnableAPIKeyPayload{
				APIKey: &db.ApiKey{
					ID:        1,
					Value:     "api-key-12345",
					CreatedBy: 456,
				},
			},
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
				input: model.EnableAPIKeyInput{ID: 1},
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
				EnableApiKeyFunc: func(ctx context.Context, id int64) (db.ApiKey, error) {
					return db.ApiKey{}, errors.New("database error")
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.EnableAPIKeyInput{ID: 1},
			},
			wantErr:     true,
			wantErrText: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := enableapikey.Resolve(tt.args.ctx, tt.env, tt.args.input)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrText)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
