package revokeusersubgraphaccess_test

import (
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/revokeusersubgraphaccess"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	_ "trip2g/internal/dbmate/sqlite"
)

type sqliteEnv struct {
	*db.WriteQueries

	actor *usertoken.Data
}

func (e *sqliteEnv) CurrentAdminUserToken(_ context.Context) (*usertoken.Data, error) {
	return e.actor, nil
}

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "revoke_test.sqlite3")

	u, err := url.Parse("sqlite:" + dbPath)
	require.NoError(t, err)

	dbm := dbmate.New(u)
	dbm.MigrationsDir = []string{"../../../../db/migrations"}
	dbm.AutoDumpSchema = false
	require.NoError(t, dbm.CreateAndMigrate())

	conn, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	return conn
}

func insertUser(ctx context.Context, t *testing.T, wq *db.WriteQueries, email string) int64 {
	t.Helper()

	params := db.InsertUserWithEmailParams{Email: email, CreatedVia: "test"}

	user, err := wq.InsertUserWithEmail(ctx, params)
	require.NoError(t, err)

	return user.ID
}

func TestResolveOnSQLite(t *testing.T) {
	ctx := context.Background()
	conn := openMigratedDB(t)
	wq := db.NewWriteQueries(conn)

	adminID := insertUser(ctx, t, wq, "admin@example.com")
	_, err := wq.InsertAdmin(ctx, db.InsertAdminParams{UserID: adminID})
	require.NoError(t, err)

	readerID := insertUser(ctx, t, wq, "reader@example.com")

	require.NoError(t, wq.InsertSubgraph(ctx, "paid"))
	subgraph, err := wq.SubgraphByName(ctx, "paid")
	require.NoError(t, err)

	accessParams := db.AdminCreateUserSubgraphAccessParams{
		UserID:     readerID,
		SubgraphID: subgraph.ID,
		CreatedBy:  &adminID,
	}

	access, err := wq.AdminCreateUserSubgraphAccess(ctx, accessParams)
	require.NoError(t, err)

	active, err := wq.ListActiveUserSubgraphAccessesByUserID(ctx, readerID)
	require.NoError(t, err)
	require.Len(t, active, 1)

	env := &sqliteEnv{
		WriteQueries: wq,
		actor:        &usertoken.Data{ID: int(adminID), Role: "admin"},
	}

	t.Run("empty reason is refused", func(t *testing.T) {
		payload, resolveErr := revokeusersubgraphaccess.Resolve(ctx, env, revokeusersubgraphaccess.Input{ID: access.ID, Reason: " "})
		require.NoError(t, resolveErr)
		require.Equal(t, &model.ErrorPayload{Message: "Reason is required"}, payload)

		stillActive, listErr := wq.ListActiveUserSubgraphAccessesByUserID(ctx, readerID)
		require.NoError(t, listErr)
		require.Len(t, stillActive, 1)
	})

	t.Run("admin revokes", func(t *testing.T) {
		payload, resolveErr := revokeusersubgraphaccess.Resolve(ctx, env, revokeusersubgraphaccess.Input{ID: access.ID, Reason: "refunded"})
		require.NoError(t, resolveErr)

		success, ok := payload.(*model.RevokeUserSubgraphAccessPayload)
		require.True(t, ok, "got %#v", payload)
		require.NotNil(t, success.Access.RevokeID)

		revoke, revokeErr := wq.RevokeByID(ctx, *success.Access.RevokeID)
		require.NoError(t, revokeErr)
		require.Equal(t, revokeusersubgraphaccess.TargetType, revoke.TargetType)
		require.Equal(t, access.ID, revoke.TargetID)
		require.Equal(t, adminID, revoke.ByID)
		require.Equal(t, "refunded", *revoke.Reason)

		remaining, listErr := wq.ListActiveUserSubgraphAccessesByUserID(ctx, readerID)
		require.NoError(t, listErr)
		require.Empty(t, remaining)

		names, namesErr := wq.ListActiveSubgraphNamesByUserID(ctx, readerID)
		require.NoError(t, namesErr)
		require.Empty(t, names)
	})

	t.Run("second revoke is refused", func(t *testing.T) {
		payload, resolveErr := revokeusersubgraphaccess.Resolve(ctx, env, revokeusersubgraphaccess.Input{ID: access.ID, Reason: "again"})
		require.NoError(t, resolveErr)
		require.Equal(t, &model.ErrorPayload{Message: "Access already revoked"}, payload)
	})

	t.Run("unknown access is refused", func(t *testing.T) {
		payload, resolveErr := revokeusersubgraphaccess.Resolve(ctx, env, revokeusersubgraphaccess.Input{ID: access.ID + 1000, Reason: "refunded"})
		require.NoError(t, resolveErr)
		require.Equal(t, &model.ErrorPayload{Message: "Access not found"}, payload)
	})
}
