package processpatreonwebhook_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"trip2g/internal/case/processpatreonwebhook"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/patreon"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg processpatreonwebhook_test . Env

type Env interface {
	Logger() logger.Logger
	PatreonCredentials(ctx context.Context, id int64) (db.PatreonCredential, error)
	AllActivePatreonCredentials(ctx context.Context) ([]db.PatreonCredential, error)
	PatreonListCampaigns(token string) ([]patreon.Campaign, error)
	PatreonListPatrons(token string, campaignID string) (*patreon.PatronsResponse, error)
	UpdatePatreonCredentialsSyncedAt(ctx context.Context, id int64) error
	SetPatreonMemberCurrentTier(ctx context.Context, arg db.SetPatreonMemberCurrentTierParams) error
	GetPatreonCampaignsByCredentialsID(ctx context.Context, credentialsID int64) ([]db.PatreonCampaign, error)
	GetPatreonTierByTierID(ctx context.Context, arg db.GetPatreonTierByTierIDParams) (db.PatreonTier, error)
	GetPatreonMemberByPatreonIDAndCampaignID(ctx context.Context, arg db.GetPatreonMemberByPatreonIDAndCampaignIDParams) (db.PatreonMember, error)
	UpsertPatreonCampaign(ctx context.Context, arg db.UpsertPatreonCampaignParams) error
	UpsertPatreonTier(ctx context.Context, arg db.UpsertPatreonTierParams) error
	UpsertPatreonMember(ctx context.Context, arg db.UpsertPatreonMemberParams) error
}

func TestResolve_Success(t *testing.T) {
	// Setup
	reqCtx := &fasthttp.RequestCtx{}
	credentialID := int64(123)

	expectedCredentials := db.PatreonCredential{
		ID:                 123,
		CreatorAccessToken: "test_token",
		CreatedBy:          1,
	}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			require.Equal(t, credentialID, id)
			return expectedCredentials, nil
		},
		PatreonListCampaignsFunc: func(token string) ([]patreon.Campaign, error) {
			require.Equal(t, "test_token", token)
			return []patreon.Campaign{}, nil
		},
		GetPatreonCampaignsByCredentialsIDFunc: func(ctx context.Context, credentialsID int64) ([]db.PatreonCampaign, error) {
			require.Equal(t, credentialID, credentialsID)
			return []db.PatreonCampaign{}, nil
		},
		UpdatePatreonCredentialsSyncedAtFunc: func(ctx context.Context, id int64) error {
			require.Equal(t, credentialID, id)
			return nil
		},
	}

	// Execute
	result, err := processpatreonwebhook.Resolve(reqCtx, env, credentialID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)

	// Verify that refresh was called
	require.Len(t, env.PatreonCredentialsCalls(), 2) // Once in webhook, once in refresh
	require.Len(t, env.UpdatePatreonCredentialsSyncedAtCalls(), 1)
}

func TestResolve_CredentialsNotFound(t *testing.T) {
	// Setup
	reqCtx := &fasthttp.RequestCtx{}
	credentialID := int64(999)

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			return db.PatreonCredential{}, sql.ErrNoRows
		},
	}

	// Execute
	result, err := processpatreonwebhook.Resolve(reqCtx, env, credentialID)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "credentials not found")

	// Verify mock calls
	require.Len(t, env.PatreonCredentialsCalls(), 1)
}

func TestResolve_RefreshDataError(t *testing.T) {
	// Setup
	reqCtx := &fasthttp.RequestCtx{}
	credentialID := int64(123)

	expectedCredentials := db.PatreonCredential{
		ID:                 123,
		CreatorAccessToken: "test_token",
		CreatedBy:          1,
	}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			return expectedCredentials, nil
		},
		PatreonListCampaignsFunc: func(token string) ([]patreon.Campaign, error) {
			return nil, errors.New("API error")
		},
		GetPatreonCampaignsByCredentialsIDFunc: func(ctx context.Context, credentialsID int64) ([]db.PatreonCampaign, error) {
			return []db.PatreonCampaign{}, nil
		},
	}

	// Execute
	result, err := processpatreonwebhook.Resolve(reqCtx, env, credentialID)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed to refresh patreon data")
}

func TestResolve_DatabaseError(t *testing.T) {
	// Setup
	reqCtx := &fasthttp.RequestCtx{}
	credentialID := int64(123)

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			return db.PatreonCredential{}, errors.New("database connection error")
		},
	}

	// Execute
	result, err := processpatreonwebhook.Resolve(reqCtx, env, credentialID)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed to get patreon credentials")
}