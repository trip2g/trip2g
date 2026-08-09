package mcp_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"

	"trip2g/internal/case/mcp"
	"trip2g/internal/fedinstr"
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

func callFederatedInstructions(env mcp.Env, args string) mcp.Response {
	params := mcp.CallToolParams{Name: "federated_instructions", Arguments: json.RawMessage(args)}
	paramsJSON, _ := json.Marshal(params)
	return mcp.ResolveForTest(context.Background(), env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})
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
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:     func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		FederationClientFunc: func(_ context.Context, kbID string) (appmodel.Federation, error) {
			require.Equal(t, "bob", kbID)
			return federation, nil
		},
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	resp := callFederatedInstructions(env, `{"kb_id":"bob"}`)
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

	resp := callFederatedInstructions(env, `{"kb_id":"bob/nietzsche"}`)
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
		LatestNoteViewsFunc:             func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:                 func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		FederationClientFunc:            func(context.Context, string) (appmodel.Federation, error) { return federation, nil },
		FederationMaxDepthFunc:          func() int { return 2 },
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	// bob/nietzsche/deep = 3 segments > max depth 2.
	resp := callFederatedInstructions(env, `{"kb_id":"bob/nietzsche/deep"}`)
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
		LatestNoteViewsFunc:             func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:                 func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		FederationClientFunc:            func(context.Context, string) (appmodel.Federation, error) { return federation, nil },
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	first := callFederatedInstructions(env, `{"kb_id":"bob"}`)
	require.Nil(t, first.Error)
	second := callFederatedInstructions(env, `{"kb_id":"bob"}`)
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
		LatestNoteViewsFunc:             func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:                 func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		FederationClientFunc:            func(context.Context, string) (appmodel.Federation, error) { return federation, nil },
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	var wg sync.WaitGroup
	var failures int32
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if callFederatedInstructions(env, `{"kb_id":"bob"}`).Error != nil {
				atomic.AddInt32(&failures, 1)
			}
		}()
	}
	wg.Wait()
	require.Zero(t, atomic.LoadInt32(&failures), "no concurrent call should error")
}

func TestFederatedInstructionsUnknownKBID(t *testing.T) {
	nvs := fedInstrNoteViews()
	cache := fedinstr.New()
	env := &EnvMock{
		LatestNoteViewsFunc:             func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc:                 func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		CachedFederatedInstructionsFunc: cache.CachedFederatedInstructions,
		StoreFederatedInstructionsFunc:  cache.StoreFederatedInstructions,
	}

	resp := callFederatedInstructions(env, `{"kb_id":"ghost"}`)
	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	payload := decodePayload[mcp.FederationStatusPayload](t, result)
	require.Equal(t, "federation_not_configured", payload.Status)
}

func TestFederatedInstructionsRequiresKBID(t *testing.T) {
	env := &EnvMock{}
	resp := callFederatedInstructions(env, `{}`)
	require.NotNil(t, resp.Error)
	require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
}
