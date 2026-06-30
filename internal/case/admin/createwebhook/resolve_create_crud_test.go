package createwebhook_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/admin/createwebhook"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type mockEnv struct {
	inserted *db.InsertWebhookParams
}

func (m *mockEnv) CurrentAdminUserToken(_ context.Context) (*usertoken.Data, error) {
	return &usertoken.Data{ID: 1, Role: "admin"}, nil
}

func (m *mockEnv) IsDevMode() bool { return false }

func (m *mockEnv) InsertWebhook(_ context.Context, params db.InsertWebhookParams) (db.ChangeWebhook, error) {
	m.inserted = &params
	return db.ChangeWebhook{ID: 7}, nil
}

func TestResolve_PersistsConcurrencyAndAttach(t *testing.T) {
	env := &mockEnv{}
	out, err := createwebhook.Resolve(context.Background(), env, model.ChangeWebhookCreateInput{
		URL:              "https://example.com/h",
		IncludePatterns:  []string{"boards/sprint.md"},
		AttachNotes:      []string{"boards/**", "roles/**"},
		TransformJsonnet: ptr.To("{ x: 1 }"),
		ConcurrencyMode:  ptr.To("skip"),
	})
	require.NoError(t, err)
	_, isErr := out.(*model.ErrorPayload)
	require.False(t, isErr)
	require.NotNil(t, env.inserted)
	require.Equal(t, "skip", env.inserted.ConcurrencyMode)
	require.Equal(t, `["boards/**","roles/**"]`, env.inserted.AttachNotes)
	require.Equal(t, "{ x: 1 }", env.inserted.TransformJsonnet)
}

func TestResolve_RejectsBadConcurrencyMode(t *testing.T) {
	env := &mockEnv{}
	out, err := createwebhook.Resolve(context.Background(), env, model.ChangeWebhookCreateInput{
		URL:             "https://example.com/h",
		IncludePatterns: []string{"boards/sprint.md"},
		ConcurrencyMode: ptr.To("bogus"),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok)
	require.Equal(t, "concurrencyMode", ep.ByFields[0].Name)
	require.Nil(t, env.inserted, "must not insert on invalid concurrency_mode")
}

func TestResolve_InvalidTransformJsonnet_ReturnsErrorPayload(t *testing.T) {
	env := &mockEnv{}
	bad := "}{ not jsonnet"
	in := model.ChangeWebhookCreateInput{
		URL:              "https://example.com/hook",
		IncludePatterns:  []string{"**"},
		TransformJsonnet: &bad,
	}
	payload, err := createwebhook.Resolve(context.Background(), env, in)
	require.NoError(t, err) // validation error -> (ErrorPayload, nil)

	ep, ok := payload.(*model.ErrorPayload)
	require.True(t, ok, "expected *model.ErrorPayload, got %T", payload)
	require.NotEmpty(t, ep.ByFields)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
	require.Nil(t, env.inserted, "must not insert on invalid transform_jsonnet")
}

// F9(a): http:// URL with pass_api_key must be rejected at create time.
func TestResolve_RejectsHTTPWithPassAPIKey(t *testing.T) {
	env := &mockEnv{}
	out, err := createwebhook.Resolve(context.Background(), env, model.ChangeWebhookCreateInput{
		URL:             "http://example.com/hook",
		IncludePatterns: []string{"**"},
		PassAPIKey:      ptr.To(true),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for http+passAPIKey, got %T", out)
	require.Equal(t, "url", ep.ByFields[0].Name)
	require.Nil(t, env.inserted, "must not insert when URL is insecure")
}

// F9(a): https:// URL with pass_api_key must be accepted.
func TestResolve_AcceptsHTTPSWithPassAPIKey(t *testing.T) {
	env := &mockEnv{}
	out, err := createwebhook.Resolve(context.Background(), env, model.ChangeWebhookCreateInput{
		URL:             "https://example.com/hook",
		IncludePatterns: []string{"**"},
		PassAPIKey:      ptr.To(true),
	})
	require.NoError(t, err)
	_, isErr := out.(*model.ErrorPayload)
	require.False(t, isErr, "https+passAPIKey must be accepted")
	require.NotNil(t, env.inserted)
}

// F9(b): transform_jsonnet + pass_api_key must be rejected at create time.
func TestResolve_RejectsTransformWithPassAPIKey(t *testing.T) {
	env := &mockEnv{}
	out, err := createwebhook.Resolve(context.Background(), env, model.ChangeWebhookCreateInput{
		URL:              "https://example.com/hook",
		IncludePatterns:  []string{"**"},
		PassAPIKey:       ptr.To(true),
		TransformJsonnet: ptr.To(`{ x: 1 }`),
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for transform+passAPIKey, got %T", out)
	require.Equal(t, "transformJsonnet", ep.ByFields[0].Name)
	require.Nil(t, env.inserted, "must not insert when transform+passAPIKey conflict")
}
