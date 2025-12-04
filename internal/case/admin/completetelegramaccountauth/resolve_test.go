package completetelegramaccountauth_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"trip2g/internal/case/admin/completetelegramaccountauth"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg completetelegramaccountauth_test . Env

type Env = completetelegramaccountauth.Env

func ptr[T any](v T) *T {
	return &v
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		env         Env
		input       model.AdminCompleteTelegramAccountAuthInput
		wantErr     bool
		wantPayload bool
		errContains string
	}{
		{
			name: "validation error - empty phone",
			env:  &EnvMock{},
			input: model.AdminCompleteTelegramAccountAuthInput{
				Phone: "",
				Code:  "12345",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "phone",
		},
		{
			name: "validation error - empty code",
			env:  &EnvMock{},
			input: model.AdminCompleteTelegramAccountAuthInput{
				Phone: "+1234567890",
				Code:  "",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "code",
		},
		{
			name: "admin token error",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
			},
			input: model.AdminCompleteTelegramAccountAuthInput{
				Phone: "+1234567890",
				Code:  "12345",
			},
			wantErr:     true,
			wantPayload: false,
		},
		{
			name: "no pending auth",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				TelegramAccountCompleteAuthFunc: func(ctx context.Context, phone, code, password string) (*appmodel.TelegramCompleteAuthResult, error) {
					return nil, errors.New("no pending authentication")
				},
				TelegramAccountGetPasswordHintFunc: func(phone string) string {
					return ""
				},
			},
			input: model.AdminCompleteTelegramAccountAuthInput{
				Phone: "+1234567890",
				Code:  "12345",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "Authentication failed",
		},
		{
			name: "2FA password required",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				TelegramAccountCompleteAuthFunc: func(ctx context.Context, phone, code, password string) (*appmodel.TelegramCompleteAuthResult, error) {
					return nil, errors.New("2FA password required")
				},
				TelegramAccountGetPasswordHintFunc: func(phone string) string {
					return "your pet name"
				},
			},
			input: model.AdminCompleteTelegramAccountAuthInput{
				Phone: "+1234567890",
				Code:  "12345",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "2FA password required. Hint: your pet name",
		},
		{
			name: "successful auth - new account",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				TelegramAccountCompleteAuthFunc: func(ctx context.Context, phone, code, password string) (*appmodel.TelegramCompleteAuthResult, error) {
					return &appmodel.TelegramCompleteAuthResult{
						SessionData: []byte("session"),
						DisplayName: "Test User",
						IsPremium:   true,
						APIID:       12345,
						APIHash:     "abc123",
					}, nil
				},
				GetTelegramAccountByPhoneFunc: func(ctx context.Context, phone string) (db.TelegramAccount, error) {
					return db.TelegramAccount{}, sql.ErrNoRows
				},
				InsertTelegramAccountFunc: func(ctx context.Context, arg db.InsertTelegramAccountParams) (db.TelegramAccount, error) {
					return db.TelegramAccount{
						ID:          1,
						Phone:       arg.Phone,
						DisplayName: arg.DisplayName,
						IsPremium:   arg.IsPremium,
					}, nil
				},
				TelegramAccountGetAppConfigFunc: func(ctx context.Context, accountID int64) (string, error) {
					return `{"caption_length_limit_default": 1024}`, nil
				},
				UpdateTelegramAccountAppConfigFunc: func(ctx context.Context, arg db.UpdateTelegramAccountAppConfigParams) error {
					return nil
				},
			},
			input: model.AdminCompleteTelegramAccountAuthInput{
				Phone: "+1234567890",
				Code:  "12345",
			},
			wantErr:     false,
			wantPayload: true,
		},
		{
			name: "successful auth - existing account (re-auth)",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				TelegramAccountCompleteAuthFunc: func(ctx context.Context, phone, code, password string) (*appmodel.TelegramCompleteAuthResult, error) {
					return &appmodel.TelegramCompleteAuthResult{
						SessionData: []byte("new_session"),
						DisplayName: "Test User",
						IsPremium:   true,
						APIID:       12345,
						APIHash:     "abc123",
					}, nil
				},
				GetTelegramAccountByPhoneFunc: func(ctx context.Context, phone string) (db.TelegramAccount, error) {
					return db.TelegramAccount{
						ID:          5,
						Phone:       phone,
						DisplayName: "Old User",
						Enabled:     0,
					}, nil
				},
				UpdateTelegramAccountFunc: func(ctx context.Context, arg db.UpdateTelegramAccountParams) error {
					require.Equal(t, int64(5), arg.ID)
					require.Equal(t, []byte("new_session"), arg.SessionData)
					require.Equal(t, int64(1), arg.Enabled.Int64)
					return nil
				},
				TelegramAccountGetAppConfigFunc: func(ctx context.Context, accountID int64) (string, error) {
					return `{"caption_length_limit_default": 1024}`, nil
				},
				UpdateTelegramAccountAppConfigFunc: func(ctx context.Context, arg db.UpdateTelegramAccountAppConfigParams) error {
					return nil
				},
			},
			input: model.AdminCompleteTelegramAccountAuthInput{
				Phone: "+1234567890",
				Code:  "12345",
			},
			wantErr:     false,
			wantPayload: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := completetelegramaccountauth.Resolve(context.Background(), tt.env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantPayload {
				p, ok := payload.(*model.AdminCompleteTelegramAccountAuthPayload)
				require.True(t, ok, "expected AdminCompleteTelegramAccountAuthPayload")
				require.NotNil(t, p.Account)
			} else {
				errPayload, ok := payload.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload, got %T", payload)
				assertErrorContains(t, errPayload, tt.errContains)
			}
		})
	}
}

// assertErrorContains checks if the error payload contains the expected message
// either in Message field or in ByFields (checks both field name and value)
func assertErrorContains(t *testing.T, payload *model.ErrorPayload, expected string) {
	t.Helper()

	if payload.Message != "" {
		require.Contains(t, payload.Message, expected)
		return
	}

	// Check ByFields for ozzo validation errors (check both name and value)
	for _, field := range payload.ByFields {
		if field.Name == expected || strings.Contains(field.Name, expected) ||
			field.Value == expected || strings.Contains(field.Value, expected) {
			return
		}
	}

	t.Errorf("expected error payload to contain %q, got Message=%q, ByFields=%v", expected, payload.Message, payload.ByFields)
}

func TestResolve_TrimsWhitespace(t *testing.T) {
	env := &EnvMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 1}, nil
		},
		TelegramAccountCompleteAuthFunc: func(ctx context.Context, phone, code, password string) (*appmodel.TelegramCompleteAuthResult, error) {
			require.Equal(t, "+1234567890", phone, "phone should be trimmed")
			require.Equal(t, "12345", code, "code should be trimmed")
			require.Equal(t, "password", password, "password should be trimmed")
			return nil, errors.New("no pending authentication")
		},
		TelegramAccountGetPasswordHintFunc: func(phone string) string {
			return ""
		},
	}

	input := model.AdminCompleteTelegramAccountAuthInput{
		Phone:    "  +1234567890  ",
		Code:     "  12345  ",
		Password: ptr("  password  "),
	}

	// This will fail because there's no pending auth, but it tests trimming
	payload, err := completetelegramaccountauth.Resolve(context.Background(), env, input)
	require.NoError(t, err)

	errPayload, ok := payload.(*model.ErrorPayload)
	require.True(t, ok)
	require.Contains(t, errPayload.Message, "Authentication failed")
}
