package deliverchangewebhook_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"trip2g/internal/case/backjob/deliverchangewebhook"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

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
