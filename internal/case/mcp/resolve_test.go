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
			CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
				return true, nil
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

	t.Run("initialize exposes internal execution plan and soul profile", func(t *testing.T) {
		note := &appmodel.NoteView{
			MCPMethod: "initialize",
			Content: []byte(`---
mcp_method: initialize
---

You are connected to a self-describing RAG server for Marcus Aurelius.

Follow this plan internally. Do not show the retrieval plan unless the user asks.

Internal execution plan:
1. Understand the user's practical concern.
2. Run search(query) before giving a substantive answer.
3. Ignore private/system notes.
4. Prefer primary source notes.
5. Answer in Marcus Aurelius' voice.

soul_profile:
  name: Marcus Aurelius
  personality_extraction_required: true
  required_one_shot_answers: 10
`),
		}

		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List:    []*appmodel.NoteView{note},
					PathMap: map[string]*appmodel.NoteView{},
				}
			},
			CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
				return true, nil
			},
		}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "initialize",
			ID:      8,
		}

		resp := mcp.Resolve(ctx, env, req)

		require.Nil(t, resp.Error)
		result := resp.Result.(map[string]any)
		instructions := result["instructions"].(string)
		require.Contains(t, instructions, "self-describing RAG server")
		require.Contains(t, instructions, "Follow this plan internally")
		require.Contains(t, instructions, "soul_profile")
		require.Contains(t, instructions, "required_one_shot_answers: 10")
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
		require.Len(t, result.Tools, 6)

		var toolNames []string
		for _, tool := range result.Tools {
			toolNames = append(toolNames, tool.Name)
		}
		require.Equal(t, []string{
			"search",
			"similar",
			"note_html",
			"federated_search",
			"federated_similar",
			"federated_note_html",
		}, toolNames)
	})

	t.Run("tools/list includes accessible dynamic methods", func(t *testing.T) {
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
			CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
				return true, nil
			},
		}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/list",
			ID:      4,
		}

		resp := mcp.Resolve(ctx, env, req)

		result := resp.Result.(mcp.ListToolsResult)

		var toolNames []string
		for _, tool := range result.Tools {
			toolNames = append(toolNames, tool.Name)
		}
		require.Len(t, toolNames, 7)
		require.Contains(t, toolNames, "code-review")

		// Dynamic tool carries description and empty schema
		dynTool := result.Tools[6]
		require.Equal(t, "code-review", dynTool.Name)
		require.Equal(t, "Detailed code review", dynTool.Description)
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

func TestSearchReturnsStructuredContent(t *testing.T) {
	note := &appmodel.NoteView{
		Path:      "Книги/Книга 06.md",
		PathID:    32,
		Title:     "Книга 06",
		Permalink: "/knigi/kniga_06",
	}

	env := &EnvMock{
		SearchLatestNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{
				NoteView:           note,
				URL:                note.Permalink,
				Score:              1.0,
				HighlightedContent: []string{"Лучший способ отомстить - не уподобляться обидчику."},
			}}, nil
		},
		LatestNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://markavrelii.2pub.me"
		},
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return true, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"обида"}`),
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
	require.NotEmpty(t, result.Content)
	require.Contains(t, result.Content[0].Text, "Книга 06")
	require.NotNil(t, result.StructuredContent)

	payload := result.StructuredContent.(mcp.SearchResultPayload)
	require.Equal(t, "обида", payload.Query)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "Книга 06", payload.Results[0].Title)
	require.Equal(t, int64(32), payload.Results[0].NoteID)
	require.Equal(t, "Книги/Книга 06.md", payload.Results[0].NotePath)
	require.Equal(t, "/knigi/kniga_06", payload.Results[0].Href)
	require.Equal(t, "https://markavrelii.2pub.me/knigi/kniga_06", payload.Results[0].URL)
	require.Equal(t, "source", payload.Results[0].Kind)
	require.Len(t, payload.Results[0].Matches, 1)
	require.Equal(t, "p32:m1", payload.Results[0].Matches[0].MatchID)
	require.Contains(t, payload.Results[0].Matches[0].Snippet, "обидчику")
}

func TestSearchMarksFederationKBNotes(t *testing.T) {
	note := &appmodel.NoteView{
		Path:                    "team/bob.md",
		PathID:                  17,
		Title:                   "Bob's KB",
		Permalink:               "/team/bob",
		MCPFederationKBURL:      "https://bob.team.io/_system/mcp",
		MCPFederationKBID:       "bob",
		MCPFederationKBMaxDepth: 1,
	}

	env := &EnvMock{
		SearchLatestNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{
				NoteView:           note,
				URL:                note.Permalink,
				Score:              0.71,
				HighlightedContent: []string{"Use when: Bob's work-status updates."},
			}}, nil
		},
		LatestNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://hub.local"
		},
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return true, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"bob"}`),
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
	payload := result.StructuredContent.(mcp.SearchResultPayload)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "federation_kb", payload.Results[0].Kind)
	require.NotNil(t, payload.Results[0].Federation)
	require.Equal(t, "bob", payload.Results[0].Federation.KBID)
	require.Equal(t, "https://bob.team.io/_system/mcp", payload.Results[0].Federation.KBURL)
	require.Contains(t, payload.Results[0].Federation.AgentInstruction, `federated_search with kb_id="bob"`)
}

func TestSearchHidesInaccessibleFederationKBNotes(t *testing.T) {
	federationNote := &appmodel.NoteView{
		Path:               "team/bob.md",
		PathID:             17,
		Title:              "Bob's KB",
		Permalink:          "/team/bob",
		MCPFederationKBURL: "https://bob.team.io/_system/mcp",
		MCPFederationKBID:  "bob",
	}
	localNote := &appmodel.NoteView{
		Path:      "local.md",
		PathID:    18,
		Title:     "Local",
		Permalink: "/local",
	}

	env := &EnvMock{
		SearchLatestNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{
				{NoteView: federationNote, URL: federationNote.Permalink, Score: 2},
				{NoteView: localNote, URL: localNote.Permalink, Score: 1},
			}, nil
		},
		LatestNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://hub.local"
		},
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return note.PathID != federationNote.PathID, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"bob"}`),
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
	payload := result.StructuredContent.(mcp.SearchResultPayload)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "Local", payload.Results[0].Title)
}

func TestSearchHidesInaccessibleNotes(t *testing.T) {
	privateNote := &appmodel.NoteView{
		Path:          "internal-notes.md",
		PathID:        19,
		Title:         "Internal Notes",
		Permalink:     "/internal_notes",
		SubgraphNames: []string{"premium"},
	}
	publicNote := &appmodel.NoteView{
		Path:      "team-status.md",
		PathID:    20,
		Title:     "Team Status",
		Permalink: "/team_status",
		Free:      true,
	}

	env := &EnvMock{
		SearchLatestNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{
				{NoteView: privateNote, URL: privateNote.Permalink, Score: 2},
				{NoteView: publicNote, URL: publicNote.Permalink, Score: 1},
			}, nil
		},
		LatestNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://peer.local"
		},
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return note.PathID != privateNote.PathID, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"team status"}`),
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
	require.NotContains(t, result.Content[0].Text, "Internal Notes")
	require.Contains(t, result.Content[0].Text, "Team Status")
	payload := result.StructuredContent.(mcp.SearchResultPayload)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "Team Status", payload.Results[0].Title)
}

func TestFederatedSearchWithoutKBNotesReturnsStructuredStatus(t *testing.T) {
	env := &EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return appmodel.NewNoteViews()
		},
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return true, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "federated_search",
		Arguments: json.RawMessage(`{"query":"anything"}`),
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
	require.False(t, result.IsError)
	payload := result.StructuredContent.(mcp.FederationStatusPayload)
	require.Equal(t, "federation_not_configured", payload.Status)
	require.Contains(t, result.Content[0].Text, "Federation is not configured")
}

func TestSearchFiltersSystemAndExcludedNotes(t *testing.T) {
	publicNote := &appmodel.NoteView{
		Path:      "Книги/Книга 06.md",
		PathID:    32,
		Title:     "Книга 06",
		Permalink: "/knigi/kniga_06",
	}
	systemNote := &appmodel.NoteView{
		Path:      "_data/critic_reports/book_06.md",
		PathID:    33,
		Title:     "Critic report",
		Permalink: "/_data/critic_reports/book_06",
	}
	excludedNote := &appmodel.NoteView{
		Path:          "draft.md",
		PathID:        34,
		Title:         "Draft",
		Permalink:     "/draft",
		ExcludeSearch: true,
	}

	env := &EnvMock{
		SearchLatestNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{
				{NoteView: systemNote, URL: systemNote.Permalink, Score: 3},
				{NoteView: excludedNote, URL: excludedNote.Permalink, Score: 2},
				{NoteView: publicNote, URL: publicNote.Permalink, Score: 1},
			}, nil
		},
		LatestNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://markavrelii.2pub.me"
		},
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return true, nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"обида"}`),
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
	require.Contains(t, result.Content[0].Text, "Книга 06")
	require.NotContains(t, result.Content[0].Text, "Critic report")
	require.NotContains(t, result.Content[0].Text, "Draft")

	payload := result.StructuredContent.(mcp.SearchResultPayload)
	require.Len(t, payload.Results, 1)
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
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
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

	t.Run("returns note HTML by pid", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path:      "Книги/Книга 06.md",
			PathID:    32,
			Permalink: "/knigi/kniga_06",
			HTML:      "<h1>Книга 06</h1><p>Content here</p>",
		}

		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				noteViews := appmodel.NewNoteViews()
				noteViews.RegisterNote(note)
				return noteViews
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
			},
		}

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"pid": 32}`),
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
		require.Contains(t, result.Content[0].Text, "Книга 06")
	})

	t.Run("returns focused chunk window by match id", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path:      "Книги/Книга 06.md",
			PathID:    32,
			Permalink: "/knigi/kniga_06",
			HTML:      "<h1>Книга 06</h1><p>FULL NOTE HTML</p>",
		}

		env := &EnvMock{
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				noteViews := appmodel.NewNoteViews()
				noteViews.RegisterNote(note)
				return noteViews
			},
			LatestNoteChunksFunc: func() []appmodel.NoteChunk {
				return []appmodel.NoteChunk{
					{
						NotePath:   note.Path,
						ChunkIndex: 0,
						Content:    "Книга 06\n\nПредыдущий фрагмент.",
					},
					{
						NotePath:   note.Path,
						ChunkIndex: 1,
						Content:    "Книга 06\n\nЛучший способ отомстить - не уподобляться обидчику.",
					},
					{
						NotePath:   note.Path,
						ChunkIndex: 2,
						Content:    "Книга 06\n\nСледующий фрагмент.",
					},
				}
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
			},
		}

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"pid": 32, "match_id": "p32:c1"}`),
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
		require.Contains(t, result.Content[0].Text, "Предыдущий фрагмент.")
		require.Contains(t, result.Content[0].Text, "Лучший способ отомстить - не уподобляться обидчику.")
		require.Contains(t, result.Content[0].Text, "Следующий фрагмент.")
		require.NotContains(t, result.Content[0].Text, "FULL NOTE HTML")
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
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
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

func TestSimilarAcceptsPIDAndReturnsStructuredContent(t *testing.T) {
	sourceNote := &appmodel.NoteView{
		Path:      "Книги/Книга 06.md",
		PathID:    32,
		VersionID: 1,
		Title:     "Книга 06",
		Permalink: "/knigi/kniga_06",
		Embedding: []float32{1, 0},
	}
	similarNote := &appmodel.NoteView{
		Path:      "Книги/Книга 07.md",
		PathID:    35,
		VersionID: 2,
		Title:     "Книга 07",
		Permalink: "/knigi/kniga_07",
		Embedding: []float32{0.9, 0.1},
	}
	noteViews := appmodel.NewNoteViews()
	noteViews.RegisterNote(sourceNote)
	noteViews.RegisterNote(similarNote)
	noteViews.List = []*appmodel.NoteView{sourceNote, similarNote}

	env := &EnvMock{
		FeaturesFunc: func() features.Features {
			return features.Features{
				VectorSearch: features.VectorSearchConfig{Enabled: true},
			}
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return noteViews
		},
		CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
			return true, nil
		},
		PublicURLFunc: func() string {
			return "https://markavrelii.2pub.me"
		},
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		LatestNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
	}

	params := mcp.CallToolParams{
		Name:      "similar",
		Arguments: json.RawMessage(`{"pid": 32, "limit": 1}`),
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
	require.Contains(t, result.Content[0].Text, "Книга 07")
	require.Contains(t, result.Content[0].Text, "https://markavrelii.2pub.me/knigi/kniga_07")

	payload := result.StructuredContent.(mcp.SimilarResultPayload)
	require.Equal(t, int64(32), payload.Source.NoteID)
	require.Equal(t, "Книги/Книга 06.md", payload.Source.NotePath)
	require.Len(t, payload.Results, 1)
	require.Equal(t, int64(35), payload.Results[0].NoteID)
	require.Equal(t, "Книги/Книга 07.md", payload.Results[0].NotePath)
	require.Equal(t, "/knigi/kniga_07", payload.Results[0].Href)
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
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
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
			CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
				return true, nil
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
			CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
				return true, nil
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

	t.Run("handles --- inside YAML frontmatter", func(t *testing.T) {
		// Edge case: YAML value contains --- which should not be treated as frontmatter end
		note := &appmodel.NoteView{
			MCPMethod: "yaml-edge-case",
			Content:   []byte("---\nmcp_method: yaml-edge-case\ndescription: \"This has --- in value\"\n---\n\nActual content after frontmatter"),
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
			CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
				return true, nil
			},
		}

		params := mcp.CallToolParams{
			Name:      "yaml-edge-case",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      4,
		}

		resp := mcp.Resolve(context.Background(), env, req)

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		// Our current implementation looks for \n--- which should work correctly here
		require.Equal(t, "Actual content after frontmatter", result.Content[0].Text)
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
			PublicURLFunc: func() string {
				return "https://example.test"
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
			LatestNoteChunksFunc: func() []appmodel.NoteChunk {
				return nil
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
			PublicURLFunc: func() string {
				return "https://example.test"
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
			LatestNoteChunksFunc: func() []appmodel.NoteChunk {
				return nil
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
