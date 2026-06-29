package fleet

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// legacySpecVer reproduces the pre-schema-bump specVer hash (no schema token,
// no content flag folded in). The current specVer must differ from this so the
// always-on include_content / passApiKey constants force exactly one webhook
// recreate on the upgrade path.
func legacySpecVer(role Role) string {
	h := sha256.Sum256([]byte(strings.Join([]string{
		strings.Join(role.TriggerInclude, ","),
		strings.Join(role.TriggerExclude, ","),
		strings.Join(role.ReadPatterns, ","),
		strings.Join(role.WritePatterns, ","),
		strings.Join(role.AttachNotes, ","),
		strings.Join(role.TriggerOn, ","),
		strconv.Itoa(role.MaxDepth),
		role.Concurrency,
	}, "|")))
	return base64.RawURLEncoding.EncodeToString(h[:6])
}

// TestSpecVer_SchemaBumpRotatesMarker asserts the schema-version bump makes
// specVer differ from the legacy hash for a fixed role, so every marker rotates
// exactly once and webhooks pick up the always-on include_content flag.
func TestSpecVer_SchemaBumpRotatesMarker(t *testing.T) {
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1, Concurrency: "skip"}
	require.NotEqual(t, legacySpecVer(role), specVer(role),
		"schema bump must rotate the spec version vs the pre-fix value")
}

// TestReconcile_SchemaBumpRecreatesLegacyWebhook asserts a webhook stored with a
// legacy (pre-bump) marker is deleted and recreated, and the recreated webhook
// enables include_content.
func TestReconcile_SchemaBumpRecreatesLegacyWebhook(t *testing.T) {
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1, Concurrency: "skip"}
	legacyMarker := "fleet:f1:" + role.NotePath + "#" + legacySpecVer(role)

	var createInputs []map[string]any
	var deletedIDs []int64
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			switch {
			case strings.Contains(q, "allChangeWebhooks"):
				return json.RawMessage(`{"admin":{"allChangeWebhooks":{"nodes":[{"id":7,"description":"` + legacyMarker + `"}]}}}`), nil
			case strings.Contains(q, "changeWebhookCreate"):
				createInputs = append(createInputs, vars["input"].(map[string]any))
				return json.RawMessage(`{"admin":{"changeWebhookCreate":{"webhook":{"id":8}}}}`), nil
			case strings.Contains(q, "changeWebhookDelete"):
				deletedIDs = append(deletedIDs, vars["input"].(map[string]any)["id"].(int64))
			}
			return json.RawMessage(`{}`), nil
		},
	}
	require.NoError(t, newReconciler(client).Reconcile(context.Background(), []Role{role}))
	require.Equal(t, []int64{7}, deletedIDs, "legacy-marker webhook must be deleted")
	require.Len(t, createInputs, 1, "a recreated webhook must be created")
	require.Equal(t, true, createInputs[0]["includeContent"],
		"recreated webhook must enable include_content")
}

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
				return json.RawMessage(`{"admin":{"allChangeWebhooks":{"nodes":[]}}}`), nil
			case strings.Contains(q, "changeWebhookCreate"):
				created = vars["input"].(map[string]any)
				return json.RawMessage(`{"admin":{"changeWebhookCreate":{"webhook":{"id":7},"secret":"s"}}}`), nil
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

// TestReconcile_RequestsNoteContent is a regression test for the content gap:
// trip2g only populates changes[].content when the webhook has include_content
// enabled (see matchChange in handlenotewebhooks). The fleet's Jet templates
// reference change_file.Content, so the reconciler must request it.
func TestReconcile_RequestsNoteContent(t *testing.T) {
	var created map[string]any
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			switch {
			case strings.Contains(q, "allChangeWebhooks"):
				return json.RawMessage(`{"admin":{"allChangeWebhooks":{"nodes":[]}}}`), nil
			case strings.Contains(q, "changeWebhookCreate"):
				created = vars["input"].(map[string]any)
				return json.RawMessage(`{"admin":{"changeWebhookCreate":{"webhook":{"id":7},"secret":"s"}}}`), nil
			}
			return nil, nil
		},
	}
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1}
	require.NoError(t, newReconciler(client).Reconcile(context.Background(), []Role{role}))
	require.NotNil(t, created)
	require.Equal(t, true, created["includeContent"],
		"reconciler must enable include_content so changes[].content is populated")
}

// TestReconcile_SendsTimeoutSeconds asserts the reconciler passes the role's
// (defaulted) timeout into the create input as timeoutSeconds, so trip2g waits
// long enough for a slow agent run instead of closing the delivery at its 60s
// default.
func TestReconcile_SendsTimeoutSeconds(t *testing.T) {
	cases := []struct {
		name    string
		role    Role
		wantSec int64
	}{
		{"explicit", Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1, TimeoutSeconds: 120}, 120},
		{"defaulted", Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1}, int64(defaultTimeoutSeconds)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var created map[string]any
			client := &ClientMock{
				GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
					switch {
					case strings.Contains(q, "allChangeWebhooks"):
						return json.RawMessage(`{"admin":{"allChangeWebhooks":{"nodes":[]}}}`), nil
					case strings.Contains(q, "changeWebhookCreate"):
						created = vars["input"].(map[string]any)
						return json.RawMessage(`{"admin":{"changeWebhookCreate":{"webhook":{"id":7}}}}`), nil
					}
					return nil, nil
				},
			}
			require.NoError(t, newReconciler(client).Reconcile(context.Background(), []Role{tc.role}))
			require.NotNil(t, created)
			require.Equal(t, tc.wantSec, created["timeoutSeconds"])
		})
	}
}

// TestSpecVer_TimeoutSecondsRotatesMarker asserts changing timeout_seconds
// rotates the spec version (and hence the marker), so the webhook is recreated
// with the new timeoutSeconds.
func TestSpecVer_TimeoutSecondsRotatesMarker(t *testing.T) {
	base := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1}
	bumped := base
	bumped.TimeoutSeconds = 120
	require.NotEqual(t, specVer(base), specVer(bumped),
		"changing timeout_seconds must rotate the spec version")
}

func TestReconcile_NoChangeWhenMarkerMatches(t *testing.T) {
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1, Concurrency: "skip"}
	desc := markerFor("f1", role)
	var createCalls, updateCalls, deleteCalls int
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, _ map[string]any) (json.RawMessage, error) {
			switch {
			case strings.Contains(q, "allChangeWebhooks"):
				return json.RawMessage(`{"admin":{"allChangeWebhooks":{"nodes":[{"id":7,"description":"` + desc + `"}]}}}`), nil
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
				return json.RawMessage(`{"admin":{"allChangeWebhooks":{"nodes":[
					{"id":7,"description":"fleet:f1:roles/old.md#deadbeef"},
					{"id":8,"description":"some-other-integration"}
				]}}}`), nil
			case strings.Contains(q, "changeWebhookDelete"):
				deletedIDs = append(deletedIDs, vars["input"].(map[string]any)["id"].(int64))
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
				return json.RawMessage(`{"admin":{"allChangeWebhooks":{"nodes":[{"id":7,"description":"fleet:f1:roles/a.md#x"}]}}}`), nil
			}
			if strings.Contains(q, "changeWebhookDelete") {
				deletedIDs = append(deletedIDs, vars["input"].(map[string]any)["id"].(int64))
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
				return json.RawMessage(`{"admin":{"allChangeWebhooks":{"nodes":[]}}}`), nil
			}
			if strings.Contains(q, "changeWebhookCreate") {
				// Server returns an ErrorPayload instead of a ChangeWebhookCreatePayload.
				return json.RawMessage(`{"admin":{"changeWebhookCreate":{"message":"url is required"}}}`), nil
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
				return json.RawMessage(`{"admin":{"allChangeWebhooks":{"nodes":[{"id":42,"description":"fleet:f1:roles/old.md#deadbeef"}]}}}`), nil
			}
			if strings.Contains(q, "changeWebhookDelete") {
				return json.RawMessage(`{"admin":{"changeWebhookDelete":{"message":"not found"}}}`), nil
			}
			return json.RawMessage(`{}`), nil
		},
	}
	err := newReconciler(client).Reconcile(context.Background(), nil) // no desired → delete stale
	require.Error(t, err, "ErrorPayload from changeWebhookDelete must propagate as an error")
	require.Contains(t, err.Error(), "not found")
}
