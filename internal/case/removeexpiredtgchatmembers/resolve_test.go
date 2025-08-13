package removeexpiredtgchatmembers

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	tests := []struct {
		name           string
		filter         Filter
		setup          func(*EnvMock)
		expectedResult *Result
		expectedError  string
	}{
		{
			name: "success - remove expired access for user",
			filter: Filter{
				UserID: ptr(int64(123)),
			},
			setup: func(env *EnvMock) {
				// Mock data: user has access to chat but no active subgraph subscription
				accesses := []db.ListTgBotChatSubgraphAccessesRow{
					{
						TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
							ChatID:     1,
							UserID:     123,
							SubgraphID: 10,
							CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
							JoinedAt:   sql.NullTime{Time: time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC), Valid: true},
						},
						Subgraph: db.Subgraph{
							ID:     10,
							Name:   "premium",
							Color:  sql.NullString{String: "blue", Valid: true},
							Hidden: false,
						},
						TgBotChat: db.TgBotChat{
							ID:         1,
							TelegramID: -1001234567890,
							ChatType:   "supergroup",
							ChatTitle:  "Premium Chat",
							BotID:      1,
						},
					},
				}

				env.ListTgBotChatSubgraphAccessesFunc = func(ctx context.Context, filter db.ListTgBotChatSubgraphAccessesParams) ([]db.ListTgBotChatSubgraphAccessesRow, error) {
					return accesses, nil
				}

				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					require.Equal(t, int64(123), userID)
					return []string{}, nil // No active subscriptions - access expired
				}

				env.UserByIDFunc = func(ctx context.Context, id int64) (db.User, error) {
					require.Equal(t, int64(123), id)
					return db.User{
						ID:       123,
						TgUserID: sql.NullInt64{Int64: 987654321, Valid: true},
					}, nil
				}

				env.KickTelegramChatMemberFunc = func(ctx context.Context, chatID, userID int64) error {
					require.Equal(t, int64(1), chatID)
					require.Equal(t, int64(123), userID)
					return nil
				}

				env.RemoveTgChatMemberFunc = func(ctx context.Context, arg db.RemoveTgChatMemberParams) error {
					require.Equal(t, int64(987654321), arg.UserID)
					require.Equal(t, int64(-1001234567890), arg.ChatID)
					return nil
				}

				env.SendTelegramMessageFunc = func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
					require.Equal(t, int64(1), chatID)
					return nil
				}

				env.DeleteTgBotChatSubgraphAccessFunc = func(ctx context.Context, arg db.DeleteTgBotChatSubgraphAccessParams) error {
					require.Equal(t, int64(1), arg.ChatID)
					require.Equal(t, int64(123), arg.UserID)
					require.Equal(t, int64(10), arg.SubgraphID)
					return nil
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedResult: &Result{
				RemovedCount: 1,
				Errors:       []error{},
			},
		},
		{
			name: "success - user has active subscription, no removal needed",
			filter: Filter{
				UserID: ptr(int64(123)),
			},
			setup: func(env *EnvMock) {
				accesses := []db.ListTgBotChatSubgraphAccessesRow{
					{
						TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
							ChatID:     1,
							UserID:     123,
							SubgraphID: 10,
						},
						Subgraph: db.Subgraph{
							ID:   10,
							Name: "premium",
						},
						TgBotChat: db.TgBotChat{
							ID:         1,
							TelegramID: -1001234567890,
							ChatTitle:  "Premium Chat",
						},
					},
				}

				env.ListTgBotChatSubgraphAccessesFunc = func(ctx context.Context, filter db.ListTgBotChatSubgraphAccessesParams) ([]db.ListTgBotChatSubgraphAccessesRow, error) {
					return accesses, nil
				}

				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					return []string{"premium"}, nil // User has active premium subscription
				}

				env.UserByIDFunc = func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:       123,
						TgUserID: sql.NullInt64{Int64: 987654321, Valid: true},
					}, nil
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedResult: &Result{
				RemovedCount: 0,
				Errors:       []error{},
			},
		},
		{
			name: "success - filter by chat ID",
			filter: Filter{
				ChatID: ptr(int64(1)),
			},
			setup: func(env *EnvMock) {
				env.ListTgBotChatSubgraphAccessesFunc = func(ctx context.Context, filter db.ListTgBotChatSubgraphAccessesParams) ([]db.ListTgBotChatSubgraphAccessesRow, error) {
					require.Equal(t, sql.NullInt64{Int64: 1, Valid: true}, filter.ChatID)
					return []db.ListTgBotChatSubgraphAccessesRow{}, nil
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedResult: &Result{
				RemovedCount: 0,
				Errors:       []error{},
			},
		},
		{
			name: "error - failed to list accesses",
			filter: Filter{
				UserID: ptr(int64(123)),
			},
			setup: func(env *EnvMock) {
				env.ListTgBotChatSubgraphAccessesFunc = func(ctx context.Context, filter db.ListTgBotChatSubgraphAccessesParams) ([]db.ListTgBotChatSubgraphAccessesRow, error) {
					return nil, errors.New("database error")
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedError: "failed to list tg bot chat subgraph accesses: database error",
		},
		{
			name: "error - failed to get user",
			filter: Filter{
				UserID: ptr(int64(123)),
			},
			setup: func(env *EnvMock) {
				accesses := []db.ListTgBotChatSubgraphAccessesRow{
					{
						TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
							UserID: 123,
						},
						Subgraph: db.Subgraph{Name: "premium"},
					},
				}

				env.ListTgBotChatSubgraphAccessesFunc = func(ctx context.Context, filter db.ListTgBotChatSubgraphAccessesParams) ([]db.ListTgBotChatSubgraphAccessesRow, error) {
					return accesses, nil
				}

				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					return []string{}, nil // No active subscription
				}

				env.UserByIDFunc = func(ctx context.Context, id int64) (db.User, error) {
					return db.User{}, errors.New("user not found")
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedResult: &Result{
				RemovedCount: 0,
				Errors: []error{
					errors.New("failed to get user by ID 123: user not found"),
				},
			},
		},
		{
			name: "error - failed to remove chat member",
			filter: Filter{
				UserID: ptr(int64(123)),
			},
			setup: func(env *EnvMock) {
				accesses := []db.ListTgBotChatSubgraphAccessesRow{
					{
						TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
							ChatID:     1,
							UserID:     123,
							SubgraphID: 10,
						},
						Subgraph: db.Subgraph{
							ID:   10,
							Name: "premium",
						},
						TgBotChat: db.TgBotChat{
							ID:         1,
							TelegramID: -1001234567890,
							ChatTitle:  "Premium Chat",
						},
					},
				}

				env.ListTgBotChatSubgraphAccessesFunc = func(ctx context.Context, filter db.ListTgBotChatSubgraphAccessesParams) ([]db.ListTgBotChatSubgraphAccessesRow, error) {
					return accesses, nil
				}

				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					return []string{}, nil // No active subscription
				}

				env.UserByIDFunc = func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:       123,
						TgUserID: sql.NullInt64{Int64: 987654321, Valid: true},
					}, nil
				}

				env.KickTelegramChatMemberFunc = func(ctx context.Context, chatID, userID int64) error {
					return nil // Let the kick succeed, but the database removal will fail
				}

				env.RemoveTgChatMemberFunc = func(ctx context.Context, arg db.RemoveTgChatMemberParams) error {
					return errors.New("failed to remove from telegram")
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedResult: &Result{
				RemovedCount: 0,
				Errors: []error{
					errors.New("failed to process expired access for user 123 in chat 1 (subgraph premium): failed to remove chat member from database (telegramUserID: 987654321, chatID: -1001234567890): failed to remove from telegram"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := &EnvMock{}
			test.setup(env)

			result, err := Resolve(context.Background(), env, test.filter)

			if test.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedError)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, test.expectedResult.RemovedCount, result.RemovedCount)

				if len(test.expectedResult.Errors) == 0 {
					require.Empty(t, result.Errors)
				} else {
					require.Len(t, result.Errors, len(test.expectedResult.Errors))
					for i, expectedErr := range test.expectedResult.Errors {
						require.Contains(t, result.Errors[i].Error(), expectedErr.Error())
					}
				}
			}
		})
	}
}

func TestProcessUser(t *testing.T) {
	tests := []struct {
		name          string
		userID        int64
		accesses      []*db.ListTgBotChatSubgraphAccessesRow
		setup         func(*EnvMock)
		expectedCount int
		expectedError string
	}{
		{
			name:   "success - remove one expired access",
			userID: 123,
			accesses: []*db.ListTgBotChatSubgraphAccessesRow{
				{
					TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
						ChatID:     1,
						UserID:     123,
						SubgraphID: 10,
					},
					Subgraph: db.Subgraph{
						Name: "premium",
					},
					TgBotChat: db.TgBotChat{
						ID:         1,
						TelegramID: -1001234567890,
						ChatTitle:  "Premium Chat",
					},
				},
			},
			setup: func(env *EnvMock) {
				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					return []string{}, nil // No active subscriptions
				}

				env.UserByIDFunc = func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:       123,
						TgUserID: sql.NullInt64{Int64: 987654321, Valid: true},
					}, nil
				}

				env.KickTelegramChatMemberFunc = func(ctx context.Context, chatID, userID int64) error {
					return nil
				}

				env.RemoveTgChatMemberFunc = func(ctx context.Context, arg db.RemoveTgChatMemberParams) error {
					return nil
				}

				env.SendTelegramMessageFunc = func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
					return nil
				}

				env.DeleteTgBotChatSubgraphAccessFunc = func(ctx context.Context, arg db.DeleteTgBotChatSubgraphAccessParams) error {
					return nil
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedCount: 1,
		},
		{
			name:   "success - user has valid access, no removal",
			userID: 123,
			accesses: []*db.ListTgBotChatSubgraphAccessesRow{
				{
					TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
						ChatID:     1,
						UserID:     123,
						SubgraphID: 10,
					},
					Subgraph: db.Subgraph{
						Name: "premium",
					},
				},
			},
			setup: func(env *EnvMock) {
				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					return []string{"premium"}, nil // User has active premium subscription
				}

				env.UserByIDFunc = func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:       123,
						TgUserID: sql.NullInt64{Int64: 987654321, Valid: true},
					}, nil
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedCount: 0,
		},
		{
			name:   "error - failed to get user subgraphs",
			userID: 123,
			accesses: []*db.ListTgBotChatSubgraphAccessesRow{
				{
					Subgraph: db.Subgraph{Name: "premium"},
				},
			},
			setup: func(env *EnvMock) {
				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					return nil, errors.New("database error")
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedError: "database error",
		},
		{
			name:   "error - user not found",
			userID: 123,
			accesses: []*db.ListTgBotChatSubgraphAccessesRow{
				{
					Subgraph: db.Subgraph{Name: "premium"},
				},
			},
			setup: func(env *EnvMock) {
				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					return []string{}, nil
				}

				env.UserByIDFunc = func(ctx context.Context, id int64) (db.User, error) {
					return db.User{}, errors.New("user not found")
				}

				env.LoggerFunc = func() logger.Logger {
					return &logger.TestLogger{Prefix: "[TEST]"}
				}
			},
			expectedError: "failed to get user by ID 123: user not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := &EnvMock{}
			test.setup(env)

			count, err := processUser(context.Background(), env, test.userID, test.accesses)

			if test.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedCount, count)
			}
		})
	}
}

func TestProcessExpiredAccess(t *testing.T) {
	tests := []struct {
		name          string
		user          *db.User
		access        *db.ListTgBotChatSubgraphAccessesRow
		setup         func(*EnvMock)
		expectedError string
	}{
		{
			name: "success - complete removal process",
			user: &db.User{
				ID:       123,
				TgUserID: sql.NullInt64{Int64: 987654321, Valid: true},
			},
			access: &db.ListTgBotChatSubgraphAccessesRow{
				TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
					ChatID:     1,
					UserID:     123,
					SubgraphID: 10,
				},
				Subgraph: db.Subgraph{
					Name: "premium",
				},
				TgBotChat: db.TgBotChat{
					ID:         1,
					TelegramID: -1001234567890,
					ChatTitle:  "Premium Chat",
				},
			},
			setup: func(env *EnvMock) {
				env.KickTelegramChatMemberFunc = func(ctx context.Context, chatID, userID int64) error {
					require.Equal(t, int64(1), chatID)
					require.Equal(t, int64(123), userID)
					return nil
				}

				env.RemoveTgChatMemberFunc = func(ctx context.Context, arg db.RemoveTgChatMemberParams) error {
					require.Equal(t, int64(987654321), arg.UserID)
					require.Equal(t, int64(-1001234567890), arg.ChatID)
					return nil
				}

				env.SendTelegramMessageFunc = func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
					require.Equal(t, int64(1), chatID)
					return nil
				}

				env.DeleteTgBotChatSubgraphAccessFunc = func(ctx context.Context, arg db.DeleteTgBotChatSubgraphAccessParams) error {
					require.Equal(t, int64(1), arg.ChatID)
					require.Equal(t, int64(123), arg.UserID)
					require.Equal(t, int64(10), arg.SubgraphID)
					return nil
				}
			},
		},
		{
			name: "success - user without telegram ID",
			user: &db.User{
				ID:       123,
				TgUserID: sql.NullInt64{Valid: false}, // No telegram ID
			},
			access: &db.ListTgBotChatSubgraphAccessesRow{
				TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
					ChatID:     1,
					UserID:     123,
					SubgraphID: 10,
				},
				Subgraph: db.Subgraph{
					Name: "premium",
				},
				TgBotChat: db.TgBotChat{
					ID:         1,
					TelegramID: -1001234567890,
					ChatTitle:  "Premium Chat",
				},
			},
			setup: func(env *EnvMock) {
				// KickTelegramChatMember should NOT be called since user has no telegram ID
				// (the function checks user.TgUserID.Valid before calling)

				env.RemoveTgChatMemberFunc = func(ctx context.Context, arg db.RemoveTgChatMemberParams) error {
					require.Equal(t, int64(0), arg.UserID) // Should be 0 when no telegram ID
					return nil
				}

				// SendTelegramMessage should not be called since user has no telegram ID

				env.DeleteTgBotChatSubgraphAccessFunc = func(ctx context.Context, arg db.DeleteTgBotChatSubgraphAccessParams) error {
					return nil
				}
			},
		},
		{
			name: "error - failed to remove chat member",
			user: &db.User{
				ID:       123,
				TgUserID: sql.NullInt64{Int64: 987654321, Valid: true},
			},
			access: &db.ListTgBotChatSubgraphAccessesRow{
				TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
					ChatID:     1,
					UserID:     123,
					SubgraphID: 10,
				},
				TgBotChat: db.TgBotChat{
					ID:         1,
					TelegramID: -1001234567890,
				},
			},
			setup: func(env *EnvMock) {
				env.KickTelegramChatMemberFunc = func(ctx context.Context, chatID, userID int64) error {
					require.Equal(t, int64(1), chatID)
					require.Equal(t, int64(123), userID)
					return nil // Let kick succeed, database removal will fail
				}

				env.RemoveTgChatMemberFunc = func(ctx context.Context, arg db.RemoveTgChatMemberParams) error {
					return errors.New("telegram API error")
				}
			},
			expectedError: "failed to remove chat member from database (telegramUserID: 987654321, chatID: -1001234567890): telegram API error",
		},
		{
			name: "error - failed to send notification",
			user: &db.User{
				ID:       123,
				TgUserID: sql.NullInt64{Int64: 987654321, Valid: true},
			},
			access: &db.ListTgBotChatSubgraphAccessesRow{
				TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
					ChatID:     1,
					UserID:     123,
					SubgraphID: 10,
				},
				Subgraph: db.Subgraph{
					Name: "premium",
				},
				TgBotChat: db.TgBotChat{
					ID:         1,
					TelegramID: -1001234567890,
					ChatTitle:  "Premium Chat",
				},
			},
			setup: func(env *EnvMock) {
				env.KickTelegramChatMemberFunc = func(ctx context.Context, chatID, userID int64) error {
					require.Equal(t, int64(1), chatID)
					require.Equal(t, int64(123), userID)
					return nil
				}

				env.RemoveTgChatMemberFunc = func(ctx context.Context, arg db.RemoveTgChatMemberParams) error {
					return nil
				}

				env.SendTelegramMessageFunc = func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
					return errors.New("telegram send error")
				}
			},
			expectedError: "failed to send expiration notification to telegram user 987654321: failed to send telegram message to user: telegram send error",
		},
		{
			name: "error - failed to delete access record",
			user: &db.User{
				ID:       123,
				TgUserID: sql.NullInt64{Int64: 987654321, Valid: true},
			},
			access: &db.ListTgBotChatSubgraphAccessesRow{
				TgBotChatSubgraphAccess: db.TgBotChatSubgraphAccess{
					ChatID:     1,
					UserID:     123,
					SubgraphID: 10,
				},
				Subgraph: db.Subgraph{
					Name: "premium",
				},
				TgBotChat: db.TgBotChat{
					ID:         1,
					TelegramID: -1001234567890,
					ChatTitle:  "Premium Chat",
				},
			},
			setup: func(env *EnvMock) {
				env.KickTelegramChatMemberFunc = func(ctx context.Context, chatID, userID int64) error {
					require.Equal(t, int64(1), chatID)
					require.Equal(t, int64(123), userID)
					return nil
				}

				env.RemoveTgChatMemberFunc = func(ctx context.Context, arg db.RemoveTgChatMemberParams) error {
					return nil
				}

				env.SendTelegramMessageFunc = func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
					return nil
				}

				env.DeleteTgBotChatSubgraphAccessFunc = func(ctx context.Context, arg db.DeleteTgBotChatSubgraphAccessParams) error {
					return errors.New("database deletion error")
				}
			},
			expectedError: "failed to delete access record: database deletion error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := &EnvMock{}
			test.setup(env)

			err := processExpiredAccess(context.Background(), env, test.user, test.access)

			if test.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetUserSubgraphs(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		setup          func(*EnvMock)
		expectedResult map[string]struct{}
		expectedError  string
	}{
		{
			name:   "success - user has multiple subgraphs",
			userID: 123,
			setup: func(env *EnvMock) {
				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					require.Equal(t, int64(123), userID)
					return []string{"premium", "basic", "vip"}, nil
				}
			},
			expectedResult: map[string]struct{}{
				"premium": {},
				"basic":   {},
				"vip":     {},
			},
		},
		{
			name:   "success - user has no subgraphs",
			userID: 123,
			setup: func(env *EnvMock) {
				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					return []string{}, nil
				}
			},
			expectedResult: map[string]struct{}{},
		},
		{
			name:   "error - database error",
			userID: 123,
			setup: func(env *EnvMock) {
				env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
					return nil, errors.New("database connection failed")
				}
			},
			expectedError: "failed to list active user subgraphs: database connection failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := &EnvMock{}
			test.setup(env)

			result, err := getUserSubgraphs(context.Background(), env, test.userID)

			if test.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedError)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedResult, result)
			}
		})
	}
}

func TestSendExpirationNotification(t *testing.T) {
	tests := []struct {
		name           string
		chatID         int64
		telegramUserID int64
		chatTitle      string
		subgraphName   string
		setup          func(*EnvMock)
		expectedError  string
	}{
		{
			name:           "success - send notification",
			chatID:         1,
			telegramUserID: 987654321,
			chatTitle:      "Premium Chat",
			subgraphName:   "premium",
			setup: func(env *EnvMock) {
				env.SendTelegramMessageFunc = func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
					require.Equal(t, int64(1), chatID)

					// Verify the message content
					telegramMsg, ok := msg.(tgbotapi.MessageConfig)
					require.True(t, ok)
					require.Equal(t, int64(987654321), telegramMsg.ChatID)
					require.Contains(t, telegramMsg.Text, "Premium Chat")
					require.Contains(t, telegramMsg.Text, "premium")
					require.Contains(t, telegramMsg.Text, "истёк")

					return nil
				}
			},
		},
		{
			name:           "error - telegram send failed",
			chatID:         1,
			telegramUserID: 987654321,
			chatTitle:      "Premium Chat",
			subgraphName:   "premium",
			setup: func(env *EnvMock) {
				env.SendTelegramMessageFunc = func(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
					return errors.New("telegram API unavailable")
				}
			},
			expectedError: "failed to send telegram message to user: telegram API unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := &EnvMock{}
			test.setup(env)

			err := sendExpirationNotification(context.Background(), env, test.chatID, test.telegramUserID, test.chatTitle, test.subgraphName)

			if test.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Helper function to create a pointer to int64
func ptr(v int64) *int64 {
	return &v
}

// Add some debug helpers if needed
func debugResult(result *Result) {
	if result != nil {
		pretty.Printf("Result: %# v\n", result)
	}
}
