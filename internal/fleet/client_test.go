package fleet

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPClient_AdminLaneHeadersAndPath(t *testing.T) {
	var gotPath, gotKey, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-Api-Key")
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "admin-key", srv.Client())
	raw, err := c.GraphQLAdmin(context.Background(), "query{ok}", map[string]any{"x": 1})
	require.NoError(t, err)

	require.Equal(t, "/_system/mcp", gotPath)
	require.Equal(t, "admin-key", gotKey)
	require.Empty(t, gotAuth)
	require.Contains(t, gotBody, `"query":"query{ok}"`)
	require.Contains(t, gotBody, `"variables":{"x":1}`)

	require.JSONEq(t, `{"ok":true}`, string(raw))
}

func TestHTTPClient_ScopedLaneBearer(t *testing.T) {
	var gotPath, gotKey, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-Api-Key")
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "admin-key", srv.Client())
	_, err := c.GraphQLScoped(context.Background(), "scoped-token", "mutation{x}", nil)
	require.NoError(t, err)

	require.Equal(t, "/_system/graphql", gotPath)
	require.Empty(t, gotKey)
	require.Equal(t, "Bearer scoped-token", gotAuth)
}

func TestHTTPClient_GraphQLErrorsSurface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "k", srv.Client())
	_, err := c.GraphQLAdmin(context.Background(), "q", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}
