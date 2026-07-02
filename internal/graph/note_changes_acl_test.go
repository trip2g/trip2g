package graph

// Tests for per-subscriber ACL on the noteChanges subscription:
//
//   noteChangesAuth     — who may subscribe, and which credential kinds bypass
//                         per-note ACL (only real instance API keys do)
//   filterChangesByACL  — every emitted change is gated by CanReadNote (the
//                         same primitive as the MCP read path); fail-closed
//
// Vuln being fixed: noteChanges was a firehose — the stream was filtered only
// by the caller's glob patterns, so any credential holder (incl. a scoped
// webhook shortapitoken with a "**" include) received change events for notes
// it could not read. Ordinary signed-in users were rejected outright
// (ErrMissingKey) instead of getting an ACL-scoped stream.

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/case/checkapikey"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/notebus"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/usertoken"
)

const testShortAPISecret = "test-secret"

// authCtx returns a context carrying an appreq built from a fasthttp request
// with the given headers and pre-set user token (nil = anonymous).
func authCtx(token *usertoken.Data, headers map[string]string) context.Context {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.SetRequestURI("http://example.com/graphql")
	for k, v := range headers {
		fctx.Request.Header.Set(k, v)
	}
	req := &appreq.Request{Req: fctx}
	req.SetUserToken(token)
	req.StoreInContext()
	return fctx
}

func authEnvMock(token *usertoken.Data, keyByValue func(value string) (db.ApiKey, error)) *noteChangesAuthEnvMock {
	return &noteChangesAuthEnvMock{
		CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) { return token, nil },
		ApiKeyByValueFunc: func(ctx context.Context, value string) (db.ApiKey, error) {
			if keyByValue == nil {
				return db.ApiKey{}, sql.ErrNoRows
			}
			return keyByValue(value)
		},
		InsertAPIKeyLogFunc:       func(ctx context.Context, arg db.InsertAPIKeyLogParams) error { return nil },
		UpsertAPIKeyLogActionFunc: func(ctx context.Context, name string) error { return nil },
		UpsertAPIKeyLogIPFunc:     func(ctx context.Context, ip string) error { return nil },
		ShortAPITokenSecretFunc:   func() string { return testShortAPISecret },
	}
}

func TestNoteChangesAuth(t *testing.T) {
	webhookToken, err := shortapitoken.Sign(shortapitoken.Data{
		ReadPatterns: []string{"boards/**"},
	}, testShortAPISecret, time.Hour)
	require.NoError(t, err)

	tests := []struct {
		name          string
		token         *usertoken.Data
		headers       map[string]string
		keyByValue    func(value string) (db.ApiKey, error)
		wantBypassACL bool
		wantErr       error
	}{
		{
			name:          "admin session token subscribes, does not bypass ACL",
			token:         &usertoken.Data{ID: 1, Role: "admin"},
			wantBypassACL: false,
		},
		{
			name:    "instance API key bypasses per-note ACL",
			headers: map[string]string{"X-API-Key": "instance-key"},
			keyByValue: func(value string) (db.ApiKey, error) {
				return db.ApiKey{ID: 7}, nil
			},
			wantBypassACL: true,
		},
		{
			name:          "webhook shortapitoken subscribes, does not bypass ACL",
			headers:       map[string]string{"Authorization": "Bearer " + webhookToken},
			wantBypassACL: false,
		},
		{
			name:          "personal user token subscribes, does not bypass ACL",
			token:         &usertoken.Data{ID: 5, Role: "user"},
			wantBypassACL: false,
		},
		{
			name:    "anonymous is rejected",
			wantErr: checkapikey.ErrMissingKey,
		},
		{
			name:    "invalid API key is rejected",
			headers: map[string]string{"X-API-Key": "bogus"},
			wantErr: checkapikey.ErrInvalidKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := authCtx(tt.token, tt.headers)
			env := authEnvMock(tt.token, tt.keyByValue)

			bypassACL, authErr := noteChangesAuth(ctx, env)

			if tt.wantErr != nil {
				require.ErrorIs(t, authErr, tt.wantErr)
				return
			}
			require.NoError(t, authErr)
			require.Equal(t, tt.wantBypassACL, bypassACL)
		})
	}
}

func TestNoteChangesAuth_WebhookTokenStaysScoped(t *testing.T) {
	// Regression: a webhook shortapitoken must keep its read_patterns scoping —
	// noteChangesAuth must leave the appreq scoped so CanReadNote enforces the
	// patterns on every emitted change.
	webhookToken, err := shortapitoken.Sign(shortapitoken.Data{
		ReadPatterns: []string{"boards/**"},
	}, testShortAPISecret, time.Hour)
	require.NoError(t, err)

	ctx := authCtx(nil, map[string]string{"Authorization": "Bearer " + webhookToken})
	env := authEnvMock(nil, nil)

	bypassACL, err := noteChangesAuth(ctx, env)
	require.NoError(t, err)
	require.False(t, bypassACL)

	require.True(t, appreq.Scoped(ctx))
	require.Equal(t, []string{"boards/**"}, appreq.WebhookReadPatterns(ctx))
}

// aclNvs builds a NoteViews cache with notes in subgraphs "a" and "b".
func aclNvs(notes ...*appmodel.NoteView) *appmodel.NoteViews {
	nvs := appmodel.NewNoteViews()
	for _, n := range notes {
		nvs.Map[n.Permalink] = n
		nvs.PathMap[n.Path] = n
	}
	return nvs
}

func aclNote(pathID int64, path, permalink string, subgraphs ...string) *appmodel.NoteView {
	return &appmodel.NoteView{
		Path:          path,
		Permalink:     permalink,
		PathID:        pathID,
		SubgraphNames: subgraphs,
	}
}

// canReadSubgraphA allows notes that belong to subgraph "a" only — simulating
// a subscriber with access to A but not B.
func canReadSubgraphA(_ context.Context, note *appmodel.NoteView) (bool, error) {
	return slices.Contains(note.SubgraphNames, "a"), nil
}

func aclEnv(nvs *appmodel.NoteViews, canRead func(context.Context, *appmodel.NoteView) (bool, error)) *noteChangesACLEnvMock {
	return &noteChangesACLEnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:     canRead,
		LoggerFunc:          func() logger.Logger { return &logger.DummyLogger{} },
	}
}

func TestFilterChangesByACL(t *testing.T) {
	nvs := aclNvs(
		aclNote(1, "a/one.md", "/a-one", "a"),
		aclNote(2, "b/secret.md", "/b-secret", "b"),
		aclNote(3, "a/two.md", "/a-two", "a"),
	)

	changes := []notebus.Change{
		{PathID: 1, Path: "a/one.md", Event: "create"},
		{PathID: 2, Path: "b/secret.md", Event: "update"},
		{PathID: 3, Path: "a/two.md", Event: "remove"},
	}

	t.Run("subscriber with access to A but not B receives only A changes", func(t *testing.T) {
		// The glob include is "**" upstream — the ACL must still drop B.
		got := filterChangesByACL(context.Background(), aclEnv(nvs, canReadSubgraphA), changes)

		require.Equal(t, []notebus.Change{
			{PathID: 1, Path: "a/one.md", Event: "create"},
			{PathID: 3, Path: "a/two.md", Event: "remove"},
		}, got)
	})

	t.Run("admin-like subscriber (CanReadNote always true) receives all changes", func(t *testing.T) {
		alwaysTrue := func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil }

		got := filterChangesByACL(context.Background(), aclEnv(nvs, alwaysTrue), changes)

		require.Equal(t, changes, got)
	})

	t.Run("change with unresolvable note is dropped (fail-closed)", func(t *testing.T) {
		alwaysTrue := func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil }
		unknown := []notebus.Change{{PathID: 99, Path: "gone/nowhere.md", Event: "remove"}}

		got := filterChangesByACL(context.Background(), aclEnv(nvs, alwaysTrue), unknown)

		require.Empty(t, got)
	})

	t.Run("stale PathID falls back to path lookup", func(t *testing.T) {
		alwaysTrue := func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil }
		stale := []notebus.Change{{PathID: 99, Path: "a/one.md", Event: "update"}}

		got := filterChangesByACL(context.Background(), aclEnv(nvs, alwaysTrue), stale)

		require.Equal(t, stale, got)
	})

	t.Run("CanReadNote error drops the change (fail-closed)", func(t *testing.T) {
		failing := func(context.Context, *appmodel.NoteView) (bool, error) {
			return true, errors.New("db down")
		}

		got := filterChangesByACL(context.Background(), aclEnv(nvs, failing), changes)

		require.Empty(t, got)
	})

	t.Run("nil NoteViews drops everything (fail-closed)", func(t *testing.T) {
		alwaysTrue := func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil }

		got := filterChangesByACL(context.Background(), aclEnv(nil, alwaysTrue), changes)

		require.Empty(t, got)
	})

	t.Run("input slice is not mutated and result does not alias it", func(t *testing.T) {
		input := slices.Clone(changes)

		got := filterChangesByACL(context.Background(), aclEnv(nvs, canReadSubgraphA), input)

		require.Equal(t, changes, input)
		require.NotEmpty(t, got)
		got[0].Path = "mutated"
		require.Equal(t, changes, input)
	})
}
