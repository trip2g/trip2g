package setpatreontiersubgraphs

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

func TestSetPatreonTierSubgraphs(t *testing.T) {
	tests := []struct {
		name        string
		input       Input
		setupMock   func(*EnvMock)
		expectError bool
		expectOk    bool
	}{
		{
			name: "success - set multiple subgraphs",
			input: Input{
				TierID:      1,
				SubgraphIds: []int64{2, 3, 4},
			},
			setupMock: func(env *EnvMock) {
				env.DeletePatreonTierSubgraphsByTierIDFunc = func(ctx context.Context, tierID int64) error {
					require.Equal(t, int64(1), tierID)
					return nil
				}

				callCount := 0
				env.InsertPatreonTierSubgraphFunc = func(ctx context.Context, arg db.InsertPatreonTierSubgraphParams) error {
					require.Equal(t, int64(1), arg.TierID)
					require.Equal(t, int64(1), arg.CreatedBy)

					switch callCount {
					case 0:
						require.Equal(t, int64(2), arg.SubgraphID)
					case 1:
						require.Equal(t, int64(3), arg.SubgraphID)
					case 2:
						require.Equal(t, int64(4), arg.SubgraphID)
					default:
						t.Fatal("too many insert calls")
					}
					callCount++
					return nil
				}

				env.PatreonTierByIDFunc = func(ctx context.Context, id int64) (db.PatreonTier, error) {
					require.Equal(t, int64(1), id)
					return db.PatreonTier{ID: 1, TierID: "tier_1", Title: "Test Tier"}, nil
				}
			},
			expectOk: true,
		},
		{
			name: "success - set empty subgraphs (clear all)",
			input: Input{
				TierID:      1,
				SubgraphIds: []int64{},
			},
			setupMock: func(env *EnvMock) {
				env.DeletePatreonTierSubgraphsByTierIDFunc = func(ctx context.Context, tierID int64) error {
					require.Equal(t, int64(1), tierID)
					return nil
				}
				// For empty array, no inserts should be called, but we still need to provide the function
				env.InsertPatreonTierSubgraphFunc = func(ctx context.Context, arg db.InsertPatreonTierSubgraphParams) error {
					return nil
				}

				env.PatreonTierByIDFunc = func(ctx context.Context, id int64) (db.PatreonTier, error) {
					require.Equal(t, int64(1), id)
					return db.PatreonTier{ID: 1, TierID: "tier_1", Title: "Test Tier"}, nil
				}
			},
			expectOk: true,
		},
		{
			name: "invalid tier id",
			input: Input{
				TierID:      0,
				SubgraphIds: []int64{2},
			},
			setupMock: func(env *EnvMock) {
				// No DB calls expected
			},
			expectError: true,
		},
		{
			name: "delete fails",
			input: Input{
				TierID:      1,
				SubgraphIds: []int64{2},
			},
			setupMock: func(env *EnvMock) {
				env.DeletePatreonTierSubgraphsByTierIDFunc = func(ctx context.Context, tierID int64) error {
					return errors.New("database error")
				}
			},
			expectError: false, // System errors return as error, not ErrorPayload
		},
		{
			name: "insert fails partway through",
			input: Input{
				TierID:      1,
				SubgraphIds: []int64{2, 3, 4},
			},
			setupMock: func(env *EnvMock) {
				env.DeletePatreonTierSubgraphsByTierIDFunc = func(ctx context.Context, tierID int64) error {
					return nil
				}

				callCount := 0
				env.InsertPatreonTierSubgraphFunc = func(ctx context.Context, arg db.InsertPatreonTierSubgraphParams) error {
					if callCount == 1 {
						// Fail on second insert
						return errors.New("insert failed")
					}
					callCount++
					return nil
				}
			},
			expectError: false, // System errors return as error, not ErrorPayload
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setupMock(env)

			result, err := Resolve(context.Background(), env, tt.input)

			switch {
			case tt.expectError:
				// Expect an ErrorPayload to be returned
				require.NoError(t, err)
				require.NotNil(t, result)
				errorPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload")
				// Check either Message or ByFields is populated
				require.True(t, errorPayload.Message != "" || len(errorPayload.ByFields) > 0, "expected error details")
			case tt.expectOk:
				require.NoError(t, err)
				require.NotNil(t, result)
				payload, ok := result.(*model.SetPatreonTierSubgraphsPayload)
				if !ok {
					t.Logf("result type: %T, result: %+v", result, result)
				}
				require.True(t, ok, "expected SetPatreonTierSubgraphsPayload")
				require.NotNil(t, payload.Tier)
				require.Equal(t, int64(1), payload.Tier.ID)
			default:
				// System error case
				require.Error(t, err)
				require.Nil(t, result)
			}
		})
	}
}
