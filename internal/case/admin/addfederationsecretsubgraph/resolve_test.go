package addfederationsecretsubgraph_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/addfederationsecretsubgraph"
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
		input    model.AddFederationSecretSubgraphInput
		mockFunc func() *envMock
		wantErr  bool
		wantType string
		wantMsg  string
	}{
		{
			name:  "success",
			input: model.AddFederationSecretSubgraphInput{Kid: "my-kid", SubgraphID: 5},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.InsertFederationSecretSubgraphFunc = func(ctx context.Context, arg db.InsertFederationSecretSubgraphParams) error {
					require.Equal(t, "my-kid", arg.Kid)
					require.Equal(t, int64(5), arg.SubgraphID)
					require.Equal(t, int64(1), arg.CreatedBy)
					return nil
				}
				return mock
			},
			wantType: "payload",
		},
		{
			name:  "validation error - empty kid",
			input: model.AddFederationSecretSubgraphInput{Kid: "", SubgraphID: 5},
			mockFunc: func() *envMock {
				return &envMock{}
			},
			wantType: "error",
		},
		{
			name:  "auth error",
			input: model.AddFederationSecretSubgraphInput{Kid: "my-kid", SubgraphID: 5},
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
			name:  "unique violation",
			input: model.AddFederationSecretSubgraphInput{Kid: "my-kid", SubgraphID: 5},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.InsertFederationSecretSubgraphFunc = func(ctx context.Context, arg db.InsertFederationSecretSubgraphParams) error {
					return errors.New("UNIQUE constraint failed: federation_secret_subgraphs.kid, federation_secret_subgraphs.subgraph_id")
				}
				return mock
			},
			wantType: "error",
			wantMsg:  "This subgraph is already assigned to this federation secret",
		},
		{
			name:  "database error",
			input: model.AddFederationSecretSubgraphInput{Kid: "my-kid", SubgraphID: 5},
			mockFunc: func() *envMock {
				mock := &envMock{}
				mock.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				mock.InsertFederationSecretSubgraphFunc = func(ctx context.Context, arg db.InsertFederationSecretSubgraphParams) error {
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
			got, err := addfederationsecretsubgraph.Resolve(context.Background(), env, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			switch tt.wantType {
			case "payload":
				p, ok := got.(*model.AddFederationSecretSubgraphPayload)
				require.True(t, ok, "expected AddFederationSecretSubgraphPayload")
				require.True(t, p.Success)
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
