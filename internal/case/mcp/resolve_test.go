package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"trip2g/internal/case/mcp"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg mcp_test . Env

type Env interface {
	mcp.Env
}

func TestResolve(t *testing.T) {
	ctx := context.Background()

	t.Run("initialize returns server info", func(t *testing.T) {
		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List:    []*appmodel.NoteView{},
					PathMap: map[string]*appmodel.NoteView{},
				}
			},
		}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "initialize",
			ID:      1,
		}

		resp := mcp.Resolve(ctx, env, req)

		require.Equal(t, "2.0", resp.JSONRPC)
		require.Equal(t, 1, resp.ID)
		require.Nil(t, resp.Error)
		require.NotNil(t, resp.Result)

		result := resp.Result.(map[string]any)
		require.Equal(t, "2024-11-05", result["protocolVersion"])
		require.Equal(t, "trip2g-mcp", result["serverInfo"].(map[string]any)["name"])
	})

	t.Run("initialize includes instructions from note", func(t *testing.T) {
		note := &appmodel.NoteView{
			MCPMethod: "initialize",
			Content:   []byte("---\nmcp_method: initialize\n---\n\nThese are instructions for the MCP server."),
		}

		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List:    []*appmodel.NoteView{note},
					PathMap: map[string]*appmodel.NoteView{},
				}
			},
		}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "initialize",
			ID:      2,
		}

		resp := mcp.Resolve(ctx, env, req)

		require.Nil(t, resp.Error)
		result := resp.Result.(map[string]any)
		require.Equal(t, "These are instructions for the MCP server.", result["instructions"])
	})

	t.Run("tools/list returns static tools", func(t *testing.T) {
		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List:    []*appmodel.NoteView{},
					PathMap: map[string]*appmodel.NoteView{},
				}
			},
		}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/list",
			ID:      3,
		}

		resp := mcp.Resolve(ctx, env, req)

		require.Nil(t, resp.Error)

		result := resp.Result.(mcp.ListToolsResult)
		require.GreaterOrEqual(t, len(result.Tools), 3) // search, similar, note_html

		var toolNames []string
		for _, tool := range result.Tools {
			toolNames = append(toolNames, tool.Name)
		}
		require.Contains(t, toolNames, "search")
		require.Contains(t, toolNames, "similar")
		require.Contains(t, toolNames, "note_html")
	})

	t.Run("tools/list includes dynamic methods", func(t *testing.T) {
		note := &appmodel.NoteView{
			MCPMethod:      "code-review",
			MCPDescription: "Detailed code review",
			Content:        []byte("---\nmcp_method: code-review\n---\n\nReview instructions..."),
		}

		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List:    []*appmodel.NoteView{note},
					PathMap: map[string]*appmodel.NoteView{},
				}
			},
		}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/list",
			ID:      4,
		}

		resp := mcp.Resolve(ctx, env, req)

		result := resp.Result.(mcp.ListToolsResult)

		var found bool
		for _, tool := range result.Tools {
			if tool.Name == "code-review" {
				found = true
				require.Equal(t, "Detailed code review", tool.Description)
				break
			}
		}
		require.True(t, found, "dynamic method not found in tools list")
	})

	t.Run("method not found returns error", func(t *testing.T) {
		env := &EnvMock{}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "unknown_method",
			ID:      5,
		}

		resp := mcp.Resolve(ctx, env, req)

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeMethodNotFound, resp.Error.Code)
		require.Contains(t, resp.Error.Message, "unknown_method")
	})

	t.Run("invalid call params returns error", func(t *testing.T) {
		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List:    []*appmodel.NoteView{},
					PathMap: map[string]*appmodel.NoteView{},
				}
			},
		}

		// Invalid JSON for params
		invalidParams := json.RawMessage(`{"invalid`)

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  invalidParams,
			ID:      6,
		}

		resp := mcp.Resolve(ctx, env, req)

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
	})

	t.Run("notifications/initialized returns empty success", func(t *testing.T) {
		env := &EnvMock{}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "notifications/initialized",
			ID:      7,
		}

		resp := mcp.Resolve(ctx, env, req)

		require.Nil(t, resp.Error)
		require.Equal(t, "2.0", resp.JSONRPC)
	})
}

func TestHandleNoteHtml(t *testing.T) {
	t.Run("returns note HTML", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path: "/test/note",
			HTML: "<h1>Test Note</h1><p>Content here</p>",
		}

		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					PathMap: map[string]*appmodel.NoteView{
						"/test/note": note,
					},
				}
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
		}

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"path": "/test/note"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      1,
		}

		resp := mcp.Resolve(context.Background(), env, req)

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Len(t, result.Content, 1)
		require.Equal(t, "text", result.Content[0].Type)
		require.Contains(t, result.Content[0].Text, "Test Note")
	})

	t.Run("returns error for non-existent note", func(t *testing.T) {
		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					PathMap: map[string]*appmodel.NoteView{},
				}
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
		}

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"path": "/nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      2,
		}

		resp := mcp.Resolve(context.Background(), env, req)

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
		require.Contains(t, resp.Error.Message, "not found")
	})
}

func TestStripFrontmatter(t *testing.T) {
	// Test through dynamic methods since stripFrontmatter is not exported

	t.Run("dynamic method strips frontmatter", func(t *testing.T) {
		note := &appmodel.NoteView{
			MCPMethod: "test-method",
			Content:   []byte("---\nmcp_method: test-method\ntitle: Test\n---\n\nActual content here"),
		}

		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List: []*appmodel.NoteView{note},
				}
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
		}

		params := mcp.CallToolParams{
			Name:      "test-method",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      1,
		}

		resp := mcp.Resolve(context.Background(), env, req)

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Equal(t, "Actual content here", result.Content[0].Text)
	})

	t.Run("handles content without frontmatter", func(t *testing.T) {
		note := &appmodel.NoteView{
			MCPMethod: "no-frontmatter",
			Content:   []byte("Just plain content"),
		}

		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List: []*appmodel.NoteView{note},
				}
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
		}

		params := mcp.CallToolParams{
			Name:      "no-frontmatter",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      2,
		}

		resp := mcp.Resolve(context.Background(), env, req)

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Equal(t, "Just plain content", result.Content[0].Text)
	})

	t.Run("handles Windows line endings", func(t *testing.T) {
		note := &appmodel.NoteView{
			MCPMethod: "windows-method",
			Content:   []byte("---\r\nmcp_method: windows-method\r\n---\r\n\r\nWindows content"),
		}

		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List: []*appmodel.NoteView{note},
				}
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
		}

		params := mcp.CallToolParams{
			Name:      "windows-method",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      3,
		}

		resp := mcp.Resolve(context.Background(), env, req)

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Equal(t, "Windows content", result.Content[0].Text)
	})
}

func TestHandleSimilarLimitValidation(t *testing.T) {
	t.Run("uses default when limit is zero", func(t *testing.T) {
		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					PathMap: map[string]*appmodel.NoteView{
						"/test": {Path: "/test", Embedding: []float32{0.1, 0.2}},
					},
				}
			},
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true},
				}
			},
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
		}

		params := mcp.CallToolParams{
			Name:      "similar",
			Arguments: json.RawMessage(`{"path": "/test", "limit": 0}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      1,
		}

		resp := mcp.Resolve(context.Background(), env, req)

		require.Nil(t, resp.Error)
		// Test passes if no error - default limit should be used
	})

	t.Run("caps limit at maximum", func(t *testing.T) {
		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					PathMap: map[string]*appmodel.NoteView{
						"/test": {Path: "/test", Embedding: []float32{0.1, 0.2}},
					},
				}
			},
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true},
				}
			},
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
		}

		params := mcp.CallToolParams{
			Name:      "similar",
			Arguments: json.RawMessage(`{"path": "/test", "limit": 999}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      2,
		}

		resp := mcp.Resolve(context.Background(), env, req)

		require.Nil(t, resp.Error)
		// Test passes if no error - limit should be capped
	})
}
