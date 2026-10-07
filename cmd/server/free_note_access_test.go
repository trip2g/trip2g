package main

import (
	"context"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/case/submitform"
	"trip2g/internal/db"
	gmodel "trip2g/internal/graph/model"
	"trip2g/internal/mdloader"
	"trip2g/internal/noteloader"
	"trip2g/internal/userbans"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

type formSubmitApp struct {
	*app
}

func (formSubmitApp) EnqueueSendFormSubmitEmail(context.Context, int64) error {
	return nil
}

type freeNoteVault struct {
	app      *app
	freeID   int64
	closedID int64
	memberID int64
	adminID  int64
}

func seedFreeNoteVault(t *testing.T) freeNoteVault {
	t.Helper()
	ctx := context.Background()

	a := newSingleLoaderTestApp(t)
	a.appState.UserBans = userbans.New(a.Queries)
	a.latestNoteLoader = noteloader.New("latest", makeLatestNoteLoaderWrapper(a), mdloader.Config{})

	err := a.WriteQueries.InsertSubgraph(ctx, "paid")
	require.NoError(t, err)

	form := "form:\n  turnstile: false\n  fields:\n    - name: message\n      type: text\n      required: true\n"
	vault := freeNoteVault{
		app:      a,
		freeID:   insertLatestNote(t, a, "public.md", "---\nfree: true\nsubgraph: paid\n"+form+"---\nOpen to everyone.\n"),
		closedID: insertLatestNote(t, a, "chapter.md", "---\nsubgraph: paid\n"+form+"---\nFor subscribers.\n"),
	}

	memberParams := db.InsertUserWithEmailParams{Email: "member@example.com", CreatedVia: "test"}
	member, err := a.WriteQueries.InsertUserWithEmail(ctx, memberParams)
	require.NoError(t, err)
	vault.memberID = member.ID

	adminParams := db.InsertUserWithEmailParams{Email: "admin@example.com", CreatedVia: "test"}
	admin, err := a.WriteQueries.InsertUserWithEmail(ctx, adminParams)
	require.NoError(t, err)
	vault.adminID = admin.ID

	err = a.latestNoteLoader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true})
	require.NoError(t, err)

	return vault
}

func (v freeNoteVault) readerContext(t *testing.T, token *usertoken.Data) context.Context {
	t.Helper()

	fctx := newTestRequest(t, v.app)
	req, err := appreq.FromCtx(fctx)
	require.NoError(t, err)
	req.SetUserToken(token)
	return fctx
}

func TestFreeNoteIsReadableByEveryReader(t *testing.T) {
	v := seedFreeNoteVault(t)

	member := &usertoken.Data{ID: int(v.memberID), Role: "user"}
	admin := &usertoken.Data{ID: int(v.adminID), Role: "admin"}

	tests := []struct {
		name  string
		token *usertoken.Data
		note  int64
		want  bool
	}{
		{name: "guest reads the free note", token: nil, note: v.freeID, want: true},
		{name: "signed-in user without a grant reads the free note", token: member, note: v.freeID, want: true},
		{name: "admin reads the free note", token: admin, note: v.freeID, want: true},
		{name: "guest cannot read the closed note", token: nil, note: v.closedID, want: false},
		{name: "signed-in user without a grant cannot read the closed note", token: member, note: v.closedID, want: false},
		{name: "admin reads the closed note", token: admin, note: v.closedID, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := v.readerContext(t, tt.token)
			note := v.app.LatestNoteViews().GetByVersionID(tt.note)
			require.NotNil(t, note)

			got, err := v.app.CanReadNote(ctx, note)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)

			got, err = v.app.CanReadNoteVersion(ctx, note.VersionID)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSubmitFormOnFreeNoteBySignedInUserWithoutGrant(t *testing.T) {
	v := seedFreeNoteVault(t)
	member := &usertoken.Data{ID: int(v.memberID), Role: "user"}
	message := "hello"
	fields := []submitform.FieldValue{{Name: "message", StringValue: &message}}

	freeInput := submitform.Input{NoteVersionID: v.freeID, Fields: fields}
	payload, err := submitform.Resolve(v.readerContext(t, member), formSubmitApp{v.app}, freeInput)
	require.NoError(t, err)
	require.IsType(t, &gmodel.SubmitFormPayload{}, payload)

	closedInput := submitform.Input{NoteVersionID: v.closedID, Fields: fields}
	payload, err = submitform.Resolve(v.readerContext(t, member), formSubmitApp{v.app}, closedInput)
	require.NoError(t, err)
	require.Equal(t, &gmodel.ErrorPayload{Message: "form_not_found"}, payload)
}
