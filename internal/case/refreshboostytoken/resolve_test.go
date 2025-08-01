package refreshboostytoken

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/boosty"
	"trip2g/internal/db"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	ctx := context.Background()

	// Mock auth data
	authData := boosty.AuthData{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
		ExpiresAt:    "1234567890",
	}
	authDataJSON, _ := json.Marshal(authData)

	// Mock credential
	mockCred := db.BoostyCredential{
		ID:       1,
		AuthData: string(authDataJSON),
		DeviceID: "test-device-id",
		BlogName: "test-blog",
	}

	// Mock client
	mockClient := &boosty.ClientMock{
		RefreshTokenFunc: func() (*boosty.RefreshTokenResponse, error) {
			return &boosty.RefreshTokenResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
			}, nil
		},
	}

	tests := []struct {
		name      string
		credID    int64
		setupMock func(env *EnvMock)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "successful token refresh",
			credID: 1,
			setupMock: func(env *EnvMock) {
				env.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					require.Equal(t, int64(1), id)
					return mockCred, nil
				}
				env.BoostyClientByCredentialsIDFunc = func(ctx context.Context, credentialID int64) (boosty.Client, error) {
					require.Equal(t, int64(1), credentialID)
					return mockClient, nil
				}
				env.UpdateBoostyCredentialsTokensFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsTokensParams) (db.BoostyCredential, error) {
					require.Equal(t, int64(1), arg.ID)
					// Check that auth data contains new tokens
					var authData boosty.AuthData
					err := json.Unmarshal([]byte(arg.AuthData), &authData)
					require.NoError(t, err)
					require.Equal(t, "new-access-token", authData.AccessToken)
					require.Equal(t, "new-refresh-token", authData.RefreshToken)
					require.True(t, arg.ExpiresAt.Valid)
					require.True(t, arg.ExpiresAt.Time.After(time.Now()))
					return db.BoostyCredential{}, nil
				}
			},
			wantErr: false,
		},
		{
			name:   "credential not found",
			credID: 999,
			setupMock: func(env *EnvMock) {
				env.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{}, sql.ErrNoRows
				}
			},
			wantErr: true,
			errMsg:  "failed to get boosty credential",
		},
		{
			name:   "client creation fails",
			credID: 1,
			setupMock: func(env *EnvMock) {
				env.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return mockCred, nil
				}
				env.BoostyClientByCredentialsIDFunc = func(ctx context.Context, credentialID int64) (boosty.Client, error) {
					return nil, fmt.Errorf("failed to create client")
				}
			},
			wantErr: true,
			errMsg:  "failed to get boosty client",
		},
		{
			name:   "token refresh fails",
			credID: 1,
			setupMock: func(env *EnvMock) {
				env.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return mockCred, nil
				}
				env.BoostyClientByCredentialsIDFunc = func(ctx context.Context, credentialID int64) (boosty.Client, error) {
					return &boosty.ClientMock{
						RefreshTokenFunc: func() (*boosty.RefreshTokenResponse, error) {
							return nil, fmt.Errorf("API error")
						},
					}, nil
				}
			},
			wantErr: true,
			errMsg:  "failed to refresh token",
		},
		{
			name:   "invalid auth data after refresh",
			credID: 1,
			setupMock: func(env *EnvMock) {
				env.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return db.BoostyCredential{
						ID:       1,
						AuthData: "invalid json",
						DeviceID: "test-device-id",
						BlogName: "test-blog",
					}, nil
				}
				env.BoostyClientByCredentialsIDFunc = func(ctx context.Context, credentialID int64) (boosty.Client, error) {
					return mockClient, nil
				}
			},
			wantErr: true,
			errMsg:  "failed to unmarshal auth data",
		},
		{
			name:   "update fails",
			credID: 1,
			setupMock: func(env *EnvMock) {
				env.BoostyCredentialsFunc = func(ctx context.Context, id int64) (db.BoostyCredential, error) {
					return mockCred, nil
				}
				env.BoostyClientByCredentialsIDFunc = func(ctx context.Context, credentialID int64) (boosty.Client, error) {
					return mockClient, nil
				}
				env.UpdateBoostyCredentialsTokensFunc = func(ctx context.Context, arg db.UpdateBoostyCredentialsTokensParams) (db.BoostyCredential, error) {
					return db.BoostyCredential{}, fmt.Errorf("database error")
				}
			},
			wantErr: true,
			errMsg:  "failed to update boosty credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We cannot test the actual token refresh without mocking the boosty client
			// This test focuses on the database interaction logic

			// All tests can now run with mocked client

			env := &EnvMock{}
			tt.setupMock(env)

			err := Resolve(ctx, env, tt.credID)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
