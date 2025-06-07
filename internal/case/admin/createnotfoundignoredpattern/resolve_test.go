package createnotfoundignoredpattern

import (
	"context"
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
		input  model.CreateNotFoundIgnoredPatternInput
		env    func() *EnvMock
		assert func(t *testing.T, result model.CreateNotFoundIgnoredPatternOrErrorPayload, err error)
	}{
		{
			name: "success",
			input: model.CreateNotFoundIgnoredPatternInput{
				Pattern: "^/admin/.*",
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				env.InsertNotFoundIgnoredPatternFunc = func(ctx context.Context, arg db.InsertNotFoundIgnoredPatternParams) (db.NotFoundIgnoredPattern, error) {
					return db.NotFoundIgnoredPattern{
						ID:        1,
						Pattern:   "^/admin/.*",
						CreatedBy: 1,
					}, nil
				}
				return env
			},
			assert: func(t *testing.T, result model.CreateNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.NoError(t, err)
				payload, ok := result.(*model.CreateNotFoundIgnoredPatternPayload)
				require.True(t, ok)
				require.Equal(t, int64(1), payload.NotFoundIgnoredPattern.ID)
				require.Equal(t, "^/admin/.*", payload.NotFoundIgnoredPattern.Pattern)
				require.Equal(t, int64(1), payload.NotFoundIgnoredPattern.CreatedBy)
			},
		},
		{
			name: "invalid regex pattern",
			input: model.CreateNotFoundIgnoredPatternInput{
				Pattern: "[invalid-regex",
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				return env
			},
			assert: func(t *testing.T, result model.CreateNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.NoError(t, err)
				errorPayload, ok := result.(*model.ErrorPayload)
				require.True(t, ok)
				require.Contains(t, errorPayload.Message, "invalid regex pattern")
			},
		},
		{
			name: "admin auth failed",
			input: model.CreateNotFoundIgnoredPatternInput{
				Pattern: "^/admin/.*",
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("admin auth failed")
				}
				return env
			},
			assert: func(t *testing.T, result model.CreateNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get admin user token")
			},
		},
		{
			name: "database insert failed",
			input: model.CreateNotFoundIgnoredPatternInput{
				Pattern: "^/admin/.*",
			},
			env: func() *EnvMock {
				env := &EnvMock{}
				env.CurrentAdminUserTokenFunc = func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				}
				env.InsertNotFoundIgnoredPatternFunc = func(ctx context.Context, arg db.InsertNotFoundIgnoredPatternParams) (db.NotFoundIgnoredPattern, error) {
					return db.NotFoundIgnoredPattern{}, errors.New("database error")
				}
				return env
			},
			assert: func(t *testing.T, result model.CreateNotFoundIgnoredPatternOrErrorPayload, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to insert not found ignored pattern")
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
