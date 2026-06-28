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
