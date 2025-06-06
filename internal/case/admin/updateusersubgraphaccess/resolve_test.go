package updateusersubgraphaccess_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/updateusersubgraphaccess"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updateusersubgraphaccess_test . Env

type Env interface {
	UpdateUserSubgraphAccess(ctx context.Context, arg db.UpdateUserSubgraphAccessParams) (db.UserSubgraphAccess, error)
}

type envMock = EnvMock

func TestRequest_Resolve(t *testing.T) {
	expiresAt := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	type fields struct {
		ID         int64
		ExpiresAt  *time.Time
		SubgraphID int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name        string
		fields      fields
		env         updateusersubgraphaccess.Env
		args        args
		want        model.UpdateUserSubgraphAccessOrErrorPayload
		wantErr     bool
		wantErrText string
	}{
		{
			name: "successful update with expires_at",
			fields: fields{
				ID:         123,
				ExpiresAt:  &expiresAt,
				SubgraphID: 456,
			},
			env: &envMock{
				UpdateUserSubgraphAccessFunc: func(ctx context.Context, arg db.UpdateUserSubgraphAccessParams) (db.UserSubgraphAccess, error) {
					return db.UserSubgraphAccess{
						ID:         123,
						ExpiresAt:  db.ToNullableTime(&expiresAt),
						SubgraphID: 456,
						UserID:     789,
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want: &model.UpdateUserSubgraphAccessPayload{
				UserSubgraphAccess: &db.UserSubgraphAccess{
					ID:         123,
					ExpiresAt:  db.ToNullableTime(&expiresAt),
					SubgraphID: 456,
					UserID:     789,
				},
			},
		},
		{
			name: "successful update with nil expires_at",
			fields: fields{
				ID:         456,
				ExpiresAt:  nil,
				SubgraphID: 789,
			},
			env: &envMock{
				UpdateUserSubgraphAccessFunc: func(ctx context.Context, arg db.UpdateUserSubgraphAccessParams) (db.UserSubgraphAccess, error) {
					return db.UserSubgraphAccess{
						ID:         456,
						ExpiresAt:  db.ToNullableTime(nil),
						SubgraphID: 789,
						UserID:     123,
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want: &model.UpdateUserSubgraphAccessPayload{
				UserSubgraphAccess: &db.UserSubgraphAccess{
					ID:         456,
					ExpiresAt:  db.ToNullableTime(nil),
					SubgraphID: 789,
					UserID:     123,
				},
			},
		},
		{
			name: "database error",
			fields: fields{
				ID:         789,
				ExpiresAt:  &expiresAt,
				SubgraphID: 123,
			},
			env: &envMock{
				UpdateUserSubgraphAccessFunc: func(ctx context.Context, arg db.UpdateUserSubgraphAccessParams) (db.UserSubgraphAccess, error) {
					return db.UserSubgraphAccess{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr:     true,
			wantErrText: "failed to update user subgraph access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &updateusersubgraphaccess.Request{
				ID:         tt.fields.ID,
				ExpiresAt:  tt.fields.ExpiresAt,
				SubgraphID: tt.fields.SubgraphID,
			}
			got, err := req.Resolve(tt.args.ctx, tt.env)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrText)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)

			if env, ok := tt.env.(*envMock); ok {
				require.Len(t, env.UpdateUserSubgraphAccessCalls(), 1)
				call := env.UpdateUserSubgraphAccessCalls()[0]
				require.Equal(t, tt.fields.ID, call.Arg.ID)
				require.Equal(t, tt.fields.SubgraphID, call.Arg.SubgraphID)

				if tt.fields.ExpiresAt != nil {
					require.True(t, call.Arg.ExpiresAt.Valid)
					require.Equal(t, *tt.fields.ExpiresAt, call.Arg.ExpiresAt.Time)
				} else {
					require.False(t, call.Arg.ExpiresAt.Valid)
				}
			}
		})
	}
}
