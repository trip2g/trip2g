package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"trip2g/internal/case/mcp"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type federationMock struct {
	searchFunc          func(ctx context.Context, params appmodel.FederationSearchParams) (appmodel.FederationResult, error)
	federatedSearchFunc func(ctx context.Context, params appmodel.FederationSearchParams) (appmodel.FederationResult, error)
	noteHTMLFunc        func(ctx context.Context, params appmodel.FederationNoteHTMLParams) (appmodel.FederationResult, error)
	expandFunc          func(ctx context.Context, params appmodel.FederationExpandParams) (appmodel.FederationResult, error)
	federatedExpandFunc func(ctx context.Context, params appmodel.FederationExpandParams) (appmodel.FederationResult, error)
}

func (m *federationMock) Search(ctx context.Context, params appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
	if m.searchFunc == nil {
		panic("unexpected Search call")
	}
	return m.searchFunc(ctx, params)
}

func (m *federationMock) Similar(ctx context.Context, params appmodel.FederationSimilarParams) (appmodel.FederationResult, error) {
	panic("unexpected Similar call")
}

func (m *federationMock) NoteHTML(ctx context.Context, params appmodel.FederationNoteHTMLParams) (appmodel.FederationResult, error) {
	if m.noteHTMLFunc == nil {
		panic("unexpected NoteHTML call")
	}
	return m.noteHTMLFunc(ctx, params)
}

func (m *federationMock) FederatedSearch(ctx context.Context, params appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
	if m.federatedSearchFunc == nil {
		panic("unexpected FederatedSearch call")
	}
	return m.federatedSearchFunc(ctx, params)
}

func (m *federationMock) FederatedSimilar(ctx context.Context, params appmodel.FederationSimilarParams) (appmodel.FederationResult, error) {
	panic("unexpected FederatedSimilar call")
}

func (m *federationMock) FederatedNoteHTML(ctx context.Context, params appmodel.FederationNoteHTMLParams) (appmodel.FederationResult, error) {
	panic("unexpected FederatedNoteHTML call")
}

func (m *federationMock) GraphQLRequest(ctx context.Context, params appmodel.FederationGraphQLParams) (appmodel.FederationResult, error) {
	panic("unexpected GraphQLRequest call")
}

func (m *federationMock) Expand(ctx context.Context, params appmodel.FederationExpandParams) (appmodel.FederationResult, error) {
	if m.expandFunc == nil {
		panic("unexpected Expand call")
	}
	return m.expandFunc(ctx, params)
}

func (m *federationMock) FederatedExpand(ctx context.Context, params appmodel.FederationExpandParams) (appmodel.FederationResult, error) {
	if m.federatedExpandFunc == nil {
		panic("unexpected FederatedExpand call")
	}
	return m.federatedExpandFunc(ctx, params)
}

func TestFederatedSearchUsesMockedFederationClient(t *testing.T) {
	kbNote := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://bob.example/_system/mcp",
		MCPFederationKBID:  "bob",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

	var gotQuery string
	federation := &federationMock{
		searchFunc: func(ctx context.Context, params appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			gotQuery = params.Query
			return appmodel.FederationResult{
				Content:           []appmodel.FederationContent{{Type: "text", Text: "remote bob"}},
				StructuredContent: json.RawMessage(`{"results":[{"title":"remote"}]}`),
			}, nil
		},
	}
	env := &EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return nvs
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		FederationClientFunc: func(_ context.Context, kbID string) (appmodel.Federation, error) {
			require.Equal(t, "bob", kbID)
			return federation, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "federated_search",
		Arguments: json.RawMessage(`{"query":"status","kb_id":"bob"}`),
	}
	paramsJSON, _ := json.Marshal(params)
	resp := mcp.Resolve(context.Background(), env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})

	require.Nil(t, resp.Error)
	require.Equal(t, "status", gotQuery)
	result := resp.Result.(mcp.CallToolResult)
	require.Equal(t, "remote bob", result.Content[0].Text)
	// A directly-federated base stamps kb_id = peer name onto every result.
	var payload mcp.SearchResultPayload
	require.NoError(t, json.Unmarshal(result.StructuredContent.(json.RawMessage), &payload))
	require.Len(t, payload.Results, 1)
	require.Equal(t, "bob", payload.Results[0].KBID)
}

func TestFederatedNoteHTMLToleratesStringPID(t *testing.T) {
	// Models replay search match ids ("p36:c2") as pid; the hub must not
	// reject the whole call — it forwards the valid path with pid unset.
	kbNote := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://bob.example/_system/mcp",
		MCPFederationKBID:  "nietzsche",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

	var gotParams appmodel.FederationNoteHTMLParams
	federation := &federationMock{
		noteHTMLFunc: func(ctx context.Context, params appmodel.FederationNoteHTMLParams) (appmodel.FederationResult, error) {
			gotParams = params
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "remote note body"}},
			}, nil
		},
	}
	env := &EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return nvs
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		FederationClientFunc: func(_ context.Context, kbID string) (appmodel.Federation, error) {
			return federation, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "federated_note_html",
		Arguments: json.RawMessage(`{"kb_id":"nietzsche","path":"concepts/volya-k-vlasti.md","pid":"p36:c2"}`),
	}
	paramsJSON, _ := json.Marshal(params)
	resp := mcp.Resolve(context.Background(), env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.Equal(t, "remote note body", result.Content[0].Text)
	require.Equal(t, "concepts/volya-k-vlasti.md", gotParams.Path)
	require.Zero(t, gotParams.PID)
}

func TestFederatedNoteHTMLForwardsMatchIDOnly(t *testing.T) {
	// federated_search's description tells agents to open a found chunk with
	// federated_note_html(kb_id=..., match_id=...) — the hub must forward a
	// match_id-only read instead of rejecting it.
	kbNote := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://bob.example/_system/mcp",
		MCPFederationKBID:  "nietzsche",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

	var gotParams appmodel.FederationNoteHTMLParams
	federation := &federationMock{
		noteHTMLFunc: func(ctx context.Context, params appmodel.FederationNoteHTMLParams) (appmodel.FederationResult, error) {
			gotParams = params
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "focused chunk"}},
			}, nil
		},
	}
	env := &EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return nvs
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		FederationClientFunc: func(_ context.Context, kbID string) (appmodel.Federation, error) {
			return federation, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "federated_note_html",
		Arguments: json.RawMessage(`{"kb_id":"nietzsche","match_id":"p12:c0","note_id":""}`),
	}
	paramsJSON, _ := json.Marshal(params)
	resp := mcp.Resolve(context.Background(), env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.Equal(t, "focused chunk", result.Content[0].Text)
	require.Equal(t, "p12:c0", gotParams.MatchID)
	require.Empty(t, gotParams.NoteID)
}

func TestFederatedSearchDelegatesNestedKBID(t *testing.T) {
	kbNote := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://bob.example/_system/mcp",
		MCPFederationKBID:  "bob",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

	var gotKBID string
	federation := &federationMock{
		federatedSearchFunc: func(ctx context.Context, params appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			gotKBID = params.KBID
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "remote nested"}},
			}, nil
		},
	}
	env := &EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return nvs
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		FederationClientFunc: func(_ context.Context, kbID string) (appmodel.Federation, error) {
			require.Equal(t, "bob", kbID)
			return federation, nil
		},
		FederationMaxDepthFunc: func() int { return 3 },
	}

	params := mcp.CallToolParams{
		Name:      "federated_search",
		Arguments: json.RawMessage(`{"query":"status","kb_id":"bob/deep"}`),
	}
	paramsJSON, _ := json.Marshal(params)
	resp := mcp.Resolve(context.Background(), env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})

	require.Nil(t, resp.Error)
	require.Equal(t, "deep", gotKBID)
	result := resp.Result.(mcp.CallToolResult)
	require.Equal(t, "remote nested", result.Content[0].Text)
}

func TestFederatedExpandUsesMockedFederationClient(t *testing.T) {
	kbNote := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://bob.example/_system/mcp",
		MCPFederationKBID:  "bob",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

	var gotPath []string
	federation := &federationMock{
		expandFunc: func(_ context.Context, params appmodel.FederationExpandParams) (appmodel.FederationResult, error) {
			gotPath = params.TocPath
			return appmodel.FederationResult{
				Content:           []appmodel.FederationContent{{Type: "text", Text: "remote children"}},
				StructuredContent: json.RawMessage(`{"children":[{"title":"Sub"}]}`),
			}, nil
		},
	}
	env := &EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		FederationClientFunc: func(_ context.Context, kbID string) (appmodel.Federation, error) {
			require.Equal(t, "bob", kbID)
			return federation, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "federated_expand",
		Arguments: json.RawMessage(`{"kb_id":"bob","pid":42,"toc_path":["Setup"]}`),
	}
	paramsJSON, _ := json.Marshal(params)
	resp := mcp.Resolve(context.Background(), env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})

	require.Nil(t, resp.Error)
	require.Equal(t, []string{"Setup"}, gotPath)
	result := resp.Result.(mcp.CallToolResult)
	require.Equal(t, "remote children", result.Content[0].Text)
	require.JSONEq(t, `{"children":[{"title":"Sub"}]}`, string(result.StructuredContent.(json.RawMessage)))
}
