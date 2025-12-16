package updatetelegramaccount_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"trip2g/internal/case/admin/updatetelegramaccount"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

// Note: sql import is kept for sql.ErrNoRows usage

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updatetelegramaccount_test . Env

type Env = updatetelegramaccount.Env

func ptr[T any](v T) *T {
	return &v
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		env         Env
		input       model.AdminUpdateTelegramAccountInput
		wantErr     bool
		wantPayload bool
		errContains string
	}{
		{
			name: "successful update with display name",
			env: &EnvMock{
				GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
					return db.TelegramAccount{ID: id, Phone: "+1234567890", DisplayName: "Updated Name"}, nil
				},
				UpdateTelegramAccountFunc: func(ctx context.Context, arg db.UpdateTelegramAccountParams) error {
					require.Equal(t, int64(1), arg.ID)
					require.NotNil(t, arg.DisplayName)
					require.Equal(t, "New Name", *arg.DisplayName)
					return nil
				},
			},
			input: model.AdminUpdateTelegramAccountInput{
				ID:          1,
				DisplayName: ptr("New Name"),
			},
			wantErr:     false,
			wantPayload: true,
		},
		{
			name: "successful update with enabled flag",
			env: &EnvMock{
				GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
					return db.TelegramAccount{ID: id, Phone: "+1234567890", Enabled: 1}, nil
				},
				UpdateTelegramAccountFunc: func(ctx context.Context, arg db.UpdateTelegramAccountParams) error {
					require.NotNil(t, arg.Enabled)
					require.Equal(t, int64(1), *arg.Enabled)
					return nil
				},
			},
			input: model.AdminUpdateTelegramAccountInput{
				ID:      1,
				Enabled: ptr(true),
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
			input: model.AdminUpdateTelegramAccountInput{
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
			input: model.AdminUpdateTelegramAccountInput{
				ID: 1,
			},
			wantErr:     true,
			wantPayload: false,
		},
		{
			name: "database error on update",
			env: &EnvMock{
				GetTelegramAccountByIDFunc: func(ctx context.Context, id int64) (db.TelegramAccount, error) {
					return db.TelegramAccount{ID: id}, nil
				},
				UpdateTelegramAccountFunc: func(ctx context.Context, arg db.UpdateTelegramAccountParams) error {
					return errors.New("update failed")
				},
			},
			input: model.AdminUpdateTelegramAccountInput{
				ID:          1,
				DisplayName: ptr("Test"),
			},
			wantErr:     true,
			wantPayload: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := updatetelegramaccount.Resolve(context.Background(), tt.env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantPayload {
				p, ok := payload.(*model.AdminUpdateTelegramAccountPayload)
				require.True(t, ok, "expected AdminUpdateTelegramAccountPayload")
				require.NotNil(t, p.Account)
			} else {
				errPayload, ok := payload.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload, got %T", payload)
				require.Contains(t, errPayload.Message, tt.errContains)
			}
		})
	}
}
