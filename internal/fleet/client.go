package fleet

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg fleet . Client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is the trip2g API surface the fleet depends on. Two lanes:
//   - GraphQLAdmin: POST /_system/mcp with X-API-Key (full-admin elevation),
//     used by discovery + reconcile only.
//   - GraphQLScoped: POST /_system/graphql with a Bearer shortapitoken,
//     used for all per-delivery note IO (the only lane RemoteKB ever touches).
type Client interface {
	GraphQLAdmin(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error)
	GraphQLScoped(ctx context.Context, token, query string, vars map[string]any) (json.RawMessage, error)
}

type httpClient struct {
	baseURL  string
	adminKey string
	hc       *http.Client
}

// NewHTTPClient builds the concrete HTTP-backed Client.
func NewHTTPClient(baseURL, adminKey string, hc *http.Client) Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &httpClient{baseURL: baseURL, adminKey: adminKey, hc: hc}
}

// mcpRequestID is the JSON-RPC id used for admin-lane tools/call requests. The
// MCP endpoint echoes it back; a constant is sufficient since requests are
// request/response and never multiplexed on one connection.
const mcpRequestID = 1

// GraphQLAdmin runs a GraphQL operation through the MCP graphql_request tool.
// /_system/mcp is a JSON-RPC 2.0 MCP endpoint (NOT a raw GraphQL endpoint), so
// the operation is wrapped in a tools/call envelope and the GraphQL data is
// unwrapped from result.structuredContent.data.
func (c *httpClient) GraphQLAdmin(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error) {
	return c.doMCP(ctx, query, vars)
}

func (c *httpClient) GraphQLScoped(ctx context.Context, token, query string, vars map[string]any) (json.RawMessage, error) {
	return c.do(ctx, "/_system/graphql", map[string]string{"Authorization": "Bearer " + token}, query, vars)
}

// doMCP posts a JSON-RPC tools/call graphql_request envelope to /_system/mcp and
// returns the GraphQL data (result.structuredContent.data). A JSON-RPC error is
// surfaced as a Go error.
func (c *httpClient) doMCP(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      mcpRequestID,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "graphql_request",
			"arguments": map[string]any{
				"query":     query,
				"variables": vars,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/_system/mcp", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Api-Key", c.adminKey)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var env struct {
		Result *struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			StructuredContent struct {
				Data json.RawMessage `json:"data"`
			} `json:"structuredContent"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if decErr := json.NewDecoder(resp.Body).Decode(&env); decErr != nil {
		return nil, fmt.Errorf("decode mcp response (HTTP %d): %w", resp.StatusCode, decErr)
	}
	if env.Error != nil {
		return nil, fmt.Errorf("%s", env.Error.Message)
	}
	if env.Result == nil {
		return nil, fmt.Errorf("mcp response missing result (HTTP %d)", resp.StatusCode)
	}
	if len(env.Result.StructuredContent.Data) > 0 {
		return env.Result.StructuredContent.Data, nil
	}
	// Fallback: some responses carry the GraphQL envelope only in the text
	// content block; parse the data key out of it.
	if len(env.Result.Content) > 0 && env.Result.Content[0].Text != "" {
		var inner struct {
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal([]byte(env.Result.Content[0].Text), &inner) == nil && len(inner.Data) > 0 {
			return inner.Data, nil
		}
	}
	return nil, fmt.Errorf("mcp response missing structuredContent.data (HTTP %d)", resp.StatusCode)
}

func (c *httpClient) do(ctx context.Context, path string, headers map[string]string, query string, vars map[string]any) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if decErr := json.NewDecoder(resp.Body).Decode(&env); decErr != nil {
		return nil, fmt.Errorf("decode graphql response (HTTP %d): %w", resp.StatusCode, decErr)
	}
	if len(env.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", env.Errors[0].Message)
	}
	return env.Data, nil
}
