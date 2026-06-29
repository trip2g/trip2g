package fleet

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

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

	c := NewHTTPClient(srv.URL, srv.Client())
	_, err := c.GraphQLScoped(context.Background(), "scoped-token", "mutation{x}", nil)
	require.NoError(t, err)

	require.Equal(t, "/_system/graphql", gotPath)
	require.Empty(t, gotKey)
	require.Equal(t, "Bearer scoped-token", gotAuth)
	// Scoped lane posts raw GraphQL (NOT a JSON-RPC envelope).
	require.Contains(t, gotBody, `"query":"mutation{x}"`)
	require.NotContains(t, gotBody, "jsonrpc")
}

// TestHTTPClient_ScopedGraphQLErrorsSurface asserts the scoped lane decodes the
// raw GraphQL {data, errors} shape and surfaces errors[0].message.
func TestHTTPClient_ScopedGraphQLErrorsSurface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, srv.Client())
	_, err := c.GraphQLScoped(context.Background(), "tok", "q", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}
