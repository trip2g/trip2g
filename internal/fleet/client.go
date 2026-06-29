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
//   - GraphQLAdmin: POST raw GraphQL to /_system/graphql with a HAT-minted admin
//     session cookie (full-admin elevation), used by discovery + reconcile only.
//   - GraphQLScoped: POST /_system/graphql with a Bearer shortapitoken,
//     used for all per-delivery note IO (the only lane RemoteKB ever touches).
type Client interface {
	GraphQLAdmin(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error)
	GraphQLScoped(ctx context.Context, token, query string, vars map[string]any) (json.RawMessage, error)
}

type httpClient struct {
	baseURL string
	hat     *hatAuthenticator
	hc      *http.Client
}

// NewHTTPClient builds the concrete HTTP-backed Client. The admin lane mints a
// Hot-Auth-Token from jwtSecret (the shared user-token/JWT secret), exchanges it
// at /_system/hat for an admin session cookie (self-provisioning adminEmail),
// and rides that cookie on raw admin{} GraphQL.
func NewHTTPClient(baseURL, jwtSecret, adminEmail string, hc *http.Client) Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &httpClient{
		baseURL: baseURL,
		hat:     newHATAuthenticator(baseURL, jwtSecret, adminEmail, hc),
		hc:      hc,
	}
}

// GraphQLAdmin runs a GraphQL operation on the admin lane: raw GraphQL on
// /_system/graphql carrying the HAT-minted admin session cookie. It mints the
// cookie on first use, and on a 401 it re-runs the HAT exchange and retries once.
func (c *httpClient) GraphQLAdmin(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error) {
	if len(c.hat.cached()) == 0 {
		if err := c.hat.authenticate(ctx); err != nil {
			return nil, err
		}
	}
	raw, status, err := c.doAdmin(ctx, query, vars)
	if err != nil && status == http.StatusUnauthorized {
		if aerr := c.hat.authenticate(ctx); aerr != nil {
			return nil, aerr
		}
		raw, _, err = c.doAdmin(ctx, query, vars)
	}
	return raw, err
}

func (c *httpClient) GraphQLScoped(ctx context.Context, token, query string, vars map[string]any) (json.RawMessage, error) {
	raw, _, err := c.do(ctx, "/_system/graphql", token, nil, query, vars)
	return raw, err
}

// doAdmin posts raw GraphQL to /_system/graphql with the cached admin session
// cookie(s). It returns the HTTP status alongside the result so GraphQLAdmin can
// detect a 401 and re-authenticate.
func (c *httpClient) doAdmin(ctx context.Context, query string, vars map[string]any) (json.RawMessage, int, error) {
	return c.do(ctx, "/_system/graphql", "", c.hat.cached(), query, vars)
}

// do posts a raw GraphQL {query, variables} request and decodes the standard
// {data, errors} envelope. A bearer token (scoped lane) and/or cookies (admin
// lane) authenticate it. A 401 is returned as an error with the HTTP status so
// the admin lane can re-authenticate.
func (c *httpClient) do(ctx context.Context, path, bearer string, cookies []*http.Cookie, query string, vars map[string]any) (json.RawMessage, int, error) {
	body, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	for _, ck := range cookies {
		req.AddCookie(ck)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, resp.StatusCode, fmt.Errorf("graphql unauthorized (HTTP %d)", resp.StatusCode)
	}

	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if decErr := json.NewDecoder(resp.Body).Decode(&env); decErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode graphql response (HTTP %d): %w", resp.StatusCode, decErr)
	}
	if len(env.Errors) > 0 {
		return nil, resp.StatusCode, fmt.Errorf("graphql error: %s", env.Errors[0].Message)
	}
	return env.Data, resp.StatusCode, nil
}
