package createoutboundfederationsecret_test

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"trip2g/internal/case/admin/createoutboundfederationsecret"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func validSecret() string {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(i)
	}
	return hex.EncodeToString(b)
}

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.CreateOutboundFederationSecretInput
		mockFunc func() *envMock
		wantErr  bool
		wantType string
		wantMsg  string
	}{
		{
			name: "success",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       "peer-kid",
				SecretHex: validSecret(),
				KbURL:     "https://peer.example.com",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					require.Len(t, plaintext, 32)
					return []byte("encrypted"), nil
				}
				mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
					require.Equal(t, "peer-kid", arg.Kid)
					require.NotNil(t, arg.KbUrl)
					require.Equal(t, "https://peer.example.com", *arg.KbUrl)
					return db.FederationSecret{ID: 10, Kid: arg.Kid}, nil
				}
				return mock
			},
			wantType: "payload",
		},
		{
			name: "validation error - empty kid",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       "",
				SecretHex: validSecret(),
				KbURL:     "https://peer.example.com",
			},
			mockFunc: func() *envMock { return &envMock{} },
			wantType: "error",
		},
		{
			name: "validation error - empty secretHex",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       "peer-kid",
				SecretHex: "",
				KbURL:     "https://peer.example.com",
			},
			mockFunc: func() *envMock { return &envMock{} },
			wantType: "error",
		},
		{
			name: "validation error - empty kbURL",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       "peer-kid",
				SecretHex: validSecret(),
				KbURL:     "",
			},
			mockFunc: func() *envMock { return &envMock{} },
			wantType: "error",
		},
		{
			name: "invalid hex - wrong length",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       "peer-kid",
				SecretHex: "aabbccdd",
				KbURL:     "https://peer.example.com",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				return mock
			},
			wantType: "error",
			wantMsg:  "secretHex must be exactly 32 bytes (64 hex characters)",
		},
		{
			name: "invalid hex - not hex",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       "peer-kid",
				SecretHex: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
				KbURL:     "https://peer.example.com",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				return mock
			},
			wantType: "error",
			wantMsg:  "secretHex must be exactly 32 bytes (64 hex characters)",
		},
		{
			name: "auth error",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       "peer-kid",
				SecretHex: validSecret(),
				KbURL:     "https://peer.example.com",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				}
				return mock
			},
			wantErr: true,
		},
		{
			name: "unique violation",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       "peer-kid",
				SecretHex: validSecret(),
				KbURL:     "https://peer.example.com",
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
					return db.FederationSecret{}, errors.New("UNIQUE constraint failed: federation_secrets.kid")
				}
				return mock
			},
			wantType: "error",
			wantMsg:  "Federation secret with this kid already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := tt.mockFunc()
			got, err := createoutboundfederationsecret.Resolve(context.Background(), env, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			switch tt.wantType {
			case "payload":
				p, ok := got.(*model.CreateOutboundFederationSecretPayload)
				require.True(t, ok, "expected CreateOutboundFederationSecretPayload")
				require.NotZero(t, p.ID)
				require.Equal(t, tt.input.Kid, p.Kid)
			case "error":
				ep, ok := got.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload")
				if tt.wantMsg != "" {
					require.Equal(t, tt.wantMsg, ep.Message)
				}
			}
		})
	}
}
