package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
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
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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
			Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}`),
		}

		resp := mcp.ResolveForTest(ctx, env, req)

		require.Equal(t, "2.0", resp.JSONRPC)
		require.Equal(t, 1, resp.ID)
		require.Nil(t, resp.Error)
		require.NotNil(t, resp.Result)

		result := resp.Result.(map[string]any)
		// The protocol version is negotiated: a client asking for 2024-11-05 keeps it.
		require.Equal(t, "2024-11-05", result["protocolVersion"])
		require.Equal(t, "trip2g-mcp", result["serverInfo"].(map[string]any)["name"])
		require.NotNil(t, result["capabilities"].(map[string]any)["tools"], "tools capability must stay advertised")
	})

	t.Run("initialize includes instructions from note", func(t *testing.T) {
		note := &appmodel.NoteView{
			MCPMethod: "initialize",
			Content:   []byte("---\nmcp_method: initialize\n---\n\nThese are instructions for the MCP server."),
		}

		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(ctx, env, req)

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
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(ctx, env, req)

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
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List:    []*appmodel.NoteView{},
					PathMap: map[string]*appmodel.NoteView{},
				}
			},
			FederatedGraphQLEnabledFunc: func() bool { return false },
			FederationMaxDepthFunc:      func() int { return 3 },
		}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/list",
			ID:      3,
		}

		resp := mcp.ResolveForTest(ctx, env, req)

		require.Nil(t, resp.Error)

		result := resp.Result.(mcp.ListToolsResult)
		require.Len(t, result.Tools, 9)

		var toolNames []string
		for _, tool := range result.Tools {
			toolNames = append(toolNames, tool.Name)
		}
		// The transport lists tools in name order, so compare as a set.
		require.ElementsMatch(t, []string{
			"search",
			"similar",
			"note_html",
			"expand",
			"federated_search",
			"federated_similar",
			"federated_note_html",
			"federated_expand",
			"federated_instructions",
		}, toolNames)
	})

	t.Run("tools/list includes accessible dynamic methods", func(t *testing.T) {
		note := &appmodel.NoteView{
			MCPMethod:      "code-review",
			MCPDescription: "Detailed code review",
			Content:        []byte("---\nmcp_method: code-review\n---\n\nReview instructions..."),
		}

		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					List:    []*appmodel.NoteView{note},
					PathMap: map[string]*appmodel.NoteView{},
				}
			},
			CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
				return true, nil
			},
			FederatedGraphQLEnabledFunc: func() bool { return false },
			FederationMaxDepthFunc:      func() int { return 3 },
		}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/list",
			ID:      4,
		}

		resp := mcp.ResolveForTest(ctx, env, req)

		result := resp.Result.(mcp.ListToolsResult)

		var toolNames []string
		for _, tool := range result.Tools {
			toolNames = append(toolNames, tool.Name)
		}
		require.Len(t, toolNames, 10)
		require.Contains(t, toolNames, "code-review")

		// Dynamic tool carries description and empty schema
		var dynTool mcp.Tool
		for _, tool := range result.Tools {
			if tool.Name == "code-review" {
				dynTool = tool
			}
		}
		require.Equal(t, "code-review", dynTool.Name)
		require.Equal(t, "Detailed code review", dynTool.Description)
	})

	t.Run("tools/list dedupes notes sharing an mcp_method", func(t *testing.T) {
		// Localized en/ru notes both declaring mcp_method: instructions must not
		// produce two tools with the same name. The first note in path-sorted
		// order wins, matching handleDynamicMethod's resolution.
		en := &appmodel.NoteView{
			Path:           "_mcp_instructions.md",
			MCPMethod:      "instructions",
			MCPDescription: "Full tool reference (EN)",
			Content:        []byte("EN instructions"),
		}
		ru := &appmodel.NoteView{
			Path:           "ru/_mcp_instructions.md",
			MCPMethod:      "instructions",
			MCPDescription: "MCP Инструкции (RU)",
			Content:        []byte("RU instructions"),
		}
		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{List: []*appmodel.NoteView{en, ru}, PathMap: map[string]*appmodel.NoteView{}}
			},
			CanReadNoteFunc:             func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
			FederatedGraphQLEnabledFunc: func() bool { return false },
			FederationMaxDepthFunc:      func() int { return 3 },
		}

		resp := mcp.ResolveForTest(ctx, env, mcp.Request{JSONRPC: "2.0", Method: "tools/list", ID: 6})
		result := resp.Result.(mcp.ListToolsResult)

		count := 0
		var desc string
		for _, tool := range result.Tools {
			if tool.Name == "instructions" {
				count++
				desc = tool.Description
			}
		}
		require.Equal(t, 1, count, "instructions must appear exactly once")
		require.Equal(t, "Full tool reference (EN)", desc, "first note in path order wins")
	})

	t.Run("method not found returns error", func(t *testing.T) {
		env := &EnvMock{}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "unknown_method",
			ID:      5,
		}

		resp := mcp.ResolveForTest(ctx, env, req)

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeMethodNotFound, resp.Error.Code)
		require.Contains(t, resp.Error.Message, "unknown_method")
	})

	t.Run("invalid call params returns error", func(t *testing.T) {
		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(ctx, env, req)

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
	})

	// A notification carries no id and gets no JSON-RPC response at all: the
	// transport answers 202 Accepted with an empty body, per the MCP spec.
	t.Run("notifications/initialized returns no response body", func(t *testing.T) {
		env := &EnvMock{}

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "notifications/initialized",
			ID:      7,
		}

		resp := mcp.ResolveForTest(ctx, env, req)

		require.Nil(t, resp.Error)
		require.Empty(t, resp.JSONRPC)
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
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{
				NoteView:           note,
				URL:                note.Permalink,
				Score:              1.0,
				HighlightedContent: []string{"Лучший способ отомстить - не уподобляться обидчику."},
			}}, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://markavrelii.2pub.me"
		},
		NoteURLFunc: func(note *appmodel.NoteView) string {
			return "https://markavrelii.2pub.me" + note.Permalink
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

	resp := mcp.ResolveForTest(context.Background(), env, req)

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.NotEmpty(t, result.Content)
	require.Contains(t, result.Content[0].Text, "Книга 06")
	require.NotNil(t, result.StructuredContent)

	payload := decodePayload[mcp.SearchResultPayload](t, result)
	require.Equal(t, "обида", payload.Query)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "Книга 06", payload.Results[0].Title)
	require.Equal(t, int64(32), payload.Results[0].NoteID)
	require.Equal(t, "Книги/Книга 06.md", payload.Results[0].NotePath)
	require.Equal(t, "/knigi/kniga_06", payload.Results[0].Href)
	require.Equal(t, "https://markavrelii.2pub.me/knigi/kniga_06", payload.Results[0].URL)
	require.Equal(t, "source", payload.Results[0].Kind)
	require.Len(t, payload.Results[0].Matches, 1)
	// No chunk resolved: match_id is omitted (never an unusable "m"-form id).
	require.Empty(t, payload.Results[0].Matches[0].MatchID)
	require.NotContains(t, result.Content[0].Text, "match_id:")
	require.Contains(t, payload.Results[0].Matches[0].Snippet, "обидчику")
}

func TestExpandReturnsDirectChildren(t *testing.T) {
	note := &appmodel.NoteView{
		Path:   "en/user/example",
		PathID: 42,
		Title:  "Example",
		Headings: appmodel.NoteViewHeadings{
			{Text: "Setup", Level: 1, ID: "setup"},
			{Text: "Install", Level: 2, ID: "install"},
			{Text: "Events", Level: 1, ID: "events"},
		},
	}

	env := &EnvMock{
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return &appmodel.NoteViews{
				List:    []*appmodel.NoteView{note},
				PathMap: map[string]*appmodel.NoteView{note.Path: note},
			}
		},
		LoggerFunc: func() logger.Logger {
			return &logger.DummyLogger{}
		},
		CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
			return true, nil
		},
	}

	call := func(args string) mcp.ExpandPayload {
		params := mcp.CallToolParams{Name: "expand", Arguments: json.RawMessage(args)}
		paramsJSON, _ := json.Marshal(params)
		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 1,
		})
		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.NotEmpty(t, result.Content)
		return decodePayload[mcp.ExpandPayload](t, result)
	}

	// Top level: two level-1 sections, both with children.
	top := call(`{"path":"en/user/example"}`)
	require.Equal(t, int64(42), top.NoteID)
	require.Len(t, top.Children, 2)
	require.Equal(t, "Setup", top.Children[0].Title)
	require.True(t, top.Children[0].HasChildren)
	require.Equal(t, "Events", top.Children[1].Title)
	require.False(t, top.Children[1].HasChildren)

	// Drill into Setup → one leaf subsection.
	setup := call(`{"path":"en/user/example","toc_path":["Setup"]}`)
	require.Len(t, setup.Children, 1)
	require.Equal(t, "Install", setup.Children[0].Title)
	require.Equal(t, []string{"Setup", "Install"}, setup.Children[0].Path)
	require.False(t, setup.Children[0].HasChildren)
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
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{
				NoteView:           note,
				URL:                note.Permalink,
				Score:              0.71,
				HighlightedContent: []string{"Use when: Bob's work-status updates."},
			}}, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://hub.local"
		},
		NoteURLFunc: func(note *appmodel.NoteView) string {
			return "https://hub.local" + note.Permalink
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

	resp := mcp.ResolveForTest(context.Background(), env, req)

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	payload := decodePayload[mcp.SearchResultPayload](t, result)
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
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{
				{NoteView: federationNote, URL: federationNote.Permalink, Score: 2},
				{NoteView: localNote, URL: localNote.Permalink, Score: 1},
			}, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://hub.local"
		},
		NoteURLFunc: func(note *appmodel.NoteView) string {
			return "https://hub.local" + note.Permalink
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

	resp := mcp.ResolveForTest(context.Background(), env, req)

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	payload := decodePayload[mcp.SearchResultPayload](t, result)
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
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{
				{NoteView: privateNote, URL: privateNote.Permalink, Score: 2},
				{NoteView: publicNote, URL: publicNote.Permalink, Score: 1},
			}, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://peer.local"
		},
		NoteURLFunc: func(note *appmodel.NoteView) string {
			return "https://peer.local" + note.Permalink
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
	resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	})

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.NotContains(t, result.Content[0].Text, "Internal Notes")
	require.Contains(t, result.Content[0].Text, "Team Status")
	payload := decodePayload[mcp.SearchResultPayload](t, result)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "Team Status", payload.Results[0].Title)
}

func TestFederatedSearchWithoutKBNotesReturnsStructuredStatus(t *testing.T) {
	env := &EnvMock{
		SiteConfigFunc:      func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		LatestNoteViewsFunc: appmodel.NewNoteViews,
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

	resp := mcp.ResolveForTest(context.Background(), env, req)

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.False(t, result.IsError)
	payload := decodePayload[mcp.FederationStatusPayload](t, result)
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
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{
				{NoteView: systemNote, URL: systemNote.Permalink, Score: 3},
				{NoteView: excludedNote, URL: excludedNote.Permalink, Score: 2},
				{NoteView: publicNote, URL: publicNote.Permalink, Score: 1},
			}, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://markavrelii.2pub.me"
		},
		NoteURLFunc: func(note *appmodel.NoteView) string {
			return "https://markavrelii.2pub.me" + note.Permalink
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

	resp := mcp.ResolveForTest(context.Background(), env, req)

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.Contains(t, result.Content[0].Text, "Книга 06")
	require.NotContains(t, result.Content[0].Text, "Critic report")
	require.NotContains(t, result.Content[0].Text, "Draft")

	payload := decodePayload[mcp.SearchResultPayload](t, result)
	require.Len(t, payload.Results, 1)
}

func TestSearch_CustomDomainURL(t *testing.T) {
	note := &appmodel.NoteView{
		PathID:    99,
		Path:      "custom-note.md",
		Title:     "Custom Domain Note",
		Permalink: "/custom-note",
	}

	env := &EnvMock{
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{
				NoteView: note,
				URL:      note.Permalink,
				Score:    1.0,
			}}, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk {
			return nil
		},
		FeaturesFunc: func() features.Features {
			return features.Features{}
		},
		PublicURLFunc: func() string {
			return "https://main.example.com"
		},
		NoteURLFunc: func(n *appmodel.NoteView) string {
			if n.PathID == 99 {
				return "https://customdomain.test/custom-path"
			}
			return "https://main.example.com" + n.Permalink
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
		Arguments: json.RawMessage(`{"query":"custom"}`),
	}
	paramsJSON, _ := json.Marshal(params)
	req := mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	}

	resp := mcp.ResolveForTest(context.Background(), env, req)

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.NotNil(t, result.StructuredContent)

	payload := decodePayload[mcp.SearchResultPayload](t, result)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "https://customdomain.test/custom-path", payload.Results[0].URL)
	// Href is always the permalink path, URL is the full domain-aware URL
	require.Equal(t, "/custom-note", payload.Results[0].Href)
}

func noteHTMLEnv(note *appmodel.NoteView) *EnvMock {
	return &EnvMock{
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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
}

func TestHandleNoteHtml(t *testing.T) {
	t.Run("returns note HTML", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path: "/test/note",
			HTML: "<h1>Test Note</h1><p>Content here</p>",
		}

		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

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
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

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
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Contains(t, result.Content[0].Text, "Предыдущий фрагмент.")
		require.Contains(t, result.Content[0].Text, "Лучший способ отомстить - не уподобляться обидчику.")
		require.Contains(t, result.Content[0].Text, "Следующий фрагмент.")
		require.NotContains(t, result.Content[0].Text, "FULL NOTE HTML")
	})

	t.Run("match_id alone resolves the note and returns its focused chunk", func(t *testing.T) {
		// Search results hand back match_id as the primary pointer and the
		// tool description advertises note reads by match_id — a match_id-only
		// call must resolve the note instead of demanding another ref.
		note := &appmodel.NoteView{
			Path:      "Книги/Книга 06.md",
			PathID:    32,
			Permalink: "/knigi/kniga_06",
			HTML:      "<h1>Книга 06</h1><p>FULL NOTE HTML</p>",
		}

		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				noteViews := appmodel.NewNoteViews()
				noteViews.RegisterNote(note)
				return noteViews
			},
			LatestNoteChunksFunc: func() []appmodel.NoteChunk {
				return []appmodel.NoteChunk{
					{
						NotePath:   note.Path,
						ChunkIndex: 1,
						Content:    "Книга 06\n\nЛучший способ отомстить - не уподобляться обидчику.",
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
			Arguments: json.RawMessage(`{"match_id": "p32:c1"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 30,
		})

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Contains(t, result.Content[0].Text, "Лучший способ отомстить - не уподобляться обидчику.")
		require.NotContains(t, result.Content[0].Text, "FULL NOTE HTML")
	})

	t.Run("absolute URL href resolves", func(t *testing.T) {
		// Search results return absolute URLs in their url field; models feed
		// them back as href — the path component must resolve like a relative href.
		note := &appmodel.NoteView{
			Path:      "concepts/sverkhchelovek.md",
			PathID:    12,
			Permalink: "/concepts/sverkhchelovek",
			HTML:      "<h1>Сверхчеловек</h1>",
		}

		env := noteHTMLEnv(note)

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"href": "https://nietzsche.2pub.me/concepts/sverkhchelovek"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 31,
		})

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Contains(t, result.Content[0].Text, "Сверхчеловек")
	})

	t.Run("empty note_id string does not break resolution via href", func(t *testing.T) {
		// Live repro: {kb_id, href: absolute URL, match_id, note_id: ""} —
		// the empty note_id must fall through, not fail the whole call.
		note := &appmodel.NoteView{
			Path:      "concepts/sverkhchelovek.md",
			PathID:    12,
			Permalink: "/concepts/sverkhchelovek",
			HTML:      "<h1>Сверхчеловек</h1>",
		}

		env := noteHTMLEnv(note)

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"note_id": "", "href": "https://nietzsche.2pub.me/concepts/sverkhchelovek"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 32,
		})

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Contains(t, result.Content[0].Text, "Сверхчеловек")
	})

	t.Run("bad toc_path returns error with top-level sections, not the full note", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path:      "Книги/Книга 06.md",
			PathID:    32,
			Permalink: "/knigi/kniga_06",
			HTML:      `<div data-header="Интро"><h1>Интро</h1><p>FULL NOTE HTML</p></div>`,
			Headings: appmodel.NoteViewHeadings{
				{Text: "Интро", Level: 1, ID: "intro"},
				{Text: "Раздел", Level: 1, ID: "razdel"},
			},
		}

		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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
			Arguments: json.RawMessage(`{"pid": 32, "toc_path": ["Переименованный заголовок"]}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 4,
		})

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
		require.Contains(t, resp.Error.Message, "section not found")
		require.Contains(t, resp.Error.Message, "Интро")
		require.Contains(t, resp.Error.Message, "Раздел")
		require.NotContains(t, resp.Error.Message, "FULL NOTE HTML")
	})

	t.Run("bad match_id returns error, not the full note", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path:      "Книги/Книга 06.md",
			PathID:    32,
			Permalink: "/knigi/kniga_06",
			HTML:      `<div data-header="Интро"><h1>Интро</h1><p>FULL NOTE HTML</p></div>`,
			Headings: appmodel.NoteViewHeadings{
				{Text: "Интро", Level: 1, ID: "intro"},
			},
		}

		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				noteViews := appmodel.NewNoteViews()
				noteViews.RegisterNote(note)
				return noteViews
			},
			LatestNoteChunksFunc: func() []appmodel.NoteChunk {
				return nil
			},
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
			},
		}

		// Legacy "m"-form match_id that parseChunkMatchID cannot resolve.
		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"pid": 32, "match_id": "p32:m1"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 5,
		})

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
		require.Contains(t, resp.Error.Message, "no focused window")
		require.NotContains(t, resp.Error.Message, "FULL NOTE HTML")
	})

	t.Run("bad match_id falls back to a valid toc_path", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path:      "Книги/Книга 06.md",
			PathID:    32,
			Permalink: "/knigi/kniga_06",
			HTML:      `<div data-header="Интро"><h1>Интро</h1><p>Section body.</p></div>`,
			Headings: appmodel.NoteViewHeadings{
				{Text: "Интро", Level: 1, ID: "intro"},
			},
		}

		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				noteViews := appmodel.NewNoteViews()
				noteViews.RegisterNote(note)
				return noteViews
			},
			LatestNoteChunksFunc: func() []appmodel.NoteChunk {
				return nil
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
			Arguments: json.RawMessage(`{"pid": 32, "match_id": "p32:c9", "toc_path": ["Интро"]}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 6,
		})

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Contains(t, result.Content[0].Text, "Section body.")
	})

	// Models routinely replay ids from search results as pid: the numeric
	// note_id as a string, a chunk match_id ("p36:c2"), even a path. Those
	// calls usually carry a valid path too — a bogus pid must not eclipse it.
	t.Run("falls back to path when pid does not resolve", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path:      "concepts/volya-k-vlasti.md",
			PathID:    36,
			Permalink: "/concepts/volya-k-vlasti",
			HTML:      "<h1>Воля к власти</h1>",
		}

		env := noteHTMLEnv(note)

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"pid": 999999, "path": "concepts/volya-k-vlasti.md"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 8,
		})

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Contains(t, result.Content[0].Text, "Воля к власти")
	})

	t.Run("tolerates match_id-shaped string pid and resolves via path", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path:      "concepts/volya-k-vlasti.md",
			PathID:    36,
			Permalink: "/concepts/volya-k-vlasti",
			HTML:      "<h1>Воля к власти</h1>",
		}

		env := noteHTMLEnv(note)

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"pid": "p36:c2", "path": "concepts/volya-k-vlasti.md"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 9,
		})

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Contains(t, result.Content[0].Text, "Воля к власти")
	})

	t.Run("accepts numeric string pid", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path:      "ru/hub/pascal.md",
			PathID:    70,
			Permalink: "/ru/hub/pascal",
			HTML:      "<h1>Паскаль</h1>",
		}

		env := noteHTMLEnv(note)

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"pid": "70"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 10,
		})

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		require.Contains(t, result.Content[0].Text, "Паскаль")
	})

	t.Run("string pid alone returns clear error naming the bad pid", func(t *testing.T) {
		note := &appmodel.NoteView{
			Path:      "concepts/volya-k-vlasti.md",
			PathID:    36,
			Permalink: "/concepts/volya-k-vlasti",
			HTML:      "<h1>Воля к власти</h1>",
		}

		env := noteHTMLEnv(note)

		params := mcp.CallToolParams{
			Name:      "note_html",
			Arguments: json.RawMessage(`{"pid": "p36:c2"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 11,
		})

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
		require.Contains(t, resp.Error.Message, "p36:c2")
	})

	t.Run("returns error instead of empty success for note with no rendered content", func(t *testing.T) {
		// A frontmatter-only or otherwise empty note must not answer a
		// valid-looking read with text:"" — a retrieval endpoint returning
		// silent empty content poisons downstream consumers.
		note := &appmodel.NoteView{
			Path:      "demo/pointer.md",
			PathID:    77,
			Permalink: "/demo/pointer",
			HTML:      "",
		}

		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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
			Arguments: json.RawMessage(`{"pid": 77}`),
		}
		paramsJSON, _ := json.Marshal(params)

		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 7,
		})

		require.NotNil(t, resp.Error)
		require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
		require.Contains(t, resp.Error.Message, "demo/pointer.md")
		require.Contains(t, resp.Error.Message, "no rendered content")
	})

	t.Run("returns error for non-existent note", func(t *testing.T) {
		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

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
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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
		NoteURLFunc: func(note *appmodel.NoteView) string {
			return "https://markavrelii.2pub.me" + note.Permalink
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

	resp := mcp.ResolveForTest(context.Background(), env, req)

	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	require.Contains(t, result.Content[0].Text, "Книга 07")
	require.Contains(t, result.Content[0].Text, "https://markavrelii.2pub.me/knigi/kniga_07")

	payload := decodePayload[mcp.SimilarResultPayload](t, result)
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
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

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
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

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
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

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
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		// Our current implementation looks for \n--- which should work correctly here
		require.Equal(t, "Actual content after frontmatter", result.Content[0].Text)
	})
}

func TestHandleSimilarLimitValidation(t *testing.T) {
	t.Run("uses default when limit is zero", func(t *testing.T) {
		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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
			NoteURLFunc: func(note *appmodel.NoteView) string {
				return "https://example.test" + note.Permalink
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

		require.Nil(t, resp.Error)
		// Test passes if no error - default limit should be used
	})

	t.Run("caps limit at maximum", func(t *testing.T) {
		env := &EnvMock{
			SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
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
			NoteURLFunc: func(note *appmodel.NoteView) string {
				return "https://example.test" + note.Permalink
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

		resp := mcp.ResolveForTest(context.Background(), env, req)

		require.Nil(t, resp.Error)
		// Test passes if no error - limit should be capped
	})
}

// makeSearchNote creates a NoteView with a given PathID, path, and title for search tests.
func makeSearchNote(pathID int64, path, title string) *appmodel.NoteView {
	return &appmodel.NoteView{
		Path:      path,
		PathID:    pathID,
		Title:     title,
		Permalink: "/" + path,
	}
}

// makeSearchEnv builds a minimal Env mock that returns results from SearchLatestNotes.
func makeSearchEnv(t *testing.T, results []appmodel.SearchResult) *EnvMock {
	t.Helper()
	return &EnvMock{
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc: func(query string) ([]appmodel.SearchResult, error) {
			return results, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk { return nil },
		FeaturesFunc:       func() features.Features { return features.Features{} },
		PublicURLFunc:      func() string { return "https://example.test" },
		NoteURLFunc: func(note *appmodel.NoteView) string {
			return "https://example.test" + note.Permalink
		},
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
		CanReadNoteFunc: func(_ context.Context, _ *appmodel.NoteView) (bool, error) {
			return true, nil
		},
	}
}

// searchCall issues a tools/call search request and returns the payload.
func searchCall(t *testing.T, env *EnvMock, argsJSON string) mcp.SearchResultPayload {
	t.Helper()
	params := mcp.CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(argsJSON),
	}
	paramsJSON, _ := json.Marshal(params)
	resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
		JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 1,
	})
	require.Nil(t, resp.Error, "unexpected error: %v", resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	return decodePayload[mcp.SearchResultPayload](t, result)
}

// makeNResults builds N search results with distinct notes.
func makeNResults(n int) []appmodel.SearchResult {
	results := make([]appmodel.SearchResult, n)
	for i := range n {
		note := makeSearchNote(int64(i+1), fmt.Sprintf("note%d.md", i+1), fmt.Sprintf("Note %d", i+1))
		results[i] = appmodel.SearchResult{
			NoteView:           note,
			URL:                note.Permalink,
			Score:              float64(n - i),
			HighlightedContent: []string{fmt.Sprintf("Snippet for note %d", i+1)},
		}
	}
	return results
}

func TestSearchLimitAndDetailLimit(t *testing.T) {
	t.Run("defaults: 6 total, 3 full detail", func(t *testing.T) {
		env := makeSearchEnv(t, makeNResults(10))
		payload := searchCall(t, env, `{"query":"test"}`)

		// default limit = 6
		require.Len(t, payload.Results, 6)

		// first 3 have Matches, results[3..5] are previews (Matches nil)
		for i := range 3 {
			require.NotNil(t, payload.Results[i].Matches, "result[%d] should have Matches", i)
		}
		for i := 3; i < 6; i++ {
			require.Nil(t, payload.Results[i].Matches, "result[%d] should be a preview (no Matches)", i)
		}
	})

	t.Run("explicit limit honored", func(t *testing.T) {
		env := makeSearchEnv(t, makeNResults(10))
		payload := searchCall(t, env, `{"query":"test","limit":4}`)
		require.Len(t, payload.Results, 4)
	})

	t.Run("explicit detail_limit honored", func(t *testing.T) {
		env := makeSearchEnv(t, makeNResults(8))
		payload := searchCall(t, env, `{"query":"test","limit":5,"detail_limit":2}`)
		require.Len(t, payload.Results, 5)
		for i := range 2 {
			require.NotNil(t, payload.Results[i].Matches, "result[%d] should have Matches", i)
		}
		for i := 2; i < 5; i++ {
			require.Nil(t, payload.Results[i].Matches, "result[%d] should be preview", i)
		}
	})

	t.Run("detail_limit clamped to limit", func(t *testing.T) {
		env := makeSearchEnv(t, makeNResults(8))
		// detail_limit=10 > limit=4 → clamp to 4, all have Matches
		payload := searchCall(t, env, `{"query":"test","limit":4,"detail_limit":10}`)
		require.Len(t, payload.Results, 4)
		for i := range payload.Results {
			require.NotNil(t, payload.Results[i].Matches, "result[%d] should have Matches (clamped detail_limit)", i)
		}
	})

	t.Run("limit capped at MaxSearchLimit", func(t *testing.T) {
		env := makeSearchEnv(t, makeNResults(25))
		payload := searchCall(t, env, `{"query":"test","limit":999}`)
		require.LessOrEqual(t, len(payload.Results), mcp.MaxSearchLimit)
	})

	t.Run("total results do not exceed limit when fewer results exist", func(t *testing.T) {
		env := makeSearchEnv(t, makeNResults(3))
		payload := searchCall(t, env, `{"query":"test"}`) // default limit=6
		require.Len(t, payload.Results, 3)                // only 3 available
	})

	t.Run("detail_limit=0 defaults to DefaultSearchDetailLimit", func(t *testing.T) {
		env := makeSearchEnv(t, makeNResults(8))
		payload := searchCall(t, env, `{"query":"test","limit":6,"detail_limit":0}`)
		// default detail_limit=3
		require.Len(t, payload.Results, 6)
		for i := range 3 {
			require.NotNil(t, payload.Results[i].Matches, "result[%d] should have Matches", i)
		}
		for i := 3; i < 6; i++ {
			require.Nil(t, payload.Results[i].Matches, "result[%d] should be preview", i)
		}
	})

	t.Run("text output shows snippets for detail results, path-only for previews", func(t *testing.T) {
		env := makeSearchEnv(t, makeNResults(4))
		params := mcp.CallToolParams{
			Name:      "search",
			Arguments: json.RawMessage(`{"query":"test","limit":4,"detail_limit":2}`),
		}
		paramsJSON, _ := json.Marshal(params)
		resp := mcp.ResolveForTest(context.Background(), env, mcp.Request{
			JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 1,
		})
		require.Nil(t, resp.Error)
		result := resp.Result.(mcp.CallToolResult)
		text := result.Content[0].Text
		// Full detail results include snippet; previews should include "[preview]" marker or similar
		require.Contains(t, text, "Snippet for note 1")
		require.Contains(t, text, "Snippet for note 2")
		// Preview results (index 2,3) should NOT contain their snippets in the text
		require.NotContains(t, text, "Snippet for note 3")
		require.NotContains(t, text, "Snippet for note 4")
	})
}
