package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
)

func TestListActiveBoostySubgraphNamesAcceptsWriterActiveStatus(t *testing.T) {
	ctx := context.Background()
	conn, queries, cleanup := setupTestDB(t)
	defer cleanup()

	writeQueries := db.NewWriteQueries(conn)
	adminID := insertTestAdminUser(t, conn)

	var userID int64
	err := conn.QueryRowContext(ctx, `
		insert into users (email, created_via)
		values ('boosty-reader@example.com', 'boosty')
		returning id
	`).Scan(&userID)
	require.NoError(t, err)

	require.NoError(t, writeQueries.InsertSubgraph(ctx, "boosty-paid"))
	subgraph, err := queries.SubgraphByName(ctx, "boosty-paid")
	require.NoError(t, err)

	credentials, err := writeQueries.InsertBoostyCredentials(ctx, db.InsertBoostyCredentialsParams{
		CreatedBy: adminID,
		AuthData:  `{}`,
		DeviceID:  "device-1",
		BlogName:  "test-blog",
	})
	require.NoError(t, err)

	const boostyTierID int64 = 101
	require.NoError(t, writeQueries.UpsertBoostyTier(ctx, db.UpsertBoostyTierParams{
		CredentialsID: credentials.ID,
		BoostyID:      boostyTierID,
		Name:          "Paid tier",
		Data:          `{}`,
	}))
	tierID, err := queries.GetBoostyTierIDByCredentialsAndBoostyID(ctx, db.GetBoostyTierIDByCredentialsAndBoostyIDParams{
		CredentialsID: credentials.ID,
		BoostyID:      boostyTierID,
	})
	require.NoError(t, err)

	require.NoError(t, writeQueries.InsertBoostyTierSubgraph(ctx, db.InsertBoostyTierSubgraphParams{
		TierID:     tierID,
		SubgraphID: subgraph.ID,
		CreatedBy:  adminID,
	}))
	require.NoError(t, writeQueries.UpsertBoostyMember(ctx, db.UpsertBoostyMemberParams{
		CredentialsID: credentials.ID,
		BoostyID:      202,
		Email:         "boosty-reader@example.com",
		Status:        "active",
		Data:          `{}`,
		CurrentTierID: &tierID,
	}))

	got, err := queries.ListActiveBoostySubgraphNamesByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, []string{"boosty-paid"}, got,
		"an active Boosty member written by refreshboostydata must receive the tier's subgraphs")
}
