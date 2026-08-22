package createoutboundfederationsecret_test

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"trip2g/internal/case/admin/createoutboundfederationsecret"
	"trip2g/internal/db"
	"trip2g/internal/federationkey"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
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
				Kid:       ptr("peer-kid"),
				SecretHex: ptr(validSecret()),
				KbURL:     ptr("https://peer.example.com"),
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
				Kid:       ptr(""),
				SecretHex: ptr(validSecret()),
				KbURL:     ptr("https://peer.example.com"),
			},
			mockFunc: func() *envMock { return &envMock{} },
			wantType: "error",
		},
		{
			name: "validation error - empty secretHex",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       ptr("peer-kid"),
				SecretHex: ptr(""),
				KbURL:     ptr("https://peer.example.com"),
			},
			mockFunc: func() *envMock { return &envMock{} },
			wantType: "error",
		},
		{
			name: "validation error - empty kbURL",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       ptr("peer-kid"),
				SecretHex: ptr(validSecret()),
				KbURL:     ptr(""),
			},
			mockFunc: func() *envMock { return &envMock{} },
			wantType: "error",
		},
		{
			name: "invalid hex - wrong length",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       ptr("peer-kid"),
				SecretHex: ptr("aabbccdd"),
				KbURL:     ptr("https://peer.example.com"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				return mock
			},
			wantType: "error",
			wantMsg:  "the secret must be exactly 32 bytes (64 hex characters)",
		},
		{
			name: "invalid hex - not hex",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       ptr("peer-kid"),
				SecretHex: ptr("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"),
				KbURL:     ptr("https://peer.example.com"),
			},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				return mock
			},
			wantType: "error",
			wantMsg:  "the secret must be exactly 32 bytes (64 hex characters)",
		},
		{
			name: "auth error",
			input: model.CreateOutboundFederationSecretInput{
				Kid:       ptr("peer-kid"),
				SecretHex: ptr(validSecret()),
				KbURL:     ptr("https://peer.example.com"),
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
				Kid:       ptr("peer-kid"),
				SecretHex: ptr(validSecret()),
				KbURL:     ptr("https://peer.example.com"),
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
			// This table is about validation and the insert. Rotation is a call
			// to a peer with its own tests below, and leaving it on here would
			// make every case need a peer to answer.
			if tt.input.Rotate == nil {
				tt.input.Rotate = ptr(false)
			}
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
				require.Equal(t, *tt.input.Kid, p.Kid)
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

func ptr[T any](value T) *T {
	return &value
}

// The default is rotation, and what it has to produce is a row holding a key the
// peer accepted rather than the one that was handed over: the handover travels
// through a chat, and leaving it live is the whole reason this exists.
func TestResolveRotatesByDefault(t *testing.T) {
	t.Parallel()

	handedOver, err := hex.DecodeString(validSecret())
	require.NoError(t, err)

	var proposed []byte
	mock := &envMock{}
	mock.FederationAllowsPlainHTTPFunc = func() bool { return false }
	mock.PublicURLFunc = func() string { return "https://hub.example" }
	mock.AuditLoggerFunc = func() logger.Logger { return &logger.DummyLogger{} }
	mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
		return &usertoken.Data{ID: 1}, nil
	}
	mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
		return append([]byte("enc:"), plaintext...), nil
	}
	mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
		require.Equal(t, append([]byte("enc:"), handedOver...), arg.SecretCrypt,
			"the row starts on the handed-over key, which the rotation below then moves off")
		return db.FederationSecret{ID: 10, Kid: arg.Kid}, nil
	}
	mock.RotateFederationSecretFunc = func(ctx context.Context, arg db.RotateFederationSecretParams) (int64, error) {
		require.Equal(t, int64(10), arg.ID)
		require.Equal(t, append([]byte("enc:"), proposed...), arg.NewSecretCrypt)
		return 1, nil
	}
	mock.FederationPeerClientFunc = func(peer appmodel.FederationPeer) appmodel.Federation {
		return &peerStub{
			rotate: func(params appmodel.MCPRotateSecretParams) {
				require.Equal(t, handedOver, peer.Secret, "the rotation is authorised by the key being retired")
				decoded, decodeErr := hex.DecodeString(params.SecretHex)
				require.NoError(t, decodeErr)
				require.Len(t, decoded, 32)
				require.NotEqual(t, handedOver, decoded)
				proposed = decoded
			},
		}
	}

	input := model.CreateOutboundFederationSecretInput{
		Kid:       ptr("peer-kid"),
		SecretHex: ptr(validSecret()),
		KbURL:     ptr("https://peer.example.com"),
	}

	payload, err := createoutboundfederationsecret.Resolve(context.Background(), mock, input)

	require.NoError(t, err)
	require.IsType(t, &model.CreateOutboundFederationSecretPayload{}, payload)
	require.NotEmpty(t, proposed, "the peer was never asked to take a new key")
}

// A peer that will not rotate leaves nothing behind. A row holding the
// handed-over key would be a working link resting on the credential the flag
// exists to retire, and it would look like success.
func TestResolveWritesNothingWhenThePeerRefuses(t *testing.T) {
	t.Parallel()

	mock := &envMock{}
	mock.FederationAllowsPlainHTTPFunc = func() bool { return false }
	mock.PublicURLFunc = func() string { return "https://hub.example" }
	mock.FederationPeerClientFunc = func(peer appmodel.FederationPeer) appmodel.Federation {
		return &peerStub{err: errors.New("connection refused")}
	}
	mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
		t.Fatal("a row was inserted for a peer that refused the rotation")
		return db.FederationSecret{}, nil
	}

	input := model.CreateOutboundFederationSecretInput{
		Kid:       ptr("peer-kid"),
		SecretHex: ptr(validSecret()),
		KbURL:     ptr("https://peer.example.com"),
	}

	payload, err := createoutboundfederationsecret.Resolve(context.Background(), mock, input)

	require.NoError(t, err)
	require.IsType(t, &model.ErrorPayload{}, payload)
}

// Rotation puts a fresh key on the wire, so it refuses an address that cannot
// carry one — unless the deployment has already said it federates over addresses
// that are not on the open internet.
func TestResolveRefusesPlainHTTP(t *testing.T) {
	t.Parallel()

	mock := &envMock{}
	mock.FederationAllowsPlainHTTPFunc = func() bool { return false }
	mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
		t.Fatal("a row was inserted for an address rotation refused")
		return db.FederationSecret{}, nil
	}

	input := model.CreateOutboundFederationSecretInput{
		Kid:       ptr("peer-kid"),
		SecretHex: ptr(validSecret()),
		KbURL:     ptr("http://peer.example.com"),
	}

	payload, err := createoutboundfederationsecret.Resolve(context.Background(), mock, input)

	require.NoError(t, err)
	errPayload, ok := payload.(*model.ErrorPayload)
	require.True(t, ok)
	require.Contains(t, errPayload.Message, "https")
}

// peerStub answers a rotation and nothing else; every other federated call
// panics, so a test that reaches one is a test whose subject moved.
type peerStub struct {
	appmodel.Federation
	rotate func(params appmodel.MCPRotateSecretParams)
	err    error
}

func (p *peerStub) RotateSecret(_ context.Context, params appmodel.MCPRotateSecretParams) (appmodel.FederationResult, error) {
	if p.err != nil {
		return appmodel.FederationResult{}, p.err
	}
	if p.rotate != nil {
		p.rotate(params)
	}
	return appmodel.FederationResult{}, nil
}

func (p *peerStub) Search(_ context.Context, _ appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
	return appmodel.FederationResult{IsError: true}, nil
}

// The packed key exists so an operator copies one value instead of three. It has
// to produce exactly the pairing the other side described.
func TestResolveAcceptsAPackedKey(t *testing.T) {
	t.Parallel()

	key, err := federationkey.Encode(federationkey.Handover{
		KID:       "peer-kid",
		KBURL:     "https://peer.example.com/_system/mcp",
		SecretHex: validSecret(),
	})
	require.NoError(t, err)

	mock := &envMock{}
	mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
		return &usertoken.Data{ID: 1}, nil
	}
	mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) { return plaintext, nil }
	mock.InsertFederationSecretFunc = func(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error) {
		require.Equal(t, "peer-kid", arg.Kid)
		require.Equal(t, "https://peer.example.com/_system/mcp", *arg.KbUrl)
		return db.FederationSecret{ID: 10, Kid: arg.Kid}, nil
	}

	payload, err := createoutboundfederationsecret.Resolve(context.Background(), mock, model.CreateOutboundFederationSecretInput{
		Key:    ptr(key),
		Rotate: ptr(false),
	})

	require.NoError(t, err)
	require.IsType(t, &model.CreateOutboundFederationSecretPayload{}, payload)
}

// Neither a key nor the three fields is a request that cannot be answered, and
// saying which is missing is the difference between one retry and three.
func TestResolveRefusesWithNeitherKeyNorFields(t *testing.T) {
	t.Parallel()

	payload, err := createoutboundfederationsecret.Resolve(context.Background(), &envMock{},
		model.CreateOutboundFederationSecretInput{Rotate: ptr(false)})

	require.NoError(t, err)
	errPayload, ok := payload.(*model.ErrorPayload)
	require.True(t, ok)
	require.Contains(t, errPayload.Message, "kid")
}
