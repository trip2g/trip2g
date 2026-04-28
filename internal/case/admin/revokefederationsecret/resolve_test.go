package revokefederationsecret_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/revokefederationsecret"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       int64
		mockFunc func() *envMock
		wantErr  bool
		wantType string
	}{
		{
			name: "success",
			id:   42,
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.RevokeFederationSecretFunc = func(ctx context.Context, id int64) error {
					require.Equal(t, int64(42), id)
					return nil
				}
				return mock
			},
			wantType: "payload",
		},
		{
			name: "auth error",
			id:   42,
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
			name: "database error",
			id:   42,
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.RevokeFederationSecretFunc = func(ctx context.Context, id int64) error {
					return errors.New("database error")
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
			got, err := revokefederationsecret.Resolve(context.Background(), env, tt.id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			p, ok := got.(*model.RevokeFederationSecretPayload)
			require.True(t, ok, "expected RevokeFederationSecretPayload")
			require.Equal(t, tt.id, p.RevokedID)
		})
	}
}
