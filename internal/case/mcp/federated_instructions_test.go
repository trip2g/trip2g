package mcp_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"

	"trip2g/internal/case/mcp"
	"trip2g/internal/fedinstr"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func fedInstrNoteViews() *appmodel.NoteViews {
	kbNote := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://bob.example/_system/mcp",
		MCPFederationKBID:  "bob",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}
	return nvs
}

func federatedInstructionsRequest(args string) mcp.Request {
	params := mcp.CallToolParams{Name: "federated_instructions", Arguments: json.RawMessage(args)}
	paramsJSON, _ := json.Marshal(params)
	return mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	}
}

func callFederatedInstructions(t *testing.T, env mcp.Env, args string) mcp.Response {
	t.Helper()
	return callMCP(t, env, federatedInstructionsRequest(args))
}

func TestFederatedInstructionsDirectPeer(t *testing.T) {
	nvs := fedInstrNoteViews()
	cache := fedinstr.New()
	federation := &federationMock{
		instructionsFunc: func(_ context.Context) (appmodel.FederationResult, error) {
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "bob's guidance"}},
			}, nil
		},
	}
	env := &EnvMock{
		MCPMetricsFunc:      func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:     func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		FederationClientFunc: func(_ context.Context, kbID string) (appmodel.Federation, error) {
			require.Equal(t, "bob", kbID)
			return federation, nil
		},
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	resp := callFederatedInstructions(t, env, `{"kb_id":"bob"}`)
	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.Equal(t, "bob's guidance", result.Content[0].Text)
}

func TestFederatedInstructionsNestedForwards(t *testing.T) {
	nvs := fedInstrNoteViews()
	cache := fedinstr.New()
	var gotKBID string
	federation := &federationMock{
		federatedInstructionsFunc: func(_ context.Context, params appmodel.FederationInstructionsParams) (appmodel.FederationResult, error) {
			gotKBID = params.KBID
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "nietzsche's guidance"}},
			}, nil
		},
	}
	env := &EnvMock{
		MCPMetricsFunc:      func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:     func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		FederationClientFunc: func(_ context.Context, kbID string) (appmodel.Federation, error) {
			require.Equal(t, "bob", kbID)
			return federation, nil
		},
		FederationMaxDepthFunc:          func() int { return 5 },
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	resp := callFederatedInstructions(t, env, `{"kb_id":"bob/nietzsche"}`)
	require.Nil(t, resp.Error)
	require.Equal(t, "nietzsche", gotKBID, "rest of the path must forward to the next hop")
	result := resp.Result.(mcp.CallToolResult)
	require.Equal(t, "nietzsche's guidance", result.Content[0].Text)
}

func TestFederatedInstructionsRespectsMaxDepth(t *testing.T) {
	nvs := fedInstrNoteViews()
	cache := fedinstr.New()
	federation := &federationMock{} // must never be called: depth is rejected up front
	env := &EnvMock{
		MCPMetricsFunc:                  func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc:             func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:                 func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		FederationClientFunc:            func(context.Context, string) (appmodel.Federation, error) { return federation, nil },
		FederationMaxDepthFunc:          func() int { return 2 },
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	// bob/nietzsche/deep = 3 segments > max depth 2.
	resp := callFederatedInstructions(t, env, `{"kb_id":"bob/nietzsche/deep"}`)
	require.NotNil(t, resp.Error)
	require.Equal(t, mcp.ErrCodeInternal, resp.Error.Code)
}

func TestFederatedInstructionsCacheHitSkipsForward(t *testing.T) {
	nvs := fedInstrNoteViews()
	cache := fedinstr.New()
	var calls int32
	federation := &federationMock{
		instructionsFunc: func(_ context.Context) (appmodel.FederationResult, error) {
			atomic.AddInt32(&calls, 1)
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "guidance once"}},
			}, nil
		},
	}
	env := &EnvMock{
		MCPMetricsFunc:                  func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc:             func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:                 func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		FederationClientFunc:            func(context.Context, string) (appmodel.Federation, error) { return federation, nil },
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	first := callFederatedInstructions(t, env, `{"kb_id":"bob"}`)
	require.Nil(t, first.Error)
	second := callFederatedInstructions(t, env, `{"kb_id":"bob"}`)
	require.Nil(t, second.Error)

	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "second call must be served from cache")
	require.Equal(t, "guidance once", second.Result.(mcp.CallToolResult).Content[0].Text)
}

func TestFederatedInstructionsCacheConcurrent(t *testing.T) {
	nvs := fedInstrNoteViews()
	cache := fedinstr.New()
	federation := &federationMock{
		instructionsFunc: func(_ context.Context) (appmodel.FederationResult, error) {
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "guidance"}},
			}, nil
		},
	}
	env := &EnvMock{
		MCPMetricsFunc:                  func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc:             func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:                 func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		FederationClientFunc:            func(context.Context, string) (appmodel.Federation, error) { return federation, nil },
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	// The calls run concurrently but are only inspected once the goroutines
	// have joined: testify's FailNow is unsafe off the test goroutine.
	var wg sync.WaitGroup
	bodies := make([][]byte, 20)
	errs := make([]error, 20)
	for i := range bodies {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bodies[i], errs[i] = rawMCPCall(env, federatedInstructionsRequest(`{"kb_id":"bob"}`))
		}()
	}
	wg.Wait()

	for i := range bodies {
		require.NoError(t, errs[i])
		resp := decodeMCPResponse(t, federatedInstructionsRequest(`{"kb_id":"bob"}`), bodies[i])
		require.Nil(t, resp.Error, "no concurrent call should error")
	}
}

func TestFederatedInstructionsUnknownKBID(t *testing.T) {
	nvs := fedInstrNoteViews()
	cache := fedinstr.New()
	env := &EnvMock{
		MCPMetricsFunc:                  func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc:             func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:                 func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	resp := callFederatedInstructions(t, env, `{"kb_id":"ghost"}`)
	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	payload := decodePayload[mcp.FederationStatusPayload](t, result)
	require.Equal(t, "federation_not_configured", payload.Status)
}

func TestFederatedInstructionsRequiresKBID(t *testing.T) {
	env := &EnvMock{MCPMetricsFunc: func() *metrics.MCPMetrics { return nil }}
	resp := callFederatedInstructions(t, env, `{}`)
	require.NotNil(t, resp.Error)
	require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
}
