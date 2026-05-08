package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

// mcpToolsListBody returns a JSON-RPC body for tools/list.
func mcpToolsListBody(t *testing.T) []byte {
	t.Helper()
	b, err := json.Marshal(mcp.Request{JSONRPC: "2.0", Method: "tools/list", ID: 1})
	require.NoError(t, err)
	return b
}

// mcpToolsCallBody returns a JSON-RPC body for tools/call with the given tool and args.
func mcpToolsCallBody(t *testing.T, name string, args map[string]any) []byte {
	t.Helper()
	rawArgs, err := json.Marshal(args)
	require.NoError(t, err)
	body, err := json.Marshal(mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		ID:      1,
		Params:  mustJSON(t, mcp.CallToolParams{Name: name, Arguments: rawArgs}),
	})
	require.NoError(t, err)
	return body
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// TestAPIKeyAdminTools_ToolsListVisibility: graphql_* tools must appear ONLY when
// the API key has enable_mcp_admin_tools=true.
func TestAPIKeyAdminTools_ToolsListVisibility(t *testing.T) {
	cases := []struct {
		name       string
		adminTools bool
		shouldList bool
	}{
		{"admin tools enabled exposes graphql tools", true, true},
		{"admin tools disabled hides graphql tools", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := buildDispatchEnv(t, true)
			adminTools := tc.adminTools
			env.ResolveAPIKeyFunc = func(_ context.Context, _, _ string) (*db.ApiKey, error) {
				return &db.ApiKey{ID: 1, EnableMcpAdminTools: &adminTools}, nil
			}

			fasthttpCtx := buildMCPFasthttpCtx(mcpToolsListBody(t), "")
			fasthttpCtx.Request.Header.Set("X-API-Key", "any-key")

			req := wiredRequest(fasthttpCtx, env, nil)
			defer appreq.Release(req)

			_, err := (&mcp.Endpoint{}).Handle(req)
			require.NoError(t, err)

			var resp mcp.Response
			require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
			require.Nil(t, resp.Error)

			rawResult, err := json.Marshal(resp.Result)
			require.NoError(t, err)
			var listResult mcp.ListToolsResult
			require.NoError(t, json.Unmarshal(rawResult, &listResult))

			toolNames := make(map[string]bool, len(listResult.Tools))
			for _, tool := range listResult.Tools {
				toolNames[tool.Name] = true
			}

			require.Equal(t, tc.shouldList, toolNames["graphql_introspection"],
				"graphql_introspection visibility mismatch")
			require.Equal(t, tc.shouldList, toolNames["graphql_request"],
				"graphql_request visibility mismatch")
		})
	}
}

// TestAPIKeyAdminTools_GraphQLRequestDispatched: tools/call graphql_request with
// admin tools enabled must invoke env.GraphQLRequest.
func TestAPIKeyAdminTools_GraphQLRequestDispatched(t *testing.T) {
	env := buildDispatchEnv(t, true)
	adminTools := true
	env.ResolveAPIKeyFunc = func(_ context.Context, _, _ string) (*db.ApiKey, error) {
		return &db.ApiKey{ID: 1, EnableMcpAdminTools: &adminTools}, nil
	}
	gqlCalled := false
	env.GraphQLRequestFunc = func(_ context.Context, query string, vars map[string]any) ([]byte, error) {
		gqlCalled = true
		require.Contains(t, query, "mutation")
		require.Equal(t, map[string]any{"id": float64(42)}, vars)
		return []byte(`{"data":{"adminMutation":{"setApiKeyMcpAdminTools":{"apiKey":{"id":42}}}}}`), nil
	}

	body := mcpToolsCallBody(t, "graphql_request", map[string]any{
		"query":     "mutation { adminMutation { setApiKeyMcpAdminTools(input: {id: 42, enabled: true}) { ... on SetApiKeyMcpAdminToolsPayload { apiKey { id } } } } }",
		"variables": map[string]any{"id": 42},
	})

	fasthttpCtx := buildMCPFasthttpCtx(body, "")
	fasthttpCtx.Request.Header.Set("X-API-Key", "admin-key")

	req := wiredRequest(fasthttpCtx, env, nil)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
	require.Nil(t, resp.Error, "expected success, got: %v", resp.Error)
	require.True(t, gqlCalled, "GraphQLRequest must be invoked")

	rawResult, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var callResult mcp.CallToolResult
	require.NoError(t, json.Unmarshal(rawResult, &callResult))
	require.Len(t, callResult.Content, 1)
	require.Contains(t, callResult.Content[0].Text, `"setApiKeyMcpAdminTools"`)
}

// TestAPIKeyAdminTools_GraphQLRequestBlockedWithoutFlag: tools/call graphql_request
// without admin tools must return MethodNotFound and NOT invoke env.GraphQLRequest.
func TestAPIKeyAdminTools_GraphQLRequestBlockedWithoutFlag(t *testing.T) {
	env := buildDispatchEnv(t, true)
	adminTools := false
	env.ResolveAPIKeyFunc = func(_ context.Context, _, _ string) (*db.ApiKey, error) {
		return &db.ApiKey{ID: 1, EnableMcpAdminTools: &adminTools}, nil
	}
	env.GraphQLRequestFunc = func(_ context.Context, _ string, _ map[string]any) ([]byte, error) {
		t.Error("GraphQLRequest must NOT be called when admin tools disabled")
		return nil, errors.New("must not be called")
	}

	body := mcpToolsCallBody(t, "graphql_request", map[string]any{
		"query": "{ __typename }",
	})

	fasthttpCtx := buildMCPFasthttpCtx(body, "")
	fasthttpCtx.Request.Header.Set("X-API-Key", "non-admin-key")

	req := wiredRequest(fasthttpCtx, env, nil)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
	require.NotNil(t, resp.Error)
	require.Equal(t, mcp.ErrCodeMethodNotFound, resp.Error.Code)
	require.Contains(t, resp.Error.Message, "graphql_request")
}

// TestAPIKeyAdminTools_GraphQLIntrospectionFiltering: tools/call graphql_introspection
// must invoke env.GraphQLRequest with the introspection query, then filter by pattern.
func TestAPIKeyAdminTools_GraphQLIntrospectionFiltering(t *testing.T) {
	env := buildDispatchEnv(t, true)
	adminTools := true
	env.ResolveAPIKeyFunc = func(_ context.Context, _, _ string) (*db.ApiKey, error) {
		return &db.ApiKey{ID: 1, EnableMcpAdminTools: &adminTools}, nil
	}
	// Return a stub introspection response with two types — one matches the pattern.
	env.GraphQLRequestFunc = func(_ context.Context, query string, _ map[string]any) ([]byte, error) {
		require.Contains(t, query, "__schema")
		return []byte(`{
			"data": {
				"__schema": {
					"queryType": {"name": "Query"},
					"types": [
						{"kind": "OBJECT", "name": "ApiKey", "fields": []},
						{"kind": "OBJECT", "name": "Unrelated", "fields": []}
					]
				}
			}
		}`), nil
	}

	body := mcpToolsCallBody(t, "graphql_introspection", map[string]any{
		"pattern": "ApiKey",
	})

	fasthttpCtx := buildMCPFasthttpCtx(body, "")
	fasthttpCtx.Request.Header.Set("X-API-Key", "admin-key")

	req := wiredRequest(fasthttpCtx, env, nil)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
	require.Nil(t, resp.Error, "expected success, got: %v", resp.Error)

	rawResult, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var callResult mcp.CallToolResult
	require.NoError(t, json.Unmarshal(rawResult, &callResult))
	require.Len(t, callResult.Content, 1)
	text := callResult.Content[0].Text
	require.Contains(t, text, `"name":"ApiKey"`)
	require.NotContains(t, text, `"name":"Unrelated"`)
}
