package main

import (
	"context"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"

	"github.com/stretchr/testify/require"
)

func TestNoteLoaders_OnlyLatestValidates(t *testing.T) {
	a := newSingleLoaderTestApp(t)
	a.createNoteLoaders()
	ctx := context.Background()

	rulesID := insertLatestNote(t, a, "rules.md", "```jsonnet\n[{level: 'error', msg: 'ran'}]\n```\n")
	noteID := insertLatestNote(t, a, "note.md", "---\nvalidate: \"[[rules]]\"\n---\nbody\n")

	userParams := db.InsertUserWithEmailParams{Email: "admin@example.com", CreatedVia: "test"}
	user, err := a.WriteQueries.InsertUserWithEmail(ctx, userParams)
	require.NoError(t, err)
	_, err = a.WriteQueries.InsertAdmin(ctx, db.InsertAdminParams{UserID: user.ID})
	require.NoError(t, err)
	release, err := a.WriteQueries.InsertRelease(ctx, db.InsertReleaseParams{CreatedBy: user.ID, IsLive: true})
	require.NoError(t, err)
	for _, versionID := range []int64{rulesID, noteID} {
		params := db.InsertReleaseNoteVersionParams{ReleaseID: release.ID, NoteVersionID: versionID}
		require.NoError(t, a.WriteQueries.InsertReleaseNoteVersion(ctx, params))
	}

	options := noteloader.LoadOptions{SkipSearchIndex: true}
	require.NoError(t, a.latestNoteLoader.Load(ctx, options))
	require.NoError(t, a.liveNoteLoader.Load(ctx, options))

	expected := []model.NoteWarning{{Level: model.NoteWarningCritical, Message: "validate rules.md: ran"}}
	require.Equal(t, expected, a.latestNoteLoader.NoteByPath("/note").Warnings)

	live := a.liveNoteLoader.NoteByPath("/note")
	require.NotNil(t, live)
	require.Empty(t, live.Warnings)

	single, err := a.LoadNoteViewByVersionID(ctx, noteID)
	require.NoError(t, err)
	require.Empty(t, single.Warnings, "a single-note load can't see the validator and must not report it missing")
}
