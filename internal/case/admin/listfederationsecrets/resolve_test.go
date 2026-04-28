package listfederationsecrets_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/listfederationsecrets"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mockFunc func() *envMock
		wantErr  bool
		wantLen  int
	}{
		{
			name: "success - returns rows",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
					return []db.ListFederationSecretsRow{
						{ID: 1, Kid: "kid-1", CreatedAt: time.Now(), CreatedBy: 1, SubgraphCount: 2},
						{ID: 2, Kid: "kid-2", CreatedAt: time.Now(), CreatedBy: 1, SubgraphCount: 0},
					}, nil
				}
				return mock
			},
			wantLen: 2,
		},
		{
			name: "success - empty list",
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
					return nil, nil
				}
				return mock
			},
			wantLen: 0,
		},
		{
			name: "auth error",
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
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
					return nil, errors.New("database error")
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
			got, err := listfederationsecrets.Resolve(context.Background(), env)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantLen)
		})
	}
}
