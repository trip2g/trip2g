package listnotefieldhistory_test

import (
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/listnotefieldhistory"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	_ "trip2g/internal/dbmate/sqlite"
)

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "field_history_test.sqlite3")

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

func exec(t *testing.T, conn *sql.DB, query string, args ...any) {
	t.Helper()

	_, err := conn.Exec(query, args...)
	require.NoError(t, err)
}

func TestResolveOnSQLite(t *testing.T) {
	ctx := context.Background()
	conn := openMigratedDB(t)
	wq := db.NewWriteQueries(conn)

	userParams := db.InsertUserWithEmailParams{Email: "alexey@example.com", CreatedVia: "test"}

	user, err := wq.InsertUserWithEmail(ctx, userParams)
	require.NoError(t, err)

	_, err = wq.InsertAdmin(ctx, db.InsertAdminParams{UserID: user.ID})
	require.NoError(t, err)

	exec(t, conn, "insert into api_keys (id, value, created_by, description) values (3, 'k', ?, 'Obsidian sync')", user.ID)
	exec(t, conn, "insert into note_paths (id, value, value_hash, latest_content_hash) values (1, ?, 'h', 'h')", notePath)

	insertVersion := "insert into note_versions (id, path_id, version, content, created_by_user_id, created_by_api_key_id, created_by_client)" +
		" values (?, 1, ?, ?, ?, ?, ?)"
	exec(t, conn, insertVersion, 10, 1, "---\nstatus: Planned\n---\nShip it\n", nil, nil, nil)
	exec(t, conn, insertVersion, 11, 2, "---\nstatus: Planned\nowner: Anna\n---\nShip it\n", nil, 3, nil)
	exec(t, conn, insertVersion, 12, 3, "---\nstatus: Done\n---\nShipped\n", user.ID, nil, "web")

	insertFrontmatter := "insert into note_version_frontmatters (version_id, data) values (?, ?)"
	exec(t, conn, insertFrontmatter, 11, `{"owner":"Anna","status":"Planned"}`)
	exec(t, conn, insertFrontmatter, 12, `{"status":"Done"}`)

	got, err := listnotefieldhistory.Resolve(ctx, wq, listnotefieldhistory.Filter{Path: notePath})
	require.NoError(t, err)
	require.Equal(t, int32(3), got.TotalCount)

	done := got.Nodes[0]
	require.Equal(t, model.AdminNoteFieldHistoryKindChanged, done.Kind)
	require.Equal(t, "alexey@example.com · via web", done.Author.Label)
	require.Equal(t, &user.ID, done.Author.UserID)
	require.True(t, done.BodyChanged)
	wantChanges := []model.AdminNoteFieldChange{
		{Key: "owner", From: "Anna"},
		{Key: "status", From: "Planned", To: "Done"},
	}
	require.Equal(t, wantChanges, done.Changes)

	owner := got.Nodes[1]
	require.Equal(t, model.AdminNoteFieldHistoryKindUnknown, owner.Kind)
	require.Equal(t, "Obsidian sync", owner.Author.Label)
	require.Equal(t, ptr(int64(3)), owner.Author.APIKeyID)
	require.False(t, owner.BodyChanged)

	first := got.Nodes[2]
	require.Equal(t, model.AdminNoteFieldHistoryKindUnknown, first.Kind)
	require.Empty(t, first.Author.Label)
}
