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

func (c *httpClient) GraphQLAdmin(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error) {
	return c.do(ctx, "/_system/mcp", map[string]string{"X-Api-Key": c.adminKey}, query, vars)
}

func (c *httpClient) GraphQLScoped(ctx context.Context, token, query string, vars map[string]any) (json.RawMessage, error) {
	return c.do(ctx, "/_system/graphql", map[string]string{"Authorization": "Bearer " + token}, query, vars)
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
