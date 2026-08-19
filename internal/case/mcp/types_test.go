package mcp

import (
	"encoding/json"
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestSearchResultItemFederationJSON(t *testing.T) {
	plain := SearchResultItem{
		Title:    "Local",
		NoteID:   1,
		NotePath: "local.md",
		Href:     "/local",
		URL:      "https://hub.local/local",
		Kind:     "note",
		Score:    1,
	}

	plainJSON, err := json.Marshal(plain)
	require.NoError(t, err)
	require.NotContains(t, string(plainJSON), "federation")

	federated := plain
	federated.Federation = &FederationRef{
		KBID:             "bob",
		KBURL:            "https://bob.team.io/_system/mcp",
		AgentInstruction: "Use federated_search.",
	}

	federatedJSON, err := json.Marshal(federated)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"title": "Local",
		"note_id": 1,
		"note_path": "local.md",
		"href": "/local",
		"url": "https://hub.local/local",
		"kind": "note",
		"score": 1,
		"federation": {
			"kb_id": "bob",
			"kb_url": "https://bob.team.io/_system/mcp",
			"agent_instruction": "Use federated_search."
		}
	}`, string(federatedJSON))
}

func TestFederatedArgumentsJSON(t *testing.T) {
	search := model.MCPSearchParams{
		Query: "design notes",
		KBID:  "bob",
		KBIDs: []string{"bob", "github"},
	}

	data, err := json.Marshal(search)
	require.NoError(t, err)
	require.JSONEq(t, `{"query":"design notes","kb_id":"bob","kb_ids":["bob","github"]}`, string(data))

	similar := model.MCPSimilarParams{KBID: "bob", PID: model.PID{Value: 10}, Limit: 3}
	data, err = json.Marshal(similar)
	require.NoError(t, err)
	require.JSONEq(t, `{"kb_id":"bob","pid":10,"limit":3}`, string(data))

	html := model.MCPNoteHTMLParams{KBID: "bob", MatchID: "m1"}
	data, err = json.Marshal(html)
	require.NoError(t, err)
	require.JSONEq(t, `{"kb_id":"bob","match_id":"m1"}`, string(data))
}
