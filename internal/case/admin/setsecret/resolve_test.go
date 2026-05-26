package setsecret_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/setsecret"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		value    string
		mockFunc func() *envMock
		wantErr  bool
	}{
		{
			name:  "success",
			key:   "change_webhooks:1:auth_token",
			value: "Bearer secret123",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("encrypted"), nil
				}
				mock.UpsertSecretFunc = func(ctx context.Context, arg db.UpsertSecretParams) (db.Secret, error) {
					require.Equal(t, "change_webhooks:1:auth_token", arg.Key)
					require.Equal(t, []byte("encrypted"), arg.ValueCrypt)
					require.Equal(t, int64(1), arg.CreatedBy)
					return db.Secret{Key: arg.Key}, nil
				}
				return mock
			},
		},
		{
			name:  "auth error",
			key:   "k",
			value: "v",
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
			name:  "encrypt error",
			key:   "k",
			value: "v",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return nil, errors.New("encrypt fail")
				}
				return mock
			},
			wantErr: true,
		},
		{
			name:  "db error",
			key:   "k",
			value: "v",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.EncryptDataFunc = func(plaintext []byte) ([]byte, error) {
					return []byte("enc"), nil
				}
				mock.UpsertSecretFunc = func(ctx context.Context, arg db.UpsertSecretParams) (db.Secret, error) {
					return db.Secret{}, errors.New("db error")
				}
				return mock
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := setsecret.Resolve(context.Background(), tt.mockFunc(), tt.key, tt.value)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
