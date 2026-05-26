package getsecret_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"trip2g/internal/case/admin/getsecret"
	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		mockFunc func() *envMock
		wantVal  string
		wantErr  bool
	}{
		{
			name: "success",
			key:  "change_webhooks:1:auth_token",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.GetSecretFunc = func(ctx context.Context, key string) (db.Secret, error) {
					require.Equal(t, "change_webhooks:1:auth_token", key)
					return db.Secret{Key: key, ValueCrypt: []byte("encrypted")}, nil
				}
				mock.DecryptDataFunc = func(ciphertext []byte) ([]byte, error) {
					return []byte("Bearer secret123"), nil
				}
				return mock
			},
			wantVal: "Bearer secret123",
		},
		{
			name: "not found",
			key:  "missing",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.GetSecretFunc = func(ctx context.Context, key string) (db.Secret, error) {
					return db.Secret{}, sql.ErrNoRows
				}
				return mock
			},
			wantErr: true,
		},
		{
			name: "decrypt error",
			key:  "k",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.GetSecretFunc = func(ctx context.Context, key string) (db.Secret, error) {
					return db.Secret{Key: key, ValueCrypt: []byte("bad")}, nil
				}
				mock.DecryptDataFunc = func(ciphertext []byte) ([]byte, error) {
					return nil, errors.New("decrypt fail")
				}
				return mock
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, err := getsecret.Resolve(context.Background(), tt.mockFunc(), tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantVal, val)
		})
	}
}
