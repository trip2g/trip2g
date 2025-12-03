package deletetelegramaccount_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"trip2g/internal/case/admin/deletetelegramaccount"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg deletetelegramaccount_test . Env

type Env = deletetelegramaccount.Env

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		env         Env
		input       model.AdminDeleteTelegramAccountInput
		wantErr     bool
		wantPayload bool
		errContains string
	}{
		{
			name: "successful delete",
			env: &EnvMock{
				GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
					return db.TelegramAccount{ID: id, Phone: "+1234567890"}, nil
				},
				DeleteTelegramAccountFunc: func(ctx context.Context, id int64) error {
					return nil
				},
			},
			input: model.AdminDeleteTelegramAccountInput{
				ID: 1,
			},
			wantErr:     false,
			wantPayload: true,
		},
		{
			name: "account not found",
			env: &EnvMock{
				GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
					return db.TelegramAccount{}, sql.ErrNoRows
				},
			},
			input: model.AdminDeleteTelegramAccountInput{
				ID: 999,
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "Account not found",
		},
		{
			name: "database error on get",
			env: &EnvMock{
				GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
					return db.TelegramAccount{}, errors.New("database error")
				},
			},
			input: model.AdminDeleteTelegramAccountInput{
				ID: 1,
			},
			wantErr:     true,
			wantPayload: false,
		},
		{
			name: "database error on delete",
			env: &EnvMock{
				GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
					return db.TelegramAccount{ID: id}, nil
				},
				DeleteTelegramAccountFunc: func(ctx context.Context, id int64) error {
					return errors.New("delete failed")
				},
			},
			input: model.AdminDeleteTelegramAccountInput{
				ID: 1,
			},
			wantErr:     true,
			wantPayload: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := deletetelegramaccount.Resolve(context.Background(), tt.env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantPayload {
				p, ok := payload.(*model.AdminDeleteTelegramAccountPayload)
				require.True(t, ok, "expected AdminDeleteTelegramAccountPayload")
				require.True(t, p.Success)
			} else {
				errPayload, ok := payload.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload, got %T", payload)
				require.Contains(t, errPayload.Message, tt.errContains)
			}
		})
	}
}
