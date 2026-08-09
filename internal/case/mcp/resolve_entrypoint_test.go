package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"trip2g/internal/case/mcp"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// Resolve is the package's use-case entry point: it answers a JSON-RPC message
// for an already-authenticated context, without any HTTP of the caller's own.
// Endpoint.Handle is the HTTP adapter over the same core, so these cases pin
// the entry point itself rather than the transport around it.
func TestResolveEntryPoint(t *testing.T) {
	env := func() *EnvMock {
		note := &appmodel.NoteView{
			Path:      "guide.md",
			PathID:    1,
			MCPMethod: "guide",
			Content:   []byte("---\nmcp_method: guide\n---\n\nGuide body."),
		}
		views := appmodel.NewNoteViews()
		views.List = []*appmodel.NoteView{note}
		views.PathMap[note.Path] = note

		return &EnvMock{
			MCPMetricsFunc:              func() *metrics.MCPMetrics { return nil },
			LoggerFunc:                  func() logger.Logger { return &logger.DummyLogger{} },
			LatestNoteViewsFunc:         func() *appmodel.NoteViews { return views },
			CanReadNoteFunc:             func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
			FederationMaxDepthFunc:      func() int { return 3 },
			FederatedGraphQLEnabledFunc: func() bool { return false },
			SiteConfigFunc:              func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		}
	}

	t.Run("lists tools", func(t *testing.T) {
		resp := mcp.Resolve(context.Background(), env(), mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/list",
			ID:      1,
		})

		require.Nil(t, resp.Error)
		var result mcp.ListToolsResult
		remarshal(t, resp.Result, &result)
		require.NotEmpty(t, result.Tools)
	})

	t.Run("calls a note-registered tool", func(t *testing.T) {
		params, err := json.Marshal(mcp.CallToolParams{Name: "guide", Arguments: json.RawMessage(`{}`)})
		require.NoError(t, err)

		resp := mcp.Resolve(context.Background(), env(), mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  params,
			ID:      2,
		})

		require.Nil(t, resp.Error)
		var result mcp.CallToolResult
		remarshal(t, resp.Result, &result)
		require.Len(t, result.Content, 1)
		require.Equal(t, "Guide body.", result.Content[0].Text)
	})

	t.Run("reports an unknown tool as method not found", func(t *testing.T) {
		params, err := json.Marshal(mcp.CallToolParams{Name: "nope", Arguments: json.RawMessage(`{}`)})
		require.NoError(t, err)

		resp := mcp.Resolve(context.Background(), env(), mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  params,
			ID:      3,
		})

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeMethodNotFound, resp.Error.Code)
		require.Equal(t, "Method not found: nope", resp.Error.Message)
	})

	// MethodOverride is what ?method= feeds in: it picks which note supplies the
	// initialize instructions.
	t.Run("serves instructions from the overridden note", func(t *testing.T) {
		resp := mcp.Resolve(context.Background(), env(), mcp.Request{
			JSONRPC:        "2.0",
			Method:         "initialize",
			ID:             4,
			MethodOverride: "guide",
		})

		require.Nil(t, resp.Error)
		var result map[string]any
		remarshal(t, resp.Result, &result)
		require.Equal(t, "Guide body.", result["instructions"])
	})

	t.Run("answers a notification with nothing", func(t *testing.T) {
		resp := mcp.Resolve(context.Background(), env(), mcp.Request{
			JSONRPC: "2.0",
			Method:  "notifications/initialized",
		})

		require.Nil(t, resp.Error)
		require.Empty(t, resp.JSONRPC)
	})
}

// remarshal re-decodes a Result into a concrete type. Resolve returns whatever
// the JSON envelope held, so callers narrow it themselves.
func remarshal(t *testing.T, from any, into any) {
	t.Helper()

	encoded, err := json.Marshal(from)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, into))
}
