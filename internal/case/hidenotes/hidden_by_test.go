package hidenotes_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/case/checkapikey"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/case/hidenotes"
	"trip2g/internal/case/updatenotes"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	internalmodel "trip2g/internal/model"
	"trip2g/internal/notebus"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/usertoken"

	_ "trip2g/internal/dbmate/sqlite"
)

const hiddenByTestSecret = "hidden-by-test-secret"

// hiddenByEnv backs checkapikey, hidenotes and updatenotes with a real
// SQLite database that enforces foreign keys, so a hide attributed to a
// non-admin fails the way it does in production.
type hiddenByEnv struct {
	*db.Queries

	wq    *db.WriteQueries
	token *usertoken.Data
}

func (e *hiddenByEnv) CurrentUserToken(_ context.Context) (*usertoken.Data, error) {
	return e.token, nil
}

func (e *hiddenByEnv) InsertAPIKeyLog(ctx context.Context, arg db.InsertAPIKeyLogParams) error {
	return e.wq.InsertAPIKeyLog(ctx, arg)
}

func (e *hiddenByEnv) UpsertAPIKeyLogAction(ctx context.Context, name string) error {
	return e.wq.UpsertAPIKeyLogAction(ctx, name)
}

func (e *hiddenByEnv) UpsertAPIKeyLogIP(ctx context.Context, ip string) error {
	return e.wq.UpsertAPIKeyLogIP(ctx, ip)
}

func (e *hiddenByEnv) ShortAPITokenSecret() string {
	return hiddenByTestSecret
}

func (e *hiddenByEnv) HideNotePath(ctx context.Context, params db.HideNotePathParams) error {
	return e.wq.HideNotePath(ctx, params)
}

func (e *hiddenByEnv) LatestNoteViews() *internalmodel.NoteViews {
	return internalmodel.NewNoteViews()
}

func (e *hiddenByEnv) PrepareLatestNotes(_ context.Context, _ bool) (*internalmodel.NoteViews, error) {
	return internalmodel.NewNoteViews(), nil
}

func (e *hiddenByEnv) Logger() logger.Logger {
	return slog.Default()
}

func (e *hiddenByEnv) PublishNoteChanges(_ notebus.Batch) {}

func (e *hiddenByEnv) HandleNoteWebhooks(_ context.Context, _ []handlenotewebhooks.NoteChange, _ int) error {
	return nil
}

func (e *hiddenByEnv) InsertNote(_ context.Context, _ internalmodel.RawNote) (internalmodel.NoteSaveResult, error) {
	return internalmodel.NoteSaveResult{}, nil
}

func (e *hiddenByEnv) HandleLatestNotesAfterSave(_ context.Context, _ []int64) error {
	return nil
}

func openHiddenByDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "hidden_by_test.sqlite3")

	u, err := url.Parse("sqlite:" + dbPath)
	require.NoError(t, err)

	dbm := dbmate.New(u)
	dbm.MigrationsDir = []string{"../../../db/migrations"}
	dbm.AutoDumpSchema = false
	require.NoError(t, dbm.CreateAndMigrate())

	conn, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	return conn
}

func insertAdmin(ctx context.Context, t *testing.T, wq *db.WriteQueries, email string) int64 {
	t.Helper()

	user, err := wq.InsertUserWithEmail(ctx, db.InsertUserWithEmailParams{Email: email, CreatedVia: "test"})
	require.NoError(t, err)

	_, err = wq.InsertAdmin(ctx, db.InsertAdminParams{UserID: user.ID})
	require.NoError(t, err)

	return user.ID
}

func hiddenBy(ctx context.Context, t *testing.T, conn *sql.DB, path string) *int64 {
	t.Helper()

	var v sql.NullInt64
	err := conn.QueryRowContext(ctx, "select hidden_by from note_paths where value = ?", path).Scan(&v)
	require.NoError(t, err)
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

// TestHiddenByPerAuthPath runs each credential through checkapikey and then
// through both hide entry points (hideNotes and the updateNotes hide change),
// asserting what lands in note_paths.hidden_by.
func TestHiddenByPerAuthPath(t *testing.T) {
	conn := openHiddenByDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rq := db.New(conn)
	wq := db.NewWriteQueries(conn)

	sessionAdmin := insertAdmin(ctx, t, wq, "session@example.com")
	keyOwner := insertAdmin(ctx, t, wq, "keyowner@example.com")
	webhookOwner := insertAdmin(ctx, t, wq, "webhookowner@example.com")

	const plainKey = "hidden-by-test-key"
	hash := sha256.Sum256([]byte(plainKey))
	keyParams := db.InsertAPIKeyParams{
		Value:       hex.EncodeToString(hash[:]),
		CreatedBy:   keyOwner,
		Description: "test key",
	}
	_, err := wq.InsertAPIKey(ctx, keyParams)
	require.NoError(t, err)

	signWebhookToken := func(createdBy int64) string {
		data := shortapitoken.Data{
			Depth:         1,
			ReadPatterns:  []string{"**"},
			WritePatterns: []string{"**"},
			DeliveryKind:  "change",
			DeliveryID:    1,
			CreatedBy:     createdBy,
		}
		tok, signErr := shortapitoken.Sign(data, hiddenByTestSecret, time.Minute)
		require.NoError(t, signErr)
		return tok
	}

	type want struct {
		hiddenBy *int64
		errText  string
	}

	tests := []struct {
		name       string
		token      *usertoken.Data
		header     [2]string
		hideNotes  want
		updateHide want
	}{
		{
			name:       "admin session or personal token",
			token:      &usertoken.Data{ID: int(sessionAdmin), Role: "admin"},
			hideNotes:  want{hiddenBy: &sessionAdmin},
			updateHide: want{hiddenBy: &sessionAdmin},
		},
		{
			name:       "admin token without a user id",
			token:      &usertoken.Data{Role: "admin"},
			hideNotes:  want{errText: updatenotes.NoHideActorMessage},
			updateHide: want{errText: updatenotes.NoHideActorMessage},
		},
		{
			name:       "api key",
			header:     [2]string{"X-API-Key", plainKey},
			hideNotes:  want{hiddenBy: &keyOwner},
			updateHide: want{hiddenBy: &keyOwner},
		},
		{
			name:       "webhook token",
			header:     [2]string{"Authorization", "Bearer " + signWebhookToken(webhookOwner)},
			hideNotes:  want{errText: appreq.ErrScopedToken.Error()},
			updateHide: want{hiddenBy: &webhookOwner},
		},
		{
			name:       "webhook token issued without a creator",
			header:     [2]string{"Authorization", "Bearer " + signWebhookToken(0)},
			hideNotes:  want{errText: appreq.ErrScopedToken.Error()},
			updateHide: want{errText: updatenotes.NoHideActorMessage},
		},
	}

	pathSeq := 0
	newPath := func() string {
		pathSeq++
		p := "hide/" + string(rune('a'+pathSeq)) + ".md"
		_, insErr := conn.ExecContext(ctx,
			"insert into note_paths (value, value_hash, latest_content_hash) values (?, ?, '')", p, p)
		require.NoError(t, insErr)
		return p
	}

	requestCtx := func(header [2]string) context.Context {
		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("http://example.com/graphql")
		if header[0] != "" {
			reqCtx.Request.Header.Set(header[0], header[1])
		}
		return appreq.NewContext(ctx, &appreq.Request{Req: reqCtx})
	}

	check := func(t *testing.T, path string, w want, payload any) {
		t.Helper()
		if w.errText != "" {
			ep, ok := payload.(*model.ErrorPayload)
			require.True(t, ok, "expected *ErrorPayload, got %T", payload)
			require.Equal(t, w.errText, ep.Message)
			require.Nil(t, hiddenBy(ctx, t, conn, path))
			return
		}
		require.Equal(t, w.hiddenBy, hiddenBy(ctx, t, conn, path))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &hiddenByEnv{Queries: rq, wq: wq, token: tt.token}

			t.Run("hideNotes", func(t *testing.T) {
				reqCtx := requestCtx(tt.header)
				apiKey, resolveErr := checkapikey.Resolve(reqCtx, env, "hide_notes")
				require.NoError(t, resolveErr)

				path := newPath()
				input := model.HideNotesInput{Paths: []string{path}, ApiKey: *apiKey}
				payload, hideErr := hidenotes.Resolve(reqCtx, env, input)
				require.NoError(t, hideErr)
				check(t, path, tt.hideNotes, payload)
			})

			t.Run("updateNotes hide", func(t *testing.T) {
				reqCtx := requestCtx(tt.header)
				apiKey, resolveErr := checkapikey.Resolve(reqCtx, env, "update_notes")
				require.NoError(t, resolveErr)

				path := newPath()
				input := model.UpdateNotesInput{
					ApiKey:  *apiKey,
					Changes: []model.NoteChangeInput{{Hide: &model.NoteChangeHideInput{Path: path}}},
				}
				payload, hideErr := updatenotes.Resolve(reqCtx, env, input)
				require.NoError(t, hideErr)
				check(t, path, tt.updateHide, payload)
			})
		})
	}
}
