package setapikeymcpadmintools_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/setapikeymcpadmintools"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

func TestResolve_AdminCanEnable(t *testing.T) {
	enabled := true
	env := &EnvMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{Role: "admin"}, nil
		},
		SetApiKeyMcpAdminToolsFunc: func(ctx context.Context, arg db.SetApiKeyMcpAdminToolsParams) error {
			require.Equal(t, int64(42), arg.ID)
			require.Equal(t, &enabled, arg.Enabled)
			return nil
		},
		ApiKeyByIDFunc: func(ctx context.Context, id int64) (db.ApiKey, error) {
			return db.ApiKey{ID: id}, nil
		},
	}

	result, err := setapikeymcpadmintools.Resolve(context.Background(), env, model.SetAPIKeyMcpAdminToolsInput{ID: 42, Enabled: true})
	require.NoError(t, err)
	payload, ok := result.(*model.SetAPIKeyMcpAdminToolsPayload)
	require.True(t, ok)
	require.Equal(t, int64(42), payload.APIKey.ID)
}

func TestResolve_NonAdminRejected(t *testing.T) {
	env := &EnvMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return nil, errors.New("not admin")
		},
	}

	_, err := setapikeymcpadmintools.Resolve(context.Background(), env, model.SetAPIKeyMcpAdminToolsInput{ID: 1, Enabled: true})
	require.Error(t, err)
}
