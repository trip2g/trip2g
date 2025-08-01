package restorepatreoncredentials

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

func TestResolve_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	input := model.RestorePatreonCredentialsInput{
		ID: 123,
	}

	expectedToken := &usertoken.Data{
		ID: 1,
	}

	expectedCredentials := db.PatreonCredential{
		ID:                 123,
		CreatorAccessToken: "test_token",
		CreatedBy:          1,
	}

	env := &EnvMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return expectedToken, nil
		},
		RestorePatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			require.Equal(t, int64(123), id)
			return expectedCredentials, nil
		},
		StartPatreonRefreshBackgroundJobFunc: func(ctx context.Context, credentialsID int64, immediately bool) error {
			require.Equal(t, int64(123), credentialsID)
			return nil
		},
	}

	// Execute
	result, err := Resolve(ctx, env, input)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	payload, ok := result.(*model.RestorePatreonCredentialsPayload)
	require.True(t, ok)
	require.Equal(t, &expectedCredentials, payload.PatreonCredentials)

	// Verify mock calls
	require.Len(t, env.CurrentAdminUserTokenCalls(), 1)
	require.Len(t, env.RestorePatreonCredentialsCalls(), 1)
	require.Len(t, env.StartPatreonRefreshBackgroundJobCalls(), 1)
}

func TestResolve_AuthError(t *testing.T) {
	// Setup
	ctx := context.Background()
	input := model.RestorePatreonCredentialsInput{
		ID: 123,
	}

	env := &EnvMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return nil, errors.New("unauthorized")
		},
	}

	// Execute
	result, err := Resolve(ctx, env, input)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed to get current admin user token")

	// Verify mock calls
	require.Len(t, env.CurrentAdminUserTokenCalls(), 1)
	require.Empty(t, env.RestorePatreonCredentialsCalls())
}

func TestResolve_StartBackgroundJobError(t *testing.T) {
	// Setup
	ctx := context.Background()
	input := model.RestorePatreonCredentialsInput{
		ID: 123,
	}

	expectedToken := &usertoken.Data{
		ID: 1,
	}

	expectedCredentials := db.PatreonCredential{
		ID:                 123,
		CreatorAccessToken: "test_token",
		CreatedBy:          1,
	}

	env := &EnvMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return expectedToken, nil
		},
		RestorePatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			require.Equal(t, int64(123), id)
			return expectedCredentials, nil
		},
		StartPatreonRefreshBackgroundJobFunc: func(ctx context.Context, credentialsID int64, immediately bool) error {
			require.Equal(t, int64(123), credentialsID)
			return errors.New("failed to start background job")
		},
	}

	// Execute
	result, err := Resolve(ctx, env, input)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed to start Patreon refresh background jobs")

	// Verify mock calls
	require.Len(t, env.CurrentAdminUserTokenCalls(), 1)
	require.Len(t, env.RestorePatreonCredentialsCalls(), 1)
	require.Len(t, env.StartPatreonRefreshBackgroundJobCalls(), 1)
}
