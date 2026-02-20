package checkapikey_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/case/checkapikey"
	"trip2g/internal/db"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/usertoken"
)

func setupRequestContextWithBearer(token string) (*fasthttp.RequestCtx, *appreq.Request) {
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("Authorization", "Bearer "+token)
	reqCtx.Request.SetRequestURI("http://example.com/test")

	req := &appreq.Request{Req: reqCtx}
	req.StoreInContext()

	return reqCtx, req
}

func TestResolveWithBearerToken(t *testing.T) {
	const testSecret = "test-secret-key-for-jwt"

	tests := []struct {
		name              string
		tokenData         shortapitoken.Data
		tokenTTL          time.Duration
		wantErr           error
		checkVirtualKey   bool
		checkWebhookData  bool
		expectedDepth     int
		expectedReadPats  []string
		expectedWritePats []string
	}{
		{
			name: "valid Bearer token with webhook depth and patterns",
			tokenData: shortapitoken.Data{
				Depth:         3,
				ReadPatterns:  []string{"read/*", "view/*"},
				WritePatterns: []string{"update/*"},
			},
			tokenTTL:          time.Hour,
			checkVirtualKey:   true,
			checkWebhookData:  true,
			expectedDepth:     3,
			expectedReadPats:  []string{"read/*", "view/*"},
			expectedWritePats: []string{"update/*"},
		},
		{
			name: "valid Bearer token with zero depth",
			tokenData: shortapitoken.Data{
				Depth:         0,
				ReadPatterns:  []string{},
				WritePatterns: []string{},
			},
			tokenTTL:          time.Hour,
			checkVirtualKey:   true,
			checkWebhookData:  true,
			expectedDepth:     0,
			expectedReadPats:  []string{},
			expectedWritePats: []string{},
		},
		{
			name: "expired Bearer token",
			tokenData: shortapitoken.Data{
				Depth:         1,
				ReadPatterns:  []string{"read/*"},
				WritePatterns: []string{},
			},
			tokenTTL: -time.Hour, // Expired.
			wantErr:  checkapikey.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := shortapitoken.Sign(tt.tokenData, testSecret, tt.tokenTTL)
			require.NoError(t, err)

			env := &EnvMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, nil
				},
				ShortAPITokenSecretFunc: func() string {
					return testSecret
				},
			}

			ctx, req := setupRequestContextWithBearer(token)

			apiKey, err := checkapikey.Resolve(ctx, env, "test-action")

			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			if tt.checkVirtualKey {
				require.NotNil(t, apiKey)
				require.Equal(t, int64(0), apiKey.ID, "should be virtual key with ID 0")
				require.Equal(t, "shortapitoken", apiKey.Value)
				require.Equal(t, "Short API token (webhook)", apiKey.Description)
				require.False(t, apiKey.SkipWebhooks)
			}

			if tt.checkWebhookData {
				require.Equal(t, tt.expectedDepth, req.WebhookDepth)
				require.Equal(t, tt.expectedReadPats, req.WebhookReadPatterns)
				require.Equal(t, tt.expectedWritePats, req.WebhookWritePatterns)
			}
		})
	}
}

func TestResolvePrefersAPIKeyOverBearer(t *testing.T) {
	const testSecret = "test-secret-key"

	// Create a valid Bearer token.
	tokenData := shortapitoken.Data{
		Depth:         5,
		ReadPatterns:  []string{"read/*"},
		WritePatterns: []string{"write/*"},
	}
	token, err := shortapitoken.Sign(tokenData, testSecret, time.Hour)
	require.NoError(t, err)

	// Setup request with BOTH X-API-Key and Authorization headers.
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("X-API-Key", "my-api-key")
	reqCtx.Request.Header.Set("Authorization", "Bearer "+token)
	reqCtx.Request.SetRequestURI("http://example.com/test")

	req := &appreq.Request{Req: reqCtx}
	req.StoreInContext()

	// Mock environment that tracks which method was called.
	apiKeyByValueCalled := false
	env := &EnvMock{
		CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return nil, nil
		},
		ApiKeyByValueFunc: func(ctx context.Context, value string) (db.ApiKey, error) {
			apiKeyByValueCalled = true
			// Return a real API key (not a virtual one).
			return db.ApiKey{
				ID:    999,
				Value: value,
			}, nil
		},
		UpsertAPIKeyLogActionFunc: func(ctx context.Context, name string) error {
			return nil
		},
		UpsertAPIKeyLogIPFunc: func(ctx context.Context, ip string) error {
			return nil
		},
		InsertAPIKeyLogFunc: func(ctx context.Context, arg db.InsertAPIKeyLogParams) error {
			return nil
		},
		ShortAPITokenSecretFunc: func() string {
			return testSecret
		},
	}

	apiKey, err := checkapikey.Resolve(reqCtx, env, "test-action")
	require.NoError(t, err)
	require.NotNil(t, apiKey)

	// X-API-Key should have been used (ID != 0 means real API key).
	require.Equal(t, int64(999), apiKey.ID)
	require.True(t, apiKeyByValueCalled)

	// Webhook data should NOT be set.
	require.Equal(t, 0, req.WebhookDepth)
	require.Empty(t, req.WebhookReadPatterns)
	require.Empty(t, req.WebhookWritePatterns)
}

func TestResolveInvalidBearerToken(t *testing.T) {
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("Authorization", "Bearer invalid-token-xyz")
	reqCtx.Request.SetRequestURI("http://example.com/test")

	req := &appreq.Request{Req: reqCtx}
	req.StoreInContext()

	env := &EnvMock{
		CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return nil, nil
		},
		ShortAPITokenSecretFunc: func() string {
			return "test-secret"
		},
	}

	apiKey, err := checkapikey.Resolve(reqCtx, env, "test-action")

	require.Error(t, err)
	require.ErrorIs(t, err, checkapikey.ErrInvalidToken)
	require.Nil(t, apiKey)
}
