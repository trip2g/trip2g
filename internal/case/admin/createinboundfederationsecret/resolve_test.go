package createinboundfederationsecret_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/createinboundfederationsecret"
	"trip2g/internal/db"
	"trip2g/internal/federationkey"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.CreateInboundFederationSecretInput
		mockFunc func() *envMock
		wantErr  bool
		wantType string // "payload" or "error"
		wantMsg  string
	}{
		{
			name:  "success - generates secret and returns hex",
			input: model.CreateInboundFederationSecretInput{Kid: "my-key-id"},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.PublicURLFunc = func() string { return "https://base.example" }
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					require.Len(t, plaintext, 32, "secret must be 32 bytes")
					return []byte("encrypted"), nil
				}
				mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
					require.Equal(t, "my-key-id", arg.Kid)
					require.Nil(t, arg.KbUrl)
					require.Equal(t, int64(1), arg.CreatedBy)
					return db.FederationSecret{ID: 42, Kid: arg.Kid}, nil
				}
				return mock
			},
			wantType: "payload",
		},
		{
			name:  "success - with description",
			input: model.CreateInboundFederationSecretInput{Kid: "my-key-id", Description: strPtr("test desc")},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.PublicURLFunc = func() string { return "https://base.example" }
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
					require.NotNil(t, arg.Description)
					require.Equal(t, "test desc", *arg.Description)
					return db.FederationSecret{ID: 43, Kid: arg.Kid}, nil
				}
				return mock
			},
			wantType: "payload",
		},
		{
			name:  "validation error - empty kid",
			input: model.CreateInboundFederationSecretInput{Kid: ""},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			wantType: "error",
		},
		{
			name:  "auth error",
			input: model.CreateInboundFederationSecretInput{Kid: "my-key-id"},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.PublicURLFunc = func() string { return "https://base.example" }
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				}
				return mock
			},
			wantErr: true,
		},
		{
			name:  "encryption error",
			input: model.CreateInboundFederationSecretInput{Kid: "my-key-id"},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.PublicURLFunc = func() string { return "https://base.example" }
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return nil, errors.New("encryption failed")
				}
				return mock
			},
			wantErr: true,
		},
		{
			name:  "unique violation",
			input: model.CreateInboundFederationSecretInput{Kid: "duplicate-kid"},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.PublicURLFunc = func() string { return "https://base.example" }
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
		{
			name: "success - caller-supplied secretHex used verbatim",
			input: model.CreateInboundFederationSecretInput{
				Kid:       "my-key-id",
				SecretHex: strPtr("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.PublicURLFunc = func() string { return "https://base.example" }
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					require.Len(t, plaintext, 32)
					require.Equal(t, byte(0x00), plaintext[0])
					require.Equal(t, byte(0xff), plaintext[31])
					return []byte("encrypted"), nil
				}
				mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
					return db.FederationSecret{ID: 50, Kid: arg.Kid}, nil
				}
				return mock
			},
			wantType: "payload",
		},
		{
			name: "validation error - secretHex wrong length",
			input: model.CreateInboundFederationSecretInput{
				Kid:       "my-key-id",
				SecretHex: strPtr("deadbeef"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.PublicURLFunc = func() string { return "https://base.example" }
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				return mock
			},
			wantType: "error",
		},
		{
			name: "validation error - secretHex not hex",
			input: model.CreateInboundFederationSecretInput{
				Kid:       "my-key-id",
				SecretHex: strPtr("zzzz"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.PublicURLFunc = func() string { return "https://base.example" }
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				return mock
			},
			wantType: "error",
		},
		{
			name:  "database error",
			input: model.CreateInboundFederationSecretInput{Kid: "my-key-id"},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.PublicURLFunc = func() string { return "https://base.example" }
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
					return db.FederationSecret{}, errors.New("database error")
				}
				return mock
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := tt.mockFunc()
			got, err := createinboundfederationsecret.Resolve(context.Background(), env, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			switch tt.wantType {
			case "payload":
				p, ok := got.(*model.CreateInboundFederationSecretPayload)
				require.True(t, ok, "expected CreateInboundFederationSecretPayload")
				require.NotZero(t, p.ID)
				require.Equal(t, tt.input.Kid, p.Kid)
				require.Len(t, p.SecretHex, 64, "hex-encoded 32 bytes = 64 chars")
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

func strPtr(s string) *string { return &s }

// The key is what the other operator pastes, so what the issuing side named has
// to survive the packing. Asserted by decoding rather than by reading the
// payload's other fields: a kb_id that reaches the admin page but not the key is
// a slug nobody downstream ever sees.
func TestResolvePacksWhatTheIssuerNamed(t *testing.T) {
	t.Parallel()

	input := model.CreateInboundFederationSecretInput{
		Kid:   "alice-2026",
		KbID:  strPtr("bob-team"),
		About: strPtr("Bob's work-status updates"),
	}

	got, err := createinboundfederationsecret.Resolve(context.Background(), packingEnv(t), input)
	require.NoError(t, err)

	payload, ok := got.(*model.CreateInboundFederationSecretPayload)
	require.True(t, ok)

	decoded, err := federationkey.Decode(payload.Key)
	require.NoError(t, err)
	require.Equal(t, "alice-2026", decoded.KID)
	require.Equal(t, "https://base.example/_system/mcp", decoded.KBURL)
	require.Equal(t, payload.SecretHex, decoded.SecretHex)
	require.Equal(t, "bob-team", decoded.KBID)
	require.Equal(t, "Bob's work-status updates", decoded.About)
}

// Naming the slug is optional, and the fallback is the hostname because that is
// also what trip2g uses when a KB-note names no mcp_federation_kb_id — the
// suggestion and the default agree, so an operator who ignores the field gets
// the same base id either way.
func TestResolveFallsBackToTheHostname(t *testing.T) {
	t.Parallel()

	input := model.CreateInboundFederationSecretInput{Kid: "alice-2026"}

	got, err := createinboundfederationsecret.Resolve(context.Background(), packingEnv(t), input)
	require.NoError(t, err)

	payload, ok := got.(*model.CreateInboundFederationSecretPayload)
	require.True(t, ok)

	decoded, err := federationkey.Decode(payload.Key)
	require.NoError(t, err)
	require.Equal(t, "base.example", decoded.KBID)
	require.Empty(t, decoded.About)
}

func packingEnv(t *testing.T) *envMock {
	t.Helper()

	mock := &envMock{}
	mock.PublicURLFunc = func() string { return "https://base.example" }
	mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
		return &usertoken.Data{ID: 1}, nil
	}
	mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
		return []byte("encrypted"), nil
	}
	mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
		return db.FederationSecret{ID: 60, Kid: arg.Kid}, nil
	}

	return mock
}
