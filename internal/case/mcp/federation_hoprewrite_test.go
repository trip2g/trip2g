package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"trip2g/internal/case/mcp"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// singlePeerEnv builds an env exposing one federation peer named kbID whose
// client is fed. This mirrors a hub with exactly one directly-federated base.
func singlePeerEnv(kbID string, fed *federationMock) *EnvMock {
	note := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://" + kbID + ".example/_system/mcp",
		MCPFederationKBID:  kbID,
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(note)}
	return &EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc: func(context.Context, *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		FederationClientFunc: func(context.Context, string) (appmodel.Federation, error) {
			return fed, nil
		},
	}
}

func fedSingleSearchPayload(t *testing.T, result mcp.CallToolResult) mcp.SearchResultPayload {
	t.Helper()
	raw, ok := result.StructuredContent.(json.RawMessage)
	require.True(t, ok, "expected json.RawMessage, got %T", result.StructuredContent)
	var payload mcp.SearchResultPayload
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}

// The hub owning child "philosophers" must prefix its own segment onto whatever
// kb_id the child reported, so a nietzsche note reached through philosophers
// arrives at the caller addressed philosophers/nietzsche.
func TestFederatedSearchPrefixesNestedKBID(t *testing.T) {
	var gotKBID string
	fed := &federationMock{
		federatedSearchFunc: func(_ context.Context, params appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			gotKBID = params.KBID
			// The philosophers hub already stamped its child segment on the way up.
			return appmodel.FederationResult{
				Content:           []appmodel.FederationContent{{Type: "text", Text: "will to power"}},
				StructuredContent: json.RawMessage(`{"results":[{"title":"WtP","kb_id":"nietzsche"}]}`),
			}, nil
		},
	}
	env := singlePeerEnv("philosophers", fed)

	result := callFederatedSearch(t, env, `{"query":"power","kb_id":"philosophers/nietzsche"}`)

	require.Equal(t, "nietzsche", gotKBID)
	payload := fedSingleSearchPayload(t, result)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "philosophers/nietzsche", payload.Results[0].KBID)
}

// The middle hub stamps its own child segment onto a leaf result that carried no
// kb_id yet. Composed with the outer hop this is what accrues to the full path.
func TestFederatedSearchStampsMiddleHopSegment(t *testing.T) {
	fed := &federationMock{
		searchFunc: func(context.Context, appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			// A leaf base answers over its own notes: no kb_id yet.
			return appmodel.FederationResult{
				Content:           []appmodel.FederationContent{{Type: "text", Text: "leaf"}},
				StructuredContent: json.RawMessage(`{"results":[{"title":"WtP"}]}`),
			}, nil
		},
	}
	env := singlePeerEnv("nietzsche", fed)

	result := callFederatedSearch(t, env, `{"query":"power","kb_id":"nietzsche"}`)

	payload := fedSingleSearchPayload(t, result)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "nietzsche", payload.Results[0].KBID)
}

// Blind fan-out results keep the correct per-base kb_id for each direct peer.
func TestFederatedSearchFanoutStampsPerBaseKBID(t *testing.T) {
	env := fanoutEnv(2, 10, 5, 0, func(kbID string) fanoutSearchFn {
		return func(context.Context, appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			return appmodel.FederationResult{
				Content:           []appmodel.FederationContent{{Type: "text", Text: "hit " + kbID}},
				StructuredContent: json.RawMessage(`{"results":[{"title":"` + kbID + `"}]}`),
			}, nil
		}
	})

	result := callFederatedSearch(t, env, `{"query":"status"}`)
	payload := fanoutPayload(t, result)
	require.Len(t, payload.Results, 2)

	byBase := map[string]string{}
	for _, r := range payload.Results {
		var sp mcp.SearchResultPayload
		require.NoError(t, json.Unmarshal(r.Result.StructuredContent, &sp))
		require.Len(t, sp.Results, 1)
		byBase[r.KBID] = sp.Results[0].KBID
	}
	require.Equal(t, map[string]string{"kb1": "kb1", "kb2": "kb2"}, byBase)
}

// A not-configured error from a child composes across the hop: the parent
// prefixes its child segment so the suggested address is the caller's full path.
func TestFederatedSearchComposedNotConfiguredHint(t *testing.T) {
	fed := &federationMock{
		federatedSearchFunc: func(context.Context, appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			return appmodel.FederationResult{
				Content:           []appmodel.FederationContent{{Type: "text", Text: "not configured"}},
				StructuredContent: json.RawMessage(`{"status":"federation_not_configured","kb_id":"ghost"}`),
			}, nil
		},
	}
	env := singlePeerEnv("philosophers", fed)

	result := callFederatedSearch(t, env, `{"query":"x","kb_id":"philosophers/ghost"}`)

	require.Contains(t, result.Content[0].Text, "philosophers/ghost")
	raw := result.StructuredContent.(json.RawMessage)
	var status mcp.FederationStatusPayload
	require.NoError(t, json.Unmarshal(raw, &status))
	require.Equal(t, "federation_not_configured", status.Status)
	require.Equal(t, "philosophers/ghost", status.KBID)
}

// A flat kb_id unknown to this hub keeps a generic hint — the hub cannot invent
// a specific parent, only tell the caller how nested bases are addressed.
func TestFederatedSearchFlatNotConfiguredGeneric(t *testing.T) {
	fed := &federationMock{} // never called: montaigne is not a direct peer
	env := singlePeerEnv("philosophers", fed)

	result := callFederatedSearch(t, env, `{"query":"x","kb_id":"montaigne"}`)

	text := result.Content[0].Text
	require.Contains(t, text, "<hub>/montaigne")
	require.NotContains(t, text, "philosophers/montaigne")
}
