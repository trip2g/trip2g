package db_test

import (
	"context"
	"testing"

	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

func TestFederationSecretQueries(t *testing.T) {
	conn, queries, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	writeQueries := db.NewWriteQueries(conn)
	adminUserID := insertTestAdminUser(t, conn)

	kbURL := "https://bob.team.io/_system/mcp"
	oldOutbound, err := writeQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid:         "bob-old",
		SecretCrypt: []byte("old-outbound"),
		KbUrl:       &kbURL,
		CreatedBy:   adminUserID,
	})
	require.NoError(t, err)
	mustExec(t, conn, `update federation_secrets set created_at = '2026-01-01 00:00:00' where id = ?`, oldOutbound.ID)

	newOutbound, err := writeQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid:         "bob-new",
		SecretCrypt: []byte("new-outbound"),
		KbUrl:       &kbURL,
		CreatedBy:   adminUserID,
	})
	require.NoError(t, err)
	mustExec(t, conn, `update federation_secrets set created_at = '2026-01-02 00:00:00' where id = ?`, newOutbound.ID)

	revokedOutbound, err := writeQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid:         "bob-revoked",
		SecretCrypt: []byte("revoked-outbound"),
		KbUrl:       &kbURL,
		CreatedBy:   adminUserID,
	})
	require.NoError(t, err)
	mustExec(t, conn, `update federation_secrets set created_at = '2026-01-03 00:00:00' where id = ?`, revokedOutbound.ID)
	require.NoError(t, writeQueries.RevokeFederationSecret(ctx, revokedOutbound.ID))

	gotOutbound, err := queries.FederationSecretByKBURL(ctx, &kbURL)
	require.NoError(t, err)
	require.Equal(t, newOutbound.ID, gotOutbound.ID)
	require.Equal(t, "bob-new", gotOutbound.Kid)

	oldInbound, err := writeQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid:         "alice",
		SecretCrypt: []byte("old-inbound"),
		CreatedBy:   adminUserID,
	})
	require.NoError(t, err)
	mustExec(t, conn, `update federation_secrets set created_at = '2026-01-01 00:00:00' where id = ?`, oldInbound.ID)

	newInbound, err := writeQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid:         "alice",
		SecretCrypt: []byte("new-inbound"),
		CreatedBy:   adminUserID,
	})
	require.NoError(t, err)
	mustExec(t, conn, `update federation_secrets set created_at = '2026-01-02 00:00:00' where id = ?`, newInbound.ID)

	revokedInbound, err := writeQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid:         "alice",
		SecretCrypt: []byte("revoked-inbound"),
		CreatedBy:   adminUserID,
	})
	require.NoError(t, err)
	mustExec(t, conn, `update federation_secrets set created_at = '2026-01-03 00:00:00' where id = ?`, revokedInbound.ID)
	require.NoError(t, writeQueries.RevokeFederationSecret(ctx, revokedInbound.ID))

	otherURL := "https://alice-public.example/_system/mcp"
	_, err = writeQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid:         "alice",
		SecretCrypt: []byte("outbound-same-kid"),
		KbUrl:       &otherURL,
		CreatedBy:   adminUserID,
	})
	require.NoError(t, err)

	gotInbound, err := queries.FederationSecretByKID(ctx, "alice")
	require.NoError(t, err)
	require.Equal(t, newInbound.ID, gotInbound.ID)
	require.Nil(t, gotInbound.KbUrl)

	require.NoError(t, writeQueries.InsertSubgraph(ctx, "team-status"))
	require.NoError(t, writeQueries.InsertSubgraph(ctx, "private-notes"))
	teamStatus, err := queries.SubgraphByName(ctx, "team-status")
	require.NoError(t, err)
	privateNotes, err := queries.SubgraphByName(ctx, "private-notes")
	require.NoError(t, err)
	require.NoError(t, writeQueries.InsertFederationSecretSubgraph(ctx, db.InsertFederationSecretSubgraphParams{
		Kid:        "alice",
		SubgraphID: privateNotes.ID,
		CreatedBy:  adminUserID,
	}))
	require.NoError(t, writeQueries.InsertFederationSecretSubgraph(ctx, db.InsertFederationSecretSubgraphParams{
		Kid:        "alice",
		SubgraphID: teamStatus.ID,
		CreatedBy:  adminUserID,
	}))

	subgraphs, err := queries.ListFederationSecretSubgraphsByKID(ctx, "alice")
	require.NoError(t, err)
	require.Equal(t, []string{"private-notes", "team-status"}, subgraphs)

	require.NoError(t, writeQueries.DeleteFederationSecretSubgraph(ctx, db.DeleteFederationSecretSubgraphParams{
		Kid:        "alice",
		SubgraphID: privateNotes.ID,
	}))
	subgraphs, err = queries.ListFederationSecretSubgraphsByKID(ctx, "alice")
	require.NoError(t, err)
	require.Equal(t, []string{"team-status"}, subgraphs)
}

// The retire-on-verify write decides from a row read earlier in the same
// request, so a rotation that lands in between must not have its freshly staged
// previous key wiped by that stale decision — the staged key is exactly what the
// rotation's own lost-response retry depends on.
func TestClearFederationSecretPrevIsConditional(t *testing.T) {
	conn, queries, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	writeQueries := db.NewWriteQueries(conn)
	adminUserID := insertTestAdminUser(t, conn)

	row, err := writeQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid:         "peer",
		SecretCrypt: []byte("k1"),
		CreatedBy:   adminUserID,
	})
	require.NoError(t, err)

	// current = k2, prev = k1. A request reads this state and decides to retire k1.
	require.NoError(t, writeQueries.RotateFederationSecret(ctx, db.RotateFederationSecretParams{
		SecretCrypt: []byte("k2"),
		ID:          row.ID,
	}))

	// A second rotation lands first: current = k3, prev = k2.
	require.NoError(t, writeQueries.RotateFederationSecret(ctx, db.RotateFederationSecretParams{
		SecretCrypt: []byte("k3"),
		ID:          row.ID,
	}))

	// The stale decision now executes, naming the key it actually saw.
	require.NoError(t, writeQueries.ClearFederationSecretPrev(ctx, db.ClearFederationSecretPrevParams{
		ID:              row.ID,
		PrevSecretCrypt: []byte("k1"),
	}))

	after, err := queries.FederationSecretByID(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, []byte("k2"), after.PrevSecretCrypt,
		"a stale clear wiped the key the newer rotation staged for its own healing window")

	// Naming what is actually there still retires it.
	require.NoError(t, writeQueries.ClearFederationSecretPrev(ctx, db.ClearFederationSecretPrevParams{
		ID:              row.ID,
		PrevSecretCrypt: []byte("k2"),
	}))

	after, err = queries.FederationSecretByID(ctx, row.ID)
	require.NoError(t, err)
	require.Nil(t, after.PrevSecretCrypt)
}
