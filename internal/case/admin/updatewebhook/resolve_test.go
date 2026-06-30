package updatewebhook_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/updatewebhook"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type mockEnv struct {
	updated      *db.UpdateWebhookParams
	existing     db.ChangeWebhook
	secretValues map[string]string
}

func (m *mockEnv) CurrentAdminUserToken(_ context.Context) (*usertoken.Data, error) {
	return &usertoken.Data{ID: 1, Role: "admin"}, nil
}

func (m *mockEnv) IsDevMode() bool { return false }

func (m *mockEnv) UpdateWebhook(_ context.Context, p db.UpdateWebhookParams) (db.ChangeWebhook, error) {
	m.updated = &p
	return db.ChangeWebhook{ID: p.ID}, nil
}

func (m *mockEnv) WebhookByID(_ context.Context, id int64) (db.ChangeWebhook, error) {
	if m.existing.ID != 0 {
		return m.existing, nil
	}
	// Default safe existing state: HTTPS, pass_api_key=false, no transform.
	return db.ChangeWebhook{ID: id, Url: "https://example.com/hook"}, nil
}

func (m *mockEnv) GetSecretValues(_ context.Context, _ string) (map[string]string, error) {
	return m.secretValues, nil
}

func TestResolve_UpdatesConcurrencyAndAttach(t *testing.T) {
	env := &mockEnv{}
	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
		ID:               7,
		AttachNotes:      []string{"boards/**"},
		TransformJsonnet: ptr.To("{ y: 2 }"),
		ConcurrencyMode:  ptr.To("queue_one"),
	})
	require.NoError(t, err)
	_, isErr := out.(*model.ErrorPayload)
	require.False(t, isErr)
	require.NotNil(t, env.updated.ConcurrencyMode)
	require.Equal(t, "queue_one", *env.updated.ConcurrencyMode)
	require.NotNil(t, env.updated.AttachNotes)
	require.Equal(t, `["boards/**"]`, *env.updated.AttachNotes)
	require.NotNil(t, env.updated.TransformJsonnet)
	require.Equal(t, "{ y: 2 }", *env.updated.TransformJsonnet)
}

func TestResolve_RejectsBadConcurrencyMode(t *testing.T) {
	env := &mockEnv{}
	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
		ID:              7,
		ConcurrencyMode: ptr.To("nope"),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok)
	require.Equal(t, "concurrencyMode", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update on invalid concurrency_mode")
}

// F9(a): updating URL to http:// while enabling pass_api_key must be rejected.
func TestResolve_RejectsHTTPWithPassAPIKey(t *testing.T) {
	env := &mockEnv{}
	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
		ID:         7,
		URL:        ptr.To("http://example.com/hook"),
		PassAPIKey: ptr.To(true),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for http+passAPIKey update, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when URL is insecure with pass_api_key")
}

// F9(b): enabling transform_jsonnet + pass_api_key in same update must be rejected.
func TestResolve_RejectsTransformWithPassAPIKey(t *testing.T) {
	env := &mockEnv{}
	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
		ID:               7,
		TransformJsonnet: ptr.To(`{ x: 1 }`),
		PassAPIKey:       ptr.To(true),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for transform+passAPIKey update, got %T", out)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when transform+passAPIKey conflict")
}

// Update-path cross-field bypass (i): updating URL to http alone when existing webhook has
// pass_api_key=true must be rejected (the HTTPS guard must use effective merged state).
func TestResolve_RejectsHTTPURL_WhenExistingPassAPIKeyTrue(t *testing.T) {
	env := &mockEnv{
		existing: db.ChangeWebhook{ID: 7, Url: "https://example.com/hook", PassApiKey: true},
	}
	// Input provides only URL — no PassAPIKey field.
	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
		ID:  7,
		URL: ptr.To("http://example.com/hook"),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when updating URL to http on pass_api_key=true webhook, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when effective URL is insecure with pass_api_key")
}

// Update-path cross-field bypass (ii): enabling pass_api_key alone when existing webhook has
// an http URL must be rejected (the HTTPS guard must use effective merged state).
func TestResolve_RejectsPassAPIKeyTrue_WhenExistingHTTPURL(t *testing.T) {
	env := &mockEnv{
		existing: db.ChangeWebhook{ID: 7, Url: "http://example.com/hook"},
	}
	// Input provides only PassAPIKey — no URL field.
	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
		ID:         7,
		PassAPIKey: ptr.To(true),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when enabling pass_api_key on http webhook, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when enabling pass_api_key on insecure URL")
}

// Update-path cross-field bypass (iii): enabling transform_jsonnet alone when existing webhook
// has pass_api_key=true must be rejected (F9(b) must use effective merged state).
func TestResolve_RejectsTransformJsonnet_WhenExistingPassAPIKeyTrue(t *testing.T) {
	env := &mockEnv{
		existing: db.ChangeWebhook{ID: 7, Url: "https://example.com/hook", PassApiKey: true},
	}
	// Input provides only TransformJsonnet — no PassAPIKey field.
	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
		ID:               7,
		TransformJsonnet: ptr.To(`{ x: 1 }`),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when enabling transform on pass_api_key=true webhook, got %T", out)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when transform+effective-pass_api_key conflict")
}

// F9(a) secrets: updating URL to http when secrets are attached must be rejected even
// when pass_api_key=false (decrypted secrets travel in the body independently).
func TestResolve_RejectsHTTPURL_WhenSecretsAttached(t *testing.T) {
	env := &mockEnv{
		existing:     db.ChangeWebhook{ID: 7, Url: "https://example.com/hook"},
		secretValues: map[string]string{"change_webhooks:7:token": "secret-val"},
	}
	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
		ID:  7,
		URL: ptr.To("http://example.com/hook"),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when updating URL to http with secrets attached, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when URL is insecure and secrets are attached")
}

// F9(b) secrets: enabling transform_jsonnet when secrets are attached must be rejected even
// when pass_api_key=false (transform output drops secrets from the body silently).
func TestResolve_RejectsTransformJsonnet_WhenSecretsAttached(t *testing.T) {
	env := &mockEnv{
		existing:     db.ChangeWebhook{ID: 7, Url: "https://example.com/hook"},
		secretValues: map[string]string{"change_webhooks:7:token": "secret-val"},
	}
	out, err := updatewebhook.Resolve(context.Background(), env, model.ChangeWebhookUpdateInput{
		ID:               7,
		TransformJsonnet: ptr.To(`{ x: 1 }`),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when enabling transform with secrets attached, got %T", out)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when transform+secrets conflict")
}
