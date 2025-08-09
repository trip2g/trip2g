package settgchatsubgraphinvites

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		input      Input
		setupMocks func(*EnvMock)
		wantErr    bool
		errMsg     string
		checkResp  func(t *testing.T, resp Payload)
	}{
		{
			name: "success - add multiple invites",
			input: Input{
				ChatID:      123,
				SubgraphIds: []int64{1, 2, 3},
			},
			setupMocks: func(m *EnvMock) {
				// Mock getting admin token
				m.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 42}, nil
				}

				// Mock delete existing invites
				m.DeleteTgChatSubgraphInvitesByChatIDFunc = func(ctx context.Context, chatID int64) error {
					require.Equal(t, int64(123), chatID)
					return nil
				}

				// Mock insert new invites
				insertCount := 0
				m.InsertTgChatSubgraphInviteFunc = func(ctx context.Context, arg db.InsertTgChatSubgraphInviteParams) (db.TgBotChatSubgraphInvite, error) {
					insertCount++

					// Verify parameters
					require.Equal(t, int64(123), arg.ChatID)
					require.Equal(t, int64(42), arg.CreatedBy)

					// Check that subgraph IDs are as expected
					switch insertCount {
					case 1:
						require.Equal(t, int64(1), arg.SubgraphID)
					case 2:
						require.Equal(t, int64(2), arg.SubgraphID)
					case 3:
						require.Equal(t, int64(3), arg.SubgraphID)
					default:
						t.Fatalf("unexpected insert count: %d", insertCount)
					}

					return db.TgBotChatSubgraphInvite{
						ChatID:     arg.ChatID,
						SubgraphID: arg.SubgraphID,
						CreatedBy:  arg.CreatedBy,
					}, nil
				}
			},
			checkResp: func(t *testing.T, resp Payload) {
				payload, ok := resp.(*model.SetTgChatSubgraphInvitesPayload)
				require.True(t, ok, "expected SetTgChatSubgraphInvitesPayload, got %T", resp)
				require.True(t, payload.Success)
				require.Equal(t, int64(123), payload.ChatID)
			},
		},
		{
			name: "success - clear all invites (empty array)",
			input: Input{
				ChatID:      456,
				SubgraphIds: []int64{},
			},
			setupMocks: func(m *EnvMock) {
				m.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 42}, nil
				}

				m.DeleteTgChatSubgraphInvitesByChatIDFunc = func(ctx context.Context, chatID int64) error {
					require.Equal(t, int64(456), chatID)
					return nil
				}

				// InsertTgChatSubgraphInvite should not be called for empty array
				m.InsertTgChatSubgraphInviteFunc = func(ctx context.Context, arg db.InsertTgChatSubgraphInviteParams) (db.TgBotChatSubgraphInvite, error) {
					t.Fatal("InsertTgChatSubgraphInvite should not be called for empty array")
					return db.TgBotChatSubgraphInvite{}, nil
				}
			},
			checkResp: func(t *testing.T, resp Payload) {
				payload, ok := resp.(*model.SetTgChatSubgraphInvitesPayload)
				require.True(t, ok)
				require.True(t, payload.Success)
				require.Equal(t, int64(456), payload.ChatID)
			},
		},
		{
			name: "validation error - missing chat ID",
			input: Input{
				ChatID:      0, // Required field
				SubgraphIds: []int64{1, 2},
			},
			setupMocks: func(m *EnvMock) {
				// No mocks should be called due to validation failure
			},
			checkResp: func(t *testing.T, resp Payload) {
				errPayload, ok := resp.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload, got %T", resp)
				// Check either Message or ByFields contains the error
				hasError := errPayload.Message != "" && strings.Contains(errPayload.Message, "chatId")
				for _, field := range errPayload.ByFields {
					if strings.Contains(field.Value, "chatId") || field.Name == "chatId" {
						hasError = true
						break
					}
				}
				require.True(t, hasError, "Expected error about chatId, got: %+v", errPayload)
			},
		},
		{
			name: "validation error - nil subgraph IDs",
			input: Input{
				ChatID:      123,
				SubgraphIds: nil, // Must be non-nil (can be empty array though)
			},
			setupMocks: func(m *EnvMock) {
				// No mocks should be called due to validation failure
			},
			checkResp: func(t *testing.T, resp Payload) {
				errPayload, ok := resp.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload, got %T", resp)
				// Check either Message or ByFields contains the error
				hasError := errPayload.Message != "" && strings.Contains(errPayload.Message, "subgraphIds")
				for _, field := range errPayload.ByFields {
					if strings.Contains(field.Value, "subgraphIds") || field.Name == "subgraphIds" {
						hasError = true
						break
					}
				}
				require.True(t, hasError, "Expected error about subgraphIds, got: %+v", errPayload)
			},
		},
		{
			name: "error - delete existing invites fails",
			input: Input{
				ChatID:      123,
				SubgraphIds: []int64{1},
			},
			setupMocks: func(m *EnvMock) {
				m.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 42}, nil
				}

				m.DeleteTgChatSubgraphInvitesByChatIDFunc = func(ctx context.Context, chatID int64) error {
					return errors.New("database error")
				}
			},
			wantErr: true,
			errMsg:  "failed to delete existing chat subgraph invites",
		},
		{
			name: "error - insert invite fails",
			input: Input{
				ChatID:      123,
				SubgraphIds: []int64{1, 2},
			},
			setupMocks: func(m *EnvMock) {
				m.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 42}, nil
				}

				m.DeleteTgChatSubgraphInvitesByChatIDFunc = func(ctx context.Context, chatID int64) error {
					return nil
				}

				insertCount := 0
				m.InsertTgChatSubgraphInviteFunc = func(ctx context.Context, arg db.InsertTgChatSubgraphInviteParams) (db.TgBotChatSubgraphInvite, error) {
					insertCount++
					if insertCount == 2 {
						// Fail on second insert
						return db.TgBotChatSubgraphInvite{}, errors.New("unique constraint violation")
					}
					return db.TgBotChatSubgraphInvite{
						ChatID:     arg.ChatID,
						SubgraphID: arg.SubgraphID,
						CreatedBy:  arg.CreatedBy,
					}, nil
				}
			},
			wantErr: true,
			errMsg:  "failed to insert chat subgraph invite 2",
		},
		{
			name: "success - replace existing invites",
			input: Input{
				ChatID:      789,
				SubgraphIds: []int64{5, 6},
			},
			setupMocks: func(m *EnvMock) {
				m.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 99}, nil
				}

				m.DeleteTgChatSubgraphInvitesByChatIDFunc = func(ctx context.Context, chatID int64) error {
					require.Equal(t, int64(789), chatID)
					return nil
				}

				m.InsertTgChatSubgraphInviteFunc = func(ctx context.Context, arg db.InsertTgChatSubgraphInviteParams) (db.TgBotChatSubgraphInvite, error) {
					require.Equal(t, int64(789), arg.ChatID)
					require.Equal(t, int64(99), arg.CreatedBy)
					require.Contains(t, []int64{5, 6}, arg.SubgraphID)

					return db.TgBotChatSubgraphInvite{
						ChatID:     arg.ChatID,
						SubgraphID: arg.SubgraphID,
						CreatedBy:  arg.CreatedBy,
					}, nil
				}
			},
			checkResp: func(t *testing.T, resp Payload) {
				payload, ok := resp.(*model.SetTgChatSubgraphInvitesPayload)
				require.True(t, ok)
				require.True(t, payload.Success)
				require.Equal(t, int64(789), payload.ChatID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			if tt.setupMocks != nil {
				tt.setupMocks(env)
			}

			resp, err := Resolve(ctx, env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			if tt.checkResp != nil {
				tt.checkResp(t, resp)
			}

			// Verify all expected calls were made
			if len(env.calls.DeleteTgChatSubgraphInvitesByChatID) > 0 {
				t.Logf("DeleteTgChatSubgraphInvitesByChatID called %d times", len(env.calls.DeleteTgChatSubgraphInvitesByChatID))
			}
			if len(env.calls.InsertTgChatSubgraphInvite) > 0 {
				t.Logf("InsertTgChatSubgraphInvite called %d times", len(env.calls.InsertTgChatSubgraphInvite))
			}
		})
	}
}

// TestResolve_Integration tests the complete flow with more complex scenarios.
func TestResolve_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("should handle duplicate subgraph IDs gracefully", func(t *testing.T) {
		env := &EnvMock{
			CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
				return &usertoken.Data{ID: 1}, nil
			},
			DeleteTgChatSubgraphInvitesByChatIDFunc: func(ctx context.Context, chatID int64) error {
				return nil
			},
			InsertTgChatSubgraphInviteFunc: func(ctx context.Context, arg db.InsertTgChatSubgraphInviteParams) (db.TgBotChatSubgraphInvite, error) {
				// Should still insert duplicates in the input
				return db.TgBotChatSubgraphInvite{
					ChatID:     arg.ChatID,
					SubgraphID: arg.SubgraphID,
					CreatedBy:  arg.CreatedBy,
				}, nil
			},
		}

		input := Input{
			ChatID:      100,
			SubgraphIds: []int64{1, 2, 1, 3, 2}, // Contains duplicates
		}

		resp, err := Resolve(ctx, env, input)
		require.NoError(t, err)

		payload, ok := resp.(*model.SetTgChatSubgraphInvitesPayload)
		require.True(t, ok)
		require.True(t, payload.Success)

		// Should have called insert 5 times (including duplicates)
		require.Len(t, env.calls.InsertTgChatSubgraphInvite, 5)
	})

	t.Run("should handle large number of subgraphs", func(t *testing.T) {
		// Create a large list of subgraph IDs
		var subgraphIds []int64
		for i := int64(1); i <= 100; i++ {
			subgraphIds = append(subgraphIds, i)
		}

		insertedCount := 0
		env := &EnvMock{
			CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
				return &usertoken.Data{ID: 1}, nil
			},
			DeleteTgChatSubgraphInvitesByChatIDFunc: func(ctx context.Context, chatID int64) error {
				return nil
			},
			InsertTgChatSubgraphInviteFunc: func(ctx context.Context, arg db.InsertTgChatSubgraphInviteParams) (db.TgBotChatSubgraphInvite, error) {
				insertedCount++
				return db.TgBotChatSubgraphInvite{
					ChatID:     arg.ChatID,
					SubgraphID: arg.SubgraphID,
					CreatedBy:  arg.CreatedBy,
				}, nil
			},
		}

		input := Input{
			ChatID:      999,
			SubgraphIds: subgraphIds,
		}

		resp, err := Resolve(ctx, env, input)
		require.NoError(t, err)

		payload, ok := resp.(*model.SetTgChatSubgraphInvitesPayload)
		require.True(t, ok)
		require.True(t, payload.Success)
		require.Equal(t, 100, insertedCount)
	})
}

// TestValidateRequest tests the validation function separately.
func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name      string
		input     *Input
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid input",
			input: &Input{
				ChatID:      123,
				SubgraphIds: []int64{1, 2, 3},
			},
			wantError: false,
		},
		{
			name: "valid input with empty array",
			input: &Input{
				ChatID:      123,
				SubgraphIds: []int64{},
			},
			wantError: false,
		},
		{
			name: "invalid - missing chat ID",
			input: &Input{
				ChatID:      0,
				SubgraphIds: []int64{1},
			},
			wantError: true,
			errorMsg:  "chatId",
		},
		{
			name: "invalid - nil subgraph IDs",
			input: &Input{
				ChatID:      123,
				SubgraphIds: nil,
			},
			wantError: true,
			errorMsg:  "subgraphIds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequest(tt.input)

			if tt.wantError {
				require.NotNil(t, err)
				// Check either Message or ByFields contains the error
				hasError := false
				if err.Message != "" && strings.Contains(err.Message, tt.errorMsg) {
					hasError = true
				}
				for _, field := range err.ByFields {
					if strings.Contains(field.Value, tt.errorMsg) || field.Name == tt.errorMsg {
						hasError = true
						break
					}
				}
				require.True(t, hasError, "Expected error about %s, got: %+v", tt.errorMsg, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
