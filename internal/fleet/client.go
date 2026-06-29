package fleet

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg fleet . Client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is the trip2g scoped write lane: it POSTs raw GraphQL to
// /_system/graphql authenticated by a Bearer shortapitoken, and is the only lane
// per-delivery note IO (RemoteKB) ever touches. The admin lane (discovery +
// reconcile) is a separate, genqlient-generated typed client built by
// NewAdminGraphQLClient (see adminlane.go) and is no longer part of this
// interface.
type Client interface {
	GraphQLScoped(ctx context.Context, token, query string, vars map[string]any) (json.RawMessage, error)
}

type httpClient struct {
	baseURL string
	hc      *http.Client
}

// NewHTTPClient builds the concrete HTTP-backed scoped Client.
func NewHTTPClient(baseURL string, hc *http.Client) Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &httpClient{baseURL: baseURL, hc: hc}
}

// GraphQLScoped posts a raw GraphQL {query, variables} request to
// /_system/graphql with a Bearer shortapitoken and decodes the standard
// {data, errors} envelope.
func (c *httpClient) GraphQLScoped(ctx context.Context, token, query string, vars map[string]any) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/_system/graphql", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

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
