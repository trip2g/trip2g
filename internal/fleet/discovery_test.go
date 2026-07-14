package fleet

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscoverRoles_ParsesValidSkipsInvalid(t *testing.T) {
	resp := `{"notePaths":[
		{"value":"roles/triage.md","content":"Triage the board.","latestNoteView":{"meta":[
			{"key":"fleet_id","raw":"f1"},
			{"key":"mode","raw":"change"},
			{"key":"model","raw":"gpt-4o-mini"},
			{"key":"tools","raw":"[search, patch_note]"},
			{"key":"read_patterns","raw":"[\"boards/**\"]"},
			{"key":"write_patterns","raw":"[\"boards/**\"]"},
			{"key":"trigger_include","raw":"[\"boards/sprint.md\"]"},
			{"key":"trigger_on","raw":"[update]"},
			{"key":"max_depth","raw":"1"},
			{"key":"concurrency","raw":"skip"}
		]}},
		{"value":"roles/bad.md","content":"x","latestNoteView":{"meta":[
			{"key":"fleet_id","raw":"f1"},
			{"key":"mode","raw":"change"},
			{"key":"tools","raw":"[shell]"}
		]}}
	]}`
	gql := fakeAdminGQL(func(op string, vars json.RawMessage) (string, error) {
		require.Equal(t, "DiscoverRoles", op)
		var v struct {
			Like string `json:"like"`
		}
		require.NoError(t, json.Unmarshal(vars, &v))
		require.Equal(t, "roles/%", v.Like)
		return resp, nil
	})
	d := NewDiscovery(gql, "f1", "roles/", []string{"search", "read_note", "patch_note"})
	roles, errs := d.DiscoverRoles(context.Background())
	require.Len(t, roles, 1)
	require.Equal(t, "roles/triage.md", roles[0].NotePath)
	require.Equal(t, []string{"search", "patch_note"}, roles[0].Tools)
	require.Len(t, errs, 1) // roles/bad.md rejected (shell not offered)
	require.Contains(t, errs[0].Error(), "roles/bad.md")
}

// roleNote renders one notePaths entry with a fleet_id and the minimal valid
// change-mode meta so it passes Validate when it belongs to this fleet.
func roleNote(path, fleetID string) string {
	fid := ""
	if fleetID != "" {
		fid = `{"key":"fleet_id","raw":"` + fleetID + `"},`
	}
	return `{"value":"` + path + `","content":"Body.","latestNoteView":{"meta":[
		` + fid + `
		{"key":"mode","raw":"change"},
		{"key":"trigger_include","raw":"[\"boards/**\"]"},
		{"key":"trigger_on","raw":"[update]"}
	]}}`
}

// TestDiscoverRoles_PartitionsByFleetID asserts the fleet_id partition:
// only matching-fleet_id roles are processed; a mismatched role is skipped
// silently; an untagged (empty fleet_id) role is skipped WITH an error/warning.
func TestDiscoverRoles_PartitionsByFleetID(t *testing.T) {
	resp := `{"notePaths":[` +
		roleNote("roles/mine.md", "f1") + `,` +
		roleNote("roles/theirs.md", "f2") + `,` +
		roleNote("roles/untagged.md", "") +
		`]}`
	gql := fakeAdminGQL(func(op string, _ json.RawMessage) (string, error) {
		require.Equal(t, "DiscoverRoles", op)
		return resp, nil
	})
	d := NewDiscovery(gql, "f1", "roles/", []string{"search"})
	roles, errs := d.DiscoverRoles(context.Background())

	require.Len(t, roles, 1, "only the matching-fleet_id role is processed")
	require.Equal(t, "roles/mine.md", roles[0].NotePath)

	// theirs.md is skipped silently (belongs to f2); untagged.md is a warning.
	require.Len(t, errs, 1, "untagged role warns; foreign role is silent")
	require.Contains(t, errs[0].Error(), "roles/untagged.md")
	require.Contains(t, errs[0].Error(), "fleet_id is empty")
	for _, e := range errs {
		require.NotContains(t, e.Error(), "roles/theirs.md", "foreign-fleet role must not warn")
	}
}

// TestDiscoverParsed_UnpartitionedForIntrospection asserts DiscoverParsed does
// NOT partition (it powers the cross-fleet graph/dry-run views): it returns
// every parsed role regardless of fleet_id, with fleet_id populated.
func TestDiscoverParsed_UnpartitionedForIntrospection(t *testing.T) {
	resp := `{"notePaths":[` +
		roleNote("roles/mine.md", "f1") + `,` +
		roleNote("roles/theirs.md", "f2") +
		`]}`
	gql := fakeAdminGQL(func(_ string, _ json.RawMessage) (string, error) {
		return resp, nil
	})
	d := NewDiscovery(gql, "f1", "roles/", []string{"search"})
	parsed, errs := d.DiscoverParsed(context.Background())
	require.Empty(t, errs)
	require.Len(t, parsed, 2, "DiscoverParsed keeps all roles for introspection")
	byPath := map[string]string{}
	for _, r := range parsed {
		byPath[r.NotePath] = r.FleetID
	}
	require.Equal(t, "f1", byPath["roles/mine.md"])
	require.Equal(t, "f2", byPath["roles/theirs.md"])
}
