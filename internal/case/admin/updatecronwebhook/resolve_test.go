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
	updated *db.UpdateCronWebhookParams
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
