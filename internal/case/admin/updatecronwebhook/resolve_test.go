package updatecronwebhook_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/updatecronwebhook"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type mockEnv struct {
	updated      *db.UpdateCronWebhookParams
	existing     db.CronWebhook
	secretValues map[string]string
}

func (m *mockEnv) CurrentAdminUserToken(_ context.Context) (*usertoken.Data, error) {
	return &usertoken.Data{ID: 1, Role: "admin"}, nil
}

func (m *mockEnv) UpdateCronWebhook(_ context.Context, p db.UpdateCronWebhookParams) (db.CronWebhook, error) {
	m.updated = &p
	return db.CronWebhook{ID: p.ID, CronSchedule: "0 * * * *"}, nil
}

func (m *mockEnv) UpdateCronWebhookNextRunAt(_ context.Context, _ db.UpdateCronWebhookNextRunAtParams) error {
	return nil
}

func (m *mockEnv) CronWebhookByID(_ context.Context, id int64) (db.CronWebhook, error) {
	if m.existing.ID != 0 {
		return m.existing, nil
	}
	// Default safe existing state: HTTPS, pass_api_key=false, no transform.
	return db.CronWebhook{ID: id, Url: "https://example.com/hook", CronSchedule: "0 * * * *"}, nil
}

func (m *mockEnv) GetSecretValues(_ context.Context, _ string) (map[string]string, error) {
	return m.secretValues, nil
}

// F9(a): updating URL to http:// while enabling pass_api_key must be rejected.
func TestResolve_CronUpdate_RejectsHTTPWithPassAPIKey(t *testing.T) {
	env := &mockEnv{}
	out, err := updatecronwebhook.Resolve(context.Background(), env, model.UpdateCronWebhookInput{
		ID:         9,
		URL:        ptr.To("http://example.com/hook"),
		PassAPIKey: ptr.To(true),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for http+passAPIKey cron update, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when URL is insecure with pass_api_key")
}

// F9(b): enabling transform_jsonnet + pass_api_key in same update must be rejected.
func TestResolve_CronUpdate_RejectsTransformWithPassAPIKey(t *testing.T) {
	env := &mockEnv{}
	out, err := updatecronwebhook.Resolve(context.Background(), env, model.UpdateCronWebhookInput{
		ID:               9,
		TransformJsonnet: ptr.To(`{ x: 1 }`),
		PassAPIKey:       ptr.To(true),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for transform+passAPIKey cron update, got %T", out)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when transform+passAPIKey conflict")
}

// Update-path cross-field bypass (i): updating URL to http alone when existing cron webhook has
// pass_api_key=true must be rejected (the HTTPS guard must use effective merged state).
func TestResolve_CronUpdate_RejectsHTTPURL_WhenExistingPassAPIKeyTrue(t *testing.T) {
	env := &mockEnv{
		existing: db.CronWebhook{ID: 9, Url: "https://example.com/hook", PassApiKey: true, CronSchedule: "0 * * * *"},
	}
	out, err := updatecronwebhook.Resolve(context.Background(), env, model.UpdateCronWebhookInput{
		ID:  9,
		URL: ptr.To("http://example.com/hook"),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when updating cron URL to http on pass_api_key=true webhook, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when effective URL is insecure with pass_api_key")
}

// Update-path cross-field bypass (ii): enabling pass_api_key alone when existing cron webhook has
// an http URL must be rejected (the HTTPS guard must use effective merged state).
func TestResolve_CronUpdate_RejectsPassAPIKeyTrue_WhenExistingHTTPURL(t *testing.T) {
	env := &mockEnv{
		existing: db.CronWebhook{ID: 9, Url: "http://example.com/hook", CronSchedule: "0 * * * *"},
	}
	out, err := updatecronwebhook.Resolve(context.Background(), env, model.UpdateCronWebhookInput{
		ID:         9,
		PassAPIKey: ptr.To(true),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when enabling pass_api_key on http cron webhook, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when enabling pass_api_key on insecure URL")
}

// Update-path cross-field bypass (iii): enabling transform_jsonnet alone when existing cron webhook
// has pass_api_key=true must be rejected (F9(b) must use effective merged state).
func TestResolve_CronUpdate_RejectsTransformJsonnet_WhenExistingPassAPIKeyTrue(t *testing.T) {
	env := &mockEnv{
		existing: db.CronWebhook{ID: 9, Url: "https://example.com/hook", PassApiKey: true, CronSchedule: "0 * * * *"},
	}
	out, err := updatecronwebhook.Resolve(context.Background(), env, model.UpdateCronWebhookInput{
		ID:               9,
		TransformJsonnet: ptr.To(`{ x: 1 }`),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when enabling transform on pass_api_key=true cron webhook, got %T", out)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when transform+effective-pass_api_key conflict")
}

// F9(a) secrets: updating cron URL to http when secrets are attached must be rejected even
// when pass_api_key=false (decrypted secrets travel in the body independently).
func TestResolve_CronUpdate_RejectsHTTPURL_WhenSecretsAttached(t *testing.T) {
	env := &mockEnv{
		existing:     db.CronWebhook{ID: 9, Url: "https://example.com/hook", CronSchedule: "0 * * * *"},
		secretValues: map[string]string{"cron_webhooks:9:token": "secret-val"},
	}
	out, err := updatecronwebhook.Resolve(context.Background(), env, model.UpdateCronWebhookInput{
		ID:  9,
		URL: ptr.To("http://example.com/hook"),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when updating cron URL to http with secrets attached, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when URL is insecure and secrets are attached")
}

// F9(b) secrets: enabling transform_jsonnet when secrets are attached to a cron webhook
// must be rejected even when pass_api_key=false.
func TestResolve_CronUpdate_RejectsTransformJsonnet_WhenSecretsAttached(t *testing.T) {
	env := &mockEnv{
		existing:     db.CronWebhook{ID: 9, Url: "https://example.com/hook", CronSchedule: "0 * * * *"},
		secretValues: map[string]string{"cron_webhooks:9:token": "secret-val"},
	}
	out, err := updatecronwebhook.Resolve(context.Background(), env, model.UpdateCronWebhookInput{
		ID:               9,
		TransformJsonnet: ptr.To(`{ x: 1 }`),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when enabling transform with cron secrets attached, got %T", out)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
	require.Nil(t, env.updated, "must not update when transform+secrets conflict on cron webhook")
}
