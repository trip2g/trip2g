package setboostytiersubgraphs

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	ctx := context.Background()

	mockTier := db.BoostyTier{
		ID:       1,
		Name:     "Test Tier",
		BoostyID: 123,
	}

	tests := []struct {
		name       string
		input      Input
		setupMock  func(env *EnvMock)
		wantErr    bool
		errMsg     string
		checkError bool
	}{
		{
			name: "successful update with multiple subgraphs",
			input: Input{
				TierID:      1,
				SubgraphIds: []int64{10, 20, 30},
			},
			setupMock: func(env *EnvMock) {
				env.DeleteBoostyTierSubgraphsByTierIDFunc = func(ctx context.Context, tierID int64) error {
					require.Equal(t, int64(1), tierID)
					return nil
				}
				callCount := 0
				env.InsertBoostyTierSubgraphFunc = func(ctx context.Context, arg db.InsertBoostyTierSubgraphParams) error {
					require.Equal(t, int64(1), arg.TierID)
					require.Equal(t, int64(1), arg.CreatedBy)
					expectedSubgraphIDs := []int64{10, 20, 30}
					require.Equal(t, expectedSubgraphIDs[callCount], arg.SubgraphID)
					callCount++
					return nil
				}
				env.BoostyTierByIDFunc = func(ctx context.Context, id int64) (db.BoostyTier, error) {
					require.Equal(t, int64(1), id)
					return mockTier, nil
				}
			},
			wantErr: false,
		},
		{
			name: "successful clear all subgraphs",
			input: Input{
				TierID:      1,
				SubgraphIds: []int64{},
			},
			setupMock: func(env *EnvMock) {
				env.DeleteBoostyTierSubgraphsByTierIDFunc = func(ctx context.Context, tierID int64) error {
					return nil
				}
				// InsertBoostyTierSubgraph should not be called
				env.InsertBoostyTierSubgraphFunc = func(ctx context.Context, arg db.InsertBoostyTierSubgraphParams) error {
					t.Fatal("InsertBoostyTierSubgraph should not be called when clearing all subgraphs")
					return nil
				}
				env.BoostyTierByIDFunc = func(ctx context.Context, id int64) (db.BoostyTier, error) {
					return mockTier, nil
				}
			},
			wantErr: false,
		},
		{
			name: "invalid tier ID",
			input: Input{
				TierID:      0,
				SubgraphIds: []int64{10},
			},
			setupMock: func(env *EnvMock) {
				// Nothing should be called due to validation failure
			},
			wantErr:    false, // Validation errors return ErrorPayload, not error
			checkError: true,
		},
		{
			name: "nil subgraph IDs",
			input: Input{
				TierID:      1,
				SubgraphIds: nil,
			},
			setupMock: func(env *EnvMock) {
				// Nothing should be called due to validation failure
			},
			wantErr:    false, // Validation errors return ErrorPayload, not error
			checkError: true,
		},
		{
			name: "delete fails",
			input: Input{
				TierID:      1,
				SubgraphIds: []int64{10},
			},
			setupMock: func(env *EnvMock) {
				env.DeleteBoostyTierSubgraphsByTierIDFunc = func(ctx context.Context, tierID int64) error {
					return errors.New("database error")
				}
			},
			wantErr: true,
			errMsg:  "failed to delete existing tier subgraphs",
		},
		{
			name: "insert fails",
			input: Input{
				TierID:      1,
				SubgraphIds: []int64{10, 20},
			},
			setupMock: func(env *EnvMock) {
				env.DeleteBoostyTierSubgraphsByTierIDFunc = func(ctx context.Context, tierID int64) error {
					return nil
				}
				env.InsertBoostyTierSubgraphFunc = func(ctx context.Context, arg db.InsertBoostyTierSubgraphParams) error {
					if arg.SubgraphID == 10 {
						return nil // First one succeeds
					}
					return errors.New("insert failed")
				}
			},
			wantErr: true,
			errMsg:  "failed to insert tier subgraph 20",
		},
		{
			name: "fetch tier fails",
			input: Input{
				TierID:      1,
				SubgraphIds: []int64{10},
			},
			setupMock: func(env *EnvMock) {
				env.DeleteBoostyTierSubgraphsByTierIDFunc = func(ctx context.Context, tierID int64) error {
					return nil
				}
				env.InsertBoostyTierSubgraphFunc = func(ctx context.Context, arg db.InsertBoostyTierSubgraphParams) error {
					return nil
				}
				env.BoostyTierByIDFunc = func(ctx context.Context, id int64) (db.BoostyTier, error) {
					return db.BoostyTier{}, errors.New("tier not found")
				}
			},
			wantErr: true,
			errMsg:  "failed to fetch updated tier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setupMock(env)

			result, err := Resolve(ctx, env, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				if tt.checkError {
					// Check that we got an ErrorPayload
					_, ok := result.(*model.ErrorPayload)
					require.True(t, ok, "expected ErrorPayload for validation error")
				} else {
					// Check that we got a success payload
					payload, ok := result.(*model.SetBoostyTierSubgraphsPayload)
					require.True(t, ok, "expected SetBoostyTierSubgraphsPayload")
					require.True(t, payload.Success)
					require.NotNil(t, payload.Tier)
				}
			}
		})
	}
}
