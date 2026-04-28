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
	panic("unexpected NoteHTML call")
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
		FederationClientFunc: func(kbID string) (appmodel.Federation, error) {
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
	require.JSONEq(t, `{"results":[{"title":"remote"}]}`, string(result.StructuredContent.(json.RawMessage)))
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
		FederationClientFunc: func(kbID string) (appmodel.Federation, error) {
			require.Equal(t, "bob", kbID)
			return federation, nil
		},
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
