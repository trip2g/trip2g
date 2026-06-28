package createcronwebhook_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/createcronwebhook"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type mockEnv struct {
	inserted *db.InsertCronWebhookParams
}

func (m *mockEnv) CurrentAdminUserToken(_ context.Context) (*usertoken.Data, error) {
	return &usertoken.Data{ID: 1, Role: "admin"}, nil
}

func (m *mockEnv) InsertCronWebhook(_ context.Context, p db.InsertCronWebhookParams) (db.CronWebhook, error) {
	m.inserted = &p
	return db.CronWebhook{ID: 9}, nil
}

// F9(a): http:// URL with pass_api_key must be rejected on cron webhook create.
func TestResolve_CronCreate_RejectsHTTPWithPassAPIKey(t *testing.T) {
	env := &mockEnv{}
	out, err := createcronwebhook.Resolve(context.Background(), env, model.CreateCronWebhookInput{
		URL:          "http://example.com/hook",
		CronSchedule: "0 * * * *",
		PassAPIKey:   ptr.To(true),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for http+passAPIKey cron create, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.inserted, "must not insert insecure cron webhook")
}

// F9(b): transform_jsonnet + pass_api_key must be rejected on cron webhook create.
func TestResolve_CronCreate_RejectsTransformWithPassAPIKey(t *testing.T) {
	env := &mockEnv{}
	out, err := createcronwebhook.Resolve(context.Background(), env, model.CreateCronWebhookInput{
		URL:              "https://example.com/hook",
		CronSchedule:     "0 * * * *",
		PassAPIKey:       ptr.To(true),
		TransformJsonnet: ptr.To(`{ x: 1 }`),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for transform+passAPIKey cron create, got %T", out)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
	require.Nil(t, env.inserted, "must not insert when transform+passAPIKey conflict")
}
