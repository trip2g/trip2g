package createapikey_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/case/admin/createapikey"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createapikey_test . Env

type Env interface {
	GenerateApiKey() string
	InsertApiKey(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.CreateAPIKeyInput
	}

	tests := []struct {
		name          string
		env           createapikey.Env
		args          args
		want          model.CreateAPIKeyOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful API key creation",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				GenerateApiKeyFunc: func() string {
					return "api-key-12345"
				},
				InsertApiKeyFunc: func(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error) {
					return db.ApiKey{
						ID:          1,
						Value:       params.Value,
						CreatedBy:   params.CreatedBy,
						Description: params.Description,
						CreatedAt:   time.Now(),
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateAPIKeyInput{
					Description: "Test API Key",
				},
			},
			want: &model.CreateAPIKeyPayload{
				Value: "api-key-12345",
				APIKey: &db.ApiKey{
					ID:          1,
					Value:       "api-key-12345",
					CreatedBy:   123,
					Description: "Test API Key",
					CreatedAt:   time.Time{}, // will be set by test
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.GenerateApiKeyCalls()))
				require.Equal(t, 1, len(mockEnv.InsertApiKeyCalls()))

				// Verify API key parameters
				params := mockEnv.InsertApiKeyCalls()[0].Params
				require.Equal(t, "api-key-12345", params.Value)
				require.Equal(t, int64(123), params.CreatedBy)
				require.Equal(t, "Test API Key", params.Description)
			},
		},
		{
			name: "successful API key creation with empty description",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 456}, nil
				},
				GenerateApiKeyFunc: func() string {
					return "api-key-67890"
				},
				InsertApiKeyFunc: func(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error) {
					return db.ApiKey{
						ID:          2,
						Value:       params.Value,
						CreatedBy:   params.CreatedBy,
						Description: params.Description,
						CreatedAt:   time.Now(),
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateAPIKeyInput{
					Description: "",
				},
			},
			want: &model.CreateAPIKeyPayload{
				Value: "api-key-67890",
				APIKey: &db.ApiKey{
					ID:          2,
					Value:       "api-key-67890",
					CreatedBy:   456,
					Description: "",
					CreatedAt:   time.Time{},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.GenerateApiKeyCalls()))
				require.Equal(t, 1, len(mockEnv.InsertApiKeyCalls()))

				params := mockEnv.InsertApiKeyCalls()[0].Params
				require.Equal(t, "", params.Description)
			},
		},
		{
			name: "error - admin token error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateAPIKeyInput{
					Description: "Test API Key",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.GenerateApiKeyCalls()))
				require.Equal(t, 0, len(mockEnv.InsertApiKeyCalls()))
			},
		},
		{
			name: "error - database insertion fails",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 789}, nil
				},
				GenerateApiKeyFunc: func() string {
					return "api-key-fail"
				},
				InsertApiKeyFunc: func(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error) {
					return db.ApiKey{}, errors.New("database constraint violation")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateAPIKeyInput{
					Description: "Test API Key",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.GenerateApiKeyCalls()))
				require.Equal(t, 1, len(mockEnv.InsertApiKeyCalls()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createapikey.Resolve(tt.args.ctx, tt.env, tt.args.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Skip time comparison for CreatedAt field
				if payload, ok := got.(*model.CreateAPIKeyPayload); ok {
					if want, ok := tt.want.(*model.CreateAPIKeyPayload); ok {
						require.Equal(t, want.Value, payload.Value)
						require.Equal(t, want.APIKey.ID, payload.APIKey.ID)
						require.Equal(t, want.APIKey.Value, payload.APIKey.Value)
						require.Equal(t, want.APIKey.CreatedBy, payload.APIKey.CreatedBy)
						require.Equal(t, want.APIKey.Description, payload.APIKey.Description)
						return
					}
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Resolve() = %v, want %v", got, tt.want)
					for _, desc := range pretty.Diff(got, tt.want) {
						t.Error(desc)
					}
				}
			}

			if tt.afterCallback != nil {
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}