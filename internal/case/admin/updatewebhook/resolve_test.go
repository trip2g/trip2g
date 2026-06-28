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

type mockEnv struct{ updated *db.UpdateWebhookParams }

func (m *mockEnv) CurrentAdminUserToken(_ context.Context) (*usertoken.Data, error) {
	return &usertoken.Data{ID: 1, Role: "admin"}, nil
}

func (m *mockEnv) UpdateWebhook(_ context.Context, p db.UpdateWebhookParams) (db.ChangeWebhook, error) {
	m.updated = &p
	return db.ChangeWebhook{ID: p.ID}, nil
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
