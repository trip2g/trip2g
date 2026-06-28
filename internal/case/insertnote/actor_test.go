package insertnote_test

import (
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"trip2g/internal/case/insertnote"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	"github.com/stretchr/testify/require"

	// Registers the dbmate "sqlite" driver and the modernc.org/sqlite sql driver.
	_ "trip2g/internal/dbmate/sqlite"
)

// actorEnv is a real-DB insertnote.Env whose NoteVersionActor returns a
// configurable acting user / api key / client, so tests can assert the recorded actor.
type actorEnv struct {
	*db.WriteQueries
	userID   *int64
	apiKeyID *int64
	client   *string
}

func (e *actorEnv) NoteVersionActor(_ context.Context) model.NoteActor {
	return model.NoteActor{UserID: e.userID, APIKeyID: e.apiKeyID, Client: e.client}
}

func setupActorDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "actor_test.sqlite3")

	u, err := url.Parse("sqlite:" + dbPath)
	require.NoError(t, err)

	dbm := dbmate.New(u)
	dbm.MigrationsDir = []string{"../../../db/migrations"}
	dbm.AutoDumpSchema = false
	require.NoError(t, dbm.CreateAndMigrate())

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	return conn
}

// latestVersion returns the most recent note_version row for a path.
func latestVersion(t *testing.T, rq *db.Queries, pathID int64) db.NoteVersion {
	t.Helper()
	versions, err := rq.AllNoteVersionsByPathID(context.Background(), pathID)
	require.NoError(t, err)
	require.NotEmpty(t, versions)
	return versions[0]
}

func TestInsertNoteRecordsUserActor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := setupActorDB(t)
	wq := db.NewWriteQueries(conn)
	rq := db.New(conn)

	user, err := wq.InsertUserWithEmail(ctx, db.InsertUserWithEmailParams{
		Email:      "editor@example.com",
		CreatedVia: "test",
	})
	require.NoError(t, err)

	env := &actorEnv{WriteQueries: wq, userID: &user.ID}

	pathID, _, err := insertnote.Resolve(ctx, env, model.RawNote{Path: "user-note.md", Content: "hello"})
	require.NoError(t, err)

	version := latestVersion(t, rq, pathID)
	require.NotNil(t, version.CreatedByUserID)
	require.Equal(t, user.ID, *version.CreatedByUserID)
	require.Nil(t, version.CreatedByApiKeyID)

	// Read accessor data: the editor query joins back the user email.
	editor, err := rq.NoteVersionEditor(ctx, version.ID)
	require.NoError(t, err)
	require.NotNil(t, editor.UserEmail)
	require.Equal(t, "editor@example.com", *editor.UserEmail)
	require.Nil(t, editor.ApiKeyDescription)
	require.Nil(t, editor.CreatedByClient)
	require.False(t, editor.CreatedAt.IsZero())
}

func TestInsertNoteRecordsAPIKeyActor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := setupActorDB(t)
	wq := db.NewWriteQueries(conn)
	rq := db.New(conn)

	owner, err := wq.InsertUserWithEmail(ctx, db.InsertUserWithEmailParams{
		Email:      "owner@example.com",
		CreatedVia: "test",
	})
	require.NoError(t, err)

	_, err = wq.InsertAdmin(ctx, db.InsertAdminParams{UserID: owner.ID})
	require.NoError(t, err)

	apiKey, err := wq.InsertAPIKey(ctx, db.InsertAPIKeyParams{
		Value:       "test-key-value",
		CreatedBy:   owner.ID,
		Description: "Obsidian Sync Key",
	})
	require.NoError(t, err)

	env := &actorEnv{WriteQueries: wq, apiKeyID: &apiKey.ID}

	pathID, _, err := insertnote.Resolve(ctx, env, model.RawNote{Path: "synced-note.md", Content: "from sync"})
	require.NoError(t, err)

	version := latestVersion(t, rq, pathID)
	require.NotNil(t, version.CreatedByApiKeyID)
	require.Equal(t, apiKey.ID, *version.CreatedByApiKeyID)
	require.Nil(t, version.CreatedByUserID)

	// Read accessor data: the editor query joins back the api key description.
	editor, err := rq.NoteVersionEditor(ctx, version.ID)
	require.NoError(t, err)
	require.NotNil(t, editor.ApiKeyDescription)
	require.Equal(t, "Obsidian Sync Key", *editor.ApiKeyDescription)
	require.Nil(t, editor.UserEmail)
	require.Nil(t, editor.CreatedByClient)
	require.False(t, editor.CreatedAt.IsZero())
}

func TestInsertNoteRecordsNoActorAsNull(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := setupActorDB(t)
	wq := db.NewWriteQueries(conn)
	rq := db.New(conn)

	env := &actorEnv{WriteQueries: wq} // userID and apiKeyID both nil

	pathID, _, err := insertnote.Resolve(ctx, env, model.RawNote{Path: "anon-note.md", Content: "no actor"})
	require.NoError(t, err)

	version := latestVersion(t, rq, pathID)
	require.Nil(t, version.CreatedByUserID)
	require.Nil(t, version.CreatedByApiKeyID)
	require.Nil(t, version.CreatedByClient)

	editor, err := rq.NoteVersionEditor(ctx, version.ID)
	require.NoError(t, err)
	require.Nil(t, editor.UserEmail)
	require.Nil(t, editor.ApiKeyDescription)
	require.Nil(t, editor.CreatedByClient)
}

func TestInsertNoteRecordsClientHeader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := setupActorDB(t)
	wq := db.NewWriteQueries(conn)
	rq := db.New(conn)

	clientVal := "obsidian-plugin/4.2.1"
	env := &actorEnv{WriteQueries: wq, client: &clientVal}

	pathID, _, err := insertnote.Resolve(ctx, env, model.RawNote{Path: "client-note.md", Content: "from client"})
	require.NoError(t, err)

	version := latestVersion(t, rq, pathID)
	require.NotNil(t, version.CreatedByClient)
	require.Equal(t, clientVal, *version.CreatedByClient)
	require.Nil(t, version.CreatedByUserID)
	require.Nil(t, version.CreatedByApiKeyID)

	// Read accessor: NoteVersionEditor returns created_by_client.
	editor, err := rq.NoteVersionEditor(ctx, version.ID)
	require.NoError(t, err)
	require.NotNil(t, editor.CreatedByClient)
	require.Equal(t, clientVal, *editor.CreatedByClient)
	require.Nil(t, editor.UserEmail)
	require.Nil(t, editor.ApiKeyDescription)
}
