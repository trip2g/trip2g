package revokeusersubgraphaccess_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/revokeusersubgraphaccess"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

func newEnvMock(access db.UserSubgraphAccess, accessErr error) *EnvMock {
	return &EnvMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 7, Role: "admin"}, nil
		},
		UserSubgraphAccessByIDFunc: func(ctx context.Context, id int64) (db.UserSubgraphAccess, error) {
			return access, accessErr
		},
		CreateRevokeFunc: func(ctx context.Context, arg db.CreateRevokeParams) (db.Revoke, error) {
			return db.Revoke{ID: 99, TargetType: arg.TargetType, TargetID: arg.TargetID, ByID: arg.ByID, Reason: arg.Reason}, nil
		},
		RevokeUserSubgraphAccessFunc: func(ctx context.Context, arg db.RevokeUserSubgraphAccessParams) (db.UserSubgraphAccess, error) {
			revoked := access
			revoked.RevokeID = arg.RevokeID
			return revoked, nil
		},
	}
}

func TestResolveRevokes(t *testing.T) {
	env := newEnvMock(db.UserSubgraphAccess{ID: 5, UserID: 3, SubgraphID: 2}, nil)

	input := revokeusersubgraphaccess.Input{ID: 5, Reason: "  refunded  "}

	payload, err := revokeusersubgraphaccess.Resolve(context.Background(), env, input)
	require.NoError(t, err)

	success, ok := payload.(*model.RevokeUserSubgraphAccessPayload)
	require.True(t, ok, "got %#v", payload)
	require.Equal(t, ptr.To(int64(99)), success.Access.RevokeID)

	revokeCalls := env.CreateRevokeCalls()
	require.Len(t, revokeCalls, 1)

	wantRevoke := db.CreateRevokeParams{
		TargetType: revokeusersubgraphaccess.TargetType,
		TargetID:   5,
		ByID:       7,
		Reason:     ptr.To("refunded"),
	}
	require.Equal(t, wantRevoke, revokeCalls[0].Arg)

	accessCalls := env.RevokeUserSubgraphAccessCalls()
	require.Len(t, accessCalls, 1)
	require.Equal(t, int64(5), accessCalls[0].Arg.ID)
	require.Equal(t, ptr.To(int64(99)), accessCalls[0].Arg.RevokeID)
}

func TestResolveDomainFailures(t *testing.T) {
	tests := []struct {
		name      string
		access    db.UserSubgraphAccess
		accessErr error
		reason    string
		message   string
	}{
		{
			name:    "empty reason",
			access:  db.UserSubgraphAccess{ID: 5},
			reason:  "",
			message: "Reason is required",
		},
		{
			name:    "blank reason",
			access:  db.UserSubgraphAccess{ID: 5},
			reason:  " \t ",
			message: "Reason is required",
		},
		{
			name:      "Access not found",
			accessErr: sql.ErrNoRows,
			reason:    "refunded",
			message:   "Access not found",
		},
		{
			name:    "already revoked",
			access:  db.UserSubgraphAccess{ID: 5, RevokeID: ptr.To(int64(1))},
			reason:  "refunded",
			message: "Access already revoked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnvMock(tt.access, tt.accessErr)

			input := revokeusersubgraphaccess.Input{ID: 5, Reason: tt.reason}

			payload, err := revokeusersubgraphaccess.Resolve(context.Background(), env, input)
			require.NoError(t, err)

			errorPayload, ok := payload.(*model.ErrorPayload)
			require.True(t, ok, "got %#v", payload)
			require.Equal(t, tt.message, errorPayload.Message)
			require.Empty(t, env.CreateRevokeCalls())
			require.Empty(t, env.RevokeUserSubgraphAccessCalls())
		})
	}
}

func TestResolveInfrastructureFailure(t *testing.T) {
	env := newEnvMock(db.UserSubgraphAccess{ID: 5}, nil)
	env.CreateRevokeFunc = func(ctx context.Context, arg db.CreateRevokeParams) (db.Revoke, error) {
		return db.Revoke{}, errors.New("disk I/O error")
	}

	input := revokeusersubgraphaccess.Input{ID: 5, Reason: "refunded"}

	payload, err := revokeusersubgraphaccess.Resolve(context.Background(), env, input)
	require.Error(t, err)
	require.Nil(t, payload)
	require.Empty(t, env.RevokeUserSubgraphAccessCalls())
}
