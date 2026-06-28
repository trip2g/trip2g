package fleet

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newReconciler(client Client) *Reconciler {
	return NewReconciler(client, Config{
		FleetID:     "f1",
		CallbackURL: "https://fleet.example",
	})
}

func TestReconcile_CreatesMissingWebhook(t *testing.T) {
	var created map[string]any
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			switch {
			case strings.Contains(q, "allChangeWebhooks"):
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[]}}`), nil
			case strings.Contains(q, "changeWebhookCreate"):
				created = vars["input"].(map[string]any)
				return json.RawMessage(`{"changeWebhookCreate":{"webhook":{"id":7},"secret":"s"}}`), nil
			}
			return nil, nil
		},
	}
	role := Role{
		NotePath: "roles/triage.md", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		AttachNotes: []string{"boards/**"}, TriggerInclude: []string{"boards/sprint.md"},
		TriggerOn: []string{"update"}, MaxDepth: 1, Concurrency: "skip",
	}
	require.NoError(t, newReconciler(client).Reconcile(context.Background(), []Role{role}))
	require.NotNil(t, created)
	require.Equal(t, []string{"boards/sprint.md"}, created["includePatterns"])
	require.Equal(t, "skip", created["concurrencyMode"])
	require.Equal(t, true, created["passApiKey"])
	require.Equal(t, int64(1), created["maxDepth"])
	require.Contains(t, created["description"], "fleet:f1:roles/triage.md#")
	require.True(t, strings.HasPrefix(created["url"].(string), "https://fleet.example/deliver/"))
}

func TestReconcile_NoChangeWhenMarkerMatches(t *testing.T) {
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1, Concurrency: "skip"}
	desc := markerFor("f1", role)
	var createCalls, updateCalls, deleteCalls int
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, _ map[string]any) (json.RawMessage, error) {
			switch {
			case strings.Contains(q, "allChangeWebhooks"):
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[{"id":7,"description":"` + desc + `"}]}}`), nil
			case strings.Contains(q, "changeWebhookCreate"):
				createCalls++
			case strings.Contains(q, "changeWebhookUpdate"):
				updateCalls++
			case strings.Contains(q, "changeWebhookDelete"):
				deleteCalls++
			}
			return json.RawMessage(`{}`), nil
		},
	}
	require.NoError(t, newReconciler(client).Reconcile(context.Background(), []Role{role}))
	require.Zero(t, createCalls)
	require.Zero(t, updateCalls)
	require.Zero(t, deleteCalls)
}

func TestReconcile_DeletesStaleAndLeavesForeign(t *testing.T) {
	var deletedIDs []int64
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			switch {
			case strings.Contains(q, "allChangeWebhooks"):
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[
					{"id":7,"description":"fleet:f1:roles/old.md#deadbeef"},
					{"id":8,"description":"some-other-integration"}
				]}}`), nil
			case strings.Contains(q, "changeWebhookDelete"):
				deletedIDs = append(deletedIDs, int64(vars["input"].(map[string]any)["id"].(int64)))
			}
			return json.RawMessage(`{}`), nil
		},
	}
	require.NoError(t, newReconciler(client).Reconcile(context.Background(), nil))
	require.Equal(t, []int64{7}, deletedIDs) // foreign id 8 untouched
}

func TestDeregister_DeletesAllOwned(t *testing.T) {
	var deletedIDs []int64
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			if strings.Contains(q, "allChangeWebhooks") {
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[{"id":7,"description":"fleet:f1:roles/a.md#x"}]}}`), nil
			}
			if strings.Contains(q, "changeWebhookDelete") {
				deletedIDs = append(deletedIDs, int64(vars["input"].(map[string]any)["id"].(int64)))
			}
			return json.RawMessage(`{}`), nil
		},
	}
	require.NoError(t, newReconciler(client).Deregister(context.Background()))
	require.Equal(t, []int64{7}, deletedIDs)
}

// F9(c): create() returning an ErrorPayload must surface as an error, not be silently swallowed.
func TestReconcile_Create_ErrorPayload_SurfacesAsError(t *testing.T) {
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, _ map[string]any) (json.RawMessage, error) {
			if strings.Contains(q, "allChangeWebhooks") {
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[]}}`), nil
			}
			if strings.Contains(q, "changeWebhookCreate") {
				// Server returns an ErrorPayload instead of a ChangeWebhookCreatePayload.
				return json.RawMessage(`{"changeWebhookCreate":{"message":"url is required"}}`), nil
			}
			return json.RawMessage(`{}`), nil
		},
	}
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1}
	err := newReconciler(client).Reconcile(context.Background(), []Role{role})
	require.Error(t, err, "ErrorPayload from changeWebhookCreate must propagate as an error")
	require.Contains(t, err.Error(), "url is required")
}

// F9(c): delete() returning an ErrorPayload must surface as an error, not be silently swallowed.
func TestReconcile_Delete_ErrorPayload_SurfacesAsError(t *testing.T) {
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, _ map[string]any) (json.RawMessage, error) {
			if strings.Contains(q, "allChangeWebhooks") {
				// Existing stale webhook not in desired set → will be deleted.
				return json.RawMessage(`{"allChangeWebhooks":{"nodes":[{"id":42,"description":"fleet:f1:roles/old.md#deadbeef"}]}}`), nil
			}
			if strings.Contains(q, "changeWebhookDelete") {
				return json.RawMessage(`{"changeWebhookDelete":{"message":"not found"}}`), nil
			}
			return json.RawMessage(`{}`), nil
		},
	}
	err := newReconciler(client).Reconcile(context.Background(), nil) // no desired → delete stale
	require.Error(t, err, "ErrorPayload from changeWebhookDelete must propagate as an error")
	require.Contains(t, err.Error(), "not found")
}
