package fleet

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/require"

	"trip2g/internal/fleet/trip2ggql"
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

// nodesData renders an allChangeWebhooks {nodes} data object from id/description
// pairs.
func nodesData(nodes ...[2]string) string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		parts = append(parts, fmt.Sprintf(`{"id":%s,"description":%q}`, n[0], n[1]))
	}
	return `{"admin":{"allChangeWebhooks":{"nodes":[` + strings.Join(parts, ",") + `]}}}`
}

// nodesDataURL is like nodesData but carries a url per node, for takeover tests.
func nodesDataURL(nodes ...[3]string) string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		parts = append(parts, fmt.Sprintf(`{"id":%s,"description":%q,"url":%q}`, n[0], n[1], n[2]))
	}
	return `{"admin":{"allChangeWebhooks":{"nodes":[` + strings.Join(parts, ",") + `]}}}`
}

const createOKData = `{"admin":{"changeWebhookCreate":{"__typename":"ChangeWebhookCreatePayload","webhook":{"id":8}}}}`

const deleteOKData = `{"admin":{"changeWebhookDelete":{"__typename":"ChangeWebhookDeletePayload","deletedId":0}}}`

const updateOKData = `{"admin":{"changeWebhookUpdate":{"__typename":"ChangeWebhookUpdatePayload","webhook":{"id":7}}}}`

// updateInputFrom decodes an UpdateChangeWebhook request's typed input variable.
func updateInputFrom(t *testing.T, vars json.RawMessage) trip2ggql.ChangeWebhookUpdateInput {
	t.Helper()
	var v struct {
		Input trip2ggql.ChangeWebhookUpdateInput `json:"input"`
	}
	require.NoError(t, json.Unmarshal(vars, &v))
	return v.Input
}

// TestReconcile_TakesOverStaleURL asserts the hash-registry takeover: a webhook
// with this fleet's marker but pointing at a stale address (a restarted/moved
// fleet with the same fleet_id) is re-pointed at the fleet's current /<h>/
// delivery url via changeWebhookUpdate — no create, no delete (last-writer-wins).
func TestReconcile_TakesOverStaleURL(t *testing.T) {
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1, Concurrency: "skip"}
	marker := markerFor("f1", role)
	staleURL := "https://OLD-host.example/_fleet/" + fleetHash("f1") + "/webhook/" + urlKey(role.NotePath)

	var updates []trip2ggql.ChangeWebhookUpdateInput
	var createCalls, deleteCalls int
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesDataURL([3]string{"7", marker, staleURL}), nil
		case "UpdateChangeWebhook":
			updates = append(updates, updateInputFrom(t, vars))
			return updateOKData, nil
		case "CreateChangeWebhook":
			createCalls++
			return createOKData, nil
		case "DeleteChangeWebhook":
			deleteCalls++
			return deleteOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.Zero(t, createCalls, "takeover updates, never recreates")
	require.Zero(t, deleteCalls, "takeover updates, never deletes")
	require.Len(t, updates, 1, "the stale-url webhook must be taken over via update")
	require.Equal(t, int64(7), updates[0].Id)
	require.Equal(t, changeDeliveryURL("https://fleet.example", "f1", role.NotePath), updates[0].Url)
}

// TestReconcile_NoUpdateWhenURLMatches asserts that when the marker AND the url
// already match this fleet's current address, reconcile is a no-op (no update).
func TestReconcile_NoUpdateWhenURLMatches(t *testing.T) {
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1, Concurrency: "skip"}
	marker := markerFor("f1", role)
	currentURL := changeDeliveryURL("https://fleet.example", "f1", role.NotePath)

	var updateCalls, createCalls, deleteCalls int
	gql := fakeAdminGQL(func(op string, _ json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesDataURL([3]string{"7", marker, currentURL}), nil
		case "UpdateChangeWebhook":
			updateCalls++
			return updateOKData, nil
		case "CreateChangeWebhook":
			createCalls++
			return createOKData, nil
		case "DeleteChangeWebhook":
			deleteCalls++
			return deleteOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.Zero(t, updateCalls, "url already current => no takeover update")
	require.Zero(t, createCalls)
	require.Zero(t, deleteCalls)
}

// TestReconcile_Update_ErrorPayload_SurfacesAsError asserts an ErrorPayload from
// changeWebhookUpdate propagates instead of being swallowed.
func TestReconcile_Update_ErrorPayload_SurfacesAsError(t *testing.T) {
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1}
	marker := markerFor("f1", role)
	staleURL := "https://old.example/_fleet/x/webhook/" + urlKey(role.NotePath)
	gql := fakeAdminGQL(func(op string, _ json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesDataURL([3]string{"7", marker, staleURL}), nil
		case "UpdateChangeWebhook":
			return `{"admin":{"changeWebhookUpdate":{"__typename":"ErrorPayload","message":"url is required"}}}`, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	err := newReconciler(gql).Reconcile(context.Background(), []Role{role})
	require.Error(t, err)
	require.Contains(t, err.Error(), "url is required")
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

	var createInputs []trip2ggql.ChangeWebhookCreateInput
	var deletedIDs []int64
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData([2]string{"7", legacyMarker}), nil
		case "CreateChangeWebhook":
			createInputs = append(createInputs, createInputFrom(t, vars))
			return createOKData, nil
		case "DeleteChangeWebhook":
			deletedIDs = append(deletedIDs, deleteIDFrom(t, vars))
			return deleteOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.Equal(t, []int64{7}, deletedIDs, "legacy-marker webhook must be deleted")
	require.Len(t, createInputs, 1, "a recreated webhook must be created")
	require.True(t, createInputs[0].IncludeContent,
		"recreated webhook must enable include_content")
}

// TestReconcile_DropsExtrasSharingHashSegment asserts that when two webhooks
// share this fleet's /<h>/ (same fleet_id) — the desired one plus a stale
// superseded-spec extra — the extra is deleted while the desired one is kept.
func TestReconcile_DropsExtrasSharingHashSegment(t *testing.T) {
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1, Concurrency: "skip"}
	marker := markerFor("f1", role)
	currentURL := changeDeliveryURL("https://fleet.example", "f1", role.NotePath)
	h := fleetHash("f1")
	staleMarker := "fleet:f1:" + role.NotePath + "#deadbeef"           // same /<h>/, superseded spec
	staleURL := "https://fleet.example/_fleet/" + h + "/webhook/stale" // shares /<h>/

	var deletedIDs []int64
	var updateCalls, createCalls int
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesDataURL(
				[3]string{"7", marker, currentURL},
				[3]string{"9", staleMarker, staleURL},
			), nil
		case "DeleteChangeWebhook":
			deletedIDs = append(deletedIDs, deleteIDFrom(t, vars))
			return deleteOKData, nil
		case "UpdateChangeWebhook":
			updateCalls++
			return updateOKData, nil
		case "CreateChangeWebhook":
			createCalls++
			return createOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.Equal(t, []int64{9}, deletedIDs, "the superseded extra sharing /<h>/ is dropped")
	require.Zero(t, createCalls, "desired marker already present")
	require.Zero(t, updateCalls, "desired url already current")
}

func newReconciler(gql graphql.Client) *Reconciler {
	return NewReconciler(gql, Config{
		FleetID:     "f1",
		CallbackURL: "https://fleet.example",
	})
}

func TestReconcile_CreatesMissingWebhook(t *testing.T) {
	var created *trip2ggql.ChangeWebhookCreateInput
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(), nil
		case "CreateChangeWebhook":
			in := createInputFrom(t, vars)
			created = &in
			return createOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	role := Role{
		NotePath: "roles/triage.md", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		AttachNotes: []string{"boards/**"}, TriggerInclude: []string{"boards/sprint.md"},
		TriggerOn: []string{"update"}, MaxDepth: 1, Concurrency: "skip",
	}
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.NotNil(t, created)
	require.Equal(t, []string{"boards/sprint.md"}, created.IncludePatterns)
	require.Equal(t, "skip", created.ConcurrencyMode)
	require.True(t, created.PassApiKey)
	require.Equal(t, int64(1), created.MaxDepth)
	require.Contains(t, created.Description, "fleet:f1:roles/triage.md#")
	require.Equal(t, changeDeliveryURL("https://fleet.example", "f1", "roles/triage.md"), created.Url)
	require.Contains(t, created.Url, "/_fleet/"+fleetHash("f1")+"/webhook/")
}

// TestReconcile_RequestsNoteContent is a regression test for the content gap:
// trip2g only populates changes[].content when the webhook has include_content
// enabled (see matchChange in handlenotewebhooks). The fleet's Jet templates
// reference change_file.Content, so the reconciler must request it.
func TestReconcile_RequestsNoteContent(t *testing.T) {
	var created *trip2ggql.ChangeWebhookCreateInput
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(), nil
		case "CreateChangeWebhook":
			in := createInputFrom(t, vars)
			created = &in
			return createOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1}
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.NotNil(t, created)
	require.True(t, created.IncludeContent,
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
			var created *trip2ggql.ChangeWebhookCreateInput
			gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
				switch op {
				case "ListChangeWebhooks":
					return nodesData(), nil
				case "CreateChangeWebhook":
					in := createInputFrom(t, vars)
					created = &in
					return createOKData, nil
				}
				return "", fmt.Errorf("unexpected op %q", op)
			})
			require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{tc.role}))
			require.NotNil(t, created)
			require.Equal(t, tc.wantSec, created.TimeoutSeconds)
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
	var createCalls, deleteCalls int
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData([2]string{"7", desc}), nil
		case "CreateChangeWebhook":
			createCalls++
			return createOKData, nil
		case "DeleteChangeWebhook":
			deleteCalls++
			return deleteOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.Zero(t, createCalls)
	require.Zero(t, deleteCalls)
}

func TestReconcile_DeletesStaleAndLeavesForeign(t *testing.T) {
	var deletedIDs []int64
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(
				[2]string{"7", "fleet:f1:roles/old.md#deadbeef"},
				[2]string{"8", "some-other-integration"},
			), nil
		case "DeleteChangeWebhook":
			deletedIDs = append(deletedIDs, deleteIDFrom(t, vars))
			return deleteOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), nil))
	require.Equal(t, []int64{7}, deletedIDs) // foreign id 8 untouched
}

func TestDeregister_DeletesAllOwned(t *testing.T) {
	var deletedIDs []int64
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData([2]string{"7", "fleet:f1:roles/a.md#x"}), nil
		case "DeleteChangeWebhook":
			deletedIDs = append(deletedIDs, deleteIDFrom(t, vars))
			return deleteOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Deregister(context.Background()))
	require.Equal(t, []int64{7}, deletedIDs)
}

// F9(c): create() returning an ErrorPayload must surface as an error, not be silently swallowed.
func TestReconcile_Create_ErrorPayload_SurfacesAsError(t *testing.T) {
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(), nil
		case "CreateChangeWebhook":
			// Server returns an ErrorPayload instead of a ChangeWebhookCreatePayload.
			return `{"admin":{"changeWebhookCreate":{"__typename":"ErrorPayload","message":"url is required"}}}`, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	role := Role{NotePath: "roles/triage.md", Mode: "change", MaxDepth: 1}
	err := newReconciler(gql).Reconcile(context.Background(), []Role{role})
	require.Error(t, err, "ErrorPayload from changeWebhookCreate must propagate as an error")
	require.Contains(t, err.Error(), "url is required")
}

// F9(c): delete() returning an ErrorPayload must surface as an error, not be silently swallowed.
func TestReconcile_Delete_ErrorPayload_SurfacesAsError(t *testing.T) {
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			// Existing stale webhook not in desired set → will be deleted.
			return nodesData([2]string{"42", "fleet:f1:roles/old.md#deadbeef"}), nil
		case "DeleteChangeWebhook":
			return `{"admin":{"changeWebhookDelete":{"__typename":"ErrorPayload","message":"not found"}}}`, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	err := newReconciler(gql).Reconcile(context.Background(), nil) // no desired → delete stale
	require.Error(t, err, "ErrorPayload from changeWebhookDelete must propagate as an error")
	require.Contains(t, err.Error(), "not found")
}

// -- Cron webhook reconcile tests --

// newCronRole returns a minimal cron-mode role that passes Validate.
func newCronRole() Role {
	return Role{
		NotePath:       "roles/kb-refresh.md",
		Mode:           "cron",
		CronSchedule:   "0 */6 * * *",
		ReadPatterns:   []string{"kb/**"},
		WritePatterns:  []string{"kb/**"},
		AttachNotes:    []string{"kb/**"},
		TimeoutSeconds: 120,
	}
}

// cronNodesData builds an allCronWebhooks {nodes} data object from id/description pairs.
func cronNodesData(nodes ...[2]string) string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		parts = append(parts, fmt.Sprintf(`{"id":%s,"description":%q}`, n[0], n[1]))
	}
	return `{"admin":{"allCronWebhooks":{"nodes":[` + strings.Join(parts, ",") + `]}}}`
}

const createCronOKData = `{"admin":{"createCronWebhook":{"__typename":"CreateCronWebhookPayload","cronWebhook":{"id":10}}}}`
const deleteCronOKData = `{"admin":{"deleteCronWebhook":{"__typename":"DeleteCronWebhookPayload","deletedId":0}}}`

// createCronInputFrom decodes a CreateCronWebhook request's typed input variable.
func createCronInputFrom(t *testing.T, vars json.RawMessage) trip2ggql.CreateCronWebhookInput {
	t.Helper()
	var v struct {
		Input trip2ggql.CreateCronWebhookInput `json:"input"`
	}
	require.NoError(t, json.Unmarshal(vars, &v))
	return v.Input
}

// TestReconcile_CreatesCronWebhookForCronRole asserts that a cron-mode role
// triggers a CreateCronWebhook call with the expected fields.
func TestReconcile_CreatesCronWebhookForCronRole(t *testing.T) {
	var createdInput *trip2ggql.CreateCronWebhookInput
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(), nil
		case "ListCronWebhooks":
			return cronNodesData(), nil
		case "CreateCronWebhook":
			in := createCronInputFrom(t, vars)
			createdInput = &in
			return createCronOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	role := newCronRole()
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.NotNil(t, createdInput, "CreateCronWebhook must be called")
	require.Equal(t, "0 */6 * * *", createdInput.CronSchedule)
	require.True(t, createdInput.PassApiKey, "cron webhook must pass api_key")
	require.True(t, createdInput.Enabled, "cron webhook must be enabled")
	require.Equal(t, int64(120), createdInput.TimeoutSeconds)
	require.Equal(t, cronDeliveryURL("https://fleet.example", "f1", "roles/kb-refresh.md"), createdInput.Url)
	require.Contains(t, createdInput.Url, "/_fleet/"+fleetHash("f1")+"/webhook/cron/")
	require.Contains(t, createdInput.Description, "fleetcron:f1:roles/kb-refresh.md#")
}

// TestReconcile_BothModeRegistersChangeAndCronWebhooks asserts that a both-mode
// role registers exactly one change webhook AND one cron webhook.
func TestReconcile_BothModeRegistersChangeAndCronWebhooks(t *testing.T) {
	var changeCalls, cronCalls int
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(), nil
		case "ListCronWebhooks":
			return cronNodesData(), nil
		case "CreateChangeWebhook":
			changeCalls++
			return createOKData, nil
		case "CreateCronWebhook":
			cronCalls++
			return createCronOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	role := Role{
		NotePath:       "roles/both.md",
		Mode:           "both",
		CronSchedule:   "*/15 * * * *",
		TriggerOn:      []string{"update"},
		TriggerInclude: []string{"notes/**"},
	}
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.Equal(t, 1, changeCalls, "one change webhook must be created")
	require.Equal(t, 1, cronCalls, "one cron webhook must be created")
}

// TestReconcile_CronWebhookNoChangeWhenMarkerMatches asserts that an existing
// cron webhook with a matching marker is not deleted and not recreated.
func TestReconcile_CronWebhookNoChangeWhenMarkerMatches(t *testing.T) {
	role := newCronRole()
	desc := cronMarkerFor("f1", role)
	var createCronCalls, deleteCronCalls int
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(), nil
		case "ListCronWebhooks":
			return cronNodesData([2]string{"10", desc}), nil
		case "CreateCronWebhook":
			createCronCalls++
			return createCronOKData, nil
		case "DeleteCronWebhook":
			deleteCronCalls++
			return deleteCronOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), []Role{role}))
	require.Zero(t, createCronCalls, "no cron webhook should be created when marker matches")
	require.Zero(t, deleteCronCalls, "no cron webhook should be deleted when marker matches")
}

// TestReconcile_DeletesStaleOwnedCronWebhook asserts that a cron webhook owned
// by this fleet but no longer desired is deleted.
func TestReconcile_DeletesStaleOwnedCronWebhook(t *testing.T) {
	var deletedCronIDs []int64
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(), nil
		case "ListCronWebhooks":
			return cronNodesData([2]string{"55", "fleetcron:f1:roles/old.md#stale"}), nil
		case "DeleteCronWebhook":
			var v struct {
				Input trip2ggql.DeleteCronWebhookInput `json:"input"`
			}
			require.NoError(t, json.Unmarshal(vars, &v))
			deletedCronIDs = append(deletedCronIDs, v.Input.Id)
			return deleteCronOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Reconcile(context.Background(), nil))
	require.Equal(t, []int64{55}, deletedCronIDs, "stale fleet-owned cron webhook must be deleted")
}

// TestDeregister_DeletesOwnedCronWebhooks asserts Deregister also removes
// cron webhooks owned by this fleet.
func TestDeregister_DeletesOwnedCronWebhooks(t *testing.T) {
	var deletedCronIDs []int64
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(), nil
		case "ListCronWebhooks":
			return cronNodesData([2]string{"20", "fleetcron:f1:roles/x.md#abc"}), nil
		case "DeleteCronWebhook":
			var v struct {
				Input trip2ggql.DeleteCronWebhookInput `json:"input"`
			}
			require.NoError(t, json.Unmarshal(vars, &v))
			deletedCronIDs = append(deletedCronIDs, v.Input.Id)
			return deleteCronOKData, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	require.NoError(t, newReconciler(gql).Deregister(context.Background()))
	require.Equal(t, []int64{20}, deletedCronIDs, "cron webhook must be removed on deregister")
}

// TestReconcile_CronCreate_ErrorPayload_SurfacesAsError asserts that an
// ErrorPayload from createCronWebhook propagates as an error.
func TestReconcile_CronCreate_ErrorPayload_SurfacesAsError(t *testing.T) {
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		switch op {
		case "ListChangeWebhooks":
			return nodesData(), nil
		case "ListCronWebhooks":
			return cronNodesData(), nil
		case "CreateCronWebhook":
			return `{"admin":{"createCronWebhook":{"__typename":"ErrorPayload","message":"invalid schedule"}}}`, nil
		}
		return "", fmt.Errorf("unexpected op %q", op)
	})
	err := newReconciler(gql).Reconcile(context.Background(), []Role{newCronRole()})
	require.Error(t, err, "ErrorPayload from createCronWebhook must propagate as an error")
	require.Contains(t, err.Error(), "invalid schedule")
}
