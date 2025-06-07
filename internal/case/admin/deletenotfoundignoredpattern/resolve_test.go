package deletenotfoundignoredpattern

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

//go:generate moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	tests := []struct {
		name   string
		input  model.DeleteNotFoundIgnoredPatternInput
		env    func() *EnvMock
		assert func(t *testing.T, result model.DeleteNotFoundIgnoredPatternOrErrorPayload, err error)
	}{
		{
			name: "success",
			input: model.DeleteNotFoundIgnoredPatternInput{
				ID: 1,
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				env.NotFoundIgnoredPatternByIDFunc = func(ctx context.Context, id int64) (db.NotFoundIgnoredPattern, error) {
					return db.NotFoundIgnoredPattern{
						ID:        1,
						Pattern:   "^/admin/.*",
						CreatedBy: 1,
					}, nil
				}
				env.DeleteNotFoundIgnoredPatternFunc = func(ctx context.Context, id int64) error {
					return nil
				}
				return env
			},
			assert: func(t *testing.T, result model.DeleteNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.NoError(t, err)
				payload, ok := result.(*model.DeleteNotFoundIgnoredPatternPayload)
				require.True(t, ok)
				require.Equal(t, int64(1), payload.DeletedID)
			},
		},
		{
			name: "pattern not found",
			input: model.DeleteNotFoundIgnoredPatternInput{
				ID: 999,
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				env.NotFoundIgnoredPatternByIDFunc = func(ctx context.Context, id int64) (db.NotFoundIgnoredPattern, error) {
					return db.NotFoundIgnoredPattern{}, sql.ErrNoRows
				}
				return env
			},
			assert: func(t *testing.T, result model.DeleteNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.NoError(t, err)
				errorPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Equal(t, "pattern not found", errorPayload.Message)
			},
		},
		{
			name: "admin auth failed",
			input: model.DeleteNotFoundIgnoredPatternInput{
				ID: 1,
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("admin auth failed")
				}
				return env
			},
			assert: func(t *testing.T, result model.DeleteNotFoundIgnoredPatternOrErrorPayload, err error) {
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
