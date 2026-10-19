package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/case/hidenotes"
	"trip2g/internal/case/pushnotes"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/notebus"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// noteTxEnv runs pushnotes and hidenotes against the real database through the
// request transaction, and stubs only the in-memory side effects.
type noteTxEnv struct {
	*app
}

func (e noteTxEnv) PrepareLatestNotes(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
	return appmodel.NewNoteViews(), nil
}

func (e noteTxEnv) LatestNoteViews() *appmodel.NoteViews {
	return appmodel.NewNoteViews()
}

func (e noteTxEnv) HandleLatestNotesAfterSave(_ context.Context, _ []int64) error {
	return nil
}

func (e noteTxEnv) Layouts() *appmodel.Layouts {
	return &appmodel.Layouts{Map: map[string]appmodel.Layout{}}
}

func (e noteTxEnv) CheckStorageLimits(_ context.Context, _ int64) (string, error) {
	return "", nil
}

func (e noteTxEnv) PublicURL() string {
	return ""
}

func (e noteTxEnv) PublishNoteChanges(_ notebus.Batch) {}

func (e noteTxEnv) HandleNoteWebhooks(_ context.Context, _ []handlenotewebhooks.NoteChange, _ int) error {
	return nil
}

// newNoteTestRequest is an anonymous request: the note writes record no user.
func newNoteTestRequest(t *testing.T, a *app) *fasthttp.RequestCtx {
	t.Helper()

	fctx := newTestRequest(t, a)
	req, err := appreq.FromCtx(fctx)
	require.NoError(t, err)
	req.SetUserToken(nil)

	return fctx
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return base64.URLEncoding.EncodeToString(sum[:])
}

func requestTxEnv(fctx *fasthttp.RequestCtx) noteTxEnv {
	req, err := appreq.FromCtx(fctx)
	if err != nil {
		panic(err)
	}

	return noteTxEnv{app: req.Env.(*app)}
}

// noteHashInTx reads note.md's hash the only way LatestNoteContentHash
// allows: inside a write transaction.
func noteHashInTx(t *testing.T, a *app) string {
	t.Helper()

	var hash string
	err := a.WithTransaction(context.Background(), func(txCtx context.Context, txEnv *app) (bool, error) {
		var readErr error
		hash, readErr = txEnv.LatestNoteContentHash(txCtx, "note.md")
		return false, readErr
	})
	require.NoError(t, err)

	return hash
}

// Outside the request transaction another write could land between the hash
// check and the write it gates, so the read refuses to run at all.
func TestLatestNoteContentHashRequiresTx(t *testing.T) {
	a := newTxTestApp(t)

	_, err := a.LatestNoteContentHash(context.Background(), "note.md")
	require.Error(t, err)
}

// pushInTx runs one pushNotes request in its own transaction, as the GraphQL
// handler does. It reports failures instead of failing t, so it can run in a
// goroutine.
func pushInTx(fctx *fasthttp.RequestCtx, a *app, path, content, expectedHash string) (model.PushNotesOrErrorPayload, error) {
	err := a.AcquireTxEnvInRequest(fctx, "pushNotes")
	if err != nil {
		return nil, err
	}

	input := model.PushNotesInput{
		Updates: []model.PushNoteInput{{Path: path, Content: content, ExpectedHash: &expectedHash}},
	}
	payload, err := pushnotes.Resolve(fctx, requestTxEnv(fctx), input)

	releaseErr := a.ReleaseTxEnvInRequest(fctx, err == nil)
	if releaseErr != nil {
		return nil, releaseErr
	}

	return payload, err
}

// Two clients that listed the same hash push concurrently. The check and the
// write share one request transaction on the single write connection, so the
// second push waits for the first to commit, then sees the new hash and is
// refused instead of overwriting it.
func TestPushNotesExpectedHashConcurrentWrites(t *testing.T) {
	a := newTxTestApp(t)
	ctx := newNoteTestRequest(t, a)

	_, err := a.InsertNote(ctx, appmodel.RawNote{Path: "note.md", Content: "base"})
	require.NoError(t, err)
	base := contentHash("base")

	first := newNoteTestRequest(t, a)
	require.NoError(t, a.AcquireTxEnvInRequest(first, "pushNotes"))

	firstInput := model.PushNotesInput{
		Updates: []model.PushNoteInput{{Path: "note.md", Content: "first", ExpectedHash: &base}},
	}
	firstPayload, err := pushnotes.Resolve(first, requestTxEnv(first), firstInput)
	require.NoError(t, err)
	require.Empty(t, firstPayload.(*model.PushNotesPayload).HashMismatches)

	type outcome struct {
		payload model.PushNotesOrErrorPayload
		err     error
	}
	second := make(chan outcome, 1)
	secondReq := newNoteTestRequest(t, a)
	go func() {
		payload, pushErr := pushInTx(secondReq, a, "note.md", "second", base)
		second <- outcome{payload: payload, err: pushErr}
	}()

	select {
	case <-second:
		t.Fatal("the second push ran while the first push's transaction was open")
	case <-time.After(300 * time.Millisecond):
	}

	require.NoError(t, a.ReleaseTxEnvInRequest(first, true))

	var got outcome
	select {
	case got = <-second:
	case <-time.After(5 * time.Second):
		t.Fatal("the second push never ran")
	}
	require.NoError(t, got.err)

	payload, ok := got.payload.(*model.PushNotesPayload)
	require.True(t, ok, "got %T", got.payload)
	require.Equal(t, []model.NoteHashMismatch{{Path: "note.md", ActualHash: contentHash("first")}}, payload.HashMismatches)

	require.Equal(t, contentHash("first"), noteHashInTx(t, a))
}

// A delete that saw the old hash is refused once the note was edited, a hidden
// note counts as absent for the next expectedHash, and hiding it again with ""
// leaves who hid it and when untouched.
func TestHideNotesExpectedHash(t *testing.T) {
	a := newTxTestApp(t)
	ctx := newNoteTestRequest(t, a)

	user, err := a.WriteQueries.InsertUserWithEmail(ctx, db.InsertUserWithEmailParams{Email: "admin@example.com", CreatedVia: "test"})
	require.NoError(t, err)
	_, err = a.WriteQueries.InsertAdmin(ctx, db.InsertAdminParams{UserID: user.ID})
	require.NoError(t, err)

	_, err = a.InsertNote(ctx, appmodel.RawNote{Path: "note.md", Content: "base"})
	require.NoError(t, err)
	_, err = a.InsertNote(ctx, appmodel.RawNote{Path: "note.md", Content: "edited"})
	require.NoError(t, err)

	hide := func(actor int64, expectedHash string) *model.HideNotesPayload {
		fctx := newNoteTestRequest(t, a)
		require.NoError(t, a.AcquireTxEnvInRequest(fctx, "hideNotes"))

		input := model.HideNotesInput{
			Notes:  []model.HideNoteInput{{Path: "note.md", ExpectedHash: &expectedHash}},
			ApiKey: db.ApiKey{CreatedBy: actor},
		}
		result, hideErr := hidenotes.Resolve(fctx, requestTxEnv(fctx), input)
		require.NoError(t, hideErr)
		require.NoError(t, a.ReleaseTxEnvInRequest(fctx, true))

		payload, ok := result.(*model.HideNotesPayload)
		require.True(t, ok, "got %T", result)
		return payload
	}

	refused := hide(user.ID, contentHash("base"))
	require.Equal(t, []model.NoteHashMismatch{{Path: "note.md", ActualHash: contentHash("edited")}}, refused.HashMismatches)

	require.Equal(t, contentHash("edited"), noteHashInTx(t, a), "a refused hide leaves the note visible")

	require.Empty(t, hide(user.ID, contentHash("edited")).HashMismatches)

	require.Empty(t, noteHashInTx(t, a), "a hidden note has no current hash")

	hiddenBefore, err := a.Queries.ListNotePathsByValues(ctx, []string{"note.md"})
	require.NoError(t, err)
	require.Len(t, hiddenBefore, 1)
	require.Equal(t, &user.ID, hiddenBefore[0].HiddenBy)

	other, err := a.WriteQueries.InsertUserWithEmail(ctx, db.InsertUserWithEmailParams{Email: "other@example.com", CreatedVia: "test"})
	require.NoError(t, err)
	_, err = a.WriteQueries.InsertAdmin(ctx, db.InsertAdminParams{UserID: other.ID})
	require.NoError(t, err)

	require.Empty(t, hide(other.ID, "").HashMismatches)

	hiddenAfter, err := a.Queries.ListNotePathsByValues(ctx, []string{"note.md"})
	require.NoError(t, err)
	require.Len(t, hiddenAfter, 1)
	require.Equal(t, hiddenBefore[0].HiddenBy, hiddenAfter[0].HiddenBy, "hiding an already hidden note keeps who hid it")
	require.Equal(t, hiddenBefore[0].HiddenAt, hiddenAfter[0].HiddenAt, "hiding an already hidden note keeps when it was hidden")

	payload, err := pushInTx(newNoteTestRequest(t, a), a, "note.md", "restored", contentHash("edited"))
	require.NoError(t, err)
	require.Equal(t, []model.NoteHashMismatch{{Path: "note.md", ActualHash: ""}}, payload.(*model.PushNotesPayload).HashMismatches)

	payload, err = pushInTx(newNoteTestRequest(t, a), a, "note.md", "restored", "")
	require.NoError(t, err)
	require.Empty(t, payload.(*model.PushNotesPayload).HashMismatches)

	require.Equal(t, contentHash("restored"), noteHashInTx(t, a))
}
