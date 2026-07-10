package delivercronwebhook_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"trip2g/internal/case/backjob/delivercronwebhook"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/webhookutil"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg delivercronwebhook_test . Env

func baseEnv(t *testing.T, url string, secretValues map[string]string) *EnvMock {
	t.Helper()
	return &EnvMock{
		CronWebhookByIDFunc: func(_ context.Context, id int64) (db.CronWebhook, error) {
			return db.CronWebhook{
				ID:             id,
				Url:            url,
				TimeoutSeconds: 10,
				WritePatterns:  "[]",
				ReadPatterns:   "[]",
			}, nil
		},
		MarkCronWebhookDeliveryRunningFunc: func(_ context.Context, _ int64) error {
			return nil
		},
		GetSecretValuesFunc: func(_ context.Context, _ string) (map[string]string, error) {
			return secretValues, nil
		},
		UpdateCronWebhookDeliveryResultFunc: func(_ context.Context, _ db.UpdateCronWebhookDeliveryResultParams) error {
			return nil
		},
		InsertWebhookDeliveryLogFunc: func(_ context.Context, _ db.InsertWebhookDeliveryLogParams) error {
			return nil
		},
		InsertNoteFunc: func(_ context.Context, _ model.RawNote) (int64, error) {
			return 0, nil
		},
		LatestNoteViewsFunc: func() *model.NoteViews { return nil },
		EnqueueDeliverCronWebhookFunc: func(_ context.Context, _ delivercronwebhook.DeliverCronParams) error {
			return nil
		},
		ShortAPITokenSecretFunc: func() string { return "test-secret" },
		WebhookHTTPClientFunc:   func() *fasthttp.Client { return &fasthttp.Client{} },
		LoggerFunc:              func() logger.Logger { return &logger.DummyLogger{} },
	}
}

func TestResolve_SecretsInjectedInPayload(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, map[string]string{
		"cron_webhooks:5:auth_token": "tok-abc",
		"cron_webhooks:5:api_key":    "key-xyz",
	})

	err := delivercronwebhook.Resolve(context.Background(), env, delivercronwebhook.DeliverCronParams{
		CronWebhookID: 5,
		DeliveryID:    500,
		Attempt:       1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, body)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))

	secrets, ok := payload["secrets"].(map[string]any)
	require.True(t, ok, "expected secrets field in payload")
	require.Equal(t, "tok-abc", secrets["auth_token"])
	require.Equal(t, "key-xyz", secrets["api_key"])
}

func TestResolve_NoSecrets_FieldOmitted(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)

	err := delivercronwebhook.Resolve(context.Background(), env, delivercronwebhook.DeliverCronParams{
		CronWebhookID: 6,
		DeliveryID:    600,
		Attempt:       1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, body)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	_, hasSecrets := payload["secrets"]
	require.False(t, hasSecrets, "secrets field should be omitted when no secrets exist")
}

// Cron webhooks share the scoped token path with change webhooks. Invalid JSON
// must not silently widen the token to read every note.
func TestResolve_MalformedReadPatternsFailClosed(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID: id, Url: srv.URL, TimeoutSeconds: 10, PassApiKey: true,
			ReadPatterns: `{malformed`, WritePatterns: `[]`,
		}, nil
	}

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 601, Attempt: 1})
	require.NoError(t, err)

	var payload struct {
		APIToken string `json:"api_token"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	require.NotEmpty(t, payload.APIToken)
	claims, err := shortapitoken.Parse(payload.APIToken, "test-secret")
	require.NoError(t, err)
	require.Empty(t, claims.ReadPatterns,
		"malformed read_patterns must not become an all-notes scope")
}

// F2: applyCronAgentChanges with empty write_patterns must deny all writes (was: allow all).
func TestResolve_CronAgentChanges_EmptyWritePatterns_Denied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[{"path":"any.md","content":"x"}]}`))
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	// WritePatterns: "[]" → parsed as empty slice → must deny.
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID:             id,
			Url:            srv.URL,
			TimeoutSeconds: 10,
			MaxRetries:     1,
			WritePatterns:  "[]",
			ReadPatterns:   "[]",
		}, nil
	}
	insertCalled := false
	env.InsertNoteFunc = func(_ context.Context, _ model.RawNote) (int64, error) {
		insertCalled = true
		return 0, nil
	}

	var got db.UpdateCronWebhookDeliveryResultParams
	env.UpdateCronWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateCronWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 10, Attempt: 1})
	require.NoError(t, err)
	require.False(t, insertCalled, "InsertNote must not be called when write_patterns is empty (scoped deny-all)")
	require.Equal(t, "failed", got.Status, "delivery must be marked failed when agent tries to write with empty write_patterns")
}

// F2: applyCronAgentChanges with matching write_patterns still allows writes.
func TestResolve_CronAgentChanges_MatchingWritePatterns_Allowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[{"path":"notes/todo.md","content":"x"}]}`))
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID:             id,
			Url:            srv.URL,
			TimeoutSeconds: 10,
			WritePatterns:  `["notes/**"]`,
			ReadPatterns:   "[]",
		}, nil
	}
	insertCalled := false
	env.InsertNoteFunc = func(_ context.Context, _ model.RawNote) (int64, error) {
		insertCalled = true
		return 1, nil
	}

	var got db.UpdateCronWebhookDeliveryResultParams
	env.UpdateCronWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateCronWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 11, Attempt: 1})
	require.NoError(t, err)
	require.True(t, insertCalled, "InsertNote must be called when path matches write_patterns")
	require.Equal(t, "success", got.Status)
}

// F9(d): when transform_jsonnet is active on a cron webhook, the logged request_body must
// equal the actual transformed bytes sent to the endpoint (not the pre-transform struct).
func TestResolve_CronTransformJsonnet_LoggedBodyEqualsTransformedBytes(t *testing.T) {
	var httpBody []byte
	var loggedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID:               id,
			Url:              srv.URL,
			Secret:           "cron-secret",
			TimeoutSeconds:   10,
			WritePatterns:    "[]",
			ReadPatterns:     "[]",
			TransformJsonnet: `{ marker: "cron-logged" }`,
		}, nil
	}
	env.InsertWebhookDeliveryLogFunc = func(_ context.Context, p db.InsertWebhookDeliveryLogParams) error {
		if p.RequestBody != nil {
			loggedBody = *p.RequestBody
		}
		return nil
	}

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 300, Attempt: 1})
	require.NoError(t, err)
	require.NotEmpty(t, httpBody, "HTTP body must be non-empty")
	require.NotEmpty(t, loggedBody, "logged body must be non-empty")
	// The logged body must equal the bytes that were actually sent to the endpoint.
	require.Equal(t, string(httpBody), loggedBody,
		"logged request_body must match transformed bytes sent to endpoint")
}

func TestResolve_CronTransformJsonnet_AppliedAndSigned(t *testing.T) {
	var body []byte
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		gotSig = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil) // reuse the package's existing baseEnv helper
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID:               id,
			Url:              srv.URL,
			Secret:           "cron-secret",
			TimeoutSeconds:   10,
			WritePatterns:    "[]",
			ReadPatterns:     "[]",
			TransformJsonnet: `{ marker: "cron-transformed" }`,
		}, nil
	}

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 7, Attempt: 1})
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "cron-transformed", out["marker"])
	require.Equal(t, webhookutil.SignHMAC(body, "cron-secret"), gotSig)
}

// F8: patch-kind change must apply find→replace to existing note content, not overwrite with empty string.
func TestResolve_CronAgentChanges_PatchKind_FindReplace(t *testing.T) {
	const noteContent = "# Cron\nhello world\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[{"path":"notes/cron.md","kind":"patch","find":"hello","replace":"HELLO"}]}`))
	}))
	defer srv.Close()

	nvs := model.NewNoteViews()
	nvs.PathMap["notes/cron.md"] = &model.NoteView{Path: "notes/cron.md", Content: []byte(noteContent)}

	env := baseEnv(t, srv.URL, nil)
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID:             id,
			Url:            srv.URL,
			TimeoutSeconds: 10,
			WritePatterns:  `["notes/**"]`,
			ReadPatterns:   "[]",
		}, nil
	}
	env.LatestNoteViewsFunc = func() *model.NoteViews { return nvs }

	var insertedNote model.RawNote
	env.InsertNoteFunc = func(_ context.Context, note model.RawNote) (int64, error) {
		insertedNote = note
		return 1, nil
	}

	var got db.UpdateCronWebhookDeliveryResultParams
	env.UpdateCronWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateCronWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 20, Attempt: 1})
	require.NoError(t, err)
	require.Equal(t, "success", got.Status)
	require.Equal(t, "notes/cron.md", insertedNote.Path)
	require.Equal(t, "# Cron\nHELLO world\n", insertedNote.Content, "patch must replace find→replace in existing content, not empty it")
}

// F8: patch-kind change with a find string absent from the note must error without writing.
func TestResolve_CronAgentChanges_PatchKind_FindMissing_Error(t *testing.T) {
	const noteContent = "# Cron\nhello world\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[{"path":"notes/cron.md","kind":"patch","find":"MISSING","replace":"X"}]}`))
	}))
	defer srv.Close()

	nvs := model.NewNoteViews()
	nvs.PathMap["notes/cron.md"] = &model.NoteView{Path: "notes/cron.md", Content: []byte(noteContent)}

	env := baseEnv(t, srv.URL, nil)
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID:             id,
			Url:            srv.URL,
			TimeoutSeconds: 10,
			MaxRetries:     1,
			WritePatterns:  `["notes/**"]`,
			ReadPatterns:   "[]",
		}, nil
	}
	env.LatestNoteViewsFunc = func() *model.NoteViews { return nvs }

	insertCalled := false
	env.InsertNoteFunc = func(_ context.Context, _ model.RawNote) (int64, error) {
		insertCalled = true
		return 0, nil
	}

	var got db.UpdateCronWebhookDeliveryResultParams
	env.UpdateCronWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateCronWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 21, Attempt: 1})
	require.NoError(t, err)
	require.False(t, insertCalled, "InsertNote must not be called when patch find string is absent")
	require.Equal(t, "failed", got.Status, "delivery must be marked failed when patch find is missing")
}

// G4: cron webhook with attach_notes must carry matched notes in the delivery payload.
func TestResolve_CronAttachNotes_Materialized(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	nvs := model.NewNoteViews()
	nvs.PathMap["boards/sprint.md"] = &model.NoteView{
		Path:    "boards/sprint.md",
		Title:   "Sprint",
		Content: []byte("# Sprint\n- card"),
		Tags:    []string{"kanban"},
	}

	env := baseEnv(t, srv.URL, nil)
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID:             id,
			Url:            srv.URL,
			TimeoutSeconds: 10,
			WritePatterns:  "[]",
			ReadPatterns:   "[]",
			AttachNotes:    `["boards/**"]`,
		}, nil
	}
	env.LatestNoteViewsFunc = func() *model.NoteViews { return nvs }

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 40, Attempt: 1})
	require.NoError(t, err)
	require.NotEmpty(t, body, "HTTP body must be non-empty")

	var payload struct {
		AttachedNotes []map[string]any `json:"attached_notes"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Len(t, payload.AttachedNotes, 1, "expected exactly one attached note")
	require.Equal(t, "boards/sprint.md", payload.AttachedNotes[0]["path"])
	require.Equal(t, "Sprint", payload.AttachedNotes[0]["title"])
	require.Contains(t, payload.AttachedNotes[0], "content")
}

// G4: cron attach_notes gate must skip delivery (no HTTP call) when no notes match.
func TestResolve_CronAttachNotes_GateSkipWhenNoneMatch(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.CronWebhookByIDFunc = func(_ context.Context, id int64) (db.CronWebhook, error) {
		return db.CronWebhook{
			ID:             id,
			Url:            srv.URL,
			TimeoutSeconds: 10,
			WritePatterns:  "[]",
			ReadPatterns:   "[]",
			AttachNotes:    `["x/**"]`,
		}, nil
	}
	// Empty NoteViews — no notes match "x/**".
	env.LatestNoteViewsFunc = model.NewNoteViews

	var got db.UpdateCronWebhookDeliveryResultParams
	env.UpdateCronWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateCronWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := delivercronwebhook.Resolve(context.Background(), env,
		delivercronwebhook.DeliverCronParams{CronWebhookID: 1, DeliveryID: 41, Attempt: 1})
	require.NoError(t, err)
	require.False(t, called, "HTTP endpoint must not be called when attach_notes gate is not satisfied")
	require.Equal(t, "success", got.Status, "delivery must be marked success (skipped) when gate not satisfied")
}
