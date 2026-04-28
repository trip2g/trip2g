package removefederationsecretsubgraph_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/removefederationsecretsubgraph"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    model.RemoveFederationSecretSubgraphInput
		mockFunc func() *envMock
		wantErr  bool
		wantType string
	}{
		{
			name:  "success",
			input: model.RemoveFederationSecretSubgraphInput{Kid: "my-kid", SubgraphID: 5},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.DeleteFederationSecretSubgraphFunc = func(ctx context.Context, arg db.DeleteFederationSecretSubgraphParams) error {
					require.Equal(t, "my-kid", arg.Kid)
					require.Equal(t, int64(5), arg.SubgraphID)
					return nil
				}
				return mock
			},
			wantType: "payload",
		},
		{
			name:  "validation error - empty kid",
			input: model.RemoveFederationSecretSubgraphInput{Kid: "", SubgraphID: 5},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			wantType: "error",
		},
		{
			name:  "auth error",
			input: model.RemoveFederationSecretSubgraphInput{Kid: "my-kid", SubgraphID: 5},
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
			name:  "database error",
			input: model.RemoveFederationSecretSubgraphInput{Kid: "my-kid", SubgraphID: 5},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.DeleteFederationSecretSubgraphFunc = func(ctx context.Context, arg db.DeleteFederationSecretSubgraphParams) error {
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
			got, err := removefederationsecretsubgraph.Resolve(context.Background(), env, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			switch tt.wantType {
			case "payload":
				p, ok := got.(*model.RemoveFederationSecretSubgraphPayload)
				require.True(t, ok, "expected RemoveFederationSecretSubgraphPayload")
				require.True(t, p.Success)
			case "error":
				_, ok := got.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload")
			}
		})
	}
}
