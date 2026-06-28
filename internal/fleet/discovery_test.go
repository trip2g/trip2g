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
			{"key":"mode","raw":"change"},
			{"key":"tools","raw":"[shell]"}
		]}}
	]}`
	client := &ClientMock{
		GraphQLAdminFunc: func(_ context.Context, q string, vars map[string]any) (json.RawMessage, error) {
			require.Contains(t, q, "notePaths")
			require.Equal(t, "roles/%", vars["like"])
			return json.RawMessage(resp), nil
		},
	}
	d := NewDiscovery(client, "roles/", []string{"search", "read_note", "patch_note"})
	roles, errs := d.DiscoverRoles(context.Background())
	require.Len(t, roles, 1)
	require.Equal(t, "roles/triage.md", roles[0].NotePath)
	require.Equal(t, []string{"search", "patch_note"}, roles[0].Tools)
	require.Len(t, errs, 1) // roles/bad.md rejected (shell not offered)
	require.Contains(t, errs[0].Error(), "roles/bad.md")
}
