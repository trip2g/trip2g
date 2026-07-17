package fleet

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/cmd/fleet/internal/fleet/trip2ggql"
)

// hatCookieHandler writes a trip2g_token session cookie + 302, mimicking the
// real /_system/hat exchange.
func hatCookieHandler(value string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "trip2g_token", Value: value, Path: "/"})
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
	}
}

// TestAdminGQL_RawGraphQLWithCookie asserts the genqlient admin lane mints a
// session cookie via /_system/hat, then POSTs the raw GraphQL {query,variables}
// to /_system/graphql with that cookie attached (NOT an MCP/JSON-RPC envelope,
// and no X-Api-Key), and decodes the typed response.
func TestAdminGQL_RawGraphQLWithCookie(t *testing.T) {
	var gotPath, gotCookie, gotKey, gotBody string
	mux := http.NewServeMux()
	mux.Handle("/_system/hat", hatCookieHandler("sess123"))
	mux.HandleFunc("/_system/graphql", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if c, err := r.Cookie("trip2g_token"); err == nil {
			gotCookie = c.Value
		}
		gotKey = r.Header.Get("X-Api-Key")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"data":{"admin":{"allChangeWebhooks":{"nodes":[]}}}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	gql := NewAdminGraphQLClient(srv.URL, "secret", "fleet@local", srv.Client())
	resp, err := trip2ggql.ListChangeWebhooks(context.Background(), gql)
	require.NoError(t, err)
	require.Empty(t, resp.Admin.AllChangeWebhooks.Nodes)

	require.Equal(t, "/_system/graphql", gotPath)
	require.Equal(t, "sess123", gotCookie, "admin lane must attach the HAT session cookie")
	require.Empty(t, gotKey, "admin lane must not send the legacy X-Api-Key")
	require.Contains(t, gotBody, `"operationName":"ListChangeWebhooks"`)
	require.Contains(t, gotBody, "allChangeWebhooks")
	require.NotContains(t, gotBody, "jsonrpc")
}

// TestAdminGQL_GraphQLErrorSurfaces asserts a raw GraphQL error envelope from
// /_system/graphql surfaces as a Go error carrying the message.
func TestAdminGQL_GraphQLErrorSurfaces(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/_system/hat", hatCookieHandler("sess"))
	mux.HandleFunc("/_system/graphql", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"Cannot query field allChangeWebhooks"}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	gql := NewAdminGraphQLClient(srv.URL, "secret", "fleet@local", srv.Client())
	_, err := trip2ggql.ListChangeWebhooks(context.Background(), gql)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Cannot query field allChangeWebhooks")
}

// TestAdminGQL_ReauthOn401 asserts that when the session cookie is rejected
// (HTTP 401), the admin lane re-runs the HAT exchange to refresh the cookie and
// retries the request once with the fresh cookie.
func TestAdminGQL_ReauthOn401(t *testing.T) {
	var hatCalls, gqlCalls int32
	var retriedCookie string
	mux := http.NewServeMux()
	mux.HandleFunc("/_system/hat", func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&hatCalls, 1)
		http.SetCookie(w, &http.Cookie{Name: "trip2g_token", Value: "sess" + strconv.Itoa(int(n)), Path: "/"})
		w.WriteHeader(http.StatusFound)
	})
	mux.HandleFunc("/_system/graphql", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&gqlCalls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		if c, err := r.Cookie("trip2g_token"); err == nil {
			retriedCookie = c.Value
		}
		_, _ = w.Write([]byte(`{"data":{"admin":{"allChangeWebhooks":{"nodes":[]}}}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	gql := NewAdminGraphQLClient(srv.URL, "secret", "fleet@local", srv.Client())
	_, err := trip2ggql.ListChangeWebhooks(context.Background(), gql)
	require.NoError(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(&hatCalls), "HAT exchange runs once on first use + once on 401 re-auth")
	require.Equal(t, int32(2), atomic.LoadInt32(&gqlCalls), "the request is retried exactly once after re-auth")
	require.Equal(t, "sess2", retriedCookie, "the retry must carry the refreshed session cookie")
}
