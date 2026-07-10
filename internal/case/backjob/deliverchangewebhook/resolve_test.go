package deliverchangewebhook_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"trip2g/internal/case/backjob/deliverchangewebhook"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/webhookutil"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg deliverchangewebhook_test . Env

func baseEnv(t *testing.T, url string, secretValues map[string]string) *EnvMock {
	t.Helper()
	return &EnvMock{
		WebhookByIDFunc: func(_ context.Context, id int64) (db.ChangeWebhook, error) {
			return db.ChangeWebhook{
				ID:             id,
				Url:            url,
				TimeoutSeconds: 10,
				WritePatterns:  "[]",
				ReadPatterns:   "[]",
			}, nil
		},
		MarkWebhookDeliveryRunningFunc: func(_ context.Context, _ int64) error {
			return nil
		},
		GetSecretValuesFunc: func(_ context.Context, _ string) (map[string]string, error) {
			return secretValues, nil
		},
		UpdateWebhookDeliveryResultFunc: func(_ context.Context, _ db.UpdateWebhookDeliveryResultParams) error {
			return nil
		},
		InsertWebhookDeliveryLogFunc: func(_ context.Context, _ db.InsertWebhookDeliveryLogParams) error {
			return nil
		},
		InsertNoteFunc: func(_ context.Context, _ model.RawNote) (int64, error) {
			return 0, nil
		},
		PrepareLatestNotesFunc: func(_ context.Context, _ bool) (*model.NoteViews, error) {
			return nil, nil
		},
		EnqueueDeliverChangeWebhookFunc: func(_ context.Context, _ handlenotewebhooks.DeliverChangeWebhookParams) error {
			return nil
		},
		LatestNoteViewsFunc:     func() *model.NoteViews { return nil },
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
		"change_webhooks:1:auth_token": "tok-abc",
		"change_webhooks:1:api_key":    "key-xyz",
	})

	err := deliverchangewebhook.Resolve(context.Background(), env, handlenotewebhooks.DeliverChangeWebhookParams{
		WebhookID:  1,
		DeliveryID: 100,
		Attempt:    1,
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

func tokenLifetimeSeconds(t *testing.T, jwtStr string) int64 {
	t.Helper()
	parts := strings.Split(jwtStr, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims struct {
		Exp int64 `json:"exp"`
		Iat int64 `json:"iat"`
	}
	require.NoError(t, json.Unmarshal(raw, &claims))
	return claims.Exp - claims.Iat
}

func TestResolve_TokenTTLHasNoSixtyMinuteFloor(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{ID: id, Url: srv.URL, TimeoutSeconds: 10, PassApiKey: true, ReadPatterns: "[]", WritePatterns: "[]"}, nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env, handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 100, Attempt: 1})
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	tok, _ := payload["api_token"].(string)
	require.NotEmpty(t, tok)
	require.Less(t, tokenLifetimeSeconds(t, tok), int64(300), "60-min floor must be gone; TTL ~= timeout + margin")
}

func TestResolve_LoggedRequestBodyRedactsTokenAndSecrets(t *testing.T) {
	var httpBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, map[string]string{"auth_token": "tok-abc"})
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{ID: id, Url: srv.URL, TimeoutSeconds: 10, PassApiKey: true, ReadPatterns: "[]", WritePatterns: "[]"}, nil
	}
	var loggedBody string
	env.InsertWebhookDeliveryLogFunc = func(_ context.Context, p db.InsertWebhookDeliveryLogParams) error {
		if p.RequestBody != nil {
			loggedBody = *p.RequestBody
		}
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env, handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 100, Attempt: 1})
	require.NoError(t, err)

	// HTTP body to the fleet keeps the credential + secrets.
	require.Contains(t, string(httpBody), "api_token")
	require.Contains(t, string(httpBody), "tok-abc")
	// Logged body must not leak them.
	require.NotContains(t, loggedBody, "tok-abc")
	require.NotContains(t, loggedBody, "api_token")
}

func TestResolve_NoSecrets_FieldOmitted(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)

	err := deliverchangewebhook.Resolve(context.Background(), env, handlenotewebhooks.DeliverChangeWebhookParams{
		WebhookID:  2,
		DeliveryID: 200,
		Attempt:    1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, body)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	_, hasSecrets := payload["secrets"]
	require.False(t, hasSecrets, "secrets field should be omitted when no secrets exist")
}

func TestTransformExtVars_RedactsSecrets(t *testing.T) {
	payload := []byte(`{"changes":[{"path":"a.md"}],"attached_notes":[{"path":"b.md"}],` +
		`"api_token":"SECRET-TOKEN","secrets":{"k":"v"}}`)
	ev := deliverchangewebhook.TransformExtVarsForTest(payload)

	require.Contains(t, ev, "change")
	require.Contains(t, ev, "attached_notes")
	require.NotContains(t, ev, "api_token")
	require.NotContains(t, ev, "secrets")
	for _, v := range ev {
		require.NotContains(t, v, "SECRET-TOKEN")
	}
}

func TestResolve_AttachNotes_Materialized(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	nvs := model.NewNoteViews()
	nvs.RegisterNote(&model.NoteView{
		Path:    "boards/sprint.md",
		Title:   "Sprint",
		Content: []byte("# Sprint\n- card"),
		Tags:    []string{"kanban"},
	})

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
			ID: id, Url: srv.URL, TimeoutSeconds: 10,
			WritePatterns: "[]", ReadPatterns: "[]",
			AttachNotes: `["boards/**","!inbox/**"]`,
		}, nil
	}
	env.LatestNoteViewsFunc = func() *model.NoteViews { return nvs }

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 1, Attempt: 1})
	require.NoError(t, err)

	var payload struct {
		AttachedNotes []map[string]any `json:"attached_notes"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Len(t, payload.AttachedNotes, 1)
	require.Equal(t, "boards/sprint.md", payload.AttachedNotes[0]["path"])
	require.Equal(t, "Sprint", payload.AttachedNotes[0]["title"])
	require.Contains(t, payload.AttachedNotes[0], "content")
	require.Contains(t, payload.AttachedNotes[0], "meta")
}

func TestResolve_MarksRunningAtPickup(t *testing.T) {
	var markedID int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.MarkWebhookDeliveryRunningFunc = func(_ context.Context, id int64) error {
		markedID = id
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 555, Attempt: 1})
	require.NoError(t, err)
	require.EqualValues(t, 555, markedID)
}

func TestResolve_TransformJsonnet_AppliedAndSigned(t *testing.T) {
	var body []byte
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		gotSig = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
			ID:               id,
			Url:              srv.URL,
			Secret:           "hook-secret",
			TimeoutSeconds:   10,
			WritePatterns:    "[]",
			ReadPatterns:     "[]",
			TransformJsonnet: `{ marker: "transformed", n: std.length(std.parseJson(std.extVar("change"))) }`,
		}, nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{
			WebhookID:  1,
			DeliveryID: 100,
			Attempt:    1,
			Changes:    []handlenotewebhooks.ChangeInfo{{Path: "a.md", Event: "update"}},
		})
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "transformed", out["marker"])
	require.EqualValues(t, 1, out["n"])
	// HMAC must cover the TRANSFORMED bytes the server actually received.
	require.Equal(t, webhookutil.SignHMAC(body, "hook-secret"), gotSig)
}

// F9(d): when transform_jsonnet is active, the logged request_body must equal the
// actual transformed bytes sent to the endpoint (not the pre-transform struct).
func TestResolve_TransformJsonnet_LoggedBodyEqualsTransformedBytes(t *testing.T) {
	var httpBody []byte
	var loggedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
			ID:               id,
			Url:              srv.URL,
			Secret:           "hook-secret",
			TimeoutSeconds:   10,
			WritePatterns:    "[]",
			ReadPatterns:     "[]",
			TransformJsonnet: `{ marker: "logged", n: std.length(std.parseJson(std.extVar("change"))) }`,
		}, nil
	}
	env.InsertWebhookDeliveryLogFunc = func(_ context.Context, p db.InsertWebhookDeliveryLogParams) error {
		if p.RequestBody != nil {
			loggedBody = *p.RequestBody
		}
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{
			WebhookID:  1,
			DeliveryID: 200,
			Attempt:    1,
			Changes:    []handlenotewebhooks.ChangeInfo{{Path: "a.md", Event: "update"}},
		})
	require.NoError(t, err)
	require.NotEmpty(t, httpBody, "HTTP body must be non-empty")
	require.NotEmpty(t, loggedBody, "logged body must be non-empty")
	// The logged body must equal the bytes that were actually sent to the endpoint.
	require.Equal(t, string(httpBody), loggedBody,
		"logged request_body must match transformed bytes sent to endpoint")
}

func TestResolve_PersistsSpend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","tokens_used":321,"steps":3,"changes":[]}`))
	}))
	defer srv.Close()

	var got db.UpdateWebhookDeliveryResultParams
	env := baseEnv(t, srv.URL, nil)
	env.UpdateWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 1, Attempt: 1})
	require.NoError(t, err)
	require.Equal(t, "success", got.Status)
	require.NotNil(t, got.TokensUsed)
	require.EqualValues(t, 321, *got.TokensUsed)
	require.NotNil(t, got.Steps)
	require.EqualValues(t, 3, *got.Steps)
}

// F2: applyAgentChanges with empty write_patterns must deny all writes (was: allow all).
func TestResolve_AgentChanges_EmptyWritePatterns_Denied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[{"path":"any.md","content":"x"}]}`))
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	// WritePatterns: "[]" → parsed as empty slice → must deny.
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
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

	var got db.UpdateWebhookDeliveryResultParams
	env.UpdateWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 10, Attempt: 1})
	require.NoError(t, err)
	require.False(t, insertCalled, "InsertNote must not be called when write_patterns is empty (scoped deny-all)")
	require.Equal(t, "failed", got.Status, "delivery must be marked failed when agent tries to write with empty write_patterns")
}

// F2: applyAgentChanges with matching write_patterns still allows writes.
func TestResolve_AgentChanges_MatchingWritePatterns_Allowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[{"path":"notes/todo.md","content":"x"}]}`))
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
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
	env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*model.NoteViews, error) {
		return model.NewNoteViews(), nil
	}

	var got db.UpdateWebhookDeliveryResultParams
	env.UpdateWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 11, Attempt: 1})
	require.NoError(t, err)
	require.True(t, insertCalled, "InsertNote must be called when path matches write_patterns")
	require.Equal(t, "success", got.Status)
}

// F8: patch-kind change must apply find→replace to existing note content, not overwrite with empty string.
func TestResolve_AgentChanges_PatchKind_FindReplace(t *testing.T) {
	const noteContent = "# Todo\nhello world\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[{"path":"notes/todo.md","kind":"patch","find":"hello","replace":"HELLO"}]}`))
	}))
	defer srv.Close()

	nvs := model.NewNoteViews()
	nvs.PathMap["notes/todo.md"] = &model.NoteView{Path: "notes/todo.md", Content: []byte(noteContent)}

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
			ID:             id,
			Url:            srv.URL,
			TimeoutSeconds: 10,
			WritePatterns:  `["notes/**"]`,
			ReadPatterns:   "[]",
		}, nil
	}
	env.LatestNoteViewsFunc = func() *model.NoteViews { return nvs }
	env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*model.NoteViews, error) {
		return nvs, nil
	}

	var insertedNote model.RawNote
	env.InsertNoteFunc = func(_ context.Context, note model.RawNote) (int64, error) {
		insertedNote = note
		return 1, nil
	}

	var got db.UpdateWebhookDeliveryResultParams
	env.UpdateWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 20, Attempt: 1})
	require.NoError(t, err)
	require.Equal(t, "success", got.Status)
	require.Equal(t, "notes/todo.md", insertedNote.Path)
	require.Equal(t, "# Todo\nHELLO world\n", insertedNote.Content, "patch must replace find→replace in existing content, not empty it")
}

// P1: a change carrying a stale expected_hash must not overwrite the current note.
func TestResolve_AgentChanges_StaleExpectedHash_Rejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[{"path":"notes/todo.md","content":"clobber","expected_hash":"stale"}]}`))
	}))
	defer srv.Close()

	nvs := model.NewNoteViews()
	nvs.PathMap["notes/todo.md"] = &model.NoteView{Path: "notes/todo.md", Content: []byte("# Todo\ncurrent\n")}

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
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

	var got db.UpdateWebhookDeliveryResultParams
	env.UpdateWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 30, Attempt: 1})
	require.NoError(t, err)
	require.False(t, insertCalled, "InsertNote must not be called when expected_hash is stale")
	require.Equal(t, "failed", got.Status, "delivery must be marked failed on hash mismatch")
}

// P1: a change carrying the correct expected_hash must still apply.
func TestResolve_AgentChanges_MatchingExpectedHash_Applied(t *testing.T) {
	const noteContent = "# Todo\ncurrent\n"
	currentHash := webhookutil.HashContent([]byte(noteContent))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		resp := `{"status":"ok","changes":[{"path":"notes/todo.md","content":"updated","expected_hash":"` + currentHash + `"}]}`
		_, _ = w.Write([]byte(resp))
	}))
	defer srv.Close()

	nvs := model.NewNoteViews()
	nvs.PathMap["notes/todo.md"] = &model.NoteView{Path: "notes/todo.md", Content: []byte(noteContent)}

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
			ID:             id,
			Url:            srv.URL,
			TimeoutSeconds: 10,
			WritePatterns:  `["notes/**"]`,
			ReadPatterns:   "[]",
		}, nil
	}
	env.LatestNoteViewsFunc = func() *model.NoteViews { return nvs }
	env.PrepareLatestNotesFunc = func(_ context.Context, _ bool) (*model.NoteViews, error) { return nvs, nil }

	var insertedNote model.RawNote
	env.InsertNoteFunc = func(_ context.Context, note model.RawNote) (int64, error) {
		insertedNote = note
		return 1, nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 31, Attempt: 1})
	require.NoError(t, err)
	require.Equal(t, "notes/todo.md", insertedNote.Path)
	require.Equal(t, "updated", insertedNote.Content)
}

// P1: a mid-batch failure must not leave a partially applied prefix of the batch.
func TestResolve_AgentChanges_MidBatchFailure_NoPartialApply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[` +
			`{"path":"notes/a.md","content":"a"},` +
			`{"path":"outside.md","content":"b"}]}`))
	}))
	defer srv.Close()

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
			ID:             id,
			Url:            srv.URL,
			TimeoutSeconds: 10,
			MaxRetries:     1,
			WritePatterns:  `["notes/**"]`,
			ReadPatterns:   "[]",
		}, nil
	}

	insertCalled := false
	env.InsertNoteFunc = func(_ context.Context, _ model.RawNote) (int64, error) {
		insertCalled = true
		return 0, nil
	}

	var got db.UpdateWebhookDeliveryResultParams
	env.UpdateWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 32, Attempt: 1})
	require.NoError(t, err)
	require.False(t, insertCalled, "no change may be applied when any change in the batch is invalid")
	require.Equal(t, "failed", got.Status)
}

// F8: patch-kind change with a find string absent from the note must error without writing.
func TestResolve_AgentChanges_PatchKind_FindMissing_Error(t *testing.T) {
	const noteContent = "# Todo\nhello world\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","changes":[{"path":"notes/todo.md","kind":"patch","find":"MISSING","replace":"X"}]}`))
	}))
	defer srv.Close()

	nvs := model.NewNoteViews()
	nvs.PathMap["notes/todo.md"] = &model.NoteView{Path: "notes/todo.md", Content: []byte(noteContent)}

	env := baseEnv(t, srv.URL, nil)
	env.WebhookByIDFunc = func(_ context.Context, id int64) (db.ChangeWebhook, error) {
		return db.ChangeWebhook{
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

	var got db.UpdateWebhookDeliveryResultParams
	env.UpdateWebhookDeliveryResultFunc = func(_ context.Context, arg db.UpdateWebhookDeliveryResultParams) error {
		got = arg
		return nil
	}

	err := deliverchangewebhook.Resolve(context.Background(), env,
		handlenotewebhooks.DeliverChangeWebhookParams{WebhookID: 1, DeliveryID: 21, Attempt: 1})
	require.NoError(t, err)
	require.False(t, insertCalled, "InsertNote must not be called when patch find string is absent")
	require.Equal(t, "failed", got.Status, "delivery must be marked failed when patch find is missing")
}
