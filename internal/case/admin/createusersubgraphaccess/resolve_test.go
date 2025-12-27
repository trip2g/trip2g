package createusersubgraphaccess_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/createusersubgraphaccess"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		env       createusersubgraphaccess.Env
		input     model.CreateUserSubgraphAccessInput
		wantErr   bool
		wantIsErr bool
		wantLen   int
		errMsg    string
	}{
		{
			name: "successful creation with single subgraph",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{ID: 2}, nil
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{ID: id}, nil
				},
				AdminCreateUserSubgraphAccessFunc: func(ctx context.Context, arg db.AdminCreateUserSubgraphAccessParams) (db.UserSubgraphAccess, error) {
					return db.UserSubgraphAccess{
						ID:         1,
						UserID:     arg.UserID,
						SubgraphID: arg.SubgraphID,
						CreatedAt:  now,
						ExpiresAt:  arg.ExpiresAt,
						CreatedBy:  arg.CreatedBy,
					}, nil
				},
			},
			input: model.CreateUserSubgraphAccessInput{
				UserID:      2,
				SubgraphIds: []int64{10},
				ExpiresAt:   &now,
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "successful creation with multiple subgraphs",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{ID: 2}, nil
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{ID: id}, nil
				},
				AdminCreateUserSubgraphAccessFunc: func(ctx context.Context, arg db.AdminCreateUserSubgraphAccessParams) (db.UserSubgraphAccess, error) {
					return db.UserSubgraphAccess{
						ID:         arg.SubgraphID,
						UserID:     arg.UserID,
						SubgraphID: arg.SubgraphID,
						CreatedAt:  now,
					}, nil
				},
			},
			input: model.CreateUserSubgraphAccessInput{
				UserID:      2,
				SubgraphIds: []int64{10, 20, 30},
			},
			wantErr: false,
			wantLen: 3,
		},
		{
			name: "error - not authenticated",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("not authenticated")
				},
			},
			input: model.CreateUserSubgraphAccessInput{
				UserID:      2,
				SubgraphIds: []int64{10},
			},
			wantErr: true,
		},
		{
			name: "error - empty subgraphIds",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
			},
			input: model.CreateUserSubgraphAccessInput{
				UserID:      2,
				SubgraphIds: []int64{},
			},
			wantErr:   false,
			wantIsErr: true,
		},
		{
			name: "error - user not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
			},
			input: model.CreateUserSubgraphAccessInput{
				UserID:      999,
				SubgraphIds: []int64{10},
			},
			wantErr: false,
			errMsg:  "User not found",
		},
		{
			name: "error - subgraph not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{ID: 2}, nil
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{}, sql.ErrNoRows
				},
			},
			input: model.CreateUserSubgraphAccessInput{
				UserID:      2,
				SubgraphIds: []int64{999},
			},
			wantErr: false,
			errMsg:  "Subgraph 999 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createusersubgraphaccess.Resolve(context.Background(), tt.env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantIsErr || tt.errMsg != "" {
				errPayload, ok := got.(*model.ErrorPayload)
				require.True(t, ok, "expected ErrorPayload")
				if tt.errMsg != "" {
					require.Contains(t, errPayload.Message, tt.errMsg)
				}
				return
			}

			payload, ok := got.(*model.CreateUserSubgraphAccessPayload)
			require.True(t, ok, "expected CreateUserSubgraphAccessPayload")
			require.Len(t, payload.Accesses, tt.wantLen)
		})
	}
}
