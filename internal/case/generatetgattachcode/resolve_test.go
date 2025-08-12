//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

package generatetgattachcode

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name     string
		input    model.GenerateTgAttachCodeInput
		setupEnv func(*EnvMock)
		wantErr  bool
		validate func(*testing.T, model.GenerateTgAttachCodeOrErrorPayload)
	}{
		{
			name: "successful code generation",
			input: model.GenerateTgAttachCodeInput{
				BotID: 123,
			},
			setupEnv: func(env *EnvMock) {
				env.CurrentUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 456}, nil
				}
				env.TgBotFunc = func(ctx context.Context, id int64) (db.TgBot, error) {
					require.Equal(t, int64(123), id)
					return db.TgBot{
						ID:   123,
						Name: "testbot",
					}, nil
				}
				env.DeleteTgAttachCodesByUserFunc = func(ctx context.Context, userID int64) error {
					require.Equal(t, int64(456), userID)
					return nil
				}
				env.GenerateTgAttachCodeFunc = func() string {
					return "testcode"
				}
				env.InsertTgAttachCodeFunc = func(ctx context.Context, arg db.InsertTgAttachCodeParams) error {
					require.Equal(t, int64(456), arg.UserID)
					require.Equal(t, int64(123), arg.BotID)
					require.Equal(t, "testcode", arg.Code)
					return nil
				}
				env.BotStartLinkFunc = func(botID int64, param string) (string, error) {
					require.Equal(t, int64(123), botID)
					require.Equal(t, "attach_testcode", param)
					return "https://t.me/testbot?start=attach_testcode", nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result model.GenerateTgAttachCodeOrErrorPayload) {
				payload, ok := result.(*model.GenerateTgAttachCodePayload)
				require.True(t, ok, "Expected GenerateTgAttachCodePayload")
				require.Equal(t, "testcode", payload.Code)
				require.Equal(t, "https://t.me/testbot?start=attach_testcode", payload.URL)
			},
		},
		{
			name: "bot not found",
			input: model.GenerateTgAttachCodeInput{
				BotID: 999,
			},
			setupEnv: func(env *EnvMock) {
				env.CurrentUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 456}, nil
				}
				env.TgBotFunc = func(ctx context.Context, id int64) (db.TgBot, error) {
					return db.TgBot{}, sql.ErrNoRows
				}
				env.DeleteTgAttachCodesByUserFunc = func(ctx context.Context, userID int64) error {
					require.Equal(t, int64(456), userID)
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result model.GenerateTgAttachCodeOrErrorPayload) {
				errorPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok, "Expected ErrorPayload")
				require.Equal(t, "Bot not found", errorPayload.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setupEnv(env)

			result, err := Resolve(context.Background(), env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validate(t, result)
			}
		})
	}
}
