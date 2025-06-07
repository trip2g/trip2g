package updatenotfoundignoredpattern

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
		input  model.UpdateNotFoundIgnoredPatternInput
		env    func() *EnvMock
		assert func(t *testing.T, result model.UpdateNotFoundIgnoredPatternOrErrorPayload, err error)
	}{
		{
			name: "success",
			input: model.UpdateNotFoundIgnoredPatternInput{
				ID:      1,
				Pattern: "^/admin/updated/.*",
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
				env.UpdateNotFoundIgnoredPatternFunc = func(ctx context.Context, arg db.UpdateNotFoundIgnoredPatternParams) (db.NotFoundIgnoredPattern, error) {
					return db.NotFoundIgnoredPattern{
						ID:        1,
						Pattern:   "^/admin/updated/.*",
						CreatedBy: 1,
					}, nil
				}
				env.RefreshNotFoundTrackerFunc = func(ctx context.Context) error {
					return nil
				}
				return env
			},
			assert: func(t *testing.T, result model.UpdateNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.NoError(t, err)
				payload, ok := result.(*model.UpdateNotFoundIgnoredPatternPayload)
				require.True(t, ok)
				require.Equal(t, int64(1), payload.NotFoundIgnoredPattern.ID)
				require.Equal(t, "^/admin/updated/.*", payload.NotFoundIgnoredPattern.Pattern)
				require.Equal(t, int64(1), payload.NotFoundIgnoredPattern.CreatedBy)
			},
		},
		{
			name: "pattern not found",
			input: model.UpdateNotFoundIgnoredPatternInput{
				ID:      999,
				Pattern: "^/admin/updated/.*",
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				env.NotFoundIgnoredPatternByIDFunc = func(ctx context.Context, id int64) (db.NotFoundIgnoredPattern, error) {
					return db.NotFoundIgnoredPattern{}, sql.ErrNoRows
				}
				env.RefreshNotFoundTrackerFunc = func(ctx context.Context) error {
					return nil
				}
				return env
			},
			assert: func(t *testing.T, result model.UpdateNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.NoError(t, err)
				errorPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Equal(t, "pattern not found", errorPayload.Message)
			},
		},
		{
			name: "invalid regex pattern",
			input: model.UpdateNotFoundIgnoredPatternInput{
				ID:      1,
				Pattern: "[invalid-regex",
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
				env.RefreshNotFoundTrackerFunc = func(ctx context.Context) error {
					return nil
				}
				return env
			},
			assert: func(t *testing.T, result model.UpdateNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.NoError(t, err)
				errorPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Contains(t, errorPayload.Message, "invalid regex pattern")
			},
		},
		{
			name: "admin auth failed",
			input: model.UpdateNotFoundIgnoredPatternInput{
				ID:      1,
				Pattern: "^/admin/updated/.*",
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("admin auth failed")
				}
				env.RefreshNotFoundTrackerFunc = func(ctx context.Context) error {
					return nil
				}
				return env
			},
			assert: func(t *testing.T, result model.UpdateNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get admin user token")
			},
		},
		{
			name: "refresh tracker failed",
			input: model.UpdateNotFoundIgnoredPatternInput{
				ID:      1,
				Pattern: "^/admin/updated/.*",
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
				env.UpdateNotFoundIgnoredPatternFunc = func(ctx context.Context, arg db.UpdateNotFoundIgnoredPatternParams) (db.NotFoundIgnoredPattern, error) {
					return db.NotFoundIgnoredPattern{
						ID:        1,
						Pattern:   "^/admin/updated/.*",
						CreatedBy: 1,
					}, nil
				}
				env.RefreshNotFoundTrackerFunc = func(ctx context.Context) error {
					return errors.New("tracker refresh failed")
				}
				return env
			},
			assert: func(t *testing.T, result model.UpdateNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to refresh not found tracker")
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
