package starttelegramaccountauth_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"trip2g/internal/case/admin/starttelegramaccountauth"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg starttelegramaccountauth_test . Env

type Env = starttelegramaccountauth.Env

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		env         Env
		input       model.AdminStartTelegramAccountAuthInput
		wantErr     bool
		wantPayload bool
		errContains string
	}{
		{
			name: "phone already exists",
			env: &EnvMock{
				GetTelegramAccountByPhoneFunc: func(ctx context.Context, phone string) (db.TelegramAccount, error) {
					return db.TelegramAccount{Phone: phone}, nil
				},
			},
			input: model.AdminStartTelegramAccountAuthInput{
				Phone:   "+1234567890",
				APIID:   12345,
				APIHash: "abc123",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "Account with this phone already exists",
		},
		{
			name: "trims whitespace from phone",
			env: &EnvMock{
				GetTelegramAccountByPhoneFunc: func(ctx context.Context, phone string) (db.TelegramAccount, error) {
					require.Equal(t, "+1234567890", phone, "phone should be trimmed")
					return db.TelegramAccount{}, sql.ErrNoRows
				},
				TelegramAccountStartAuthFunc: func(ctx context.Context, phone string, apiID int, apiHash string) (*appmodel.TelegramStartAuthResult, error) {
					require.Equal(t, "+1234567890", phone, "phone should be trimmed")
					require.Equal(t, "abc123", apiHash, "apiHash should be trimmed")
					return nil, errors.New("some error")
				},
			},
			input: model.AdminStartTelegramAccountAuthInput{
				Phone:   "  +1234567890  ",
				APIID:   12345,
				APIHash: "  abc123  ",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "Failed to start auth",
		},
		{
			name: "successful auth start",
			env: &EnvMock{
				GetTelegramAccountByPhoneFunc: func(ctx context.Context, phone string) (db.TelegramAccount, error) {
					return db.TelegramAccount{}, sql.ErrNoRows
				},
				TelegramAccountStartAuthFunc: func(ctx context.Context, phone string, apiID int, apiHash string) (*appmodel.TelegramStartAuthResult, error) {
					return &appmodel.TelegramStartAuthResult{
						Phone: phone,
						State: appmodel.TelegramAuthStateWaitingForCode,
					}, nil
				},
			},
			input: model.AdminStartTelegramAccountAuthInput{
				Phone:   "+1234567890",
				APIID:   12345,
				APIHash: "abc123",
			},
			wantErr:     false,
			wantPayload: true,
		},
		{
			name: "validation error - empty phone",
			env:  &EnvMock{},
			input: model.AdminStartTelegramAccountAuthInput{
				Phone:   "",
				APIID:   12345,
				APIHash: "abc123",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "cannot be blank",
		},
		{
			name: "validation error - empty api hash",
			env:  &EnvMock{},
			input: model.AdminStartTelegramAccountAuthInput{
				Phone:   "+1234567890",
				APIID:   12345,
				APIHash: "",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "cannot be blank",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := starttelegramaccountauth.Resolve(context.Background(), tt.env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantPayload {
				_, ok := payload.(*model.AdminStartTelegramAccountAuthPayload)
				require.True(t, ok, "expected AdminStartTelegramAccountAuthPayload")
			} else {
				errPayload, ok := payload.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload, got %T", payload)
				assertErrorContains(t, errPayload, tt.errContains)
			}
		})
	}
}

// assertErrorContains checks if the error payload contains the expected message
// either in Message field or in ByFields
func assertErrorContains(t *testing.T, payload *model.ErrorPayload, expected string) {
	t.Helper()

	if payload.Message != "" {
		require.Contains(t, payload.Message, expected)
		return
	}

	// Check ByFields for ozzo validation errors
	for _, field := range payload.ByFields {
		if field.Value == expected || strings.Contains(field.Value, expected) {
			return
		}
	}

	t.Errorf("expected error payload to contain %q, got Message=%q, ByFields=%v", expected, payload.Message, payload.ByFields)
}
