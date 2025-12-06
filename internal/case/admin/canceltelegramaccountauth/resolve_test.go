package canceltelegramaccountauth_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"trip2g/internal/case/admin/canceltelegramaccountauth"
	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg canceltelegramaccountauth_test . Env

type Env = canceltelegramaccountauth.Env

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		env         Env
		input       model.AdminCancelTelegramAccountAuthInput
		wantErr     bool
		wantPayload bool
		errContains string
	}{
		{
			name: "no pending auth returns error",
			env: &EnvMock{
				TelegramAccountCancelAuthFunc: func(phone string) error {
					return errors.New("no pending authentication")
				},
			},
			input: model.AdminCancelTelegramAccountAuthInput{
				Phone: "+1234567890",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "Failed to cancel auth",
		},
		{
			name: "successful cancel",
			env: &EnvMock{
				TelegramAccountCancelAuthFunc: func(phone string) error {
					return nil
				},
			},
			input: model.AdminCancelTelegramAccountAuthInput{
				Phone: "+1234567890",
			},
			wantErr:     false,
			wantPayload: true,
		},
		{
			name: "validation error - empty phone",
			env:  &EnvMock{},
			input: model.AdminCancelTelegramAccountAuthInput{
				Phone: "",
			},
			wantErr:     false,
			wantPayload: false,
			errContains: "cannot be blank",
		},
		{
			name: "trims whitespace from phone",
			env: &EnvMock{
				TelegramAccountCancelAuthFunc: func(phone string) error {
					require.Equal(t, "+1234567890", phone, "phone should be trimmed")
					return nil
				},
			},
			input: model.AdminCancelTelegramAccountAuthInput{
				Phone: "  +1234567890  ",
			},
			wantErr:     false,
			wantPayload: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := canceltelegramaccountauth.Resolve(context.Background(), tt.env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantPayload {
				p, ok := payload.(*model.AdminCancelTelegramAccountAuthPayload)
				require.True(t, ok, "expected AdminCancelTelegramAccountAuthPayload")
				require.True(t, p.Success)
			} else {
				errPayload, ok := payload.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload, got %T", payload)
				assertErrorContains(t, errPayload, tt.errContains)
			}
		})
	}
}

// assertErrorContains checks if the error payload contains the expected message
// either in Message field or in ByFields.
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
