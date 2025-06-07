package resetnotfoundpath

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

//go:generate moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		input  model.ResetNotFoundPathInput
		env    func() *EnvMock
		assert func(t *testing.T, result model.ResetNotFoundPathOrErrorPayload, err error)
	}{
		{
			name: "success",
			input: model.ResetNotFoundPathInput{
				ID: 1,
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				env.NotFoundPathByIDFunc = func(ctx context.Context, id int64) (db.NotFoundPath, error) {
					return db.NotFoundPath{
						ID:        1,
						Path:      "/test/path",
						TotalHits: 50,
						LastHitAt: now,
					}, nil
				}
				env.ResetNotFoundPathTotalHitsFunc = func(ctx context.Context, id int64) (db.NotFoundPath, error) {
					return db.NotFoundPath{
						ID:        1,
						Path:      "/test/path",
						TotalHits: 1,
						LastHitAt: now.Add(time.Second),
					}, nil
				}
				return env
			},
			assert: func(t *testing.T, result model.ResetNotFoundPathOrErrorPayload, err error) {
				require.NoError(t, err)
				payload, ok := result.(*model.ResetNotFoundPathPayload)
				require.True(t, ok)
				require.Equal(t, int64(1), payload.NotFoundPath.ID)
				require.Equal(t, "/test/path", payload.NotFoundPath.Path)
				require.Equal(t, int64(1), payload.NotFoundPath.TotalHits)
			},
		},
		{
			name: "path not found",
			input: model.ResetNotFoundPathInput{
				ID: 999,
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				env.NotFoundPathByIDFunc = func(ctx context.Context, id int64) (db.NotFoundPath, error) {
					return db.NotFoundPath{}, sql.ErrNoRows
				}
				return env
			},
			assert: func(t *testing.T, result model.ResetNotFoundPathOrErrorPayload, err error) {
				require.NoError(t, err)
				errorPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Equal(t, "not found path not found", errorPayload.Message)
			},
		},
		{
			name: "admin auth failed",
			input: model.ResetNotFoundPathInput{
				ID: 1,
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("admin auth failed")
				}
				return env
			},
			assert: func(t *testing.T, result model.ResetNotFoundPathOrErrorPayload, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get admin user token")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.env()
			result, err := Resolve(context.Background(), env, tt.input)
			tt.assert(t, result, err)

			if err == nil {
				t.Logf("result: %# v", pretty.Formatter(result))
			}
		})
	}
}
