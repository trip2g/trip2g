package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"trip2g/internal/case/mcp"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type fanoutSearchFn = func(context.Context, appmodel.FederationSearchParams) (appmodel.FederationResult, error)

// fanoutEnv builds an env with n federation KB notes (kb1..kbn, registration
// order) whose clients answer Search via searchFunc, plus the fan-out knobs.
func fanoutEnv(
	n int,
	limit, concurrency int,
	timeout time.Duration,
	searchFunc func(kbID string) fanoutSearchFn,
) *EnvMock {
	nvs := appmodel.NewNoteViews()
	for i := 1; i <= n; i++ {
		kbID := fmt.Sprintf("kb%d", i)
		note := &appmodel.NoteView{
			PathID:             int64(i),
			MCPFederationKBURL: fmt.Sprintf("https://%s.example/_system/mcp", kbID),
			MCPFederationKBID:  kbID,
		}
		nvs.MCPFederationNotes = append(nvs.MCPFederationNotes, appmodel.NewMCPFederationNote(note))
	}
	return &EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		FederationClientFunc: func(_ context.Context, kbID string) (appmodel.Federation, error) {
			return &federationMock{searchFunc: searchFunc(kbID)}, nil
		},
		FederatedFanoutLimitFunc:       func() int { return limit },
		FederatedFanoutConcurrencyFunc: func() int { return concurrency },
		FederatedFanoutTimeoutFunc:     func() time.Duration { return timeout },
	}
}

func callFederatedSearch(t *testing.T, env *EnvMock, arguments string) mcp.CallToolResult {
	t.Helper()
	params := mcp.CallToolParams{
		Name:      "federated_search",
		Arguments: json.RawMessage(arguments),
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)
	resp := mcp.Resolve(context.Background(), env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})
	require.Nil(t, resp.Error)
	result, ok := resp.Result.(mcp.CallToolResult)
	require.True(t, ok, "expected CallToolResult, got %T", resp.Result)
	return result
}

func fanoutPayload(t *testing.T, result mcp.CallToolResult) mcp.FederatedCallPayload {
	t.Helper()
	payload, ok := result.StructuredContent.(mcp.FederatedCallPayload)
	require.True(t, ok, "expected FederatedCallPayload, got %T", result.StructuredContent)
	return payload
}

func TestFederatedSearchBlindFanoutCapsPeersAndReportsSkipped(t *testing.T) {
	var mu sync.Mutex
	queried := map[string]bool{}
	env := fanoutEnv(5, 3, 5, time.Second, func(kbID string) fanoutSearchFn {
		return func(_ context.Context, _ appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			mu.Lock()
			queried[kbID] = true
			mu.Unlock()
			return appmodel.FederationResult{Content: []appmodel.FederationContent{{Type: "text", Text: "hit " + kbID}}}, nil
		}
	})

	result := callFederatedSearch(t, env, `{"query":"status"}`)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, queried, 3, "blind fan-out must query at most the limit")
	require.Equal(t, map[string]bool{"kb1": true, "kb2": true, "kb3": true}, queried,
		"cap must keep the first peers in registration order")

	payload := fanoutPayload(t, result)
	require.Empty(t, payload.Errors)
	skippedIDs := make([]string, 0, len(payload.Skipped))
	for _, s := range payload.Skipped {
		require.Equal(t, "fanout_limit", s.Reason)
		skippedIDs = append(skippedIDs, s.KBID)
	}
	require.Equal(t, []string{"kb4", "kb5"}, skippedIDs,
		"un-queried peers must be reported, not dropped silently")
}

func TestFederatedSearchFanoutConcurrencyBounded(t *testing.T) {
	var inflight, peak atomic.Int64
	env := fanoutEnv(6, 10, 2, 5*time.Second, func(kbID string) fanoutSearchFn {
		return func(_ context.Context, _ appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			cur := inflight.Add(1)
			for {
				old := peak.Load()
				if cur <= old || peak.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			inflight.Add(-1)
			return appmodel.FederationResult{Content: []appmodel.FederationContent{{Type: "text", Text: kbID}}}, nil
		}
	})

	result := callFederatedSearch(t, env, `{"query":"status"}`)

	payload := fanoutPayload(t, result)
	require.Len(t, payload.Results, 6)
	require.LessOrEqual(t, peak.Load(), int64(2),
		"no more than FederatedFanoutConcurrency peers may be in flight at once")
}

func TestFederatedSearchHungPeerTimesOutOthersReturn(t *testing.T) {
	hung := make(chan struct{})
	t.Cleanup(func() { close(hung) })
	env := fanoutEnv(3, 10, 5, 100*time.Millisecond, func(kbID string) fanoutSearchFn {
		return func(_ context.Context, _ appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			if kbID == "kb2" {
				<-hung
				return appmodel.FederationResult{}, context.Canceled
			}
			return appmodel.FederationResult{Content: []appmodel.FederationContent{{Type: "text", Text: "fast " + kbID}}}, nil
		}
	})

	start := time.Now()
	result := callFederatedSearch(t, env, `{"query":"status"}`)
	require.Less(t, time.Since(start), 2*time.Second, "a hung peer must not block the fan-out")

	payload := fanoutPayload(t, result)
	gotResults := make([]string, 0, len(payload.Results))
	for _, r := range payload.Results {
		gotResults = append(gotResults, r.KBID)
	}
	require.ElementsMatch(t, []string{"kb1", "kb3"}, gotResults, "fast peers must still return results")
	require.Len(t, payload.Errors, 1)
	require.Equal(t, "kb2", payload.Errors[0].KBID)
	require.Contains(t, payload.Errors[0].Error, "deadline exceeded", "hung peer must be reported as timed-out")
	require.Empty(t, payload.Skipped, "a timed-out peer is not a cap-skipped peer")
}

func TestFederatedSearchExplicitKBIDsIgnoreFanoutLimit(t *testing.T) {
	var mu sync.Mutex
	queried := map[string]bool{}
	env := fanoutEnv(5, 2, 5, time.Second, func(kbID string) fanoutSearchFn {
		return func(_ context.Context, _ appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
			mu.Lock()
			queried[kbID] = true
			mu.Unlock()
			return appmodel.FederationResult{Content: []appmodel.FederationContent{{Type: "text", Text: kbID}}}, nil
		}
	})

	result := callFederatedSearch(t, env, `{"query":"status","kb_ids":["kb1","kb2","kb3","kb4"]}`)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, map[string]bool{"kb1": true, "kb2": true, "kb3": true, "kb4": true}, queried,
		"explicit kb_ids targeting must not be capped by the fan-out limit")

	payload := fanoutPayload(t, result)
	require.Len(t, payload.Results, 4)
	require.Empty(t, payload.Skipped)
}
