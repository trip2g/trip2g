package mcp_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"trip2g/internal/features"

	"trip2g/internal/case/mcp"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type federationMock struct {
	searchFunc          func(ctx context.Context, params appmodel.MCPSearchParams) (appmodel.FederationResult, error)
	federatedSearchFunc func(ctx context.Context, params appmodel.MCPSearchParams) (appmodel.FederationResult, error)
	noteHTMLFunc        func(ctx context.Context, params appmodel.MCPNoteHTMLParams) (appmodel.FederationResult, error)
	similarFunc         func(ctx context.Context, params appmodel.MCPSimilarParams) (appmodel.FederationResult, error)
	expandFunc          func(ctx context.Context, params appmodel.MCPExpandParams) (appmodel.FederationResult, error)
	federatedExpandFunc func(ctx context.Context, params appmodel.MCPExpandParams) (appmodel.FederationResult, error)

	instructionsFunc          func(ctx context.Context) (appmodel.FederationResult, error)
	federatedInstructionsFunc func(ctx context.Context, params appmodel.MCPInstructionsParams) (appmodel.FederationResult, error)
}

func (m *federationMock) Search(ctx context.Context, params appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
	if m.searchFunc == nil {
		panic("unexpected Search call")
	}
	return m.searchFunc(ctx, params)
}

func (m *federationMock) Similar(ctx context.Context, params appmodel.MCPSimilarParams) (appmodel.FederationResult, error) {
	if m.similarFunc == nil {
		panic("unexpected Similar call")
	}
	return m.similarFunc(ctx, params)
}

func (m *federationMock) NoteHTML(ctx context.Context, params appmodel.MCPNoteHTMLParams) (appmodel.FederationResult, error) {
	if m.noteHTMLFunc == nil {
		panic("unexpected NoteHTML call")
	}
	return m.noteHTMLFunc(ctx, params)
}

func (m *federationMock) FederatedSearch(ctx context.Context, params appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
	if m.federatedSearchFunc == nil {
		panic("unexpected FederatedSearch call")
	}
	return m.federatedSearchFunc(ctx, params)
}

func (m *federationMock) FederatedSimilar(ctx context.Context, params appmodel.MCPSimilarParams) (appmodel.FederationResult, error) {
	panic("unexpected FederatedSimilar call")
}

func (m *federationMock) FederatedNoteHTML(ctx context.Context, params appmodel.MCPNoteHTMLParams) (appmodel.FederationResult, error) {
	panic("unexpected FederatedNoteHTML call")
}

func (m *federationMock) GraphQLRequest(ctx context.Context, params appmodel.MCPGraphQLParams) (appmodel.FederationResult, error) {
	panic("unexpected GraphQLRequest call")
}

func (m *federationMock) Expand(ctx context.Context, params appmodel.MCPExpandParams) (appmodel.FederationResult, error) {
	if m.expandFunc == nil {
		panic("unexpected Expand call")
	}
	return m.expandFunc(ctx, params)
}

func (m *federationMock) FederatedExpand(ctx context.Context, params appmodel.MCPExpandParams) (appmodel.FederationResult, error) {
	if m.federatedExpandFunc == nil {
		panic("unexpected FederatedExpand call")
	}
	return m.federatedExpandFunc(ctx, params)
}

func (m *federationMock) Instructions(ctx context.Context) (appmodel.FederationResult, error) {
	if m.instructionsFunc == nil {
		panic("unexpected Instructions call")
	}
	return m.instructionsFunc(ctx)
}

func (m *federationMock) FederatedInstructions(ctx context.Context, params appmodel.MCPInstructionsParams) (appmodel.FederationResult, error) {
	if m.federatedInstructionsFunc == nil {
		panic("unexpected FederatedInstructions call")
	}
	return m.federatedInstructionsFunc(ctx, params)
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
		searchFunc: func(ctx context.Context, params appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
			gotQuery = params.Query
			return appmodel.FederationResult{
				Content:           []appmodel.FederationContent{{Type: "text", Text: "remote bob"}},
				StructuredContent: json.RawMessage(`{"results":[{"title":"remote"}]}`),
			}, nil
		},
	}
	env := &EnvMock{
		FeaturesFunc: func() features.Features { return features.Features{} },

		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
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
	resp := callMCP(t, env, mcp.Request{
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

	var gotParams appmodel.MCPNoteHTMLParams
	federation := &federationMock{
		noteHTMLFunc: func(ctx context.Context, params appmodel.MCPNoteHTMLParams) (appmodel.FederationResult, error) {
			gotParams = params
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "remote note body"}},
			}, nil
		},
	}
	env := &EnvMock{
		FeaturesFunc: func() features.Features { return features.Features{} },

		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
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
	resp := callMCP(t, env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.Equal(t, "remote note body", result.Content[0].Text)
	require.Equal(t, "concepts/volya-k-vlasti.md", gotParams.Path)
	require.Zero(t, gotParams.PID.Value)
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

	var gotParams appmodel.MCPNoteHTMLParams
	federation := &federationMock{
		noteHTMLFunc: func(ctx context.Context, params appmodel.MCPNoteHTMLParams) (appmodel.FederationResult, error) {
			gotParams = params
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "focused chunk"}},
			}, nil
		},
	}
	env := &EnvMock{
		FeaturesFunc: func() features.Features { return features.Features{} },

		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
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
	resp := callMCP(t, env, mcp.Request{
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
		federatedSearchFunc: func(ctx context.Context, params appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
			gotKBID = params.KBID
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "remote nested"}},
			}, nil
		},
	}
	env := &EnvMock{
		FeaturesFunc: func() features.Features { return features.Features{} },

		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
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
	resp := callMCP(t, env, mcp.Request{
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
		expandFunc: func(_ context.Context, params appmodel.MCPExpandParams) (appmodel.FederationResult, error) {
			gotPath = params.TocPath
			return appmodel.FederationResult{
				Content:           []appmodel.FederationContent{{Type: "text", Text: "remote children"}},
				StructuredContent: json.RawMessage(`{"children":[{"title":"Sub"}]}`),
			}, nil
		},
	}
	env := &EnvMock{
		FeaturesFunc: func() features.Features { return features.Features{} },

		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc:        func() *appmodel.NoteViews { return nvs },
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
	resp := callMCP(t, env, mcp.Request{
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

// TestFederatedCallsForwardEveryArgument pins the class of bug that made the
// hub advertise arguments it then dropped: limit and detail_limit never reached
// a peer's search, and toc_path was absent from federated_note_html entirely,
// so a remote read always returned the whole note instead of one section.
func TestFederatedCallsForwardEveryArgument(t *testing.T) {
	kbNote := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://bob.example/_system/mcp",
		MCPFederationKBID:  "nietzsche",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

	ok := appmodel.FederationResult{Content: []appmodel.FederationContent{{Type: "text", Text: "ok"}}}

	var search appmodel.MCPSearchParams
	var noteHTML appmodel.MCPNoteHTMLParams
	var similar appmodel.MCPSimilarParams
	federation := &federationMock{
		searchFunc: func(_ context.Context, p appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
			search = p
			return ok, nil
		},
		noteHTMLFunc: func(_ context.Context, p appmodel.MCPNoteHTMLParams) (appmodel.FederationResult, error) {
			noteHTML = p
			return ok, nil
		},
		similarFunc: func(_ context.Context, p appmodel.MCPSimilarParams) (appmodel.FederationResult, error) {
			similar = p
			return ok, nil
		},
	}
	env := &EnvMock{
		FeaturesFunc: func() features.Features { return features.Features{} },

		FederationMaxDepthFunc:     func() int { return 3 },
		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc:        func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		FederationClientFunc: func(_ context.Context, _ string) (appmodel.Federation, error) {
			return federation, nil
		},
	}

	call := func(name, args string) {
		t.Helper()
		paramsJSON, err := json.Marshal(mcp.CallToolParams{Name: name, Arguments: json.RawMessage(args)})
		require.NoError(t, err)
		resp := callMCP(t, env, mcp.Request{JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 1})
		require.Nil(t, resp.Error)
	}

	call("federated_search", `{"kb_id":"nietzsche","query":"воля","limit":2,"detail_limit":1}`)
	require.Equal(t, 2, search.Limit)
	require.Equal(t, 1, search.DetailLimit)
	require.Empty(t, search.KBID, "the hub's routing token must not travel to the peer")

	call("federated_note_html", `{"kb_id":"nietzsche","path":"a.md","toc_path":["Глава 1","Введение"]}`)
	require.Equal(t, []string{"Глава 1", "Введение"}, noteHTML.TocPath)

	call("federated_similar", `{"kb_id":"nietzsche","path":"a.md","limit":3}`)
	require.Equal(t, 3, similar.Limit)
}

// TestFederatedSingleKBCallTimesOut pins the deadline on the single-KB hop. Only
// the fan-out path was bounded, so a peer that accepted the connection and never
// answered left the agent waiting on a tool that never returned — worse for it
// than a fast failure.
func TestFederatedSingleKBCallTimesOut(t *testing.T) {
	kbNote := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://bob.example/_system/mcp",
		MCPFederationKBID:  "nietzsche",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

	const peerSilence = 3 * time.Second
	federation := &federationMock{
		noteHTMLFunc: func(ctx context.Context, _ appmodel.MCPNoteHTMLParams) (appmodel.FederationResult, error) {
			// A peer that accepts the call and sits on it. The upper bound only
			// keeps the test from hanging when the hop is unbounded; the hop's
			// own deadline is an order of magnitude shorter.
			select {
			case <-ctx.Done():
				return appmodel.FederationResult{}, ctx.Err()
			case <-time.After(peerSilence):
				return appmodel.FederationResult{
					Content: []appmodel.FederationContent{{Type: "text", Text: "too late"}},
				}, nil
			}
		},
	}
	env := &EnvMock{
		FeaturesFunc: func() features.Features { return features.Features{} },

		FederationMaxDepthFunc:     func() int { return 3 },
		FederatedFanoutTimeoutFunc: func() time.Duration { return 50 * time.Millisecond },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc:        func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		FederationClientFunc: func(_ context.Context, _ string) (appmodel.Federation, error) {
			return federation, nil
		},
	}

	paramsJSON, err := json.Marshal(mcp.CallToolParams{
		Name:      "federated_note_html",
		Arguments: json.RawMessage(`{"kb_id":"nietzsche","path":"a.md"}`),
	})
	require.NoError(t, err)

	start := time.Now()
	resp := callMCP(t, env, mcp.Request{JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 1})

	require.NotNil(t, resp.Error, "an unanswered peer must surface as an error, not a result")
	require.Less(t, time.Since(start), peerSilence, "the hop must give up on its own deadline")
}
