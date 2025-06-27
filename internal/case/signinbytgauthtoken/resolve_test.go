package signinbytgauthtoken

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	tests := []struct {
		name     string
		rawToken string
		setup    func(*EnvMock)
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "successful sign-in with existing user",
			rawToken: "valid_token",
			setup: func(env *EnvMock) {
				token := &model.TgAuthToken{
					ChatID: 123456789,
					BotID:  987654321,
				}
				profile := db.TgUserProfile{
					ChatID: 123456789,
					BotID:  987654321,
				}
				user := db.User{
					ID:       1,
					TgUserID: sql.NullInt64{Valid: true, Int64: 123456789},
				}

				env.ParseTgAuthTokenFunc = func(ctx context.Context, tokenStr string) (*model.TgAuthToken, error) {
					return token, nil
				}
				env.TgUserProfileByChatIDAndBotIDFunc = func(ctx context.Context, arg db.TgUserProfileByChatIDAndBotIDParams) (db.TgUserProfile, error) {
					return profile, nil
				}
				env.UserByTgUserIDFunc = func(ctx context.Context, tgUserID sql.NullInt64) (db.User, error) {
					return user, nil
				}
				env.SetupUserTokenFunc = func(ctx context.Context, userID int64) (string, error) {
					return "user_token_123", nil
				}
			},
			wantErr: false,
		},
		{
			name:     "successful sign-in with new user creation",
			rawToken: "valid_token",
			setup: func(env *EnvMock) {
				token := &model.TgAuthToken{
					ChatID: 123456789,
					BotID:  987654321,
				}
				profile := db.TgUserProfile{
					ChatID: 123456789,
					BotID:  987654321,
				}
				user := db.User{
					ID:       2,
					TgUserID: sql.NullInt64{Valid: true, Int64: 123456789},
				}

				env.ParseTgAuthTokenFunc = func(ctx context.Context, tokenStr string) (*model.TgAuthToken, error) {
					return token, nil
				}
				env.TgUserProfileByChatIDAndBotIDFunc = func(ctx context.Context, arg db.TgUserProfileByChatIDAndBotIDParams) (db.TgUserProfile, error) {
					return profile, nil
				}
				env.UserByTgUserIDFunc = func(ctx context.Context, tgUserID sql.NullInt64) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				}
				env.InsertUserWithTgUserIDFunc = func(ctx context.Context, tgUserID sql.NullInt64) (db.User, error) {
					return user, nil
				}
				env.SetupUserTokenFunc = func(ctx context.Context, userID int64) (string, error) {
					return "user_token_456", nil
				}
			},
			wantErr: false,
		},
		{
			name:     "token parsing error",
			rawToken: "invalid_token",
			setup: func(env *EnvMock) {
				env.ParseTgAuthTokenFunc = func(ctx context.Context, tokenStr string) (*model.TgAuthToken, error) {
					return nil, errors.New("invalid token format")
				}
			},
			wantErr: true,
			errMsg:  "failed to parse token",
		},
		{
			name:     "profile not found error",
			rawToken: "valid_token",
			setup: func(env *EnvMock) {
				token := &model.TgAuthToken{
					ChatID: 123456789,
					BotID:  987654321,
				}

				env.ParseTgAuthTokenFunc = func(ctx context.Context, tokenStr string) (*model.TgAuthToken, error) {
					return token, nil
				}
				env.TgUserProfileByChatIDAndBotIDFunc = func(ctx context.Context, arg db.TgUserProfileByChatIDAndBotIDParams) (db.TgUserProfile, error) {
					return db.TgUserProfile{}, sql.ErrNoRows
				}
			},
			wantErr: true,
			errMsg:  "profile not found",
		},
		{
			name:     "profile lookup database error",
			rawToken: "valid_token",
			setup: func(env *EnvMock) {
				token := &model.TgAuthToken{
					ChatID: 123456789,
					BotID:  987654321,
				}

				env.ParseTgAuthTokenFunc = func(ctx context.Context, tokenStr string) (*model.TgAuthToken, error) {
					return token, nil
				}
				env.TgUserProfileByChatIDAndBotIDFunc = func(ctx context.Context, arg db.TgUserProfileByChatIDAndBotIDParams) (db.TgUserProfile, error) {
					return db.TgUserProfile{}, errors.New("database connection error")
				}
			},
			wantErr: true,
			errMsg:  "failed to get profile by chat ID and bot ID",
		},
		{
			name:     "user lookup database error",
			rawToken: "valid_token",
			setup: func(env *EnvMock) {
				token := &model.TgAuthToken{
					ChatID: 123456789,
					BotID:  987654321,
				}
				profile := db.TgUserProfile{
					ChatID: 123456789,
					BotID:  987654321,
				}

				env.ParseTgAuthTokenFunc = func(ctx context.Context, tokenStr string) (*model.TgAuthToken, error) {
					return token, nil
				}
				env.TgUserProfileByChatIDAndBotIDFunc = func(ctx context.Context, arg db.TgUserProfileByChatIDAndBotIDParams) (db.TgUserProfile, error) {
					return profile, nil
				}
				env.UserByTgUserIDFunc = func(ctx context.Context, tgUserID sql.NullInt64) (db.User, error) {
					return db.User{}, errors.New("database connection error")
				}
			},
			wantErr: true,
			errMsg:  "failed to get user by TG user ID",
		},
		{
			name:     "user creation database error",
			rawToken: "valid_token",
			setup: func(env *EnvMock) {
				token := &model.TgAuthToken{
					ChatID: 123456789,
					BotID:  987654321,
				}
				profile := db.TgUserProfile{
					ChatID: 123456789,
					BotID:  987654321,
				}

				env.ParseTgAuthTokenFunc = func(ctx context.Context, tokenStr string) (*model.TgAuthToken, error) {
					return token, nil
				}
				env.TgUserProfileByChatIDAndBotIDFunc = func(ctx context.Context, arg db.TgUserProfileByChatIDAndBotIDParams) (db.TgUserProfile, error) {
					return profile, nil
				}
				env.UserByTgUserIDFunc = func(ctx context.Context, tgUserID sql.NullInt64) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				}
				env.InsertUserWithTgUserIDFunc = func(ctx context.Context, tgUserID sql.NullInt64) (db.User, error) {
					return db.User{}, errors.New("failed to insert user")
				}
			},
			wantErr: true,
			errMsg:  "failed to insert user with TG user ID",
		},
		{
			name:     "token setup error",
			rawToken: "valid_token",
			setup: func(env *EnvMock) {
				token := &model.TgAuthToken{
					ChatID: 123456789,
					BotID:  987654321,
				}
				profile := db.TgUserProfile{
					ChatID: 123456789,
					BotID:  987654321,
				}
				user := db.User{
					ID:       1,
					TgUserID: sql.NullInt64{Valid: true, Int64: 123456789},
				}

				env.ParseTgAuthTokenFunc = func(ctx context.Context, tokenStr string) (*model.TgAuthToken, error) {
					return token, nil
				}
				env.TgUserProfileByChatIDAndBotIDFunc = func(ctx context.Context, arg db.TgUserProfileByChatIDAndBotIDParams) (db.TgUserProfile, error) {
					return profile, nil
				}
				env.UserByTgUserIDFunc = func(ctx context.Context, tgUserID sql.NullInt64) (db.User, error) {
					return user, nil
				}
				env.SetupUserTokenFunc = func(ctx context.Context, userID int64) (string, error) {
					return "", errors.New("failed to setup token")
				}
			},
			wantErr: true,
			errMsg:  "failed to setup user token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			if tt.setup != nil {
				tt.setup(env)
			}

			ctx := context.Background()
			err := Resolve(ctx, env, tt.rawToken)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestErrProfileNotFound(t *testing.T) {
	// Test that the exported error can be checked with errors.Is
	err := ErrProfileNotFound
	require.ErrorIs(t, err, ErrProfileNotFound)
	require.Equal(t, "profile not found", err.Error())
}

func TestIsValidRedirectURL(t *testing.T) {
	trustedDomains := []string{"example.com", "localhost:8081", "api.example.com"}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "relative path - should be allowed",
			url:      "/dashboard",
			expected: true,
		},
		{
			name:     "trusted domain exact match",
			url:      "https://example.com/path",
			expected: true,
		},
		{
			name:     "trusted domain with port",
			url:      "http://localhost:8081/admin",
			expected: true,
		},
		{
			name:     "trusted subdomain",
			url:      "https://api.example.com/v1",
			expected: true,
		},
		{
			name:     "untrusted domain - should be blocked",
			url:      "https://malicious.com/phish",
			expected: false,
		},
		{
			name:     "subdomain of trusted domain - should be blocked",
			url:      "https://sub.example.com/path",
			expected: false,
		},
		{
			name:     "similar domain - should be blocked",
			url:      "https://example.com.evil.com/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			require.NoError(t, err)

			result := isValidRedirectURL(u, trustedDomains)
			require.Equal(t, tt.expected, result)
		})
	}
}
