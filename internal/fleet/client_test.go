package fleet

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHTTPClient_AdminLaneMCPEnvelope asserts the admin lane wraps the GraphQL
// request in a JSON-RPC tools/call graphql_request envelope, POSTs it to
// /_system/mcp with the X-Api-Key + Accept headers, and unwraps the GraphQL data
// out of result.structuredContent.data. This is the real /_system/mcp contract;
// the endpoint is JSON-RPC MCP, not a raw GraphQL endpoint.
func TestHTTPClient_AdminLaneMCPEnvelope(t *testing.T) {
	var gotPath, gotKey, gotAuth, gotAccept string
	var gotBody struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Name      string `json:"name"`
			Arguments struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			} `json:"arguments"`
		} `json:"params"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-Api-Key")
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"structured result"}],"structuredContent":{"data":{"ok":true}}}}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "admin-key", srv.Client())
	raw, err := c.GraphQLAdmin(context.Background(), "query{ok}", map[string]any{"x": 1})
	require.NoError(t, err)

	require.Equal(t, "/_system/mcp", gotPath)
	require.Equal(t, "admin-key", gotKey)
	require.Empty(t, gotAuth)
	require.Contains(t, gotAccept, "application/json")

	require.Equal(t, "2.0", gotBody.JSONRPC)
	require.Equal(t, "tools/call", gotBody.Method)
	require.Equal(t, "graphql_request", gotBody.Params.Name)
	require.Equal(t, "query{ok}", gotBody.Params.Arguments.Query)
	require.Equal(t, map[string]any{"x": float64(1)}, gotBody.Params.Arguments.Variables)

	require.JSONEq(t, `{"ok":true}`, string(raw))
}

// TestHTTPClient_AdminLaneErrorEnvelope asserts a JSON-RPC error envelope from
// /_system/mcp (e.g. a GraphQL/tool error) surfaces as a Go error carrying the
// error message.
func TestHTTPClient_AdminLaneErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"GraphQL request failed: Cannot query field allChangeWebhooks"}}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "k", srv.Client())
	_, err := c.GraphQLAdmin(context.Background(), "query{allChangeWebhooks}", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Cannot query field allChangeWebhooks")
}

// TestHTTPClient_AdminLaneContentTextFallback asserts that when
// structuredContent.data is absent, the admin lane falls back to parsing the
// data key out of result.content[0].text.
func TestHTTPClient_AdminLaneContentTextFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"data\":{\"ok\":true}}"}]}}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "k", srv.Client())
	raw, err := c.GraphQLAdmin(context.Background(), "query{ok}", nil)
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true}`, string(raw))
}

func TestHTTPClient_ScopedLaneBearer(t *testing.T) {
	var gotPath, gotKey, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-Api-Key")
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "admin-key", srv.Client())
	_, err := c.GraphQLScoped(context.Background(), "scoped-token", "mutation{x}", nil)
	require.NoError(t, err)

	require.Equal(t, "/_system/graphql", gotPath)
	require.Empty(t, gotKey)
	require.Equal(t, "Bearer scoped-token", gotAuth)
	// Scoped lane posts raw GraphQL (NOT a JSON-RPC envelope).
	require.Contains(t, gotBody, `"query":"mutation{x}"`)
	require.NotContains(t, gotBody, "jsonrpc")
}

// TestHTTPClient_ScopedGraphQLErrorsSurface asserts the scoped lane still decodes
// the raw GraphQL {data, errors} shape and surfaces errors[0].message.
func TestHTTPClient_ScopedGraphQLErrorsSurface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "k", srv.Client())
	_, err := c.GraphQLScoped(context.Background(), "tok", "q", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}
