package deletesecret_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/deletesecret"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		mockFunc func() *envMock
		wantErr  bool
	}{
		{
			name: "success",
			key:  "change_webhooks:1:auth_token",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.DeleteSecretFunc = func(ctx context.Context, key string) error {
					require.Equal(t, "change_webhooks:1:auth_token", key)
					return nil
				}
				return mock
			},
		},
		{
			name: "auth error",
			key:  "k",
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
			name: "db error",
			key:  "k",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.DeleteSecretFunc = func(ctx context.Context, key string) error {
					return errors.New("db error")
				}
				return mock
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := deletesecret.Resolve(context.Background(), tt.mockFunc(), tt.key)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
