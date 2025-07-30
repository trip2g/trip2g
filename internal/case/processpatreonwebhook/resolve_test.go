package processpatreonwebhook_test

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"testing"

	"trip2g/internal/case/processpatreonwebhook"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/patreon"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg processpatreonwebhook_test . Env

// generateValidSignature creates a valid HMAC-MD5 signature for testing.
func generateValidSignature(body []byte, secret string) string {
	h := hmac.New(md5.New, []byte(secret))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

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
	credentialID := int64(123)
	webhookSecret := "test_secret"
	testBody := []byte(`{"test": "data"}`)

	expectedCredentials := db.PatreonCredential{
		ID:                 123,
		CreatorAccessToken: "test_token",
		CreatedBy:          1,
		WebhookSecret:      sql.NullString{String: webhookSecret, Valid: true},
	}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			require.Equal(t, credentialID, id)
			return expectedCredentials, nil
		},
		PatreonClientByIDFunc: func(ctx context.Context, credentialsID int64) (patreon.Client, error) {
			require.Equal(t, credentialID, credentialsID)
			return &patreon.ClientMock{
				ListCampaignsFunc: func() ([]patreon.Campaign, error) {
					return []patreon.Campaign{}, nil
				},
			}, nil
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
	validSignature := generateValidSignature(testBody, webhookSecret)
	request := processpatreonwebhook.Request{
		CredentialID: credentialID,
		Signature:    validSignature,
		Body:         testBody,
	}
	result, err := processpatreonwebhook.Resolve(context.Background(), env, request)

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
	credentialID := int64(999)
	testBody := []byte(`{"test": "data"}`)

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			return db.PatreonCredential{}, sql.ErrNoRows
		},
	}

	// Execute
	request := processpatreonwebhook.Request{
		CredentialID: credentialID,
		Signature:    "any_signature",
		Body:         testBody,
	}
	result, err := processpatreonwebhook.Resolve(context.Background(), env, request)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "credentials not found")

	// Verify mock calls
	require.Len(t, env.PatreonCredentialsCalls(), 1)
}

func TestResolve_RefreshDataError(t *testing.T) {
	// Setup
	credentialID := int64(123)
	webhookSecret := "test_secret"
	testBody := []byte(`{"test": "data"}`)

	expectedCredentials := db.PatreonCredential{
		ID:                 123,
		CreatorAccessToken: "test_token",
		CreatedBy:          1,
		WebhookSecret:      sql.NullString{String: webhookSecret, Valid: true},
	}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			return expectedCredentials, nil
		},
		PatreonClientByIDFunc: func(ctx context.Context, credentialsID int64) (patreon.Client, error) {
			return &patreon.ClientMock{
				ListCampaignsFunc: func() ([]patreon.Campaign, error) {
					return nil, errors.New("API error")
				},
			}, nil
		},
		GetPatreonCampaignsByCredentialsIDFunc: func(ctx context.Context, credentialsID int64) ([]db.PatreonCampaign, error) {
			return []db.PatreonCampaign{}, nil
		},
	}

	// Execute
	validSignature := generateValidSignature(testBody, webhookSecret)
	request := processpatreonwebhook.Request{
		CredentialID: credentialID,
		Signature:    validSignature,
		Body:         testBody,
	}
	result, err := processpatreonwebhook.Resolve(context.Background(), env, request)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed to refresh patreon data")
}

func TestResolve_DatabaseError(t *testing.T) {
	// Setup
	credentialID := int64(123)
	testBody := []byte(`{"test": "data"}`)

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			return db.PatreonCredential{}, errors.New("database connection error")
		},
	}

	// Execute
	request := processpatreonwebhook.Request{
		CredentialID: credentialID,
		Signature:    "any_signature",
		Body:         testBody,
	}
	result, err := processpatreonwebhook.Resolve(context.Background(), env, request)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "failed to get patreon credentials")
}

func TestResolve_InvalidSignature(t *testing.T) {
	// Setup
	credentialID := int64(123)
	webhookSecret := "test_secret"
	testBody := []byte(`{"test": "data"}`)

	expectedCredentials := db.PatreonCredential{
		ID:                 123,
		CreatorAccessToken: "test_token",
		CreatedBy:          1,
		WebhookSecret:      sql.NullString{String: webhookSecret, Valid: true},
	}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			return expectedCredentials, nil
		},
	}

	// Execute with invalid signature
	request := processpatreonwebhook.Request{
		CredentialID: credentialID,
		Signature:    "invalid_signature",
		Body:         testBody,
	}
	result, err := processpatreonwebhook.Resolve(context.Background(), env, request)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "invalid webhook signature")
}

func TestResolve_MissingWebhookSecret(t *testing.T) {
	// Setup
	credentialID := int64(123)
	testBody := []byte(`{"test": "data"}`)

	expectedCredentials := db.PatreonCredential{
		ID:                 123,
		CreatorAccessToken: "test_token",
		CreatedBy:          1,
		WebhookSecret:      sql.NullString{Valid: false}, // No webhook secret
	}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger {
			return &logger.TestLogger{}
		},
		PatreonCredentialsFunc: func(ctx context.Context, id int64) (db.PatreonCredential, error) {
			return expectedCredentials, nil
		},
	}

	// Execute
	request := processpatreonwebhook.Request{
		CredentialID: credentialID,
		Signature:    "any_signature",
		Body:         testBody,
	}
	result, err := processpatreonwebhook.Resolve(context.Background(), env, request)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "webhook secret not configured")
}
