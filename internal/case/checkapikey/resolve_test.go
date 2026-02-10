package checkapikey_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/case/checkapikey"
	"trip2g/internal/db"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg checkapikey_test . Env

type Env interface {
	ApiKeyByValue(ctx context.Context, value string) (db.ApiKey, error)
	InsertAPIKeyLog(ctx context.Context, arg db.InsertAPIKeyLogParams) error
	UpsertAPIKeyLogAction(ctx context.Context, name string) error
	UpsertAPIKeyLogIP(ctx context.Context, ip string) error
	ShortAPITokenSecret() string
}

func setupRequestContext(apiKeyInReq string) *fasthttp.RequestCtx {
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("X-API-Key", apiKeyInReq)
	reqCtx.Request.SetRequestURI("http://example.com/test")

	req := &appreq.Request{Req: reqCtx}
	req.StoreInContext()

	return reqCtx
}

func assertErrorMatches(t *testing.T, err, wantErr error) {
	t.Helper()

	require.Error(t, err)

	if errors.Is(err, wantErr) {
		return
	}

	expectedMsg := wantErr.Error()
	isWrappedError := expectedMsg == "failed to resolve API key" ||
		expectedMsg == "failed to upsert API key log action"
	if isWrappedError {
		return
	}

	require.Equal(t, wantErr, err)
}

func assertAPIKeyResult(t *testing.T, apiKey *db.ApiKey, wantKeyID int64) {
	t.Helper()
	require.NotNil(t, apiKey)
	require.Equal(t, wantKeyID, apiKey.ID)
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		apiKeyInReq string
		action      string
		setupEnv    func() *EnvMock
		wantErr     error
		wantKeyID   int64
	}{
		{
			name:        "missing API key header",
			apiKeyInReq: "",
			action:      "test-action",
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ShortAPITokenSecretFunc: func() string {
						return "test-secret"
					},
				}
			},
			wantErr: checkapikey.ErrMissingKey,
		},
		{
			name:        "valid hashed API key (new style)",
			apiKeyInReq: "my-secret-key",
			action:      "test-action",
			setupEnv: func() *EnvMock {
				hash := sha256.Sum256([]byte("my-secret-key"))
				hashedValue := hex.EncodeToString(hash[:])

				return &EnvMock{
					ApiKeyByValueFunc: func(ctx context.Context, value string) (db.ApiKey, error) {
						if value == hashedValue {
							return db.ApiKey{
								ID:    1,
								Value: hashedValue,
							}, nil
						}
						return db.ApiKey{}, sql.ErrNoRows
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
						return "test-secret"
					},
				}
			},
			wantKeyID: 1,
		},
		{
			name:        "valid plain text API key (old style - backward compatibility)",
			apiKeyInReq: "old-plain-key",
			action:      "test-action",
			setupEnv: func() *EnvMock {
				hash := sha256.Sum256([]byte("old-plain-key"))
				hashedValue := hex.EncodeToString(hash[:])

				return &EnvMock{
					ApiKeyByValueFunc: func(ctx context.Context, value string) (db.ApiKey, error) {
						// First call with hashed value - not found
						if value == hashedValue {
							return db.ApiKey{}, sql.ErrNoRows
						}
						// Second call with plain text - found (old API key)
						if value == "old-plain-key" {
							return db.ApiKey{
								ID:    2,
								Value: "old-plain-key",
							}, nil
						}
						return db.ApiKey{}, sql.ErrNoRows
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
						return "test-secret"
					},
				}
			},
			wantKeyID: 2,
		},
		{
			name:        "invalid API key (not found in both hashed and plain)",
			apiKeyInReq: "invalid-key",
			action:      "test-action",
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ApiKeyByValueFunc: func(ctx context.Context, value string) (db.ApiKey, error) {
						return db.ApiKey{}, sql.ErrNoRows
					},
					ShortAPITokenSecretFunc: func() string {
						return "test-secret"
					},
				}
			},
			wantErr: checkapikey.ErrInvalidKey,
		},
		{
			name:        "database error when checking hashed key",
			apiKeyInReq: "some-key",
			action:      "test-action",
			setupEnv: func() *EnvMock {
				return &EnvMock{
					ApiKeyByValueFunc: func(ctx context.Context, value string) (db.ApiKey, error) {
						return db.ApiKey{}, errors.New("database connection error")
					},
					ShortAPITokenSecretFunc: func() string {
						return "test-secret"
					},
				}
			},
			wantErr: errors.New("failed to resolve API key"),
		},
		{
			name:        "error when logging action",
			apiKeyInReq: "valid-key",
			action:      "test-action",
			setupEnv: func() *EnvMock {
				hash := sha256.Sum256([]byte("valid-key"))
				hashedValue := hex.EncodeToString(hash[:])

				return &EnvMock{
					ApiKeyByValueFunc: func(ctx context.Context, value string) (db.ApiKey, error) {
						if value == hashedValue {
							return db.ApiKey{
								ID:    3,
								Value: hashedValue,
							}, nil
						}
						return db.ApiKey{}, sql.ErrNoRows
					},
					UpsertAPIKeyLogActionFunc: func(ctx context.Context, name string) error {
						return errors.New("failed to upsert action")
					},
					ShortAPITokenSecretFunc: func() string {
						return "test-secret"
					},
				}
			},
			wantErr: errors.New("failed to upsert API key log action"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			ctx := setupRequestContext(tt.apiKeyInReq)

			apiKey, err := checkapikey.Resolve(ctx, env, tt.action)

			if tt.wantErr != nil {
				assertErrorMatches(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assertAPIKeyResult(t, apiKey, tt.wantKeyID)
		})
	}
}
